package taskcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/taskcards"
)

func TestHabitsCardRendersRoot(t *testing.T) {
	var b strings.Builder
	habits := []taskcards.HabitView{
		{Title: "Morning run", Streak: 7, RecurLine: "every day"},
	}
	if err := taskcards.HabitsCard(habits).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, `id="ucard-habits"`) {
		t.Errorf("missing root id in:\n%s", out)
	}
	if !strings.Contains(out, `class="kcard ucard ucard-habits"`) {
		t.Errorf("missing root classes in:\n%s", out)
	}
}

func TestHabitsCardFlameIcon(t *testing.T) {
	var b strings.Builder
	_ = taskcards.HabitsCard(nil).Render(&b)
	out := b.String()
	if !strings.Contains(out, `/static/icons/flame.png`) {
		t.Errorf("missing flame icon in:\n%s", out)
	}
	if !strings.Contains(out, "Habits") {
		t.Errorf("missing 'Habits' heading in:\n%s", out)
	}
}

func TestHabitsCardRowTitle(t *testing.T) {
	var b strings.Builder
	habits := []taskcards.HabitView{
		{Title: "Morning run", Streak: 5, RecurLine: "every day"},
	}
	if err := taskcards.HabitsCard(habits).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "Morning run") {
		t.Errorf("missing habit title in:\n%s", out)
	}
}

func TestHabitsCardRowRecurLine(t *testing.T) {
	var b strings.Builder
	habits := []taskcards.HabitView{
		{Title: "Read a book", Streak: 3, RecurLine: "every day"},
	}
	if err := taskcards.HabitsCard(habits).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "every day") {
		t.Errorf("missing recur line in:\n%s", out)
	}
	if !strings.Contains(out, `class="kcard-meta"`) {
		t.Errorf("missing kcard-meta span in:\n%s", out)
	}
}

func TestHabitsCardRowStreak(t *testing.T) {
	var b strings.Builder
	habits := []taskcards.HabitView{
		{Title: "Meditate", Streak: 12, RecurLine: "every day"},
	}
	if err := taskcards.HabitsCard(habits).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "12d") {
		t.Errorf("missing streak '12d' in:\n%s", out)
	}
	if !strings.Contains(out, `class="habit-streak"`) {
		t.Errorf("missing habit-streak span in:\n%s", out)
	}
}

func TestHabitsCardNoRecurLine(t *testing.T) {
	var b strings.Builder
	habits := []taskcards.HabitView{
		{Title: "Push-ups", Streak: 2, RecurLine: ""},
	}
	if err := taskcards.HabitsCard(habits).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	// RecurLine is empty, so the kcard-meta span should not appear for this habit.
	// The title and streak should still appear.
	if !strings.Contains(out, "Push-ups") {
		t.Errorf("missing title in:\n%s", out)
	}
	if !strings.Contains(out, "2d") {
		t.Errorf("missing streak in:\n%s", out)
	}
}

func TestHabitsCardEmptyState(t *testing.T) {
	var b strings.Builder
	if err := taskcards.HabitsCard(nil).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "No habits yet") {
		t.Errorf("expected empty state in:\n%s", out)
	}
	if strings.Contains(out, "ucard-list") {
		t.Errorf("list should not appear in empty state:\n%s", out)
	}
}

func TestHabitsCardFooter(t *testing.T) {
	var b strings.Builder
	_ = taskcards.HabitsCard(nil).Render(&b)
	out := b.String()
	if !strings.Contains(out, `/ui/show/lifelog`) {
		t.Errorf("missing footer link in:\n%s", out)
	}
	if !strings.Contains(out, "life →") {
		t.Errorf("missing footer text in:\n%s", out)
	}
}
