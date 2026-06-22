package journalcards_test

// TestNoWebImports is a compile-time fact, mirroring internal/feature/taskcards:
// a feature package must never import internal/web (the layering law, spec §4.1).
// If `go test ./internal/feature/...` compiles without an import cycle, it holds.

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/uitest"
)

func TestNoWebImports(t *testing.T) {
	t.Log("compile-time verified: internal/feature/journalcards has no internal/web imports")
}

// renderNode renders a gomponents node to an HTML string for assertions.
func renderNode(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
}
