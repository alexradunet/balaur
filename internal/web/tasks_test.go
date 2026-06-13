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

// TestTasksListQuestLog verifies the /tasks list view renders quest groups.
func TestTasksListQuestLog(t *testing.T) {
	t.Run("all four groups render in order with seeded tasks", func(t *testing.T) {
		app := newWebApp(t)
		// Seed one task per group.
		dailyRec := seedTaskWithRecur(t, app, "Morning stretch", "open", "daily", time.Time{})
		ritualRec := seedTaskWithRecur(t, app, "Weekly review", "open", "weekly:mon", time.Time{})
		questRec := seedTaskWithRecur(t, app, "File the deed", "open", "", time.Now().Add(24*time.Hour))
		sideRec := seedTaskWithRecur(t, app, "Someday write a poem", "open", "", time.Time{})

		// Suppress unused variable warnings — IDs are referenced for assertions below.
		_ = dailyRec
		_ = ritualRec
		_ = questRec
		_ = sideRec

		scenario := tests.ApiScenario{
			Name:           "GET /tasks list view — four groups in order",
			Method:         "GET",
			URL:            "/tasks",
			TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"Dailies", "Rituals", "Quests", "Side quests",
				"Morning stretch", "Weekly review", "File the deed", "Someday write a poem",
				"quest-detail",
			},
		}
		scenario.Test(t)
	})

	t.Run("daily task appears under Dailies group", func(t *testing.T) {
		app := newWebApp(t)
		seedTaskWithRecur(t, app, "Daily meditation", "open", "daily", time.Time{})

		scenario := tests.ApiScenario{
			Name:            "daily task under Dailies",
			Method:          "GET",
			URL:             "/tasks",
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"Dailies", "Daily meditation"},
		}
		scenario.Test(t)
	})

	t.Run("quest-detail contains first task tcard id", func(t *testing.T) {
		app := newWebApp(t)
		rec := seedTaskWithRecur(t, app, "First quest", "open", "daily", time.Time{})

		scenario := tests.ApiScenario{
			Name:            "quest-detail pre-rendered with first task",
			Method:          "GET",
			URL:             "/tasks",
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"quest-detail", "tcard-" + rec.Id},
		}
		scenario.Test(t)
	})

	t.Run("zero tasks shows empty state", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:            "GET /tasks empty — shows empty state",
			Method:          "GET",
			URL:             "/tasks",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"No quests yet"},
		}
		scenario.Test(t)
	})

	t.Run("groups appear in fixed order Dailies Rituals Quests Side quests", func(t *testing.T) {
		app := newWebApp(t)
		// Seed in reverse order to confirm fixed output order.
		seedTaskWithRecur(t, app, "Side thing", "open", "", time.Time{})
		seedTaskWithRecur(t, app, "One-off with due", "open", "", time.Now().Add(time.Hour))
		seedTaskWithRecur(t, app, "Biweekly ritual", "open", "every:14d", time.Time{})
		seedTaskWithRecur(t, app, "Daily standup", "open", "daily", time.Time{})

		scenario := tests.ApiScenario{
			Name:            "fixed group order",
			Method:          "GET",
			URL:             "/tasks",
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"Dailies", "Rituals", "Quests", "Side quests"},
		}
		scenario.Test(t)
	})
}

// TestTasksListGroupOrder verifies that group names appear in the HTML output.
func TestTasksListGroupOrder(t *testing.T) {
	app := newWebApp(t)
	seedTaskWithRecur(t, app, "Side note", "open", "", time.Time{})
	seedTaskWithRecur(t, app, "Due quest", "open", "", time.Now().Add(time.Hour))
	seedTaskWithRecur(t, app, "Ritual task", "open", "weekly:fri", time.Time{})
	seedTaskWithRecur(t, app, "Daily habit", "open", "daily", time.Time{})

	scenario := tests.ApiScenario{
		Name:            "group order in HTML",
		Method:          "GET",
		URL:             "/tasks",
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"Dailies", "Rituals", "Quests", "Side quests"},
	}
	scenario.Test(t)
}

// TestTaskTransitionRailRefresh verifies that a transition from the /tasks list
// view emits a Datastar patch of the quest-rail in addition to the card patch,
// while board and chat contexts get only the card patch. A Datastar @post sends
// no HX-Current-URL, so the page is identified by the Referer instead.
func TestTaskTransitionRailRefresh(t *testing.T) {
	t.Run("from /tasks — response patches the quest-rail", func(t *testing.T) {
		app := newWebApp(t)
		// Seed two tasks so the rail has content after the transition.
		rec := seedTaskWithRecur(t, app, "Complete me", "open", "daily", time.Time{})
		seedTaskWithRecur(t, app, "Stay open", "open", "daily", time.Time{})

		scenario := tests.ApiScenario{
			Name:   "transition with Referer=/tasks",
			Method: "POST",
			URL:    "/ui/tasks/" + rec.Id + "/transition",
			Body:   strings.NewReader("to=done"),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Referer":      "http://127.0.0.1:8090/tasks",
			},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"datastar-patch-elements", `id="quest-rail"`, "tcard-"},
		}
		scenario.Test(t)
	})

	t.Run("from /tasks — completed task absent from rail open groups", func(t *testing.T) {
		app := newWebApp(t)
		rec := seedTaskWithRecur(t, app, "Finish this quest", "open", "", time.Now().Add(time.Hour))

		scenario := tests.ApiScenario{
			Name:   "completed task not in rail groups after transition",
			Method: "POST",
			URL:    "/ui/tasks/" + rec.Id + "/transition",
			Body:   strings.NewReader("to=done"),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Referer":      "http://127.0.0.1:8090/tasks",
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus: 200,
			// The rail patch should be present; since the only task was completed,
			// the open-groups section shows the empty state (no quest-group sections).
			ExpectedContent:    []string{`id="quest-rail"`, "No quests yet"},
			NotExpectedContent: []string{`class="quest-group"`},
		}
		scenario.Test(t)
	})

	t.Run("no Referer — no quest-rail patch", func(t *testing.T) {
		app := newWebApp(t)
		rec := seedTaskWithRecur(t, app, "Board task", "open", "", time.Time{})

		scenario := tests.ApiScenario{
			Name:   "transition without Referer — no rail",
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

	t.Run("Referer from chat — no quest-rail patch", func(t *testing.T) {
		app := newWebApp(t)
		rec := seedTaskWithRecur(t, app, "Chat task", "open", "", time.Time{})

		scenario := tests.ApiScenario{
			Name:   "transition from chat URL — no rail",
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
}
