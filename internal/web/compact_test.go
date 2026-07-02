package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/conversation"
)

// TestCompactAcceptPinsBoundaryToDraftedThrough is the gateway round-trip for
// the plan-241 fix: POST /ui/compact/accept must pin compacted_through to the
// posted compactDraftedThrough signal, not to the accept-click time.
func TestCompactAcceptPinsBoundaryToDraftedThrough(t *testing.T) {
	app := newWebApp(t)
	if _, err := conversation.Master(app); err != nil {
		t.Fatalf("master: %v", err)
	}

	t1 := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	body := `{"compactDraft":"folded by test","compactDraftedThrough":"` + t1.Format(time.RFC3339Nano) + `"}`

	s := tests.ApiScenario{
		Name:           "POST /ui/compact/accept pins boundary to drafted-through",
		Method:         "POST",
		URL:            "/ui/compact/accept",
		Headers:        map[string]string{"Content-Type": "application/json"},
		Body:           strings.NewReader(body),
		TestAppFactory: func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"folded by test",
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			master, err := conversation.Master(app)
			if err != nil {
				t.Fatalf("master: %v", err)
			}
			if boundary := conversation.CompactedThrough(master); !boundary.Equal(t1) {
				t.Errorf("compacted_through = %v, want %v", boundary, t1)
			}
			if got := master.GetString("summary"); !strings.Contains(got, "folded by test") {
				t.Errorf("summary = %q, want it to contain the folded text", got)
			}
		},
	}
	s.Test(t)
}

// TestCompactAcceptRejectsMissingDraftedThrough covers the guard added in
// plan 241: a commit without a parseable drafted-through timestamp must not
// write anything, and the modal must show a visible error instead.
func TestCompactAcceptRejectsMissingDraftedThrough(t *testing.T) {
	app := newWebApp(t)
	if _, err := conversation.Master(app); err != nil {
		t.Fatalf("master: %v", err)
	}

	body := `{"compactDraft":"folded by test"}`

	s := tests.ApiScenario{
		Name:            "POST /ui/compact/accept rejects a missing drafted-through timestamp",
		Method:          "POST",
		URL:             "/ui/compact/accept",
		Headers:         map[string]string{"Content-Type": "application/json"},
		Body:            strings.NewReader(body),
		TestAppFactory:  func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus:  200,
		ExpectedContent: []string{"stale or incomplete"},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			master, err := conversation.Master(app)
			if err != nil {
				t.Fatalf("master: %v", err)
			}
			if boundary := conversation.CompactedThrough(master); !boundary.IsZero() {
				t.Errorf("compacted_through = %v, want zero (nothing written)", boundary)
			}
			if got := master.GetString("summary"); got != "" {
				t.Errorf("summary = %q, want empty (nothing written)", got)
			}
		},
	}
	s.Test(t)
}
