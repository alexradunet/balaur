package taskcards_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/ui"
)

// taskcards' init() (run automatically because this is its own test binary)
// adds its Feature to the registry; RegisterAll then registers the today card
// in the ui registry.
func TestTaskcardsSelfRegisters(t *testing.T) {
	feature.RegisterAll(nil) // app captured but not invoked; we only check registration
	t.Cleanup(feature.UnregisterAll)

	if _, ok := ui.LookupCard("today"); !ok {
		t.Fatal("taskcards did not self-register the today card via the feature registry")
	}
}
