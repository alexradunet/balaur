package web

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/life"
	_ "github.com/alexradunet/balaur/migrations"
)

// TestDayCardReflectsJournalEntry proves that an entry written via life.JournalWrite
// shows up in the day view — the day card is the surface for reading/writing journal
// entries (the journal card was removed in plan 096; journaling lives in chat +
// the day card).
func TestDayCardReflectsJournalEntry(t *testing.T) {
	const entryText = "The river was quiet at dawn and the fog had not yet lifted."

	app := newWebApp(t)

	// Write an entry directly via life.JournalWrite — the same path the
	// journal_write chat tool calls.
	now := time.Now()
	rec, err := life.JournalWrite(app, entryText, now)
	if err != nil {
		t.Fatalf("JournalWrite: %v", err)
	}
	if rec == nil {
		t.Fatal("JournalWrite returned nil record")
	}

	today := now.Format(dayLayout)

	// The day view (ui.Focus) must show the journal entry text after the write.
	scenario := tests.ApiScenario{
		Name:            "day view reflects journal entry",
		Method:          "GET",
		URL:             "/ui/show/day?date=" + today,
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"day-focus", entryText},
	}
	scenario.Test(t)
}

// TestJournalAndDayRoutesRetired guards against accidental re-registration of
// the standalone /journal and /day pages. The routes are unregistered, so the
// catch-all handler redirects them home (302 → /). The day card artifact
// (/ui/show/day) lives on; the journal card was removed in plan 096.
func TestJournalAndDayRoutesRetired(t *testing.T) {
	for _, url := range []string{"/journal", "/day/2026-01-15"} {
		s := tests.ApiScenario{
			Name:           "GET " + url + " is retired (302)",
			Method:         "GET",
			URL:            url,
			TestAppFactory: newWebApp,
			ExpectedStatus: 302,
		}
		s.Test(t)
	}
}
