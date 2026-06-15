package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestHeadSwitcher(t *testing.T) {
	got := render(t, chat.HeadSwitcher(chat.HeadSwitcherProps{
		ActiveHead: "Balaur",
		Heads: []chat.Head{
			{Name: "Balaur", AvatarSrc: "/static/crest.png", Active: true},
			{Name: "Scholar", AvatarSrc: "/static/crest.png"},
		},
	}))
	for _, want := range []string{
		`<section class="head-switcher" aria-label="Head">`,
		`<span class="model-switcher-kicker">Head</span>`,
		`<span class="head-switcher-current">Balaur</span>`,
		`<ul class="head-switcher-list">`,
		`<li><button class="head-switcher-choice head-switcher-choice-active" type="button" aria-current="true"><img class="px" src="/static/crest.png" alt="" decoding="async"><span>Balaur</span></button></li>`,
		`<li><button class="head-switcher-choice" type="button"><img class="px" src="/static/crest.png" alt="" decoding="async"><span>Scholar</span></button></li>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("head switcher missing %q in: %s", want, got)
		}
	}
}
