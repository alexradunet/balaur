package web

import (
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/gguf"
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
	// The inline model chooser and the add-API form were moved to /settings/models.
	if strings.Contains(out, "model-choice-list") || strings.Contains(out, "Add OpenAI-compatible API") {
		t.Error("chatbar should no longer render the inline model chooser or add-API form")
	}
	if !strings.Contains(out, `href="/settings/models"`) {
		t.Error("chatbar should link to /settings/models to manage models")
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
	data.Gguf = gguf.Progress{Active: true, BytesDone: 500, BytesTotal: 1000, Dest: "/models/x.llamafile"}
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
	data.Gguf = gguf.Progress{}
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

	b.Reset()
	models := modelsPageData{Title: "Models", ActiveModel: "Local Qwen3.6 35B A3B", ModelChoices: []turn.ModelChoice{choice}}
	settingsModels := settingsData{Title: "Settings", Section: "models", Models: models}
	if err := tmpl.ExecuteTemplate(&b, "settings.html", settingsModels); err != nil {
		t.Fatalf("settings.html models: %v", err)
	}
	out = b.String()
	for _, want := range []string{"Available models", "Add OpenAI-compatible API", "Local Qwen3.6 35B A3B"} {
		if !strings.Contains(out, want) {
			t.Errorf("models page missing %q", want)
		}
	}
}

func TestTasksPageViewsRender(t *testing.T) {
	tmpl := parseTemplates(t)
	now := time.Now()
	for name, data := range map[string]map[string]any{
		"list":     {"Title": "Tasks", "View": "list", "QuestLog": questLogView{}},
		"calendar": {"Title": "Tasks", "View": "calendar", "Cal": buildCalendar(nil, "", now)},
		"timeline": {"Title": "Tasks", "View": "timeline", "TL": buildTimeline(nil, now)},
	} {
		var b strings.Builder
		if err := tmpl.ExecuteTemplate(&b, "tasks.html", data); err != nil {
			t.Errorf("tasks.html %s view: %v", name, err)
		}
	}
}

func TestLifePageRenders(t *testing.T) {
	tmpl := parseTemplates(t)
	points, lx, ly := sparkPoints([]float64{83, 82.6, 82.5}, sparkW, sparkH)
	if points == "" || lx == "" || ly == "" {
		t.Fatalf("sparkPoints empty: %q %q %q", points, lx, ly)
	}
	data := map[string]any{
		"Title":  "Life",
		"Habits": []lifeHabitView{{Title: "Stretch", Streak: 5, RecurLine: "repeats daily"}},
		"Kinds": []lifeKindView{
			{Kind: "weight", Unit: "kg", Count: 3, Numeric: true, LastVal: "82.5", LastAt: "Jun 11",
				Change: "-0.5 over 90d", Points: points, SparkLastX: lx, SparkLastY: ly},
			{Kind: "gratitude", Count: 1, Recent: []string{"Jun 10 — the morning was quiet"}},
		},
	}
	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, "life.html", data); err != nil {
		t.Fatalf("life.html: %v", err)
	}
	out := b.String()
	for _, want := range []string{"weight", "82.5", "polyline", "gratitude", "streak 5"} {
		if !strings.Contains(out, want) {
			t.Errorf("life page missing %q", want)
		}
	}
	// Empty state renders too.
	b.Reset()
	if err := tmpl.ExecuteTemplate(&b, "life.html", map[string]any{"Title": "Life"}); err != nil {
		t.Fatalf("life.html empty: %v", err)
	}
	if !strings.Contains(b.String(), "yours to invent") {
		t.Error("empty state missing")
	}
}

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
	if err := tmpl.ExecuteTemplate(&b, "day.html", data); err != nil {
		t.Fatalf("day.html: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		"A good, quiet day.", "remove", "Keep it", "notary papers",
		"transcript", "Call notary", "weight: 82.5 kg", "/day/2026-06-09", "/day/2026-06-11",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("day page missing %q", want)
		}
	}

	// Today: no next link, no transcript expander, honest empty states.
	b.Reset()
	today := dayData{Title: "t", Date: "2026-06-11", Label: "Thursday, June 11 2026", IsToday: true, Prev: "2026-06-10"}
	if err := tmpl.ExecuteTemplate(&b, "day.html", today); err != nil {
		t.Fatalf("day.html today: %v", err)
	}
	out = b.String()
	if strings.Contains(out, "transcript") {
		t.Error("today must not offer a transcript expander")
	}
	for _, want := range []string{"still being written", "Nothing marked done", "Nothing logged"} {
		if !strings.Contains(out, want) {
			t.Errorf("today page missing %q", want)
		}
	}
}

func TestCalendarCellsLinkToDayPages(t *testing.T) {
	tmpl := parseTemplates(t)
	var b strings.Builder
	data := map[string]any{"Title": "Tasks", "View": "calendar", "Cal": buildCalendar(nil, "2026-06", time.Date(2026, 6, 11, 12, 0, 0, 0, time.Local))}
	if err := tmpl.ExecuteTemplate(&b, "tasks.html", data); err != nil {
		t.Fatalf("tasks.html calendar: %v", err)
	}
	if !strings.Contains(b.String(), `href="/day/2026-06-11"`) {
		t.Error("calendar cells do not link to day pages")
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

// TestChatbarOOBDraftOnReady verifies that the chatbar poll response carries
// an OOB swap of chat_draft when the model is ready, and nothing about the
// draft when the model is not ready.
func TestChatbarOOBDraftOnReady(t *testing.T) {
	tmpl := parseTemplates(t)

	// Model ready: chatbar + OOB draft both present.
	ready := homeData{
		ChatReady:       true,
		ActiveModel:     "TestModel",
		ChatPlaceholder: "Speak…",
		SoulAvatarURL:   "/static/avatars/soul.png",
		OwnerName:       "Alex",
	}

	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, "chat_bar", ready); err != nil {
		t.Fatalf("chat_bar ready: %v", err)
	}
	// Simulate what chatbar handler does: also render chat_draft with DraftOOB=true.
	ready.DraftOOB = true
	if err := tmpl.ExecuteTemplate(&b, "chat_draft", ready); err != nil {
		t.Fatalf("chat_draft OOB: %v", err)
	}
	out := b.String()

	if !strings.Contains(out, `id="chat-draft"`) {
		t.Error("ready response must contain id=\"chat-draft\"")
	}
	if !strings.Contains(out, `hx-swap-oob="outerHTML"`) {
		t.Error("ready response must contain hx-swap-oob=\"outerHTML\" on the draft")
	}
	if strings.Contains(out, "disabled") {
		t.Error("ready response must not contain disabled attributes")
	}

	// Model not ready: only chatbar, no draft element at all.
	notReady := homeData{
		ChatReady:     false,
		SoulAvatarURL: "/static/avatars/soul.png",
		OwnerName:     "Alex",
	}
	b.Reset()
	if err := tmpl.ExecuteTemplate(&b, "chat_bar", notReady); err != nil {
		t.Fatalf("chat_bar not ready: %v", err)
	}
	barOnly := b.String()
	if strings.Contains(barOnly, "chat-draft") {
		t.Error("not-ready chatbar response must not contain chat-draft")
	}
}

// TestHeadChatDraftIdDecollided verifies that head-chat.html uses
// id="head-chat-draft" and not id="chat-draft", so that an OOB swap
// targeting #chat-draft from /ui/chatbar cannot clobber the head page's form.
func TestHeadChatDraftIdDecollided(t *testing.T) {
	tmpl := parseTemplates(t)

	data := headChatData{
		Title:           "TestHead · Balaur",
		HeadID:          "abc123",
		HeadName:        "TestHead",
		SoulAvatarURL:   "/static/avatars/soul.png",
		BalaurAvatarURL: "/static/avatars/balaur-01.png",
		OwnerName:       "Alex",
		ChatReady:       true,
	}

	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, "head-chat.html", data); err != nil {
		t.Fatalf("head-chat.html: %v", err)
	}
	out := b.String()

	if strings.Contains(out, `id="chat-draft"`) {
		t.Error("head-chat.html must not use id=\"chat-draft\" (collision hazard with home OOB swap)")
	}
	if !strings.Contains(out, `id="head-chat-draft"`) {
		t.Error("head-chat.html must use id=\"head-chat-draft\"")
	}
}
