package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/heads"
	"github.com/alexradunet/balaur/internal/store"
)

// HeadsTools gives the model verbs over the persona roster: switch the active
// head, create a custom head, or delete one. Heads are a capability FILTER, not a
// privilege grant — every head runs at the owner's full trust — so switching is
// benign and audited. These are wired as always-on core tools (not gated behind a
// capability group), so a scoped head can still switch back. Every mutation
// audits actor=model after the write.
func HeadsTools(app core.App) []agent.Tool {
	return []agent.Tool{
		headSwitchTool(app),
		headCreateTool(app),
		headDeleteTool(app),
	}
}

func headSwitchTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("head_switch",
			"Switch the active head (persona). Takes effect on the NEXT turn — this "+
				"turn's tools are already fixed. Use a head id or built-in key (balaur, "+
				"scholar, planner, coach, or a custom head id).",
			obj(map[string]any{"id": str("Head id or built-in key to make active.")}, "id")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("head_switch: bad arguments: %w", err)
			}
			id := strings.TrimSpace(args.ID)
			head, ok := heads.Find(app, id)
			if !ok {
				return "", fmt.Errorf("head_switch: no head %q", id)
			}
			if err := heads.SetActive(app, "model", id); err != nil {
				return "", fmt.Errorf("head_switch: %w", err)
			}
			return fmt.Sprintf("Active head set to %q — it takes effect on the next turn.", head.Name), nil
		},
	}
}

func headCreateTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("head_create",
			"Create a custom head (persona): a name, a one-line purpose, an optional "+
				"balaur-NN avatar key, and an optional capability-group filter (empty = all tools).",
			obj(map[string]any{
				"name":    str("Short display name for the persona."),
				"purpose": str("One-line description of what this persona is for."),
				"avatar":  str("Optional balaur-NN avatar key (e.g. balaur-04); empty uses the default."),
				"groups":  map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": heads.Groups}, "description": "Optional capability groups to scope the head; empty = all tools."},
			}, "name")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Name    string   `json:"name"`
				Purpose string   `json:"purpose"`
				Avatar  string   `json:"avatar"`
				Groups  []string `json:"groups"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("head_create: bad arguments: %w", err)
			}
			if strings.TrimSpace(args.Name) == "" {
				return "", fmt.Errorf("head_create: name is required")
			}
			for _, gp := range args.Groups {
				if !slices.Contains(heads.Groups, gp) {
					return "", fmt.Errorf("head_create: unknown capability group %q (valid: %s)", gp, strings.Join(heads.Groups, ", "))
				}
			}
			if args.Avatar != "" && !store.ValidBalaurAvatarKey(args.Avatar) {
				return "", fmt.Errorf("head_create: %q is not a valid balaur avatar key", args.Avatar)
			}
			id, err := heads.Create(app, "model", args.Name, args.Purpose, args.Avatar, args.Groups)
			if err != nil {
				return "", fmt.Errorf("head_create: %w", err)
			}
			return fmt.Sprintf("Created head %q (id %s).", args.Name, id), nil
		},
	}
}

func headDeleteTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("head_delete",
			"Delete a custom head by id. Built-in heads (balaur, scholar, planner, coach) "+
				"cannot be deleted.",
			obj(map[string]any{"id": str("Custom head id to delete.")}, "id")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("head_delete: bad arguments: %w", err)
			}
			id := strings.TrimSpace(args.ID)
			head, ok := heads.Find(app, id)
			if !ok {
				return "", fmt.Errorf("head_delete: no head %q", id)
			}
			if head.BuiltIn {
				return "", fmt.Errorf("head_delete: %q is a built-in head and cannot be deleted", head.Name)
			}
			if err := heads.Delete(app, "model", id); err != nil {
				return "", fmt.Errorf("head_delete: %w", err)
			}
			return fmt.Sprintf("Deleted head %q.", head.Name), nil
		},
	}
}
