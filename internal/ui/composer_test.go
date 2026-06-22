package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestComposer(t *testing.T) {
	got := render(t, ui.Composer(ui.ComposerProps{
		AvatarSrc: "/static/crest.png", Placeholder: "Speak; I am listening.",
		Tools: []string{"scroll", "tome"},
	}))
	for _, want := range []string{
		`<div class="composer">`,
		`<span class="dlg-corner dlg-corner-tl"></span>`,
		`<button class="composer-tool" type="button" disabled aria-label="scroll (coming soon)"><img src="/static/icons/scroll.png" alt="" decoding="async"></button>`,
		`<button class="composer-tool composer-sound" type="button" disabled aria-label="Sound (coming soon)"><img src="/static/icons/bell.png" alt="" decoding="async"></button>`,
		`<div class="composer-portrait">`,
		`class="balaur-avatar balaur-avatar-soul"`,
		`<form class="composer-form">`,
		`<textarea name="message" placeholder="Speak; I am listening." rows="2" autocomplete="off"></textarea>`,
		`<span class="composer-hint">unsent · enter speaks</span>`,
		`<button class="btn btn-primary btn-sm" type="submit">Send</button>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("composer missing %q in: %s", want, got)
		}
	}
}

func TestComposerDefaults(t *testing.T) {
	got := render(t, ui.Composer(ui.ComposerProps{AvatarSrc: "/static/crest.png"}))
	// default tools = scroll, tome, lens (3) + the sound button
	for _, name := range []string{"scroll", "tome", "lens"} {
		if !strings.Contains(got, `/static/icons/`+name+`.png`) {
			t.Errorf("default tool %q missing: %s", name, got)
		}
	}
}
