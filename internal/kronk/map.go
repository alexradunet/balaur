package kronk

import (
	"github.com/ardanlabs/kronk/sdk/kronk/model"

	"github.com/alexradunet/balaur/internal/llm"
)

// toKronkMessages maps Balaur's provider-neutral messages to the OpenAI-shaped
// message objects Kronk's chat template parser expects. Kronk's role strings
// match Balaur's (user/assistant/system/tool), so roles pass through unchanged.
func toKronkMessages(msgs []llm.Message) []model.D {
	out := make([]model.D, 0, len(msgs))
	for _, m := range msgs {
		switch {
		case m.Role == "assistant" && len(m.ToolCalls) > 0:
			d := model.D{"role": "assistant", "tool_calls": toWireToolCalls(m.ToolCalls)}
			if m.Content != "" {
				d["content"] = m.Content
			}
			out = append(out, d)
		case m.Role == "tool":
			out = append(out, model.D{
				"role":         "tool",
				"tool_call_id": m.ToolCallID,
				"content":      m.Content,
			})
		default:
			out = append(out, model.D{"role": m.Role, "content": m.Content})
		}
	}
	return out
}

// toWireToolCalls renders assistant tool calls in OpenAI function-call form.
func toWireToolCalls(calls []llm.ToolCall) []map[string]any {
	out := make([]map[string]any, len(calls))
	for i, tc := range calls {
		out[i] = map[string]any{
			"id":       tc.ID,
			"type":     "function",
			"function": map[string]any{"name": tc.Name, "arguments": tc.Args},
		}
	}
	return out
}

// toKronkTools renders tool specs as the OpenAI function-tool array.
func toKronkTools(tools []llm.ToolSpec) []map[string]any {
	out := make([]map[string]any, len(tools))
	for i, t := range tools {
		out[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		}
	}
	return out
}
