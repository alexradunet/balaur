package web

import (
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tasks"
)

// nudge.go — owner controls for the task nudger: enable/disable, mute for a
// window, or fire one immediately. These write owner_settings (the soft layer
// above the BALAUR_NUDGE env kill switch) and re-render the settings nudge
// section. "Nudge now" bypasses the mute — it is an explicit owner action.

// renderNudgeSection re-renders the nudge controls fragment in place.
func (h *handlers) renderNudgeSection(e *core.RequestEvent) error {
	var b strings.Builder
	if err := settingscards.NudgeSection(settingscards.BuildNudge(h.app)).Render(&b); err != nil {
		return e.InternalServerError("rendering nudge section", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	patchOuterHTML(sse, "nudge-section", b.String())
	return nil
}

// nudgeToggle flips the nudge_enabled owner setting.
func (h *handlers) nudgeToggle(e *core.RequestEvent) error {
	next := "0"
	if store.GetOwnerSetting(h.app, "nudge_enabled", "1") == "0" {
		next = "1"
	}
	if err := store.SetOwnerSetting(h.app, "nudge_enabled", next); err != nil {
		return e.InternalServerError("saving nudge_enabled", err)
	}
	return h.renderNudgeSection(e)
}

// nudgeMute silences nudges for the given number of hours.
func (h *handlers) nudgeMute(e *core.RequestEvent) error {
	hours, _ := strconv.Atoi(e.Request.FormValue("hours"))
	if hours <= 0 {
		hours = 1
	}
	until := time.Now().Add(time.Duration(hours) * time.Hour).Format(time.RFC3339)
	if err := store.SetOwnerSetting(h.app, "nudge_muted_until", until); err != nil {
		return e.InternalServerError("saving nudge_muted_until", err)
	}
	return h.renderNudgeSection(e)
}

// nudgeNow fires one nudge immediately, bypassing the mute (explicit owner
// action). The client is nil → the deterministic line ships (offline-safe).
func (h *handlers) nudgeNow(e *core.RequestEvent) error {
	if err := tasks.Nudge(h.app, nil, time.Now()); err != nil {
		return h.cardError(e, err)
	}
	return h.renderNudgeSection(e)
}
