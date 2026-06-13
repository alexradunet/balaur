package web

// cards.go — typed card registry handlers (plan 028).
// GET /ui/cards/{type}?params → one rendered card fragment.
// GET /ui/cards            → palette: HTML index of all card specs.
//
// Data access reuses existing query helpers in this package and the
// domain packages (tasks, life, knowledge, heads). No app.Save calls —
// all card endpoints are read-only GET handlers.

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/tasks"
)

// queryToMap converts url.Values to a flat map[string]string (first value
// wins). Only used by the card handlers; complex multi-value params are out
// of scope for the card registry.
func queryToMap(q url.Values) map[string]string {
	m := make(map[string]string, len(q))
	for k, vs := range q {
		if len(vs) > 0 {
			m[k] = vs[0]
		}
	}
	return m
}

// intParam reads a cleaned param by name, falling back to def if absent or empty.
// The cleaned map already has clamped values from cards.Validate.
func intParam(p map[string]string, key string, def int) int {
	if v := p[key]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// uiCardPalette handles GET /ui/cards — the palette listing all card specs.
func (h *handlers) uiCardPalette(e *core.RequestEvent) error {
	specs := cards.All()
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(e.Response, "ucard_palette", specs); err != nil {
		return e.InternalServerError("rendering card palette", err)
	}
	return nil
}

// uiCard handles GET /ui/cards/{type}?params — one rendered card fragment.
func (h *handlers) uiCard(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	spec, ok := cards.Get(typ)
	if !ok {
		return e.NotFoundError("no such card type", nil)
	}

	params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
	if err != nil {
		// Validation error: render a card-note-error strip, HTTP 200.
		e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
		e.Response.WriteHeader(http.StatusOK)
		fmt.Fprintf(e.Response, `<div class="card-note card-note-error" id="ucard-%s">%s</div>`, typ, html.EscapeString(err.Error()))
		return nil
	}

	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch typ {
	case "today":
		return h.renderCardToday(e, spec, params)
	case "quests":
		return h.renderCardQuests(e, spec, params)
	case "calendar":
		return h.renderCardCalendar(e, spec, params)
	case "timeline":
		return h.renderCardTimeline(e, spec, params)
	case "journal":
		return h.renderCardJournal(e, spec, params)
	case "measure":
		return h.renderCardMeasure(e, spec, params)
	case "lines":
		return h.renderCardLines(e, spec, params)
	case "memory":
		return h.renderCardMemory(e, spec, params)
	case "skills":
		return h.renderCardSkills(e, spec, params)
	case "heads":
		return h.renderCardHeads(e, spec, params)
	case "habits":
		return h.renderCardHabits(e, spec, params)
	}
	return e.NotFoundError("unhandled card type", nil)
}

// cardHabitsView feeds the read-only habits card: recurring tasks + streaks.
type cardHabitsView struct {
	Habits []lifeHabitView
}

func (h *handlers) renderCardHabits(e *core.RequestEvent, _ cards.Spec, _ map[string]string) error {
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_habits", cardHabitsView{
		Habits: h.buildHabits(time.Now()),
	})
}

// ---- view-model structs ----

// cardTaskRow is a compact row view for today/quests cards.
type cardTaskRow struct {
	ID, Title, Status, DueLine string
	Overdue                    bool
}

type cardTodayView struct {
	Tasks []cardTaskRow
}

type cardQuestsView struct {
	Tasks     []cardTaskRow
	ParamLine string
}

type calendarCardView struct {
	Cal calView
}

type timelineCardView struct {
	TL        tlView
	ParamLine string
}

type journalEntryRow struct {
	Time, Text string
}

type cardJournalView struct {
	Entries   []journalEntryRow
	TodayDate string
	ParamLine string
}

type cardMeasureView struct {
	Kind                           string
	HasData                        bool
	LastVal, LastAt, Unit, Change  string
	Points, SparkLastX, SparkLastY string
	Error                          string
}

type cardLinesView struct {
	Kind  string
	Lines []string
	Error string
}

type memoryRow struct {
	Title, Category string
	Importance      int
}

type cardMemoryView struct {
	Records   []memoryRow
	ParamLine string
}

type skillRow struct {
	Name, Description string
	Enabled           bool
}

type cardSkillsView struct {
	Records   []skillRow
	ParamLine string
}

type headRow struct {
	ID, Name, Status, Purpose string
}

type cardHeadsView struct {
	Heads []headRow
}

// ---- per-type renderers ----

func (h *handlers) renderCardToday(e *core.RequestEvent, _ cards.Spec, _ map[string]string) error {
	now := time.Now()
	recs, _ := tasks.OpenTasks(h.app, nil)
	bk := tasks.Bucket(recs, now)

	var rows []cardTaskRow
	for _, v := range taskViewsOf(append(bk.Overdue, bk.Today...), now) {
		rows = append(rows, cardTaskRow{
			ID: v.ID, Title: v.Title, Status: v.Status,
			DueLine: v.DueLine, Overdue: v.Overdue,
		})
	}
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_today", cardTodayView{Tasks: rows})
}

// cardQuestsManageView feeds the interactive quests card (mode=manage): open
// tasks rendered via the self-targeting card-task.html partial (#tcard-{id}).
type cardQuestsManageView struct {
	Tasks []taskView
}

func (h *handlers) renderCardQuests(e *core.RequestEvent, _ cards.Spec, params map[string]string) error {
	now := time.Now()
	if params["mode"] == "manage" {
		recs, _ := tasks.OpenTasks(h.app, nil)
		limit := intParam(params, "limit", 12)
		if len(recs) > limit {
			recs = recs[:limit]
		}
		return h.tmpl.ExecuteTemplate(e.Response, "ucard_quests_manage", cardQuestsManageView{
			Tasks: taskViewsOf(recs, now),
		})
	}
	status := params["status"]
	if status == "" {
		status = "open"
	}
	limit := intParam(params, "limit", 10)

	var recs []*core.Record
	var err error
	switch status {
	case "done":
		recs, err = h.app.FindRecordsByFilter("tasks", "status = 'done'", "-updated", limit, 0)
	case "all":
		recs, err = h.app.FindRecordsByFilter("tasks", "status != 'dropped'", "-updated", limit, 0)
	default: // "open"
		recs, err = tasks.OpenTasks(h.app, nil)
		if err == nil && len(recs) > limit {
			recs = recs[:limit]
		}
	}
	if err != nil {
		recs = nil
	}

	var rows []cardTaskRow
	for _, v := range taskViewsOf(recs, now) {
		rows = append(rows, cardTaskRow{
			ID: v.ID, Title: v.Title, Status: v.Status,
			DueLine: v.DueLine, Overdue: v.Overdue,
		})
	}
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_quests", cardQuestsView{
		Tasks:     rows,
		ParamLine: fmt.Sprintf("status: %s · limit: %d", status, limit),
	})
}

func (h *handlers) renderCardCalendar(e *core.RequestEvent, _ cards.Spec, params map[string]string) error {
	now := time.Now()
	recs, _ := tasks.OpenTasks(h.app, nil)
	cal := buildCalendar(recs, params["month"], now)
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_calendar", calendarCardView{Cal: cal})
}

func (h *handlers) renderCardTimeline(e *core.RequestEvent, _ cards.Spec, params map[string]string) error {
	days := intParam(params, "days", timelineDays)
	now := time.Now()
	recs, _ := tasks.OpenTasks(h.app, nil)
	tl := buildTimelineN(recs, now, days)
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_timeline", timelineCardView{
		TL:        tl,
		ParamLine: fmt.Sprintf("%d days", days),
	})
}

func (h *handlers) renderCardJournal(e *core.RequestEvent, _ cards.Spec, params map[string]string) error {
	limit := intParam(params, "limit", 5)
	recs, _ := h.app.FindRecordsByFilter("entries",
		"kind = 'journal'", "-noted_at", limit, 0)

	now := time.Now()
	loc := now.Location()
	var entries []journalEntryRow
	for _, r := range recs {
		entries = append(entries, journalEntryRow{
			Time: r.GetDateTime("noted_at").Time().In(loc).Format("Jan 2 15:04"),
			Text: clipText(r.GetString("text"), 200),
		})
	}
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_journal", cardJournalView{
		Entries:   entries,
		TodayDate: now.Format(dayLayout),
		ParamLine: fmt.Sprintf("last %d", limit),
	})
}

func (h *handlers) renderCardMeasure(e *core.RequestEvent, _ cards.Spec, params map[string]string) error {
	kind := params["kind"]
	days := intParam(params, "days", lifeWindowDays)
	since := time.Now().AddDate(0, 0, -days)

	view := cardMeasureView{Kind: kind}
	recs, err := life.Series(h.app, kind, since)
	if err != nil {
		view.Error = "could not load series: " + err.Error()
		return h.tmpl.ExecuteTemplate(e.Response, "ucard_measure", view)
	}

	s := life.Summarize(recs)
	if s.Points > 0 {
		view.HasData = true
		view.LastVal = fmt.Sprintf("%g", s.Last)
		view.LastAt = s.LastAt.In(time.Now().Location()).Format("Jan 2")
		view.Unit = s.Unit
		if s.Points > 1 {
			view.Change = fmt.Sprintf("%+.4g over %dd", s.Last-s.First, days)
			view.Points, view.SparkLastX, view.SparkLastY = sparkPoints(numericValues(recs), sparkW, sparkH)
		}
	}
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_measure", view)
}

func (h *handlers) renderCardLines(e *core.RequestEvent, _ cards.Spec, params map[string]string) error {
	kind := params["kind"]
	limit := intParam(params, "limit", 5)
	since := time.Now().AddDate(-1, 0, 0) // look back up to one year

	view := cardLinesView{Kind: kind}
	recs, err := life.Series(h.app, kind, since)
	if err != nil {
		view.Error = "could not load series: " + err.Error()
		return h.tmpl.ExecuteTemplate(e.Response, "ucard_lines", view)
	}

	loc := time.Now().Location()
	count := 0
	for i := len(recs) - 1; i >= 0 && count < limit; i-- {
		r := recs[i]
		line := r.GetDateTime("noted_at").Time().In(loc).Format("Jan 2")
		if t := r.GetString("text"); t != "" {
			line += " — " + clipText(t, 120)
		}
		view.Lines = append(view.Lines, line)
		count++
	}
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_lines", view)
}

// manageCardView feeds the interactive knowledge card (mode=manage): the
// proposed queue + active records, each rendered via the existing self-targeting
// card-{memory,skill}.html partials (so several cards never collide).
type manageCardView struct {
	Kind     string // "memories" | "skills" — selects the card-*.html include
	Label    string
	Icon     string
	Href     string
	Proposed []*core.Record
	Active   []*core.Record
}

// renderKnowledgeManage renders an interactive memory/skill card: proposed
// (approve/reject inline) + a capped slice of active (archive/edit inline).
func (h *handlers) renderKnowledgeManage(e *core.RequestEvent, kind knowledge.Kind, v manageCardView) error {
	v.Proposed, _ = knowledge.ListByStatus(h.app, kind, knowledge.StatusProposed)
	v.Active, _ = knowledge.FilterActive(h.app, kind, "", "")
	if len(v.Active) > 8 {
		v.Active = v.Active[:8]
	}
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_knowledge_manage", v)
}

func (h *handlers) renderCardMemory(e *core.RequestEvent, _ cards.Spec, params map[string]string) error {
	if params["mode"] == "manage" {
		return h.renderKnowledgeManage(e, knowledge.Memory, manageCardView{
			Kind: "memories", Label: "Memory", Icon: "tome", Href: "/memory",
		})
	}
	limit := intParam(params, "limit", 6)
	query := params["query"]

	recs, _ := knowledge.FilterActive(h.app, knowledge.Memory, query, "")
	if len(recs) > limit {
		recs = recs[:limit]
	}

	var rows []memoryRow
	for _, r := range recs {
		rows = append(rows, memoryRow{
			Title:      r.GetString("title"),
			Category:   r.GetString("category"),
			Importance: r.GetInt("importance"),
		})
	}

	paramLine := fmt.Sprintf("limit: %d", limit)
	if query != "" {
		paramLine += " · q: " + query
	}
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_memory", cardMemoryView{
		Records:   rows,
		ParamLine: paramLine,
	})
}

func (h *handlers) renderCardSkills(e *core.RequestEvent, _ cards.Spec, params map[string]string) error {
	if params["mode"] == "manage" {
		return h.renderKnowledgeManage(e, knowledge.Skill, manageCardView{
			Kind: "skills", Label: "Skills", Icon: "key", Href: "/skills",
		})
	}
	limit := intParam(params, "limit", 6)

	recs, _ := knowledge.FilterActive(h.app, knowledge.Skill, "", "")
	if len(recs) > limit {
		recs = recs[:limit]
	}

	var rows []skillRow
	for _, r := range recs {
		rows = append(rows, skillRow{
			Name:        r.GetString("name"),
			Description: r.GetString("description"),
			Enabled:     r.GetBool("enabled"),
		})
	}
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_skills", cardSkillsView{
		Records:   rows,
		ParamLine: fmt.Sprintf("limit: %d", limit),
	})
}

// cardHeadsManageView feeds the interactive heads card (mode=manage): active
// heads rendered via the self-targeting head_card partial (#head-{id}).
type cardHeadsManageView struct {
	Heads []headView
}

func (h *handlers) renderCardHeads(e *core.RequestEvent, _ cards.Spec, params map[string]string) error {
	recs, _ := h.app.FindRecordsByFilter("heads", "status = 'active'", "-@rowid", 0, 0)

	if params["mode"] == "manage" {
		views := make([]headView, 0, len(recs))
		for _, r := range recs {
			views = append(views, headViewFrom(h.app, r))
		}
		return h.tmpl.ExecuteTemplate(e.Response, "ucard_heads_manage", cardHeadsManageView{Heads: views})
	}

	var heads []headRow
	for _, r := range recs {
		heads = append(heads, headRow{
			ID:      r.Id,
			Name:    r.GetString("name"),
			Status:  r.GetString("status"),
			Purpose: r.GetString("purpose"),
		})
	}
	return h.tmpl.ExecuteTemplate(e.Response, "ucard_heads", cardHeadsView{Heads: heads})
}

// buildTimelineN is like buildTimeline but takes an explicit day count,
// used by the timeline card. buildTimeline in tasks.go hardcodes timelineDays.
func buildTimelineN(recs []*core.Record, now time.Time, days int) tlView {
	if days <= 0 {
		days = timelineDays
	}
	loc := now.Location()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	var v tlView
	bk := tasks.Bucket(recs, now)
	v.Overdue = taskViewsOf(bk.Overdue, now)

	for i := 0; i < days; i++ {
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
		v.Days = append(v.Days, day)
	}
	return v
}
