package lifecards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/lifecards"
)

func renderMeasure(t *testing.T, v lifecards.MeasureView) string {
	t.Helper()
	var b strings.Builder
	if err := lifecards.MeasureCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

// TestMeasureCard_HasData verifies the data-present branch renders stat,
// sparkline (polyline + circle), change, and footer link.
func TestMeasureCard_HasData(t *testing.T) {
	v := lifecards.MeasureView{
		Kind:       "weight",
		HasData:    true,
		LastVal:    "82.5",
		Unit:       "kg",
		LastAt:     "Jun 14",
		Points:     "4.0,44.0 122.0,24.0 240.0,4.0",
		SparkLastX: "240.0",
		SparkLastY: "4.0",
		Change:     "+1.5 over 90d",
	}
	out := renderMeasure(t, v)

	for _, want := range []string{
		`id="ucard-measure"`,
		`class="kcard ucard ucard-measure"`,
		`/static/icons/orb.png`,
		`Measure`,
		`weight`,            // kcard-meta kind
		`class="life-stat"`, // stat paragraph
		`82.5`,              // LastVal
		`class="life-unit"`,
		`kg`, // Unit
		`class="life-lastat"`,
		`· Jun 14`,      // LastAt
		`class="spark"`, // SVG element
		`viewBox="0 0 240 48"`,
		`width="240"`,
		`height="48"`,
		`aria-label="weight trend"`,
		`4.0,44.0 122.0,24.0 240.0,4.0`, // polyline points
		`fill="none"`,
		`cx="240.0"`, // circle last x
		`cy="4.0"`,   // circle last y
		`r="3"`,
		`class="life-change"`,
		`+1.5 over 90d`,
		`href="/focus/lifelog"`,
		`life →`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("MeasureCard(HasData) missing %q\nHTML:\n%s", want, out)
		}
	}

	// empty-state text must not appear
	if strings.Contains(out, "No weight entries yet.") {
		t.Errorf("HasData branch should not show empty state:\n%s", out)
	}
}

// TestMeasureCard_Empty verifies the no-data branch.
func TestMeasureCard_Empty(t *testing.T) {
	v := lifecards.MeasureView{Kind: "weight"}
	out := renderMeasure(t, v)

	if !strings.Contains(out, "No weight entries yet.") {
		t.Errorf("empty state missing:\n%s", out)
	}
	// stat and sparkline must not appear
	if strings.Contains(out, `class="life-stat"`) {
		t.Errorf("stat must be absent in empty state:\n%s", out)
	}
	if strings.Contains(out, `class="spark"`) {
		t.Errorf("sparkline must be absent in empty state:\n%s", out)
	}
}

// TestMeasureCard_Error verifies the error branch suppresses data and shows
// the error strip.
func TestMeasureCard_Error(t *testing.T) {
	v := lifecards.MeasureView{
		Kind:  "steps",
		Error: "could not load series: db gone",
	}
	out := renderMeasure(t, v)

	for _, want := range []string{
		`class="card-note card-note-error"`,
		`could not load series: db gone`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("MeasureCard(Error) missing %q\nHTML:\n%s", want, out)
		}
	}
	if strings.Contains(out, "No steps entries yet.") {
		t.Errorf("error branch should not show empty state:\n%s", out)
	}
	if strings.Contains(out, `class="life-stat"`) {
		t.Errorf("error branch should not show stat:\n%s", out)
	}
}

// TestMeasureCard_HasDataNoSparkline verifies that when Points is empty (only
// one data point), the SVG is omitted but the stat is still rendered.
func TestMeasureCard_HasDataNoSparkline(t *testing.T) {
	v := lifecards.MeasureView{
		Kind:    "weight",
		HasData: true,
		LastVal: "80",
		Unit:    "kg",
		LastAt:  "Jun 1",
		// Points / SparkLastX / SparkLastY all empty — only 1 point
	}
	out := renderMeasure(t, v)

	if !strings.Contains(out, `class="life-stat"`) {
		t.Errorf("stat must be present even without sparkline:\n%s", out)
	}
	if strings.Contains(out, `class="spark"`) {
		t.Errorf("sparkline SVG must be absent when Points is empty:\n%s", out)
	}
	if strings.Contains(out, `class="life-change"`) {
		t.Errorf("change paragraph must be absent when Change is empty:\n%s", out)
	}
}

// TestMeasureCard_UnitOptional checks that the unit span is absent when Unit is
// empty (matches template's {{with .Unit}}).
func TestMeasureCard_UnitOptional(t *testing.T) {
	v := lifecards.MeasureView{
		Kind:    "mood",
		HasData: true,
		LastVal: "7",
		LastAt:  "Jun 14",
	}
	out := renderMeasure(t, v)

	if strings.Contains(out, `class="life-unit"`) {
		t.Errorf("unit span should be absent when Unit is empty:\n%s", out)
	}
}
