package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestToolRow(t *testing.T) {
	got := render(t, chat.ToolRow(chat.ToolRowProps{
		Tool: "task_add", Icon: "scroll", AvatarSrc: "/static/crest.png",
		Content: "added task: water the tomatoes · every 2 days 18:00",
	}))
	for _, want := range []string{
		// Same speech-panel frame as chat.Message, marked cmsg-tool.
		`<div class="cmsg cmsg-balaur cmsg-tool">`,
		`<div class="cmsg-portrait"><img src="/static/crest.png" alt="" decoding="async"></div><div class="cmsg-panel">`,
		`<div class="cmsg-name">Balaur · Tool</div>`,
		`<img class="tool-icon" src="/static/icons/scroll.png" alt="">tool · task_add`,
		`added task: water the tomatoes · every 2 days 18:00`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("tool row missing %q in: %s", want, got)
		}
	}
}

// TestToolRowWho: the nameplate follows the active head's name, suffixed "· Tool".
func TestToolRowWho(t *testing.T) {
	got := render(t, chat.ToolRow(chat.ToolRowProps{Tool: "x", Icon: "scroll", Who: "Origin"}))
	if !strings.Contains(got, `<div class="cmsg-name">Origin · Tool</div>`) {
		t.Errorf("nameplate should read \"{Who} · Tool\": %s", got)
	}
}

// TestToolRowPending: a running tool shows cmsg-pending + a "running" ellipsis,
// and not the final "tool ·" indicator.
func TestToolRowPending(t *testing.T) {
	got := render(t, chat.ToolRow(chat.ToolRowProps{
		Tool: "settings", Icon: "key", AvatarSrc: "/static/crest.png", Pending: true,
	}))
	for _, want := range []string{
		`<div class="cmsg cmsg-balaur cmsg-tool cmsg-pending">`,
		`settings · <span class="thinking thinking-dots">running</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("pending tool row missing %q in: %s", want, got)
		}
	}
	if strings.Contains(got, "tool · settings") {
		t.Errorf("pending tool row must not show the final 'tool ·' indicator: %s", got)
	}
}

// TestToolRowArgsReasoning: the call arguments and the model's reasoning render
// in collapsed folds, escaped verbatim — the audit trail never executes markup.
func TestToolRowArgsReasoning(t *testing.T) {
	got := render(t, chat.ToolRow(chat.ToolRowProps{
		Tool: "task_add", Icon: "scroll", AvatarSrc: "/static/crest.png",
		Content:   "added task",
		Args:      `{"title":"<b>x</b>"}`,
		Reasoning: "because <reasons>",
	}))
	for _, want := range []string{
		`<details class="tool-args"><summary>arguments</summary><pre>`,
		`<details class="tool-reasoning"><summary>reasoning</summary><pre>`,
		`&lt;b&gt;x&lt;/b&gt;`,    // args escaped, not raw markup
		`because &lt;reasons&gt;`, // reasoning escaped
	} {
		if !strings.Contains(got, want) {
			t.Errorf("tool row missing %q in: %s", want, got)
		}
	}
	if strings.Contains(got, "<b>x</b>") {
		t.Errorf("args must be escaped, not raw markup: %s", got)
	}
}

// TestToolRowNoFolds: with no args/reasoning, no folds render — the trail stays
// compact (only the tool line + result).
func TestToolRowNoFolds(t *testing.T) {
	got := render(t, chat.ToolRow(chat.ToolRowProps{Tool: "x", Icon: "scroll", Content: "done"}))
	if strings.Contains(got, "tool-args") || strings.Contains(got, "tool-reasoning") {
		t.Errorf("no folds expected when args/reasoning empty: %s", got)
	}
}

// TestToolRowChip: a surfaced artifact chip rides inside the tool card body.
func TestToolRowChip(t *testing.T) {
	chip := chat.ArtifactChip(chat.ArtifactChipProps{Title: "Settings", Icon: "key", ReopenURL: "/ui/show/settings"})
	got := render(t, chat.ToolRow(chat.ToolRowProps{
		Tool: "card_show", Icon: "key", AvatarSrc: "/static/crest.png",
		Content: "showing the Settings card", Chip: chip,
	}))
	if !strings.Contains(got, "art-chip") || !strings.Contains(got, "Settings") {
		t.Errorf("tool row should embed the artifact chip: %s", got)
	}
	if !strings.Contains(got, "open ▸") {
		t.Errorf("clickable chip should render 'open ▸': %s", got)
	}
	// The chip must sit inside the panel body, not as a sibling of the row.
	if strings.Contains(got, `</div></div></div><a class="art-chip"`) {
		t.Errorf("chip should be nested in the body, not a loose sibling: %s", got)
	}
}
