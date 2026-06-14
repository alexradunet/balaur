package feature_test

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature"
)

func TestRegistryRegistersAndUnregistersAll(t *testing.T) {
	feature.Reset() // isolate from any init()-registered features
	t.Cleanup(feature.Reset)

	var registered, unregistered int
	feature.Add(feature.Funcs(
		func(core.App) { registered++ },
		func() { unregistered++ },
	))
	feature.Add(feature.Funcs(
		func(core.App) { registered++ },
		func() { unregistered++ },
	))

	feature.RegisterAll(nil)
	if registered != 2 {
		t.Fatalf("RegisterAll: registered = %d, want 2", registered)
	}

	feature.UnregisterAll()
	if unregistered != 2 {
		t.Fatalf("UnregisterAll: unregistered = %d, want 2", unregistered)
	}
}
