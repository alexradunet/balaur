package taskcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
)

func TestQuestsCardSummary(t *testing.T) {
	var b strings.Builder
	v := taskcards.QuestsView{
		ParamLine: "status: open · limit: 10",
		Rows: []taskcards.TaskView{
			{ID: "q1", Title: "Draft the letter", Status: "open", DueLine: "due tomorrow"},
		},
	}
	if err := taskcards.QuestsCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`id="ucard-quests"`, "Quest log", "status: open · limit: 10",
		`id="urow-quests-q1"`, "Draft the letter", "due tomorrow",
		`value="done"`, `value="quests"`,
		"all quests →",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q in:\n%s", want, out)
		}
	}
}

func TestQuestsCardSummaryEmpty(t *testing.T) {
	var b strings.Builder
	_ = taskcards.QuestsCard(taskcards.QuestsView{}).Render(&b)
	if !strings.Contains(b.String(), "No quests here yet.") {
		t.Errorf("expected empty state:\n%s", b.String())
	}
}

func TestQuestsManageCardRendersTaskCards(t *testing.T) {
	var b strings.Builder
	v := taskcards.QuestsView{Rows: []taskcards.TaskView{{ID: "m1", Title: "Manage me", Status: "open"}}}
	if err := taskcards.QuestsManageCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`id="ucard-quests-manage"`, "Quest log",
		`id="tcard-m1"`, "Manage me", `value="snooze"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("manage missing %q in:\n%s", want, out)
		}
	}
}

func TestQuestsManageCardEmpty(t *testing.T) {
	var b strings.Builder
	_ = taskcards.QuestsManageCard(taskcards.QuestsView{}).Render(&b)
	if !strings.Contains(b.String(), "No open quests") {
		t.Errorf("expected manage empty state:\n%s", b.String())
	}
}
