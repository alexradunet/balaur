package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestModelSwitcher(t *testing.T) {
	got := render(t, chat.ModelSwitcher(chat.ModelSwitcherProps{
		ActiveModel: "gemma3:4b", AvatarSrc: "/static/crest.png",
	}))
	for _, want := range []string{
		`<section class="model-switcher" aria-label="Model">`,
		`<span class="model-switcher-kicker">Model</span>`,
		`<span class="model-current">gemma3:4b</span>`,
		`<a class="model-switcher-manage" href="/focus/settings?section=models">`,
		`<div class="chatbar-profile-link"><span class="balaur-avatar balaur-avatar-soul" aria-hidden="true"><img class="px" src="/static/crest.png" alt="" decoding="async"></span>`,
		`<a href="/focus/settings?section=profile" class="chatbar-profile-href">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("model switcher missing %q in: %s", want, got)
		}
	}
}

func TestModelSwitcherNoModel(t *testing.T) {
	got := render(t, chat.ModelSwitcher(chat.ModelSwitcherProps{AvatarSrc: "/static/crest.png"}))
	if strings.Contains(got, "model-current") {
		t.Errorf("no ActiveModel should omit .model-current: %s", got)
	}
}
