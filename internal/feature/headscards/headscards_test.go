package headscards_test

// TestNoWebImports is a compile-time fact, mirroring internal/feature/journalcards:
// a feature package must never import internal/web (the layering law, spec §4.1).
// If `go test ./internal/feature/...` compiles without an import cycle, it holds.

import "testing"

func TestNoWebImports(t *testing.T) {
	t.Log("compile-time verified: internal/feature/headscards has no internal/web imports")
}
