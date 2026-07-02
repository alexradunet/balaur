package web

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tasks"
)

// loadTaskNode fetches a task node by id from the nodes collection and hydrates it.
func (h *handlers) loadTaskNode(id string) (*core.Record, error) {
	return tasks.Get(h.app, id)
}

// tasks.go is the life-organization surface, now expressed as cards. The
// operational list lives in the quests card's focus (taskcards.QuestsFocus —
// a flat, rhythm-grouped task-card stack, was /tasks?view=list). The month
// calendar and forward timeline are their own cards (ucard_calendar/ucard_timeline,
// via buildCalendar/buildTimelineN in cards.go) — the future-facing mirror of the
// recap telescope. Calendar and timeline are read-only projections of the
// recurrence rules; actions live on the task cards.

// ---- card + transitions ----

// taskCard loads one task card as a standalone SSE fragment (plan 093: the
// quests artifact is a flat stack; no rail/detail pane).
func (h *handlers) taskCard(e *core.RequestEvent) error {
	rec, err := h.loadTaskNode(e.Request.PathValue("id"))
	if err != nil {
		return h.cardError(e, err)
	}
	html, err := h.taskCardHTML(rec)
	if err != nil {
		return e.InternalServerError("rendering task card", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	patchOuterHTML(sse, "tcard-"+rec.Id, html)
	return nil
}

// taskCardHTML renders one task as its gomponents card (port of card-task.html)
// to a string, for embedding in an SSE patch.
func (h *handlers) taskCardHTML(rec *core.Record) (string, error) {
	return renderNodeHTML(taskcards.TaskCard(taskcards.TaskViewOf(rec, time.Now().In(store.OwnerLocation(h.app))))), nil
}

func (h *handlers) taskTransition(e *core.RequestEvent) error {
	rec, err := h.loadTaskNode(e.Request.PathValue("id"))
	if err != nil {
		return h.cardError(e, err)
	}
	now := time.Now().In(store.OwnerLocation(h.app))
	switch e.Request.FormValue("to") {
	case "done":
		if _, err := tasks.Done(h.app, rec, now); err != nil {
			return h.cardError(e, err)
		}
	case "dropped":
		if err := tasks.Drop(h.app, rec); err != nil {
			return h.cardError(e, err)
		}
	case "snooze":
		until, err := snoozeUntil(e.Request.FormValue("until"), now)
		if err != nil {
			return h.cardError(e, err)
		}
		if err := tasks.Snooze(h.app, rec, until); err != nil {
			return h.cardError(e, err)
		}
	default:
		return e.BadRequestError("unknown transition", nil)
	}
	rec, err = h.loadTaskNode(rec.Id)
	if err != nil {
		return h.cardError(e, err)
	}

	// All validation passed — open the SSE patch stream.
	sse := datastar.NewSSE(e.Response, e.Request)

	// Owner-action feedback (plan 174 S7): a toast into the body-level region.
	switch e.Request.FormValue("to") {
	case "done":
		emitToast(sse, "success", "Marked done.")
	case "dropped":
		emitToast(sse, "info", "Dropped.")
	case "snooze":
		emitToast(sse, "info", "Snoozed.")
	}

	// A compact board row (the today/quests card ✓) removes itself outright.
	// The caller names its source (a validated enum) so the row id is built
	// server-side — we never trust a free-form selector from the form.
	if src := e.Request.FormValue("src"); src == "today" || src == "quests" {
		_ = sse.PatchElements("",
			datastar.WithSelectorID("urow-"+src+"-"+rec.Id), datastar.WithModeRemove())
		return nil
	}

	// Otherwise the full task card replaces itself in place (#tcard-{id}).
	html, err := h.taskCardHTML(rec)
	if err != nil {
		return e.InternalServerError("rendering task card", err)
	}
	patchOuterHTML(sse, "tcard-"+rec.Id, html)
	return nil
}

// taskEdit applies the inline edit form (title, due, recurrence, notes) and
// re-renders the card in place. Mirrors knowledgeEdit. The form always carries
// the full visible field set, so each is a deliberate value: an empty due
// clears it (re-anchoring a recurring task to its next occurrence in
// tasks.Update); recur_from_done is not editable here and stays untouched.
func (h *handlers) taskEdit(e *core.RequestEvent) error {
	rec, err := h.loadTaskNode(e.Request.PathValue("id"))
	if err != nil {
		return h.cardError(e, err)
	}
	title := e.Request.FormValue("title")
	notes := e.Request.FormValue("notes")
	recur := e.Request.FormValue("recur")
	opts := tasks.UpdateOpts{Title: &title, Notes: &notes, Recur: &recur, SetDue: true}
	if v := strings.TrimSpace(e.Request.FormValue("due")); v != "" {
		due, err := parseLocalDue(v, store.OwnerLocation(h.app))
		if err != nil {
			return h.cardError(e, err)
		}
		opts.Due = due
	}
	if err := tasks.Update(h.app, rec, time.Now().In(store.OwnerLocation(h.app)), opts); err != nil {
		return h.cardError(e, err)
	}
	html, err := h.taskCardHTML(rec)
	if err != nil {
		return e.InternalServerError("rendering task card", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	patchOuterHTML(sse, "tcard-"+rec.Id, html)
	return nil
}

// parseLocalDue reads a datetime-local form value (minutes, optionally seconds)
// in loc. The browser emits "2006-01-02T15:04"; the seconds layout is a
// belt-and-suspenders for agents that include them.
func parseLocalDue(s string, loc *time.Location) (time.Time, error) {
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05"} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("bad due time %q", s)
}

// snoozeUntil maps the card's quick picks to concrete times.
func snoozeUntil(pick string, now time.Time) (time.Time, error) {
	switch pick {
	case "1h":
		return now.Add(time.Hour), nil
	case "tonight":
		t := time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, now.Location())
		if !t.After(now) {
			t = now.Add(time.Hour) // evening already: an hour of quiet instead
		}
		return t, nil
	case "tomorrow":
		return time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()).AddDate(0, 0, 1), nil
	}
	return time.Time{}, fmt.Errorf("unknown snooze pick %q", pick)
}

// ---- nudge polling ----

// chatNudges returns agent-initiated messages (origin "nudge" or "briefing")
// newer than `since` (unix millis) as out-of-band fragments: the messages
// append to #chat and the poller replaces itself with an advanced cursor.
// Runtime artifacts the honesty check writes during a turn (origin
// "uncommitted"/"check") are deliberately excluded — the streamed turn already
// renders those, so polling must not re-append them.
func (h *handlers) chatNudges(e *core.RequestEvent) error {
	ms, err := strconv.ParseInt(e.Request.URL.Query().Get("since"), 10, 64)
	if err != nil {
		return e.BadRequestError("bad since", err)
	}
	recs, err := h.app.FindRecordsByFilter("messages",
		"(origin = 'nudge' || origin = 'briefing') && created > {:since}", "@rowid", 20, 0,
		dbx.Params{"since": store.PBTime(time.UnixMilli(ms))})
	if err != nil {
		return e.InternalServerError("loading nudges", err)
	}
	if len(recs) == 0 {
		return nil // nothing new — the poller keeps its cursor
	}
	last := recs[len(recs)-1].GetDateTime("created").Time().UnixMilli()
	// Append the new agent messages to the chat and advance the poller's cursor
	// signal so the next poll only asks for what's newer.
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(h.renderMessages(h.messageViews(recs))), datastar.WithSelectorID("chat"), datastar.WithModeAppend())
	_ = sse.MarshalAndPatchSignals(struct {
		NudgeSince int64 `json:"nudgeSince"`
	}{last})
	return nil
}
