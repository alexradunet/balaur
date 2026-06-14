package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestClip(t *testing.T) {
	cases := []struct {
		name, in string
		n        int
		want     string
	}{
		{"short unchanged", "hello", 10, "hello"},
		{"exact length unchanged", "hello", 5, "hello"},
		{"truncated with ellipsis", "hello world", 5, "hello…"},
		{"multibyte boundary safe", "héllo", 3, "hél…"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ui.Clip(c.in, c.n); got != c.want {
				t.Fatalf("Clip(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
			}
		})
	}
}
