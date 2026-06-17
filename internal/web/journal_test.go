package web

import (
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/llmtest"
	_ "github.com/alexradunet/balaur/migrations"
)

// TestJournalArtifact: /ui/show/journal injects the journal tile artifact into
// the chat stream. The full candle surface (JournalFocus) is tested at component
// level in internal/feature/journalcards.
func TestJournalArtifact(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:            "GET /ui/show/journal injects journal artifact",
		Method:          "GET",
		URL:             "/ui/show/journal",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  200,
		ExpectedContent: []string{"candle-focus", `@post(&#39;/ui/journal&#39;`}, // the full candle write-form, not a summary
	}
	scenario.Test(t)
}

func TestJournalWrite(t *testing.T) {
	t.Run("write entry returns fragment with text", func(t *testing.T) {
		const entryText = "The evening was still and the light was good."
		scenario := tests.ApiScenario{
			Name:            "POST /ui/journal writes entry",
			Method:          "POST",
			URL:             "/ui/journal",
			Body:            strings.NewReader("text=" + entryText),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"datastar-patch-elements", entryText, "journal-candle-body"},
		}
		scenario.Test(t)
	})

	t.Run("empty text returns 400", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:            "POST /ui/journal empty text",
			Method:          "POST",
			URL:             "/ui/journal",
			Body:            strings.NewReader("text="),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  newWebApp,
			ExpectedStatus:  400,
			ExpectedContent: []string{"empty"},
		}
		scenario.Test(t)
	})
}

func TestJournalPrompt(t *testing.T) {
	t.Run("with active model returns model line", func(t *testing.T) {
		const promptLine = "Let the day speak through you."
		newPromptApp := func(tb testing.TB) *tests.TestApp {
			app := newWebApp(tb)
			seedScriptedModel(tb, app, llmtest.Text(promptLine))
			return app
		}

		scenario := tests.ApiScenario{
			Name:            "GET /ui/journal/prompt with model returns composed line",
			Method:          "GET",
			URL:             "/ui/journal/prompt",
			TestAppFactory:  newPromptApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"datastar-patch-elements", promptLine, `class="candle-prompt"`},
		}
		scenario.Test(t)
	})

	t.Run("with failing client falls back to deterministic line", func(t *testing.T) {
		newErrApp := func(tb testing.TB) *tests.TestApp {
			app := newWebApp(tb)
			seedFailingModel(tb, app)
			return app
		}

		scenario := tests.ApiScenario{
			Name:            "GET /ui/journal/prompt failing model returns fallback",
			Method:          "GET",
			URL:             "/ui/journal/prompt",
			TestAppFactory:  newErrApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"datastar-patch-elements", candlePromptFallback, `class="candle-prompt"`},
		}
		scenario.Test(t)
	})

	t.Run("without model falls back to deterministic line", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:            "GET /ui/journal/prompt no model returns fallback",
			Method:          "GET",
			URL:             "/ui/journal/prompt",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"datastar-patch-elements", candlePromptFallback, `class="candle-prompt"`},
		}
		scenario.Test(t)
	})
}

// TestJournalCandleIntegration proves that an entry written via POST /ui/journal
// increments the day tile's journal count — the candle and the day surface share
// the same underlying journal records.
func TestJournalCandleIntegration(t *testing.T) {
	const entryText = "The river was quiet at dawn and the fog had not yet lifted."

	app := newWebApp(t)

	// Write an entry directly via life.JournalWrite — the same path POST
	// /ui/journal calls.
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
		Name:            "day view reflects journal entry written via candle",
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
// catch-all handler redirects them home (302 → /). Their write paths
// (/ui/journal, /ui/day/…) and the journal/day card artifacts live on.
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
