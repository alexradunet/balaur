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
	// Task group absent.
	if got["task_add"] {
		t.Error("task_add must be absent without the tasks group")
	}
	// Always-on core present.
	if !got["offer_choices"] {
		t.Error("always-on offer_choices missing")
	}
	if !got["self"] {
		t.Error("always-on self missing")
	}
}
