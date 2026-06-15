package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestDialogueChoices(t *testing.T) {
	got := render(t, chat.DialogueChoices(chat.ChoicesProps{
		Prompt:    "How should I log this?",
		AvatarSrc: "/static/crest.png",
		Choices: []chat.Choice{
			{Label: "As a quick note", Hint: "1 line"},
			{Label: "As a full entry"},
		},
	}))
	for _, want := range []string{
		`<div class="choices">`,
		`<div class="choices-panel"><div class="choices-kicker">How should I log this?</div>`,
		`<button class="choice" type="button"><span class="choice-key">1</span><span class="choice-label">As a quick note</span><span class="choice-hint">1 line</span></button>`,
		`<button class="choice" type="button"><span class="choice-key">2</span><span class="choice-label">As a full entry</span></button>`,
		`<figure class="portrait">`,
		`class="balaur-avatar balaur-avatar-soul"`,
		`<figcaption class="who">You</figcaption>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("choices missing %q in: %s", want, got)
		}
	}
}
