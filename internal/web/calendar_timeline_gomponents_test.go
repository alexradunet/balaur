package web

import (
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

// After registering the taskcards feature, the calendar and timeline cards are
// served by their gomponents renderers via the cardInto shim.
func TestCalendarTimelineRenderViaGomponents(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}
	if _, err := tasks.Create(app, tasks.CreateOpts{
		Title: "Ship it", Due: time.Now().Add(24 * time.Hour), Source: "test",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	taskcards.Register(app)
	defer func() {
		for _, typ := range []string{"today", "quests", "calendar", "timeline"} {
			ui.UnregisterCard(typ)
		}
	}()

	for _, typ := range []string{"calendar", "timeline"} {
		if _, ok := ui.LookupCard(typ); !ok {
			t.Fatalf("%s not registered via gomponents", typ)
		}
	}

	if cal := string(h.cardHTML("calendar", nil)); !strings.Contains(cal, `id="ucard-calendar"`) {
		t.Fatalf("calendar card not rendered:\n%s", cal)
	}
	if tl := string(h.cardHTML("timeline", nil)); !strings.Contains(tl, `id="ucard-timeline"`) {
		t.Fatalf("timeline card not rendered:\n%s", tl)
	}
}
