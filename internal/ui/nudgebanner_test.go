package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestNudgeBanner(t *testing.T) {
	got := render(t, ui.NudgeBanner(ui.NudgeProps{
		When: "18:00", Message: "The tomatoes thirst.",
		Replies: []ui.NudgeReply{{Label: "It is done.", Hint: "mark done"}, {Label: "At nightfall.", Hint: "snooze · 21:00"}},
	}))
	for _, want := range []string{
		`<div class="nudge">`,
		`<img class="nudge-icon" src="/static/icons/bell.png" alt="" decoding="async">`,
		`<span class="nudge-kicker">Nudge</span>`,
		`<span class="nudge-when">18:00</span>`,
		`<p class="nudge-msg">The tomatoes thirst.</p>`,
		`<button class="nudge-reply" type="button"><span>It is done.</span><span class="nudge-reply-hint">mark done</span></button>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("nudge banner missing %q in: %s", want, got)
		}
	}
}
