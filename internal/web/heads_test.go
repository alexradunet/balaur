package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/heads"
	_ "github.com/alexradunet/balaur/migrations"
)

// buildMux triggers OnServe once and returns the fully built router mux. The
// OnServe hooks register routes and so may run only once per app; reuse the
// returned mux for every request against this app (tests.ApiScenario, by
// contrast, serves an app once and tears it down).
func buildMux(t *testing.T, app *tests.TestApp) http.Handler {
	t.Helper()
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
	return mux
}

// servePost drives a single form POST against an already-built mux.
func servePost(mux http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestSetActiveHeadSwitches(t *testing.T) {
	app := newWebApp(t)
	s := tests.ApiScenario{
		Name:           "POST /ui/heads/active switches to scholar",
		Method:         "POST",
		URL:            "/ui/heads/active",
		Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:           strings.NewReader("head=scholar"),
		TestAppFactory: func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #head-switcher",
			"Scholar",
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			if heads.Active(app).ID != "scholar" {
				t.Errorf("active head = %q, want scholar", heads.Active(app).ID)
			}
		},
	}
	s.Test(t)
}

func TestSetActiveHeadRejectsUnknown(t *testing.T) {
	app := newWebApp(t)
	s := tests.ApiScenario{
		Name:            "unknown head is 400",
		Method:          "POST",
		URL:             "/ui/heads/active",
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:            strings.NewReader("head=nope"),
		TestAppFactory:  func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  400,
		ExpectedContent: []string{"Unknown head"},
	}
	s.Test(t)
}

func TestCreateAndDeleteCustomHead(t *testing.T) {
	// One app drives both requests: create then delete. tests.ApiScenario can
	// only serve an app once, so the lifecycle runs through servePost instead.
	app := newWebApp(t)
	defer app.Cleanup()
	mux := buildMux(t, app)

	create := servePost(mux, "/ui/heads/new",
		"name=Scribe&purpose=edits+prose&balaur_avatar=balaur-07&tools=journal")
	if create.Code != 200 {
		t.Fatalf("create status = %d, want 200", create.Code)
	}
	for _, want := range []string{"datastar-patch-elements", "selector #ucard-heads", "Scribe"} {
		if !strings.Contains(create.Body.String(), want) {
			t.Errorf("create body missing %q\n%s", want, create.Body.String())
		}
	}

	// The custom head exists with its group.
	var id string
	for _, hd := range heads.List(app) {
		if hd.Name == "Scribe" {
			id = hd.ID
			if len(hd.Groups) != 1 || hd.Groups[0] != "journal" {
				t.Fatalf("Scribe groups = %v, want [journal]", hd.Groups)
			}
		}
	}
	if id == "" {
		t.Fatal("custom head Scribe not found after create")
	}

	del := servePost(mux, "/ui/heads/"+id+"/delete", "")
	if del.Code != 200 {
		t.Fatalf("delete status = %d, want 200", del.Code)
	}
	if !strings.Contains(del.Body.String(), "selector #ucard-heads") {
		t.Errorf("delete body missing %q\n%s", "selector #ucard-heads", del.Body.String())
	}

	for _, hd := range heads.List(app) {
		if hd.ID == id {
			t.Error("custom head should be deleted")
		}
	}
}

func TestDeleteBuiltinRejected(t *testing.T) {
	app := newWebApp(t)
	s := tests.ApiScenario{
		Name:            "built-in heads cannot be deleted",
		Method:          "POST",
		URL:             "/ui/heads/scholar/delete",
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory:  func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  400,
		ExpectedContent: []string{"Cannot delete"},
	}
	s.Test(t)
}
