package taskcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
)

func render(t *testing.T, v taskcards.TodayView) string {
	t.Helper()
	var b strings.Builder
	if err := taskcards.TodayCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

func TestTodayCardRendersRows(t *testing.T) {
	out := render(t, taskcards.TodayView{Rows: []taskcards.TodayRow{
		{ID: "abc123", Title: "Call the notary", Status: "open", DueLine: "due Mon, Jan 2 at 09:00"},
	}})

	for _, want := range []string{
		`id="ucard-today"`,
		`class="kcard ucard ucard-today"`,
		`id="urow-today-abc123"`,
		"Call the notary",
		"due Mon, Jan 2 at 09:00",
		`data-on:submit__prevent="@post(&#39;/ui/tasks/abc123/transition&#39;, {contentType:&#39;form&#39;})"`,
		`name="to"`, `value="done"`,
		`name="src"`, `value="today"`,
		"all quests →",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestTodayCardEmptyState(t *testing.T) {
	out := render(t, taskcards.TodayView{})
	if !strings.Contains(out, "Nothing due today.") {
		t.Errorf("missing empty state in:\n%s", out)
	}
	if strings.Contains(out, "ucard-row") {
		t.Errorf("empty view should render no rows:\n%s", out)
	}
}

func TestTodayCardNonOpenHasNoDoneForm(t *testing.T) {
	out := render(t, taskcards.TodayView{Rows: []taskcards.TodayRow{
		{ID: "x", Title: "Already done", Status: "done"},
	}})
	if strings.Contains(out, "/transition") {
		t.Errorf("non-open row must not render the done form:\n%s", out)
	}
}
