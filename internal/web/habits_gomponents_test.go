package web

import (
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

// After registering the taskcards feature, the habits card is served by its
// gomponents renderer via the cardInto shim.
func TestHabitsRendersViaGomponents(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	// Seed a recurring task (daily) with a due date — recurring tasks require
	// a due anchor (see tasks.Create validation).
	if _, err := tasks.Create(app, tasks.CreateOpts{
		Title:  "Morning run",
		Recur:  "daily",
		Due:    time.Now().Add(2 * time.Hour),
		Source: "test",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	taskcards.Register(app)
	defer func() {
		for _, typ := range []string{"today", "quests", "calendar", "timeline", "habits"} {
			ui.UnregisterCard(typ)
		}
	}()

	if _, ok := ui.LookupCard("habits"); !ok {
		t.Fatal("habits not registered via gomponents")
	}

	out := string(h.cardHTML("habits", nil))
	if !strings.Contains(out, `id="ucard-habits"`) {
		t.Fatalf("habits card root id missing:\n%s", out)
	}
	if !strings.Contains(out, "Morning run") {
		t.Fatalf("seeded habit title missing from habits card:\n%s", out)
	}
}
