package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/storetest"
)

func findTool(t *testing.T, ts []agent.Tool, name string) agent.Tool {
	t.Helper()
	for _, tool := range ts {
		if tool.Spec.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q not found", name)
	return agent.Tool{}
}

func TestTaskAddListDone(t *testing.T) {
	app := storetest.NewApp(t)
	ts := TaskTools(app)
	ctx := context.Background()

	out, err := findTool(t, ts, "task_add").Execute(ctx,
		`{"title":"Call notary","due":"2026-06-12T10:00","notes":"apartment papers"}`)
	if err != nil {
		t.Fatalf("task_add: %v", err)
	}
	if !strings.Contains(out, "Task saved") || !strings.Contains(out, "Call notary") {
		t.Errorf("task_add reply: %q", out)
	}

	out, err = findTool(t, ts, "task_list").Execute(ctx, `{"scope":"open"}`)
	if err != nil {
		t.Fatalf("task_list: %v", err)
	}
	if !strings.Contains(out, "Call notary") {
		t.Errorf("task_list missing task: %q", out)
	}
	// The reply carries ids in [brackets]; lift one out for task_done.
	start := strings.Index(out, "[") + 1
	id := out[start : start+strings.Index(out[start:], "]")]

	out, err = findTool(t, ts, "task_done").Execute(ctx, `{"id":"`+id+`"}`)
	if err != nil {
		t.Fatalf("task_done: %v", err)
	}
	if !strings.Contains(out, "Done") {
		t.Errorf("task_done reply: %q", out)
	}
}

func TestTaskAddDateOnlyDefaultsMorning(t *testing.T) {
	app := storetest.NewApp(t)
	ts := TaskTools(app)

	out, err := findTool(t, ts, "task_add").Execute(context.Background(),
		`{"title":"Renew ID","due":"2026-07-01"}`)
	if err != nil {
		t.Fatalf("task_add: %v", err)
	}
	if !strings.Contains(out, "09:00") {
		t.Errorf("date-only reply should mention the 09:00 default: %q", out)
	}
}

func TestTaskAddRejectsRecurWithoutDue(t *testing.T) {
	app := storetest.NewApp(t)
	ts := TaskTools(app)

	_, err := findTool(t, ts, "task_add").Execute(context.Background(),
		`{"title":"Gym","recur":"weekly:mon"}`)
	if err == nil {
		t.Fatal("recurring without due: want error")
	}
}

func TestTaskDoneMarksRefresh(t *testing.T) {
	app := storetest.NewApp(t)
	ts := TaskTools(app)
	ctx := context.Background()

	// create a task via task_add
	out, err := findTool(t, ts, "task_add").Execute(ctx,
		`{"title":"Buy milk","due":"2026-06-12T10:00"}`)
	if err != nil {
		t.Fatalf("task_add: %v", err)
	}

	// lift its id from task_list (same technique as TestTaskAddListDone)
	out, err = findTool(t, ts, "task_list").Execute(ctx, `{"scope":"open"}`)
	if err != nil {
		t.Fatalf("task_list: %v", err)
	}
	start := strings.Index(out, "[") + 1
	id := out[start : start+strings.Index(out[start:], "]")]

	res, err := findTool(t, ts, "task_done").Execute(ctx, `{"id":"`+id+`"}`)
	if err != nil {
		t.Fatalf("task_done: %v", err)
	}
	types, rest, ok := ParseRefresh(res)
	if !ok {
		t.Fatalf("task_done result not refresh-marked: %q", res)
	}
	if len(types) != 1 || types[0] != "today" {
		t.Fatalf("refresh types = %v, want [today]", types)
	}
	if !strings.Contains(rest, "Done") {
		t.Fatalf("model text missing: %q", rest)
	}
}
