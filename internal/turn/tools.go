package turn

import (
	"os"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/tools"
)

// Tools returns the enabled tool set: knowledge tools always (they only
// propose — the consent boundary holds), task, life and journal tools
// always (owner-consented by nature), OS access opt-in (AGENTS.md).
func Tools(app core.App) []agent.Tool {
	ts := tools.KnowledgeTools(app)
	ts = append(ts, tools.TaskTools(app)...)
	ts = append(ts, tools.LifeTools(app)...)
	ts = append(ts, tools.JournalTools(app)...)
	if os.Getenv("BALAUR_OS_ACCESS") == "1" {
		ts = append(ts, tools.OSAccess(app)...)
	}
	return ts
}
