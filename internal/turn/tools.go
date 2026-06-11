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
