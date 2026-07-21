// Phase 10 tests: index integrity audit, purge/rebuild recovery, and hardening
// (stress + crash). Run: node --test storage/phase10.test.js
import { test } from "node:test";
import assert from "node:assert/strict";

import { MemoryVault } from "./memory-vault.js";
import { MemoryIndex } from "./memory-index.js";
import { LifeIndexer } from "./life-indexer.js";
import { WorkspaceStore } from "./workspace-vault.js";
import { serializeTask } from "./entity-codec.js";
import { auditIndex, purgeAndRebuild } from "./index-integrity.js";
import { VaultError } from "./vault-errors.js";

const INST = "2026-07-21T18:00:00.000Z";

function setup() {
  const vault = new MemoryVault();
  const index = new MemoryIndex();
  const indexer = new LifeIndexer({ vault, index });
  return { vault, index, indexer };
}

function taskContent(id, title, status = "next") {
  return serializeTask({
    orbitId: id, title, status, priority: null, scheduledOn: null, dueOn: null,
    completedAt: status === "done" ? INST : null, estimateMinutes: null,
    recurrence: null, createdAt: INST, updatedAt: INST, body: "",
  });
}

async function writeTask(vault, indexer, id, title, status = "next") {
  const path = `tasks/${id}.md`;
  const content = taskContent(id, title, status);
  await vault.write(path, content);
  await indexer.indexFile(path, content, {});
  return path;
}

test("auditIndex is clean after a normal build", async () => {
  const { vault, index, indexer } = setup();
  await writeTask(vault, indexer, "task-aaa111", "One");
  await writeTask(vault, indexer, "task-bbb222", "Two");
  const audit = await auditIndex(vault, index);
  assert.equal(audit.ok, true);
  assert.deepEqual(audit.problems, []);
  assert.equal(audit.counts.sources, 2);
});

test("auditIndex detects an unindexed vault file", async () => {
  const { vault, index, indexer } = setup();
  await writeTask(vault, indexer, "task-aaa111", "One");
  await vault.write("tasks/orphan.md", taskContent("task-orphan", "Orphan")); // not indexed
  const audit = await auditIndex(vault, index);
  assert.equal(audit.ok, false);
  assert.ok(audit.problems.some((p) => p.code === "UNINDEXED_FILE" && p.path === "tasks/orphan.md"));
});

test("auditIndex detects a stale index source (file removed behind its back)", async () => {
  const { vault, index, indexer } = setup();
  const path = await writeTask(vault, indexer, "task-aaa111", "One");
  await vault.remove(path); // index still references it
  const audit = await auditIndex(vault, index);
  assert.ok(audit.problems.some((p) => p.code === "STALE_SOURCE" && p.path === path));
});

test("auditIndex detects a hash mismatch from an unreindexed external edit", async () => {
  const { vault, index, indexer } = setup();
  const path = await writeTask(vault, indexer, "task-aaa111", "One");
  await vault.write(path, taskContent("task-aaa111", "One", "done")); // external edit, not reindexed
  const audit = await auditIndex(vault, index);
  assert.ok(audit.problems.some((p) => p.code === "HASH_MISMATCH" && p.path === path));
});

test("auditIndex detects a duplicate orbit-id across files", async () => {
  const { vault, index, indexer } = setup();
  const dup = taskContent("task-dup", "A");
  await vault.write("tasks/a.md", dup); await indexer.indexFile("tasks/a.md", dup, {});
  const dup2 = taskContent("task-dup", "B");
  await vault.write("tasks/b.md", dup2); await indexer.indexFile("tasks/b.md", dup2, {});
  const audit = await auditIndex(vault, index);
  const diag = audit.problems.find((p) => p.code === "DUPLICATE_ID");
  assert.ok(diag);
  assert.equal(diag.details.orbitId, "task-dup");
  assert.equal(diag.details.paths.length, 2);
});

test("purgeAndRebuild recovers a clean audit from a corrupted index", async () => {
  const { vault, index, indexer } = setup();
  const path = await writeTask(vault, indexer, "task-aaa111", "One");
  await vault.remove(path); // make the index stale
  assert.equal((await auditIndex(vault, index)).ok, false);
  const audit = await purgeAndRebuild(vault, index, indexer);
  assert.equal(audit.ok, true, "rebuild from the vault must be clean");
});

test("stress: a large batch indexes and rebuilds with a clean audit", async () => {
  const { vault, index, indexer } = setup();
  const N = 150;
  for (let i = 0; i < N; i++) {
    const id = `task-${String(i).padStart(6, "0")}`;
    await writeTask(vault, indexer, id, `Task ${i}`, i % 3 === 0 ? "done" : "next");
  }
  assert.equal(index.allTasks().length, N);
  const audit = await purgeAndRebuild(vault, index, indexer);
  assert.equal(audit.ok, true);
  assert.equal(audit.counts.sources, N);
  assert.equal(index.allTasks().length, N);
  assert.equal(index.allTasks().filter((t) => t.status === "done").length, 50);
});

test("crash mid-save leaves the vault loadable (hierarchy not lost)", async () => {
  const { vault } = setup();
  const workspace = {
    version: 1, rootId: "canvas-root", activeId: "canvas-root",
    johnnyDecimal: { enabled: false, entries: {} },
    canvases: { "canvas-root": { id: "canvas-root", title: "Life OS", parentId: null, portalNodeId: null, path: null, document: { nodes: [], edges: [] }, camera: { x: 0, y: 0, zoom: 1 } } },
  };
  const store = new WorkspaceStore(vault);
  await store.migrate(workspace);

  // Inject a crash on the next write of a subsequent save.
  vault.failNext("write", new VaultError("disk full", { code: "STORAGE_UNAVAILABLE" }));
  const edited = structuredClone(workspace);
  edited.canvases["canvas-root"].title = "Changed";
  await assert.rejects(() => store.save(edited), /disk full|STORAGE_UNAVAILABLE/);

  // The sidecar was written last, so the previous consistent state is intact.
  const loaded = await new WorkspaceStore(vault).load();
  assert.ok(loaded.workspace);
  assert.equal(loaded.workspace.canvases["canvas-root"].title, "Life OS");
});

test("every task row is reconstructible after deleting the index", async () => {
  const { vault, index, indexer } = setup();
  const titles = [];
  for (let i = 0; i < 25; i++) {
    const id = `task-${String(i).padStart(6, "0")}`;
    await writeTask(vault, indexer, id, `Task ${i}`, i % 2 ? "done" : "next");
    titles.push(`Task ${i}`);
  }
  const before = index.allTasks().map((t) => t.title).sort();
  assert.deepEqual(before, [...titles].sort());

  const audit = await purgeAndRebuild(vault, index, indexer); // simulates deleting index storage
  assert.equal(audit.ok, true);
  assert.deepEqual(index.allTasks().map((t) => t.title).sort(), before);
});
