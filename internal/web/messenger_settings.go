package web

import (
	"strings"

	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"
)

// saveMessengerToken handles POST /ui/settings/messenger-token — persists the
// messenger gateway token via owner settings and re-renders the gateway control
// fragment. Empty value clears the token, disabling the endpoint (matches the
// fail-closed check in messenger.go). The token is never logged.
func (h *handlers) saveMessengerToken(e *core.RequestEvent) error {
	token := strings.TrimSpace(e.Request.FormValue("messenger_token"))
	// Never log the token value — it is a secret.
	if err := store.SetOwnerSetting(h.app, "messenger_token", token); err != nil {
		return e.InternalServerError("saving messenger token", err)
	}
	// Audit the state transition only — never the token value.
	state := "set"
	if token == "" {
		state = "cleared"
	}
	store.Audit(h.app, "owner", "messenger.token", "owner_settings/messenger_token", true, map[string]any{"state": state})
	view := settingscards.BuildCapabilities(h.app)
	var b strings.Builder
	if err := settingscards.MessengerGatewaySection(view).Render(&b); err != nil {
		return e.InternalServerError("rendering messenger gateway section", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	patchOuterHTML(sse, "messenger-gateway-section", b.String())
	return nil
}
