package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ardanlabs/kronk/sdk/kronk"
	"github.com/ardanlabs/kronk/sdk/kronk/model"
)

// KronkClient runs models in-process through kronk (llama.cpp loaded via
// purego — no CGO). Chat and embeddings usually need different GGUF models,
// so each lazily opens its own kronk instance on first use.
//
// Trade-off (documented per AGENTS.md): in-process inference means the
// llama.cpp shared library is downloaded on first run (kronk manages this);
// the Go binary itself stays static. Pin KRONK_LIB_VERSION for stability.
type KronkClient struct {
	ChatModelFiles  []string // GGUF file path(s) for the chat model
	EmbedModelFiles []string // GGUF file path(s) for the embedding model

	mu      sync.Mutex
	chatKrn *kronk.Kronk
	embKrn  *kronk.Kronk
}

// instance lazily creates the kronk handle guarded by mu.
func (c *KronkClient) instance(slot **kronk.Kronk, files []string, what string) (*kronk.Kronk, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if *slot != nil {
		return *slot, nil
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no %s model configured", what)
	}
	if !kronk.Initialized() {
		if err := kronk.Init(); err != nil {
			return nil, fmt.Errorf("initializing kronk: %w", err)
		}
	}
	krn, err := kronk.New(model.WithModelFiles(files))
	if err != nil {
		return nil, fmt.Errorf("loading %s model: %w", what, err)
	}
	*slot = krn
	return krn, nil
}

func toKronkMessages(msgs []Message) []model.D {
	out := make([]model.D, 0, len(msgs))
	for _, m := range msgs {
		d := model.D{"role": m.Role, "content": m.Content}
		if m.ToolCallID != "" {
			d["tool_call_id"] = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			calls := make([]model.D, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				calls = append(calls, model.D{
					"id":   tc.ID,
					"type": "function",
					"function": model.D{
						"name":      tc.Name,
						"arguments": tc.Args,
					},
				})
			}
			d["tool_calls"] = calls
		}
		out = append(out, d)
	}
	return out
}

func toKronkTools(tools []ToolSpec) []model.D {
	if len(tools) == 0 {
		return nil
	}
	out := make([]model.D, 0, len(tools))
	for _, t := range tools {
		out = append(out, model.D{
			"type": "function",
			"function": model.D{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		})
	}
	return out
}

func (c *KronkClient) ChatStream(ctx context.Context, msgs []Message, tools []ToolSpec) (<-chan Chunk, error) {
	krn, err := c.instance(&c.chatKrn, c.ChatModelFiles, "chat")
	if err != nil {
		return nil, err
	}

	d := model.D{"messages": toKronkMessages(msgs)}
	if kt := toKronkTools(tools); kt != nil {
		d["tools"] = kt
	}

	src, err := krn.ChatStreaming(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("kronk chat: %w", err)
	}

	ch := make(chan Chunk, 8)
	go func() {
		defer close(ch)
		var toolCalls []ToolCall
		for resp := range src {
			if len(resp.Choices) == 0 {
				continue
			}
			choice := resp.Choices[0]
			delta := choice.Delta
			if delta == nil {
				delta = choice.Message
			}
			if delta == nil {
				continue
			}
			if delta.Content != "" || delta.Reasoning != "" {
				select {
				case ch <- Chunk{Content: delta.Content, Reasoning: delta.Reasoning}:
				case <-ctx.Done():
					ch <- Chunk{Err: ctx.Err()}
					return
				}
			}
			for _, tc := range delta.ToolCalls {
				args, err := json.Marshal(tc.Function.Arguments)
				if err != nil {
					args = []byte("{}")
				}
				toolCalls = append(toolCalls, ToolCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
					Args: string(args),
				})
			}
		}
		ch <- Chunk{Done: true, ToolCalls: toolCalls}
	}()
	return ch, nil
}

func (c *KronkClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	krn, err := c.instance(&c.embKrn, c.EmbedModelFiles, "embedding")
	if err != nil {
		return nil, err
	}
	vecs := make([][]float32, len(texts))
	// kronk's embeddings call takes one input at a time through model.D;
	// loop keeps it simple until batching proves necessary (YAGNI).
	for i, text := range texts {
		resp, err := krn.Embeddings(ctx, model.D{"input": text})
		if err != nil {
			return nil, fmt.Errorf("embedding %d: %w", i, err)
		}
		if len(resp.Data) == 0 {
			return nil, fmt.Errorf("embedding %d: empty response", i)
		}
		vecs[i] = resp.Data[0].Embedding
	}
	return vecs, nil
}
