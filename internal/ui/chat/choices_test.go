package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestChoices(t *testing.T) {
	got := render(t, chat.Choices(chat.ChoicesProps{
		Prompt:        "How shall we proceed?",
		Nonce:         "abc123",
		OwnerName:     "Alex",
		SoulAvatarSrc: "/static/soul.png",
		Choices: []chat.ChoiceItem{
			{Label: "Option A"},
			{Label: "Option B", Hint: "recommended"},
		},
	}))

	for _, want := range []string{
		`id="choices-abc123"`,
		`class="choices-panel"`,
		`class="choices-kicker"`,
		`How shall we proceed?`,
		`data-on:click="$message = el.dataset.label; @post(&#39;/ui/chat&#39;)"`,
		`<span class="choice-key">1</span>`,
		`<span class="choice-key">2</span>`,
		`<span class="choice-label">Option A</span>`,
		`<span class="choice-label">Option B</span>`,
		`<span class="choice-hint">recommended</span>`,
		`Alex`,
		`/static/soul.png`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("choices missing %q in:\n%s", want, got)
		}
	}

	// Exactly two choice buttons.
	if count := strings.Count(got, `class="choice"`); count != 2 {
		t.Errorf("expected 2 choice buttons, got %d in:\n%s", count, got)
	}
}

// TestChoicesNoHint: a choice with empty Hint must not render a choice-hint span.
func TestChoicesNoHint(t *testing.T) {
	got := render(t, chat.Choices(chat.ChoicesProps{
		Nonce: "n1",
		Choices: []chat.ChoiceItem{
			{Label: "plain"},
		},
	}))

	if strings.Contains(got, "choice-hint") {
		t.Errorf("empty Hint must not render choice-hint span: %s", got)
	}
}
