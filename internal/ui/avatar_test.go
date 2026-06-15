package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestAvatarDecorativeDefaults(t *testing.T) {
	got := render(t, ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png"}))
	for _, want := range []string{
		`class="balaur-avatar balaur-avatar-balaur"`,
		`data-kind="balaur"`,
		`data-state="idle"`,
		`style="--avatar-size:54px"`,
		`aria-hidden="true"`,
		`<img src="/static/avatars/balaur-01.png" alt="" decoding="async">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("avatar missing %q in: %s", want, got)
		}
	}
}

func TestAvatarNamedNotHidden(t *testing.T) {
	got := render(t, ui.Avatar(ui.AvatarProps{Src: "/x.png", Kind: "soul", State: "thinking", Alt: "Wise", Size: 96}))
	if strings.Contains(got, "aria-hidden") {
		t.Errorf("named avatar (Alt set) must not be aria-hidden: %s", got)
	}
	for _, want := range []string{`balaur-avatar-soul`, `data-state="thinking"`, `style="--avatar-size:96px"`, `alt="Wise"`} {
		if !strings.Contains(got, want) {
			t.Errorf("avatar missing %q in: %s", want, got)
		}
	}
}

func TestIcon(t *testing.T) {
	got := render(t, ui.Icon("scroll"))
	if want := `<img class="tool-icon" src="/static/icons/scroll.png" alt="">`; got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
