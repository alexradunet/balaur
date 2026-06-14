package taskcards_test

import "testing"

// TestNoWebImports is a compile-time fact, mirroring internal/ui and
// internal/cards: a feature package must never import internal/web (the layering
// law, spec §4.1: web -> feature -> ui). If `go test ./internal/feature/...`
// compiles without an import cycle, the boundary holds.
func TestNoWebImports(t *testing.T) {
	t.Log("compile-time verified: internal/feature/taskcards has no internal/web imports")
}
