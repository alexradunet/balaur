// settingsfocus_capabilities.go — the Capabilities settings section: read-only
// roster of tools, gates, model, skills, and extensions, plus the writable
// messenger gateway token control. Split out of settingsfocus.go (plan 186).
package settingscards

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/self"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
	"github.com/alexradunet/balaur/internal/ui"
)

// CapabilitiesView is the read-only roster of what Balaur can do right now — the
// owner-facing mirror of the `self` tool. It also carries the writable
// MessengerToken so the gateway control can be co-located with the other gates.
type CapabilitiesView struct {
	Tools          []string
	Gates          []GateView
	Model          string // active model label; "local (default)" when none chosen
	Skills         []string
	Extensions     []ExtStatusView
	Version        string
	Commit         string
	MessengerToken string // current messenger_token; empty = gateway disabled
}

// GateView is one capability gate and its state.
type GateView struct {
	Name string
	On   bool
}

// ExtStatusView groups extensions by consent-ledger status.
type ExtStatusView struct {
	Status string
	Names  []string
}

// BuildCapabilities assembles the read-only roster from self.Inventory, fed the
// live tool names. self.Inventory returns live Go types (not JSON), so the type
// assertions are stable.
func BuildCapabilities(app core.App) CapabilitiesView {
	inv := self.Inventory(app, turn.ToolNames(app))
	cv := CapabilitiesView{}
	if tn, ok := inv["tools"].([]string); ok {
		cv.Tools = tn
	}
	if v, ok := inv["version"].(string); ok {
		cv.Version = v
	}
	if c, ok := inv["commit"].(string); ok {
		cv.Commit = c
	}
	if sk, ok := inv["skills"].([]string); ok {
		cv.Skills = sk
	}
	if gates, ok := inv["gates"].(map[string]any); ok {
		for _, name := range []string{"os_access", "recap", "nudge", "briefing"} {
			on, _ := gates[name].(bool)
			cv.Gates = append(cv.Gates, GateView{Name: name, On: on})
		}
	}
	if mc, ok := inv["model_choice"].(map[string]any); ok {
		provider, _ := mc["provider"].(string)
		model, _ := mc["model"].(string)
		kind, _ := mc["kind"].(string)
		cv.Model = strings.TrimSpace(provider + " · " + model + " (" + kind + ")")
	} else {
		cv.Model = "local (default)"
	}
	if exts, ok := inv["extensions"].(map[string][]string); ok {
		for _, status := range []string{"active", "proposed", "disabled"} {
			if names := exts[status]; len(names) > 0 {
				cv.Extensions = append(cv.Extensions, ExtStatusView{Status: status, Names: names})
			}
		}
	}
	// Read the messenger token for the gateway control. Never log this value.
	cv.MessengerToken = store.GetOwnerSetting(app, "messenger_token", "")
	return cv
}

// CapabilitiesSection renders the read-only capability roster: the live tool
// set, gates, active model, skills, and extensions — the owner-facing mirror of
// the `self` tool, where model/cloud "parity = visibility" lands (selection
// stays an owner-only consent gate elsewhere). All values render via g.Text.
func CapabilitiesSection(v CapabilitiesView) g.Node {
	pills := func(items []string) g.Node {
		tags := make([]g.Node, 0, len(items))
		for _, s := range items {
			tags = append(tags, ui.Tag(g.Text(s)))
		}
		return h.Div(h.Class("habit-strip"), g.Group(tags))
	}
	gateTags := make([]g.Node, 0, len(v.Gates))
	for _, gt := range v.Gates {
		state := "off"
		if gt.On {
			state = "on"
		}
		gateTags = append(gateTags, ui.Tag(g.Text(gt.Name+": "+state)))
	}

	out := []g.Node{
		h.H2(h.Class("profile-card-title"), g.Text("What Balaur can do")),
		h.P(h.Class("profile-hint"), g.Text("The live tool set, gates, and model. Model selection stays yours — this is visibility, not control.")),
		h.Section(h.Class("k-section"),
			h.H3(g.Text(fmt.Sprintf("Tools (%d)", len(v.Tools)))),
			pills(v.Tools),
		),
		h.Section(h.Class("k-section"),
			h.H3(g.Text("Gates")),
			h.Div(h.Class("habit-strip"), g.Group(gateTags)),
		),
		h.Section(h.Class("k-section"),
			h.H3(g.Text("Model")),
			h.P(g.Text(v.Model)),
		),
	}
	if len(v.Skills) > 0 {
		out = append(out, h.Section(h.Class("k-section"),
			h.H3(g.Text("Skills")), pills(v.Skills)))
	}
	for _, ex := range v.Extensions {
		out = append(out, h.Section(h.Class("k-section"),
			h.H3(g.Text("Extensions · "+ex.Status)), pills(ex.Names)))
	}
	if v.Version != "" || v.Commit != "" {
		out = append(out, h.P(h.Class("profile-hint"), g.Text("Build: "+v.Version+" ("+v.Commit+")")))
	}
	out = append(out, MessengerGatewaySection(v))
	return h.Div(out...)
}

// MessengerGatewaySection renders the messenger gateway token control
// (#messenger-gateway-section). Re-render target after POST
// /ui/settings/messenger-token (outer patch #messenger-gateway-section).
// The token is shown in the owner-only settings view so the owner can copy it
// into a bridge config. Never log the token value.
func MessengerGatewaySection(v CapabilitiesView) g.Node {
	statusLine := "disabled — set a token to enable"
	if v.MessengerToken != "" {
		statusLine = "enabled"
	}
	return h.Article(
		h.Class("profile-card"), h.ID("messenger-gateway-section"),
		h.H3(h.Class("profile-card-title"), g.Text("Messenger gateway")),
		h.P(h.Class("profile-hint"), g.Text("Set a token to enable the local messenger endpoint; clear it to disable.")),
		h.P(h.Class("profile-hint"), g.Text("Status: "+statusLine+".")),
		h.Form(
			h.Class("profile-name-form"),
			data.On("submit", "@post('/ui/settings/messenger-token', {contentType:'form'})", data.ModifierPrevent),
			h.Label(h.For("messenger_token"), g.Text("Token")),
			h.Div(h.Class("profile-name-row"),
				h.Input(
					h.ID("messenger_token"),
					h.Name("messenger_token"),
					h.Type("text"),
					h.Value(v.MessengerToken),
					h.Placeholder("set a secret token…"),
					g.Attr("autocomplete", "off"),
				),
				h.Button(h.Class("btn btn-primary"), h.Type("submit"), g.Text("Save")),
			),
		),
	)
}
