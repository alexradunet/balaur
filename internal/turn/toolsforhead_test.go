package turn

import (
	"testing"

	"github.com/alexradunet/balaur/internal/heads"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestToolsForHeadGroupsAllWired(t *testing.T) {
	t.Setenv("BALAUR_OS_ACCESS", "1") // so the "os" group contributes tools
	app := storetest.NewApp(t)

	// A non-empty group slice that matches nothing returns the always-on core.
	baseline := len(ToolsForHead(app, []string{"__definitely_not_a_real_group__"}))

	for _, g := range heads.Groups {
		got := len(ToolsForHead(app, []string{g}))
		if got <= baseline {
			t.Errorf("capability group %q is in heads.Groups but ToolsForHead wires no extra tools for it (got %d, core-only baseline %d) — turn/tools.go and heads.Groups have drifted", g, got, baseline)
		}
	}
}
