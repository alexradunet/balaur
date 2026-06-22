// Package ui holds Balaur's shared gomponents rendering primitives and the
// card-renderer registry. It imports gomponents and pocketbase/core only —
// never internal/web — so feature packages can depend on it without a cycle.
package ui

import "strconv"

// Clip truncates s to n runes, appending an ellipsis when shortened. It counts
// runes, not bytes, so multi-byte text never renders a broken character.
func Clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// IntParam reads p[key] as a positive integer, returning def when the key is
// absent, empty, unparseable, or not greater than zero. Card params reach
// renderers already clamped by cards.Validate, so requiring n > 0 here only
// ever rejects malformed direct callers — every clamped value is >= 1.
func IntParam(p map[string]string, key string, def int) int {
	if n, err := strconv.Atoi(p[key]); err == nil && n > 0 {
		return n
	}
	return def
}
