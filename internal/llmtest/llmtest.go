// Package llmtest provides a shared scripted llm.Client fake so every
// package's tests script the model the same way; tests never hit a real
// model (AGENTS.md).
package llmtest

import (
	"context"
	"sync"

	"github.com/alexradunet/balaur/internal/llm"
)

// Reply is one scripted model response.
type Reply struct {
	Text  string
	Calls []llm.ToolCall
}

// Text returns a Reply that emits a text delta.
func Text(s string) Reply { return Reply{Text: s} }

// ToolCall returns a Reply that delivers one tool call in the Done chunk.
func ToolCall(id, name, args string) Reply {
	return Reply{Calls: []llm.ToolCall{{ID: id, Name: name, Args: args}}}
}

// ScriptedClient replays scripted replies, one per ChatStream call.
// Calls counts invocations. If Err is set, ChatStream returns it immediately.
type ScriptedClient struct {
	mu      sync.Mutex
	replies []Reply
	Calls   int
	Err     error
	// Respond, when non-nil, is called instead of consuming the replies queue.
	// The returned string is emitted as a text delta.
	Respond func(msgs []llm.Message) string
}

// New creates a ScriptedClient with the given reply sequence.
func New(replies ...Reply) *ScriptedClient {
	return &ScriptedClient{replies: replies}
}

func (f *ScriptedClient) ChatStream(ctx context.Context, msgs []llm.Message, tools []llm.ToolSpec) (<-chan llm.Chunk, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls++
	if f.Err != nil {
		return nil, f.Err
	}
	ch := make(chan llm.Chunk, 2)
	if f.Respond != nil {
		text := f.Respond(msgs)
		go func() {
			if text != "" {
				ch <- llm.Chunk{Content: text}
			}
			ch <- llm.Chunk{Done: true}
			close(ch)
		}()
		return ch, nil
	}
	if len(f.replies) == 0 {
		ch <- llm.Chunk{Done: true}
		close(ch)
		return ch, nil
	}
	r := f.replies[0]
	f.replies = f.replies[1:]
	if r.Text != "" {
		ch <- llm.Chunk{Content: r.Text}
	}
	ch <- llm.Chunk{Done: true, ToolCalls: r.Calls}
	close(ch)
	return ch, nil
}

func (f *ScriptedClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}
