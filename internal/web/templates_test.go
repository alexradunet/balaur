package web

import (
	"html/template"
	"strings"
	"testing"
	"time"

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

func TestNowLineGroundsTheMoment(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Bucharest")
	if err != nil {
		t.Skipf("tzdata: %v", err)
	}
	now := time.Date(2026, 6, 11, 14, 32, 0, 0, loc)
	line := nowLine(now)
	for _, want := range []string{"Thursday, June 11 2026", "14:32", "UTC+03:00", "this moment"} {
		if !strings.Contains(line, want) {
			t.Errorf("now line missing %q in: %s", want, line)
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
