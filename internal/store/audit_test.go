package store

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestListAudit(t *testing.T) {
	app := storetest.NewApp(t)

	// Seed three audit rows with different actors and actions
	Audit(app, "tasks", "task.transition", "task-1", true, nil)
	Audit(app, "os", "os.exec", "whoami", false, nil)
	Audit(app, "tasks", "task.create", "task-2", true, nil)

	// List all (no filters)
	all, err := ListAudit(app, "", "", 100)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("list all = %d rows, want 3", len(all))
	}

	// Filter by action prefix
	taskRows, err := ListAudit(app, "task.", "", 100)
	if err != nil {
		t.Fatalf("filter by action: %v", err)
	}
	if len(taskRows) != 2 {
		t.Fatalf("task. filter = %d rows, want 2", len(taskRows))
	}
	for _, r := range taskRows {
		if r.GetString("actor") != "tasks" {
			t.Fatalf("task filter returned actor %q, want tasks", r.GetString("actor"))
		}
	}

	// Filter by actor
	osRows, err := ListAudit(app, "", "os", 100)
	if err != nil {
		t.Fatalf("filter by actor: %v", err)
	}
	if len(osRows) != 1 {
		t.Fatalf("os actor filter = %d rows, want 1", len(osRows))
	}
	if osRows[0].GetString("action") != "os.exec" {
		t.Fatalf("os filter returned action %q, want os.exec", osRows[0].GetString("action"))
	}

	// Combined filters
	combined, err := ListAudit(app, "task.", "tasks", 100)
	if err != nil {
		t.Fatalf("combined filter: %v", err)
	}
	if len(combined) != 2 {
		t.Fatalf("combined filter = %d rows, want 2", len(combined))
	}

	// Ordering: newest first (reverse @rowid)
	if all[0].GetString("action") != "task.create" {
		t.Fatalf("first row (newest) has action %q, want task.create", all[0].GetString("action"))
	}
}

func TestListAuditLimit(t *testing.T) {
	app := storetest.NewApp(t)

	// Seed 5 rows
	for range 5 {
		Audit(app, "test", "action", "target", true, nil)
	}

	// Query with limit
	limited, err := ListAudit(app, "", "", 2)
	if err != nil {
		t.Fatalf("limited query: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("limited query = %d rows, want 2", len(limited))
	}
}
