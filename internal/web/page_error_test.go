package web

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

// TestUIShowUnknownTypeReturns404: GET /ui/show/<unknown> returns 404 for unknown
// card types — the endpoint is an SSE fragment, so the status is the signal.
func TestUIShowUnknownTypeReturns404(t *testing.T) {
	s := tests.ApiScenario{
		Name:            "GET /ui/show/<unknown> returns 404",
		Method:          "GET",
		URL:             "/ui/show/__nope__",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  404,
		ExpectedContent: []string{"404"},
	}
	s.Test(t)
}
