package turn

import (
	"os"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/ext"
	"github.com/alexradunet/balaur/internal/self"
	"github.com/alexradunet/balaur/internal/tools"
)

// Tools returns the enabled tool set: knowledge tools always (they only
// propose — the consent boundary holds), task, life and journal tools
// always (owner-consented by nature), OS access opt-in (AGENTS.md),
// propose_extension always (proposing is consent-safe), then approved
// balaur-extensions (collision-guarded so an extension can never shadow a
// built-in), and the read-only self tool last so its capability inventory
// reports the registry that actually shipped this turn.
func Tools(app core.App) []agent.Tool {
	ts := tools.KnowledgeTools(app)
	ts = append(ts, tools.TaskTools(app)...)
	ts = append(ts, tools.LifeTools(app)...)
	ts = append(ts, tools.JournalTools(app)...)
	ts = append(ts, tools.ChoiceTools(app)...)
	ts = append(ts, tools.UITools(app)...)
	if os.Getenv("BALAUR_OS_ACCESS") == "1" {
		ts = append(ts, tools.OSAccess(app)...)
	}
	ts = append(ts, ext.ProposeTool(app))

	taken := map[string]bool{"self": true}
	for _, t := range ts {
		taken[t.Spec.Name] = true
	}
	ts = append(ts, ext.Tools(app, taken)...)

	names := make([]string, 0, len(ts)+1)
	for _, t := range ts {
		names = append(names, t.Spec.Name)
	}
	names = append(names, "self")
	return append(ts, self.Tool(app, names))
}

// ToolsForHead returns the tool set for a head with the given capability
// groups. Empty groups returns the full Tools(app) — identical to the main
// head. Otherwise it assembles the always-on core (offer_choices, UI
// composition) plus the selected group constructors, then a self tool scoped to
// the resulting names. This is a capability filter, not a security boundary:
// the tools it returns still run with the owner's full trust. Group keys mirror
// internal/heads.Groups.
func ToolsForHead(app core.App, groups []string) []agent.Tool {
	if len(groups) == 0 {
		return Tools(app)
	}
	sel := make(map[string]bool, len(groups))
	for _, g := range groups {
		sel[g] = true
	}

	// Always-on core: interaction + UI composition.
	ts := tools.ChoiceTools(app)
	ts = append(ts, tools.UITools(app)...)

	if sel["memory"] {
		ts = append(ts, tools.KnowledgeTools(app)...)
	}
	if sel["tasks"] {
		ts = append(ts, tools.TaskTools(app)...)
	}
	if sel["life"] {
		ts = append(ts, tools.LifeTools(app)...)
	}
	if sel["journal"] {
		ts = append(ts, tools.JournalTools(app)...)
	}
	if sel["os"] && os.Getenv("BALAUR_OS_ACCESS") == "1" {
		ts = append(ts, tools.OSAccess(app)...)
	}
	if sel["extensions"] {
		ts = append(ts, ext.ProposeTool(app))
	}

	// self (and any approved extensions) come last, collision-guarded against
	// the names assembled so far — mirroring Tools(app).
	taken := map[string]bool{"self": true}
	for _, t := range ts {
		taken[t.Spec.Name] = true
	}
	if sel["extensions"] {
		ts = append(ts, ext.Tools(app, taken)...)
	}
	names := make([]string, 0, len(ts)+1)
	for _, t := range ts {
		names = append(names, t.Spec.Name)
	}
	names = append(names, "self")
	return append(ts, self.Tool(app, names))
}
