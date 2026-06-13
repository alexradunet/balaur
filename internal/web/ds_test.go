package web

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

// TestPingPatchesOverSSE is the Datastar bootstrap gate (plan 050, Phase 0):
// GET /ui/ping must answer with a datastar-patch-elements SSE event carrying
// the #ping → "pong" patch. Proves NewSSE/PatchElements work end-to-end over
// the same e.Response writer the chat stream uses.
func TestPingPatchesOverSSE(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "GET /ui/ping streams a Datastar element patch",
		Method:         "GET",
		URL:            "/ui/ping",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"event: datastar-patch-elements",
			`<span id="ping">pong</span>`,
		},
	}
	scenario.Test(t)
}
