// Phase 3 test suite (LifeIndexer) — run with:
//   node --test storage/phase3.test.js
// Exercises the indexer against the in-memory vault + index ports. The
// SQL-backed port in life-store.js mirrors MemoryIndex and is browser-verified.

import { test } from "node:test";
import assert from "node:assert/strict";

import { MemoryVault } from "./memory-vault.js";
import { MemoryIndex } from "./memory-index.js";
import { LifeIndexer, buildSourceRecord, extractCanvasPlacements, detectDuplicateIds } from "./life-indexer.js";
import { serializeTask, serializeJournal, serializeCalendarEvent, serializeHabitLog, serializeHabitEntry } from "./entity-codec.js";

const INST = "2026-07-21T18:00:00.000Z";

function taskMd(id, title, extra = {}) {
  return serializeTask({
    orbitId: id, title, status: extra.status || "next", priority: extra.priority ?? null,
    scheduledOn: extra.scheduledOn ?? null, dueOn: extra.dueOn ?? null, completedAt: extra.completedAt ?? null,
    estimateMinutes: extra.estimateMinutes ?? null, recurrence: null,
    createdAt: INST, updatedAt: INST, body: extra.body || "",
  });
}

function canvasDoc(nodes) { return JSON.stringify({ nodes, edges: [] }); }
function fileNode(id, file) { return { id, type: "file", file, x: 0, y: 0, width: 200, height: 120 }; }

function makeIndexer() {
  const vault = new MemoryVault();
  const index = new MemoryIndex();
  const indexer = new LifeIndexer({ vault, index });
  return { vault, index, indexer };
}

// --- buildSourceRecord classification ---------------------------------------

test("buildSourceRecord classifies task, untyped note, malformed, and canvas", async () => {
  const task = await buildSourceRecord("tasks/a--a1b2c3.md", taskMd("task-a1b2c3", "A"));
  assert.equal(task.record.entityType, "task");
  assert.equal(task.record.entityId, "task-a1b2c3");
  assert.equal(task.record.parseStatus, "ok");

  const note = await buildSourceRecord("notes/readme.md", "# Just a note\nNo frontmatter here.");
  assert.equal(note.record.entityType, null);
  assert.equal(note.record.parseStatus, "ok");

  const malformed = await buildSourceRecord("tasks/b--b1b2b3.md",
    "---\norbit-schema: 2\norbit-type: task\norbit-id: \"task-b\"\ntitle: \"x\"\nstatus: next\ncreated-at: \"2026-01-01T00:00:00.000Z\"\nupdated-at: \"2026-01-01T00:00:00.000Z\"\n---\n");
  assert.equal(malformed.record.parseStatus, "error");
  assert.match(malformed.record.parseError, /SCHEMA_NEWER/);

  const canvas = await buildSourceRecord("canvases/root.canvas", canvasDoc([]));
  assert.equal(canvas.record.entityType, "canvas");
  assert.equal(canvas.record.parseStatus, "ok");

  const badCanvas = await buildSourceRecord("canvases/broken.canvas", "not json");
  assert.equal(badCanvas.record.parseStatus, "error");
});

// --- per-file indexing & replacement ----------------------------------------

test("indexFile projects a task; reindexing replaces it in place", async () => {
  const { vault, index, indexer } = makeIndexer();
  await vault.write("tasks/a--a1b2c3.md", taskMd("task-a1b2c3", "A"));
  await indexer.indexFile("tasks/a--a1b2c3.md", await vault.read("tasks/a--a1b2c3.md"));
  assert.equal(index.allTasks().length, 1);
  assert.equal(index.taskById("task-a1b2c3").title, "A");
  assert.equal(index.taskById("task-a1b2c3").sourcePath, "tasks/a--a1b2c3.md");

  await vault.write("tasks/a--a1b2c3.md", taskMd("task-a1b2c3", "B"));
  await indexer.indexFile("tasks/a--a1b2c3.md", await vault.read("tasks/a--a1b2c3.md"));
  assert.equal(index.allTasks().length, 1);
  assert.equal(index.taskById("task-a1b2c3").title, "B");
});

test("a malformed file is isolated: it gets a diagnostic and does not corrupt others", async () => {
  const { vault, index, indexer } = makeIndexer();
  await vault.write("tasks/good--g1g1g1.md", taskMd("task-good", "Good"));
  await vault.write("tasks/bad--b1b1b1.md", "---\norbit-schema: 1\norbit-type: task\nstatus: next\nstatus: done\n---\n"); // duplicate key
  await indexer.rebuild();
  assert.equal(index.allTasks().length, 1);
  assert.equal(index.taskById("task-good").title, "Good");
  assert.equal(index.allSourceFiles().length, 2);
  const codes = index.allDiagnostics().map((d) => d.errorCode);
  assert.ok(codes.includes("PARSE_ERROR"));
  assert.ok(index.allDiagnostics().some((d) => d.sourcePath === "tasks/bad--b1b1b1.md"));
});

// --- duplicate identity ------------------------------------------------------

test("duplicate orbit-ids across files produce DUPLICATE_ID diagnostics", async () => {
  const { vault, index, indexer } = makeIndexer();
  await vault.write("tasks/one--x1x1x1.md", taskMd("task-dup", "One"));
  await vault.write("tasks/two--x2x2x2.md", taskMd("task-dup", "Two"));
  await indexer.rebuild();
  const dupes = index.allDiagnostics().filter((d) => d.errorCode === "DUPLICATE_ID");
  assert.equal(dupes.length, 2);
  assert.deepEqual(dupes.map((d) => d.sourcePath).sort(), ["tasks/one--x1x1x1.md", "tasks/two--x2x2x2.md"]);
});

test("detectDuplicateIds ignores null entity ids", () => {
  const diags = detectDuplicateIds([{ entityId: null, path: "a" }, { entityId: null, path: "b" }]);
  assert.equal(diags.length, 0);
});

// --- placements --------------------------------------------------------------

test("rebuild derives placements from canvas file nodes, including multiple placements", async () => {
  const { vault, index, indexer } = makeIndexer();
  await vault.write("tasks/a--a1b2c3.md", taskMd("task-a1b2c3", "A"));
  await vault.write("canvases/root.canvas", canvasDoc([fileNode("n1", "tasks/a--a1b2c3.md")]));
  await vault.write("canvases/work.canvas", canvasDoc([fileNode("n2", "tasks/a--a1b2c3.md")]));
  await indexer.rebuild();
  const placements = index.placementsForEntity("task-a1b2c3");
  assert.equal(placements.length, 2);
  assert.deepEqual(placements.map((p) => p.canvasId).sort(), ["root", "work"]);
  assert.equal(index.allTasks().length, 1);
});

test("a canvas reference to a missing entity file yields a MISSING_REFERENCE diagnostic", async () => {
  const { vault, index, indexer } = makeIndexer();
  await vault.write("canvases/root.canvas", canvasDoc([fileNode("n1", "tasks/ghost--g1g1g1.md")]));
  await indexer.rebuild();
  assert.ok(index.allDiagnostics().some((d) => d.errorCode === "MISSING_REFERENCE"));
  assert.equal(index.allPlacements().length, 0);
});

test("extractCanvasPlacements skips widgets and subcanvases", () => {
  const pathToEntity = new Map([["tasks/a--a1b2c3.md", { entityId: "task-a1b2c3", entityType: "task" }]]);
  const doc = { nodes: [
    fileNode("n1", "tasks/a--a1b2c3.md"),
    fileNode("n2", "widgets/focus.html"),
    fileNode("n3", "canvases/child.canvas"),
  ], edges: [] };
  const { placements } = extractCanvasPlacements("root", doc, pathToEntity);
  assert.equal(placements.length, 1);
  assert.equal(placements[0].entityId, "task-a1b2c3");
});

test("removing a placement (canvas edit) keeps the entity but drops the placement", async () => {
  const { vault, index, indexer } = makeIndexer();
  await vault.write("tasks/a--a1b2c3.md", taskMd("task-a1b2c3", "A"));
  await vault.write("canvases/root.canvas", canvasDoc([fileNode("n1", "tasks/a--a1b2c3.md")]));
  await indexer.rebuild();
  assert.equal(index.allPlacements().length, 1);
  const rev = vault.revision;
  await vault.write("canvases/root.canvas", canvasDoc([])); // remove the file node
  await indexer.reconcileWarm(rev);
  assert.equal(index.allPlacements().length, 0);
  assert.equal(index.allTasks().length, 1); // entity survives
});

// --- deletion reconciliation (plan §11.5) ------------------------------------

test("deleting an entity removes its projection and dangling placements, with a diagnostic", async () => {
  const { vault, index, indexer } = makeIndexer();
  await vault.write("tasks/a--a1b2c3.md", taskMd("task-a1b2c3", "A"));
  await vault.write("canvases/root.canvas", canvasDoc([fileNode("n1", "tasks/a--a1b2c3.md")]));
  await indexer.rebuild();
  assert.equal(index.allTasks().length, 1);
  assert.equal(index.allPlacements().length, 1);

  const rev = vault.revision;
  await vault.remove("tasks/a--a1b2c3.md");
  await indexer.reconcileWarm(rev);

  assert.equal(index.allTasks().length, 0);
  assert.equal(index.placementsForEntity("task-a1b2c3").length, 0);
  assert.ok(index.getSourceFile("tasks/a--a1b2c3.md") === null);
  assert.ok(index.allDiagnostics().some((d) => d.errorCode === "MISSING_REFERENCE"));
});

// --- warm reconciliation -----------------------------------------------------

test("warm reconciliation reindexes only changed paths", async () => {
  const { vault, index, indexer } = makeIndexer();
  await vault.write("tasks/a--a1b2c3.md", taskMd("task-a1b2c3", "A"));
  await vault.write("tasks/b--b1b2b3.md", taskMd("task-b1b2b3", "B"));
  await indexer.rebuild();
  const rev = vault.revision;

  await vault.write("tasks/a--a1b2c3.md", taskMd("task-a1b2c3", "A2")); // modify one
  await indexer.reconcileWarm(rev);

  assert.equal(index.taskById("task-a1b2c3").title, "A2");
  assert.equal(index.taskById("task-b1b2b3").title, "B");
  assert.equal(index.allTasks().length, 2);
});

// --- full rebuild idempotency ------------------------------------------------

test("full rebuild is idempotent", async () => {
  const { vault, index, indexer } = makeIndexer();
  await vault.write("tasks/a--a1b2c3.md", taskMd("task-a1b2c3", "A"));
  await vault.write("canvases/root.canvas", canvasDoc([fileNode("n1", "tasks/a--a1b2c3.md")]));
  const s1 = await indexer.rebuild();
  const s2 = await indexer.rebuild();
  assert.equal(s1.tasks, s2.tasks);
  assert.equal(s1.sourceFiles, s2.sourceFiles);
  assert.equal(s1.placements, s2.placements);
  assert.equal(s2.tasks, 1);
  assert.equal(s2.placements, 1);
});

// --- other entity types ------------------------------------------------------

test("journal, calendar-event, and habit-log entries are projected", async () => {
  const { vault, index, indexer } = makeIndexer();
  await vault.write("journal/2026/2026-07-21.md", serializeJournal({
    orbitId: "journal-2026-07-21", localDate: "2026-07-21", createdAt: INST, updatedAt: INST, body: "# Tue",
  }));
  await vault.write("events/dentist--m4n5o6.md", serializeCalendarEvent({
    orbitId: "event-dentist-m4n5o6", title: "Dentist", startsAt: "2026-07-24T09:00:00+03:00",
    endsAt: "2026-07-24T10:00:00+03:00", localDate: "2026-07-24", timezone: "Europe/Bucharest",
    allDay: false, source: "orbit", createdAt: INST, updatedAt: INST, body: "",
  }));
  const entry = { id: "habit-entry-r4s5t6", habit: "habit-walk", status: "done", value: 1, at: INST };
  await vault.write("habit-logs/2026/2026-07-21.md", serializeHabitLog({
    localDate: "2026-07-21", body: `- [x] Walk\n  ${serializeHabitEntry(entry)}\n`,
  }));

  await indexer.rebuild();
  assert.equal(index.allJournals().length, 1);
  assert.equal(index.allJournals()[0].localDate, "2026-07-21");
  assert.equal(index.allEvents().length, 1);
  assert.equal(index.allEvents()[0].timezone, "Europe/Bucharest");
  const entries = index.allHabitEntries();
  assert.equal(entries.length, 1);
  assert.equal(entries[0].habitId, "habit-walk");
  assert.equal(entries[0].status, "done");
});

test("index state records generation and indexed revision", async () => {
  const { vault, index, indexer } = makeIndexer();
  await vault.write("tasks/a--a1b2c3.md", taskMd("task-a1b2c3", "A"));
  await indexer.rebuild();
  assert.equal(index.getIndexState("indexedRevision"), String(vault.revision));
  assert.ok(index.getIndexState("generation"));
});
