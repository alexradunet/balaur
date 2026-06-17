package web

import (
	"fmt"
	"net/url"
	"sort"
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

// tasks.go is the life-organization surface, now expressed as cards. The
// operational list lives in the quests card's focus (taskcards.QuestsFocus —
// the rhythm-grouped quest rail + detail, was /tasks?view=list). The month
// calendar and forward timeline are their own cards (ucard_calendar/ucard_timeline,
// via buildCalendar/buildTimelineN in cards.go) — the future-facing mirror of the
// recap telescope. Calendar and timeline are read-only projections of the
// recurrence rules; actions live on the task cards.

// taskView is one task's template payload.
type taskView struct {
	ID, Title, Notes, Status string
	DueLine                  string
	Overdue                  bool
	RecurLine                string
}

func taskViewOf(rec *core.Record, now time.Time) taskView {
	v := taskView{
		ID:     rec.Id,
		Title:  rec.GetString("title"),
		Notes:  rec.GetString("notes"),
		Status: rec.GetString("status"),
	}
	if due := rec.GetDateTime("due").Time(); !due.IsZero() {
		local := due.In(now.Location())
		if local.Before(now) && v.Status == "open" {
			v.Overdue = true
			v.DueLine = tasks.Lateness(due, now) + " — was " + local.Format("Mon, Jan 2 at 15:04")
		} else {
			v.DueLine = "due " + local.Format("Mon, Jan 2 at 15:04")
		}
	}
	if rule, err := tasks.Parse(rec.GetString("recur")); err == nil && !rule.IsZero() {
		v.RecurLine = tasks.Describe(rule)
	}
	return v
}

// questGroup buckets an open task by rhythm: Dailies (daily / every:1d),
// Rituals (any other recurrence), Quests (one-off with due),
// Side quests (one-off without due).
// A recur string that fails to parse counts as one-off — same forgiving
// behaviour as the builder (which ignores Parse errors when setting RecurLine).
func questGroup(recur string, hasDue bool) string {
	rule, err := tasks.Parse(recur)
	if err != nil || rule.IsZero() {
		if hasDue {
			return "Quests"
		}
		return "Side quests"
	}
	// daily or every:1d → Dailies
	if rule.Kind == "daily" || (rule.Kind == "every" && rule.N == 1) {
		return "Dailies"
	}
	return "Rituals"
}

// ---- calendar ----

type calItem struct {
	Time, Title string
	Recurring   bool
}

type calCell struct {
	Day     int
	Date    string // YYYY-MM-DD — links the cell to its day page
	InMonth bool
	IsToday bool
	Items   []calItem
}

type calView struct {
	Label, PrevM, NextM string
	Weekdays            []string
	Weeks               [][]calCell
}

func buildCalendar(recs []*core.Record, monthParam string, now time.Time) calView {
	loc := now.Location()
	base := now
	if t, err := time.ParseInLocation("2006-01", monthParam, loc); err == nil {
		base = t
	}
	mStart := time.Date(base.Year(), base.Month(), 1, 0, 0, 0, 0, loc)
	mEnd := mStart.AddDate(0, 1, 0)
	gridStart := mondayOf(mStart)
	gridEnd := gridStart
	for gridEnd.Before(mEnd) {
		gridEnd = gridEnd.AddDate(0, 0, 7)
	}

	items := map[string][]calItem{}
	for _, r := range recs {
		rule, err := tasks.Parse(r.GetString("recur"))
		if err != nil {
			continue
		}
		due := r.GetDateTime("due").Time().In(loc)
		for _, occ := range tasks.Occurrences(rule, due, gridStart, gridEnd) {
			key := occ.Format("2006-01-02")
			items[key] = append(items[key], calItem{
				Time: occ.Format("15:04"), Title: r.GetString("title"), Recurring: !rule.IsZero(),
			})
		}
	}
	for k := range items {
		sort.Slice(items[k], func(i, j int) bool { return items[k][i].Time < items[k][j].Time })
	}

	today := now.Format("2006-01-02")
	var weeks [][]calCell
	for ws := gridStart; ws.Before(gridEnd); ws = ws.AddDate(0, 0, 7) {
		week := make([]calCell, 0, 7)
		for i := 0; i < 7; i++ {
			d := ws.AddDate(0, 0, i)
			key := d.Format("2006-01-02")
			week = append(week, calCell{
				Day:     d.Day(),
				Date:    key,
				InMonth: d.Month() == mStart.Month(),
				IsToday: key == today,
				Items:   items[key],
			})
		}
		weeks = append(weeks, week)
	}
	return calView{
		Label:    mStart.Format("January 2006"),
		PrevM:    mStart.AddDate(0, -1, 0).Format("2006-01"),
		NextM:    mStart.AddDate(0, 1, 0).Format("2006-01"),
		Weekdays: []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Weeks:    weeks,
	}
}

func mondayOf(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	wd := int(d.Weekday())
	if wd == 0 {
		wd = 7
	}
	return d.AddDate(0, 0, -(wd - 1))
}

// ---- timeline ----

const timelineDays = 14

type tlItem struct {
	Time, Title string
	Recurring   bool
}

type tlDay struct {
	Label   string
	IsToday bool
	Items   []tlItem
}

type tlView struct {
	Overdue []taskView
	Days    []tlDay
}

// ---- card + transitions ----

// taskCard loads one task card into the quests focus' quest-detail panel — the
// rail row click is a Datastar @get that inner-patches #quest-detail.
func (h *handlers) taskCard(e *core.RequestEvent) error {
	rec, err := h.app.FindRecordById("tasks", e.Request.PathValue("id"))
	if err != nil {
		return h.cardError(e, err)
	}
	html, err := h.taskCardHTML(rec)
	if err != nil {
		return e.InternalServerError("rendering task card", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(html, datastar.WithSelectorID("quest-detail"), datastar.WithModeInner())
	return nil
}

// taskCardHTML renders the card-task.html partial for one record to a string,
// for embedding in an SSE patch.
func (h *handlers) taskCardHTML(rec *core.Record) (string, error) {
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "card-task.html", taskViewOf(rec, time.Now())); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (h *handlers) taskTransition(e *core.RequestEvent) error {
	rec, err := h.app.FindRecordById("tasks", e.Request.PathValue("id"))
	if err != nil {
		return h.cardError(e, err)
	}
	now := time.Now()
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
	rec, err = h.app.FindRecordById("tasks", rec.Id)
	if err != nil {
		return h.cardError(e, err)
	}

	// All validation passed — open the SSE patch stream.
	sse := datastar.NewSSE(e.Response, e.Request)

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
	_ = sse.PatchElements(html, datastar.WithSelectorID("tcard-"+rec.Id), datastar.WithModeOuter())

	// The quests artifact (/ui/show/quests) shows a rail that must re-render after
	// a transition so the row moves/strikes. A Datastar @post is a plain fetch,
	// so we identify the surface by Referer. Detail-panel cards carry no "src",
	// so they reach here (board tiles returned above).
	if ref := e.Request.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.Path == "/ui/show/quests" {
			var rb strings.Builder
			if err := taskcards.QuestRail(taskcards.BuildQuestsFocus(h.app)).Render(&rb); err != nil {
				return e.InternalServerError("rendering quest rail", err)
			}
			_ = sse.PatchElements(rb.String(),
				datastar.WithSelectorID("quest-rail"), datastar.WithModeOuter())
		}
	}
	return nil
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

// chatNudges returns agent-initiated messages (origin != "") newer than
// `since` (unix millis) as out-of-band fragments: the messages append to
// #chat and the poller replaces itself with an advanced cursor. Chat turns
// never flow through here — the streamed POST renders those — so polling
// cannot duplicate them.
func (h *handlers) chatNudges(e *core.RequestEvent) error {
	ms, err := strconv.ParseInt(e.Request.URL.Query().Get("since"), 10, 64)
	if err != nil {
		return e.BadRequestError("bad since", err)
	}
	recs, err := h.app.FindRecordsByFilter("messages",
		"origin != '' && created > {:since}", "@rowid", 20, 0,
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
	_ = sse.PatchElements(string(h.renderMessages(h.messageViews(recs))), datastar.WithSelectorID("chat"), datastar.WithModeAppend())
	_ = sse.MarshalAndPatchSignals(struct {
		NudgeSince int64 `json:"nudgeSince"`
	}{last})
	return nil
}
