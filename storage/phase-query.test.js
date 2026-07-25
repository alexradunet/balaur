import { test } from "node:test";
import assert from "node:assert/strict";
import { MemoryIndex } from "./memory-index.js";
import { LifeQuery } from "./life-query.js";

test("LifeQuery exposes stable camelCase task and status queries", () => {
  const index = new MemoryIndex();
  index.insertEntityProjection("task", { id: "b", sourcePath: "tasks/b.md", sourceHash: "", title: "Beta", status: "next", priority: null, scheduledOn: "2026-07-21", dueOn: null, completedAt: null, estimateMinutes: null, recurrenceJson: null, createdAt: "", updatedAt: "" });
  index.insertEntityProjection("task", { id: "a", sourcePath: "tasks/a.md", sourceHash: "", title: "Alpha", status: "next", priority: 1, scheduledOn: "2026-07-21", dueOn: null, completedAt: null, estimateMinutes: null, recurrenceJson: null, createdAt: "", updatedAt: "" });
  const query = new LifeQuery(index);
  assert.deepEqual(query.today({ localDate: "2026-07-21" }).map((t) => t.id), ["a", "b"]);
  assert.equal(query.tasksByStatus("next")[0].scheduledOn, "2026-07-21");
});

test("LifeQuery computes latest daily habit state and streak", () => {
  const index = new MemoryIndex();
  index.insertEntityProjection("habit", { id: "h", sourcePath: "habits/h.md", sourceHash: "", title: "Walk", frequency: "daily", weekdaysJson: "[]", target: 1, unit: "walk", archivedAt: null, createdAt: "", updatedAt: "" });
  index.insertHabitEntries([
    { id: "e1", habitId: "h", localDate: "2026-07-20", status: "done", value: 1, occurredAt: "2026-07-20T08:00:00Z" },
    { id: "e2", habitId: "h", localDate: "2026-07-21", status: "done", value: 1, occurredAt: "2026-07-21T08:00:00Z" },
  ]);
  const habit = new LifeQuery(index).habits({ throughDate: "2026-07-21" })[0];
  assert.equal(habit.latestDailyState.localDate, "2026-07-21");
  assert.equal(habit.streak, 2);
});

test("LifeQuery compares event ranges by instant, not offset text", () => {
  const index = new MemoryIndex();
  index.insertEntityProjection("calendar-event", { id: "offset", sourcePath: "events/o.md", sourceHash: "", title: "Offset", startsAt: "2026-07-21T01:00:00+03:00", endsAt: null, localDate: "2026-07-21", timezone: "Europe/Bucharest", allDay: 0, source: "balaur", createdAt: "", updatedAt: "" });
  const query = new LifeQuery(index);
  assert.equal(query.eventsInRange("2026-07-20T21:30:00Z", "2026-07-20T22:30:00Z").length, 1);
  assert.throws(() => query.eventsInRange("not-an-instant", null), /Invalid from instant/);
});

test("LifeQuery returns journal dates and an exclusive event range", () => {
  const index = new MemoryIndex();
  index.insertEntityProjection("journal", { localDate: "2026-07-21", sourcePath: "journal/x.md", sourceHash: "", balaurId: "j", createdAt: "", updatedAt: "" });
  index.insertEntityProjection("calendar-event", { id: "e", sourcePath: "events/e.md", sourceHash: "", title: "Event", startsAt: "2026-07-21T10:00:00Z", endsAt: null, localDate: "2026-07-21", timezone: "UTC", allDay: 0, source: "balaur", createdAt: "", updatedAt: "" });
  const query = new LifeQuery(index);
  assert.equal(query.journalForDate("2026-07-21").balaurId, "j");
  assert.equal(query.eventsInRange("2026-07-21T00:00:00Z", "2026-07-22T00:00:00Z").length, 1);
});
