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
		`<div class="cmsg cmsg-balaur">`,
		`<div class="cmsg-row">`,
		// Balaur: portrait then panel.
		`<div class="cmsg-portrait"><img src="/static/crest.png" alt="" decoding="async"></div><div class="cmsg-panel">`,
		`<div class="cmsg-name">Balaur</div>`,
		`<div class="cmsg-body cmsg-md"><p>Hello there.</p>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("balaur message missing %q in: %s", want, got)
		}
	}
}

func TestMessageBalaurOrigin(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", Origin: "nudge", AvatarSrc: "/static/crest.png", Content: "x"}))
	if !strings.Contains(got, `<div class="cmsg-name">Balaur · nudge</div>`) {
		t.Errorf("origin should render after the name: %s", got)
	}
}

func TestMessageUserDefaultName(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{Role: "user", AvatarSrc: "/static/crest.png", Content: "Hi"}))
	for _, want := range []string{
		`<div class="cmsg cmsg-user">`,
		// Owner: panel then portrait (mirrored).
		`<div class="cmsg-row"><div class="cmsg-panel">`,
		`<div class="cmsg-name">You</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("user message missing %q in: %s", want, got)
		}
	}
}

func TestMessageBalaurMarkdown(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{
		Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png",
		Content: "**bold** and a list:\n\n- one\n- two",
	}))
	for _, want := range []string{
		`<strong>bold</strong>`,
		`<ul>`,
		`<li>one</li>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("balaur markdown not rendered, missing %q in: %s", want, got)
		}
	}
	if strings.Contains(got, "**bold**") {
		t.Errorf("raw markdown leaked into output: %s", got)
	}
}

func TestMessagePending(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Pending: true}))
	if !strings.Contains(got, `<div class="cmsg cmsg-balaur cmsg-pending">`) {
		t.Errorf("pending should add cmsg-pending: %s", got)
	}
	if !strings.Contains(got, `<div class="cmsg-name">Balaur</div><span class="thinking thinking-dots">thinking</span>`) {
		t.Errorf("pending empty body should be thinking-dots: %s", got)
	}
}
