// Package uitest provides the shared test helper for rendering a gomponents
// node to its HTML string, so every UI/feature test renders the same way
// without duplicating the buffer-and-fatal boilerplate.
package uitest

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)

// Render renders a gomponents node to its HTML string, failing the test via
// t.Fatalf on a render error.
func Render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}
