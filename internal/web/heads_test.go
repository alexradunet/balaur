package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/heads"
	_ "github.com/alexradunet/balaur/migrations"
)

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
