package web

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/store"
)

// TestSaveNameDatastar: posting a display name re-renders the identity card in
// place via an outer-mode patch targeting #identity-card. {contentType:'form'}
// is proven by the saved name flowing back into the fragment.
func TestSaveNameDatastar(t *testing.T) {
	app := newWebApp(t)

	scenario := tests.ApiScenario{
		Name:           "save name emits an outer patch for the identity card",
		Method:         "POST",
		URL:            "/ui/profile/name",
		Body:           strings.NewReader("display_name=Mira"),
		Headers:        sseHeaders,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #identity-card",
			"Mira",
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
	}
	scenario.Test(t)
}

// TestSetSoulAvatarDatastar: a valid soul-avatar key re-renders the soul
// section via an outer patch targeting #soul-section.
func TestSetSoulAvatarDatastar(t *testing.T) {
	if !store.ValidSoulAvatarKey("soul-02") {
		t.Fatal("expected soul-02 to be a valid avatar key")
	}
	app := newWebApp(t)

	scenario := tests.ApiScenario{
		Name:           "set soul avatar emits an outer patch for the soul section",
		Method:         "POST",
		URL:            "/ui/profile/soul-avatar",
		Body:           strings.NewReader("soul_avatar=soul-02"),
		Headers:        sseHeaders,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #soul-section",
			"avatar-choice-active",
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
	}
	scenario.Test(t)
}

// TestSetSoulAvatarInvalidIsHTTPError: a bogus key fails BEFORE any SSE is
// opened, returning a normal 400 (no patch).
func TestSetSoulAvatarInvalidIsHTTPError(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:               "bogus soul avatar key returns a 400, not an SSE patch",
		Method:             "POST",
		URL:                "/ui/profile/soul-avatar",
		Body:               strings.NewReader("soul_avatar=bogus"),
		Headers:            sseHeaders,
		ExpectedStatus:     400,
		ExpectedContent:    []string{"Invalid avatar"},
		NotExpectedContent: []string{"datastar-patch-elements"},
		TestAppFactory:     newWebApp,
	}
	scenario.Test(t)
}

// TestSetBalaurAvatarDatastar: a valid Balaur head key re-renders the Balaur
// section via an outer patch targeting #balaur-section.
func TestSetBalaurAvatarDatastar(t *testing.T) {
	if !store.ValidBalaurAvatarKey("balaur-02") {
		t.Fatal("expected balaur-02 to be a valid avatar key")
	}
	app := newWebApp(t)

	scenario := tests.ApiScenario{
		Name:           "set balaur avatar emits an outer patch for the balaur section",
		Method:         "POST",
		URL:            "/ui/profile/balaur-avatar",
		Body:           strings.NewReader("balaur_avatar=balaur-02"),
		Headers:        sseHeaders,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #balaur-section",
			"avatar-choice-active",
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
	}
	scenario.Test(t)
}

// TestSetBalaurAvatarInvalidIsHTTPError: a bogus key fails BEFORE any SSE is
// opened, returning a normal 400 (no patch).
func TestSetBalaurAvatarInvalidIsHTTPError(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:               "bogus balaur avatar key returns a 400, not an SSE patch",
		Method:             "POST",
		URL:                "/ui/profile/balaur-avatar",
		Body:               strings.NewReader("balaur_avatar=bogus"),
		Headers:            sseHeaders,
		ExpectedStatus:     400,
		ExpectedContent:    []string{"Invalid balaur avatar"},
		NotExpectedContent: []string{"datastar-patch-elements"},
		TestAppFactory:     newWebApp,
	}
	scenario.Test(t)
}
