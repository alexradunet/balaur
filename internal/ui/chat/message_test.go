package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestMessageBalaur(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{
		Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Content: "Hello there.",
	}))
	for _, want := range []string{
		`<div class="msg msg-balaur msg-with-avatar">`,
		`<figure class="portrait">`,
		`<span class="balaur-avatar balaur-avatar-balaur" data-kind="balaur" data-state="idle"`,
		`<img src="/static/crest.png" alt="" decoding="async">`,
		`<figcaption class="who">Balaur</figcaption>`,
		`<div class="msg-main"><div class="body">Hello there.</div></div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("balaur message missing %q in: %s", want, got)
		}
	}
}

func TestMessageBalaurOrigin(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", Origin: "nudge", AvatarSrc: "/static/crest.png", Content: "x"}))
	if !strings.Contains(got, `<figcaption class="who">Balaur · nudge</figcaption>`) {
		t.Errorf("origin should render after the name: %s", got)
	}
}

func TestMessageUserDefaultName(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{Role: "user", AvatarSrc: "/static/crest.png", Content: "Hi"}))
	for _, want := range []string{
		`<div class="msg msg-user msg-with-avatar">`,
		`class="balaur-avatar balaur-avatar-soul" data-kind="soul"`,
		`<figcaption class="who">You</figcaption>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("user message missing %q in: %s", want, got)
		}
	}
}

func TestMessagePending(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Pending: true}))
	if !strings.Contains(got, `<div class="msg msg-balaur msg-with-avatar msg-pending">`) {
		t.Errorf("pending should add msg-pending: %s", got)
	}
	if !strings.Contains(got, `data-state="thinking"`) {
		t.Errorf("pending balaur avatar should be thinking: %s", got)
	}
	if !strings.Contains(got, `<div class="body"><span class="thinking thinking-dots">thinking</span></div>`) {
		t.Errorf("pending empty body should be thinking-dots: %s", got)
	}
}
