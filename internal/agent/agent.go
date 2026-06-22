// Package agent implements Balaur's hand-rolled conversation loop:
// messages → model → tool calls → tool results → model … until the model
// answers in plain text. No framework; the loop is the product and stays
// inspectable.
package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/alexradunet/balaur/internal/llm"
)

// Tool is a capability the model may invoke. Execute receives the raw JSON
// arguments and returns the tool output as text shown to the model.
type Tool struct {
	Spec    llm.ToolSpec
	Execute func(ctx context.Context, argsJSON string) (string, error)
}

// ToolSpecOf is a small constructor so tool packages don't import llm
// directly for the common case.
func ToolSpecOf(name, description string, parameters map[string]any) llm.ToolSpec {
	return llm.ToolSpec{Name: name, Description: description, Parameters: parameters}
}

// Event is one observable step of a run, delivered to the caller for
// rendering (SSE → HTMX fragments) and persistence.
type Event struct {
	Kind   string // "text" | "reasoning" | "tool_start" | "tool_result" | "done" | "error"
	Text   string // delta or tool output
	Tool   string // tool name for tool_* events
	CallID string
	Err    error
}

// Loop drives one user turn to completion.
type Loop struct {
	Client   llm.Client
	Tools    []Tool
	MaxSteps int // tool-call rounds before forcing a plain answer; 0 = 8
}

func (l *Loop) maxSteps() int {
	if l.MaxSteps > 0 {
		return l.MaxSteps
	}
	return 8
}

func (l *Loop) findTool(name string) *Tool {
	for i := range l.Tools {
		if l.Tools[i].Spec.Name == name {
			return &l.Tools[i]
		}
	}
	return nil
}

func (l *Loop) specs() []llm.ToolSpec {
	out := make([]llm.ToolSpec, 0, len(l.Tools))
	for _, t := range l.Tools {
		out = append(out, t.Spec)
	}
	return out
}

// Run executes the loop. It returns the final message history (including
// assistant and tool turns appended during the run). Events stream to emit;
// emit is never called after Run returns.
func (l *Loop) Run(ctx context.Context, history []llm.Message, emit func(Event)) ([]llm.Message, error) {
	msgs := history

	for step := 0; step < l.maxSteps(); step++ {
		stream, err := l.Client.ChatStream(ctx, msgs, l.specs())
		if err != nil {
			emit(Event{Kind: "error", Err: err})
			return msgs, err
		}

		var text strings.Builder
		var calls []llm.ToolCall
		for chunk := range stream {
			if chunk.Err != nil {
				emit(Event{Kind: "error", Err: chunk.Err})
				return msgs, chunk.Err
			}
			if chunk.Content != "" {
				text.WriteString(chunk.Content)
				emit(Event{Kind: "text", Text: chunk.Content})
			}
			if chunk.Reasoning != "" {
				emit(Event{Kind: "reasoning", Text: chunk.Reasoning})
			}
			if chunk.Done {
				calls = chunk.ToolCalls
			}
		}

		msgs = append(msgs, llm.Message{Role: "assistant", Content: text.String(), ToolCalls: calls})

		if len(calls) == 0 {
			emit(Event{Kind: "done"})
			return msgs, nil
		}

		for _, call := range calls {
			emit(Event{Kind: "tool_start", Tool: call.Name, CallID: call.ID, Text: call.Args})
			result := l.executeCall(ctx, call)
			emit(Event{Kind: "tool_result", Tool: call.Name, CallID: call.ID, Text: result})
			msgs = append(msgs, llm.Message{Role: "tool", Content: result, ToolCallID: call.ID})
		}
	}

	err := fmt.Errorf("agent: exceeded %d tool rounds without a final answer", l.maxSteps())
	emit(Event{Kind: "error", Err: err})
	return msgs, err
}

func (l *Loop) executeCall(ctx context.Context, call llm.ToolCall) string {
	tool := l.findTool(call.Name)
	if tool == nil {
		return fmt.Sprintf("error: unknown tool %q", call.Name)
	}
	out, err := tool.Execute(ctx, call.Args)
	if err != nil {
		// The model sees the failure and can adapt; the caller's emit saw
		// the same string, so nothing is hidden.
		return fmt.Sprintf("error: %v", err)
	}
	return out
}
