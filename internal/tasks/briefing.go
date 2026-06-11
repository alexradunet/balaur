package tasks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
)

// The morning briefing: once per local day, after the briefing hour, Balaur
// opens the day — overdue items, today's commitments, habit streaks.
// Idempotency is DERIVED, not stored: a briefing exists for today iff an
// origin=briefing message exists since local midnight. Restart-safe,
// catch-up-safe (a box asleep at the briefing hour briefs at wake). Quiet
// days are skipped — the companion does not manufacture noise. Accepted
// consequence: a quiet morning followed by a task added mid-day produces
// the day's first briefing then; it still fires at most once.

const (
	briefingOverdueCap = 5
	briefingTodayCap   = 8
)

// BriefedToday reports whether a briefing already exists since local
// midnight of now's day.
func BriefedToday(app core.App, now time.Time) bool {
	mid := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	recs, err := app.FindRecordsByFilter("messages",
		"origin = 'briefing' && created >= {:mid}", "", 1, 0,
		dbx.Params{"mid": store.PBTime(mid)})
	return err == nil && len(recs) > 0
}

// Briefing fires the day's briefing when due: hour gate passed, not yet
// briefed today, and the day actually holds something. Text is composed by
// the model when available; the deterministic list is both the no-model
// default and the failure fallback.
func Briefing(app core.App, client llm.Client, now time.Time, hour int) error {
	if now.Hour() < hour {
		return nil
	}
	if BriefedToday(app, now) {
		return nil
	}
	recs, err := OpenTasks(app, nil)
	if err != nil {
		return err
	}
	bk := Bucket(recs, now)
	if len(bk.Overdue) == 0 && len(bk.Today) == 0 {
		return nil // quiet day
	}

	lines := dayLines(app, bk, now)
	text := deterministicBriefing(now, lines)
	if client != nil {
		if composed := composeBriefing(client, now, lines); composed != "" {
			text = composed
		}
	}

	master, err := conversation.Master(app)
	if err != nil {
		return err
	}
	if err := conversation.AppendOrigin(app, master.Id,
		llm.Message{Role: "assistant", Content: text}, "", "briefing"); err != nil {
		return err
	}
	store.Audit(app, "", "briefing", "task.briefing", "", true,
		map[string]any{"overdue": len(bk.Overdue), "today": len(bk.Today)})
	return nil
}

// dayLines renders the day's material: overdue first (most overdue first),
// then today by time. Shared by the briefing and the Today context block.
func dayLines(app core.App, bk Buckets, now time.Time) []string {
	var lines []string
	for i, r := range bk.Overdue {
		if i >= briefingOverdueCap {
			break
		}
		lines = append(lines, dayLine(app, r, now, true))
	}
	for _, r := range bk.Today {
		if len(lines) >= briefingOverdueCap+briefingTodayCap {
			break
		}
		lines = append(lines, dayLine(app, r, now, false))
	}
	return lines
}

func dayLine(app core.App, r *core.Record, now time.Time, overdue bool) string {
	var b strings.Builder
	due := r.GetDateTime("due").Time()
	if overdue {
		fmt.Fprintf(&b, "%s: %s", Lateness(due, now), r.GetString("title"))
	} else {
		fmt.Fprintf(&b, "today %s: %s", due.In(now.Location()).Format("15:04"), r.GetString("title"))
	}
	if rule, err := Parse(r.GetString("recur")); err == nil && !rule.IsZero() {
		if streak := StreakFor(app, r, now); streak > 1 {
			fmt.Fprintf(&b, " — habit, streak %d", streak)
		} else {
			b.WriteString(" — habit")
		}
	}
	if notes := compressLine(r.GetString("notes"), 60); notes != "" {
		fmt.Fprintf(&b, " (%s)", notes)
	}
	return b.String()
}

func compressLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func deterministicBriefing(now time.Time, lines []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — on the book:", now.Format("Monday, January 2"))
	for _, l := range lines {
		b.WriteString("\n- " + l)
	}
	return b.String()
}

// composeBriefing asks the model for the companion-voiced morning note.
// Returns "" on failure or a ramble — callers keep the deterministic list.
func composeBriefing(client llm.Client, now time.Time, lines []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), composeTimeout)
	defer cancel()

	msgs := []llm.Message{
		{Role: "system", Content: "You are Balaur, a wise personal companion. " +
			"Open the day with the owner: two to four short, warm, plain sentences built " +
			"from the commitments below. Mention overdue items gently, today's items with " +
			"their times, and any habit streak worth a word. " +
			"No exclamation marks, no emoji, no bullet lists, no lecturing."},
		{Role: "user", Content: now.Format("Monday, January 2") + "\n" + strings.Join(lines, "\n")},
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
	if text == "" || len(text) > 1000 {
		return ""
	}
	return text
}

// TodayBlock renders the owner's open commitments for context injection —
// the companion knows the day in every turn. Empty when nothing is on the
// book (no tokens spent on silence).
func TodayBlock(app core.App, now time.Time) string {
	recs, err := OpenTasks(app, nil)
	if err != nil {
		return ""
	}
	bk := Bucket(recs, now)
	lines := dayLines(app, bk, now)
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n## Today — the owner's open commitments\n")
	for _, l := range lines {
		b.WriteString("- " + l + "\n")
	}
	return b.String()
}
