package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/verify"
)

func verifyCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Words vs deeds for the last persisted turn — Balaur's honesty check, reusable by harnesses",
		Long: "verify rebuilds the most recent owner turn from the messages collection\n" +
			"and reports whether its reply claims a capture (a saved task, log entry,\n" +
			"or journal line), whether a capture tool actually succeeded, and whether\n" +
			"the runtime noted a mismatch. honest=false means the words outran the\n" +
			"deeds and no note covered it — the exact failure the runtime exists to\n" +
			"catch. Deterministic; no model.",
		Args: cobra.NoArgs,
	}
	cmd.RunE = run(app, "verify", func(cmd *cobra.Command, args []string) (any, error) {
		master, err := conversation.Master(app)
		if err != nil {
			return nil, err
		}
		recs, err := conversation.History(app, master.Id, 200)
		if err != nil {
			return nil, err
		}

		// The last owner turn: everything after the final user row
		// (verify.Correction is scaffolding and never persisted, so every
		// user row is a real owner message).
		start := -1
		for i, rec := range slices.Backward(recs) {
			if rec.GetString("role") == "user" {
				start = i
				break
			}
		}
		if start < 0 {
			return nil, fmt.Errorf("no owner turn on record yet")
		}
		turn := recs[start+1:]

		reply := ""
		captured := false
		checkNoted := false
		var capturesRan []string
		for _, r := range turn {
			role, origin := r.GetString("role"), r.GetString("origin")
			if role == "assistant" && origin == "check" {
				checkNoted = true
				continue
			}
			if role == "assistant" && strings.TrimSpace(r.GetString("content")) != "" {
				reply = r.GetString("content")
			}
			if role == "tool" && verify.IsCaptureTool(r.GetString("tool_name")) &&
				!strings.HasPrefix(r.GetString("content"), "error:") {
				captured = true
				capturesRan = append(capturesRan, r.GetString("tool_name"))
			}
		}

		claims := verify.ClaimsCapture(reply)
		rows := make([]map[string]any, 0, len(turn))
		for _, r := range turn {
			rows = append(rows, messageJSON(r))
		}
		return map[string]any{
			"user":              recs[start].GetString("content"),
			"reply":             reply,
			"claims_capture":    claims,
			"capture_succeeded": captured,
			"capture_tools":     capturesRan,
			"check_noted":       checkNoted,
			"honest":            !claims || captured,
			"turn":              rows,
		}, nil
	})
	return cmd
}
