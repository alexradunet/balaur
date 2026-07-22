// Phase 4 tests: version-2 whole-space backup/restore (plan §15).
// Run: node --test storage/phase4-backup.test.js
import { test } from "node:test";
import assert from "node:assert/strict";

import { MemoryVault } from "./memory-vault.js";
import { WorkspaceStore, SIDECAR_PATH, loadWorkspace } from "./workspace-vault.js";
import { serializeTask } from "./entity-codec.js";
import { serializeComponentCard } from "./component-card-codec.js";
import {
  exportBundle, serializeBundle, parseBundle, validateBundle, importBundle, assertCompleteExport,
  BACKUP_FORMAT, BACKUP_VERSION,
} from "./workspace-backup.js";
import { SchemaError, PathError } from "./vault-errors.js";

const INST = "2026-07-21T18:00:00.000Z";
const sampleTask = {
  orbitId: "task-a1b2c3", title: "Finish quarterly review", status: "next",
  priority: 1, scheduledOn: "2026-07-22", dueOn: "2026-07-25",
  completedAt: null, estimateMinutes: 45, recurrence: null,
  createdAt: INST, updatedAt: INST,
  body: "Collect the outstanding numbers and prepare the summary.",
};
const TASK_PATH = "tasks/finish-quarterly-review--a1b2c3.md";
const CARD_PATH = "cards/weekly-focus--d4e5f6.md";
const sampleCard = {
  id: "card-d4e5f6", title: "Weekly focus", recipe: "metric",
  value: "72%", label: "Deep-work target", progress: 0.72, trend: "up",
  body: "Up from 64% last week.",
};

function textNode(id, text, x = 0, y = 0) {
  return { id, type: "text", x, y, width: 200, height: 100, text };
}
function legacyWorkspace() {
  return {
    version: 1, rootId: "canvas-root", activeId: "canvas-root",
    johnnyDecimal: { enabled: false, entries: {} },
    canvases: {
      "canvas-root": {
        id: "canvas-root", title: "Life OS", parentId: null, portalNodeId: null, path: null,
        document: { nodes: [textNode("n1", "# Root"), { id: "portal-1", type: "file", x: 300, y: 0, width: 200, height: 100, file: "canvases/planning.canvas" }], edges: [{ id: "e1", fromNode: "n1", toNode: "portal-1" }] },
        camera: { x: 10, y: 20, zoom: 1 },
      },
      "canvas-planning": {
        id: "canvas-planning", title: "Planning", parentId: "canvas-root", portalNodeId: "portal-1", path: "canvases/planning.canvas",
        document: { nodes: [textNode("p1", "# Planning")], edges: [] },
        camera: { x: 0, y: 0, zoom: 0.5 },
      },
    },
  };
}
async function populatedVault() {
  const vault = new MemoryVault();
  await new WorkspaceStore(vault).migrate(legacyWorkspace());
  await vault.write(TASK_PATH, serializeTask(sampleTask));
  await vault.write(CARD_PATH, serializeComponentCard(sampleCard));
  await vault.write("notes/idea.md", "# Just a note\n\nNo frontmatter here.");
  return vault;
}

test("exportBundle produces a sorted, sidecar-separated bundle", async () => {
  const vault = await populatedVault();
  const { bundle, diagnostics } = await exportBundle(vault);
  assert.deepEqual(diagnostics, []);
  assert.equal(bundle.format, BACKUP_FORMAT);
  assert.equal(bundle.version, BACKUP_VERSION);
  assert.equal(bundle.workspace.rootId, "canvas-root");
  const paths = bundle.files.map((f) => f.path);
  assert.deepEqual(paths, [...paths].sort(), "files must be deterministically sorted");
  assert.ok(!paths.includes(SIDECAR_PATH), "sidecar must not be duplicated into files");
  assert.ok(paths.includes("canvases/root.canvas"));
  assert.ok(paths.includes("canvases/planning.canvas"));
  assert.ok(paths.includes(TASK_PATH));
  assert.ok(paths.includes("notes/idea.md"));
  assert.ok(paths.includes(CARD_PATH));
});

test("export preserves entity bytes exactly", async () => {
  const vault = await populatedVault();
  const { bundle } = await exportBundle(vault);
  const taskFile = bundle.files.find((f) => f.path === TASK_PATH);
  assert.equal(taskFile.text, serializeTask(sampleTask));
  assert.equal(taskFile.mediaType, "text/markdown");
  const cardFile = bundle.files.find((f) => f.path === CARD_PATH);
  assert.equal(cardFile.text, serializeComponentCard(sampleCard));
  assert.equal(cardFile.mediaType, "text/markdown");
});

test("export -> import round-trips into a fresh staging vault", async () => {
  const vault = await populatedVault();
  const { bundle } = await exportBundle(vault);
  const text = serializeBundle(bundle);

  const staging = new MemoryVault();
  const summary = await importBundle(staging, text);
  assert.deepEqual(summary.diagnostics, []);
  assert.equal(summary.entityCount, 2);

  const restored = (await loadWorkspace(staging)).workspace;
  assert.equal(restored.canvases["canvas-root"].title, "Life OS");
  assert.deepEqual(restored.canvases["canvas-planning"].document.nodes, [textNode("p1", "# Planning")]);
  assert.equal(await staging.read(TASK_PATH), serializeTask(sampleTask));
  assert.equal(await staging.read(CARD_PATH), serializeComponentCard(sampleCard));
});

test("export refuses a vault with an invalid canvas", async () => {
  const vault = await populatedVault();
  await vault.write("canvases/planning.canvas", JSON.stringify({ nodes: [{ bad: true }] }));
  await assert.rejects(() => exportBundle(vault), (e) => e instanceof SchemaError && e.code === "CANVAS_INVALID");
});

test("export reports unreadable files instead of dropping them silently", async () => {
  class FlakyVault extends MemoryVault {
    async read(path) {
      if (path === "notes/idea.md") throw new Error("disk error");
      return super.read(path);
    }
  }
  const vault = new FlakyVault();
  await new WorkspaceStore(vault).migrate(legacyWorkspace());
  await vault.write("notes/idea.md", "# note");
  const { bundle, diagnostics } = await exportBundle(vault);
  assert.deepEqual(diagnostics, [{ path: "notes/idea.md", code: "FILE_UNREADABLE", message: "Unreadable file: notes/idea.md" }]);
  assert.ok(!bundle.files.some((f) => f.path === "notes/idea.md"));
});

test("parseBundle rejects bad envelopes", () => {
  assert.throws(() => parseBundle("{nope"), /not valid JSON/);
  const base = { format: BACKUP_FORMAT, version: BACKUP_VERSION, workspace: {}, files: [] };
  assert.throws(() => parseBundle(JSON.stringify({ ...base, format: "other" })), SchemaError);
  assert.throws(() => parseBundle(JSON.stringify({ ...base, version: 1 })), (e) => e.code === "BUNDLE_VERSION_1");
  assert.throws(() => parseBundle(JSON.stringify({ ...base, version: 99 })), SchemaError);
  assert.throws(() => parseBundle(JSON.stringify({ ...base, files: "nope" })), SchemaError);
});

test("validateBundle rejects unsafe and duplicate paths, and invalid canvases", async () => {
  const vault = await populatedVault();
  const { bundle } = await exportBundle(vault);

  const unsafe = structuredClone(bundle);
  unsafe.files[0].path = "../evil.canvas";
  await assert.rejects(() => validateBundle(unsafe), PathError);

  const dup = structuredClone(bundle);
  dup.files.push({ ...dup.files[0] });
  await assert.rejects(() => validateBundle(dup), PathError);

  const badCanvas = structuredClone(bundle);
  badCanvas.files.find((f) => f.path === "canvases/root.canvas").text = JSON.stringify({ nodes: [{ bad: 1 }] });
  await assert.rejects(() => validateBundle(badCanvas), (e) => e.code === "CANVAS_INVALID");
});

test("validateBundle flags duplicate orbit-ids in staging", async () => {
  const vault = await populatedVault();
  const { bundle } = await exportBundle(vault);
  const dup = structuredClone(bundle);
  dup.files.push({ path: "tasks/copy--a1b2c3.md", mediaType: "text/markdown", text: serializeTask(sampleTask) });
  const summary = await validateBundle(dup);
  const diag = summary.diagnostics.find((d) => d.code === "DUPLICATE_ID");
  assert.ok(diag, "expected a DUPLICATE_ID diagnostic");
  assert.equal(diag.details.orbitId, "task-a1b2c3");
  assert.equal(diag.details.paths.length, 2);
});

test("validateBundle detects an orbit-id collision between a component card and a life entity", async () => {
  const vault = await populatedVault();
  const { bundle } = await exportBundle(vault);
  const duplicate = structuredClone(bundle);
  duplicate.files.push({
    path: "cards/task-id-copy.md",
    mediaType: "text/markdown",
    text: serializeComponentCard({ ...sampleCard, id: sampleTask.orbitId }),
  });

  const summary = await validateBundle(duplicate);
  const diagnostic = summary.diagnostics.find((item) => item.code === "DUPLICATE_ID" && item.details.orbitId === sampleTask.orbitId);
  assert.deepEqual(diagnostic.details.paths, [TASK_PATH, "cards/task-id-copy.md"]);
});

test("validateBundle diagnoses malformed component cards", async () => {
  const vault = await populatedVault();
  const { bundle } = await exportBundle(vault);
  const malformed = structuredClone(bundle);
  malformed.files.find((file) => file.path === CARD_PATH).text =
    "---\norbit-schema: 1\norbit-type: component-card\norbit-id: card-broken\ntitle: \"Broken\"\nrecipe: progress\nvalue: 12\nmaximum: 10\n---\n";

  const summary = await validateBundle(malformed);
  assert.ok(summary.diagnostics.some((item) => item.path === CARD_PATH && item.code === "ENTITY_MALFORMED"));
});

test("validateBundle detects duplicate orbit-ids claimed by malformed component cards", async () => {
  const vault = await populatedVault();
  const { bundle } = await exportBundle(vault);
  const malformedDuplicate = structuredClone(bundle);
  malformedDuplicate.files.push({
    path: "cards/malformed-duplicate.md",
    mediaType: "text/markdown",
    text: serializeComponentCard({
      ...sampleCard,
      recipe: "progress",
      value: 5,
      maximum: 10,
      label: undefined,
      progress: undefined,
      trend: undefined,
    }).replace("value: 5", "value: 11"),
  });

  const summary = await validateBundle(malformedDuplicate);
  const duplicate = summary.diagnostics.find((item) => item.code === "DUPLICATE_ID" && item.details.orbitId === sampleCard.id);
  assert.deepEqual(duplicate.details.paths, [CARD_PATH, "cards/malformed-duplicate.md"]);
  assert.ok(summary.diagnostics.some((item) => item.path === "cards/malformed-duplicate.md" && item.code === "ENTITY_MALFORMED"));
});

test("validateBundle rejects a canvas reference to a missing component card", async () => {
  const vault = await populatedVault();
  const { bundle } = await exportBundle(vault);
  const missing = structuredClone(bundle);
  const root = missing.files.find((file) => file.path === "canvases/root.canvas");
  const document = JSON.parse(root.text);
  document.nodes.push({ id: "missing-card", type: "file", file: "cards/missing.md", x: 0, y: 200, width: 360, height: 220 });
  root.text = JSON.stringify(document, null, 2) + "\n";

  await assert.rejects(() => validateBundle(missing), (error) => error.code === "CANVAS_FILE_REFERENCE");
});

test("validateBundle flags malformed entities without throwing", async () => {
  const vault = await populatedVault();
  const { bundle } = await exportBundle(vault);
  const bad = structuredClone(bundle);
  // orbit-type task but missing required title/status/created-at/updated-at.
  bad.files.push({ path: "tasks/broken--x.md", mediaType: "text/markdown", text: "---\norbit-schema: 1\norbit-type: task\norbit-id: task-broken\n---\n# Body only\n" });
  const summary = await validateBundle(bad);
  assert.ok(summary.diagnostics.some((d) => d.code === "ENTITY_MALFORMED"));
});

test("validateBundle flags a sidecar reference to a missing canvas", async () => {
  const vault = await populatedVault();
  const { bundle } = await exportBundle(vault);
  const missing = structuredClone(bundle);
  missing.files = missing.files.filter((f) => f.path !== "canvases/planning.canvas");
  await assert.rejects(() => validateBundle(missing), (e) => e.code === "CANVAS_FILE_REFERENCE");
});

test("staging import activation copies the canonical files into the reload vault", async () => {
  const source = await populatedVault();
  const { bundle } = await exportBundle(source);
  const staging = new MemoryVault();
  await importBundle(staging, serializeBundle(bundle));
  const canonical = new MemoryVault();
  await canonical.restore(await staging.snapshot());
  assert.equal(await canonical.read(SIDECAR_PATH), await staging.read(SIDECAR_PATH));
  assert.equal(await canonical.read(TASK_PATH), await staging.read(TASK_PATH));
  assert.deepEqual((await canonical.list("canvases/")).map((meta) => meta.path), (await staging.list("canvases/")).map((meta) => meta.path));
});

test("assertCompleteExport refuses to present skipped files as a complete backup", () => {
  assert.throws(() => assertCompleteExport([{ path: "notes/unreadable.md", code: "FILE_UNREADABLE" }]), (e) => e.code === "BACKUP_INCOMPLETE");
});

test("importBundle rejects a non-empty staging vault before writing", async () => {
  const source = await populatedVault();
  const { bundle } = await exportBundle(source);
  const staging = new MemoryVault();
  await staging.write("keep.md", "do not replace");
  await assert.rejects(() => importBundle(staging, serializeBundle(bundle)), (error) => error.code === "IMPORT_NOT_EMPTY");
  assert.equal(await staging.read("keep.md"), "do not replace");
});

test("importBundle writes files first and the sidecar last", async () => {
  const vault = await populatedVault();
  const { bundle } = await exportBundle(vault);
  const staging = new MemoryVault();
  await importBundle(staging, serializeBundle(bundle));
  assert.equal(await staging.exists(SIDECAR_PATH), true);
  assert.equal(await staging.exists("canvases/root.canvas"), true);
  assert.equal(await staging.exists(TASK_PATH), true);
});
