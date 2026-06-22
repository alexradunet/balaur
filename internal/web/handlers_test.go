package web

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/kronk"
	"github.com/alexradunet/balaur/internal/llmtest"
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
	// OnServe only fires inside an ApiScenario HTTP request, so tests that build
	// &handlers{} directly and render a card never populate the gomponents card
	// registry. Register the features explicitly here (web.go blank-imports
	// feature/all, so the registry is populated for the test binary); t.Cleanup
	// keeps the process-global registry clean between test apps.
	feature.RegisterAll(app)
	t.Cleanup(feature.UnregisterAll)
	return app
}

func TestChatChoices(t *testing.T) {
	newChoicesApp := func(tb testing.TB) *tests.TestApp {
		app := newWebApp(tb)
		seedScriptedModel(tb, app,
			llmtest.ToolCall("tc1", "offer_choices", `{"choices":[{"label":"Yes, do it","hint":"recommended"},{"label":"Not now"}]}`),
			llmtest.Text("I have offered you two choices."),
		)
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
			// /ui/show/settings (no section) defaults to the profile section:
			// a nav-free artifact showing identity + avatar pickers (plan 092).
			Name:           "GET /ui/show/settings injects the profile section artifact",
			Method:         "GET",
			URL:            "/ui/show/settings",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`id="identity-card"`,               // the Profile section's identity form
				`@post(&#39;/ui/profile/name&#39;`, // a working write form is present
			},
			NotExpectedContent: []string{
				`settings-layout`, `settings-nav`,
			},
		},
		{
			// /ui/show/settings?section=models renders the models panel without nav.
			Name:           "GET /ui/show/settings?section=models injects the models section artifact",
			Method:         "GET",
			URL:            "/ui/show/settings?section=models",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`id="models-panel"`,
			},
			NotExpectedContent: []string{
				`settings-nav`,
			},
		},
		{
			// Retired route (plan 056): the settings shell is the settings card
			// focus now (/ui/show/settings). The old /settings page 302s to /.
			Name:           "GET /settings is retired (302)",
			Method:         "GET",
			URL:            "/settings",
			ExpectedStatus: 302,
		},
		{
			// Retired route (plan 056): the section pages folded into the artifact.
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
			// Retired route (plan 056): /profile folded into /ui/show/settings?section=profile.
			Name:           "GET /profile is retired (302)",
			Method:         "GET",
			URL:            "/profile",
			ExpectedStatus: 302,
		},
		{
			// Retired route (plan 056): /models folded into /ui/show/settings?section=models.
			Name:           "GET /models is retired (302)",
			Method:         "GET",
			URL:            "/models",
			ExpectedStatus: 302,
		},
		{
			// Retired route (plan 053): the skills manager lives in the skills
			// The old /skills 302s to /.
			Name:           "GET /skills is retired (302)",
			Method:         "GET",
			URL:            "/skills",
			ExpectedStatus: 302,
		},
		{
			// Retired route (plan 053): the memory manager moved into the memory
			// /ui/show/memory. The old /memory page 302s to /.
			Name:           "GET /memory is retired (302)",
			Method:         "GET",
			URL:            "/memory",
			ExpectedStatus: 302,
		},
		{
			// Retired route (plan 055): the life overview moved into the lifelog
			// /ui/show/lifelog. The old /life page 302s to /.
			Name:           "GET /life is retired (302)",
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
			// Home is the single-page chat shell: the exact root renders the
			// shell with the "app" class, the domain sidebar, and the dock.
			Name:            "/ renders the single-page chat shell",
			Method:          "GET",
			URL:             "/",
			ExpectedStatus:  200,
			ExpectedContent: []string{`<html lang="en" class="app`, `class="app-shell"`, `id="chat"`},
		},
	}

	for _, scenario := range scenarios {
		scenario.TestAppFactory = newWebApp
		scenario.Test(t)
	}
}

func TestChatHandler(t *testing.T) {
	// Each ApiScenario.Test re-fires OnServe, causing route registration
	// conflicts when sharing one app. Give each scenario its own app with
	// the model already seeded so the factory is a plain getter.
	newChatApp := func(tb testing.TB) *tests.TestApp {
		app := newWebApp(tb)
		seedScriptedModel(tb, app, llmtest.Text("Hello from the fake model"))
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
			ExpectedContent: []string{"datastar-patch-elements", "Hello from the fake model", "cmsg cmsg-user", "cmsg cmsg-balaur"},
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

// seedHeadRec creates a custom head record for web handler tests.
func seedHeadRec(t testing.TB, app *tests.TestApp, name, _ string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		t.Fatalf("heads collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	if err := app.Save(rec); err != nil {
		t.Fatalf("saving head: %v", err)
	}
	return rec
}

// TestHeadsFocus: the heads card is served as an artifact via /ui/show/heads.
func TestHeadsFocus(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name:   "/ui/show/heads injects heads artifact with active head",
			Method: "GET",
			URL:    "/ui/show/heads",
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				app := newWebApp(tb)
				seedHeadRec(tb, app, "Scout", "active")
				return app
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"Scout", "ucard-heads"},
		},
	}
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestRetiredHeadsPages: plan 054 deleted the GET /heads and
// GET /heads/{id}/chat page routes. The roster now lives under Settings → Heads
// (/ui/show/settings?section=heads). The old page routes no longer exist;
// unmatched UI paths fall through to the catch-all redirect (302 → /), so
// old bookmarks self-heal onto the home.
func TestRetiredHeadsPages(t *testing.T) {
	t.Run("GET /heads is retired", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:           "GET /heads retired (302)",
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
			Name:           "GET /heads/{id}/chat retired (302)",
			Method:         "GET",
			URL:            "/heads/" + head.Id + "/chat",
			TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus: 302,
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
		Name:            "GET / carries hardening headers",
		Method:          "GET",
		URL:             "/",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  200,
		ExpectedContent: []string{"app-shell"},
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
		URL:             "/",
		Headers:         map[string]string{"Host": "localhost"},
		ExpectedStatus:  200,
		ExpectedContent: []string{"app-shell"},
	}
	scenario.TestAppFactory = newWebApp
	scenario.Test(t)
}

func TestChatCardShow(t *testing.T) {
	newCardShowApp := func(tb testing.TB) *tests.TestApp {
		app := newWebApp(tb)
		seedScriptedModel(tb, app,
			llmtest.ToolCall("tc1", "card_show", `{"type":"today"}`),
			llmtest.Text("Here is your today card."),
		)
		return app
	}

	t.Run("streamed card_show morphs panel and appends chip", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:           "chat card_show panel+chip",
			Method:         "POST",
			URL:            "/ui/chat",
			Body:           strings.NewReader("message=show+me+today"),
			Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			TestAppFactory: newCardShowApp,
			ExpectedStatus: 200,
			// The card is server-rendered in the right panel; a re-open chip goes into #chat.
			ExpectedContent: []string{`id="panel-inner"`, `art-chip`, `id="ucard-today"`},
		}
		scenario.Test(t)
	})
}

// TestUICardHistoryRendersChip verifies that a uicard tool result loaded from
// history renders as a re-open chip (not an inline card body). The chip is the
// durable transcript trace; the artifact lives in the right panel (plan 098).
func TestUICardHistoryRendersChip(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	// Simulate what messageViews produces for a uicard-marked tool result.
	marked := tools.MarkUICard("today", map[string]string{}, "showing the owner the Today card")
	typ, query, rest, ok := tools.ParseUICard(marked)
	if !ok {
		t.Fatal("ParseUICard: ok=false on well-formed marked text")
	}

	// messageViews records coordinates, not body — replicate that here.
	mv := messageView{
		Role:          "tool",
		Tool:          "card_show",
		Content:       rest,
		ArtifactType:  typ,
		ArtifactQuery: query,
		ArtifactTitle: "Today",
	}

	// renderMessages is a *handlers method; use it directly.
	out := renderNodeHTML(h.renderMessages([]messageView{mv}))
	if !strings.Contains(out, `art-chip`) {
		t.Errorf("history uicard render: missing art-chip. output:\n%s", out)
	}
	if !strings.Contains(out, `/ui/show/today`) {
		t.Errorf("history uicard render: chip missing re-open URL. output:\n%s", out)
	}
	if strings.Contains(out, `k-inline`) {
		t.Errorf("history uicard render: must not contain k-inline (card is in the panel). output:\n%s", out)
	}
	if strings.Contains(out, "hx-get") {
		t.Errorf("history uicard render: stale lazy hx-get mount present. output:\n%s", out)
	}
}

func TestDayCardArtifact(t *testing.T) {
	// GET /ui/show/day?date=... injects the full day view (ui.Focus) into chat.
	scenario := tests.ApiScenario{
		Name:               "GET /ui/show/day?date=2026-01-15 injects day card artifact",
		Method:             "GET",
		URL:                "/ui/show/day?date=2026-01-15",
		TestAppFactory:     newWebApp,
		ExpectedStatus:     200,
		ExpectedContent:    []string{"day-focus", "January"},
		NotExpectedContent: []string{"day-nav"},
	}
	scenario.Test(t)
}

// TestSelectModel covers the POST /ui/model/select handler for the three
// key paths: missing key, unknown key, and valid key.
func TestSelectModel(t *testing.T) {
	// Seed a local model with an injected fake client so there is at least one
	// selectable (non-disabled) choice available.
	newModelApp := func(tb testing.TB) (*tests.TestApp, string) {
		tb.Helper()
		app := newWebApp(tb)
		return app, seedScriptedModel(tb, app)
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
		// The injected fake client marks the local model ready, so it is not Disabled.
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

// TestDownloadOfficialModel verifies the download handler calls SaveLocalModel +
// SetActiveLLMModel via a tiny fake GGUF served by an httptest server.
// It uses a seam to override the URL/sha256 injected into modelget.Fetch
// so the test is fully offline — no real network, no real model pin.
func TestDownloadOfficialModel(t *testing.T) {
	// Build a tiny fake GGUF and compute its sha256.
	fakeContent := []byte("GGUF\x03\x00\x00\x00" + strings.Repeat("fake", 32))
	h := sha256.New()
	h.Write(fakeContent)
	fakeHash := hex.EncodeToString(h.Sum(nil))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(fakeContent)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeContent)
	}))
	t.Cleanup(srv.Close)

	// Override the curated pin lookup to point at our httptest server (any key).
	origOfficial := kronkOfficialByKey
	kronkOfficialByKey = func(string) (kronk.OfficialModel, bool) {
		return kronk.OfficialModel{
			Key:       "medium",
			Name:      "Test Model",
			URL:       srv.URL + "/model.gguf",
			SHA256:    fakeHash,
			SizeBytes: int64(len(fakeContent)),
			FileName:  "test-model.gguf",
		}, true
	}
	t.Cleanup(func() { kronkOfficialByKey = origOfficial })

	// Set models dir to a temp dir.
	modelsDir := t.TempDir()
	t.Setenv("BALAUR_MODELS_DIR", modelsDir)

	app := newWebApp(t)

	scenario := tests.ApiScenario{
		Name:            "download official model success",
		Method:          "POST",
		URL:             "/ui/model/download",
		Body:            strings.NewReader(""),
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"models-panel"},
		AfterTestFunc: func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
			// The downloaded file must exist.
			finalPath := filepath.Join(modelsDir, "test-model.gguf")
			if _, statErr := os.Stat(finalPath); statErr != nil {
				tb.Errorf("downloaded file missing: %v", statErr)
			}
			// The model must now be active.
			_, active, err := turn.ModelChoices(a)
			if err != nil {
				tb.Fatalf("ModelChoices: %v", err)
			}
			if active.Key == "" {
				tb.Error("no active model after download; expected test-model.gguf to be active")
			}
		},
	}
	scenario.Test(t)
}

// TestCloudModelConsentFlow exercises the opt-in cloud path: saving a cloud model
// does NOT activate it; the first selection returns the consent dialog (nothing
// active); confirming activates it with a cloud badge. The API key must never
// appear in any rendered response. One ApiScenario per app — the harness tears an
// app's DB down after .Test(), so prerequisites are seeded via store beforehand.
func TestCloudModelConsentFlow(t *testing.T) {
	const leakKey = "sk-LEAKTEST-must-not-appear"
	hdr := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	body := func(v url.Values) *strings.Reader { return strings.NewReader(v.Encode()) }

	t.Run("save does not activate and never leaks the key", func(t *testing.T) {
		app := newWebApp(t)
		scenario := tests.ApiScenario{
			Name:   "POST /ui/model/cloud",
			Method: "POST",
			URL:    "/ui/model/cloud",
			Body: body(url.Values{
				"name": {"OpenAI"}, "base_url": {"https://api.openai.com/v1"},
				"chat_model": {"gpt-4o"}, "label": {"GPT-4o"},
				"api_key": {leakKey}, "consent": {"1"},
			}),
			Headers:            hdr,
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			NotExpectedContent: []string{leakKey},
			AfterTestFunc: func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
				_, active, err := turn.ModelChoices(a)
				if err != nil {
					tb.Fatalf("choices: %v", err)
				}
				if active.Key != "" {
					tb.Fatalf("cloud model must not auto-activate, active=%q", active.Key)
				}
			},
		}
		scenario.Test(t)
	})

	t.Run("save requires the consent checkbox", func(t *testing.T) {
		app := newWebApp(t)
		scenario := tests.ApiScenario{
			Name:   "POST /ui/model/cloud without consent",
			Method: "POST",
			URL:    "/ui/model/cloud",
			Body: body(url.Values{
				"name": {"OpenAI"}, "base_url": {"https://api.openai.com/v1"},
				"chat_model": {"gpt-4o"}, "label": {"GPT-4o"}, "api_key": {leakKey},
			}),
			Headers:            hdr,
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			ExpectedContent:    []string{"confirm you understand"},
			NotExpectedContent: []string{leakKey},
			AfterTestFunc: func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
				models, _, err := turn.ModelChoices(a)
				if err == nil && len(models) != 0 {
					tb.Fatalf("no model should be saved without consent, got %d", len(models))
				}
			},
		}
		scenario.Test(t)
	})

	t.Run("first selection requires consent; nothing activates", func(t *testing.T) {
		app := newWebApp(t)
		id, err := store.SaveCloudModel(app, "OpenAI", "https://api.openai.com/v1", leakKey, "GPT-4o", "gpt-4o", "")
		if err != nil {
			t.Fatalf("seed cloud model: %v", err)
		}
		scenario := tests.ApiScenario{
			Name:               "POST /ui/model/select cloud first time",
			Method:             "POST",
			URL:                "/ui/model/select",
			Body:               body(url.Values{"key": {id}, "target": {"models"}}),
			Headers:            hdr,
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			ExpectedContent:    []string{"Keep local", "Yes, use GPT-4o"},
			NotExpectedContent: []string{leakKey},
			AfterTestFunc: func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
				_, active, _ := turn.ModelChoices(a)
				if active.Key != "" {
					tb.Fatalf("consent dialog must not activate, active=%q", active.Key)
				}
			},
		}
		scenario.Test(t)
	})

	t.Run("confirm activates with cloud badge", func(t *testing.T) {
		app := newWebApp(t)
		id, err := store.SaveCloudModel(app, "OpenAI", "https://api.openai.com/v1", leakKey, "GPT-4o", "gpt-4o", "")
		if err != nil {
			t.Fatalf("seed cloud model: %v", err)
		}
		scenario := tests.ApiScenario{
			Name:               "POST /ui/model/cloud/confirm",
			Method:             "POST",
			URL:                "/ui/model/cloud/confirm",
			Body:               body(url.Values{"key": {id}, "consent": {"1"}}),
			Headers:            hdr,
			TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
			ExpectedStatus:     200,
			NotExpectedContent: []string{leakKey},
			AfterTestFunc: func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
				_, active, err := turn.ModelChoices(a)
				if err != nil {
					tb.Fatalf("choices: %v", err)
				}
				if active.Key != id {
					tb.Fatalf("active key = %q, want %q", active.Key, id)
				}
				if active.Badge != "cloud" {
					tb.Errorf("active badge = %q, want cloud", active.Badge)
				}
			},
		}
		scenario.Test(t)
	})
}

// TestKnowledgeCardErrorSanitized verifies that fetching a nonexistent knowledge
// card via GET /ui/knowledge/{kind}/{id}/card renders the generic "could not load
// this card" message and does NOT expose the raw PocketBase error text.
func TestKnowledgeCardErrorSanitized(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:               "GET /ui/knowledge/memories/nonexistent/card returns generic error",
		Method:             "GET",
		URL:                "/ui/knowledge/memories/nonexistent/card",
		TestAppFactory:     newWebApp,
		ExpectedStatus:     422,
		ExpectedContent:    []string{"could not load this card"},
		NotExpectedContent: []string{"sql:", "no rows", "nonexistent"},
	}
	scenario.Test(t)
}

// TestDeleteCloudModelHandler verifies the POST /ui/model/cloud/delete handler
// removes a non-active cloud model and re-renders the models panel.
func TestDeleteCloudModelHandler(t *testing.T) {
	hdr := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	app := newWebApp(t)
	id, err := store.SaveCloudModel(app, "OpenAI", "https://api.openai.com/v1", "sk-x", "GPT-4o", "gpt-4o", "")
	if err != nil {
		t.Fatalf("seed cloud model: %v", err)
	}
	scenario := tests.ApiScenario{
		Name:            "POST /ui/model/cloud/delete removes non-active model",
		Method:          "POST",
		URL:             "/ui/model/cloud/delete",
		Body:            strings.NewReader(url.Values{"key": {id}}.Encode()),
		Headers:         hdr,
		TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"models-panel"},
		AfterTestFunc: func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
			_, err := a.FindRecordById("llm_models", id)
			if err == nil {
				tb.Fatalf("model %q should have been deleted but still exists", id)
			}
		},
	}
	scenario.Test(t)
}
