package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestMessageDraft(t *testing.T) {
	got := render(t, chat.MessageDraft(chat.DraftProps{
		AvatarSrc:   "/static/crest.png",
		Placeholder: "Speak; I am listening.",
	}))
	for _, want := range []string{
		`<div class="msg msg-user msg-draft">`,
		`class="balaur-avatar balaur-avatar-soul"`,
		`<figcaption class="who">You</figcaption>`,
		`<div class="msg-main"><form class="chat-form">`,
		`<textarea name="message" placeholder="Speak; I am listening." rows="2" autocomplete="off"></textarea>`,
		`<div class="msg-draft-foot"><span class="msg-draft-hint">enter to speak · shift+enter for a new line</span><button class="btn btn-primary btn-sm" type="submit">Speak</button></div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("draft missing %q in: %s", want, got)
		}
	}
}

func TestMessageDraftOverrides(t *testing.T) {
	got := render(t, chat.MessageDraft(chat.DraftProps{AvatarSrc: "/static/crest.png", Placeholder: "x", Who: "Alex", Hint: "go", SendLabel: "Send", Value: "draft text"}))
	if !strings.Contains(got, `<figcaption class="who">Alex</figcaption>`) {
		t.Errorf("Who override missing: %s", got)
	}
	if !strings.Contains(got, `>draft text</textarea>`) {
		t.Errorf("Value should be the textarea content: %s", got)
	}
	if !strings.Contains(got, `<span class="msg-draft-hint">go</span>`) || !strings.Contains(got, `type="submit">Send</button>`) {
		t.Errorf("Hint/SendLabel overrides missing: %s", got)
	}
}
