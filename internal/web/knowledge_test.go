package web

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/knowledge"
)

// seedProposedMemory creates a proposed memory record for the Datastar path tests.
func seedProposedMemory(t testing.TB, app *tests.TestApp, title string) string {
	t.Helper()
	rec, err := knowledge.ProposeMemory(app, knowledge.MemoryProposal{
		Title:      title,
		Content:    "seed body",
		Category:   "fact",
		Importance: 3,
	})
	if err != nil {
		t.Fatalf("propose memory: %v", err)
	}
	return rec.Id
}

// sseHeaders marks a request as a Datastar @post fetch (Accept: text/event-stream)
// carrying a form body, matching the card forms' {contentType:'form'} posts.
var sseHeaders = map[string]string{
	"Content-Type": "application/x-www-form-urlencoded",
	"Accept":       "text/event-stream",
}

// TestKnowledgeTransitionApproveDatastar: approve (to=active) re-renders the
// card in place via an outer-mode patch targeting #kcard-{id}.
func TestKnowledgeTransitionApproveDatastar(t *testing.T) {
	app := newWebApp(t)
	id := seedProposedMemory(t, app, "Approve me")

	scenario := tests.ApiScenario{
		Name:           "approve memory emits an outer patch",
		Method:         "POST",
		URL:            "/ui/knowledge/memories/" + id + "/transition",
		Body:           strings.NewReader("to=active"),
		Headers:        sseHeaders,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #kcard-" + id,
			"Approve me",
			`kcard-active`,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
	}
	scenario.Test(t)
}

// TestKnowledgeTransitionRejectDatastar: dismiss (to=rejected) removes the card
// — the SSE carries a remove-mode patch for #kcard-{id}.
func TestKnowledgeTransitionRejectDatastar(t *testing.T) {
	app := newWebApp(t)
	id := seedProposedMemory(t, app, "Dismiss me")

	scenario := tests.ApiScenario{
		Name:           "dismiss memory emits a remove patch",
		Method:         "POST",
		URL:            "/ui/knowledge/memories/" + id + "/transition",
		Body:           strings.NewReader("to=rejected"),
		Headers:        sseHeaders,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #kcard-" + id,
			"mode remove",
		},
		// The card markup must NOT be streamed back on a removal.
		NotExpectedContent: []string{"Dismiss me", "kcard-title"},
		TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
	}
	scenario.Test(t)
}

// TestKnowledgeEditDatastar: the edit form posts named fields and the patch
// carries the new title (proving {contentType:'form'} reads + outer patch).
func TestKnowledgeEditDatastar(t *testing.T) {
	app := newWebApp(t)
	id := seedProposedMemory(t, app, "Old title")

	scenario := tests.ApiScenario{
		Name:           "edit memory patches the card with the new title",
		Method:         "POST",
		URL:            "/ui/knowledge/memories/" + id + "/edit",
		Body:           strings.NewReader("title=Fresh+title&importance=4"),
		Headers:        sseHeaders,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #kcard-" + id,
			"Fresh title",
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
	}
	scenario.Test(t)
}

// TestKnowledgeTransitionInvalidIsHTTPError: a bad transition fails BEFORE any
// SSE is opened, returning a normal 422 card-error fragment (no SSE patch).
func TestKnowledgeTransitionInvalidIsHTTPError(t *testing.T) {
	app := newWebApp(t)
	id := seedProposedMemory(t, app, "Bad move")

	scenario := tests.ApiScenario{
		Name:               "illegal transition returns a card error, not an SSE patch",
		Method:             "POST",
		URL:                "/ui/knowledge/memories/" + id + "/transition",
		Body:               strings.NewReader("to=archived"), // proposed→archived is invalid
		Headers:            sseHeaders,
		ExpectedStatus:     422,
		ExpectedContent:    []string{"card-note-error"},
		NotExpectedContent: []string{"datastar-patch-elements"},
		TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
	}
	scenario.Test(t)
}
