package tools

import (
	"context"
	"strings"
	"testing"
	"time"

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
	if !strings.Contains(out, "Task saved") || !strings.Contains(out, "Buy milk") {
		t.Errorf("task_add reply: %q", out)
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

func TestTaskSnoozeMarksRefresh(t *testing.T) {
	app := storetest.NewApp(t)
	ts := TaskTools(app)
	ctx := context.Background()

	_, err := findTool(t, ts, "task_add").Execute(ctx,
		`{"title":"Read book","due":"2026-06-12T10:00"}`)
	if err != nil {
		t.Fatalf("task_add: %v", err)
	}

	out, err := findTool(t, ts, "task_list").Execute(ctx, `{"scope":"open"}`)
	if err != nil {
		t.Fatalf("task_list: %v", err)
	}
	start := strings.Index(out, "[") + 1
	id := out[start : start+strings.Index(out[start:], "]")]

	res, err := findTool(t, ts, "task_snooze").Execute(ctx,
		`{"id":"`+id+`","until":"2099-01-01"}`)
	if err != nil {
		t.Fatalf("task_snooze: %v", err)
	}
	types, rest, ok := ParseRefresh(res)
	if !ok {
		t.Fatalf("task_snooze result not refresh-marked: %q", res)
	}
	if len(types) != 1 || types[0] != "today" {
		t.Fatalf("refresh types = %v, want [today]", types)
	}
	if !strings.Contains(rest, "Snoozed") {
		t.Fatalf("model text missing 'Snoozed': %q", rest)
	}
}

func TestTaskDropMarksRefresh(t *testing.T) {
	app := storetest.NewApp(t)
	ts := TaskTools(app)
	ctx := context.Background()

	_, err := findTool(t, ts, "task_add").Execute(ctx,
		`{"title":"Cancel gym","due":"2026-06-12T10:00"}`)
	if err != nil {
		t.Fatalf("task_add: %v", err)
	}

	out, err := findTool(t, ts, "task_list").Execute(ctx, `{"scope":"open"}`)
	if err != nil {
		t.Fatalf("task_list: %v", err)
	}
	start := strings.Index(out, "[") + 1
	id := out[start : start+strings.Index(out[start:], "]")]

	res, err := findTool(t, ts, "task_drop").Execute(ctx, `{"id":"`+id+`"}`)
	if err != nil {
		t.Fatalf("task_drop: %v", err)
	}
	types, rest, ok := ParseRefresh(res)
	if !ok {
		t.Fatalf("task_drop result not refresh-marked: %q", res)
	}
	if len(types) != 1 || types[0] != "today" {
		t.Fatalf("refresh types = %v, want [today]", types)
	}
	if !strings.Contains(rest, "Dropped") {
		t.Fatalf("model text missing 'Dropped': %q", rest)
	}
}

func TestParseDueHonorsLocation(t *testing.T) {
	// A non-UTC zone two hours ahead of UTC.
	tz := time.FixedZone("test+2", 2*3600)

	// Date-only input → 09:00 in the given location, not in UTC.
	got, dateOnly, err := ParseDue("2026-07-15", tz)
	if err != nil {
		t.Fatalf("ParseDue date-only: %v", err)
	}
	if !dateOnly {
		t.Error("dateOnly should be true for date-only input")
	}
	want := time.Date(2026, 7, 15, 9, 0, 0, 0, tz)
	if !got.Equal(want) {
		t.Errorf("date-only: got %v, want %v", got, want)
	}
	// Same input in UTC gives a different wall-clock moment.
	gotUTC, _, _ := ParseDue("2026-07-15", time.UTC)
	if got.Equal(gotUTC) {
		t.Error("location should change the parsed moment, but UTC and test+2 gave equal results")
	}

	// Datetime input (no zone) is interpreted in the given location.
	got2, dateOnly2, err := ParseDue("2026-07-15T14:30", tz)
	if err != nil {
		t.Fatalf("ParseDue datetime: %v", err)
	}
	if dateOnly2 {
		t.Error("dateOnly should be false for datetime input")
	}
	want2 := time.Date(2026, 7, 15, 14, 30, 0, 0, tz)
	if !got2.Equal(want2) {
		t.Errorf("datetime: got %v, want %v", got2, want2)
	}
}

func TestRecurringTaskDoneMarksRefresh(t *testing.T) {
	app := storetest.NewApp(t)
	ts := TaskTools(app)
	ctx := context.Background()

	_, err := findTool(t, ts, "task_add").Execute(ctx,
		`{"title":"Stretch","due":"2099-01-01","recur":"daily"}`)
	if err != nil {
		t.Fatalf("task_add (recurring): %v", err)
	}

	out, err := findTool(t, ts, "task_list").Execute(ctx, `{"scope":"open"}`)
	if err != nil {
		t.Fatalf("task_list: %v", err)
	}
	start := strings.Index(out, "[") + 1
	id := out[start : start+strings.Index(out[start:], "]")]

	res, err := findTool(t, ts, "task_done").Execute(ctx, `{"id":"`+id+`"}`)
	if err != nil {
		t.Fatalf("task_done (recurring): %v", err)
	}
	types, rest, ok := ParseRefresh(res)
	if !ok {
		t.Fatalf("recurring task_done result not refresh-marked: %q", res)
	}
	if len(types) != 1 || types[0] != "today" {
		t.Fatalf("refresh types = %v, want [today]", types)
	}
	if !strings.Contains(rest, "Done") {
		t.Fatalf("model text missing 'Done': %q", rest)
	}
}
