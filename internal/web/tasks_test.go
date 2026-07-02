package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
	itasks "github.com/alexradunet/balaur/internal/tasks"
	_ "github.com/alexradunet/balaur/migrations"
)

// seedTaskWithRecur seeds a task node with a given title, status, recur rule, and optional due.
func seedTaskWithRecur(t testing.TB, app *tests.TestApp, title, status, recur string, due time.Time) *core.Record {
	t.Helper()
	// Recurring tasks require a due anchor; supply one when the caller passes zero.
	if recur != "" && due.IsZero() {
		due = time.Now().Add(time.Hour)
	}
	rec, err := itasks.Create(app, itasks.CreateOpts{Title: title, Recur: recur, Due: due})
	if err != nil {
		t.Fatalf("create task %q: %v", title, err)
	}
	if status == "done" {
		if _, err = itasks.Done(app, rec, time.Now()); err != nil {
			t.Fatalf("done task %q: %v", title, err)
		}
		rec, err = app.FindRecordById("nodes", rec.Id)
		if err != nil {
			t.Fatalf("reload done task %q: %v", title, err)
		}
		itasks.Hydrate(rec)
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

// TestTaskTransitionEmitsToast verifies a transition also patches an owner-facing
// toast pill into the body-level #toast-region (plan 174 S7).
func TestTaskTransitionEmitsToast(t *testing.T) {
	cases := []struct{ to, tone, msg string }{
		{"done", "toast-success", "Marked done."},
		{"dropped", "toast-info", "Dropped."},
	}
	for _, c := range cases {
		t.Run(c.to, func(t *testing.T) {
			app := newWebApp(t)
			rec := seedTaskWithRecur(t, app, "Toast me", "open", "", time.Now().Add(time.Hour))
			scenario := tests.ApiScenario{
				Name:            "transition emits toast: " + c.to,
				Method:          "POST",
				URL:             "/ui/tasks/" + rec.Id + "/transition",
				Body:            strings.NewReader("to=" + c.to),
				Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
				TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
				ExpectedStatus:  200,
				ExpectedContent: []string{"toast-region", c.tone, c.msg},
			}
			scenario.Test(t)
		})
	}
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

// TestTaskEdit exercises the inline edit endpoint: the form's field set is
// applied through tasks.Update and the card re-renders in place with the new
// values (the same path the chat agent's task_update drives).
func TestTaskEdit(t *testing.T) {
	app := newWebApp(t)
	rec := seedTaskWithRecur(t, app, "Ship parcel", "open", "", time.Date(2026, 6, 24, 17, 0, 0, 0, time.Local))

	// The patched card carries the new title and notes — proof the handler
	// parsed the form, ran tasks.Update, and re-rendered. Due-reschedule
	// correctness is covered by the tasks/tools unit tests; the ApiScenario
	// harness closes the app after the run, so we assert on the response.
	scenario := tests.ApiScenario{
		Name:   "edit reschedules and renames in place",
		Method: "POST",
		URL:    "/ui/tasks/" + rec.Id + "/edit",
		Body:   strings.NewReader("title=Ship+parcel+today&due=2026-06-23T17:00&recur=&notes=SameDay+box"),
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"datastar-patch-elements", "tcard-" + rec.Id, "Ship parcel today", "SameDay box"},
	}
	scenario.Test(t)
}

// TestQuestsFocusPrefillsEditForm guards a regression: the inline edit form
// must pre-fill Due and Repeat on the quests surface (BuildQuestsFocus →
// taskViewOf), not only on the standalone card route. A blank-but-present form
// on the main task page defeats the feature.
func TestQuestsFocusPrefillsEditForm(t *testing.T) {
	app := newWebApp(t)
	seedTaskWithRecur(t, app, "Water plants", "open", "daily", time.Date(2030, 3, 4, 14, 30, 0, 0, time.Local))

	s := tests.ApiScenario{
		Name:           "quests focus pre-fills the inline edit form",
		Method:         "GET",
		URL:            "/ui/show/quests",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`type="datetime-local"`,
			`value="2030-03-04T14:30"`, // DueInput pre-fill
			`name="recur"`,
			`value="daily"`, // Recur pre-fill
		},
	}
	s.Test(t)
}

// ownerZoneSkipIfHostMatches skips owner-zone regression tests when the host
// runs at UTC+14 itself, in which case an owner-zone bug and correct
// owner-zone behavior would render identically.
func ownerZoneSkipIfHostMatches(t *testing.T) {
	t.Helper()
	if _, off := time.Now().Zone(); off == 14*3600 {
		t.Skip("host zone is UTC+14; owner-zone and host-zone results coincide")
	}
}

// TestTaskEditParsesDueInOwnerZone guards the regression this plan fixes:
// taskEdit must parse the datetime-local "due" field against the owner's
// configured timezone, not the host zone.
func TestTaskEditParsesDueInOwnerZone(t *testing.T) {
	ownerZoneSkipIfHostMatches(t)
	app := newWebApp(t)
	if err := store.SetOwnerSetting(app, "timezone", "Pacific/Kiritimati"); err != nil {
		t.Fatalf("set timezone: %v", err)
	}
	rec := seedTaskWithRecur(t, app, "Tax return", "open", "", time.Now().Add(time.Hour))

	s := tests.ApiScenario{
		Name:   "edit parses due in the owner zone",
		Method: "POST",
		URL:    "/ui/tasks/" + rec.Id + "/edit",
		Body:   strings.NewReader("title=Tax+return&due=2027-03-01T15:00&recur=&notes="),
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"tcard-" + rec.Id},
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
			got, err := app.FindRecordById("nodes", rec.Id)
			if err != nil {
				tb.Fatalf("reload: %v", err)
			}
			itasks.Hydrate(got)

			loc, err := time.LoadLocation("Pacific/Kiritimati")
			if err != nil {
				tb.Fatalf("load zone: %v", err)
			}
			want := time.Date(2027, 3, 1, 15, 0, 0, 0, loc)
			if d := got.GetDateTime("due").Time(); !d.Equal(want) {
				tb.Errorf("stored due = %v, want %v", d, want)
			}
		},
	}
	s.Test(t)
}

// TestQuestsFocusRendersOwnerZoneDue proves BuildQuestsFocus resolves now in
// the owner's zone: the edit-form due pre-fill (DueInput) must render in the
// owner zone, not the host zone.
func TestQuestsFocusRendersOwnerZoneDue(t *testing.T) {
	ownerZoneSkipIfHostMatches(t)
	app := newWebApp(t)
	if err := store.SetOwnerSetting(app, "timezone", "Pacific/Kiritimati"); err != nil {
		t.Fatalf("set timezone: %v", err)
	}
	// 02:00 UTC = 16:00 same day in UTC+14.
	seedTaskWithRecur(t, app, "Owner zone due", "open", "", time.Date(2030, 3, 4, 2, 0, 0, 0, time.UTC))

	s := tests.ApiScenario{
		Name:            "quests focus renders the owner-zone due pre-fill",
		Method:          "GET",
		URL:             "/ui/show/quests",
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{`value="2030-03-04T16:00"`},
	}
	s.Test(t)
}

// TestTaskSnoozeTomorrowUsesOwnerZone guards the "tomorrow" snooze quick-pick:
// its 09:00 target must land in the owner's zone, not the host zone.
func TestTaskSnoozeTomorrowUsesOwnerZone(t *testing.T) {
	ownerZoneSkipIfHostMatches(t)
	app := newWebApp(t)
	if err := store.SetOwnerSetting(app, "timezone", "Pacific/Kiritimati"); err != nil {
		t.Fatalf("set timezone: %v", err)
	}
	loc, err := time.LoadLocation("Pacific/Kiritimati")
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	rec := seedTaskWithRecur(t, app, "Snooze me", "open", "", time.Now().Add(time.Hour))

	// Compute the expectation before running the scenario (known negligible
	// edge: crossing owner-zone midnight between here and the request).
	ownerNow := time.Now().In(loc)
	want := time.Date(ownerNow.Year(), ownerNow.Month(), ownerNow.Day(), 9, 0, 0, 0, loc).AddDate(0, 0, 1)

	s := tests.ApiScenario{
		Name:   "snooze tomorrow targets the owner zone",
		Method: "POST",
		URL:    "/ui/tasks/" + rec.Id + "/transition",
		Body:   strings.NewReader("to=snooze&until=tomorrow"),
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"tcard-" + rec.Id},
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
			got, err := app.FindRecordById("nodes", rec.Id)
			if err != nil {
				tb.Fatalf("reload: %v", err)
			}
			itasks.Hydrate(got)

			props := nodes.Props(got)
			raw, _ := props["snoozed_until"].(string)
			gotUntil, err := store.ParsePBTime(raw)
			if err != nil {
				tb.Fatalf("parse snoozed_until %q: %v", raw, err)
			}
			if !gotUntil.Equal(want) {
				tb.Errorf("snoozed_until = %v, want %v", gotUntil, want)
			}
		},
	}
	s.Test(t)
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
