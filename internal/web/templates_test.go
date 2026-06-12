package web

import (
	"html/template"
	"strings"
	"testing"
	"time"

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
	choice := turn.ModelChoice{Key: "m1", Provider: "kronk", Model: "model.gguf", Name: "Local Qwen3.6 35B A3B", Detail: "model.gguf · on this box", Badge: "local", Active: true}
	data := homeData{Title: "Balaur", ChatReady: true, ActiveModel: "Local Qwen3.6 35B A3B", ChatPlaceholder: "Speak...", ModelChoices: []turn.ModelChoice{choice}}
	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, "chat_bar", data); err != nil {
		t.Fatalf("chat_bar: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "model-choice-list") || !strings.Contains(out, "Add OpenAI-compatible API") {
		t.Error("chatbar should render the inline model chooser")
	}
	if !strings.Contains(out, `<textarea name="message"`) {
		t.Error("chatbar should render the current textarea input when chat is ready")
	}

	b.Reset()
	data.ChatReady = false
	if err := tmpl.ExecuteTemplate(&b, "chat_bar", data); err != nil {
		t.Fatalf("chat_bar not ready: %v", err)
	}
	if strings.Contains(b.String(), `input type="text"`) {
		t.Error("chatbar should not render the old disabled text input when chat is not ready")
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
		"list":     {"Title": "Tasks", "View": "list", "Buckets": bucketsView{}},
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
