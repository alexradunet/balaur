package lifecards_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/feature/lifecards"
)

func renderNode(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

// TestLifelogFocusContract guards the class/markup contract the served CSS
// (life-grid, life-card, spark, habit-tag, k-section, …) depends on — a port of
// the legacy life_body template must keep these byte-for-byte.
func TestLifelogFocusContract(t *testing.T) {
	got := renderNode(t, lifecards.LifelogFocus(lifecards.LifelogFocusView{
		Habits: []lifecards.LifeHabitView{{Title: "Stretch", Streak: 5, RecurLine: "repeats daily"}},
		Kinds: []lifecards.LifeKindFocusView{
			{Kind: "weight", Unit: "kg", Count: 12, Numeric: true, LastVal: "82.5", LastAt: "Jun 11",
				Change: "-0.8 over 90d", Points: "4.0,40.0 236.0,8.0", SparkLastX: "236.0", SparkLastY: "8.0"},
			{Kind: "gratitude", Count: 2, Recent: []string{"Jun 10 — quiet morning"}},
		},
	}))
	for _, want := range []string{
		`<section class="k-section">`,
		`<h2 class="k-heading">Habits</h2>`,
		`<span class="tag habit-tag" title="repeats daily">Stretch · streak 5</span>`,
		`<div class="stitch">`,
		`<div class="k-grid life-grid">`,
		`<article class="kcard life-card">`,
		`<span class="kcard-kind">▪ weight</span>`,
		`<span class="kcard-meta">12 entries</span>`,
		`<p class="life-stat">82.5 <span class="life-unit">kg</span> <span class="life-lastat">· Jun 11</span></p>`,
		`<svg class="spark" viewBox="0 0 240 48"`,
		`<polyline points="4.0,40.0 236.0,8.0" fill="none">`,
		`<circle cx="236.0" cy="8.0" r="3">`,
		`<p class="life-change">-0.8 over 90d</p>`,
		`<ul class="life-lines"><li>Jun 10 — quiet morning</li></ul>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lifelog focus missing %q in:\n%s", want, got)
		}
	}
}

// TestLifelogFocusEmpty: no tracked kinds → the invitation empty state.
func TestLifelogFocusEmpty(t *testing.T) {
	got := renderNode(t, lifecards.LifelogFocus(lifecards.LifelogFocusView{}))
	if !strings.Contains(got, `<p class="k-empty">Nothing tracked yet.`) {
		t.Errorf("empty lifelog focus should show the k-empty invitation; got:\n%s", got)
	}
	if strings.Contains(got, "life-grid") {
		t.Error("empty lifelog focus should not render a grid")
	}
}
