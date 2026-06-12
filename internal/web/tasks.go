package web

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tasks"
)

// /tasks is the life-organization surface: the operational list (cards with
// actions), a month calendar, and a forward timeline — the future-facing
// mirror of the recap telescope (the telescope looks back, the timeline
// looks ahead). Calendar and timeline are read-only projections of the
// recurrence rules; actions live on the list cards.

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

func taskViewsOf(recs []*core.Record, now time.Time) []taskView {
	out := make([]taskView, 0, len(recs))
	for _, r := range recs {
		out = append(out, taskViewOf(r, now))
	}
	return out
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

// questGroupView is one rhythm group in the quest-log rail.
type questGroupView struct {
	Name  string
	Tasks []taskView
}

// questLogView is the full quest-log list template payload.
type questLogView struct {
	Groups       []questGroupView
	First        *taskView // first open task (for server-side panel pre-render)
	DoneRecently []taskView
}

// buildQuestLog groups open tasks by rhythm and returns the view.
func buildQuestLog(openRecs []*core.Record, doneRecs []*core.Record, now time.Time) questLogView {
	groups := map[string]*questGroupView{
		"Dailies":     {Name: "Dailies"},
		"Rituals":     {Name: "Rituals"},
		"Quests":      {Name: "Quests"},
		"Side quests": {Name: "Side quests"},
	}
	order := []string{"Dailies", "Rituals", "Quests", "Side quests"}

	for _, rec := range openRecs {
		tv := taskViewOf(rec, now)
		grp := questGroup(rec.GetString("recur"), !rec.GetDateTime("due").Time().IsZero())
		groups[grp].Tasks = append(groups[grp].Tasks, tv)
	}

	var result []questGroupView
	for _, name := range order {
		g := groups[name]
		if len(g.Tasks) > 0 {
			result = append(result, *g)
		}
	}

	var first *taskView
	for i := range result {
		if len(result[i].Tasks) > 0 {
			t := result[i].Tasks[0]
			first = &t
			break
		}
	}

	var done []taskView
	for _, r := range doneRecs {
		done = append(done, taskViewOf(r, now))
	}

	return questLogView{
		Groups:       result,
		First:        first,
		DoneRecently: done,
	}
}

func (h *handlers) tasksPage(e *core.RequestEvent) error {
	view := e.Request.URL.Query().Get("view")
	if view != "calendar" && view != "timeline" {
		view = "list"
	}
	now := time.Now()
	recs, err := tasks.OpenTasks(h.app, nil)
	if err != nil {
		return e.InternalServerError("loading tasks", err)
	}

	data := map[string]any{"Title": "Tasks", "View": view}
	switch view {
	case "calendar":
		data["Cal"] = buildCalendar(recs, e.Request.URL.Query().Get("m"), now)
	case "timeline":
		data["TL"] = buildTimeline(recs, now)
	default:
		var doneRecs []*core.Record
		if dr, err := h.app.FindRecordsByFilter("tasks", "status = 'done'", "-updated", 6, 0); err == nil {
			doneRecs = dr
		}
		data["QuestLog"] = buildQuestLog(recs, doneRecs, now)
	}
	return h.render(e, "tasks.html", data)
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

func buildTimeline(recs []*core.Record, now time.Time) tlView {
	loc := now.Location()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	var v tlView
	bk := tasks.Bucket(recs, now)
	v.Overdue = taskViewsOf(bk.Overdue, now)

	for i := 0; i < timelineDays; i++ {
		ds := dayStart.AddDate(0, 0, i)
		de := ds.AddDate(0, 0, 1)
		day := tlDay{IsToday: i == 0, Label: ds.Format("Monday, January 2")}
		switch i {
		case 0:
			day.Label = "Today · " + day.Label
		case 1:
			day.Label = "Tomorrow · " + day.Label
		}
		for _, r := range recs {
			rule, err := tasks.Parse(r.GetString("recur"))
			if err != nil {
				continue
			}
			due := r.GetDateTime("due").Time().In(loc)
			for _, occ := range tasks.Occurrences(rule, due, ds, de) {
				day.Items = append(day.Items, tlItem{
					Time: occ.Format("15:04"), Title: r.GetString("title"), Recurring: !rule.IsZero(),
				})
			}
		}
		sort.Slice(day.Items, func(a, b int) bool { return day.Items[a].Time < day.Items[b].Time })
		v.Days = append(v.Days, day)
	}
	return v
}

// ---- card + transitions ----

func (h *handlers) taskCard(e *core.RequestEvent) error {
	rec, err := h.app.FindRecordById("tasks", e.Request.PathValue("id"))
	if err != nil {
		return h.cardError(e, err)
	}
	return h.renderTaskCard(e, rec)
}

func (h *handlers) renderTaskCard(e *core.RequestEvent, rec *core.Record) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "card-task.html", taskViewOf(rec, time.Now())); err != nil {
		return e.InternalServerError("rendering task card", err)
	}
	return nil
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
	return h.renderTaskCard(e, rec)
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
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if len(recs) == 0 {
		e.Response.WriteHeader(http.StatusOK)
		return nil
	}
	last := recs[len(recs)-1].GetDateTime("created").Time().UnixMilli()
	fmt.Fprint(e.Response, `<div hx-swap-oob="beforeend:#chat">`)
	if err := h.tmpl.ExecuteTemplate(e.Response, "chat-messages.html", h.messageViews(recs)); err != nil {
		return e.InternalServerError("rendering nudges", err)
	}
	fmt.Fprintf(e.Response,
		`</div><div id="nudge-poll" hx-swap-oob="outerHTML" hx-get="/ui/chat/nudges?since=%d" hx-trigger="every 30s" hx-swap="none"></div>`,
		last)
	return nil
}
