package cli

import (
	"context"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/tools"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/verify"
)

// chatClients resolves the model client for chat turns — the same active
// choice the web chatbar uses. A package var so tests inject a fake
// client (tests never hit a real model — AGENTS.md).
var chatClients = func(app core.App) (llm.Client, error) {
	// The CLI runs outside OnServe, so the serve-time Kronk engine is absent;
	// create one here (native runtime + model load stay lazy until inference).
	eng := kronk.FromStore(app)
	if eng == nil {
		eng = kronk.NewEngine(kronk.LibPath(), kronk.Processor())
		app.Store().Set(kronk.StoreKey, eng)
	}
	src := &turn.ClientSource{Engine: eng}
	return src.Active(app)
}

// toolEvent is one tool round of the turn, paired start→result. Proposal
// markers are parsed out the way the web gateway parses them into cards:
// the model-facing text lands in Result, the record reference in Proposal.
type toolEvent struct {
	Tool     string         `json:"tool"`
	Args     string         `json:"args"`
	Result   string         `json:"result"`
	IsError  bool           `json:"is_error"`
	Proposal map[string]any `json:"proposal,omitempty"`
}

func chatCmd(app core.App) *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "chat <message>",
		Short: "Run one real companion turn (model + tools) and report it as JSON",
		Long: "chat sends one owner message through the same turn pipeline the web UI\n" +
			"runs (internal/turn): context assembly, the agent loop, the runtime\n" +
			"honesty check, persistence into the master conversation. The JSON\n" +
			"report carries the reply, every tool call with its result, and the\n" +
			"words-vs-deeds verdict — enough for an external harness to assert on\n" +
			"behavior, not just text.",
		Args: cobra.ExactArgs(1),
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "turn deadline (local models can be slow)")
	cmd.RunE = run(app, "chat", func(cmd *cobra.Command, args []string) (any, error) {
		client, err := chatClients(app)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()

		events := []toolEvent{}
		pending := map[string]int{}
		emit := func(ev agent.Event) {
			switch ev.Kind {
			case "tool_start":
				pending[ev.CallID] = len(events)
				events = append(events, toolEvent{Tool: ev.Tool, Args: ev.Text})
			case "tool_result":
				if i, ok := pending[ev.CallID]; ok {
					text := ev.Text
					if _, r, ok := tools.ParseRefresh(text); ok {
						text = r // drop the live-refresh marker; CLI has no UI to patch
					}
					kind, id, rest, marked := tools.ParseProposal(text)
					events[i].Result = rest
					events[i].IsError = strings.HasPrefix(rest, "error:")
					if marked {
						events[i].Proposal = map[string]any{"kind": kind, "id": id}
					}
				}
			}
		}

		res, runErr := turn.Run(ctx, app, client, args[0], emit)
		if runErr != nil {
			return nil, runErr
		}

		memories := make([]map[string]any, 0, len(res.UsedMemories))
		for _, m := range res.UsedMemories {
			memories = append(memories, map[string]any{
				"id":    m.Id,
				"title": m.GetString("title"),
			})
		}
		return map[string]any{
			"reply":             res.Reply,
			"tools":             events,
			"claims_capture":    verify.ClaimsCapture(res.Reply),
			"capture_succeeded": verify.CaptureSucceeded(res.Turn),
			"check_note":        res.CheckNote,
			"used_memories":     memories,
			"messages_appended": len(res.Turn),
		}, nil
	})
	return cmd
}
