package web

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/journalcards"
	"github.com/alexradunet/balaur/internal/ui"
)

// After registering the journalcards feature, the day card is served by its
// gomponents renderer via the cardInto shim.
func TestDayRenderViaGomponents(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	journalcards.Register(app)
	defer func() {
		ui.UnregisterCard("day")
	}()

	if _, ok := ui.LookupCard("day"); !ok {
		t.Fatal("day not registered via gomponents")
	}

	if d := string(h.cardHTML("day", nil)); !strings.Contains(d, `id="ucard-day"`) {
		t.Fatalf("day card not rendered:\n%s", d)
	}
}
