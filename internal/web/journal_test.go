package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/store"
	_ "github.com/alexradunet/balaur/migrations"
)

// newFakePromptServer returns a fake SSE model server that always responds
// with the given prompt text.
func newFakePromptServer(line string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"%s\"}}]}\n\n", line)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

// newFakeErrorServer returns a server that always responds with 500.
func newFakeErrorServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model unavailable", http.StatusInternalServerError)
	}))
}

func TestJournalPage(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:            "GET /journal returns candle page",
		Method:          "GET",
		URL:             "/journal",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  200,
		ExpectedContent: []string{"candle-page", "Keep it", "journal-form"},
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
			ExpectedContent: []string{entryText, "journal-candle-body"},
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
		sseSrv := newFakePromptServer(promptLine)
		t.Cleanup(func() { sseSrv.Close() })

		newPromptApp := func(tb testing.TB) *tests.TestApp {
			app := newWebApp(tb)
			id, _ := store.SaveOpenAIModel(app, "fake", sseSrv.URL+"/v1", "", "Fake", "fake-model", "", false)
			store.SetActiveLLMModel(app, id, "test")
			return app
		}

		scenario := tests.ApiScenario{
			Name:            "GET /ui/journal/prompt with model returns composed line",
			Method:          "GET",
			URL:             "/ui/journal/prompt",
			TestAppFactory:  newPromptApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{promptLine, `class="candle-prompt"`},
		}
		scenario.Test(t)
	})

	t.Run("with failing client falls back to deterministic line", func(t *testing.T) {
		errSrv := newFakeErrorServer()
		t.Cleanup(func() { errSrv.Close() })

		newErrApp := func(tb testing.TB) *tests.TestApp {
			app := newWebApp(tb)
			id, _ := store.SaveOpenAIModel(app, "fake-err", errSrv.URL+"/v1", "", "Fake", "fake-model", "", false)
			store.SetActiveLLMModel(app, id, "test")
			return app
		}

		scenario := tests.ApiScenario{
			Name:            "GET /ui/journal/prompt failing model returns fallback",
			Method:          "GET",
			URL:             "/ui/journal/prompt",
			TestAppFactory:  newErrApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{candlePromptFallback, `class="candle-prompt"`},
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
			ExpectedContent: []string{candlePromptFallback, `class="candle-prompt"`},
		}
		scenario.Test(t)
	})
}

// TestJournalCandleIntegration proves that an entry written via POST /ui/journal
// appears in GET /day/{today} — the candle and day pages share the same
// underlying journal records.
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

	// The entry must appear on the day page.
	scenario := tests.ApiScenario{
		Name:            "entry written via candle path appears on day page",
		Method:          "GET",
		URL:             "/day/" + today,
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{entryText},
	}
	scenario.Test(t)
}
