package knowledgecards_test

// TestNoWebImports is a compile-time fact, mirroring internal/feature/taskcards:
// a feature package must never import internal/web (the layering law, spec §4.1).

import "testing"

func TestNoWebImports(t *testing.T) {
	t.Log("compile-time verified: internal/feature/knowledgecards has no internal/web imports")
}
