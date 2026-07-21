// Phase 4a tests: canonical workspace persistence on a VaultStore.
// Run: node --test storage/phase4.test.js
import { test } from "node:test";
import assert from "node:assert/strict";

import { MemoryVault } from "./memory-vault.js";
import { isCanvas } from "./canvas-validate.js";
import {
  SIDECAR_PATH, SIDECAR_FORMAT, SIDECAR_VERSION, ROOT_CANVAS_PATH,
  canvasPathFor, toSidecar, parseSidecar, sidecarToJSON, canvasToJSON,
  loadWorkspace, hasWorkspace, WorkspaceStore,
} from "./workspace-vault.js";
import { ConflictError, SchemaError, PathError } from "./vault-errors.js";

function doc(...nodes) {
  return { nodes, edges: [] };
}
function textNode(id, text, x = 0, y = 0) {
  return { id, type: "text", x, y, width: 200, height: 100, text };
}

// A legacy in-memory workspace: root + one nested canvas reached via a portal
// file node, mirroring the shape app.js loads from localStorage today.
function legacyWorkspace() {
  return {
    version: 1,
    rootId: "canvas-root",
    activeId: "canvas-root",
    johnnyDecimal: { enabled: true, entries: { "10-19": { title: "Life admin" } } },
    canvases: {
      "canvas-root": {
        id: "canvas-root", title: "Life OS", parentId: null, portalNodeId: null, path: null,
        document: {
          nodes: [textNode("n1", "# Root"), { id: "portal-1", type: "file", x: 300, y: 0, width: 200, height: 100, file: "canvases/planning.canvas" }],
          edges: [{ id: "e1", fromNode: "n1", toNode: "portal-1" }],
        },
        camera: { x: 10, y: 20, zoom: 1 },
      },
      "canvas-planning": {
        id: "canvas-planning", title: "Planning", parentId: "canvas-root", portalNodeId: "portal-1", path: "canvases/planning.canvas",
        document: doc(textNode("p1", "# Planning")),
        camera: { x: 0, y: 0, zoom: 0.5 },
      },
    },
  };
}

test("canvasPathFor maps root, stored, and derived paths; rejects unsafe paths", () => {
  assert.equal(canvasPathFor({ id: "canvas-root" }, "canvas-root"), ROOT_CANVAS_PATH);
  assert.equal(canvasPathFor({ id: "c2", path: "canvases/planning.canvas" }, "root"), "canvases/planning.canvas");
  assert.equal(canvasPathFor({ id: "c3" }, "root"), "canvases/c3.canvas");
  assert.throws(() => canvasPathFor({ id: "c4", path: "../escape.canvas" }, "root"), PathError);
  assert.throws(() => canvasPathFor({ id: "c5", path: "/abs.canvas" }, "root"), PathError);
});

test("toSidecar strips documents and stamps format/version/paths", () => {
  const sidecar = toSidecar(legacyWorkspace());
  assert.equal(sidecar.format, SIDECAR_FORMAT);
  assert.equal(sidecar.version, SIDECAR_VERSION);
  assert.equal(sidecar.rootId, "canvas-root");
  assert.equal(sidecar.canvases["canvas-root"].path, ROOT_CANVAS_PATH);
  assert.equal(sidecar.canvases["canvas-planning"].path, "canvases/planning.canvas");
  for (const record of Object.values(sidecar.canvases)) {
    assert.ok(!("document" in record), "sidecar records must not embed documents");
  }
  assert.equal(sidecar.canvases["canvas-root"].title, "Life OS");
  assert.deepEqual(sidecar.canvases["canvas-planning"].camera, { x: 0, y: 0, zoom: 0.5 });
});

test("parseSidecar round-trips and rejects malformed sidecars", () => {
  const sidecar = toSidecar(legacyWorkspace());
  const parsed = parseSidecar(JSON.stringify(sidecar));
  assert.equal(parsed.rootId, "canvas-root");
  assert.equal(parsed.activeId, "canvas-root");

  assert.throws(() => parseSidecar("{not json"), /not valid JSON/);
  assert.throws(() => parseSidecar(JSON.stringify({ ...sidecar, format: "nope" })), SchemaError);
  assert.throws(() => parseSidecar(JSON.stringify({ ...sidecar, version: 99 })), SchemaError);
  assert.throws(() => parseSidecar(JSON.stringify({ ...sidecar, rootId: "missing" })), SchemaError);
  const badPath = structuredClone(sidecar);
  badPath.canvases["canvas-planning"].path = "../evil.canvas";
  assert.throws(() => parseSidecar(JSON.stringify(badPath)), PathError);
  // activeId falling back to root when unknown
  const noActive = { ...sidecar, activeId: "ghost" };
  assert.equal(parseSidecar(JSON.stringify(noActive)).activeId, "canvas-root");
});

test("migrate then load reconstructs an equivalent workspace", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  const original = legacyWorkspace();
  await store.migrate(original);

  assert.equal(await hasWorkspace(vault), true);
  const result = await store.load();
  assert.deepEqual(result.diagnostics, []);
  const ws = result.workspace;
  assert.equal(ws.rootId, "canvas-root");
  assert.equal(ws.activeId, "canvas-root");
  assert.equal(ws.canvases["canvas-root"].title, "Life OS");
  assert.equal(ws.canvases["canvas-planning"].title, "Planning");
  assert.equal(ws.canvases["canvas-planning"].parentId, "canvas-root");
  assert.deepEqual(ws.canvases["canvas-root"].document, original.canvases["canvas-root"].document);
  assert.deepEqual(ws.canvases["canvas-planning"].document, original.canvases["canvas-planning"].document);
  assert.deepEqual(ws.johnnyDecimal, original.johnnyDecimal);
});

test("every canvas file is independently valid JSON Canvas", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  await store.migrate(legacyWorkspace());
  for (const meta of await vault.list("canvases/")) {
    const parsed = JSON.parse(await vault.read(meta.path));
    assert.ok(isCanvas(parsed), `${meta.path} must be independently valid`);
    assert.equal(meta.mediaType, "application/jsoncanvas+json");
  }
});

test("the sidecar file contains no embedded documents", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  await store.migrate(legacyWorkspace());
  const raw = JSON.parse(await vault.read(SIDECAR_PATH));
  for (const record of Object.values(raw.canvases)) assert.ok(!("document" in record));
});

test("load returns null on a fresh vault", async () => {
  const vault = new MemoryVault();
  assert.equal(await hasWorkspace(vault), false);
  assert.equal(await loadWorkspace(vault), null);
  assert.equal(await new WorkspaceStore(vault).load(), null);
});

test("save persists edits and a reload reflects them", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  await store.migrate(legacyWorkspace());

  const loaded = (await store.load()).workspace;
  loaded.canvases["canvas-root"].title = "Renamed Root";
  loaded.canvases["canvas-root"].document.nodes.push(textNode("n2", "# Added", 40, 40));
  await store.save(loaded);

  const reloaded = (await new WorkspaceStore(vault).load()).workspace;
  assert.equal(reloaded.canvases["canvas-root"].title, "Renamed Root");
  assert.ok(reloaded.canvases["canvas-root"].document.nodes.some((n) => n.id === "n2"));
});

test("an external edit triggers a write conflict and leaves the sidecar untouched", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  await store.migrate(legacyWorkspace());
  const sidecarBefore = (await vault.stat(SIDECAR_PATH)).hash;

  // External actor rewrites the root canvas behind the store's back.
  await vault.write(ROOT_CANVAS_PATH, canvasToJSON(doc(textNode("ext", "# External edit"))));

  const ws = (await store.load()).workspace; // repopulates hashes from the vault
  // Now simulate a stale store that still believes the old hash is current.
  store.hashes.set(ROOT_CANVAS_PATH, "stale-hash");
  ws.canvases["canvas-root"].title = "Local change";
  await assert.rejects(() => store.save(ws), ConflictError);

  // Sidecar was written last, so it must be unchanged after the failed canvas write.
  assert.equal((await vault.stat(SIDECAR_PATH)).hash, sidecarBefore);
});

test("a missing canvas file loads as an empty document with a diagnostic, hierarchy intact", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  await store.migrate(legacyWorkspace());
  await vault.remove("canvases/planning.canvas");

  const result = await new WorkspaceStore(vault).load();
  assert.equal(result.workspace.canvases["canvas-planning"].parentId, "canvas-root");
  assert.deepEqual(result.workspace.canvases["canvas-planning"].document, { nodes: [], edges: [] });
  assert.ok(isCanvas(result.workspace.canvases["canvas-root"].document));
  assert.deepEqual(result.diagnostics, [{ path: "canvases/planning.canvas", code: "CANVAS_MISSING", message: "Missing canvas file: canvases/planning.canvas" }]);
});

test("an invalid canvas file loads with a CANVAS_INVALID diagnostic", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  await store.migrate(legacyWorkspace());
  await vault.write("canvases/planning.canvas", JSON.stringify({ nodes: [{ bad: true }] }));

  const result = await new WorkspaceStore(vault).load();
  assert.deepEqual(result.workspace.canvases["canvas-planning"].document, { nodes: [], edges: [] });
  assert.equal(result.diagnostics[0].code, "CANVAS_INVALID");
});

test("removing a canvas from the workspace deletes its orphaned file on save", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  await store.migrate(legacyWorkspace());

  const ws = (await store.load()).workspace;
  delete ws.canvases["canvas-planning"];
  ws.canvases["canvas-root"].document = doc(textNode("n1", "# Root")); // drop portal node + edge
  const result = await store.save(ws);

  assert.deepEqual(result.orphans, ["canvases/planning.canvas"]);
  assert.equal(await vault.exists("canvases/planning.canvas"), false);
  assert.equal(await vault.exists(ROOT_CANVAS_PATH), true);
});

test("migrate refuses to overwrite an existing vault (fresh precondition)", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  await store.migrate(legacyWorkspace());
  await assert.rejects(() => store.migrate(legacyWorkspace()), ConflictError);
});

test("refuses to write an invalid canvas document", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  const ws = legacyWorkspace();
  ws.canvases["canvas-root"].document = { nodes: [{ broken: true }], edges: [] };
  await assert.rejects(() => store.migrate(ws), SchemaError);
});

test("concurrent saves are serialized; the last enqueued wins", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  await store.migrate(legacyWorkspace());

  const wsA = (await store.load()).workspace;
  wsA.canvases["canvas-root"].title = "A";
  const wsB = structuredClone(wsA);
  wsB.canvases["canvas-root"].title = "B";

  await Promise.all([store.save(wsA), store.save(wsB)]);
  const final = (await new WorkspaceStore(vault).load()).workspace;
  assert.equal(final.canvases["canvas-root"].title, "B");
});

test("extracted isCanvas validator agrees on valid and invalid documents", () => {
  assert.ok(isCanvas(doc(textNode("a", "hi"))));
  assert.ok(isCanvas({ nodes: [], edges: [] }));
  assert.equal(isCanvas({ nodes: [{ id: "x", type: "bogus", x: 0, y: 0, width: 1, height: 1 }], edges: [] }), false);
  assert.equal(isCanvas(null), false);
});

test("sidecarToJSON and canvasToJSON are stable, pretty-printed text", () => {
  const ws = legacyWorkspace();
  assert.ok(sidecarToJSON(ws).endsWith("\n"));
  assert.ok(canvasToJSON(ws.canvases["canvas-root"].document).includes("\n  "));
});
