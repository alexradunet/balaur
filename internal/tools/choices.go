package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
)

// ChoicesMarker prefixes tool results that carry dialogue choices, so the web
// layer can render a live choice panel instead of a plain tool row. Format:
// marker + JSON head (prompt + choices array) + newline + model-facing text.
const ChoicesMarker = "\x00balaur-choices:"

// Choice is one option in an offer_choices result.
type Choice struct {
	Label string `json:"label"`          // shown to the owner
	Hint  string `json:"hint,omitempty"` // mono hint, e.g. "add recurring task"
}

// choicesHead is the self-describing JSON head line embedded in a marked result.
type choicesHead struct {
	Prompt  string   `json:"prompt"`
	Choices []Choice `json:"choices"`
}

// MarkChoices builds a marked choices result.
// Format: marker + JSON(choicesHead) + "\n" + modelText.
func MarkChoices(prompt string, choices []Choice, modelText string) string {
	head := choicesHead{Prompt: prompt, Choices: choices}
	b, _ := json.Marshal(head)
	return ChoicesMarker + string(b) + "\n" + modelText
}

// ParseChoices splits a marked choices result. ok is false for ordinary text.
func ParseChoices(s string) (prompt string, choices []Choice, modelText string, ok bool) {
	if !strings.HasPrefix(s, ChoicesMarker) {
		return "", nil, s, false
	}
	s = strings.TrimPrefix(s, ChoicesMarker)
	headLine, rest, _ := strings.Cut(s, "\n")
	var head choicesHead
	if err := json.Unmarshal([]byte(headLine), &head); err != nil {
		return "", nil, rest, false
	}
	return head.Prompt, head.Choices, rest, true
}

// ChoiceTools returns the offer_choices tool.
func ChoiceTools(app core.App) []agent.Tool {
	return []agent.Tool{offerChoicesTool(app)}
}

func offerChoicesTool(_ core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("offer_choices",
			"Offer the owner 2–5 concrete reply choices when a decision has clear options. "+
				"The owner may click one — it arrives as their next message — or type anything else. "+
				"Do not use it for open-ended questions.",
			obj(map[string]any{
				"prompt": str("Kicker line shown above the choices. Defaults to \"Your word\" if omitted."),
				"choices": map[string]any{
					"type":        "array",
					"description": "2–5 concrete reply options.",
					"minItems":    2,
					"maxItems":    5,
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"label": map[string]any{"type": "string", "description": "Choice text shown to the owner."},
							"hint":  map[string]any{"type": "string", "description": "Optional mono hint line, e.g. 'add recurring task'."},
						},
						"required": []string{"label"},
					},
				},
			}, "choices")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Prompt  string   `json:"prompt"`
				Choices []Choice `json:"choices"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("offer_choices: bad arguments: %w", err)
			}
			if args.Prompt == "" {
				args.Prompt = "Your word"
			}
			if len(args.Choices) < 2 {
				return fmt.Sprintf("offer_choices: need at least 2 choices, got %d", len(args.Choices)), nil
			}
			if len(args.Choices) > 5 {
				return fmt.Sprintf("offer_choices: need at most 5 choices, got %d", len(args.Choices)), nil
			}

			// Build model-facing enumeration.
			var b strings.Builder
			fmt.Fprint(&b, "offered choices:")
			for i, c := range args.Choices {
				fmt.Fprintf(&b, " %d) %s", i+1, c.Label)
			}
			modelText := b.String()

			return MarkChoices(args.Prompt, args.Choices, modelText), nil
		},
	}
}
