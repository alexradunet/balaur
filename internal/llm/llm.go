// Package llm defines Balaur's single seam to language models: the Client
// interface (ChatStream + Embed) the agent loop talks to. Everything above this
// package is provider-agnostic. Two implementations sit behind the seam: the
// local default — a GGUF model run in-process via the embedded Kronk engine
// (internal/kronk) — and an opt-in remote one (OpenAIClient, this package) that
// speaks the OpenAI-compatible HTTP API. The remote path is never the default
// and is reached only by the owner's explicit, consented model selection.
package llm

import "context"

// Message is one turn of a conversation in provider-neutral form.
type Message struct {
	Role       string     `json:"role"` // system | user | assistant | tool
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // set on role=tool replies
}

// ToolCall is a model request to invoke a named tool with JSON arguments.
type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"args"` // raw JSON object
}

// ToolSpec describes a callable tool in OpenAI function-tool form.
type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema object
}

// Chunk is one streamed increment of a model reply.
type Chunk struct {
	Content   string     // text delta, may be empty
	Reasoning string     // thinking delta, may be empty
	ToolCalls []ToolCall // complete tool calls, delivered once known
	Done      bool       // final chunk
	Err       error      // terminal error; stream ends after this
}

// Client is the one interface the agent loop talks to.
type Client interface {
	// ChatStream sends the conversation and streams the reply. The returned
	// channel is closed after a Done or Err chunk. Implementations must
	// honor ctx cancellation.
	ChatStream(ctx context.Context, msgs []Message, tools []ToolSpec) (<-chan Chunk, error)

	// Embed returns one embedding vector per input text.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
