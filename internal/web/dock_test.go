package web

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/alexradunet/balaur/migrations"
)

// TestDockConversationMaster: with no head, the swap patches #dock-convo with
// the master draft (@post('/ui/chat')) and shows no back-to-main affordance.
func TestDockConversationMaster(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /ui/dock/conversation → master draft",
		Method:         "GET",
		URL:            "/ui/dock/conversation",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #dock-convo",
			// SSE element payloads JSON-escape forward slashes (\/).
			`@post('\/ui\/chat')`,
			// The swap re-enables the master-only nudge poll.
			`"dockMaster":true`,
		},
		NotExpectedContent: []string{"back to main"},
	}
	s.Test(t)
}

// TestDockConversationBranch: with an active head, the swap patches #dock-convo
// with the branch draft pointing at the head's turn endpoint, the back-to-main
// header, and the head's name.
func TestDockConversationBranch(t *testing.T) {
	app := newWebApp(t)
	head := seedHeadRec(t, app, "Scribe", "active")
	s := tests.ApiScenario{
		Name:           "GET /ui/dock/conversation?head=… → branch draft",
		Method:         "GET",
		URL:            "/ui/dock/conversation?head=" + head.Id,
		TestAppFactory: func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #dock-convo",
			// SSE element payloads JSON-escape forward slashes (\/).
			`@post('\/ui\/heads\/` + head.Id + `\/chat')`,
			"back to main",
			"Scribe",
			// The swap silences the master-only nudge poll on a branch.
			`"dockMaster":false`,
			// No model is configured in the test app, so the greeting must
			// surface the model-unavailable error — not a blank <p></p>.
			"no active model is available",
		},
		// The branch greeting must render the model error, never an empty
		// paragraph (the bug this fix closes).
		NotExpectedContent: []string{"<p></p>"},
	}
	s.Test(t)
}

// TestDockConversationMergedForbidden: a non-active head cannot be opened in
// the dock.
func TestDockConversationMergedForbidden(t *testing.T) {
	app := newWebApp(t)
	head := seedHeadRec(t, app, "OldHead", "merged")
	s := tests.ApiScenario{
		Name:            "merged head is forbidden",
		Method:          "GET",
		URL:             "/ui/dock/conversation?head=" + head.Id,
		TestAppFactory:  func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  403,
		ExpectedContent: []string{"not active"},
	}
	s.Test(t)
}

// TestDockConversationUnknownHead: an unknown head id is 404.
func TestDockConversationUnknownHead(t *testing.T) {
	s := tests.ApiScenario{
		Name:            "unknown head id is 404",
		Method:          "GET",
		URL:             "/ui/dock/conversation?head=nope",
		TestAppFactory:  newWebApp,
		ExpectedStatus:  404,
		ExpectedContent: []string{"not found"},
	}
	s.Test(t)
}
