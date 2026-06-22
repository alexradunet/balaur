package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/store"
)

// LifeTools gives the model the owner-defined tracking verbs. Nothing is
// predefined: the owner invents kinds in conversation, and the tools only
// provide the infrastructure. The owner's statement is the consent.
func LifeTools(app core.App) []agent.Tool {
	return []agent.Tool{
		logEntryTool(app),
		entrySeriesTool(app),
		entryDropTool(app),
	}
}

// clipN bounds a short human line (os.go's clip guards whole tool outputs).
func clipN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func logEntryTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("log_entry",
			"Log something the owner tracks about their life — a measurement, a practice, a milestone. "+
				"The owner invents the kinds; reuse existing ones (see entry_series without a kind) before coining a new one.",
			obj(map[string]any{
				"kind":      str("Short tracker name, e.g. weight, mood, sleep, pages-read. Owner-defined."),
				"value_num": map[string]any{"type": "number", "description": "The numeric value when there is one (82.5, 7, 6.5). Omit for text-only entries — do not send 0."},
				"unit":      str("Optional unit for the number: kg, h, km, reps."),
				"text":      str("Optional human line — the owner's words or useful detail."),
				"details":   map[string]any{"type": "object", "description": "Optional structured extras, e.g. {\"exercise\":\"bench\",\"sets\":3,\"reps\":5}."},
				"noted_at":  str("Optional backdate: " + DueFormats + ". Defaults to now."),
			}, "kind")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Kind     string         `json:"kind"`
				ValueNum float64        `json:"value_num"`
				Unit     string         `json:"unit"`
				Text     string         `json:"text"`
				Details  map[string]any `json:"details"`
				NotedAt  string         `json:"noted_at"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("log_entry: bad arguments: %w", err)
			}
			loc := store.OwnerLocation(app)
			notedAt, _, err := ParseDue(args.NotedAt, loc)
			if err != nil {
				return "", fmt.Errorf("log_entry: %w", err)
			}
			rec, err := life.Log(app, life.LogOpts{
				Kind: args.Kind, ValueNum: args.ValueNum, Unit: args.Unit,
				Text: args.Text, Details: args.Details, NotedAt: notedAt,
			})
			if err != nil {
				return "", fmt.Errorf("log_entry: %w", err)
			}
			var b strings.Builder
			fmt.Fprintf(&b, "Logged %s", rec.GetString("kind"))
			if v := rec.GetFloat("value_num"); v != 0 {
				fmt.Fprintf(&b, ": %g %s", v, rec.GetString("unit"))
			} else if t := rec.GetString("text"); t != "" {
				fmt.Fprintf(&b, ": %s", clipN(t, 80))
			}
			fmt.Fprintf(&b, " (%s). id: %s",
				rec.GetDateTime("noted_at").Time().In(loc).Format("Jan 2 15:04"), rec.Id)
			return b.String(), nil
		},
	}
}

func entrySeriesTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("entry_series",
			"Read the owner's life log. Without a kind: the inventory of what they track. "+
				"With a kind: the recent series — numeric summary and last entries.",
			obj(map[string]any{
				"kind": str("Tracker name; omit to list all kinds the owner uses."),
				"days": map[string]any{"type": "integer", "description": "Lookback window in days, default 30."},
			})),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Kind string `json:"kind"`
				Days int    `json:"days"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("entry_series: bad arguments: %w", err)
			}
			if args.Days <= 0 {
				args.Days = 30
			}
			loc := store.OwnerLocation(app)

			if strings.TrimSpace(args.Kind) == "" {
				kinds, err := life.Kinds(app)
				if err != nil {
					return "", fmt.Errorf("entry_series: %w", err)
				}
				if len(kinds) == 0 {
					return "The owner tracks nothing yet. Kinds are theirs to invent.", nil
				}
				var b strings.Builder
				b.WriteString("The owner's trackers:\n")
				for _, k := range kinds {
					fmt.Fprintf(&b, "- %s — %d entries", k.Kind, k.Count)
					if k.NumCount > 0 {
						b.WriteString(", numeric")
						if k.Unit != "" {
							fmt.Fprintf(&b, " (%s)", k.Unit)
						}
					}
					if !k.Last.IsZero() {
						fmt.Fprintf(&b, ", last %s", k.Last.In(loc).Format("Jan 2"))
					}
					b.WriteString("\n")
				}
				return b.String(), nil
			}

			now := time.Now().In(loc)
			recs, err := life.Series(app, args.Kind, now.AddDate(0, 0, -args.Days))
			if err != nil {
				return "", fmt.Errorf("entry_series: %w", err)
			}
			if len(recs) == 0 {
				return fmt.Sprintf("No %q entries in the last %d days.", life.NormalizeKind(args.Kind), args.Days), nil
			}
			var b strings.Builder
			if s := life.Summarize(recs); s.Points > 0 {
				fmt.Fprintf(&b, "%s over %dd: last %g %s (min %g, max %g, first %g, change %+.4g)\n",
					life.NormalizeKind(args.Kind), args.Days, s.Last, s.Unit, s.Min, s.Max, s.First, s.Last-s.First)
			}
			from := max(len(recs)-5, 0)
			for _, r := range recs[from:] {
				fmt.Fprintf(&b, "- [%s] %s", r.Id, r.GetDateTime("noted_at").Time().In(loc).Format("Jan 2 15:04"))
				if v := r.GetFloat("value_num"); v != 0 {
					fmt.Fprintf(&b, " — %g %s", v, r.GetString("unit"))
				}
				if t := r.GetString("text"); t != "" {
					fmt.Fprintf(&b, " — %s", clipN(t, 80))
				}
				b.WriteString("\n")
			}
			return b.String(), nil
		},
	}
}

func entryDropTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("entry_drop",
			"Delete one mistaken life-log entry (a correction). Get the id from entry_series.",
			obj(map[string]any{
				"id": str("Entry id from entry_series or a log_entry confirmation."),
			}, "id")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("entry_drop: bad arguments: %w", err)
			}
			kind, err := life.Drop(app, args.ID)
			if err != nil {
				return "", fmt.Errorf("entry_drop: %w", err)
			}
			return fmt.Sprintf("Dropped one %s entry.", kind), nil
		},
	}
}
