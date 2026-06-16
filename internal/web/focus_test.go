package web

import (
	"net/url"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

// TestFocusFullLoad: a direct browser load renders the whole shell (topbar +
// focus chrome + dock), with the card label and a Back link.
func TestFocusFullLoad(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/quests renders the shell",
		Method:         "GET",
		URL:            "/focus/quests?from=abc",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`class="focus"`,
			`Quest log`,
			`id="dock"`,
			// New gomponents shell: the top-nav topbar with the active domain
			// (Quests) riding gold — replaces the legacy Boards/Settings topbar
			// and the "← Back to board" control.
			`class="topbar"`,
			`href="/focus/quests" aria-current="page"`,
		},
		// The obsolete board-relative chrome is gone.
		NotExpectedContent: []string{`focus-back`, `← Back`, `>Boards<`, `Engine room`},
	}
	s.Test(t)
}

// TestFocusDatastarPatch: a Datastar @get patches #main only (no full doc, no
// dock), and reflects the canonical URL without the transient from.
func TestFocusDatastarPatch(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "Datastar @get /focus/quests patches #main",
		Method:         "GET",
		URL:            "/focus/quests?status=open&from=abc",
		Headers:        map[string]string{"Accept": "text/event-stream"},
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			`selector #main`,
			`class="focus"`,
		},
		NotExpectedContent: []string{"<!DOCTYPE", `id="dock"`},
	}
	s.Test(t)
}

func TestFocusBackHrefRejectsUnsafe(t *testing.T) {
	for _, from := range []string{
		`x') ;alert(1);('`,
		`a/b`,
		`a b`,
		`"`,
		`../evil`,
		``,
	} {
		if got := focusBackHref(from); got != "/boards" {
			t.Errorf("focusBackHref(%q) = %q, want /boards (unsafe must fall back)", from, got)
		}
	}
	if got := focusBackHref("abc123"); got != "/boards/abc123" {
		t.Errorf("focusBackHref(%q) = %q, want /boards/abc123", "abc123", got)
	}
}

// TestFocusUnknownType: an unregistered card type 404s.
func TestFocusUnknownType(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/nope is 404",
		Method:         "GET",
		URL:            "/focus/nope",
		TestAppFactory: newWebApp,
		ExpectedStatus: 404,
		// PocketBase's JSON error serializer sentence-cases the message:
		// "no such card type" is emitted as "No such card type" in the body.
		ExpectedContent: []string{"No such card type"},
	}
	s.Test(t)
}

func TestFocusCanonicalQuery(t *testing.T) {
	// from is transient: it must never appear in the reflected canonical URL.
	q := url.Values{"status": {"open"}, "from": {"abc"}}
	if got := focusCanonicalQuery(q); got != "status=open" {
		t.Errorf("focusCanonicalQuery(%v) = %q; want %q", q, got, "status=open")
	}
	// Only-from query → empty string (no trailing "?").
	if got := focusCanonicalQuery(url.Values{"from": {"abc"}}); got != "" {
		t.Errorf("only-from query: got %q, want empty", got)
	}
}

// TestFocusQuestsShowsRail: /focus/quests renders the rhythm rail (not the flat
// manage list), so expanding the quests card gives the full quest-log surface.
func TestFocusQuestsShowsRail(t *testing.T) {
	app := newWebApp(t)
	// Seed one open task so the rail has a group.
	col, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		t.Fatalf("tasks collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("title", "Walk the dog")
	rec.Set("status", "open")
	if err := app.Save(rec); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	s := tests.ApiScenario{
		Name:           "GET /focus/quests shows the quest rail",
		Method:         "GET",
		URL:            "/focus/quests",
		TestAppFactory: func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`id="quest-rail"`,
			`id="quest-detail"`,
			"Walk the dog",
		},
	}
	s.Test(t)
}

// TestFocusJournalShowsCandle: /focus/journal renders the candle (write form +
// guided tab), so expanding the journal card gives the full writing surface.
func TestFocusJournalShowsCandle(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/journal shows the candle",
		Method:         "GET",
		URL:            "/focus/journal",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`@post(&#39;/ui/journal&#39;`, // write form (gomponents HTML-escapes ' → &#39;)
			`/ui/journal/prompt`,          // guided tab
			`id="journal-candle-body"`,
		},
	}
	s.Test(t)
}

// TestFocusDayShowsSections: /focus/day renders the day-of-life sections.
func TestFocusDayShowsSections(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/day shows the day view",
		Method:         "GET",
		URL:            "/focus/day",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`id="day-journal"`,
			"What got done",
			"The day's log",
			`@post('/ui/day/`, // the day journal write form
		},
	}
	s.Test(t)
}

// TestUiCardDayTile: the day tile renders the day-of-life summary.
func TestUiCardDayTile(t *testing.T) {
	s := tests.ApiScenario{
		Name:            "GET /ui/cards/day renders the tile",
		Method:          "GET",
		URL:             "/ui/cards/day",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  200,
		ExpectedContent: []string{"ucard-day", "journal", `/focus/day?date=`},
	}
	s.Test(t)
}

// TestUiCardLifelogTile: the lifelog tile renders, linking to its focus.
func TestUiCardLifelogTile(t *testing.T) {
	s := tests.ApiScenario{
		Name:            "GET /ui/cards/lifelog renders the tile",
		Method:          "GET",
		URL:             "/ui/cards/lifelog",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  200,
		ExpectedContent: []string{"ucard-lifelog", `/focus/lifelog`},
	}
	s.Test(t)
}

// TestFocusLifelogShowsOverview: /focus/lifelog renders the life overview body.
// A fresh app has no life kinds, so seed one numeric kind to surface the tracked
// grid (life-grid) rather than the empty state.
func TestFocusLifelogShowsOverview(t *testing.T) {
	app := newWebApp(t)
	seedLifeEntry(t, app, "weight", 72.5, "", time.Now().Add(-time.Hour))
	s := tests.ApiScenario{
		Name:            "GET /focus/lifelog shows the overview",
		Method:          "GET",
		URL:             "/focus/lifelog",
		TestAppFactory:  func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"life-grid", "weight"},
	}
	s.Test(t)
}

// TestFocusMemoryShowsManager: /focus/memory renders the full knowledge manager
// (active section + search), not the compact manage tile.
func TestFocusMemoryShowsManager(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/memory shows the manager",
		Method:         "GET",
		URL:            "/focus/memory",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`id="k-active-grid"`,
			"Active",
			`/ui/knowledge/memories/grid`, // the live-search control
		},
	}
	s.Test(t)
}

// TestFocusMemoryShowsProposed: a proposed memory surfaces in the focus with its
// approve action — the consent queue ("Awaiting your word") renders in the card
// focus, not just the active grid. Guards the new render path that folds
// memory's old standalone proposed page into the expanded card.
func TestFocusMemoryShowsProposed(t *testing.T) {
	app := newWebApp(t)
	seedProposedMemory(t, app, "Remembers the dog name")

	s := tests.ApiScenario{
		Name:           "GET /focus/memory shows the proposed queue",
		Method:         "GET",
		URL:            "/focus/memory",
		TestAppFactory: func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"Awaiting your word",       // the proposed-section heading
			`k-heading-proposed`,       // the proposed section class
			"Remembers the dog name",   // the seeded record's title
			`name="to" value="active"`, // the approve form's transition payload
			`<button class="btn btn-primary btn-sm" type="submit">Approve</button>`,
		},
	}
	s.Test(t)
}

// TestFocusSkillsShowsManager: /focus/skills renders the skills manager.
func TestFocusSkillsShowsManager(t *testing.T) {
	s := tests.ApiScenario{
		Name:            "GET /focus/skills shows the manager",
		Method:          "GET",
		URL:             "/focus/skills",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  200,
		ExpectedContent: []string{`id="k-active-grid"`, `/ui/knowledge/skills/grid`},
	}
	s.Test(t)
}

func TestFocusMissingRequiredParam(t *testing.T) {
	s := tests.ApiScenario{
		Name:            "GET /focus/measure without kind → 400",
		Method:          "GET",
		URL:             "/focus/measure",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  400,
		ExpectedContent: []string{"kind"},
	}
	s.Test(t)
}

// TestUiCardSettingsTile: the settings tile renders with section links.
func TestUiCardSettingsTile(t *testing.T) {
	s := tests.ApiScenario{
		Name:            "GET /ui/cards/settings renders the tile",
		Method:          "GET",
		URL:             "/ui/cards/settings",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  200,
		ExpectedContent: []string{"ucard-settings", `/focus/settings?section=models`},
	}
	s.Test(t)
}

// TestFocusSettingsProfile: /focus/settings renders the profile section by default.
func TestFocusSettingsProfile(t *testing.T) {
	s := tests.ApiScenario{
		Name:               "GET /focus/settings → profile section",
		Method:             "GET",
		URL:                "/focus/settings",
		TestAppFactory:     newWebApp,
		ExpectedStatus:     200,
		ExpectedContent:    []string{`id="identity-card"`, "settings-nav"},
		NotExpectedContent: []string{">Skills<"},
	}
	s.Test(t)
}

// TestFocusSettingsModels: ?section=models renders the models panel.
func TestFocusSettingsModels(t *testing.T) {
	s := tests.ApiScenario{
		Name:            "GET /focus/settings?section=models → models panel",
		Method:          "GET",
		URL:             "/focus/settings?section=models",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  200,
		ExpectedContent: []string{"settings-nav", `id="models-panel"`},
	}
	s.Test(t)
}
