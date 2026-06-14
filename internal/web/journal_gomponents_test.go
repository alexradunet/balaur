package web

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/journalcards"
	"github.com/alexradunet/balaur/internal/ui"
)

// After registering the journalcards feature, the journal and day cards are
// served by their gomponents renderers via the cardInto shim.
func TestJournalDayRenderViaGomponents(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	journalcards.Register(app)
	defer func() {
		ui.UnregisterCard("journal")
		ui.UnregisterCard("day")
	}()

	for _, typ := range []string{"journal", "day"} {
		if _, ok := ui.LookupCard(typ); !ok {
			t.Fatalf("%s not registered via gomponents", typ)
		}
	}

	if j := string(h.cardHTML("journal", nil)); !strings.Contains(j, `id="ucard-journal"`) {
		t.Fatalf("journal card not rendered:\n%s", j)
	}
	if d := string(h.cardHTML("day", nil)); !strings.Contains(d, `id="ucard-day"`) {
		t.Fatalf("day card not rendered:\n%s", d)
	}
}
