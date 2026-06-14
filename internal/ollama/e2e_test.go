package ollama

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/llm"
)

// TestE2EToolCall runs a real tool-calling round against a live Ollama on the
// default CPU model. Opt-in: BALAUR_OLLAMA_E2E=1 with `ollama serve` up and
// `ollama pull gemma4:e4b` done. It asserts the model emits a structured
// tool_call the agent loop can consume — the contract openai_test.go locks in.
func TestE2EToolCall(t *testing.T) {
	if os.Getenv("BALAUR_OLLAMA_E2E") != "1" {
		t.Skip("set BALAUR_OLLAMA_E2E=1 with a live Ollama + gemma4:e4b to run")
	}
	client := NewClient(ChatModel())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	tools := []llm.ToolSpec{{
		Name:        "get_weather",
		Description: "Get the current weather for a city",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{"city": map[string]any{"type": "string"}},
			"required":   []string{"city"},
		},
	}}
	msgs := []llm.Message{{Role: "user", Content: "Use the get_weather tool for Paris."}}
	ch, err := client.ChatStream(ctx, msgs, tools)
	if err != nil {
		t.Fatal(err)
	}
	var sawToolCall bool
	var toolArgs string
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("stream error: %v", chunk.Err)
		}
		if len(chunk.ToolCalls) > 0 {
			sawToolCall = true
			if chunk.ToolCalls[0].Name != "get_weather" {
				t.Errorf("tool name = %q", chunk.ToolCalls[0].Name)
			}
			toolArgs = chunk.ToolCalls[0].Args
		}
	}
	if !sawToolCall {
		t.Fatal("model did not emit a structured tool_call for an explicit tool request")
	}
	// The arguments must be valid JSON carrying the city we asked about.
	var args map[string]any
	if err := json.Unmarshal([]byte(toolArgs), &args); err != nil {
		t.Fatalf("tool args not valid JSON: %q (%v)", toolArgs, err)
	}
	city, _ := args["city"].(string)
	if !strings.Contains(strings.ToLower(city), "paris") {
		t.Fatalf("city arg = %q, want it to contain Paris (full args: %q)", city, toolArgs)
	}
}
