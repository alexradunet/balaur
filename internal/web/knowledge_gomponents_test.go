package web

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/knowledgecards"
	"github.com/alexradunet/balaur/internal/ui"
)

// After registering the knowledgecards feature, the memory and skills cards are
// served by their gomponents renderers via the cardInto shim, in both summary
// and manage modes.
func TestKnowledgeRenderViaGomponents(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	knowledgecards.Register(app)
	defer func() {
		ui.UnregisterCard("memory")
		ui.UnregisterCard("skills")
	}()

	for _, typ := range []string{"memory", "skills"} {
		if _, ok := ui.LookupCard(typ); !ok {
			t.Fatalf("%s not registered via gomponents", typ)
		}
	}

	cases := []struct {
		typ, summaryID string
	}{
		{"memory", "ucard-memory"},
		{"skills", "ucard-skills"},
	}
	for _, c := range cases {
		if s := string(h.cardHTML(c.typ, nil)); !strings.Contains(s, `id="`+c.summaryID+`"`) {
			t.Fatalf("%s summary not rendered:\n%s", c.typ, s)
		}
		if m := string(h.cardHTML(c.typ, map[string]string{"mode": "manage"})); !strings.Contains(m, "ucard-manage") {
			t.Fatalf("%s manage not rendered:\n%s", c.typ, m)
		}
	}
}
