package web

import (
	"net/url"
	"testing"

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
			`focus-back`,
			// Go's html/template escapes the slashes inside the data-on JS-string
			// context (@get('\/boards\/abc') — functionally identical in JS), so
			// assert the Back link via its plain href, which keeps literal slashes.
			`href="/boards/abc"`,
			`Quest log`,
			`id="dock"`,
		},
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
			`@post('/ui/journal'`, // write form
			`/ui/journal/prompt`,  // guided tab
			`id="journal-candle-body"`,
		},
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
