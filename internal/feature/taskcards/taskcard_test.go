package taskcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
)

func renderTask(t *testing.T, v taskcards.TaskView) string {
	t.Helper()
	var b strings.Builder
	if err := taskcards.TaskCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

func TestTaskCardOpenHasAllActions(t *testing.T) {
	out := renderTask(t, taskcards.TaskView{
		ID: "t1", Title: "Call the notary", Status: "open",
		DueLine: "due Mon, Jan 2 at 09:00", RecurLine: "every day", Notes: "ask about the deed",
	})
	for _, want := range []string{
		`id="tcard-t1"`,
		`class="kcard tcard tcard-open"`,
		"Call the notary",
		"every day",
		"due Mon, Jan 2 at 09:00",
		"ask about the deed",
		`value="done"`, "Done",
		`value="snooze"`, "Snooze", `value="1h"`, `value="tonight"`, `value="tomorrow"`,
		`value="dropped"`, "Drop",
		`data-on:submit__prevent="@post(`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestTaskCardNonOpenShowsStatusNoActions(t *testing.T) {
	out := renderTask(t, taskcards.TaskView{ID: "t2", Title: "Done thing", Status: "done"})
	if strings.Contains(out, "/transition") {
		t.Errorf("non-open task must not render action forms:\n%s", out)
	}
	if !strings.Contains(out, "done") {
		t.Errorf("expected status text:\n%s", out)
	}
}

func TestTaskCardOverdueClass(t *testing.T) {
	out := renderTask(t, taskcards.TaskView{ID: "t3", Title: "Late", Status: "open", DueLine: "overdue 2d", Overdue: true})
	if !strings.Contains(out, "tcard-overdue") {
		t.Errorf("expected tcard-overdue class on the due line:\n%s", out)
	}
}
