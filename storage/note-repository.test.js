// Contract suite for the file-canonical note repository and note catalog
// (project 003, ADR-0004). A Note is a path-identified notes/*.md file with no
// mandatory frontmatter and no orbit-id; its identity is its path and its kind
// (inbox/reference/ai) is an inert body marker. Run: node --test storage/note-repository.test.js
import { test } from "node:test";
import assert from "node:assert/strict";

import { MemoryVault } from "./memory-vault.js";
import { MemoryIndex } from "./memory-index.js";
import { LifeIndexer } from "./life-indexer.js";
import { NoteCatalog } from "./note-catalog.js";
import { FileNoteRepository } from "./note-repository.js";
import { isCanvas } from "./canvas-validate.js";

const NOW = "2026-07-24T12:00:00.000Z";
const CANVASES = [
  ["canvas-root", "canvases/root.canvas"],
  ["canvas-daily", "canvases/daily.canvas"],
];

function setup() {
  const vault = new MemoryVault();
  const index = new MemoryIndex();
  const idToPath = new Map(CANVASES);
  const pathToId = new Map(CANVASES.map(([id, p]) => [p, id]));
  const indexer = new LifeIndexer({ vault, index, canvasIdFromPath: (p) => pathToId.get(p) });
  const catalog = new NoteCatalog({ vault });
  const repo = new FileNoteRepository({
    vault, index, indexer, catalog,
    canvasPathFromId: (id) => idToPath.get(id) || null,
    now: () => NOW,
  });
  return { vault, index, indexer, catalog, repo };
}

async function seedCanvases(vault) {
  for (const [, path] of CANVASES) {
    await vault.write(path, JSON.stringify({ nodes: [], edges: [] }, null, 2) + "\n");
  }
}

// --- create + index ----------------------------------------------------------

test("createNote writes a canonical note file and indexes it as an untyped source record", async () => {
  const { repo, vault, index } = setup();
  const { path, note } = await repo.createNote({ title: "First thought", body: "# First thought\n\nSome body text." });
  assert.equal(path, "notes/first-thought.md");
  assert.equal(await vault.read(path), "# First thought\n\nSome body text.");
  const record = index.getSourceFile(path);
  assert.equal(record.entityType, null, "a note is an untyped source record");
  assert.equal(record.parseStatus, "ok");
  assert.equal(note.title, "First thought");
  assert.equal(note.kind, null);
});

test("createNote prepends the inert kind marker for an inbox note", async () => {
  const { repo, vault } = setup();
  const { path, note } = await repo.createNote({ title: "Trip idea", body: "# Trip idea\n\nAutumn city break.", kind: "inbox" });
  const stored = await vault.read(path);
  assert.ok(stored.startsWith("<!-- orbit:inbox -->\n"), "marker is the first body line");
  assert.match(stored, /# Trip idea/);
  assert.match(stored, /Autumn city break\./);
  assert.equal(note.kind, "inbox");
});

test("createNote with a canvasId adds a standard file-node placement", async () => {
  const { repo, vault } = setup();
  await seedCanvases(vault);
  const { path, placement } = await repo.createNote({ title: "On canvas", body: "Body.", canvasId: "canvas-root", geometry: { x: 10, y: 20, width: 300, height: 180 } });
  const doc = JSON.parse(await vault.read("canvases/root.canvas"));
  assert.ok(isCanvas(doc), "canvas stays valid after placement");
  const node = doc.nodes.find((n) => n.type === "file" && n.file === path);
  assert.ok(node, "canvas must contain a file node pointing at the note");
  assert.equal(node.x, 10);
  assert.equal(node.width, 300);
  assert.equal(placement.nodeId, node.id);
  assert.equal(placement.canvasId, "canvas-root");
});

test("a note placement uses only standard JSON Canvas file-node fields (no orbit-id)", async () => {
  const { repo, vault } = setup();
  await seedCanvases(vault);
  const { path } = await repo.createNote({ title: "Standard", body: "Body.", canvasId: "canvas-root" });
  const doc = JSON.parse(await vault.read("canvases/root.canvas"));
  const node = doc.nodes.find((n) => n.file === path);
  assert.equal(node.type, "file");
  assert.deepEqual(Object.keys(node).sort(), ["file", "height", "id", "type", "width", "x", "y"]);
});

test("createNote accepts an explicit path override for seeding", async () => {
  const { repo, vault } = setup();
  const { path } = await repo.createNote({ path: "notes/start-here.md", body: "# Start here\n\nWelcome." });
  assert.equal(path, "notes/start-here.md");
  assert.equal(await vault.read(path), "# Start here\n\nWelcome.");
});

// --- collision-safe path allocation ------------------------------------------

test("allocatePath avoids collisions with a readable numeric suffix", async () => {
  const { repo } = setup();
  const a = await repo.createNote({ title: "New Thought", body: "One." });
  const b = await repo.createNote({ title: "New Thought", body: "Two." });
  const c = await repo.createNote({ title: "New Thought", body: "Three." });
  assert.equal(a.path, "notes/new-thought.md");
  assert.equal(b.path, "notes/new-thought-2.md");
  assert.equal(c.path, "notes/new-thought-3.md");
});

test("allocatePath avoids a case-fold collision", async () => {
  const { repo } = setup();
  const a = await repo.createNote({ title: "New Thought", body: "One." });
  const b = await repo.createNote({ title: "new thought", body: "Two." });
  assert.equal(a.path, "notes/new-thought.md");
  assert.equal(b.path, "notes/new-thought-2.md");
});

// --- placement add/remove ----------------------------------------------------

test("addPlacement rejects a duplicate node id with CANVAS_ID_DUPLICATE", async () => {
  const { repo, vault } = setup();
  await seedCanvases(vault);
  const { path } = await repo.createNote({ title: "Placed", body: "Body." });
  await repo.addPlacement(path, "canvas-root", { id: "dup-node" });
  await assert.rejects(() => repo.addPlacement(path, "canvas-root", { id: "dup-node" }), (err) => err.code === "CANVAS_ID_DUPLICATE");
});

test("addPlacement rejects a missing canvas with CANVAS_NOT_FOUND", async () => {
  const { repo } = setup();
  const { path } = await repo.createNote({ title: "Placed", body: "Body." });
  await assert.rejects(() => repo.addPlacement(path, "canvas-nope", {}), (err) => err.code === "CANVAS_NOT_FOUND");
});

test("addPlacement rejects a missing note with NOTE_NOT_FOUND", async () => {
  const { repo, vault } = setup();
  await seedCanvases(vault);
  await assert.rejects(() => repo.addPlacement("notes/ghost.md", "canvas-root", {}), (err) => err.code === "NOTE_NOT_FOUND");
});

test("addPlacement rejects non-integer or non-positive geometry with CANVAS_GEOMETRY_INVALID", async () => {
  const { repo, vault } = setup();
  await seedCanvases(vault);
  const { path } = await repo.createNote({ title: "Placed", body: "Body." });
  await assert.rejects(() => repo.addPlacement(path, "canvas-root", { width: 0 }), (err) => err.code === "CANVAS_GEOMETRY_INVALID");
  await assert.rejects(() => repo.addPlacement(path, "canvas-root", { x: 1.5, width: 100, height: 100 }), (err) => err.code === "CANVAS_GEOMETRY_INVALID");
});

test("removePlacement removes the node and its incident edges, keeping the canvas valid and the note intact", async () => {
  const { repo, vault } = setup();
  await seedCanvases(vault);
  const { path } = await repo.createNote({ title: "Removable", body: "Body.", canvasId: "canvas-root", geometry: { id: "note-node" } });
  // Add an unrelated node and an edge incident to the note node.
  const doc = JSON.parse(await vault.read("canvases/root.canvas"));
  doc.nodes.push({ id: "other", type: "text", text: "Other", x: 0, y: 0, width: 100, height: 100 });
  doc.edges.push({ id: "e1", fromNode: "note-node", toNode: "other" });
  await vault.write("canvases/root.canvas", JSON.stringify(doc, null, 2) + "\n");

  const result = await repo.removePlacement("canvas-root", "note-node");
  assert.equal(result.removed, true);
  const after = JSON.parse(await vault.read("canvases/root.canvas"));
  assert.ok(isCanvas(after));
  assert.ok(!after.nodes.some((n) => n.id === "note-node"));
  assert.ok(!after.edges.some((e) => e.id === "e1"), "incident edge must be removed");
  assert.ok(after.nodes.some((n) => n.id === "other"), "unrelated node survives");
  assert.equal(await vault.exists(path), true, "note file must survive placement removal");
});

test("removePlacement reports removed:false when the node is absent", async () => {
  const { repo, vault } = setup();
  await seedCanvases(vault);
  const result = await repo.removePlacement("canvas-root", "ghost");
  assert.equal(result.removed, false);
});

// --- update ------------------------------------------------------------------

test("updateNote rewrites the whole body under the expected hash", async () => {
  const { repo, vault, index } = setup();
  const { path } = await repo.createNote({ title: "Editable", body: "Old body." });
  const updated = await repo.updateNote(path, "# Editable\n\nNew body.");
  assert.equal(await vault.read(path), "# Editable\n\nNew body.");
  assert.equal(updated.body, "# Editable\n\nNew body.");
  assert.equal(updated.title, "Editable");
  assert.equal(index.getSourceFile(path).parseStatus, "ok");
});

test("updateNote rejects a write that happened behind the repository's back", async () => {
  const { repo, vault } = setup();
  const { path } = await repo.createNote({ title: "Editable", body: "Old body." });
  await vault.write(path, "External edit."); // no reindex, no reconcile
  await assert.rejects(() => repo.updateNote(path, "New body."), (err) => err.code === "WRITE_CONFLICT");
  assert.equal(await vault.read(path), "External edit.", "the external edit survives the rejected update");
});

test("updateNote rejects a missing note with NOTE_NOT_FOUND", async () => {
  const { repo } = setup();
  await assert.rejects(() => repo.updateNote("notes/ghost.md", "x"), (err) => err.code === "NOTE_NOT_FOUND");
});

// --- delete everywhere -------------------------------------------------------

test("deleteNote removes every placement then the file, keeping canvases valid", async () => {
  const { repo, vault } = setup();
  await seedCanvases(vault);
  const { path } = await repo.createNote({ title: "Doomed", body: "Body.", canvasId: "canvas-root", geometry: { id: "n1" } });
  await repo.addPlacement(path, "canvas-daily", { id: "n2" });
  const result = await repo.deleteNote(path);
  assert.equal(result.removedPlacements, 2);
  assert.equal(await vault.exists(path), false);
  for (const canvasPath of ["canvases/root.canvas", "canvases/daily.canvas"]) {
    const doc = JSON.parse(await vault.read(canvasPath));
    assert.ok(isCanvas(doc));
    assert.ok(!doc.nodes.some((n) => n.file === path));
  }
});

test("deleteNote rejects a missing note with NOTE_NOT_FOUND", async () => {
  const { repo } = setup();
  await assert.rejects(() => repo.deleteNote("notes/ghost.md"), (err) => err.code === "NOTE_NOT_FOUND");
});

// --- drain (re-place by path) ------------------------------------------------

test("replacePlacement drains a note from one canvas to another, preserving path identity", async () => {
  const { repo, vault } = setup();
  await seedCanvases(vault);
  const { path } = await repo.createNote({ title: "Drainable", body: "Body.", canvasId: "canvas-daily", geometry: { id: "drain-node" } });
  const before = await vault.read(path);
  const result = await repo.replacePlacement(path, "canvas-daily", "drain-node", "canvas-root", { id: "drained-node", x: 5, y: 6, width: 200, height: 120 });
  assert.equal(result.removed.removed, true);
  assert.equal(result.added.canvasId, "canvas-root");
  assert.equal(result.added.path, path, "identity (path) is unchanged");
  assert.equal(result.path, path);
  assert.equal(await vault.read(path), before, "the note file is untouched");
  const daily = JSON.parse(await vault.read("canvases/daily.canvas"));
  const root = JSON.parse(await vault.read("canvases/root.canvas"));
  assert.ok(!daily.nodes.some((n) => n.id === "drain-node"), "source placement removed");
  const target = root.nodes.find((n) => n.id === "drained-node");
  assert.ok(target, "target placement added");
  assert.equal(target.file, path, "target placement references the same path");
  assert.ok(isCanvas(daily) && isCanvas(root));
});

test("replacePlacement adds nothing when the source placement is absent", async () => {
  const { repo, vault } = setup();
  await seedCanvases(vault);
  const { path } = await repo.createNote({ title: "Drainable", body: "Body." });
  const result = await repo.replacePlacement(path, "canvas-daily", "ghost", "canvas-root", {});
  assert.equal(result.removed.removed, false);
  assert.equal(result.added, null);
  const root = JSON.parse(await vault.read("canvases/root.canvas"));
  assert.equal(root.nodes.length, 0, "no placement is added when nothing was removed");
});

// --- regression pin ----------------------------------------------------------

test("REGRESSION PIN: note placements stay out of the disposable placements index", async () => {
  const { repo, vault, index } = setup();
  await seedCanvases(vault);
  const { path } = await repo.createNote({ title: "Pinned", body: "Body.", canvasId: "canvas-root", geometry: { id: "pin-node" } });

  // The placement is a valid file node on the canvas...
  const doc = JSON.parse(await vault.read("canvases/root.canvas"));
  assert.ok(doc.nodes.some((n) => n.type === "file" && n.file === path && n.id === "pin-node"));
  assert.ok(isCanvas(doc));

  // ...but the disposable placements index does NOT track it: notes have no
  // orbit-id and notes/ is not an entity directory, so extractCanvasPlacements
  // skips them.
  assert.equal(index.allPlacements().filter((p) => p.sourcePath === path).length, 0);
  assert.equal(index.placementsForEntity(path).length, 0);
  assert.equal(index.allPlacements().length, 0, "no note placement may leak into the index");

  // Yet drain and delete-everywhere still resolve the placement by canvas scan.
  const drained = await repo.replacePlacement(path, "canvas-root", "pin-node", "canvas-daily", { id: "pin-moved" });
  assert.equal(drained.removed.removed, true);
  assert.equal(drained.added.canvasId, "canvas-daily");
  const deleted = await repo.deleteNote(path);
  assert.equal(deleted.removedPlacements, 1);
  assert.equal(await vault.exists(path), false);
});

// --- catalog projection ------------------------------------------------------

test("catalog derives the title from the first heading, falling back to the slug", async () => {
  const vault = new MemoryVault();
  await vault.write("notes/with-heading.md", "# My Heading\n\nBody.");
  await vault.write("notes/no-heading.md", "Just body text, no heading.");
  const catalog = new NoteCatalog({ vault });
  await catalog.rebuild();
  assert.equal(catalog.getByPath("notes/with-heading.md").title, "My Heading");
  assert.equal(catalog.getByPath("notes/no-heading.md").title, "no-heading");
});

test("catalog reads kind from inert body markers", async () => {
  const vault = new MemoryVault();
  await vault.write("notes/inbox.md", "<!-- orbit:inbox -->\n# Inbox");
  await vault.write("notes/reference.md", "<!-- orbit:reference -->\n# Ref");
  await vault.write("notes/ai.md", "<!-- orbit:ai-card -->\n# AI");
  await vault.write("notes/plain.md", "# Plain");
  const catalog = new NoteCatalog({ vault });
  await catalog.rebuild();
  assert.equal(catalog.getByPath("notes/inbox.md").kind, "inbox");
  assert.equal(catalog.getByPath("notes/reference.md").kind, "reference");
  assert.equal(catalog.getByPath("notes/ai.md").kind, "ai");
  assert.equal(catalog.getByPath("notes/plain.md").kind, null);
});

test("catalog records canvas placements and a missing-note diagnostic", async () => {
  const vault = new MemoryVault();
  await vault.write("notes/real.md", "# Real");
  await vault.write("canvases/root.canvas", JSON.stringify({ nodes: [
    { id: "n1", type: "file", file: "notes/real.md", x: 0, y: 0, width: 100, height: 100 },
    { id: "n2", type: "file", file: "notes/missing.md", x: 0, y: 0, width: 100, height: 100 },
  ], edges: [] }, null, 2) + "\n");
  const catalog = new NoteCatalog({ vault });
  await catalog.rebuild();
  assert.equal(catalog.getByPath("notes/real.md").placements.length, 1);
  assert.equal(catalog.getByPath("notes/real.md").placements[0].nodeId, "n1");
  assert.ok(catalog.diagnostics().some((d) => d.code === "NOTE_FILE_MISSING" && d.path === "notes/missing.md"));
});

test("NoteCatalog requires a vault", () => {
  assert.throws(() => new NoteCatalog({}), TypeError);
});
