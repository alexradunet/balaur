package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/store"
	_ "github.com/alexradunet/balaur/migrations"
)

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

// TestPanelCollapsed verifies the collapse-when-empty derive and explicit override.
func TestPanelCollapsed(t *testing.T) {
	tests := []struct {
		name          string
		collapsed     string // "" means unset
		panelActive   string // "" means unset
		wantCollapsed bool
	}{
		{"explicit 1", "1", "", true},
		{"explicit 0", "0", "/ui/show/quests", false},
		{"explicit 0 with no active", "0", "", false},
		{"unset + no active (collapse-when-empty)", "", "", true},
		{"unset + active set (expand)", "", "/ui/show/quests", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newWebApp(t)
			if tt.collapsed != "" {
				if err := store.SetOwnerSetting(app, panelCollapsedKey, tt.collapsed); err != nil {
					t.Fatalf("SetOwnerSetting collapsed: %v", err)
				}
			}
			if tt.panelActive != "" {
				if err := store.SetOwnerSetting(app, panelActiveKey, tt.panelActive); err != nil {
					t.Fatalf("SetOwnerSetting active: %v", err)
				}
			}
			h := &handlers{app: app}
			got := h.panelCollapsed()
			if got != tt.wantCollapsed {
				t.Errorf("panelCollapsed() = %v, want %v", got, tt.wantCollapsed)
			}
		})
	}
}

// TestPanelWidthCSS verifies the inline style override and clamping.
func TestPanelWidthCSS(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"unset", "", ""},
		{"valid 600", "600", "--w-panel:600px"},
		{"below min (99)", "99", ""},
		{"above max (5000) clamped", "5000", "--w-panel:1100px"},
		{"non-numeric", "abc", ""},
		{"at min (320)", "320", "--w-panel:320px"},
		{"at max (1100)", "1100", "--w-panel:1100px"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newWebApp(t)
			if tt.raw != "" {
				if err := store.SetOwnerSetting(app, panelWidthKey, tt.raw); err != nil {
					t.Fatalf("SetOwnerSetting: %v", err)
				}
			}
			h := &handlers{app: app}
			got := h.panelWidthCSS()
			if got != tt.want {
				t.Errorf("panelWidthCSS() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestUIPanelCollapse verifies POST /ui/panel/collapse persists the flag.
func TestUIPanelCollapse(t *testing.T) {
	t.Run("on=1 sets panel_collapsed=1", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "POST /ui/panel/collapse on=1",
			Method:         "POST",
			URL:            "/ui/panel/collapse",
			Body:           strings.NewReader("on=1"),
			Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory: newWebApp,
			ExpectedStatus: http.StatusNoContent,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				got := store.GetOwnerSetting(app, panelCollapsedKey, "")
				if got != "1" {
					tb.Errorf("panel_collapsed = %q, want %q", got, "1")
				}
			},
		}
		s.Test(t)
	})
	t.Run("on=0 sets panel_collapsed=0", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "POST /ui/panel/collapse on=0",
			Method:         "POST",
			URL:            "/ui/panel/collapse",
			Body:           strings.NewReader("on=0"),
			Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory: newWebApp,
			ExpectedStatus: http.StatusNoContent,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				got := store.GetOwnerSetting(app, panelCollapsedKey, "")
				if got != "0" {
					tb.Errorf("panel_collapsed = %q, want %q", got, "0")
				}
			},
		}
		s.Test(t)
	})
}

// TestUIPanelWidth verifies POST /ui/panel/width persists the width with clamping.
func TestUIPanelWidth(t *testing.T) {
	t.Run("valid px=600 stored as 600", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "POST /ui/panel/width px=600",
			Method:         "POST",
			URL:            "/ui/panel/width",
			Body:           strings.NewReader("px=600"),
			Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory: newWebApp,
			ExpectedStatus: http.StatusNoContent,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				got := store.GetOwnerSetting(app, panelWidthKey, "")
				if got != "600" {
					tb.Errorf("panel_width = %q, want %q", got, "600")
				}
			},
		}
		s.Test(t)
	})
	t.Run("px below min clamped to 320", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "POST /ui/panel/width px=100 clamped",
			Method:         "POST",
			URL:            "/ui/panel/width",
			Body:           strings.NewReader("px=100"),
			Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory: newWebApp,
			ExpectedStatus: http.StatusNoContent,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				got := store.GetOwnerSetting(app, panelWidthKey, "")
				if got != "320" {
					tb.Errorf("panel_width = %q, want clamped to %q", got, "320")
				}
			},
		}
		s.Test(t)
	})
	t.Run("px above max clamped to 1100", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:           "POST /ui/panel/width px=5000 clamped",
			Method:         "POST",
			URL:            "/ui/panel/width",
			Body:           strings.NewReader("px=5000"),
			Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory: newWebApp,
			ExpectedStatus: http.StatusNoContent,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				got := store.GetOwnerSetting(app, panelWidthKey, "")
				if got != "1100" {
					tb.Errorf("panel_width = %q, want clamped to %q", got, "1100")
				}
			},
		}
		s.Test(t)
	})
	t.Run("bad px returns 400", func(t *testing.T) {
		s := tests.ApiScenario{
			Name:            "POST /ui/panel/width px=abc → 400",
			Method:          "POST",
			URL:             "/ui/panel/width",
			Body:            strings.NewReader("px=abc"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  newWebApp,
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{"Bad width"},
		}
		s.Test(t)
	})
}
