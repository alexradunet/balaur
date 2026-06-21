package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// chatTimeout caps one remote streaming generation when the caller supplies no
// deadline, mirroring kronk.chatStreamTimeout so a slow provider is not cut off
// mid-reply.
const chatTimeout = 10 * time.Minute

// OpenAIClient is Balaur's opt-in remote llm.Client: it speaks the
// OpenAI-compatible Chat Completions + embeddings HTTP API, which OpenAI,
// OpenRouter, Groq, Together, Anthropic (via its compat shim), LM Studio and
// llama-server all serve. It is NEVER the default — a turn reaches it only when
// the owner has explicitly selected and consented to a cloud model (see the
// consent gate in internal/web). Provider choice is explicit configuration;
// Balaur never auto-routes to it.
type OpenAIClient struct {
	BaseURL    string // e.g. https://api.openai.com/v1 or http://127.0.0.1:11435/v1
	APIKey     string // Bearer token; empty for keyless local servers
	Model      string // chat model id, e.g. "gpt-4o"
	EmbedModel string // empty: use Model for embeddings too
	HTTP       *http.Client
}

func (c *OpenAIClient) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: chatTimeout}
}

// post sends a JSON body to BaseURL+path. Non-2xx responses are surfaced as
// wrapped errors carrying the provider's body so a bad key or model id reads
// clearly; the API key is never included.
func (c *OpenAIClient) post(ctx context.Context, path string, body any) (*http.Response, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return nil, fmt.Errorf("%s: %s: %s", path, resp.Status, strings.TrimSpace(buf.String()))
	}
	return resp, nil
}

// wire types — the subset of the OpenAI API Balaur uses.

type wireMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type wireToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func toWire(msgs []Message) []wireMessage {
	out := make([]wireMessage, 0, len(msgs))
	for _, m := range msgs {
		wm := wireMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			var w wireToolCall
			w.ID = tc.ID
			w.Type = "function"
			w.Function.Name = tc.Name
			w.Function.Arguments = tc.Args
			wm.ToolCalls = append(wm.ToolCalls, w)
		}
		out = append(out, wm)
	}
	return out
}

func toWireTools(tools []ToolSpec) []map[string]any {
	if len(tools) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		})
	}
	return out
}

// ChatStream implements llm.Client over the OpenAI streaming chat endpoint. The
// returned channel is closed after a single terminal Done (carrying the
// assembled tool calls) or an Err chunk, and all sends honor ctx cancellation —
// the same contract the agent loop relies on for the local path.
func (c *OpenAIClient) ChatStream(ctx context.Context, msgs []Message, tools []ToolSpec) (<-chan Chunk, error) {
	body := map[string]any{
		"model":    c.Model,
		"messages": toWire(msgs),
		"stream":   true,
	}
	// Omit the tools key when empty — a null/empty tools field makes some
	// OpenAI-compatible servers (llama.cpp) fail tool-call parser generation,
	// and omitting it is the correct OpenAI shape when no tools are offered
	// (same rule the local kronk client follows).
	if wt := toWireTools(tools); wt != nil {
		body["tools"] = wt
	}
	resp, err := c.post(ctx, "/chat/completions", body)
	if err != nil {
		return nil, err
	}

	ch := make(chan Chunk, 8)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		// Accumulate streamed tool-call fragments by index; OpenAI streams a
		// call's id/name once and its arguments in pieces. order preserves
		// first-seen index order so the assembled calls match the model's
		// intent regardless of map iteration (mirrors the kronk bridge).
		calls := map[int]*ToolCall{}
		var order []int
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				break
			}
			var ev struct {
				Choices []struct {
					Delta struct {
						Content   string `json:"content"`
						Reasoning string `json:"reasoning_content"`
						ToolCalls []struct {
							Index    int    `json:"index"`
							ID       string `json:"id"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &ev); err != nil || len(ev.Choices) == 0 {
				continue
			}
			d := ev.Choices[0].Delta
			if d.Content != "" || d.Reasoning != "" {
				select {
				case ch <- Chunk{Content: d.Content, Reasoning: d.Reasoning}:
				case <-ctx.Done():
					ch <- Chunk{Err: ctx.Err()}
					return
				}
			}
			for _, tc := range d.ToolCalls {
				acc := calls[tc.Index]
				if acc == nil {
					acc = &ToolCall{}
					calls[tc.Index] = acc
					order = append(order, tc.Index)
				}
				if tc.ID != "" {
					acc.ID = tc.ID
				}
				if tc.Function.Name != "" {
					acc.Name = tc.Function.Name
				}
				acc.Args += tc.Function.Arguments
			}
		}
		if err := sc.Err(); err != nil {
			ch <- Chunk{Err: fmt.Errorf("reading stream: %w", err)}
			return
		}
		final := Chunk{Done: true}
		for _, idx := range order {
			final.ToolCalls = append(final.ToolCalls, *calls[idx])
		}
		ch <- final
	}()
	return ch, nil
}

// Embed implements llm.Client over the OpenAI embeddings endpoint. It exists to
// satisfy the interface; Balaur's default recall path does NOT route memory
// through a remote client (see internal/knowledge), so a cloud chat model keeps
// embeddings local unless the owner separately opts in.
func (c *OpenAIClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	model := c.EmbedModel
	if model == "" {
		model = c.Model
	}
	resp, err := c.post(ctx, "/embeddings", map[string]any{
		"model": model,
		"input": texts,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decoding embeddings: %w", err)
	}
	vecs := make([][]float32, len(out.Data))
	for _, d := range out.Data {
		if d.Index < 0 || d.Index >= len(vecs) {
			return nil, fmt.Errorf("embedding index %d out of range", d.Index)
		}
		vecs[d.Index] = d.Embedding
	}
	return vecs, nil
}
