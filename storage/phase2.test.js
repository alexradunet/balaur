// Phase 2 test suite (in-memory VaultStore) — run with:
//   node --test storage/phase2.test.js

import { test } from "node:test";
import assert from "node:assert/strict";

import { MemoryVault } from "./memory-vault.js";
import { mediaTypeFor } from "./vault-store.js";
import { ConflictError, PathError, VaultError } from "./vault-errors.js";

test("mediaTypeFor infers from extension", () => {
  assert.equal(mediaTypeFor("canvases/root.canvas"), "application/jsoncanvas+json");
  assert.equal(mediaTypeFor("tasks/x.md"), "text/markdown");
  assert.equal(mediaTypeFor("widgets/w.html"), "text/html");
  assert.equal(mediaTypeFor("blob.bin"), "application/octet-stream");
});

test("create, read, stat, exists, list", async () => {
  const vault = new MemoryVault();
  assert.equal(await vault.exists("tasks/a.md"), false);
  const meta = await vault.write("tasks/a.md", "# A\n");
  assert.equal(meta.path, "tasks/a.md");
  assert.equal(meta.mediaType, "text/markdown");
  assert.match(meta.hash, /^[0-9a-f]{64}$/);
  assert.equal(meta.size, 4);
  assert.equal(await vault.read("tasks/a.md"), "# A\n");
  assert.equal(await vault.exists("tasks/a.md"), true);
  const stat = await vault.stat("tasks/a.md");
  assert.equal(stat.hash, meta.hash);
  assert.equal(stat.content, undefined); // metadata has no content
  await vault.write("tasks/b.md", "# B\n");
  await vault.write("habits/h.md", "h");
  const listed = (await vault.list("tasks/")).map((r) => r.path);
  assert.deepEqual(listed, ["tasks/a.md", "tasks/b.md"]);
});

test("modify updates hash and revision; read missing throws NOT_FOUND", async () => {
  const vault = new MemoryVault();
  const first = await vault.write("t.md", "one");
  const second = await vault.write("t.md", "two");
  assert.notEqual(first.hash, second.hash);
  assert.ok(second.revision > first.revision);
  await assert.rejects(() => vault.read("missing.md"), (e) => e.code === "NOT_FOUND");
});

test("optimistic hash preconditions", async () => {
  const vault = new MemoryVault();
  const meta = await vault.write("t.md", "v1", { expectedHash: null }); // create
  assert.ok(meta.hash);
  // create-when-exists with expectedHash:null conflicts
  await assert.rejects(() => vault.write("t.md", "v2", { expectedHash: null }), ConflictError);
  // wrong hash conflicts and does not overwrite
  await assert.rejects(() => vault.write("t.md", "v2", { expectedHash: "deadbeef" }), ConflictError);
  assert.equal(await vault.read("t.md"), "v1");
  // correct hash succeeds
  const updated = await vault.write("t.md", "v2", { expectedHash: meta.hash });
  assert.equal(await vault.read("t.md"), "v2");
  assert.notEqual(updated.hash, meta.hash);
});

test("move renames, emits oldPath, and rejects existing destination", async () => {
  const vault = new MemoryVault();
  await vault.write("a.md", "x");
  await vault.write("b.md", "y");
  const events = [];
  vault.subscribe((c) => events.push(c));
  const moved = await vault.move("a.md", "c.md");
  assert.equal(moved.path, "c.md");
  assert.equal(await vault.exists("a.md"), false);
  assert.equal(await vault.read("c.md"), "x");
  assert.deepEqual(events.at(-1), { type: "move", path: "c.md", oldPath: "a.md", hash: moved.hash });
  await assert.rejects(() => vault.move("c.md", "b.md"), ConflictError);
});

test("remove deletes and rejects missing", async () => {
  const vault = new MemoryVault();
  await vault.write("a.md", "x");
  assert.equal(await vault.remove("a.md"), true);
  assert.equal(await vault.exists("a.md"), false);
  await assert.rejects(() => vault.remove("a.md"), (e) => e.code === "NOT_FOUND");
});

test("revision journal is monotonic and changesSince filters", async () => {
  const vault = new MemoryVault();
  const rev0 = vault.revision;
  await vault.write("a.md", "1");
  await vault.write("b.md", "2");
  await vault.write("a.md", "3");
  assert.ok(vault.revision > rev0);
  const all = vault.changesSince(0);
  assert.equal(all.length, 3);
  const recent = vault.changesSince(all[0].revision);
  assert.equal(recent.length, 2);
  assert.deepEqual(all.map((e) => e.operation), ["create", "create", "modify"]);
});

test("snapshot and restore round-trip", async () => {
  const vault = new MemoryVault();
  await vault.write("tasks/a.md", "# A");
  await vault.write("canvases/root.canvas", '{"nodes":[],"edges":[]}');
  const snap = await vault.snapshot();
  assert.equal(snap.files.length, 2);
  assert.deepEqual(snap.files.map((f) => f.path), ["canvases/root.canvas", "tasks/a.md"]);

  const fresh = new MemoryVault();
  const result = await fresh.restore(snap);
  assert.equal(result.count, 2);
  assert.equal(await fresh.read("tasks/a.md"), "# A");
  // hashes match across vaults for identical content
  assert.equal((await fresh.stat("tasks/a.md")).hash, (await vault.stat("tasks/a.md")).hash);
});

test("injected failures leave state intact", async () => {
  const vault = new MemoryVault();
  await vault.write("a.md", "original");
  const before = await vault.stat("a.md");

  vault.failNext("write", new VaultError("boom", { code: "INJECTED" }));
  await assert.rejects(() => vault.write("a.md", "changed", { expectedHash: before.hash }), (e) => e.code === "INJECTED");
  assert.equal(await vault.read("a.md"), "original"); // unchanged

  vault.failNext("remove");
  await assert.rejects(() => vault.remove("a.md"), (e) => e.code === "INJECTED");
  assert.equal(await vault.exists("a.md"), true); // still present
});

test("subscribers receive create/modify/remove events", async () => {
  const vault = new MemoryVault();
  const events = [];
  const unsub = vault.subscribe((c) => events.push(c.type));
  await vault.write("a.md", "1");
  await vault.write("a.md", "2");
  await vault.remove("a.md");
  unsub();
  await vault.write("b.md", "3"); // not observed after unsubscribe
  assert.deepEqual(events, ["create", "modify", "remove"]);
});

test("path safety and case-fold collisions are enforced", async () => {
  const vault = new MemoryVault();
  await assert.rejects(() => vault.write("../escape.md", "x"), PathError);
  await assert.rejects(() => vault.write("/abs.md", "x"), PathError);
  await vault.write("tasks/Report.md", "x");
  await assert.rejects(() => vault.write("tasks/report.md", "y"), PathError); // case-fold collision
  // exact same path is an update, not a collision
  await vault.write("tasks/Report.md", "y");
  assert.equal(await vault.read("tasks/Report.md"), "y");
});
