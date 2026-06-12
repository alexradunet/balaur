package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
	"github.com/alexradunet/balaur/internal/store"
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
	app := newWebApp(t)
	sseSrv := newFakeSSEServer("Hello from the fake model")
	t.Cleanup(func() { sseSrv.Close() })

	id, _ := store.SaveOpenAIModel(app, "fake", sseSrv.URL, "", "Fake", "fake-model", "", false)
	store.SetActiveLLMModel(app, id, "test")

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
			Name:            "chat with client_rendered=1",
			Method:          "POST",
			URL:             "/ui/chat",
			Body:            strings.NewReader("message=hello&client_rendered=1"),
			Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			ExpectedStatus:  200,
			ExpectedContent: []string{"Hello from the fake model"},
			NotExpectedContent: []string{"msg msg-user"},
		},
		{
			Name:           "chat empty message",
			Method:         "POST",
			URL:            "/ui/chat",
			Body:           strings.NewReader("message="),
			Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			ExpectedStatus: 400,
		},
	}

	for _, scenario := range scenarios {
		scenario.TestAppFactory = func(tb testing.TB) *tests.TestApp { return app }
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
		Name:           "transition task to done",
		Method:         "POST",
		URL:            "/ui/tasks/" + rec.Id + "/transition",
		Body:           strings.NewReader("action=done"),
		Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		ExpectedStatus: 200,
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
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
		ExpectedContent: []string{"chatbar"},
	}
	scenario.TestAppFactory = newWebApp
	scenario.Test(t)
}
