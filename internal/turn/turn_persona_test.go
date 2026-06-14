package turn

import (
	"strings"
	"testing"
)

func TestHeadFlavorMainIsEmpty(t *testing.T) {
	if got := headFlavor("Balaur", ""); got != "" {
		t.Errorf("main head flavor = %q, want empty", got)
	}
}

func TestHeadFlavorSpecialistFramesPurpose(t *testing.T) {
	got := headFlavor("Scholar", "explains and researches")
	if !strings.Contains(got, "Scholar") || !strings.Contains(got, "explains and researches") {
		t.Errorf("flavor should name the head and its purpose; got %q", got)
	}
}
