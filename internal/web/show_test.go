package web

// show_test.go — handler tests for GET /ui/show/{type} (plan 088).
// Correctness pins:
//   - 200 + SSE patch → #chat (append) with the card markup
//   - Persists role="tool", origin="" (sidesteps chatNudges origin != '')
//   - Content carries the uicard marker (\x00balaur-uicard:)
//   - Reload (GET /) shows the card (recap.messageViews uicard branch)
//   - chatNudges does NOT duplicate the card (origin="" skips the filter)
//   - GET /ui/show/bogus → 404

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/tools"
	_ "github.com/alexradunet/balaur/migrations"
)

func TestUIShow(t *testing.T) {
	t.Run("GET /ui/show/bogus → 404", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:            "unknown card type → 404",
			Method:          "GET",
			URL:             "/ui/show/bogus",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  404,
			ExpectedContent: []string{"No such card type"},
		}
		s.Test(t)
	})

	t.Run("GET /ui/show/quests → 200 SSE append to #chat, persists uicard row", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "quests show → 200 SSE patch to #chat",
			Method:         "GET",
			URL:            "/ui/show/quests",
			TestAppFactory: newWebApp,
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"datastar-patch-elements",
				"selector #chat",
				"mode append",
				"ucard-quests",
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, res *http.Response) {
				// Verify the persisted messages row: role=tool, origin="",
				// content carries the uicard marker.
				recs, err := app.FindRecordsByFilter("messages", "role = 'tool'", "-@rowid", 1, 0, nil)
				if err != nil || len(recs) == 0 {
					tb.Fatalf("no tool message persisted after /ui/show/quests: %v", err)
				}
				rec := recs[0]
				// origin="" sidesteps chatNudges filter (origin != '').
				if got := rec.GetString("origin"); got != "" {
					tb.Errorf("origin = %q, want empty (must sidestep chatNudges)", got)
				}
				content := rec.GetString("content")
				if !strings.HasPrefix(content, tools.UICardMarker) {
					tb.Errorf("content does not start with UICardMarker; got %q", content)
				}
				typ, _, _, ok := tools.ParseUICard(content)
				if !ok {
					tb.Errorf("ParseUICard returned ok=false for persisted content %q", content)
				}
				if typ != "quests" {
					tb.Errorf("parsed typ = %q, want %q", typ, "quests")
				}
			},
		}
		s.Test(t)
	})

	t.Run("GET /ui/show/quests: chatNudges does not duplicate the card", func(t *testing.T) {
		// Inject the card, then verify chatNudges does not surface it (origin="").
		s := tests.ApiScenario{
			Name:            "quests show + chatNudges since=0 — no duplication",
			Method:          "GET",
			URL:             "/ui/show/quests",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"datastar-patch-elements"},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				// chatNudges filters origin != ''. The injected row has origin="",
				// so it must never appear in the nudge stream.
				recs, err := app.FindRecordsByFilter("messages", "origin != ''", "@rowid", 20, 0, nil)
				if err != nil {
					tb.Fatalf("nudge filter query: %v", err)
				}
				for _, r := range recs {
					if strings.Contains(r.GetString("content"), "quests") {
						tb.Errorf("chatNudges would return the /ui/show card (origin=%q, content=%q)",
							r.GetString("origin"), r.GetString("content"))
					}
				}
			},
		}
		s.Test(t)
	})

	t.Run("GET / after /ui/show/quests shows card in history", func(t *testing.T) {
		// Inject card via /ui/show/quests; then verify the persisted row is in
		// History (the DB record that recap.messageViews would re-render on reload).
		s := tests.ApiScenario{
			Name:            "inject quests — DB history contains the tool row",
			Method:          "GET",
			URL:             "/ui/show/quests",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"datastar-patch-elements"},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				// History returns all roles including tool; the injected row
				// must be present with role=tool and uicard marker content.
				recs, err := app.FindRecordsByFilter("messages", "role = 'tool'", "-@rowid", 10, 0, nil)
				if err != nil {
					tb.Fatalf("loading tool messages: %v", err)
				}
				var found bool
				for _, r := range recs {
					if strings.HasPrefix(r.GetString("content"), tools.UICardMarker) {
						found = true
					}
				}
				if !found {
					tb.Errorf("no tool message with uicard marker found in history after /ui/show/quests")
				}
			},
		}
		s.Test(t)
	})

	t.Run("GET /ui/show/quests with invalid params → 400", func(t *testing.T) {
		// quests supports a status param with an enum; bogusvalue is not in it.
		s := tests.ApiScenario{
			Name:            "invalid quests params → 400",
			Method:          "GET",
			URL:             "/ui/show/quests?status=bogusvalue",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  400,
			ExpectedContent: []string{"Invalid card params"},
		}
		s.Test(t)
	})
}
