package web

import (
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

func TestQuestGroup(t *testing.T) {
	tests := []struct {
		name   string
		recur  string
		hasDue bool
		want   string
	}{
		{name: "daily rule", recur: "daily", hasDue: false, want: "Dailies"},
		{name: "daily rule with due", recur: "daily", hasDue: true, want: "Dailies"},
		{name: "every:1d rule", recur: "every:1d", hasDue: false, want: "Dailies"},
		{name: "every:3d rule", recur: "every:3d", hasDue: false, want: "Rituals"},
		{name: "weekly:mon rule", recur: "weekly:mon", hasDue: false, want: "Rituals"},
		{name: "monthly:1 rule", recur: "monthly:1", hasDue: false, want: "Rituals"},
		{name: "bad rule with due", recur: "bogus-rule", hasDue: true, want: "Quests"},
		{name: "bad rule no due", recur: "bogus-rule", hasDue: false, want: "Side quests"},
		{name: "empty with due", recur: "", hasDue: true, want: "Quests"},
		{name: "empty no due", recur: "", hasDue: false, want: "Side quests"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := questGroup(tc.recur, tc.hasDue)
			if got != tc.want {
				t.Errorf("questGroup(%q, %v) = %q, want %q", tc.recur, tc.hasDue, got, tc.want)
			}
		})
	}
}

// seedTaskWithRecur seeds a task with a given title, status, recur rule, and optional due.
func seedTaskWithRecur(t testing.TB, app *tests.TestApp, title, status, recur string, due time.Time) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		t.Fatalf("find tasks collection: %v", err)
	}
	rec := core.NewRecord(coll)
	rec.Set("title", title)
	rec.Set("status", status)
	if recur != "" {
		rec.Set("recur", recur)
	}
	if !due.IsZero() {
		rec.Set("due", due.UTC())
	}
	if err := app.Save(rec); err != nil {
		t.Fatalf("save task: %v", err)
	}
	return rec
}

// TestQuestsArtifactEndpoint verifies GET /ui/show/quests injects a quests
// artifact into the chat stream. The flat stack rendering (Dailies/Rituals/Quests
// groups) is tested at component level in internal/feature/taskcards.
func TestQuestsArtifactEndpoint(t *testing.T) {
	t.Run("quests tile artifact injected", func(t *testing.T) {
		app := newWebApp(t)
		seedTaskWithRecur(t, app, "Morning stretch", "open", "daily", time.Time{})

		scenario := tests.ApiScenario{
			Name:            "GET /ui/show/quests injects quests artifact",
			Method:          "GET",
			URL:             "/ui/show/quests",
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"quest-stack", "Morning stretch"}, // flat stack (ui.Focus), task present
		}
		scenario.Test(t)
	})

	t.Run("zero tasks shows empty state", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:            "GET /ui/show/quests empty — shows empty state",
			Method:          "GET",
			URL:             "/ui/show/quests",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"No quests yet. Speak one in the chat."}, // QuestsFocus empty state
		}
		scenario.Test(t)
	})
}

// TestTaskTransitionRailRefresh verifies that a transition from any surface
// emits only the in-place card patch (#tcard-{id} outer). The quests artifact
// is now a flat stack — no separate rail OOB patch (plan 093).
func TestTaskTransitionRailRefresh(t *testing.T) {
	t.Run("from /ui/show/quests — response patches the card in place", func(t *testing.T) {
		app := newWebApp(t)
		rec := seedTaskWithRecur(t, app, "Complete me", "open", "daily", time.Time{})
		seedTaskWithRecur(t, app, "Stay open", "open", "daily", time.Time{})

		scenario := tests.ApiScenario{
			Name:   "transition with Referer=/ui/show/quests",
			Method: "POST",
			URL:    "/ui/tasks/" + rec.Id + "/transition",
			Body:   strings.NewReader("to=done"),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Referer":      "http://127.0.0.1:8090/ui/show/quests",
			},
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			ExpectedContent:    []string{"datastar-patch-elements", "tcard-"},
			NotExpectedContent: []string{`id="quest-rail"`},
		}
		scenario.Test(t)
	})

	t.Run("from /ui/show/quests — only card patch emitted", func(t *testing.T) {
		app := newWebApp(t)
		rec := seedTaskWithRecur(t, app, "Finish this quest", "open", "", time.Now().Add(time.Hour))

		scenario := tests.ApiScenario{
			Name:   "single card patch, no rail",
			Method: "POST",
			URL:    "/ui/tasks/" + rec.Id + "/transition",
			Body:   strings.NewReader("to=done"),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Referer":      "http://127.0.0.1:8090/ui/show/quests",
			},
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			ExpectedContent:    []string{"datastar-patch-elements", "tcard-"},
			NotExpectedContent: []string{`id="quest-rail"`},
		}
		scenario.Test(t)
	})

	t.Run("no Referer — card patch only", func(t *testing.T) {
		app := newWebApp(t)
		rec := seedTaskWithRecur(t, app, "Board task", "open", "", time.Time{})

		scenario := tests.ApiScenario{
			Name:   "transition without Referer",
			Method: "POST",
			URL:    "/ui/tasks/" + rec.Id + "/transition",
			Body:   strings.NewReader("to=done"),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			ExpectedContent:    []string{"tcard-"},
			NotExpectedContent: []string{`id="quest-rail"`},
		}
		scenario.Test(t)
	})

	t.Run("Referer from chat — card patch only", func(t *testing.T) {
		app := newWebApp(t)
		rec := seedTaskWithRecur(t, app, "Chat task", "open", "", time.Time{})

		scenario := tests.ApiScenario{
			Name:   "transition from chat URL",
			Method: "POST",
			URL:    "/ui/tasks/" + rec.Id + "/transition",
			Body:   strings.NewReader("to=dropped"),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Referer":      "http://127.0.0.1:8090/",
			},
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			NotExpectedContent: []string{`id="quest-rail"`},
		}
		scenario.Test(t)
	})

	t.Run("board ✓ row — remove patch, no card or rail", func(t *testing.T) {
		app := newWebApp(t)
		rec := seedTaskWithRecur(t, app, "Row task", "open", "", time.Time{})

		// A board today/quests ✓ sends src=today|quests; the handler removes the
		// matching row by a server-built id rather than rendering the card.
		scenario := tests.ApiScenario{
			Name:   "transition with src=today removes the row",
			Method: "POST",
			URL:    "/ui/tasks/" + rec.Id + "/transition",
			Body:   strings.NewReader("to=done&src=today"),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			ExpectedContent:    []string{"datastar-patch-elements", "mode remove", "urow-today-" + rec.Id},
			NotExpectedContent: []string{"tcard-", `id="quest-rail"`},
		}
		scenario.Test(t)
	})

	t.Run("board ✓ row from quests — remove patch, no card or rail", func(t *testing.T) {
		app := newWebApp(t)
		rec := seedTaskWithRecur(t, app, "Quest row task", "open", "", time.Time{})

		// The quests-tile ✓ sends src=quests; the handler removes the matching
		// row by a server-built id (urow-quests-{id}) rather than rendering the card.
		scenario := tests.ApiScenario{
			Name:   "transition with src=quests removes the row",
			Method: "POST",
			URL:    "/ui/tasks/" + rec.Id + "/transition",
			Body:   strings.NewReader("to=done&src=quests"),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			ExpectedContent:    []string{"datastar-patch-elements", "mode remove", "urow-quests-" + rec.Id},
			NotExpectedContent: []string{"tcard-", `id="quest-rail"`},
		}
		scenario.Test(t)
	})
}

// TestTasksRouteRetired guards against accidental re-registration of the
// standalone /tasks page. The route is unregistered, so PocketBase's index
// the catch-all handler redirects it home (302 → /) rather than serving
// its own page. The guard asserts that fallback, not a 200 task surface.
func TestTasksRouteRetired(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /tasks is retired (302)",
		Method:         "GET",
		URL:            "/tasks",
		TestAppFactory: newWebApp,
		ExpectedStatus: 302,
	}
	s.Test(t)
}
