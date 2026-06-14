package web

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/headscards"
	"github.com/alexradunet/balaur/internal/ui"
)

// TestHeadsRenderViaGomponents verifies that after registering the headscards
// feature, the heads card is served by its gomponents renderer via the
// cardInto shim — end-to-end within the web layer.
func TestHeadsRenderViaGomponents(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	headscards.Register(app)
	defer ui.UnregisterCard("heads")

	if _, ok := ui.LookupCard("heads"); !ok {
		t.Fatal("heads not registered via gomponents")
	}

	if out := string(h.cardHTML("heads", nil)); !strings.Contains(out, `id="ucard-heads"`) {
		t.Fatalf("heads card not rendered:\n%s", out)
	}
}
