package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/store"
)

// ProfileTools lets the model update the owner's display profile — name and
// avatars. This is reversible, audited config (not durable knowledge), so the
// writes are direct and attributed actor=model. Model/cloud selection stays
// owner-only (a hard consent gate) and is deliberately NOT a tool here.
func ProfileTools(app core.App) []agent.Tool {
	return []agent.Tool{profileSetTool(app)}
}

func profileSetTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("profile_set",
			"Update the owner's display profile: display name, soul avatar (the owner's "+
				"portrait), and/or your own balaur avatar. Only the fields you pass change.",
			obj(map[string]any{
				"display_name":  str("Optional: the owner's display name (trimmed to 60 chars)."),
				"soul_avatar":   str("Optional: a soul-NN avatar key for the owner's portrait."),
				"balaur_avatar": str("Optional: a balaur-NN avatar key for your portrait."),
			})),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				DisplayName  *string `json:"display_name"`
				SoulAvatar   *string `json:"soul_avatar"`
				BalaurAvatar *string `json:"balaur_avatar"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("profile_set: bad arguments: %w", err)
			}
			var changed []string
			if args.DisplayName != nil {
				name := strings.TrimSpace(*args.DisplayName)
				if len(name) > 60 {
					name = name[:60]
				}
				if err := store.SetOwnerSetting(app, "display_name", name); err != nil {
					return "", fmt.Errorf("profile_set: %w", err)
				}
				changed = append(changed, "display_name")
			}
			if args.SoulAvatar != nil {
				if !store.ValidSoulAvatarKey(*args.SoulAvatar) {
					return "", fmt.Errorf("profile_set: %q is not a valid soul avatar key", *args.SoulAvatar)
				}
				if err := store.SetOwnerSetting(app, "soul_avatar", *args.SoulAvatar); err != nil {
					return "", fmt.Errorf("profile_set: %w", err)
				}
				changed = append(changed, "soul_avatar")
			}
			if args.BalaurAvatar != nil {
				if !store.ValidBalaurAvatarKey(*args.BalaurAvatar) {
					return "", fmt.Errorf("profile_set: %q is not a valid balaur avatar key", *args.BalaurAvatar)
				}
				if err := store.SetOwnerSetting(app, "balaur_avatar", *args.BalaurAvatar); err != nil {
					return "", fmt.Errorf("profile_set: %w", err)
				}
				changed = append(changed, "balaur_avatar")
			}
			if len(changed) == 0 {
				return "", fmt.Errorf("profile_set: nothing to change — pass at least one field")
			}
			store.Audit(app, "model", "profile.edit", "owner", true, map[string]any{"fields": changed})
			return "Updated profile: " + strings.Join(changed, ", ") + ".", nil
		},
	}
}
