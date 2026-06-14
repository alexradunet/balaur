package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tools"
	"github.com/alexradunet/balaur/internal/turn"
	_ "github.com/alexradunet/balaur/migrations"
)

func newWebApp(t testing.TB) *tests.TestApp {
	t.Helper()
	// httptest requests default to Host "example.com"; allow it for tests
	// only — production allows loopback + BALAUR_ALLOWED_HOSTS.
	t.Setenv("BALAUR_ALLOWED_HOSTS", "example.com")
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

// newChoicesSSEServer creates a two-turn fake model server. The first request
// returns an offer_choices tool call; the second request (after tool execution)
// returns plain text so the agent completes.
func newChoicesSSEServer() *httptest.Server {
	calls := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		calls++
		if calls == 1 {
			// First turn: return an offer_choices tool call.
			args := `{\"choices\":[{\"label\":\"Yes, do it\",\"hint\":\"recommended\"},{\"label\":\"Not now\"}]}`
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"tc1\",\"function\":{\"name\":\"offer_choices\",\"arguments\":\"\"}}]}}]}\n\n")
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"name\":\"offer_choices\",\"arguments\":\"%s\"}}]}}]}\n\n", args)
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{}}],\"finish_reason\":\"tool_calls\"}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
		} else {
			// Second turn: plain text reply after tool executes.
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"I have offered you two choices.\"}}]}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
		}
	}))
}

func TestChatChoices(t *testing.T) {
	sseSrv := newChoicesSSEServer()
	t.Cleanup(func() { sseSrv.Close() })

	newChoicesApp := func(tb testing.TB) *tests.TestApp {
		app := newWebApp(tb)
		id, _ := store.SaveOpenAIModel(app, "fake", sseSrv.URL+"/v1", "", "Fake", "fake-model", "", false)
		store.SetActiveLLMModel(app, id, "test")
		return app
	}

	t.Run("streamed choices yield live panel", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:            "chat choices panel streamed",
			Method:          "POST",
			URL:             "/ui/chat",
			Body:            strings.NewReader("message=should+I+do+it"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  newChoicesApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{`class="choices-panel"`, `class="choice"`, "Yes, do it", "Not now"},
		}
		scenario.Test(t)
	})
}

// TestChoicesHistoryInert verifies that when a conversation containing a
// choices tool result is loaded from history (page-load path), the choices
// render as a plain inert tool row — no live panel, no clickable buttons.
func TestChoicesHistoryInert(t *testing.T) {
	tmpl := parseTemplates(t)

	// Simulate what messageViews produces for a tool message that carried choices.
	marked := tools.MarkChoices("Your word",
		[]tools.Choice{{Label: "Option A"}, {Label: "Option B"}},
		"offered choices: 1) Option A 2) Option B")

	// Parse as messageViews would.
	var content string
	if _, _, modelText, ok := tools.ParseChoices(marked); ok {
		content = clipText(modelText, 2000)
	}

	mv := messageView{
		Role:    "tool",
		Tool:    "offer_choices",
		Content: content,
	}

	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, "chat-msg-tool", mv); err != nil {
		t.Fatalf("chat-msg-tool: %v", err)
	}
	out := b.String()
	if strings.Contains(out, "choices-panel") {
		t.Error("history render of choices tool result must not contain choices-panel (must be inert)")
	}
	if strings.Contains(out, `class="choice"`) {
		t.Error("history render of choices tool result must not contain clickable choice buttons")
	}
	if !strings.Contains(out, "offered choices:") {
		t.Errorf("history render missing model text 'offered choices:': %s", out)
	}
}

func TestSettingsPages(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			// The settings shell is the settings card focus now (plan 056).
			Name:            "GET /focus/settings?section=profile renders profile section",
			Method:          "GET",
			URL:             "/focus/settings?section=profile",
			ExpectedStatus:  200,
			ExpectedContent: []string{"identity-card", "settings-nav"},
		},
		{
			Name:            "GET /focus/settings?section=models renders models section",
			Method:          "GET",
			URL:             "/focus/settings?section=models",
			ExpectedStatus:  200,
			ExpectedContent: []string{"models-panel"},
		},
		{
			// Retired route (plan 056): the settings shell is the settings card
			// focus now (/focus/settings, covered by TestFocusSettings*). The old
			// /settings page 302s to /boards.
			Name:           "GET /settings is retired (302)",
			Method:         "GET",
			URL:            "/settings",
			ExpectedStatus: 302,
		},
		{
			// Retired route (plan 056): the section pages folded into the focus.
			Name:           "GET /settings/profile is retired (302)",
			Method:         "GET",
			URL:            "/settings/profile",
			ExpectedStatus: 302,
		},
		{
			Name:           "GET /settings/models is retired (302)",
			Method:         "GET",
			URL:            "/settings/models",
			ExpectedStatus: 302,
		},
		{
			// Retired route (plan 056): /profile folded into the settings focus
			// (/focus/settings?section=profile).
			Name:           "GET /profile is retired (302)",
			Method:         "GET",
			URL:            "/profile",
			ExpectedStatus: 302,
		},
		{
			// Retired route (plan 056): /models folded into the settings focus
			// (/focus/settings?section=models).
			Name:           "GET /models is retired (302)",
			Method:         "GET",
			URL:            "/models",
			ExpectedStatus: 302,
		},
		{
			// Retired route (plan 053): the skills manager lives in the skills
			// card focus + /settings/skills now. The old /skills 302s to /boards.
			Name:           "GET /skills is retired (302 to /boards)",
			Method:         "GET",
			URL:            "/skills",
			ExpectedStatus: 302,
		},
		{
			// Retired route (plan 053): the memory manager moved into the memory
			// card focus (/focus/memory, covered by TestFocusMemoryShowsManager).
			// The old /memory page 302s to /boards.
			Name:           "GET /memory is retired (302 to /boards)",
			Method:         "GET",
			URL:            "/memory",
			ExpectedStatus: 302,
		},
		{
			// Retired route (plan 055): the life overview moved into the lifelog
			// card focus (/focus/lifelog, covered by TestFocusLifelogShowsOverview).
			// The old /life page 302s to /boards.
			Name:           "GET /life is retired (302 to /boards)",
			Method:         "GET",
			URL:            "/life",
			ExpectedStatus: 302,
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
			Name:           "/ redirects to the board dashboard (board-as-home)",
			Method:         "GET",
			URL:            "/",
			ExpectedStatus: 302,
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
			// The master chat now streams Datastar element patches: the server
			// echoes the owner's bubble, then morphs the assistant body with the
			// model's text. Both ride inside datastar-patch-elements SSE events.
			Name:            "chat streams Datastar element patches",
			Method:          "POST",
			URL:             "/ui/chat",
			Body:            strings.NewReader("message=hello"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			ExpectedStatus:  200,
			ExpectedContent: []string{"datastar-patch-elements", "Hello from the fake model", "msg msg-user", "msg msg-balaur"},
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

// TestHeadsFocus: the heads roster is a card focus (plan 054 retired GET
// /heads). The active head appears in the heads focus body.
func TestHeadsFocus(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name:   "active head appears on /focus/heads",
			Method: "GET",
			URL:    "/focus/heads",
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				app := newWebApp(tb)
				seedHeadRec(tb, app, "Scout", "active")
				return app
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"Scout", "head-"},
		},
	}
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestRetiredHeadsPages: plan 054 deleted the GET /heads and
// GET /heads/{id}/chat page routes. The roster now lives at /focus/heads and a
// head's branch chat opens in the dock (GET /ui/dock/conversation?head={id},
// covered by TestDockConversationBranch). The old page routes no longer exist;
// unmatched UI paths fall through to the board-home redirect (302 → /boards), so
// old bookmarks self-heal onto the boards dashboard.
func TestRetiredHeadsPages(t *testing.T) {
	t.Run("GET /heads is retired", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:           "GET /heads retired (302 to /boards)",
			Method:         "GET",
			URL:            "/heads",
			TestAppFactory: newWebApp,
			ExpectedStatus: 302,
		}
		scenario.Test(t)
	})
	t.Run("GET /heads/{id}/chat is retired", func(t *testing.T) {
		app := newWebApp(t)
		head := seedHeadRec(t, app, "Scout", "active")
		scenario := tests.ApiScenario{
			Name:           "GET /heads/{id}/chat retired (302 to /boards)",
			Method:         "GET",
			URL:            "/heads/" + head.Id + "/chat",
			TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus: 302,
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

	t.Run("streams Datastar element patches with user bubble", func(t *testing.T) {
		app, head := newHeadChatApp(t)
		// Like the master chat, the head chat now streams Datastar element
		// patches: the server echoes the owner's bubble, then morphs the
		// assistant body with the model's text. Both ride inside
		// datastar-patch-elements SSE events.
		scenario := tests.ApiScenario{
			Name:            "head chat basic",
			Method:          "POST",
			URL:             "/ui/heads/" + head.Id + "/chat",
			Body:            strings.NewReader("message=hello"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"datastar-patch-elements", "Hello from Scout", "msg msg-user"},
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

// TestGuardRejectsNonLoopbackHost verifies the DNS-rebinding guard rejects a
// non-loopback host that is not in BALAUR_ALLOWED_HOSTS (set to "example.com"
// by newWebApp), and allows a loopback IP.
//
// ApiScenario always builds requests with Host "example.com" (httptest default),
// so we drive the router directly to control req.Host.
func TestGuardRejectsNonLoopbackHost(t *testing.T) {
	// serveRequest builds the router from a newWebApp app and fires a single
	// GET "/" with the given host, returning the HTTP status code.
	serveRequest := func(t *testing.T, host string) int {
		t.Helper()
		app := newWebApp(t)
		defer app.Cleanup()

		baseRouter, err := apis.NewRouter(app)
		if err != nil {
			t.Fatalf("NewRouter: %v", err)
		}
		se := &core.ServeEvent{App: app, Router: baseRouter}
		if err := app.OnServe().Trigger(se, func(e *core.ServeEvent) error { return nil }); err != nil {
			t.Fatalf("OnServe trigger: %v", err)
		}
		mux, err := se.Router.BuildMux()
		if err != nil {
			t.Fatalf("BuildMux: %v", err)
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.Host = host
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Result().StatusCode
	}

	t.Run("evil.test host is rejected with 403", func(t *testing.T) {
		if got := serveRequest(t, "evil.test"); got != http.StatusForbidden {
			t.Errorf("expected 403 for evil.test, got %d", got)
		}
	})
	t.Run("loopback IP host passes guard", func(t *testing.T) {
		if got := serveRequest(t, "127.0.0.1:8090"); got == http.StatusForbidden {
			t.Errorf("expected non-403 for 127.0.0.1:8090, got %d", got)
		}
	})
}

// TestHardeningHeaders verifies the three hardening response headers are
// present on Balaur's own surfaces (not PocketBase's /api or /_).
func TestHardeningHeaders(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:            "GET /focus/memory carries hardening headers",
		Method:          "GET",
		URL:             "/focus/memory",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  200,
		ExpectedContent: []string{"k-active-grid"},
		AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
			for _, hdr := range []struct{ name, want string }{
				{"X-Content-Type-Options", "nosniff"},
				{"X-Frame-Options", "DENY"},
				{"Referrer-Policy", "same-origin"},
			} {
				if got := res.Header.Get(hdr.name); got != hdr.want {
					tb.Errorf("header %s = %q; want %q", hdr.name, got, hdr.want)
				}
			}
		},
	}
	scenario.Test(t)
}

func TestOriginGuard(t *testing.T) {
	// Origin guard is bound at the start of Register. This test verifies
	// it allows localhost (the test app default) and doesn't break normal requests.
	scenario := tests.ApiScenario{
		Name:            "origin guard allows localhost",
		Method:          "GET",
		URL:             "/focus/memory",
		Headers:         map[string]string{"Host": "localhost"},
		ExpectedStatus:  200,
		ExpectedContent: []string{"k-active-grid"},
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

	t.Run("GET /focus/settings?section=models shows provider name and key set, not secret", func(t *testing.T) {
		app, _, _, _, _ := newProviderApp(t)
		scenario := tests.ApiScenario{
			Name:               "models focus shows provider",
			Method:             "GET",
			URL:                "/focus/settings?section=models",
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

// newCardShowSSEServer creates a two-turn fake model server. The first request
// returns a card_show tool call (for the "today" card); the second returns plain
// text so the agent completes.
func newCardShowSSEServer() *httptest.Server {
	calls := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		calls++
		if calls == 1 {
			// First turn: return a card_show tool call.
			args := `{\"type\":\"today\"}`
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"tc1\",\"function\":{\"name\":\"card_show\",\"arguments\":\"\"}}]}}]}\n\n")
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"name\":\"card_show\",\"arguments\":\"%s\"}}]}}]}\n\n", args)
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{}}],\"finish_reason\":\"tool_calls\"}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
		} else {
			// Second turn: plain text reply after tool executes.
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Here is your today card.\"}}]}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
		}
	}))
}

func TestChatCardShow(t *testing.T) {
	sseSrv := newCardShowSSEServer()
	t.Cleanup(func() { sseSrv.Close() })

	newCardShowApp := func(tb testing.TB) *tests.TestApp {
		app := newWebApp(tb)
		id, _ := store.SaveOpenAIModel(app, "fake", sseSrv.URL+"/v1", "", "Fake", "fake-model", "", false)
		store.SetActiveLLMModel(app, id, "test")
		return app
	}

	t.Run("streamed card_show yields k-inline embed", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:           "chat card_show inline embed",
			Method:         "POST",
			URL:            "/ui/chat",
			Body:           strings.NewReader("message=show+me+today"),
			Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory: newCardShowApp,
			ExpectedStatus: 200,
			// The card is now server-rendered inline (no lazy hx-get mount).
			ExpectedContent: []string{`class="k-inline"`, `id="ucard-today"`},
		}
		scenario.Test(t)
	})
}

// TestUICardHistoryRendersCardInline verifies that a uicard tool result loaded
// from history embeds the card SERVER-RENDERED inline (the k-inline div carries
// the card markup directly — no lazy hx-get mount).
func TestUICardHistoryRendersCardInline(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	// Simulate what messageViews produces for a uicard-marked tool result.
	marked := tools.MarkUICard("today", map[string]string{}, "showing the owner the Today card")
	typ, query, rest, ok := tools.ParseUICard(marked)
	if !ok {
		t.Fatal("ParseUICard: ok=false on well-formed marked text")
	}

	mv := messageView{
		Role:     "tool",
		Tool:     "card_show",
		Content:  rest,
		CardBody: h.uicardBody(typ, query),
	}

	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "chat-msg-tool", mv); err != nil {
		t.Fatalf("chat-msg-tool: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, `class="k-inline"`) {
		t.Errorf("history uicard render: missing k-inline wrapper. output:\n%s", out)
	}
	if !strings.Contains(out, `id="ucard-today"`) {
		t.Errorf("history uicard render: card not embedded inline. output:\n%s", out)
	}
	if strings.Contains(out, "hx-get") {
		t.Errorf("history uicard render: stale lazy hx-get mount present. output:\n%s", out)
	}
}

// TestModelHandlers exercises the Ollama-backed model handlers without a live
// Ollama daemon. Both paths return before any network call: the progress
// fragment reads the idle in-process snapshot, and the delete guard short-
// circuits on the active model.
func TestModelHandlers(t *testing.T) {
	t.Run("pull progress endpoint renders when idle", func(t *testing.T) {
		// A fresh ollama.Default Snapshot is idle; this renders the empty
		// progress fragment (no Ollama daemon needed).
		scenario := tests.ApiScenario{
			Name:            "pull progress endpoint renders when idle",
			Method:          "GET",
			URL:             "/ui/model/gguf/progress",
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"gguf-progress"},
		}
		scenario.Test(t)
	})

	t.Run("delete refuses the active model", func(t *testing.T) {
		// Seed and activate a local model tag on the same app the handler uses
		// so the delete guard fires before any Ollama call.
		newGuardApp := func(tb testing.TB) *tests.TestApp {
			app := newWebApp(tb)
			id, err := store.SaveLocalModel(app, "gemma4:e4b", "embeddinggemma")
			if err != nil {
				tb.Fatalf("SaveLocalModel: %v", err)
			}
			if err := store.SetActiveLLMModel(app, id, "owner"); err != nil {
				tb.Fatalf("SetActiveLLMModel: %v", err)
			}
			return app
		}
		scenario := tests.ApiScenario{
			Name:            "delete refuses the active model",
			Method:          "POST",
			URL:             "/ui/model/gguf/delete",
			Body:            strings.NewReader("name=gemma4:e4b"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  newGuardApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"active model"},
		}
		scenario.Test(t)
	})

	t.Run("GPU preset tag is accepted", func(t *testing.T) {
		// Posting the GPU preset tag is allowlisted; the handler starts the pull
		// (returns immediately) and re-renders the models panel. No live Ollama
		// needed — Pull only launches a goroutine.
		scenario := tests.ApiScenario{
			Name:               "GPU preset tag accepted",
			Method:             "POST",
			URL:                "/ui/model/gguf/download",
			Body:               strings.NewReader("target=models&tag=gemma4:26b"),
			Headers:            map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:     newWebApp,
			ExpectedStatus:     200,
			ExpectedContent:    []string{"gguf-progress"},
			NotExpectedContent: []string{"unknown model preset"},
		}
		scenario.Test(t)
	})

	t.Run("unknown preset tag is rejected", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:            "unknown preset tag rejected",
			Method:          "POST",
			URL:             "/ui/model/gguf/download",
			Body:            strings.NewReader("target=models&tag=evil:1b"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  newWebApp,
			ExpectedStatus:  200,
			ExpectedContent: []string{"unknown model preset"},
		}
		scenario.Test(t)
	})
}

// TestDayPageRendersOnEmptyDB verifies that the day focus
// (GET /focus/day?date=…) returns 200 even when the database contains no entries
// or tasks — a blank day must not 500. (The standalone /day page is retired.)
func TestDayPageRendersOnEmptyDB(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:            "GET /focus/day?date=2026-01-15 renders 200 on empty DB",
		Method:          "GET",
		URL:             "/focus/day?date=2026-01-15",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  200,
		ExpectedContent: []string{"day-title", "January"},
	}
	scenario.Test(t)
}

// TestSelectModel covers the POST /ui/model/select handler for the three
// key paths: missing key, unknown key, and valid key.
func TestSelectModel(t *testing.T) {
	// Seed a provider+model so there is at least one choice available.
	newModelApp := func(tb testing.TB) (*tests.TestApp, string) {
		tb.Helper()
		app := newWebApp(tb)
		mid, err := store.SaveOpenAIModel(app, "Prov1", "https://p1.example.com/v1", "sk-test", "Model A", "model-a", "", false)
		if err != nil {
			tb.Fatalf("SaveOpenAIModel: %v", err)
		}
		return app, mid
	}

	t.Run("missing key returns 400", func(t *testing.T) {
		app, _ := newModelApp(t)
		scenario := tests.ApiScenario{
			Name:            "select model missing key",
			Method:          "POST",
			URL:             "/ui/model/select",
			Body:            strings.NewReader(""),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  400,
			ExpectedContent: []string{"Missing model key"},
		}
		scenario.Test(t)
	})

	t.Run("unknown key returns 400", func(t *testing.T) {
		app, _ := newModelApp(t)
		scenario := tests.ApiScenario{
			Name:            "select model unknown key",
			Method:          "POST",
			URL:             "/ui/model/select",
			Body:            strings.NewReader("key=no-such-model"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  400,
			ExpectedContent: []string{"Model is not available"},
		}
		scenario.Test(t)
	})

	t.Run("valid key activates model and returns 200", func(t *testing.T) {
		app, _ := newModelApp(t)

		// Derive the real key by calling ModelChoices — do not hardcode the format.
		choices, _, err := turn.ModelChoices(app)
		if err != nil {
			t.Fatalf("ModelChoices: %v", err)
		}
		// The openai model is not local so it is not Disabled (no file check).
		var validKey string
		for _, c := range choices {
			if !c.Disabled {
				validKey = c.Key
				break
			}
		}
		if validKey == "" {
			t.Fatal("no enabled model choice found in test app; cannot test valid key path")
		}

		scenario := tests.ApiScenario{
			Name:            "select model valid key",
			Method:          "POST",
			URL:             "/ui/model/select",
			Body:            strings.NewReader("key=" + validKey),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:  200,
			ExpectedContent: []string{"chatbar"},
			AfterTestFunc: func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
				// Verify the model is now active.
				_, active, err := turn.ModelChoices(a)
				if err != nil {
					tb.Fatalf("ModelChoices after select: %v", err)
				}
				if active.Key != validKey {
					tb.Errorf("active key = %q, want %q", active.Key, validKey)
				}
			},
		}
		scenario.Test(t)
	})
}
