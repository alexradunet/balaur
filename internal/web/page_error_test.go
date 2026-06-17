package web

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

// TestFocusUnknownTypeRendersShellError: a direct GET to an unknown card type
// renders the Hearthwood shell error page (HTML, status 404), not the raw
// PocketBase JSON ApiError.
func TestFocusUnknownTypeRendersShellError(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/<unknown> renders the shell error page, not JSON",
		Method:         "GET",
		URL:            "/focus/__nope__",
		TestAppFactory: newWebApp,
		ExpectedStatus: 404,
		ExpectedContent: []string{
			"<!doctype html>",
			`class="empty"`, // the EmptyState body rendered in the shell
			`href="/"`,      // Back home link
		},
		NotExpectedContent: []string{
			`"status":404`, // not the raw JSON ApiError
		},
	}
	s.Test(t)
}
