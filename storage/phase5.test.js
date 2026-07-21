// Phase 5 tests: file-canonical task repository (plan §13/§14.4).
// Run: node --test storage/phase5.test.js
import { test } from "node:test";
import assert from "node:assert/strict";

import { MemoryVault } from "./memory-vault.js";
import { MemoryIndex } from "./memory-index.js";
import { LifeIndexer } from "./life-indexer.js";
import { FileTaskRepository } from "./task-repository.js";
import { patchFields } from "./frontmatter.js";
import { TaskCodec } from "./entity-codec.js";

const NOW = "2026-07-21T18:00:00.000Z";
const CANVASES = [
  ["canvas-root", "canvases/root.canvas"],
  ["canvas-planning", "canvases/planning.canvas"],
];

function setup() {
  const vault = new MemoryVault();
  const index = new MemoryIndex();
  const idToPath = new Map(CANVASES);
  const pathToId = new Map(CANVASES.map(([id, p]) => [p, id]));
  const indexer = new LifeIndexer({ vault, index, canvasIdFromPath: (p) => pathToId.get(p) });
  const repo = new FileTaskRepository({
    vault, index, indexer,
    canvasPathFromId: (id) => idToPath.get(id) || null,
    now: () => NOW,
  });
  return { vault, index, indexer, repo };
}

async function seedCanvases(vault) {
  for (const [, path] of CANVASES) {
    await vault.write(path, JSON.stringify({ nodes: [], edges: [] }, null, 2) + "\n");
  }
}

test("createTask writes a canonical .md file and indexes it", async () => {
  const { repo, vault, index } = setup();
  const { id, path, task } = await repo.createTask({ id: "task-a1b2c3", title: "Finish quarterly review", body: "Collect the numbers." });
  assert.equal(id, "task-a1b2c3");
  assert.match(path, /^tasks\/finish-quarterly-review--a1b2c3\.md$/);
  const stored = await vault.read(path);
  assert.match(stored, /orbit-type: task/);
  assert.match(stored, /orbit-id: "?task-a1b2c3"?/);
  assert.match(stored, /Collect the numbers\./);
  const row = index.taskById(id);
  assert.equal(row.title, "Finish quarterly review");
  assert.equal(row.status, "next");
  assert.equal(row.sourcePath, path);
  assert.equal(task.status, "next");
});

test("createTask requires a title", async () => {
  const { repo } = setup();
  await assert.rejects(() => repo.createTask({ title: "   " }), /title is required/);
});

test("createTask with a canvasId adds a standard file-node placement", async () => {
  const { repo, vault, index } = setup();
  await seedCanvases(vault);
  const { id, path } = await repo.createTask({ id: "task-a1b2c3", title: "Review", canvasId: "canvas-root", geometry: { x: 10, y: 20, width: 300, height: 180 } });
  const doc = JSON.parse(await vault.read("canvases/root.canvas"));
  const node = doc.nodes.find((n) => n.type === "file" && n.file === path);
  assert.ok(node, "canvas must contain a file node pointing at the task");
  assert.equal(node.x, 10);
  assert.equal(node.width, 300);
  const placements = index.placementsForEntity(id);
  assert.equal(placements.length, 1);
  assert.equal(placements[0].canvasId, "canvas-root");
  assert.equal(placements[0].nodeId, node.id);
});

test("getTask reads the canonical file back", async () => {
  const { repo } = setup();
  await repo.createTask({ id: "task-a1b2c3", title: "Review", body: "Body text.", priority: 2 });
  const task = await repo.getTask("task-a1b2c3");
  assert.equal(task.title, "Review");
  assert.equal(task.priority, 2);
  assert.equal(task.body, "Body text.");
});

test("updateTask patches frontmatter preservation-first and bumps updated-at", async () => {
  const { repo, vault, index } = setup();
  const { path } = await repo.createTask({ id: "task-a1b2c3", title: "Review", body: "Keep me.", scheduledOn: "2026-07-22" });
  const updated = await repo.updateTask("task-a1b2c3", { status: "scheduled", priority: 1 });
  assert.equal(updated.status, "scheduled");
  assert.equal(updated.priority, 1);
  assert.equal(updated.body, "Keep me.", "body must be preserved");
  assert.equal(updated.updatedAt, NOW);
  assert.equal(updated.scheduledOn, "2026-07-22", "untouched fields preserved");
  assert.equal(index.taskById("task-a1b2c3").status, "scheduled");
  assert.match(await vault.read(path), /Keep me\./);
});

test("updateTask preserves unknown frontmatter fields", async () => {
  const { repo, vault, indexer } = setup();
  const { path } = await repo.createTask({ id: "task-a1b2c3", title: "Review" });
  // Inject an application-unknown field directly into the file, then index the
  // external edit so the repository's expected-hash precondition aligns.
  const content = await vault.read(path);
  const injected = content.replace(/^---\n/m, "---\ncustom-field: hello\n");
  await vault.write(path, injected);
  await indexer.indexFile(path, injected, {});
  await repo.updateTask("task-a1b2c3", { status: "done" });
  assert.match(await vault.read(path), /custom-field: hello/, "unknown field must survive the patch");
});

test("updateTask detects an unindexed external edit as a write conflict", async () => {
  const { repo, vault } = setup();
  const { path } = await repo.createTask({ id: "task-a1b2c3", title: "Review" });
  await vault.write(path, (await vault.read(path)).replace(/^---\n/m, "---\ncustom-field: hello\n"));
  // No reindex: the repository still believes the old hash is current.
  await assert.rejects(() => repo.updateTask("task-a1b2c3", { status: "done" }), /Hash mismatch|WRITE_CONFLICT|conflict/i);
});

test("updateTask can replace the body while preserving frontmatter", async () => {
  const { repo } = setup();
  await repo.createTask({ id: "task-a1b2c3", title: "Review", body: "Old body." });
  const updated = await repo.updateTask("task-a1b2c3", { body: "New body." });
  assert.equal(updated.body, "New body.");
  assert.equal(updated.title, "Review");
});

test("updateTask rejects unknown fields", async () => {
  const { repo } = setup();
  await repo.createTask({ id: "task-a1b2c3", title: "Review" });
  await assert.rejects(() => repo.updateTask("task-a1b2c3", { bogus: 1 }), /Unknown task field/);
});

test("completeTask and reopenTask toggle status and completed-at", async () => {
  const { repo, index } = setup();
  await repo.createTask({ id: "task-a1b2c3", title: "Review" });
  const done = await repo.completeTask("task-a1b2c3");
  assert.equal(done.status, "done");
  assert.equal(done.completedAt, NOW);
  assert.equal(index.taskById("task-a1b2c3").status, "done");
  const reopened = await repo.reopenTask("task-a1b2c3");
  assert.equal(reopened.status, "next");
  assert.equal(reopened.completedAt, null);
});

test("a task can have multiple placements across canvases", async () => {
  const { repo, vault, index } = setup();
  await seedCanvases(vault);
  const { id } = await repo.createTask({ id: "task-a1b2c3", title: "Review", canvasId: "canvas-root" });
  await repo.addPlacement(id, "canvas-planning", { x: 5, y: 5 });
  const placements = index.placementsForEntity(id);
  assert.equal(placements.length, 2);
  assert.deepEqual(placements.map((p) => p.canvasId).sort(), ["canvas-planning", "canvas-root"]);
});

test("removePlacement removes one placement but keeps the task and others", async () => {
  const { repo, vault, index } = setup();
  await seedCanvases(vault);
  const { id, path } = await repo.createTask({ id: "task-a1b2c3", title: "Review", canvasId: "canvas-root" });
  const second = await repo.addPlacement(id, "canvas-planning");
  const result = await repo.removePlacement("canvas-planning", second.nodeId);
  assert.equal(result.removed, true);
  assert.equal(index.placementsForEntity(id).length, 1);
  assert.equal(index.placementsForEntity(id)[0].canvasId, "canvas-root");
  assert.ok(await vault.read(path), "task file must survive placement removal");
  const planningDoc = JSON.parse(await vault.read("canvases/planning.canvas"));
  assert.ok(!planningDoc.nodes.some((n) => n.id === second.nodeId));
});

test("deleteTask removes every placement and the canonical file", async () => {
  const { repo, vault, index } = setup();
  await seedCanvases(vault);
  const { id, path } = await repo.createTask({ id: "task-a1b2c3", title: "Review", canvasId: "canvas-root" });
  await repo.addPlacement(id, "canvas-planning");
  const result = await repo.deleteTask(id);
  assert.equal(result.removedPlacements, 2);
  assert.equal(await vault.exists(path), false);
  assert.equal(index.taskById(id), null);
  assert.equal(index.placementsForEntity(id).length, 0);
  const rootDoc = JSON.parse(await vault.read("canvases/root.canvas"));
  assert.ok(!rootDoc.nodes.some((n) => n.file === path));
});

test("manually editing a task file then reindexing updates the index (Today follows files)", async () => {
  const { repo, vault, index, indexer } = setup();
  const { id, path } = await repo.createTask({ id: "task-a1b2c3", title: "Review" });
  const edited = patchFields(await vault.read(path), { status: "done", "completed-at": NOW }, TaskCodec.spec);
  await vault.write(path, edited);
  await indexer.indexFile(path, edited, {});
  assert.equal(index.taskById(id).status, "done");
  assert.equal(index.taskById(id).completedAt, NOW);
});

test("wiping the index and rebuilding reconstructs tasks and placements from files", async () => {
  const { repo, vault, index, indexer } = setup();
  await seedCanvases(vault);
  await repo.createTask({ id: "task-a1b2c3", title: "Review", canvasId: "canvas-root" });
  await repo.createTask({ id: "task-d4e5f6", title: "Book checkup", canvasId: "canvas-planning", status: "scheduled" });

  // Simulate deleting SQLite: clear the whole index, then cold-rebuild from files.
  index.clearAll();
  assert.equal(index.allTasks().length, 0);
  await indexer.rebuild();

  const tasks = index.allTasks();
  assert.equal(tasks.length, 2);
  assert.equal(index.taskById("task-a1b2c3").title, "Review");
  assert.equal(index.taskById("task-d4e5f6").status, "scheduled");
  assert.equal(index.placementsForEntity("task-a1b2c3").length, 1);
  assert.equal(index.placementsForEntity("task-d4e5f6").length, 1);
});

test("no custom Canvas node type or field is used for placements", async () => {
  const { repo, vault } = setup();
  await seedCanvases(vault);
  const { path } = await repo.createTask({ id: "task-a1b2c3", title: "Review", canvasId: "canvas-root" });
  const doc = JSON.parse(await vault.read("canvases/root.canvas"));
  const node = doc.nodes.find((n) => n.file === path);
  assert.equal(node.type, "file", "placement must be a standard file node");
  assert.deepEqual(Object.keys(node).sort(), ["file", "height", "id", "type", "width", "x", "y"]);
});
