package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestIntParam(t *testing.T) {
	p := map[string]string{
		"limit": "7",
		"days":  "30",
		"zero":  "0",
		"neg":   "-4",
		"empty": "",
		"junk":  "abc",
	}
	cases := []struct {
		name, key string
		def, want int
	}{
		{"present positive", "limit", 6, 7},
		{"multi-digit", "days", 90, 30},
		{"absent key uses default", "missing", 12, 12},
		{"empty string uses default", "empty", 5, 5},
		{"non-numeric uses default", "junk", 5, 5},
		{"zero uses default", "zero", 10, 10},
		{"negative uses default", "neg", 10, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ui.IntParam(p, c.key, c.def); got != c.want {
				t.Fatalf("IntParam(p, %q, %d) = %d, want %d", c.key, c.def, got, c.want)
			}
		})
	}
}

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
