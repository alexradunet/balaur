package chat_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestArtifactExpanded(t *testing.T) {
	got := render(t, chat.Artifact(chat.ArtifactProps{
		Title: "Quests", Icon: "scroll", Body: g.Text("BODY"),
	}))
	for _, want := range []string{
		"artifact-head",
		"artifact-head-title",
		"Quests",
		"artifact-body",
		"/static/icons/scroll.png",
		"BODY",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expanded artifact missing %q in: %s", want, got)
		}
	}
	if strings.Contains(got, "artifact--collapsed") {
		t.Errorf("expanded artifact must not contain artifact--collapsed: %s", got)
	}
}

func TestArtifactCollapsed(t *testing.T) {
	got := render(t, chat.Artifact(chat.ArtifactProps{
		Title: "Quests", Icon: "scroll", Collapsed: true, Body: g.Text("BODY"),
	}))
	if !strings.Contains(got, "artifact--collapsed") {
		t.Errorf("collapsed artifact missing artifact--collapsed: %s", got)
	}
}

func TestArtifactNoIcon(t *testing.T) {
	got := render(t, chat.Artifact(chat.ArtifactProps{
		Title: "Memory", Body: g.Text("BODY"),
	}))
	if !strings.Contains(got, "Memory") {
		t.Errorf("no-icon artifact missing title Memory: %s", got)
	}
	if strings.Contains(got, "/static/icons/") {
		t.Errorf("no-icon artifact must not contain /static/icons/: %s", got)
	}
}

func TestArtifactEmptyTitleFallback(t *testing.T) {
	got := render(t, chat.Artifact(chat.ArtifactProps{Body: g.Text("BODY")}))
	if !strings.Contains(got, "Artifact") {
		t.Errorf("empty title must fall back to 'Artifact': %s", got)
	}
}

func TestArtifactInnerID(t *testing.T) {
	got := render(t, chat.Artifact(chat.ArtifactProps{
		Title: "Test", InnerID: "tool-3-card", Body: g.Text("BODY"),
	}))
	if !strings.Contains(got, `id="tool-3-card"`) {
		t.Errorf("innerID must appear on body div: %s", got)
	}
}
