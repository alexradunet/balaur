package storybook

import (
	"testing"

	// Blank-import every feature package so their init() self-registrations run.
	// feature.RegisterAll below then wires each feature's renderer into the ui
	// registry — the same path web.Register takes at startup.
	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/feature"
	_ "github.com/alexradunet/balaur/internal/feature/all"
	"github.com/alexradunet/balaur/internal/ui"
)

// TestEverySpecHasARenderer guards the seam between the two card registries,
// which are coupled only by string keys: cards.All() is the typed source of
// truth (internal/cards), ui.LookupCard resolves the gomponents renderer
// (internal/ui). A spec with no renderer compiles and passes the rest of the
// suite, then fails at runtime in cardSizeInto ("unhandled card type %q") the
// first time the card is summoned. This turns that into a fast unit failure.
//
// Exception: "chronicle" is the one spec whose renderer is registered in
// internal/web/web.go (not a feature package, and not at init time), so
// feature.RegisterAll does not wire it here. We skip it deliberately.
func TestEverySpecHasARenderer(t *testing.T) {
	feature.RegisterAll(nil) // app captured but not invoked; we only check registration
	t.Cleanup(feature.UnregisterAll)

	for _, spec := range cards.All() {
		if spec.Type == "chronicle" {
			continue // renderer lives in internal/web/web.go, not a feature package
		}
		if _, ok := ui.LookupCard(spec.Type); !ok {
			t.Errorf("card spec %q has no registered renderer — a feature package must call ui.RegisterCard(%q, …) (or, if its renderer lives in internal/web like chronicle, add it to the skip list here)", spec.Type, spec.Type)
		}
	}
}
