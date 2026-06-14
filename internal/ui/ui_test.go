package ui_test

// TestNoWebImports is a compile-time fact, mirroring internal/cards: the ui
// package must never import internal/web. The layering law (spec §4.1) is
// web -> feature -> ui; internal/web already imports internal/ui (the cardInto
// shim), so an internal/web import here would be an import cycle and fail to
// build. If `go test ./internal/ui/...` compiles without a cycle error, the
// boundary holds.

import "testing"

func TestNoWebImports(t *testing.T) {
	t.Log("compile-time verified: internal/ui has no internal/web imports")
}
