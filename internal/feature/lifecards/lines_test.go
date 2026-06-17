package lifecards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/lifecards"
)

func renderLines(t *testing.T, v lifecards.LinesView) string {
	t.Helper()
	var b strings.Builder
	if err := lifecards.LinesCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

// TestLinesCard_WithLines verifies the list branch renders all lines.
func TestLinesCard_WithLines(t *testing.T) {
	v := lifecards.LinesView{
		Kind:  "reading",
		Lines: []string{"Jun 14 — Finished The Pragmatic Programmer", "Jun 12 — Chapter 5"},
	}
	out := renderLines(t, v)

	for _, want := range []string{
		`id="ucard-lines"`,
		`class="kcard ucard ucard-lines"`,
		`/static/icons/orb.png`,
		`Recent lines`,
		`reading`,            // kcard-meta kind
		`class="life-lines"`, // ul element
		`Jun 14 — Finished The Pragmatic Programmer`,
		`Jun 12 — Chapter 5`,
		`href="/ui/show/lifelog"`,
		`life →`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("LinesCard(lines) missing %q\nHTML:\n%s", want, out)
		}
	}
	if strings.Contains(out, "No reading entries yet.") {
		t.Errorf("list branch should not show empty state:\n%s", out)
	}
}

// TestLinesCard_Empty verifies the no-lines branch.
func TestLinesCard_Empty(t *testing.T) {
	v := lifecards.LinesView{Kind: "reading"}
	out := renderLines(t, v)

	if !strings.Contains(out, "No reading entries yet.") {
		t.Errorf("empty state missing:\n%s", out)
	}
	if strings.Contains(out, `class="life-lines"`) {
		t.Errorf("list must be absent in empty state:\n%s", out)
	}
}

// TestLinesCard_Error verifies the error branch suppresses content and shows
// the error strip.
func TestLinesCard_Error(t *testing.T) {
	v := lifecards.LinesView{
		Kind:  "sleep",
		Error: "could not load series: db error",
	}
	out := renderLines(t, v)

	for _, want := range []string{
		`class="card-note card-note-error"`,
		`could not load series: db error`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("LinesCard(Error) missing %q\nHTML:\n%s", want, out)
		}
	}
	if strings.Contains(out, "No sleep entries yet.") {
		t.Errorf("error branch should not show empty state:\n%s", out)
	}
	if strings.Contains(out, `class="life-lines"`) {
		t.Errorf("error branch should not show list:\n%s", out)
	}
}

// TestLinesCard_EachLineIsLi checks every line becomes its own <li>.
func TestLinesCard_EachLineIsLi(t *testing.T) {
	v := lifecards.LinesView{
		Kind:  "books",
		Lines: []string{"Line one", "Line two", "Line three"},
	}
	out := renderLines(t, v)

	count := strings.Count(out, "<li>")
	if count != 3 {
		t.Errorf("expected 3 <li> elements, got %d\nHTML:\n%s", count, out)
	}
}
