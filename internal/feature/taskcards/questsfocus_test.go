package taskcards_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
)

func renderQuestNode(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

// TestQuestsFocusContract guards the class/markup contract the served CSS
// (quest-stack, k-section, k-heading, k-count, tasks-stack) and task card
// selectors (#tcard-{id}) depend on — flat stack, no rail (plan 093).
// Note: gomponents HTML-escapes attribute values, so ' → &#39; and > → &gt;.
func TestQuestsFocusContract(t *testing.T) {
	got := renderQuestNode(t, taskcards.QuestsFocus(taskcards.QuestsFocusView{
		Groups: []taskcards.QuestGroupView{
			{Name: "Dailies", Tasks: []taskcards.TaskView{
				{ID: "t1", Title: "Morning stretch", Status: "open"},
			}},
			{Name: "Quests", Tasks: []taskcards.TaskView{
				{ID: "t2", Title: "File the deed", Status: "open", DueLine: "due Mon, Jun 16 at 09:00", Overdue: true},
			}},
		},
		DoneRecently: []taskcards.TaskView{
			{ID: "t3", Title: "Done thing", Status: "done"},
		},
	}))

	for _, want := range []string{
		`class="quest-stack"`,
		`class="k-section"`,
		`class="k-heading"`,
		`class="k-count"`,
		`class="tasks-stack"`,
		`Morning stretch`,
		`id="tcard-t1"`,
		`File the deed`,
		`id="tcard-t2"`,
		`Done recently`,
		`Done thing`,
		`id="tcard-t3"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("QuestsFocus missing %q in:\n%s", want, got)
		}
	}

	// Rail and detail pane must be absent
	for _, absent := range []string{
		`quest-rail`,
		`quest-detail`,
		`quest-log`,
		`quest-row`,
	} {
		if strings.Contains(got, absent) {
			t.Errorf("QuestsFocus must not render %q (flat stack, plan 093)", absent)
		}
	}
}

// TestQuestsFocusEmpty: no groups → shows the empty state message.
func TestQuestsFocusEmpty(t *testing.T) {
	got := renderQuestNode(t, taskcards.QuestsFocus(taskcards.QuestsFocusView{}))
	for _, want := range []string{
		`class="quest-stack"`,
		`No quests yet`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("empty QuestsFocus missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "k-section") {
		t.Error("empty QuestsFocus must not render k-section groups")
	}
	if strings.Contains(got, "tcard-") {
		t.Error("empty QuestsFocus must not render task cards")
	}
}
