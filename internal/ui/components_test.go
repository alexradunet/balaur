package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestErrorStripRendersAndEscapes(t *testing.T) {
	var b strings.Builder
	if err := ui.ErrorStrip(`<script>evil()</script>`).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, `class="card-note card-note-error"`) {
		t.Fatalf("missing error classes: %q", got)
	}
	if strings.Contains(got, "<script>") {
		t.Fatalf("model/user string not escaped (firewall breach): %q", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("expected escaped text: %q", got)
	}
}
