// Package storetest provides the shared test-app constructor so every
// package's tests boot the same Balaur schema without duplicating setup.
package storetest

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	// Blank import registers the Balaur schema migrations, which
	// tests.NewTestApp runs during bootstrap.
	_ "github.com/alexradunet/balaur/migrations"
)

// NewApp builds a throwaway PocketBase app with the Balaur schema applied,
// rooted in t.TempDir() and cleaned up with the test.
func NewApp(t *testing.T) core.App {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}
