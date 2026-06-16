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
// (quest-rail, quest-detail, quest-row, quest-group-title, k-count, …) and the
// Datastar SSE selectors (#quest-rail, #quest-detail) depend on.
// Note: gomponents HTML-escapes attribute values, so ' → &#39; and > → &gt;.
func TestQuestsFocusContract(t *testing.T) {
	first := taskcards.TaskView{ID: "t1", Title: "Morning stretch", Status: "open", RecurLine: "every day"}
	got := renderQuestNode(t, taskcards.QuestsFocus(taskcards.QuestsFocusView{
		Groups: []taskcards.QuestGroupView{
			{Name: "Dailies", Tasks: []taskcards.TaskView{
				{ID: "t1", Title: "Morning stretch", Status: "open"},
			}},
			{Name: "Quests", Tasks: []taskcards.TaskView{
				{ID: "t2", Title: "File the deed", Status: "open", DueLine: "due Mon, Jun 16 at 09:00", Overdue: true},
			}},
		},
		First: &first,
		DoneRecently: []taskcards.TaskView{
			{ID: "t3", Title: "Done thing", Status: "done"},
		},
	}))

	for _, want := range []string{
		`class="quest-log"`,
		`class="quest-rail" id="quest-rail"`,
		`class="quest-group"`,
		`class="quest-group-title"`,
		`class="k-count"`,
		`class="quest-row"`,
		// gomponents escapes ' → &#39; in attribute values
		`@get(&#39;/ui/tasks/t1/card&#39;)`,
		`Morning stretch`,
		`class="quest-row quest-overdue"`,
		`@get(&#39;/ui/tasks/t2/card&#39;)`,
		`File the deed`,
		`class="quest-due"`,
		`due Mon, Jun 16 at 09:00`,
		`class="quest-done"`,
		`Done recently`,
		`@get(&#39;/ui/tasks/t3/card&#39;)`,
		`Done thing`,
		`class="quest-detail" id="quest-detail"`,
		`id="tcard-t1"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("QuestsFocus missing %q in:\n%s", want, got)
		}
	}
}

// TestQuestsFocusEmpty: no groups → shows the k-empty message; no detail card.
func TestQuestsFocusEmpty(t *testing.T) {
	got := renderQuestNode(t, taskcards.QuestsFocus(taskcards.QuestsFocusView{}))
	for _, want := range []string{
		`id="quest-rail"`,
		`class="k-empty"`,
		`No quests yet`,
		`id="quest-detail"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("empty QuestsFocus missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "quest-group") {
		t.Error("empty QuestsFocus must not render quest-group sections")
	}
	if strings.Contains(got, "tcard-") {
		t.Error("empty QuestsFocus must not render a task detail card")
	}
}

// TestQuestRailContract: the rail fragment alone carries the nav, groups, and done.
func TestQuestRailContract(t *testing.T) {
	got := renderQuestNode(t, taskcards.QuestRail(taskcards.QuestsFocusView{
		Groups: []taskcards.QuestGroupView{
			{Name: "Rituals", Tasks: []taskcards.TaskView{
				{ID: "r1", Title: "Weekly review", Status: "open"},
			}},
		},
		DoneRecently: []taskcards.TaskView{
			{ID: "d1", Title: "Sent report", Status: "done"},
		},
	}))
	for _, want := range []string{
		`<nav class="quest-rail" id="quest-rail">`,
		`class="quest-group"`,
		`class="quest-group-title"`,
		`Weekly review`,
		`@get(&#39;/ui/tasks/r1/card&#39;)`,
		`class="quest-done"`,
		`Done recently`,
		`@get(&#39;/ui/tasks/d1/card&#39;)`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("QuestRail missing %q in:\n%s", want, got)
		}
	}
}

// TestQuestRailEmpty: no groups → k-empty; no done section when DoneRecently nil.
func TestQuestRailEmpty(t *testing.T) {
	got := renderQuestNode(t, taskcards.QuestRail(taskcards.QuestsFocusView{}))
	if !strings.Contains(got, `class="k-empty"`) {
		t.Error("empty QuestRail must show k-empty")
	}
	if strings.Contains(got, "quest-group") {
		t.Error("empty QuestRail must not render quest-group")
	}
	if strings.Contains(got, "quest-done") {
		t.Error("empty QuestRail must not render quest-done when DoneRecently is empty")
	}
}
