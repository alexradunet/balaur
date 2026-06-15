package storybook_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/storybook"
)

func TestBodyRendersAtoms(t *testing.T) {
	var b strings.Builder
	if err := storybook.Body().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		`<h1`,
		`class="btn btn-primary"`,
		`class="btn btn-ghost"`,
		`class="btn btn-wood"`,
		`class="btn btn-primary btn-sm"`,
		`class="tag"`,
		`class="kcard-pips"`,
		`class="pip pip-on"`,
		`class="card"`,
		`class="balaur-avatar`,
		`class="tool-icon"`,
		`class="stitch"`,
		`class="folk-band"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("storybook body missing %q", want)
		}
	}
}
