package web

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

// A registered gomponents renderer overrides the legacy switch for its type.
func TestCardIntoShimOverridesLegacy(t *testing.T) {
	ui.RegisterCard("__ph0_probe", func(ui.CardSize, map[string]string) (g.Node, error) {
		return g.Text("PROBE-OK"), nil
	})
	defer ui.UnregisterCard("__ph0_probe")

	h := &handlers{}
	var b strings.Builder
	if err := h.cardInto(&b, "__ph0_probe", nil); err != nil {
		t.Fatalf("cardInto: %v", err)
	}
	if b.String() != "PROBE-OK" {
		t.Fatalf("shim not used; got %q", b.String())
	}
}

// An unregistered type still reaches the legacy switch (here: its default).
func TestCardIntoFallsBackForUnregistered(t *testing.T) {
	h := &handlers{}
	var b strings.Builder
	err := h.cardInto(&b, "__ph0_unknown", nil)
	if err == nil || !strings.Contains(err.Error(), "unhandled card type") {
		t.Fatalf("expected unhandled-type fallback error, got %v", err)
	}
}
