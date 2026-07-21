// Phase 9 tests: Node filesystem vault adapter (plan §10/§18).
// Run: node --test storage/phase9.test.js
import { test, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";
import { promises as fsp } from "node:fs";
import os from "node:os";
import nodePath from "node:path";

import { FsVault } from "./fs-vault.js";
import { WorkspaceStore } from "./workspace-vault.js";
import { ConflictError, PathError } from "./vault-errors.js";

let dir;
let vault;
beforeEach(async () => {
  dir = await fsp.mkdtemp(nodePath.join(os.tmpdir(), "vault-test-"));
  vault = new FsVault(dir);
});
afterEach(async () => {
  await fsp.rm(dir, { recursive: true, force: true });
});

test("write/read round-trips and auto-creates nested directories", async () => {
  await vault.write("a/b/c.txt", "hello");
  assert.equal(await vault.read("a/b/c.txt"), "hello");
  // The file really is on disk.
  assert.equal(await fsp.readFile(nodePath.join(dir, "a/b/c.txt"), "utf8"), "hello");
});

test("exists and stat report size, hash, and media type", async () => {
  assert.equal(await vault.exists("f.md"), false);
  await vault.write("f.md", "hello");
  assert.equal(await vault.exists("f.md"), true);
  const stat = await vault.stat("f.md");
  assert.equal(stat.size, 5);
  assert.equal(stat.mediaType, "text/markdown");
  assert.match(stat.hash, /^[0-9a-f]{64}$/);
  assert.equal((await vault.stat("missing.md")), null);
});

test("read of a missing file throws NOT_FOUND", async () => {
  await assert.rejects(() => vault.read("nope.md"), (e) => e.code === "NOT_FOUND");
});

test("list returns all files sorted and filters by prefix", async () => {
  await vault.write("canvases/root.canvas", "{}");
  await vault.write("tasks/x.md", "x");
  await vault.write("notes/y.md", "y");
  assert.deepEqual((await vault.list("")).map((m) => m.path), ["canvases/root.canvas", "notes/y.md", "tasks/x.md"]);
  assert.deepEqual((await vault.list("canvases/")).map((m) => m.path), ["canvases/root.canvas"]);
});

test("expectedHash preconditions enforce optimistic concurrency", async () => {
  const meta = await vault.write("f.md", "one", { expectedHash: null });
  await vault.write("f.md", "two", { expectedHash: meta.hash });
  assert.equal(await vault.read("f.md"), "two");
  const current = await vault.stat("f.md");
  await assert.rejects(() => vault.write("f.md", "three", { expectedHash: meta.hash }), ConflictError);
  await assert.rejects(() => vault.write("f.md", "x", { expectedHash: null }), ConflictError);
  assert.equal(await vault.read("f.md"), "two", "failed writes must not modify content");
  assert.equal((await vault.stat("f.md")).hash, current.hash);
});

test("remove deletes the file and errors when absent", async () => {
  await vault.write("f.md", "x");
  await vault.remove("f.md");
  assert.equal(await vault.exists("f.md"), false);
  await assert.rejects(() => vault.remove("f.md"), /Not found/);
});

test("move relocates content and rejects an existing destination", async () => {
  await vault.write("a.md", "content");
  await vault.write("b.md", "other");
  await assert.rejects(() => vault.move("a.md", "b.md"), ConflictError);
  await vault.move("a.md", "c.md");
  assert.equal(await vault.exists("a.md"), false);
  assert.equal(await vault.read("c.md"), "content");
});

test("changesSince reports operations after a revision", async () => {
  await vault.write("a.md", "1");
  const rev = vault.revision;
  await vault.write("b.md", "2");
  await vault.remove("a.md");
  const changes = vault.changesSince(rev);
  assert.deepEqual(changes.map((c) => `${c.operation}:${c.path}`), ["create:b.md", "remove:a.md"]);
});

test("snapshot and restore round-trip the whole vault", async () => {
  await vault.write("a.md", "1");
  await vault.write("dir/b.md", "2");
  const snap = await vault.snapshot();
  await vault.remove("a.md");
  await vault.write("dir/b.md", "changed");
  await vault.restore(snap);
  assert.equal(await vault.read("a.md"), "1");
  assert.equal(await vault.read("dir/b.md"), "2");
});

test("rejects unsafe paths (traversal and absolute)", async () => {
  await assert.rejects(() => vault.write("../escape.md", "x"), PathError);
  await assert.rejects(() => vault.read("/abs.md"), PathError);
  await assert.rejects(() => vault.write("a/../../b.md", "x"), PathError);
});

test("integrates with WorkspaceStore: a canonical workspace persists to a real filesystem", async () => {
  const workspace = {
    version: 1, rootId: "canvas-root", activeId: "canvas-root",
    johnnyDecimal: { enabled: false, entries: {} },
    canvases: {
      "canvas-root": {
        id: "canvas-root", title: "Life OS", parentId: null, portalNodeId: null, path: null,
        document: { nodes: [{ id: "n1", type: "text", x: 0, y: 0, width: 200, height: 100, text: "# Root" }], edges: [] },
        camera: { x: 1, y: 2, zoom: 1 },
      },
    },
  };
  const store = new WorkspaceStore(vault);
  await store.migrate(workspace);

  // Real files exist on disk.
  assert.equal(await vault.exists(".orbit/workspace.json"), true);
  assert.equal(await vault.exists("canvases/root.canvas"), true);
  const sidecarOnDisk = JSON.parse(await fsp.readFile(nodePath.join(dir, ".orbit/workspace.json"), "utf8"));
  assert.equal(sidecarOnDisk.canvases["canvas-root"].path, "canvases/root.canvas");
  assert.ok(!("document" in sidecarOnDisk.canvases["canvas-root"]));

  // A fresh store (new session) loads the same workspace from disk.
  const reloaded = (await new WorkspaceStore(vault).load()).workspace;
  assert.equal(reloaded.canvases["canvas-root"].title, "Life OS");
  assert.deepEqual(reloaded.canvases["canvas-root"].document, workspace.canvases["canvas-root"].document);
});
