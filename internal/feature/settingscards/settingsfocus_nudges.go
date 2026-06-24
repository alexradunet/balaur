// settingsfocus_nudges.go — the Nudges settings section: on/off toggle, mute
// windows, and manual nudge. Split out of settingsfocus.go (plan 186).
package settingscards

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/store"
)

// NudgeView is the view-model for the nudge controls section.
type NudgeView struct {
	Enabled    bool   // the nudge_enabled owner setting (default on)
	MutedUntil string // human label of the active mute window end; empty if not muted
}

// BuildNudge reads the owner's nudge controls from owner_settings.
func BuildNudge(app core.App) NudgeView {
	v := NudgeView{Enabled: store.GetOwnerSetting(app, "nudge_enabled", "1") != "0"}
	if until := store.GetOwnerSetting(app, "nudge_muted_until", ""); until != "" {
		now := time.Now()
		if t, err := time.Parse(time.RFC3339, until); err == nil && now.Before(t) {
			v.MutedUntil = t.In(now.Location()).Format("Mon 15:04")
		}
	}
	return v
}

// NudgeSection renders the nudge controls (#nudge-section): on/off, mute
// windows, and a manual "nudge me now". Re-render target after the /ui/nudge/*
// handlers (outer patch #nudge-section).
func NudgeSection(v NudgeView) g.Node {
	post := func(url string) g.Node {
		return data.On("submit", "@post('"+url+"', {contentType:'form'})", data.ModifierPrevent)
	}
	status := "Nudges are on."
	if !v.Enabled {
		status = "Nudges are off."
	} else if v.MutedUntil != "" {
		status = "Muted until " + v.MutedUntil + "."
	}
	toggleLabel := "Turn off"
	if !v.Enabled {
		toggleLabel = "Turn on"
	}
	muteBtn := func(hours, label string) g.Node {
		return h.Form(post("/ui/nudge/mute"),
			h.Input(h.Type("hidden"), h.Name("hours"), h.Value(hours)),
			h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text(label)),
		)
	}
	return h.Article(h.Class("profile-card"), h.ID("nudge-section"),
		h.H2(h.Class("profile-card-title"), g.Text("Nudges")),
		h.P(h.Class("profile-hint"), g.Text("Reminders for due tasks, delivered as one chat message. "+status)),
		h.Div(h.Class("kcard-actions"),
			h.Form(post("/ui/nudge/toggle"),
				h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"), g.Text(toggleLabel)),
			),
			h.Form(post("/ui/nudge/now"),
				h.Button(h.Class("btn btn-ghost btn-sm"), h.Type("submit"), g.Text("Nudge me now")),
			),
		),
		h.P(h.Class("profile-hint"), g.Text("Mute for a while:")),
		h.Div(h.Class("kcard-actions"),
			muteBtn("1", "1 hour"),
			muteBtn("4", "4 hours"),
			muteBtn("8", "8 hours"),
			muteBtn("24", "until tomorrow"),
		),
	)
}
