package web

// show_test.go — handler tests for GET /ui/show/{type} (plan 101).
// Correctness pins:
//   - 200 + SSE morph → #panel-inner (panel swap) with the card markup
//   - Does NOT append a chip or persist a tool row (owner directive: no pollution)
//   - panel_active owner_setting is written with the canonical re-summon URL
//   - GET /ui/show/close clears panel_active and morphs the empty panel
//   - GET /ui/show/bogus → 404
//   - GET /ui/show/quests?status=bogusvalue → 400

import (
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/store"
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

	t.Run("GET /ui/show/quests → 200 SSE: morphs panel, no chip, no persisted row", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "quests show → 200 SSE: panel morph, no chip",
			Method:         "GET",
			URL:            "/ui/show/quests",
			TestAppFactory: newWebApp,
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"datastar-patch-elements",
				`id="panel-inner"`, // panel morph (single-active)
				"quest-stack",      // card body is in the panel
			},
			// Must NOT append a chip or persist a chat row.
			NotExpectedContent: []string{
				"art-chip",
				`selector "#chat"`,
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				// panel_active must be written with the canonical re-summon URL.
				active := store.GetOwnerSetting(app, panelActiveKey, "")
				if active != "/ui/show/quests" {
					tb.Errorf("panel_active = %q, want %q", active, "/ui/show/quests")
				}
			},
		}
		s.Test(t)
	})

	t.Run("GET /ui/show/quests → also re-patches #navrail with the active highlight", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "quests show → nav rail re-patched, Quests active",
			Method:         "GET",
			URL:            "/ui/show/quests",
			TestAppFactory: newWebApp,
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`id="navrail"`,        // the rail is re-patched in the same SSE response
				"navrail-btn-active",  // a primary icon is highlighted
				`aria-current="page"`, // ...the active one carries aria-current
			},
		}
		s.Test(t)
	})

	t.Run("GET /ui/show/quests: no tool row persisted (headline regression guard)", func(t *testing.T) {
		// This is the owner directive test: opening a card must NOT persist a
		// role=tool row. Count tool rows before and after; expect unchanged.
		s := tests.ApiScenario{
			Name:            "quests show — no tool row persisted",
			Method:          "GET",
			URL:             "/ui/show/quests",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"datastar-patch-elements"},
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				// Capture the tool-row count before the request.
				master, err := conversation.Master(app)
				if err != nil {
					tb.Fatalf("conversation.Master: %v", err)
				}
				hist, err := conversation.History(app, master.Id, 200)
				if err != nil {
					tb.Fatalf("conversation.History: %v", err)
				}
				var count int
				for _, r := range hist {
					if r.GetString("role") == "tool" {
						count++
					}
				}
				// Store the count for AfterTestFunc by setting an owner setting
				// we can read back — this avoids shared mutable state.
				_ = store.SetOwnerSetting(app, "test_tool_row_count_before", string(rune('0'+count)))
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				before := store.GetOwnerSetting(app, "test_tool_row_count_before", "?")
				master, err := conversation.Master(app)
				if err != nil {
					tb.Fatalf("conversation.Master: %v", err)
				}
				hist, err := conversation.History(app, master.Id, 200)
				if err != nil {
					tb.Fatalf("conversation.History: %v", err)
				}
				var after int
				for _, r := range hist {
					if r.GetString("role") == "tool" {
						after++
					}
				}
				afterStr := string(rune('0' + after))
				if afterStr != before {
					tb.Errorf("tool row count changed: before=%s after=%s — /ui/show/quests persisted a row (owner-pollution regression)", before, afterStr)
				}
			},
		}
		s.Test(t)
	})

	t.Run("GET /ui/show/close → 200: morphs empty panel, panel_active cleared", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "close → empty panel + panel_active cleared",
			Method:         "GET",
			URL:            "/ui/show/close",
			TestAppFactory: newWebApp,
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"datastar-patch-elements",
				`id="panel-inner"`,
				"panel-empty",
				`id="navrail"`, // the rail is re-patched too
			},
			// Nothing is open after close → no primary icon highlighted.
			NotExpectedContent: []string{"navrail-btn-active"},
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				// Pre-seed panel_active so we can verify it is cleared.
				if err := store.SetOwnerSetting(app, panelActiveKey, "/ui/show/quests"); err != nil {
					tb.Fatalf("SetOwnerSetting: %v", err)
				}
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				got := store.GetOwnerSetting(app, panelActiveKey, "")
				if got != "" {
					tb.Errorf("panel_active after close = %q; want empty", got)
				}
			},
		}
		s.Test(t)
	})

	t.Run("GET /ui/show/quests?status=bogusvalue → 400", func(t *testing.T) {
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
