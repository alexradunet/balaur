package web

import (
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

func TestQuestsRendersViaGomponents(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}
	if _, err := tasks.Create(app, tasks.CreateOpts{Title: "Draft the letter", Due: time.Now().Add(2 * time.Hour), Source: "test"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	taskcards.Register(app)
	defer ui.UnregisterCard("today")
	defer ui.UnregisterCard("quests")

	if _, ok := ui.LookupCard("quests"); !ok {
		t.Fatal("quests not registered via gomponents") // fails before this task wires it
	}

	// Summary mode (default).
	summary := string(h.cardHTML("quests", nil))
	if !strings.Contains(summary, `id="ucard-quests"`) || !strings.Contains(summary, "Draft the letter") {
		t.Fatalf("summary not rendered via gomponents:\n%s", summary)
	}

	// Manage mode renders the full task card.
	manage := string(h.cardHTML("quests", map[string]string{"mode": "manage"}))
	if !strings.Contains(manage, `id="ucard-quests-manage"`) || !strings.Contains(manage, "Snooze") {
		t.Fatalf("manage not rendered via gomponents:\n%s", manage)
	}
}
