package web

import (
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/ollama"
	"github.com/alexradunet/balaur/internal/turn"
	webassets "github.com/alexradunet/balaur/web"
)

// parseTemplates mirrors Register's template setup. The runtime uses
// template.Must, which panics at serve start — these tests surface template
// errors at test time instead.
func parseTemplates(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("").Funcs(funcs).ParseFS(webassets.FS, "templates/*.html")
	if err != nil {
		t.Fatalf("parsing templates: %v", err)
	}
	return tmpl
}

func TestTemplatesParse(t *testing.T) {
	parseTemplates(t)
}

func TestTaskCardRenders(t *testing.T) {
	tmpl := parseTemplates(t)
	var b strings.Builder
	v := taskView{ID: "abc123", Title: "Call notary", Status: "open",
		DueLine: "due Thu, Jun 12 at 10:00", RecurLine: "repeats daily", Notes: "papers"}
	if err := tmpl.ExecuteTemplate(&b, "card-task.html", v); err != nil {
		t.Fatalf("card-task: %v", err)
	}
	out := b.String()
	for _, want := range []string{"tcard-abc123", "Call notary", "Done", "Snooze", "Drop"} {
		if !strings.Contains(out, want) {
			t.Errorf("card missing %q", want)
		}
	}
}

func TestModelsPageAndCleanChatbarRender(t *testing.T) {
	tmpl := parseTemplates(t)
	choice := turn.ModelChoice{Key: "m1", Provider: "local", Model: "model.gguf", Name: "Local Qwen3.6 35B A3B", Detail: "model.gguf · on this box", Badge: "local", Active: true}
	data := homeData{Title: "Balaur", ChatReady: true, ActiveModel: "Local Qwen3.6 35B A3B", ChatPlaceholder: "Speak...", ModelChoices: []turn.ModelChoice{choice}}

	// chat_bar is now a slim ledge — no form, just the model switcher + profile link.
	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, "chat_bar", data); err != nil {
		t.Fatalf("chat_bar: %v", err)
	}
	out := b.String()
	// The inline model chooser and the add-API form were moved to the settings
	// card's models focus.
	if strings.Contains(out, "model-choice-list") || strings.Contains(out, "Add OpenAI-compatible API") {
		t.Error("chatbar should no longer render the inline model chooser or add-API form")
	}
	if !strings.Contains(out, `href="/focus/settings?section=models"`) {
		t.Error("chatbar should link to the settings models focus to manage models")
	}
	// The form now lives in chat_draft, not in the chatbar.
	if strings.Contains(out, `name="message"`) {
		t.Error("chatbar must not contain the message textarea — it belongs in chat_draft")
	}

	b.Reset()
	data.ChatReady = false
	if err := tmpl.ExecuteTemplate(&b, "chat_bar", data); err != nil {
		t.Fatalf("chat_bar not ready: %v", err)
	}
	if strings.Contains(b.String(), `input type="text"`) {
		t.Error("chatbar should not render the old disabled text input when chat is not ready")
	}

	// While a model is downloading, the chatbar shows a loading bar, not the form.
	b.Reset()
	data.ChatReady = false
	data.Pull = ollama.PullSnapshot{Active: true, BytesDone: 500, BytesTotal: 1000, Dest: "/models/x.llamafile"}
	if err := tmpl.ExecuteTemplate(&b, "chat_bar", data); err != nil {
		t.Fatalf("chat_bar downloading: %v", err)
	}
	dl := b.String()
	if !strings.Contains(dl, "chatbar-download") || !strings.Contains(dl, "<progress") {
		t.Error("chatbar should render a download progress bar while a model is downloading")
	}
	if strings.Contains(dl, `<textarea name="message"`) {
		t.Error("chatbar must not render the chat input — it belongs in chat_draft")
	}

	// chat_draft renders the composer form in the flow.
	b.Reset()
	data.ChatReady = true
	data.Pull = ollama.PullSnapshot{}
	if err := tmpl.ExecuteTemplate(&b, "chat_draft", data); err != nil {
		t.Fatalf("chat_draft: %v", err)
	}
	draft := b.String()
	if !strings.Contains(draft, `id="chat-draft"`) {
		t.Error("chat_draft must render #chat-draft")
	}
	if !strings.Contains(draft, `<textarea name="message"`) {
		t.Error("chat_draft must contain the message textarea when ready")
	}
	// Not-ready: textarea and button are disabled.
	b.Reset()
	data.ChatReady = false
	if err := tmpl.ExecuteTemplate(&b, "chat_draft", data); err != nil {
		t.Fatalf("chat_draft not ready: %v", err)
	}
	notReady := b.String()
	if !strings.Contains(notReady, "disabled") {
		t.Error("chat_draft: textarea/button must be disabled when chat is not ready")
	}

	// The settings shell is the settings card focus now (plan 056): the models
	// section renders via the shared settings_body define.
	b.Reset()
	models := modelsPageData{ActiveModel: "Local Qwen3.6 35B A3B", ModelChoices: []turn.ModelChoice{choice}}
	settingsModels := settingsData{Section: "models", Models: models}
	if err := tmpl.ExecuteTemplate(&b, "settings_body", settingsModels); err != nil {
		t.Fatalf("settings_body models: %v", err)
	}
	out = b.String()
	for _, want := range []string{"Available models", "Add OpenAI-compatible API", "Local Qwen3.6 35B A3B"} {
		if !strings.Contains(out, want) {
			t.Errorf("models section missing %q", want)
		}
	}
}

// TestQuestsFocusListRenders renders the quests focus body (tasks_list) — the
// surface formerly at /tasks?view=list. tasks_list reads .QuestLog, so it must
// be passed via map[string]any{"QuestLog": ...}. The calendar/timeline views
// are now their own cards (covered by cards_test.go's TestUiCardAllTypesRender).
func TestQuestsFocusListRenders(t *testing.T) {
	tmpl := parseTemplates(t)
	now := time.Now()
	ql := buildQuestLog(nil, nil, now)
	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, "tasks_list", map[string]any{"QuestLog": ql}); err != nil {
		t.Fatalf("tasks_list: %v", err)
	}
	out := b.String()
	for _, want := range []string{`id="quest-rail"`, `id="quest-detail"`} {
		if !strings.Contains(out, want) {
			t.Errorf("tasks_list missing %q", want)
		}
	}
}

// TestLifeBodyRenders: the life overview body (life_body) — now the lifelog
// card's focus, formerly the /life page — renders both populated and empty.
func TestLifeBodyRenders(t *testing.T) {
	tmpl := parseTemplates(t)
	points, lx, ly := sparkPoints([]float64{83, 82.6, 82.5}, sparkW, sparkH)
	if points == "" || lx == "" || ly == "" {
		t.Fatalf("sparkPoints empty: %q %q %q", points, lx, ly)
	}
	data := map[string]any{
		"Habits": []lifeHabitView{{Title: "Stretch", Streak: 5, RecurLine: "repeats daily"}},
		"Kinds": []lifeKindView{
			{Kind: "weight", Unit: "kg", Count: 3, Numeric: true, LastVal: "82.5", LastAt: "Jun 11",
				Change: "-0.5 over 90d", Points: points, SparkLastX: lx, SparkLastY: ly},
			{Kind: "gratitude", Count: 1, Recent: []string{"Jun 10 — the morning was quiet"}},
		},
	}
	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, "life_body", data); err != nil {
		t.Fatalf("life_body: %v", err)
	}
	out := b.String()
	for _, want := range []string{"weight", "82.5", "polyline", "gratitude", "streak 5", "life-grid"} {
		if !strings.Contains(out, want) {
			t.Errorf("life body missing %q", want)
		}
	}
	// Empty state renders too.
	b.Reset()
	if err := tmpl.ExecuteTemplate(&b, "life_body", map[string]any{}); err != nil {
		t.Fatalf("life_body empty: %v", err)
	}
	if !strings.Contains(b.String(), "yours to invent") {
		t.Error("empty state missing")
	}
}

// TestDayPageRenders: the day view is now the day card's focus body (day_focus).
// The standalone day.html doc is retired, so this renders the body fragment and
// asserts its sections; prev/next deep-link into the focus (/focus/day?date=…).
func TestDayPageRenders(t *testing.T) {
	tmpl := parseTemplates(t)
	data := dayData{
		Title: "Wednesday, June 10", Date: "2026-06-10",
		Label: "Wednesday, June 10 2026",
		Prev:  "2026-06-09", Next: "2026-06-11",
		Journal:    []dayJournalView{{ID: "j1", Time: "21:40", Text: "A good, quiet day."}},
		Recap:      "You sorted the notary papers and trained in the evening.",
		RecapStart: "1780000000",
		Done:       []dayLineView{{Time: "10:12", Text: "Call notary"}},
		Logs:       []dayLineView{{Time: "08:00", Text: "weight: 82.5 kg"}},
	}
	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, "day_focus", data); err != nil {
		t.Fatalf("day_focus: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		"A good, quiet day.", "remove", "Keep it", "notary papers",
		"transcript", "Call notary", "weight: 82.5 kg",
		"/focus/day?date=2026-06-09", "/focus/day?date=2026-06-11",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("day focus missing %q", want)
		}
	}

	// Today: no next link, no transcript expander, honest empty states.
	b.Reset()
	today := dayData{Title: "t", Date: "2026-06-11", Label: "Thursday, June 11 2026", IsToday: true, Prev: "2026-06-10"}
	if err := tmpl.ExecuteTemplate(&b, "day_focus", today); err != nil {
		t.Fatalf("day_focus today: %v", err)
	}
	out = b.String()
	if strings.Contains(out, "transcript") {
		t.Error("today must not offer a transcript expander")
	}
	for _, want := range []string{"still being written", "Nothing marked done", "Nothing logged"} {
		if !strings.Contains(out, want) {
			t.Errorf("today focus missing %q", want)
		}
	}
}

func TestCalendarCellsLinkToDayPages(t *testing.T) {
	tmpl := parseTemplates(t)
	var b strings.Builder
	// The calendar surface is now the calendar card (ucard_calendar); its cells
	// deep-link into the day card's focus (/focus/day?date=…), which replaced the
	// retired /day page.
	cal := buildCalendar(nil, "2026-06", time.Date(2026, 6, 11, 12, 0, 0, 0, time.Local))
	if err := tmpl.ExecuteTemplate(&b, "ucard_calendar", calendarCardView{Cal: cal}); err != nil {
		t.Fatalf("ucard_calendar: %v", err)
	}
	if !strings.Contains(b.String(), `href="/focus/day?date=2026-06-11"`) {
		t.Error("calendar cells do not link to the day focus")
	}
}

func TestToolIconRendersAsImg(t *testing.T) {
	tmpl := parseTemplates(t)
	var b strings.Builder
	mv := messageView{Tool: "task_add", Content: "added a task"}
	if err := tmpl.ExecuteTemplate(&b, "chat-msg-tool", mv); err != nil {
		t.Fatalf("chat-msg-tool: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, `<img class="tool-icon" src="/static/icons/`) {
		t.Errorf("tool row should render pixel icon img, got: %s", out)
	}
	if strings.Contains(out, `<span class="tool-icon"`) {
		t.Error("tool row should no longer use span.tool-icon glyph")
	}
}

func TestChatMsgBalaurPortraitStructure(t *testing.T) {
	tmpl := parseTemplates(t)
	var b strings.Builder
	mv := messageView{
		BalaurAvatarURL: "/static/avatars/balaur-01.png",
		WhoLabel:        "Balaur",
		Content:         "Hello, traveller.",
	}
	if err := tmpl.ExecuteTemplate(&b, "chat-msg-balaur", mv); err != nil {
		t.Fatalf("chat-msg-balaur: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, `<figure class="portrait">`) {
		t.Errorf("chat-msg-balaur should contain portrait figure, got:\n%s", out)
	}
	// .who must be inside the portrait figure (as figcaption), not inside .msg-main
	msgMainIdx := strings.Index(out, `class="msg-main"`)
	whoIdx := strings.Index(out, `class="who"`)
	if msgMainIdx == -1 {
		t.Error("chat-msg-balaur missing msg-main div")
	}
	if whoIdx == -1 {
		t.Error("chat-msg-balaur missing .who element")
	}
	if msgMainIdx != -1 && whoIdx != -1 && whoIdx > msgMainIdx {
		t.Errorf("chat-msg-balaur: .who appears inside .msg-main (at %d > %d); it must be in .portrait instead", whoIdx, msgMainIdx)
	}
}

func TestChatMsgUserPortraitStructure(t *testing.T) {
	tmpl := parseTemplates(t)
	var b strings.Builder
	mv := messageView{
		SoulAvatarURL: "/static/avatars/soul.png",
		OwnerName:     "Alex",
		Content:       "Tell me more.",
	}
	if err := tmpl.ExecuteTemplate(&b, "chat-msg-user", mv); err != nil {
		t.Fatalf("chat-msg-user: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, `<figure class="portrait">`) {
		t.Errorf("chat-msg-user should contain portrait figure, got:\n%s", out)
	}
	msgMainIdx := strings.Index(out, `class="msg-main"`)
	whoIdx := strings.Index(out, `class="who"`)
	if msgMainIdx != -1 && whoIdx != -1 && whoIdx > msgMainIdx {
		t.Errorf("chat-msg-user: .who appears inside .msg-main; it must be in .portrait instead")
	}
}

func TestChatStreamingBalancedDivs(t *testing.T) {
	// The open/close fragment pair is load-bearing: unclosed tags in
	// chat-balaur-open must equal the closes in chat-balaur-close.
	tmpl := parseTemplates(t)
	mv := messageView{
		BalaurAvatarURL: "/static/avatars/balaur-01.png",
		WhoLabel:        "Balaur",
	}

	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, "chat-balaur-open", mv); err != nil {
		t.Fatalf("chat-balaur-open: %v", err)
	}
	// Simulate streaming content.
	b.WriteString("streamed token text")
	if err := tmpl.ExecuteTemplate(&b, "chat-balaur-close", messageView{}); err != nil {
		t.Fatalf("chat-balaur-close: %v", err)
	}

	out := b.String()
	openCount := strings.Count(out, "<div")
	closeCount := strings.Count(out, "</div")
	if openCount != closeCount {
		t.Errorf("streaming open/close: unbalanced divs: %d <div> vs %d </div>\nHTML:\n%s",
			openCount, closeCount, out)
	}
}

func TestSparkPointsScaling(t *testing.T) {
	points, _, ly := sparkPoints([]float64{1, 2, 3}, 240, 48)
	if points == "" {
		t.Fatal("no points")
	}
	// Three values: three coordinate pairs; the max (last) sits near the top.
	if got := len(strings.Fields(points)); got != 3 {
		t.Errorf("point pairs = %d, want 3", got)
	}
	if !strings.HasPrefix(ly, "4") { // pad = 4.0 at the maximum
		t.Errorf("last y = %s, want near top pad", ly)
	}
	if p, _, _ := sparkPoints([]float64{5}, 240, 48); p != "" {
		t.Errorf("single point should not draw a line: %q", p)
	}
}

// TestChatbarPollAndDraft verifies the chatbar carries the 2s Datastar poll
// only while no model is ready (so it stops once ready) and the ready draft is
// enabled — no htmx OOB attributes remain.
func TestChatbarPollAndDraft(t *testing.T) {
	tmpl := parseTemplates(t)

	ready := homeData{
		ChatReady:       true,
		ActiveModel:     "TestModel",
		ChatPlaceholder: "Speak…",
		SoulAvatarURL:   "/static/avatars/soul.png",
		OwnerName:       "Alex",
	}
	var rb strings.Builder
	if err := tmpl.ExecuteTemplate(&rb, "chat_bar", ready); err != nil {
		t.Fatalf("chat_bar ready: %v", err)
	}
	if err := tmpl.ExecuteTemplate(&rb, "chat_draft", ready); err != nil {
		t.Fatalf("chat_draft ready: %v", err)
	}
	readyOut := rb.String()
	if !strings.Contains(readyOut, `id="chatbar"`) {
		t.Error("ready chatbar must contain id=\"chatbar\"")
	}
	if strings.Contains(readyOut, "data-on:interval") {
		t.Error("ready chatbar must NOT poll — the 2s interval stops once a model is ready")
	}
	if !strings.Contains(readyOut, `id="chat-draft"`) {
		t.Error("ready response must contain the draft composer")
	}
	if strings.Contains(readyOut, "disabled") {
		t.Error("ready draft must not be disabled")
	}
	if strings.Contains(readyOut, "hx-") {
		t.Error("no htmx attributes may remain on the chatbar/draft")
	}

	// Not ready: the chatbar carries the 2s Datastar poll.
	notReady := homeData{ChatReady: false, SoulAvatarURL: "/static/avatars/soul.png", OwnerName: "Alex"}
	var nb strings.Builder
	if err := tmpl.ExecuteTemplate(&nb, "chat_bar", notReady); err != nil {
		t.Fatalf("chat_bar not ready: %v", err)
	}
	if !strings.Contains(nb.String(), `data-on:interval__duration.2s="@get('/ui/chatbar')"`) {
		t.Error("not-ready chatbar must poll every 2s via Datastar")
	}
}
