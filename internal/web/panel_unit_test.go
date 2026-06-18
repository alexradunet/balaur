package web

import (
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/store"
	_ "github.com/alexradunet/balaur/migrations"
)

// TestUIPanelNav covers GET /ui/panel/{type}: the in-panel navigation door that
// morphs #panel-inner without persisting a transcript row or appending a chip.
func TestUIPanelNav(t *testing.T) {
	t.Run("GET /ui/panel/memory?category=person → morph + k-tabs, no chip", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "GET /ui/panel/memory?category=person",
			Method:         "GET",
			URL:            "/ui/panel/memory?category=person",
			TestAppFactory: newWebApp,
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"datastar-patch-elements",
				`id="panel-inner"`,
				`class="k-tabs"`,
				"k-active-grid",
			},
			// Must NOT append a chip or persist a chat row.
			NotExpectedContent: []string{
				"art-chip",
				`selector "#chat"`,
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				// panel_active must be set to the canonical show URL.
				got := store.GetOwnerSetting(app, panelActiveKey, "")
				want := "/ui/show/memory?category=person"
				if got != want {
					tb.Errorf("panel_active = %q; want %q", got, want)
				}
			},
		}
		s.Test(t)
	})

	t.Run("GET /ui/panel/close → empty panel, panel_active cleared", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "GET /ui/panel/close",
			Method:         "GET",
			URL:            "/ui/panel/close",
			TestAppFactory: newWebApp,
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"datastar-patch-elements",
				`id="panel-inner"`,
				"panel-empty",
			},
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				// Pre-seed panel_active so we can verify it is cleared.
				if err := store.SetOwnerSetting(app, panelActiveKey, "/ui/show/quests"); err != nil {
					tb.Fatalf("SetOwnerSetting: %v", err)
				}
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				// panel_active must be cleared after close. Using "" as default
				// so a returned "" means the stored value is empty or absent.
				got := store.GetOwnerSetting(app, panelActiveKey, "")
				if got != "" {
					tb.Errorf("panel_active after close = %q; want empty", got)
				}
			},
		}
		s.Test(t)
	})

	t.Run("GET /ui/panel/bogus → 404", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:            "GET /ui/panel/bogus",
			Method:          "GET",
			URL:             "/ui/panel/bogus",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  404,
			ExpectedContent: []string{"No such card type"},
		}
		s.Test(t)
	})
}

func TestParseShowURL(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantTyp   string
		wantQuery string
		wantOK    bool
	}{
		{"simple type", "/ui/show/quests", "quests", "", true},
		{"with query", "/ui/show/memory?category=fact", "memory", "category=fact", true},
		{"empty type", "/ui/show/", "", "", false},
		{"bad prefix", "/ui/cards/quests", "", "", false},
		{"empty string", "", "", "", false},
		{"show_cards prefix", "/ui/show_cards/foo", "", "", false},
		{"type with subpath", "/ui/show/settings?section=models", "settings", "section=models", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, query, ok := parseShowURL(tt.raw)
			if ok != tt.wantOK {
				t.Errorf("parseShowURL(%q) ok=%v, want %v", tt.raw, ok, tt.wantOK)
			}
			if typ != tt.wantTyp {
				t.Errorf("parseShowURL(%q) typ=%q, want %q", tt.raw, typ, tt.wantTyp)
			}
			if query != tt.wantQuery {
				t.Errorf("parseShowURL(%q) query=%q, want %q", tt.raw, query, tt.wantQuery)
			}
		})
	}
}

func TestShowURL(t *testing.T) {
	tests := []struct {
		typ   string
		query string
		want  string
	}{
		{"quests", "", "/ui/show/quests"},
		{"memory", "category=fact", "/ui/show/memory?category=fact"},
		{"settings", "section=models", "/ui/show/settings?section=models"},
	}
	for _, tt := range tests {
		got := showURL(tt.typ, tt.query)
		if got != tt.want {
			t.Errorf("showURL(%q, %q) = %q, want %q", tt.typ, tt.query, got, tt.want)
		}
	}
}
