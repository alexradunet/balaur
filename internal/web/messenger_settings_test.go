package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/store"
)

// TestSaveMessengerTokenSetsAndPatches: posting a token persists it and
// re-renders the gateway control via an outer-mode SSE patch targeting
// #messenger-gateway-section. The re-rendered section reflects the new value
// (the stored token appears in the input's value attribute), proving persistence.
func TestSaveMessengerTokenSetsAndPatches(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "save messenger token emits an outer patch for #messenger-gateway-section",
		Method:         "POST",
		URL:            "/ui/settings/messenger-token",
		Body:           strings.NewReader("messenger_token=secret42"),
		Headers:        sseHeaders,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #messenger-gateway-section",
			"messenger-gateway-section",
			// The re-render reflects the stored value — proves the handler persisted it.
			`value="secret42"`,
		},
		TestAppFactory: newWebApp,
		AfterTestFunc: func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
			recs, err := a.FindRecordsByFilter("audit_log", "action = 'messenger.token'", "", 0, 0)
			if err != nil || len(recs) != 1 {
				tb.Fatalf("expected exactly 1 messenger.token audit row, got %d (err %v)", len(recs), err)
			}
			rec := recs[0]
			if rec.GetString("actor") != "owner" {
				tb.Errorf("actor = %q, want %q", rec.GetString("actor"), "owner")
			}
			if rec.GetString("target") != "owner_settings/messenger_token" {
				tb.Errorf("target = %q, want %q", rec.GetString("target"), "owner_settings/messenger_token")
			}
			if !rec.GetBool("allowed") {
				tb.Errorf("allowed = false, want true")
			}
			raw, err := json.Marshal(rec)
			if err != nil {
				tb.Fatalf("marshal audit record: %v", err)
			}
			if !strings.Contains(string(raw), `"state":"set"`) {
				tb.Errorf("audit record missing state:set; got: %s", raw)
			}
			if strings.Contains(string(raw), "secret42") {
				tb.Errorf("audit record must never contain the token value; got: %s", raw)
			}
		},
	}
	scenario.Test(t)
}

// TestSaveMessengerTokenClearsOnEmpty: posting an empty value clears the stored
// token (disables the endpoint). The re-rendered section shows "disabled" status.
func TestSaveMessengerTokenClearsOnEmpty(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "empty token clears messenger_token (disables endpoint, shows disabled status)",
		Method:         "POST",
		URL:            "/ui/settings/messenger-token",
		Body:           strings.NewReader("messenger_token="),
		Headers:        sseHeaders,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			"selector #messenger-gateway-section",
			"disabled",
		},
		TestAppFactory: newWebApp,
		AfterTestFunc: func(tb testing.TB, a *tests.TestApp, _ *http.Response) {
			recs, err := a.FindRecordsByFilter("audit_log", "action = 'messenger.token'", "", 0, 0)
			if err != nil || len(recs) != 1 {
				tb.Fatalf("expected exactly 1 messenger.token audit row, got %d (err %v)", len(recs), err)
			}
			raw, err := json.Marshal(recs[0])
			if err != nil {
				tb.Fatalf("marshal audit record: %v", err)
			}
			if !strings.Contains(string(raw), `"state":"cleared"`) {
				tb.Errorf("audit record missing state:cleared; got: %s", raw)
			}
		},
	}
	scenario.Test(t)
}

// TestSaveMessengerTokenWithExistingSeeded: a pre-seeded token is replaced by
// posting a new one. The re-rendered section no longer contains the old value.
func TestSaveMessengerTokenWithExistingSeeded(t *testing.T) {
	app := newWebApp(t)
	// Seed an existing token before the request.
	if err := store.SetOwnerSetting(app, "messenger_token", "oldtoken"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	scenario := tests.ApiScenario{
		Name:           "new token replaces the previous one",
		Method:         "POST",
		URL:            "/ui/settings/messenger-token",
		Body:           strings.NewReader("messenger_token=newtoken"),
		Headers:        sseHeaders,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"datastar-patch-elements",
			`value="newtoken"`,
		},
		// oldtoken must not appear in the re-rendered output (it was replaced)
		NotExpectedContent: []string{"oldtoken"},
		TestAppFactory:     func(tb testing.TB) *tests.TestApp { return app },
	}
	scenario.Test(t)
}
