package settingscards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/settingscards"
)

func TestSettingsCard(t *testing.T) {
	var b strings.Builder
	if err := settingscards.SettingsCard().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`id="ucard-settings"`,
		`class="kcard ucard ucard-settings"`,
		"Settings",
		`href="/focus/settings?section=profile"`, "Profile",
		`href="/focus/settings?section=models"`, "Models &amp; APIs",
		`href="/focus/settings"`, "open settings →",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
