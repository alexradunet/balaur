package web

import (
	"html/template"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/feature/lifecards"
	"github.com/alexradunet/balaur/internal/turn"
	_ "github.com/alexradunet/balaur/migrations"
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
	// The inline model chooser was moved to the settings card's models focus.
	if strings.Contains(out, "model-choice-list") {
		t.Error("chatbar should no longer render the inline model chooser")
	}
	if !strings.Contains(out, `href="/ui/show/settings?section=models"`) {
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

	// The chat input is now the storybook ui.Composer, rendered in Go
	// (composerNode) and injected into the dock — not a chat_draft template.
	data.ChatReady = true
	var cb strings.Builder
	if err := composerNode(data).Render(&cb); err != nil {
		t.Fatalf("composer render: %v", err)
	}
	draft := cb.String()
	if !strings.Contains(draft, `id="chat-draft"`) {
		t.Error("composer must render #chat-draft")
	}
	if !strings.Contains(draft, `<textarea name="message"`) {
		t.Error("composer must contain the message textarea when ready")
	}
	if !strings.Contains(draft, `data-on:submit`) || !strings.Contains(draft, `/ui/chat`) {
		t.Error("composer form must @post to /ui/chat")
	}
	// Not-ready: textarea and button are disabled.
	data.ChatReady = false
	var ncb strings.Builder
	if err := composerNode(data).Render(&ncb); err != nil {
		t.Fatalf("composer not-ready render: %v", err)
	}
	if !strings.Contains(ncb.String(), "disabled") {
		t.Error("composer: textarea/button must be disabled when chat is not ready")
	}

}

// TestQuestsArtifact verifies /ui/show/quests injects the quests card artifact
// into the chat stream as a flat task-card stack (plan 093: no rail, no detail pane).
func TestQuestsArtifact(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /ui/show/quests injects quests artifact",
		Method:         "GET",
		URL:            "/ui/show/quests",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`class="quest-stack"`, // flat stack (ui.Focus), not a summary tile
		},
		NotExpectedContent: []string{
			`id="quest-rail"`,
			`class="quest-log"`,
		},
	}
	s.Test(t)
}

// TestLifeBodyRenders: the lifelog focus renders via the gomponents
// lifecards.LifelogFocus renderer — both populated and empty.
func TestLifeBodyRenders(t *testing.T) {
	// Populated: weight (numeric + spark) + gratitude (text) + habit strip.
	v := lifecards.LifelogFocusView{
		Habits: []lifecards.LifeHabitView{{Title: "Stretch", Streak: 5, RecurLine: "repeats daily"}},
		Kinds: []lifecards.LifeKindFocusView{
			{Kind: "weight", Unit: "kg", Count: 3, Numeric: true, LastVal: "82.5", LastAt: "Jun 11",
				Change: "-0.5 over 90d", Points: "4,40 236,8", SparkLastX: "236", SparkLastY: "8"},
			{Kind: "gratitude", Count: 1, Recent: []string{"Jun 10 — the morning was quiet"}},
		},
	}
	var b strings.Builder
	if err := lifecards.LifelogFocus(v).Render(&b); err != nil {
		t.Fatalf("LifelogFocus render: %v", err)
	}
	out := b.String()
	for _, want := range []string{"weight", "82.5", "polyline", "gratitude", "streak 5", "life-grid"} {
		if !strings.Contains(out, want) {
			t.Errorf("lifelog focus missing %q", want)
		}
	}
	// Empty state: no tracked kinds → invitation text.
	b.Reset()
	if err := lifecards.LifelogFocus(lifecards.LifelogFocusView{}).Render(&b); err != nil {
		t.Fatalf("LifelogFocus empty render: %v", err)
	}
	if !strings.Contains(b.String(), "yours to invent") {
		t.Error("empty state missing")
	}
}

// TestDayArtifact verifies /ui/show/day injects the full day view (ui.Focus) into chat.
// Full DayFocus rendering is covered by internal/feature/journalcards/dayfocus_test.go.
func TestDayArtifact(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:            "GET /ui/show/day injects day artifact",
		Method:          "GET",
		URL:             "/ui/show/day",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  200,
		ExpectedContent: []string{"day-focus"},
	}
	scenario.Test(t)
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
	if err := composerNode(ready).Render(&rb); err != nil {
		t.Fatalf("composer ready: %v", err)
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
	// Tool wells are permanently disabled (no handler yet); the textarea and
	// submit button must NOT be disabled when the chat is ready.
	if strings.Contains(openingTag(readyOut, "<textarea"), "disabled") {
		t.Error("ready draft textarea must not be disabled")
	}
	// Locate the submit button by finding type="submit" then walking back to its
	// enclosing <button tag, and confirm that opening tag carries no disabled attr.
	if before, _, ok := strings.Cut(readyOut, `type="submit"`); ok {
		start := strings.LastIndex(before, "<button")
		if start >= 0 && strings.Contains(openingTag(readyOut[start:], "<button"), "disabled") {
			t.Error("ready draft submit button must not be disabled")
		}
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

// openingTag returns the substring of html from the first occurrence of marker
// up to and including the next '>'. Returns "" if marker is not found.
func openingTag(html, marker string) string {
	i := strings.Index(html, marker)
	if i < 0 {
		return ""
	}
	if j := strings.Index(html[i:], ">"); j >= 0 {
		return html[i : i+j+1]
	}
	return html[i:]
}
