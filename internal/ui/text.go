// Package ui holds Balaur's shared gomponents rendering primitives and the
// card-renderer registry. It imports gomponents and pocketbase/core only —
// never internal/web — so feature packages can depend on it without a cycle.
package ui

// Clip truncates s to n runes, appending an ellipsis when shortened. It counts
// runes, not bytes, so multi-byte text never renders a broken character.
func Clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
