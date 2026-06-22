package web

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/lifecards"
	"github.com/alexradunet/balaur/internal/ui"
)

// After registering the lifecards feature, the measure, lines, and lifelog cards
// are served by their gomponents renderers via the cardInto shim.
func TestLifeRenderViaGomponents(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app}

	lifecards.Register(app)
	defer func() {
		for _, typ := range []string{"measure", "lines", "lifelog"} {
			ui.UnregisterCard(typ)
		}
	}()

	for _, typ := range []string{"measure", "lines", "lifelog"} {
		if _, ok := ui.LookupCard(typ); !ok {
			t.Fatalf("%s not registered via gomponents", typ)
		}
	}

	// measure + lines require a kind param (cards.Validate enforces it).
	if s := renderNodeHTML(h.cardHTML("measure", map[string]string{"kind": "weight"})); !strings.Contains(s, `id="ucard-measure"`) {
		t.Fatalf("measure not rendered:\n%s", s)
	}
	if s := renderNodeHTML(h.cardHTML("lines", map[string]string{"kind": "mood"})); !strings.Contains(s, `id="ucard-lines"`) {
		t.Fatalf("lines not rendered:\n%s", s)
	}
	if s := renderNodeHTML(h.cardHTML("lifelog", nil)); !strings.Contains(s, `id="ucard-lifelog"`) {
		t.Fatalf("lifelog not rendered:\n%s", s)
	}
}
