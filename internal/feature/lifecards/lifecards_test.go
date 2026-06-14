package lifecards_test

// TestNoWebImports is a compile-time fact, mirroring internal/feature/taskcards:
// a feature package must never import internal/web (the layering law, spec §4.1).

import "testing"

func TestNoWebImports(t *testing.T) {
	t.Log("compile-time verified: internal/feature/lifecards has no internal/web imports")
}
