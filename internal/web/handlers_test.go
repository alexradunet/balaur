package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/store"
	_ "github.com/alexradunet/balaur/migrations"
)

func newWebApp(t testing.TB) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("test app: %v", err)
	}
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := Register(se); err != nil {
			return err
		}
		return se.Next()
	})
	return app
}

func newFakeSSEServer(text string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"%s\"}}]}\n\n", text)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

func TestSettingsPages(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name:           "GET /settings redirects",
			Method:         "GET",
			URL:            "/settings",
			ExpectedStatus: 302,
		},
		{
			Name:            "GET /settings/profile renders profile section",
			Method:          "GET",
			URL:             "/settings/profile",
			ExpectedStatus:  200,
			ExpectedContent: []string{"identity-card", "settings-nav"},
		},
		{
			Name:            "GET /settings/models renders models section",
			Method:          "GET",
			URL:             "/settings/models",
			ExpectedStatus:  200,
			ExpectedContent: []string{"models-panel"},
		},
		{
			Name:            "GET /settings/skills renders skills section",
			Method:          "GET",
			URL:             "/settings/skills",
			ExpectedStatus:  200,
			ExpectedContent: []string{"k-active-grid"},
		},
		{
			Name:           "GET /settings/bogus redirects",
			Method:         "GET",
			URL:            "/settings/bogus",
			ExpectedStatus: 302,
		},
		{
			Name:           "GET /profile redirects",
			Method:         "GET",
			URL:            "/profile",
			ExpectedStatus: 302,
		},
		{
			Name:           "GET /models redirects",
			Method:         "GET",
			URL:            "/models",
			ExpectedStatus: 302,
		},
		{
			Name:           "GET /skills redirects",
			Method:         "GET",
			URL:            "/skills",
			ExpectedStatus: 302,
		},
		{
			Name:            "GET /memory still renders k-active-grid",
			Method:          "GET",
			URL:             "/memory",
			ExpectedStatus:  200,
			ExpectedContent: []string{"k-active-grid"},
		},
	}

	for _, scenario := range scenarios {
		scenario.TestAppFactory = newWebApp
		scenario.Test(t)
	}
}

func TestHandlerHomePage(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name:            "home",
			Method:          "GET",
			URL:             "/",
			ExpectedStatus:  200,
			ExpectedContent: []string{"chatbar"},
		},
	}

	for _, scenario := range scenarios {
		scenario.TestAppFactory = newWebApp
		scenario.Test(t)
	}
}

func TestChatHandler(t *testing.T) {
	sseSrv := newFakeSSEServer("Hello from the fake model")
	t.Cleanup(func() { sseSrv.Close() })

	// Each ApiScenario.Test re-fires OnServe, causing route registration
	// conflicts when sharing one app. Give each scenario its own app with
	// the model already seeded so the factory is a plain getter.
	newChatApp := func(tb testing.TB) *tests.TestApp {
		app := newWebApp(tb)
		id, _ := store.SaveOpenAIModel(app, "fake", sseSrv.URL+"/v1", "", "Fake", "fake-model", "", false)
		store.SetActiveLLMModel(app, id, "test")
		return app
	}

	scenarios := []tests.ApiScenario{
		{
			Name:            "chat with client_rendered=0",
			Method:          "POST",
			URL:             "/ui/chat",
			Body:            strings.NewReader("message=hello"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			ExpectedStatus:  200,
			ExpectedContent: []string{"Hello from the fake model", "msg msg-user"},
		},
		{
			Name:               "chat with client_rendered=1",
			Method:             "POST",
			URL:                "/ui/chat",
			Body:               strings.NewReader("message=hello&client_rendered=1"),
			Headers:            map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			ExpectedStatus:     200,
			ExpectedContent:    []string{"Hello from the fake model"},
			NotExpectedContent: []string{"msg msg-user"},
		},
		{
			Name:            "chat empty message",
			Method:          "POST",
			URL:             "/ui/chat",
			Body:            strings.NewReader("message="),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			ExpectedStatus:  400,
			ExpectedContent: []string{"Empty message"},
		},
	}

	for _, scenario := range scenarios {
		scenario.TestAppFactory = newChatApp
		scenario.Test(t)
	}
}

func TestTaskTransition(t *testing.T) {
	app := newWebApp(t)
	coll, _ := app.FindCollectionByNameOrId("tasks")
	rec := core.NewRecord(coll)
	rec.Set("title", "Test")
	rec.Set("status", "open")
	app.Save(rec)

	scenario := tests.ApiScenario{
		Name:            "transition task to done",
		Method:          "POST",
		URL:             "/ui/tasks/" + rec.Id + "/transition",
		Body:            strings.NewReader("to=done"),
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		ExpectedStatus:  200,
		ExpectedContent: []string{"tcard-done", "Test"},
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
	}
	scenario.Test(t)
}

// seedHeadRec creates an active head record for web handler tests.
func seedHeadRec(t testing.TB, app *tests.TestApp, name, status string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		t.Fatalf("heads collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("status", status)
	rec.SetEmail(fmt.Sprintf("head-%d@balaur.local", time.Now().UnixNano()))
	rec.SetRandomPassword()
	if err := app.Save(rec); err != nil {
		t.Fatalf("saving head: %v", err)
	}
	return rec
}

func TestHeadsPage(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name:   "active head appears on /heads",
			Method: "GET",
			URL:    "/heads",
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				app := newWebApp(tb)
				seedHeadRec(tb, app, "Scout", "active")
				return app
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"Scout"},
		},
	}
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestHeadChatPage(t *testing.T) {
	// URLs contain record IDs that must be seeded first, so we run subtests
	// rather than an ApiScenario table.
	t.Run("active head", func(t *testing.T) {
		app := newWebApp(t)
		head := seedHeadRec(t, app, "Scout", "active")
		scenario := tests.ApiScenario{
			Name:            "active head renders chat page",
			Method:          "GET",
			URL:             "/heads/" + head.Id + "/chat",
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"Scout"},
		}
		scenario.Test(t)
	})
	t.Run("merged head is forbidden", func(t *testing.T) {
		app := newWebApp(t)
		head := seedHeadRec(t, app, "OldHead", "merged")
		scenario := tests.ApiScenario{
			Name:            "merged head is forbidden",
			Method:          "GET",
			URL:             "/heads/" + head.Id + "/chat",
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  403,
			ExpectedContent: []string{"not active"},
		}
		scenario.Test(t)
	})
	t.Run("unknown id is not found", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:            "unknown head id is 404",
			Method:          "GET",
			URL:             "/heads/nope/chat",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  404,
			ExpectedContent: []string{"not found"},
		}
		scenario.Test(t)
	})
}

func TestHeadChat(t *testing.T) {
	sseSrv := newFakeSSEServer("Hello from Scout")
	t.Cleanup(func() { sseSrv.Close() })

	newHeadChatApp := func(tb testing.TB) (*tests.TestApp, *core.Record) {
		app := newWebApp(tb)
		id, _ := store.SaveOpenAIModel(app, "fake", sseSrv.URL+"/v1", "", "Fake", "fake-model", "", false)
		store.SetActiveLLMModel(app, id, "test")
		head := seedHeadRec(tb, app, "Scout", "active")
		return app, head
	}

	t.Run("message without client_rendered includes user bubble", func(t *testing.T) {
		app, head := newHeadChatApp(t)
		scenario := tests.ApiScenario{
			Name:            "head chat basic",
			Method:          "POST",
			URL:             "/ui/heads/" + head.Id + "/chat",
			Body:            strings.NewReader("message=hello"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"Hello from Scout", "msg msg-user"},
		}
		scenario.Test(t)
	})
	t.Run("message with client_rendered=1 skips user bubble", func(t *testing.T) {
		app, head := newHeadChatApp(t)
		scenario := tests.ApiScenario{
			Name:               "head chat client rendered",
			Method:             "POST",
			URL:                "/ui/heads/" + head.Id + "/chat",
			Body:               strings.NewReader("message=hello&client_rendered=1"),
			Headers:            map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			ExpectedContent:    []string{"Hello from Scout"},
			NotExpectedContent: []string{"msg msg-user"},
		}
		scenario.Test(t)
	})
	t.Run("empty message returns 400", func(t *testing.T) {
		app, head := newHeadChatApp(t)
		scenario := tests.ApiScenario{
			Name:            "head chat empty message",
			Method:          "POST",
			URL:             "/ui/heads/" + head.Id + "/chat",
			Body:            strings.NewReader("message="),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  400,
			ExpectedContent: []string{"Empty message"},
		}
		scenario.Test(t)
	})
}

func TestSetHeadAvatar(t *testing.T) {
	// Confirm balaur-01 is a valid key before using it in the test.
	if !store.ValidBalaurAvatarKey("balaur-01") {
		t.Fatal("expected balaur-01 to be a valid avatar key")
	}

	t.Run("valid avatar key returns 200", func(t *testing.T) {
		app := newWebApp(t)
		head := seedHeadRec(t, app, "Scout", "active")
		scenario := tests.ApiScenario{
			Name:            "set head avatar valid key",
			Method:          "POST",
			URL:             "/ui/heads/" + head.Id + "/avatar",
			Body:            strings.NewReader("balaur_avatar=balaur-01"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"Scout"},
		}
		scenario.Test(t)
	})
	t.Run("bogus avatar key returns 400", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:            "set head avatar bogus key",
			Method:          "POST",
			URL:             "/ui/heads/someid/avatar",
			Body:            strings.NewReader("balaur_avatar=bogus"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  newWebApp,
			ExpectedStatus:  400,
			ExpectedContent: []string{"Invalid balaur avatar"},
		}
		scenario.Test(t)
	})
}

func TestOriginGuard(t *testing.T) {
	// Origin guard is bound at the start of Register. This test verifies
	// it allows localhost (the test app default) and doesn't break normal requests.
	scenario := tests.ApiScenario{
		Name:            "origin guard allows localhost",
		Method:          "GET",
		URL:             "/",
		Headers:         map[string]string{"Host": "localhost"},
		ExpectedStatus:  200,
		ExpectedContent: []string{"chatbar"},
	}
	scenario.TestAppFactory = newWebApp
	scenario.Test(t)
}

func TestProviderManager(t *testing.T) {
	const secretKey = "sk-test-secret-zzz"

	// newProviderApp seeds two providers and makes prov1's model active.
	// Returns (app, prov1ID, model1ID, prov2ID, model2ID).
	newProviderApp := func(tb testing.TB) (*tests.TestApp, string, string, string, string) {
		tb.Helper()
		app := newWebApp(tb)
		mid1, err := store.SaveOpenAIModel(app, "Prov1", "https://p1.example.com/v1", secretKey, "Model A", "model-a", "", false)
		if err != nil {
			tb.Fatalf("save prov1: %v", err)
		}
		mid2, err := store.SaveOpenAIModel(app, "Prov2", "https://p2.example.com/v1", "sk-other", "Model B", "model-b", "", false)
		if err != nil {
			tb.Fatalf("save prov2: %v", err)
		}
		if err := store.SetActiveLLMModel(app, mid1, "test"); err != nil {
			tb.Fatalf("set active: %v", err)
		}
		providers, err := store.ListOpenAIProviders(app)
		if err != nil {
			tb.Fatalf("list providers: %v", err)
		}
		var prov1ID, prov2ID string
		for _, p := range providers {
			if p.Name == "Prov1" {
				prov1ID = p.ID
			} else if p.Name == "Prov2" {
				prov2ID = p.ID
			}
		}
		if prov1ID == "" || prov2ID == "" {
			tb.Fatalf("provider IDs not found; prov1=%q prov2=%q", prov1ID, prov2ID)
		}
		return app, prov1ID, mid1, prov2ID, mid2
	}

	t.Run("GET /settings/models shows provider name and key set, not secret", func(t *testing.T) {
		app, _, _, _, _ := newProviderApp(t)
		scenario := tests.ApiScenario{
			Name:               "models page shows provider",
			Method:             "GET",
			URL:                "/settings/models",
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			ExpectedContent:    []string{"Prov1", "key set"},
			NotExpectedContent: []string{secretKey},
		}
		scenario.Test(t)
	})

	t.Run("delete active provider returns refusal message", func(t *testing.T) {
		app, prov1ID, _, _, _ := newProviderApp(t)
		// Verify state before scenario (scenario's AfterTestFunc runs while app is alive).
		scenario := tests.ApiScenario{
			Name:            "delete active provider refused",
			Method:          "POST",
			URL:             "/ui/model/provider/" + prov1ID + "/delete",
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"active model", "models-panel"},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				// Prov1 should still be listed after the refused delete.
				providers, err := store.ListOpenAIProviders(app)
				if err != nil {
					t.Fatalf("list after refused delete: %v", err)
				}
				var found bool
				for _, p := range providers {
					if p.ID == prov1ID {
						found = true
						break
					}
				}
				if !found {
					t.Fatal("provider was deleted despite having the active model")
				}
			},
		}
		scenario.Test(t)
	})

	t.Run("delete after re-pointing active succeeds", func(t *testing.T) {
		app, prov1ID, _, _, mid2 := newProviderApp(t)
		// Re-point active to prov2's model.
		if err := store.SetActiveLLMModel(app, mid2, "test"); err != nil {
			t.Fatalf("re-point active: %v", err)
		}
		scenario := tests.ApiScenario{
			Name:            "delete prov1 after re-point",
			Method:          "POST",
			URL:             "/ui/model/provider/" + prov1ID + "/delete",
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"models-panel"},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				// Prov1 should be gone.
				providers, err := store.ListOpenAIProviders(app)
				if err != nil {
					t.Fatalf("list after delete: %v", err)
				}
				for _, p := range providers {
					if p.ID == prov1ID {
						t.Fatal("provider still present after delete")
					}
				}
			},
		}
		scenario.Test(t)
	})

	t.Run("update with blank key keeps existing secret", func(t *testing.T) {
		app, prov1ID, _, _, _ := newProviderApp(t)
		scenario := tests.ApiScenario{
			Name:            "update provider blank key",
			Method:          "POST",
			URL:             "/ui/model/provider/" + prov1ID + "/save",
			Body:            strings.NewReader("name=Prov1+Renamed&base_url=https%3A%2F%2Fp1.example.com%2Fv1&api_key="),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"models-panel"},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				// Raw record should still have the original key.
				raw, err := app.FindRecordById("llm_providers", prov1ID)
				if err != nil {
					t.Fatalf("find raw: %v", err)
				}
				if raw.GetString("api_key") != secretKey {
					t.Fatalf("key overwritten: got %q, want %q", raw.GetString("api_key"), secretKey)
				}
			},
		}
		scenario.Test(t)
	})
}

func TestGgufHandlers(t *testing.T) {
	t.Run("delete path traversal returns 200 panel", func(t *testing.T) {
		// The traversal is rejected inside gguf.Delete; the handler re-renders the
		// panel with the error message (200, not a 4xx) per the plan's HTMX pattern.
		scenario := tests.ApiScenario{
			Name:            "delete path traversal",
			Method:          "POST",
			URL:             "/ui/model/gguf/delete",
			Body:            strings.NewReader("name=../../evil.gguf"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"models-panel"},
		}
		scenario.Test(t)
	})

	t.Run("download ftp URL returns panel with error", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:            "download ftp scheme rejected",
			Method:          "POST",
			URL:             "/ui/model/gguf/download",
			Body:            strings.NewReader("url=ftp://x/m.gguf"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"models-panel", "http or https"},
		}
		scenario.Test(t)
	})

	t.Run("end-to-end tiny GGUF download", func(t *testing.T) {
		// Serve a tiny GGUF payload from a local httptest server. The server
		// uses a channel to synchronise with the test: it waits until the
		// test signals "allow response", so we can verify the file and record
		// AFTER the goroutine has definitely completed.
		payload := []byte("GGUFtiny test model payload for handler test")
		allowResp := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			<-allowResp // wait until the test allows the response
			w.Write(payload)
		}))
		defer srv.Close()

		// Use a single app for all sub-scenarios so the background goroutine
		// does not outlive the app. DisableTestAppCleanup prevents automatic
		// cleanup; we clean up manually after all checks are done.
		e2eApp := newWebApp(t)
		t.Cleanup(e2eApp.Cleanup)
		factory := func(tb testing.TB) *tests.TestApp { return e2eApp }

		// POST to gguf/download — starts the background goroutine; panel returned immediately.
		downloadScenario := tests.ApiScenario{
			Name:                  "gguf download starts",
			Method:                "POST",
			URL:                   "/ui/model/gguf/download",
			Body:                  strings.NewReader("url=" + srv.URL + "/testmodel.gguf&activate=1"),
			Headers:               map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:        factory,
			DisableTestAppCleanup: true,
			ExpectedStatus:        200,
			ExpectedContent:       []string{"models-panel"},
		}
		downloadScenario.Test(t)

		// Allow the server to respond — the goroutine is now blocked waiting
		// for the HTTP response. Unblocking it lets it complete the download.
		close(allowResp)

		// Poll for the file — the goroutine should finish very quickly now
		// that the server has responded with the 44-byte payload.
		destFile := filepath.Join(e2eApp.DataDir(), "models", "testmodel.gguf")
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, statErr := os.Stat(destFile); statErr == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if _, statErr := os.Stat(destFile); statErr != nil {
			t.Fatalf("GGUF file not found at %s after download", destFile)
		}

		// Poll for the llm_models record created by onDone.
		deadline = time.Now().Add(5 * time.Second)
		var foundModel bool
		for time.Now().Before(deadline) {
			models, listErr := store.ListLLMModels(e2eApp)
			if listErr == nil {
				for _, m := range models {
					if m.Kind == "kronk" && filepath.Base(m.ChatModel) == "testmodel.gguf" {
						foundModel = true
						break
					}
				}
			}
			if foundModel {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !foundModel {
			t.Fatal("llm_models record for testmodel.gguf not found after download")
		}

		// Confirm the progress endpoint renders (idle on any fresh app).
		progressScenario := tests.ApiScenario{
			Name:            "gguf progress endpoint renders when idle",
			Method:          "GET",
			URL:             "/ui/model/gguf/progress",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"gguf-progress"},
		}
		progressScenario.Test(t)
	})
}
