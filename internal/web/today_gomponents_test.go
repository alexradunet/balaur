package web

import (
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

// After registering the taskcards feature, cardInto's shim renders the today
// card via the gomponents component (not the legacy template), showing live data.
func TestTodayRendersViaGomponentsAfterRegister(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	// Seed an open task due at NOON TODAY (local) so it deterministically lands
	// in today's bucket (a task with no due date would bucket as "someday" and
	// never reach the today card). now.Add(2h) was flaky: near local midnight in
	// a non-UTC zone it rolls into the next calendar day and buckets as upcoming.
	now := time.Now()
	noon := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	if _, err := tasks.Create(app, tasks.CreateOpts{
		Title:  "Call the notary",
		Due:    noon,
		Source: "test",
	}); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	taskcards.Register(app)
	defer ui.UnregisterCard("today") // keep the global registry clean for other tests

	out := string(h.cardHTML("today", nil))
	if !strings.Contains(out, `id="ucard-today"`) {
		t.Fatalf("today card not rendered:\n%s", out)
	}
	if !strings.Contains(out, "Call the notary") {
		t.Fatalf("seeded task missing from today card (gomponents path not used?):\n%s", out)
	}
	// The gomponents path renders the datastar form attribute verbatim.
	if !strings.Contains(out, "data-on:submit__prevent") {
		t.Fatalf("expected the gomponents done-form attribute:\n%s", out)
	}
}
