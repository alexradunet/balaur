package web

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/ui"
)

// After registering the settingscards feature, the settings tile is served by
// its gomponents renderer via the cardInto shim.
func TestSettingsRenderViaGomponents(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app}

	settingscards.Register(app)
	defer ui.UnregisterCard("settings")

	if _, ok := ui.LookupCard("settings"); !ok {
		t.Fatal("settings not registered via gomponents")
	}
	if s := renderNodeHTML(h.cardHTML("settings", nil)); !strings.Contains(s, `id="ucard-settings"`) {
		t.Fatalf("settings card not rendered:\n%s", s)
	}
}
