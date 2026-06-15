package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestChatBar(t *testing.T) {
	got := render(t, chat.ChatBar(chat.ChatBarProps{
		ActiveHead:  "Balaur",
		Heads:       []chat.Head{{Name: "Balaur", AvatarSrc: "/static/crest.png", Active: true}},
		ActiveModel: "gemma3:4b",
		AvatarSrc:   "/static/crest.png",
	}))
	for _, want := range []string{
		`<div class="chatbar chatbar-slim">`,
		`<section class="head-switcher" aria-label="Head">`,
		`<section class="model-switcher" aria-label="Model">`,
		`<span class="head-switcher-current">Balaur</span>`,
		`<span class="model-current">gemma3:4b</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("chatbar missing %q in: %s", want, got)
		}
	}
}
