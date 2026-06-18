package chat_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestPanelWithTitleIconBody(t *testing.T) {
	got := render(t, chat.Panel(chat.PanelProps{
		Title: "Quest Log", Icon: "scroll", Body: g.Text("BODY"),
	}))
	for _, want := range []string{
		"panel-head",
		"panel-head-title",
		"Quest Log",
		`id="panel-body"`,
		"panel-close",
		"/static/icons/scroll.png",
		"BODY",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("panel with title/icon/body missing %q in: %s", want, got)
		}
	}
}

func TestPanelEmpty(t *testing.T) {
	got := render(t, chat.Panel(chat.PanelProps{}))
	if !strings.Contains(got, "panel-empty") {
		t.Errorf("empty panel missing panel-empty: %s", got)
	}
	if !strings.Contains(got, "Pick a domain from the rail") {
		t.Errorf("empty panel missing placeholder copy: %s", got)
	}
	if strings.Contains(got, "panel-head") {
		t.Errorf("empty panel must not contain panel-head: %s", got)
	}
}

func TestPanelRootID(t *testing.T) {
	got := render(t, chat.Panel(chat.PanelProps{Title: "X", Body: g.Text("B")}))
	if !strings.Contains(got, `id="panel-inner"`) {
		t.Errorf("panel root must have id=panel-inner: %s", got)
	}
}

func TestArtifactChipClickable(t *testing.T) {
	got := render(t, chat.ArtifactChip(chat.ArtifactChipProps{
		Title:     "Quest Log",
		Icon:      "scroll",
		ReopenURL: "/ui/show/quests",
	}))
	for _, want := range []string{
		"<a",
		"art-chip",
		"data-on:click__prevent",
		"/ui/show/quests",
		"open ▸",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("clickable chip missing %q in: %s", want, got)
		}
	}
}

func TestArtifactChipNonClickable(t *testing.T) {
	got := render(t, chat.ArtifactChip(chat.ArtifactChipProps{
		Title: "Cluster",
	}))
	if strings.Contains(got, "data-on:click") {
		t.Errorf("non-clickable chip must not contain data-on:click: %s", got)
	}
	if !strings.Contains(got, "shown earlier") {
		t.Errorf("non-clickable chip must contain 'shown earlier': %s", got)
	}
	if strings.Contains(got, "<a") {
		t.Errorf("non-clickable chip must not be an <a> element: %s", got)
	}
}
