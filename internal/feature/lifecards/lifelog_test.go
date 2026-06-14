package lifecards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/lifecards"
)

// TestLifelogCardRootID verifies the root element has id="ucard-lifelog".
func TestLifelogCardRootID(t *testing.T) {
	var b strings.Builder
	v := lifecards.LifelogView{}
	if err := lifecards.LifelogCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, `id="ucard-lifelog"`) {
		t.Errorf("missing root id in:\n%s", out)
	}
	if !strings.Contains(out, `class="kcard ucard ucard-lifelog"`) {
		t.Errorf("missing root classes in:\n%s", out)
	}
}

// TestLifelogCardHeader verifies the orb icon and "Life" heading.
func TestLifelogCardHeader(t *testing.T) {
	var b strings.Builder
	_ = lifecards.LifelogCard(lifecards.LifelogView{}).Render(&b)
	out := b.String()
	if !strings.Contains(out, `/static/icons/orb.png`) {
		t.Errorf("missing orb icon in:\n%s", out)
	}
	if !strings.Contains(out, "Life") {
		t.Errorf("missing 'Life' heading in:\n%s", out)
	}
}

// TestLifelogCardTrackedKindRow verifies a tracked-kind row renders kind + count.
func TestLifelogCardTrackedKindRow(t *testing.T) {
	var b strings.Builder
	v := lifecards.LifelogView{
		Kinds: []lifecards.LifeKindView{
			{Kind: "weight", Count: 42},
		},
	}
	if err := lifecards.LifelogCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, `class="ucard-stats"`) {
		t.Errorf("missing ucard-stats ul in:\n%s", out)
	}
	if !strings.Contains(out, "weight") {
		t.Errorf("missing kind name in:\n%s", out)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("missing kind count in:\n%s", out)
	}
	if !strings.Contains(out, `class="kcard-meta"`) {
		t.Errorf("missing kcard-meta span for count in:\n%s", out)
	}
}

// TestLifelogCardHabitRow verifies a habit row renders title and streak.
func TestLifelogCardHabitRow(t *testing.T) {
	var b strings.Builder
	v := lifecards.LifelogView{
		Habits: []lifecards.LifeHabitView{
			{Title: "Morning run", Streak: 7, RecurLine: "every day"},
		},
	}
	if err := lifecards.LifelogCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, `class="habit-strip"`) {
		t.Errorf("missing habit-strip div in:\n%s", out)
	}
	if !strings.Contains(out, "Morning run") {
		t.Errorf("missing habit title in:\n%s", out)
	}
	if !strings.Contains(out, "· 7") {
		t.Errorf("missing streak separator in:\n%s", out)
	}
	if !strings.Contains(out, `class="tag habit-tag"`) {
		t.Errorf("missing tag habit-tag class in:\n%s", out)
	}
	if !strings.Contains(out, `title="every day"`) {
		t.Errorf("missing title attr (recur line) in:\n%s", out)
	}
}

// TestLifelogCardHabitZeroStreak verifies streak is omitted when zero.
func TestLifelogCardHabitZeroStreak(t *testing.T) {
	var b strings.Builder
	v := lifecards.LifelogView{
		Habits: []lifecards.LifeHabitView{
			{Title: "Meditate", Streak: 0, RecurLine: "every day"},
		},
	}
	if err := lifecards.LifelogCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "Meditate") {
		t.Errorf("missing habit title:\n%s", out)
	}
	if strings.Contains(out, "· 0") {
		t.Errorf("streak=0 should not render '· 0' in:\n%s", out)
	}
}

// TestLifelogCardEmptyKinds verifies the empty-state message when no kinds tracked.
func TestLifelogCardEmptyKinds(t *testing.T) {
	var b strings.Builder
	v := lifecards.LifelogView{}
	if err := lifecards.LifelogCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "Nothing tracked yet") {
		t.Errorf("expected empty-state message in:\n%s", out)
	}
	if strings.Contains(out, `class="ucard-stats"`) {
		t.Errorf("ucard-stats should not appear in empty state:\n%s", out)
	}
}

// TestLifelogCardNoHabitsStrip verifies the habit-strip is absent when no habits.
func TestLifelogCardNoHabitsStrip(t *testing.T) {
	var b strings.Builder
	v := lifecards.LifelogView{}
	if err := lifecards.LifelogCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if strings.Contains(out, `class="habit-strip"`) {
		t.Errorf("habit-strip should not appear when no habits:\n%s", out)
	}
}

// TestLifelogCardFooter verifies the footer link.
func TestLifelogCardFooter(t *testing.T) {
	var b strings.Builder
	_ = lifecards.LifelogCard(lifecards.LifelogView{}).Render(&b)
	out := b.String()
	if !strings.Contains(out, `/focus/lifelog`) {
		t.Errorf("missing footer link in:\n%s", out)
	}
	if !strings.Contains(out, "open life →") {
		t.Errorf("missing footer text in:\n%s", out)
	}
}
