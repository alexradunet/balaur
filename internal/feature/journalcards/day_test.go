package journalcards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/journalcards"
)

func renderDay(t *testing.T, v journalcards.DayView) string {
	t.Helper()
	var b strings.Builder
	if err := journalcards.DayCard(v).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

func TestDayCardRootID(t *testing.T) {
	out := renderDay(t, journalcards.DayView{
		Date:  "2026-06-14",
		Label: "Sunday, June 14 2026",
	})
	if !strings.Contains(out, `id="ucard-day"`) {
		t.Errorf("missing id=ucard-day in:\n%s", out)
	}
	if !strings.Contains(out, `class="kcard ucard ucard-day"`) {
		t.Errorf("missing root classes in:\n%s", out)
	}
}

func TestDayCardScrollIcon(t *testing.T) {
	out := renderDay(t, journalcards.DayView{
		Date:  "2026-06-14",
		Label: "Sunday, June 14 2026",
	})
	if !strings.Contains(out, `/static/icons/scroll.png`) {
		t.Errorf("missing scroll icon in:\n%s", out)
	}
	if !strings.Contains(out, `day`) {
		t.Errorf("missing 'day' text in:\n%s", out)
	}
}

func TestDayCardLabelTag(t *testing.T) {
	out := renderDay(t, journalcards.DayView{
		Date:  "2026-06-14",
		Label: "Sunday, June 14 2026",
	})
	if !strings.Contains(out, `class="tag"`) {
		t.Errorf("missing tag span in:\n%s", out)
	}
	if !strings.Contains(out, "Sunday, June 14 2026") {
		t.Errorf("missing label text in:\n%s", out)
	}
}

func TestDayCardStatCounts(t *testing.T) {
	out := renderDay(t, journalcards.DayView{
		Date:     "2026-06-14",
		Label:    "Sunday, June 14 2026",
		JournalN: 5,
		DoneN:    3,
		LogN:     2,
	})
	for _, want := range []string{
		"5 journal",
		"3 done",
		"2 logged",
		`class="ucard-stats"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestDayCardRecapKept(t *testing.T) {
	out := renderDay(t, journalcards.DayView{
		Date:     "2026-06-14",
		Label:    "Sunday, June 14 2026",
		HasRecap: true,
		IsToday:  false,
	})
	if !strings.Contains(out, "recap kept") {
		t.Errorf("missing 'recap kept' when HasRecap=true in:\n%s", out)
	}
	if strings.Contains(out, "still being written") {
		t.Errorf("should not contain 'still being written' when HasRecap=true in:\n%s", out)
	}
	if strings.Contains(out, "no recap") {
		t.Errorf("should not contain 'no recap' when HasRecap=true in:\n%s", out)
	}
}

func TestDayCardStillBeingWritten(t *testing.T) {
	out := renderDay(t, journalcards.DayView{
		Date:     "2026-06-14",
		Label:    "Sunday, June 14 2026",
		HasRecap: false,
		IsToday:  true,
	})
	if !strings.Contains(out, "still being written") {
		t.Errorf("missing 'still being written' when IsToday=true && HasRecap=false in:\n%s", out)
	}
	if strings.Contains(out, "recap kept") {
		t.Errorf("should not contain 'recap kept' in:\n%s", out)
	}
	if strings.Contains(out, "no recap") {
		t.Errorf("should not contain 'no recap' in:\n%s", out)
	}
}

func TestDayCardNoRecap(t *testing.T) {
	out := renderDay(t, journalcards.DayView{
		Date:     "2026-06-13",
		Label:    "Saturday, June 13 2026",
		HasRecap: false,
		IsToday:  false,
	})
	if !strings.Contains(out, "no recap") {
		t.Errorf("missing 'no recap' when HasRecap=false && IsToday=false in:\n%s", out)
	}
	if strings.Contains(out, "recap kept") {
		t.Errorf("should not contain 'recap kept' in:\n%s", out)
	}
	if strings.Contains(out, "still being written") {
		t.Errorf("should not contain 'still being written' in:\n%s", out)
	}
}

func TestDayCardFooter(t *testing.T) {
	out := renderDay(t, journalcards.DayView{
		Date:  "2026-06-14",
		Label: "Sunday, June 14 2026",
	})
	if !strings.Contains(out, `href="/focus/day?date=2026-06-14"`) {
		t.Errorf("missing footer link with date in:\n%s", out)
	}
	if !strings.Contains(out, "open the day →") {
		t.Errorf("missing footer text in:\n%s", out)
	}
}
