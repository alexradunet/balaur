package web

import (
	"context"
	"fmt"
	"html"
	"html/template"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/llm"
)

// /journal — the candle: an immersive writing page for the owner's own words.
// Free-hand (default) or guided by one model-composed prompt line.
// Entries are the same journal records as the day pages and chat tool.

const candlePromptFallback = "Write what the day left behind. I am listening."
const candlePromptTimeout = 30 * time.Second

type candleJournalView struct {
	ID, Time, Text, Date string
}

type candleData struct {
	Title     string
	MainClass string
	Dock      homeData
	Today     string // YYYY-MM-DD, for the write form target
	Journal   []candleJournalView
}

func (h *handlers) journalPage(e *core.RequestEvent) error {
	now := time.Now()
	data, err := h.buildCandleData(now)
	if err != nil {
		return e.InternalServerError("loading journal", err)
	}
	return h.render(e, "journal.html", data)
}

// journalWrite handles POST /ui/journal: writes an entry for today, then
// re-renders the journal_candle_body fragment.
func (h *handlers) journalWrite(e *core.RequestEvent) error {
	now := time.Now()
	text := strings.TrimSpace(e.Request.FormValue("text"))
	if text == "" {
		// Mirror dayJournalWrite's empty-input behavior: 400.
		return e.BadRequestError("nothing to keep — the entry is empty", nil)
	}
	if _, err := life.JournalWrite(h.app, text, now); err != nil {
		return e.InternalServerError("writing journal entry", err)
	}
	return h.renderCandleBody(e, now)
}

// journalPrompt handles GET /ui/journal/prompt: returns one guided prompt
// line. Model-composed when a client is available, deterministic fallback on
// any error or no model (AGENTS.md: deterministic offline default, LLM opt-in).
func (h *handlers) journalPrompt(e *core.RequestEvent) error {
	line := candlePromptFallback

	if client, err := h.clients.Active(h.app); err == nil {
		if composed := composeJournalPrompt(client); composed != "" {
			line = composed
		}
	}

	// Escape text — never template.HTML from an LLM response.
	frag := `<p class="candle-prompt">` + html.EscapeString(line) + `</p>`
	// Patch the prompt line into its target container by id (inner HTML).
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(frag,
		datastar.WithSelectorID("candle-prompt"), datastar.WithModeInner())
	return nil
}

// composeJournalPrompt asks the model for a single journal prompt line in the
// companion voice. Returns "" on any error or implausibly long result — caller
// keeps the deterministic fallback in that case.
func composeJournalPrompt(client llm.Client) string {
	ctx, cancel := context.WithTimeout(context.Background(), candlePromptTimeout)
	defer cancel()

	msgs := []llm.Message{
		{Role: "system", Content: "You are Balaur, a wise personal companion. " +
			"Write one short, warm prompt inviting the owner to write in their journal. " +
			"Plain prose, no questions, no exclamation marks, no flattery, no emoji. " +
			"Keep it under 140 characters."},
		{Role: "user", Content: "Give me one journal prompt for today."},
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
	if text == "" || len(text) > 280 {
		return ""
	}
	// Clip to ~140 chars at a word boundary.
	if len(text) > 140 {
		text = text[:140]
		if i := strings.LastIndexByte(text, ' '); i > 80 {
			text = text[:i]
		}
	}
	return text
}

func (h *handlers) buildCandleData(now time.Time) (candleData, error) {
	today := dayStartOf(now)
	loc := now.Location()

	var convID string
	if master, err := conversation.Master(h.app); err == nil {
		convID = master.Id
	}
	dd, err := life.Day(h.app, convID, today)
	if err != nil {
		return candleData{}, fmt.Errorf("buildCandleData: %w", err)
	}

	dock, _ := h.dockData()
	data := candleData{
		Title:     "Journal",
		MainClass: "candle-page",
		Dock:      dock,
		Today:     today.Format(dayLayout),
	}
	for _, r := range dd.Journal {
		data.Journal = append(data.Journal, candleJournalView{
			ID:   r.Id,
			Time: r.GetDateTime("noted_at").Time().In(loc).Format("15:04"),
			Text: r.GetString("text"),
			Date: today.Format(dayLayout),
		})
	}
	return data, nil
}

func (h *handlers) renderCandleBody(e *core.RequestEvent, now time.Time) error {
	data, err := h.buildCandleData(now)
	if err != nil {
		return e.InternalServerError("rendering journal", err)
	}
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "journal_candle_body", data); err != nil {
		return e.InternalServerError("rendering candle body", err)
	}
	// All work succeeded — open the SSE and morph the body in place by its id.
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(b.String(),
		datastar.WithSelectorID("journal-candle-body"), datastar.WithModeOuter())
	return nil
}

// journalFocusHTML renders the journal card's focus body: the candle
// (free/guided write + guided prompt + today's history). Was the /journal page.
func (h *handlers) journalFocusHTML() template.HTML {
	data, err := h.buildCandleData(time.Now())
	if err != nil {
		h.app.Logger().Warn("journal focus render failed", "err", err)
		return cardErrorStrip("could not open the journal")
	}
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "journal_focus", data); err != nil {
		h.app.Logger().Warn("journal focus template failed", "err", err)
		return cardErrorStrip("could not render the journal")
	}
	return template.HTML(b.String())
}

// journalPageDayEntries returns journal entries for a day, for use by the
// integration assertion in tests (the candle write path and day page share
// the same underlying journal records).
func (h *handlers) journalPageDayEntries(d time.Time) ([]candleJournalView, error) {
	now := time.Now()
	loc := now.Location()
	var convID string
	if master, err := conversation.Master(h.app); err == nil {
		convID = master.Id
	}
	dayData, err := life.Day(h.app, convID, d)
	if err != nil {
		return nil, err
	}
	views := make([]candleJournalView, 0, len(dayData.Journal))
	for _, r := range dayData.Journal {
		views = append(views, candleJournalView{
			ID:   r.Id,
			Time: r.GetDateTime("noted_at").Time().In(loc).Format("15:04"),
			Text: r.GetString("text"),
			Date: d.Format(dayLayout),
		})
	}
	return views, nil
}
