package taskcards

// Internal test (package taskcards, not taskcards_test) so it can access
// renderTasks / filterBucket without exporting them.

import (
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/storetest"
)

// seedTask writes one task record and returns it.
func seedTask(t *testing.T, app core.App, title, status string, due time.Time) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		t.Fatalf("find tasks collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("title", title)
	rec.Set("status", status)
	if !due.IsZero() {
		rec.Set("due", due.UTC().Format("2006-01-02 15:04:05.000Z"))
	}
	if err := app.Save(rec); err != nil {
		t.Fatalf("save task %q: %v", title, err)
	}
	return rec
}

func renderTasksToString(app core.App, params map[string]string) string {
	node := renderTasks(app, params)
	var b strings.Builder
	if err := node.Render(&b); err != nil {
		panic(err)
	}
	return b.String()
}

func TestRenderTasksOpenDefault(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()

	seedTask(t, app, "Open task A", "open", now.Add(48*time.Hour))
	seedTask(t, app, "Open task B", "open", now.Add(96*time.Hour))
	seedTask(t, app, "Done task", "done", time.Time{})

	out := renderTasksToString(app, map[string]string{})
	// Must render open tasks as individual TaskCards.
	if !strings.Contains(out, "Open task A") {
		t.Errorf("missing 'Open task A' in:\n%s", out)
	}
	if !strings.Contains(out, "Open task B") {
		t.Errorf("missing 'Open task B' in:\n%s", out)
	}
	// Must NOT render done tasks when status defaults to open.
	if strings.Contains(out, "Done task") {
		t.Errorf("'Done task' should not appear in open-only render")
	}
	// Must be a bare stack — no ucard-quests container.
	if strings.Contains(out, "ucard-quests") {
		t.Errorf("bare tasks stack must not contain ucard-quests container chrome")
	}
	// Tasks are wrapped in .tasks-stack.
	if !strings.Contains(out, "tasks-stack") {
		t.Errorf("expected .tasks-stack wrapper")
	}
	// Each task is a tcard-{id}.
	if !strings.Contains(out, "tcard-") {
		t.Errorf("expected tcard-* ids in render")
	}
}

func TestRenderTasksStatusDone(t *testing.T) {
	app := storetest.NewApp(t)

	seedTask(t, app, "Open A", "open", time.Time{})
	seedTask(t, app, "Done B", "done", time.Time{})

	out := renderTasksToString(app, map[string]string{"status": "done"})
	if !strings.Contains(out, "Done B") {
		t.Errorf("missing 'Done B':\n%s", out)
	}
	// open task must not appear when status=done
	if strings.Contains(out, "Open A") {
		t.Errorf("'Open A' must not appear when status=done")
	}
}

func TestRenderTasksBucketOverdue(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()

	seedTask(t, app, "Overdue task", "open", now.Add(-24*time.Hour))
	seedTask(t, app, "Future task", "open", now.Add(72*time.Hour))

	out := renderTasksToString(app, map[string]string{"bucket": "overdue"})
	if !strings.Contains(out, "Overdue task") {
		t.Errorf("missing 'Overdue task' in overdue bucket:\n%s", out)
	}
	if strings.Contains(out, "Future task") {
		t.Errorf("'Future task' must not appear in overdue bucket")
	}
}

func TestRenderTasksEmptyState(t *testing.T) {
	app := storetest.NewApp(t)
	// No tasks seeded — expect the empty state.
	out := renderTasksToString(app, map[string]string{})
	if !strings.Contains(out, "No tasks match") {
		t.Errorf("expected 'No tasks match' empty state:\n%s", out)
	}
	if strings.Contains(out, "tasks-stack") {
		t.Errorf("empty state must not render tasks-stack")
	}
}

func TestFilterBucketEmpty(t *testing.T) {
	// Empty bucket string must return recs unchanged.
	recs := []*core.Record{{}}
	got := filterBucket(recs, "", time.Now())
	if len(got) != 1 {
		t.Errorf("filterBucket with empty bucket changed length: got %d, want 1", len(got))
	}
}
