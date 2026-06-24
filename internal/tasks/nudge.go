package tasks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
)

// The nudger: a minute cron calls Nudge; everything due-and-unfired gets
// one message into the master conversation. nudged_at on the record is the
// fired-state, so firing is idempotent across restarts, and the first tick
// after downtime is the catch-up (the query finds whatever came due while
// the box slept). Same shape as the recap catch-up job.

// nudgeBatchLimit bounds one tick's pickup; the rest fires next minute.
const nudgeBatchLimit = 20

// NudgeSuppressed reports whether the owner has muted or disabled nudges via
// owner_settings — the soft, UI-driven layer above the BALAUR_NUDGE env kill
// switch (which the cron scheduler honors separately). nudge_enabled "0" turns
// them off; nudge_muted_until (RFC3339) silences them until that time. The
// manual "nudge now" control bypasses this by calling Nudge directly.
func NudgeSuppressed(app core.App, now time.Time) bool {
	if store.GetOwnerSetting(app, "nudge_enabled", "1") == "0" {
		return true
	}
	if until := store.GetOwnerSetting(app, "nudge_muted_until", ""); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil && now.Before(t) {
			return true
		}
	}
	return false
}

// composeTimeout bounds the optional model call; the deterministic line
// must never wait long on a slow model.
const composeTimeout = 60 * time.Second

// DueForNudge returns open tasks whose nudge should fire at now: due has
// passed, never fired (or re-armed by snooze/recurrence), snooze elapsed.
// Tasks are now type=task nodes; we load all active task nodes and filter in Go.
func DueForNudge(app core.App, now time.Time) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, "task", nodes.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("tasks: loading task nodes for nudge: %w", err)
	}
	var out []*core.Record
	for _, r := range recs {
		hydrate(r)
		if r.GetString("status") != "open" {
			continue
		}
		due := r.GetDateTime("due").Time()
		if due.IsZero() || !due.Before(now) {
			continue
		}
		if r.GetString("nudged_at") != "" {
			continue
		}
		if su := r.GetString("snoozed_until"); su != "" {
			suTime := r.GetDateTime("snoozed_until").Time()
			if !suTime.IsZero() && suTime.After(now) {
				continue
			}
		}
		out = append(out, r)
		if len(out) >= nudgeBatchLimit {
			break
		}
	}
	return out, nil
}

// Nudge fires every due reminder as ONE assistant message (origin=nudge) in
// the master conversation — one interruption, not four. The message text is
// model-composed in Balaur's voice when a client is available, with a
// deterministic line as both the no-model default and the failure fallback
// (deterministic, offline, free is the default — AGENTS.md). Each fired
// task is marked and audited.
func Nudge(app core.App, client llm.Client, now time.Time) error {
	recs, err := DueForNudge(app, now)
	if err != nil || len(recs) == 0 {
		return err
	}

	text := deterministicNudge(recs, now)
	if client != nil {
		if composed := composeNudge(client, recs, now); composed != "" {
			text = composed
		}
	}

	master, err := conversation.Master(app)
	if err != nil {
		return err
	}
	if err := conversation.AppendOrigin(app, master.Id,
		llm.Message{Role: "assistant", Content: text}, "", "nudge"); err != nil {
		return err
	}
	for _, rec := range recs {
		props := nodes.Props(rec)
		props["nudged_at"] = fmtTime(now.UTC())
		rec.Set("props", props)
		dehydrate(rec)
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("marking nudge on %q: %w", rec.GetString("title"), err)
		}
		hydrate(rec)
		store.Audit(app, "nudge", "task.nudge", rec.Id, true,
			map[string]any{"title": rec.GetString("title")})
	}
	return nil
}

// DueLine renders a task's due time as one owner-facing line: an overdue
// open task reads "<lateness> — was <when>", everything else "due <when>".
// status is the task's status (only "open" tasks read as overdue).
func DueLine(due, now time.Time, status string) string {
	local := due.In(now.Location())
	when := local.Format("Mon, Jan 2 at 15:04")
	if local.Before(now) && status == "open" {
		return Lateness(due, now) + " — was " + when
	}
	return "due " + when
}

// Lateness renders how a due time stands relative to now, in human terms.
func Lateness(due, now time.Time) string {
	due = due.In(now.Location())
	switch d := now.Sub(due); {
	case d < 2*time.Minute:
		return "due now"
	case d < time.Hour:
		return fmt.Sprintf("overdue %dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("overdue %dh", int(d.Hours()))
	default:
		return fmt.Sprintf("overdue %dd", int(d.Hours()/24))
	}
}

func deterministicNudge(recs []*core.Record, now time.Time) string {
	if len(recs) == 1 {
		return fmt.Sprintf("Reminder: %s — %s.",
			recs[0].GetString("title"), Lateness(recs[0].GetDateTime("due").Time(), now))
	}
	var b strings.Builder
	b.WriteString("Reminders:")
	for _, r := range recs {
		fmt.Fprintf(&b, "\n- %s (%s)", r.GetString("title"), Lateness(r.GetDateTime("due").Time(), now))
	}
	return b.String()
}

// composeNudge asks the model for a short companion-voiced reminder.
// Returns "" on any failure or an implausibly long ramble — callers keep
// the deterministic text in that case.
func composeNudge(client llm.Client, recs []*core.Record, now time.Time) string {
	ctx, cancel := context.WithTimeout(context.Background(), composeTimeout)
	defer cancel()

	var lines strings.Builder
	for _, r := range recs {
		fmt.Fprintf(&lines, "- %s (%s)", r.GetString("title"), Lateness(r.GetDateTime("due").Time(), now))
		if notes := strings.TrimSpace(r.GetString("notes")); notes != "" {
			fmt.Fprintf(&lines, " — context: %s", notes)
		}
		lines.WriteString("\n")
	}
	msgs := []llm.Message{
		{Role: "system", Content: "You are Balaur, a wise personal companion. " +
			"Remind the owner of what is due, in one or two short, warm, plain sentences. " +
			"Name each task. Use ONLY the tasks listed below — never invent tasks, " +
			"times, or streaks. No exclamation marks, no emoji, no guilt, no flattery."},
		{Role: "user", Content: "It is " + now.Format("Monday, 15:04") + ".\nDue now:\n" + lines.String()},
	}
	stream, err := client.ChatStream(ctx, msgs, nil)
	if err != nil {
		return ""
	}
	text, err := llm.Collect(stream)
	if err != nil {
		return ""
	}
	text = strings.TrimSpace(text)
	if text == "" || len(text) > 800 {
		return ""
	}
	return text
}
