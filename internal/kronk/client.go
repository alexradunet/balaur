package kronk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ardanlabs/kronk/sdk/kronk/model"

	"github.com/alexradunet/balaur/internal/llm"
)

// chatStreamTimeout caps a single local streaming generation. Kronk's
// ChatStreaming requires a context with a deadline; this is the default applied
// when the caller supplies none (web/CLI request contexts usually have none).
// Generous on purpose — a slow CPU generating up to max_tokens must not be cut
// off mid-reply.
const chatStreamTimeout = 10 * time.Minute

// agentTemperature is the fixed sampling temperature for local inference.
// Mirrors the constant in internal/llm/openai.go — kept separate to avoid a
// cross-package dependency for a single float.
const agentTemperature = 0.3

// Client implements llm.Client against in-process Kronk models held by an Engine.
// chatPath and embedPath are absolute GGUF file paths.
type Client struct {
	eng       *Engine
	chatPath  string
	embedPath string
}

// Client returns an llm.Client bound to the given chat and embedding GGUF files.
func (e *Engine) Client(chatPath, embedPath string) *Client {
	return &Client{eng: e, chatPath: chatPath, embedPath: embedPath}
}

// ChatStream sends the conversation to the resident chat model and bridges
// Kronk's streamed response into llm.Chunks.
func (c *Client) ChatStream(ctx context.Context, msgs []llm.Message, tools []llm.ToolSpec) (<-chan llm.Chunk, error) {
	krn, err := c.eng.chatModel(ctx, c.chatPath)
	if err != nil {
		return nil, err
	}
	// Kronk's streaming inference requires a context with a deadline. Web/CLI
	// request contexts usually have none, so add a generous cap when one is
	// absent (an existing caller deadline is honored). cancel fires when the
	// stream is fully drained — see bridge.
	var cancel context.CancelFunc = func() {}
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, chatStreamTimeout)
	}
	d := model.D{
		"messages":    toKronkMessages(msgs),
		"max_tokens":  2048,
		"temperature": agentTemperature,
	}
	// Omit the tools key when empty — llama.cpp's template parser rejects a null
	// tools field (same rule the OpenAI HTTP client follows).
	if len(tools) > 0 {
		d["tools"] = toKronkTools(tools)
	}
	src, err := krn.ChatStreaming(ctx, d)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("local chat stream: %w", err)
	}
	return bridge(ctx, src, cancel), nil
}

// Embed returns one embedding vector per input text from the resident embedding
// model. KISS first slice: one input per call (no batch index reordering).
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	krn, err := c.eng.embedModel(ctx, c.embedPath)
	if err != nil {
		return nil, err
	}
	out := make([][]float32, len(texts))
	for i, t := range texts {
		resp, err := krn.Embeddings(ctx, model.D{"input": t})
		if err != nil {
			return nil, fmt.Errorf("local embed: %w", err)
		}
		if len(resp.Data) == 0 {
			return nil, fmt.Errorf("local embed: model returned no vector for input %d", i)
		}
		out[i] = resp.Data[0].Embedding
	}
	return out, nil
}

// bridge converts Kronk's <-chan model.ChatResponse into <-chan llm.Chunk. It
// streams content/reasoning deltas, accumulates tool calls by index, and emits
// the assembled calls on a single terminal Chunk{Done:true}. Sends are guarded by
// ctx so a cancelled or gone consumer cannot wedge the goroutine.
func bridge(ctx context.Context, src <-chan model.ChatResponse, cancel context.CancelFunc) <-chan llm.Chunk {
	out := make(chan llm.Chunk, 8)
	go func() {
		defer close(out)
		defer cancel() // release the stream's deadline context once drained

		calls := map[int]*llm.ToolCall{}
		var order []int
		send := func(ch llm.Chunk) bool {
			select {
			case out <- ch:
				return true
			case <-ctx.Done():
				return false
			}
		}

		for resp := range src {
			if len(resp.Choices) == 0 {
				continue
			}
			choice := resp.Choices[0]
			msg := choice.Delta
			if msg == nil {
				msg = choice.Message
			}
			if msg != nil {
				if msg.Content != "" || msg.Reasoning != "" {
					if !send(llm.Chunk{Content: msg.Content, Reasoning: msg.Reasoning}) {
						return
					}
				}
				for _, tc := range msg.ToolCalls {
					cur, ok := calls[tc.Index]
					if !ok {
						cur = &llm.ToolCall{}
						calls[tc.Index] = cur
						order = append(order, tc.Index)
					}
					if tc.ID != "" {
						cur.ID = tc.ID
					}
					if tc.Function.Name != "" {
						cur.Name = tc.Function.Name
					}
					// Kronk delivers arguments as a parsed object. Marshal the
					// underlying map (not ToolCallArguments, whose MarshalJSON
					// emits a quoted JSON *string* per the OpenAI wire spec) so
					// llm.ToolCall.Args carries the raw JSON object Balaur expects.
					if len(tc.Function.Arguments) > 0 {
						if b, err := json.Marshal(map[string]any(tc.Function.Arguments)); err == nil {
							cur.Args = string(b)
						}
					}
				}
			}
			if choice.FinishReason() == model.FinishReasonError {
				send(llm.Chunk{Err: fmt.Errorf("local model reported an error")})
				return
			}
		}

		var assembled []llm.ToolCall
		for _, idx := range order {
			assembled = append(assembled, *calls[idx])
		}
		send(llm.Chunk{Done: true, ToolCalls: assembled})
	}()
	return out
}
