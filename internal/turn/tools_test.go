package turn

import (
	"testing"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/storetest"
)

func toolNameSet(ts []agent.Tool) map[string]bool {
	m := make(map[string]bool, len(ts))
	for _, t := range ts {
		m[t.Spec.Name] = true
	}
	return m
}

func TestToolsForHeadEmptyIsFullSet(t *testing.T) {
	app := storetest.NewApp(t)
	full := toolNameSet(Tools(app))
	got := toolNameSet(ToolsForHead(app, nil))
	if len(got) != len(full) {
		t.Fatalf("empty groups returned %d tools, want full %d", len(got), len(full))
	}
}

func TestToolsForHeadMemoryOnly(t *testing.T) {
	app := storetest.NewApp(t)
	got := toolNameSet(ToolsForHead(app, []string{"memory"}))
	// Memory group present.
	if !got["recall"] {
		t.Error("memory group should include recall")
	}
	// The propose-and-approve edit verb rides the memory group.
	if !got["propose_edit"] {
		t.Error("memory group should include propose_edit")
	}
	// Task group absent.
	if got["task_add"] {
		t.Error("task_add must be absent without the tasks group")
	}
	// Always-on core present — including persona/profile management, so a scoped
	// head can still switch back and manage identity.
	if !got["offer_choices"] {
		t.Error("always-on offer_choices missing")
	}
	if !got["head_switch"] {
		t.Error("always-on head_switch missing — a scoped head could not switch back")
	}
	if !got["profile_set"] {
		t.Error("always-on profile_set missing")
	}
	if !got["self"] {
		t.Error("always-on self missing")
	}
}
