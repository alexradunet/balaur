package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

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

func TestComposerDeciding(t *testing.T) {
	got := render(t, ui.Composer(ui.ComposerProps{
		AvatarSrc: "/static/crest.png", Prompt: "How should I log this?",
		Choices: []ui.ComposerChoice{{Label: "A quick note", Hint: "1 line"}, {Label: "A journal entry"}},
	}))
	for _, want := range []string{
		`<div class="composer composer-deciding">`,
		`<div class="composer-kicker">How should I log this?</div>`,
		`<div class="choices-panel composer-choices">`,
		`<span class="choice-label">A quick note</span>`,
		// the embedded manual-input row, keyed after the choices (3rd here)
		`<div class="choice choice-type"><span class="choice-key">3</span><input type="text" placeholder="type your answer…" autocomplete="off">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("deciding composer missing %q in: %s", want, got)
		}
	}
	// deciding mode replaces the draft — no textarea.
	if strings.Contains(got, "<textarea") {
		t.Errorf("deciding composer should not render the draft textarea: %s", got)
	}
}

func TestComposerDecision(t *testing.T) {
	card := g.El("article", g.Attr("class", "kcard"), g.Text("CARD"))
	got := render(t, ui.Composer(ui.ComposerProps{
		AvatarSrc: "/static/crest.png", Prompt: "Shall I keep this?", Decision: card,
	}))
	for _, want := range []string{
		`<div class="composer composer-deciding">`,
		`<div class="composer-kicker">Shall I keep this?</div>`,
		`<div class="composer-decision"><article class="kcard">CARD</article></div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("decision composer missing %q in: %s", want, got)
		}
	}
	// the surfaced card replaces both draft and choices.
	if strings.Contains(got, "<textarea") || strings.Contains(got, "choices-panel") {
		t.Errorf("decision mode should render neither draft nor choices: %s", got)
	}
}
