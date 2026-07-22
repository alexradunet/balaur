import { test } from "node:test";
import assert from "node:assert/strict";

import {
  GENERATED_OPERATION_SPEC,
  describeGeneratedOperation,
  recoverGeneratedPlacementFailure,
  validateGeneratedOperation,
} from "./generated-operations.js";
import { ComponentCardCatalog } from "../storage/component-card-catalog.js";
import { parseComponentCard } from "../storage/component-card-codec.js";
import { FileComponentCardRepository } from "../storage/component-card-repository.js";
import { MemoryVault } from "../storage/memory-vault.js";
import { componentCardPath } from "../storage/vault-path.js";

const context = (overrides = {}) => ({
  canvasId: "root",
  canvasIds: new Set(["root", "other"]),
  cardIds: new Set(["card-existing"]),
  nodeIds: new Set(["node-existing"]),
  cards: new Map([["card-existing", {
    id: "card-existing",
    title: "Existing metric",
    recipe: "metric",
    value: "64%",
    label: "Deep work",
    progress: 0.64,
    trend: "flat",
    body: "Last week.",
    path: "cards/existing-metric--card-existing.md",
  }]]),
  ...overrides,
});

const metricCreate = (overrides = {}) => ({
  type: "component-card.create",
  card: {
    id: "card-new",
    title: "Weekly focus",
    recipe: "metric",
    fields: { value: "72%", label: "Deep-work target", progress: 0.72, trend: "up" },
    body: "Up from 64% last week.",
  },
  placement: { x: 240, y: 180, width: 360, height: 220, color: "5" },
  ...overrides,
});

const widgetCreate = (overrides = {}) => ({
  type: "widget.create",
  widget: {
    path: "widgets/focus-dial.html",
    title: "Focus dial",
    source: "<!doctype html><title>Focus dial</title><p>Ready</p>",
  },
  placement: { x: 240, y: 180, width: 420, height: 260, color: "5" },
  ...overrides,
});

const assertInvalid = (operation, pattern, operationContext = context()) => {
  assert.throws(() => validateGeneratedOperation(operation, operationContext), pattern);
};

test("normalizes typed create and update operations without mutating input", () => {
  const create = metricCreate({ card: { ...metricCreate().card, id: "card-weekly", path: "cards/weekly.md" } });
  const before = structuredClone(create);
  assert.deepEqual(validateGeneratedOperation(create, context()), {
    type: "component-card.create",
    card: {
      id: "card-weekly",
      path: "cards/weekly.md",
      title: "Weekly focus",
      recipe: "metric",
      fields: { value: "72%", label: "Deep-work target", progress: 0.72, trend: "up" },
      body: "Up from 64% last week.",
    },
    canvasId: "root",
    placement: { x: 240, y: 180, width: 360, height: 220, color: "5" },
  });
  assert.deepEqual(create, before);

  assert.deepEqual(validateGeneratedOperation({
    type: "component-card.update",
    id: "card-existing",
    patch: { fields: { value: "80%", progress: 0.8, trend: "up" }, body: "This week." },
    canvasId: "other",
    placement: { id: "node-new", x: -20, y: 40, width: 400, height: 240, color: "#aBc123" },
  }, context()), {
    type: "component-card.update",
    id: "card-existing",
    patch: { fields: { value: "80%", progress: 0.8, trend: "up" }, body: "This week." },
    canvasId: "other",
    placement: { id: "node-new", x: -20, y: 40, width: 400, height: 240, color: "#abc123" },
  });
});

test("rejects callable own properties before serialization can invoke toJSON", () => {
  let invoked = false;
  const operation = metricCreate();
  operation.toJSON = () => { invoked = true; return metricCreate(); };
  assertInvalid(operation, /plain JSON data/i);
  assert.equal(invoked, false);
});

test("accepts every declarative recipe and rejects unknown operation types and recipes", () => {
  const cards = [
    { title: "Metric", recipe: "metric", fields: { value: "7", progress: 0, trend: "down" } },
    { title: "Progress", recipe: "progress", fields: { value: 7, maximum: 10, unit: "hours" } },
    { title: "Callout", recipe: "callout", fields: { tone: "warning" }, body: "Watch this." },
    { title: "List", recipe: "list", fields: {}, body: "- One" },
    { title: "Timeline", recipe: "timeline", fields: {}, body: "## 2026-07-22" },
  ];
  for (const card of cards) assert.equal(validateGeneratedOperation(metricCreate({ card }), context()).card.recipe, card.recipe);
  assertInvalid({ type: "component-card.delete", id: "card-existing" }, /Unsupported generated operation/);
  assertInvalid(metricCreate({ card: { title: "Bad", recipe: "chart", fields: {} } }), /recipe/i);
  assertInvalid({ type: "widget.delete", path: "widgets/focus.html" }, /Unsupported generated operation/);
});

test("rejects unknown or host-code-shaped fields at every operation level", () => {
  assertInvalid({ ...metricCreate(), source: "document.body.innerHTML=''" }, /Unknown.*source/i);
  assertInvalid(metricCreate({ card: { ...metricCreate().card, html: "<script>host()</script>" } }), /Unknown.*html/i);
  assertInvalid(metricCreate({ card: { ...metricCreate().card, fields: { ...metricCreate().card.fields, onclick: "host()" } } }), /Unknown.*onclick/i);
  assertInvalid(metricCreate({ placement: { ...metricCreate().placement, rotate: 20 } }), /Unknown.*rotate/i);
  assertInvalid({ type: "component-card.update", id: "card-existing", patch: { id: "replacement" } }, /Unknown.*id/i);
  assertInvalid({ type: "component-card.update", id: "card-existing", patch: {} }, /must change/i);
});

test("validates widget.create source, path, placement, and capability disclosure", () => {
  const operation = widgetCreate();
  const before = structuredClone(operation);
  const normalized = validateGeneratedOperation(operation, context());
  assert.deepEqual(normalized, {
    ...operation,
    canvasId: "root",
  });
  assert.deepEqual(operation, before);
  assert.deepEqual(describeGeneratedOperation(normalized), {
    title: "Create live widget",
    summary: "Create “Focus dial” on canvas root",
    details: [
      "File: widgets/focus-dial.html",
      "Source: 52 UTF-8 bytes",
      "Capabilities: sandboxed scripts, inline styles, data/blob images and media",
      "Unavailable: host data/mutation, storage, network, forms, popups, navigation, workers, nested frames",
      "Self-navigation attempts pause the widget; hard request suppression is not claimed",
      "Placement: 420×260 at 240, 180 · color 5",
    ],
    source: operation.widget.source,
  });

  assertInvalid(widgetCreate({ widget: { ...operation.widget, path: "../escape.html" } }), /path|traversal/i);
  assertInvalid(widgetCreate({ widget: { ...operation.widget, path: "cards/focus.html" } }), /widgets\//i);
  assertInvalid(widgetCreate({ widget: { ...operation.widget, path: "widgets/focus.txt" } }), /\.html/i);
  assertInvalid(widgetCreate({ widget: { ...operation.widget, title: "Other title" } }), /title/i);
  assertInvalid(widgetCreate({ widget: { ...operation.widget, source: "<title>Focus dial</title><img src='https://example.invalid/x.png'>" } }), /external|resource|URL/i);
  assertInvalid(widgetCreate({ widget: { ...operation.widget, onclick: "host()" } }), /Unknown.*onclick/i);
  assertInvalid({ ...operation, repository: "host" }, /Unknown.*repository/i);
});

test("rejects prototype-bearing input recursively", () => {
  const inherited = Object.create({ source: "host()" });
  Object.assign(inherited, metricCreate());
  assertInvalid(inherited, /plain data object/i);

  const fields = Object.create({ onclick: "host()" });
  Object.assign(fields, metricCreate().card.fields);
  assertInvalid(metricCreate({ card: { ...metricCreate().card, fields } }), /plain data object/i);

  class Placement { constructor() { Object.assign(this, metricCreate().placement); } }
  assertInvalid(metricCreate({ placement: new Placement() }), /plain data object/i);
});

test("enforces identifier, duplicate, target Canvas, and safe path contracts", () => {
  assertInvalid(metricCreate({ card: { ...metricCreate().card, id: "../bad" } }), /orbit-id/i);
  assertInvalid(metricCreate({ card: { ...metricCreate().card, id: "card-existing" } }), /already exists/i);
  assertInvalid(metricCreate({ card: { ...metricCreate().card, path: "../escape.md" } }), /path/i);
  assertInvalid(metricCreate({ card: { ...metricCreate().card, path: "notes/not-a-card.md" } }), /cards\//i);
  assertInvalid(metricCreate({ card: { ...metricCreate().card, path: "cards/not-markdown.txt" } }), /\.md/i);
  assertInvalid(metricCreate({ card: { ...metricCreate().card, path: `cards/${"x".repeat(256)}.md` } }), /255 bytes/i);
  assertInvalid(metricCreate({ canvasId: "missing" }), /Unknown target canvas/i);
  assertInvalid(metricCreate({ placement: { ...metricCreate().placement, id: "node-existing" } }), /already exists/i);
  assertInvalid({ type: "component-card.update", id: "card-missing", patch: { title: "Nope" } }, /Unknown component card/i);
});

test("enforces all Unicode, UTF-8, and serialized operation byte bounds", () => {
  assert.equal(validateGeneratedOperation(metricCreate({ card: { ...metricCreate().card, title: "😀".repeat(160) } }), context()).card.title.length, 320);
  assertInvalid(metricCreate({ card: { ...metricCreate().card, title: "😀".repeat(161) } }), /title/i);
  for (const key of ["value", "label"]) {
    assert.equal(validateGeneratedOperation(metricCreate({ card: { ...metricCreate().card, fields: { ...metricCreate().card.fields, [key]: "😀".repeat(160) } } }), context()).card.fields[key].length, 320);
    assertInvalid(metricCreate({ card: { ...metricCreate().card, fields: { ...metricCreate().card.fields, [key]: "😀".repeat(161) } } }), new RegExp(key, "i"));
  }
  const progressCard = { title: "Progress", recipe: "progress", fields: { value: 1, maximum: 2, unit: "😀".repeat(32) } };
  assert.equal(validateGeneratedOperation(metricCreate({ card: progressCard }), context()).card.fields.unit.length, 64);
  assertInvalid(metricCreate({ card: { ...progressCard, fields: { ...progressCard.fields, unit: "😀".repeat(33) } } }), /unit/i);
  assert.equal(validateGeneratedOperation(metricCreate({ card: { ...metricCreate().card, body: "é".repeat(GENERATED_OPERATION_SPEC.maxBodyBytes / 2) } }), context()).card.body.length, GENERATED_OPERATION_SPEC.maxBodyBytes / 2);
  assertInvalid(metricCreate({ card: { ...metricCreate().card, body: "é".repeat(GENERATED_OPERATION_SPEC.maxBodyBytes / 2 + 1) } }), /32 KiB/i);

  const tooLarge = metricCreate({ padding: "x".repeat(GENERATED_OPERATION_SPEC.maxOperationBytes) });
  assertInvalid(tooLarge, /too large/i);
});

test("enforces each recipe field domain and cross-field invariant", () => {
  for (const progress of [-0.01, 1.01, Infinity, "0.5"]) assertInvalid(metricCreate({ card: { ...metricCreate().card, fields: { value: "x", progress } } }), /progress/i);
  assertInvalid(metricCreate({ card: { ...metricCreate().card, fields: { value: "x", trend: "sideways" } } }), /trend/i);
  assertInvalid(metricCreate({ card: { title: "Metric", recipe: "metric", fields: {} } }), /value/i);

  const progress = (fields) => metricCreate({ card: { title: "Progress", recipe: "progress", fields } });
  for (const fields of [{ value: -1, maximum: 2 }, { value: 3, maximum: 2 }, { value: 1, maximum: 0 }, { value: NaN, maximum: 2 }, { value: 1, maximum: Infinity }]) assertInvalid(progress(fields), /(value|maximum)/i);
  assertInvalid(metricCreate({ card: { title: "Callout", recipe: "callout", fields: { tone: "loud" } } }), /tone/i);
  assertInvalid(metricCreate({ card: { title: "List", recipe: "list", fields: { value: "wrong" } } }), /value/i);
});

test("enforces inclusive geometry bounds and standard JSON Canvas colors", () => {
  const { maxCoordinateMagnitude, maxDimension } = GENERATED_OPERATION_SPEC;
  for (const value of [-maxCoordinateMagnitude, maxCoordinateMagnitude]) {
    assert.equal(validateGeneratedOperation(metricCreate({ placement: { ...metricCreate().placement, x: value } }), context()).placement.x, value);
  }
  for (const value of [-maxCoordinateMagnitude - 1, maxCoordinateMagnitude + 1, 1.5]) assertInvalid(metricCreate({ placement: { ...metricCreate().placement, x: value } }), /geometry/i);
  assertInvalid(metricCreate({ placement: { ...metricCreate().placement, x: NaN } }), /plain JSON data/i);
  for (const value of [1, maxDimension]) assert.equal(validateGeneratedOperation(metricCreate({ placement: { ...metricCreate().placement, width: value } }), context()).placement.width, value);
  for (const value of [0, maxDimension + 1, 1.5]) assertInvalid(metricCreate({ placement: { ...metricCreate().placement, width: value } }), /geometry/i);
  assertInvalid(metricCreate({ placement: { ...metricCreate().placement, width: Infinity } }), /plain JSON data/i);
  for (const color of ["1", "6", "#abcdef", "#ABCDEF"]) assert.doesNotThrow(() => validateGeneratedOperation(metricCreate({ placement: { ...metricCreate().placement, color } }), context()));
  for (const color of ["0", "7", "red", "#abcd", "#gggggg", 5]) assertInvalid(metricCreate({ placement: { ...metricCreate().placement, color } }), /color/i);
});

test("validates update patches against the current recipe and permits placement-only updates", () => {
  assertInvalid({ type: "component-card.update", id: "card-existing", patch: { fields: { maximum: 10 } } }, /maximum/i);
  assertInvalid({ type: "component-card.update", id: "card-existing", patch: { recipe: "progress", fields: { value: 8 } } }, /maximum/i);
  assert.deepEqual(validateGeneratedOperation({
    type: "component-card.update",
    id: "card-existing",
    patch: { recipe: "progress", fields: { value: 8, maximum: 10, unit: "hours" } },
  }, context()).patch, {
    recipe: "progress",
    fields: { value: 8, label: null, progress: null, trend: null, maximum: 10, unit: "hours" },
  });
  const patchOnly = validateGeneratedOperation({
    type: "component-card.update",
    id: "card-existing",
    patch: { fields: { value: "81%" } },
  }, context());
  assert.equal(Object.hasOwn(patchOnly, "canvasId"), false);
  assert.deepEqual(validateGeneratedOperation({
    type: "component-card.update",
    id: "card-existing",
    canvasId: "other",
    placement: { x: 0, y: 0, width: 360, height: 220 },
  }, context()).patch, {});
});

test("descriptions are deterministic plain data for create and update", () => {
  const create = validateGeneratedOperation(metricCreate(), context());
  assert.deepEqual(describeGeneratedOperation(create), {
    title: "Create metric card",
    summary: "Create “Weekly focus” on canvas root",
    details: [
      "ID: card-new",
      `File: ${componentCardPath("Weekly focus", "card-new")}`,
      "Target canvas: root",
      "Recipe: metric",
      "Value: 72%",
      "Label: Deep-work target",
      "Progress: 72%",
      "Trend: up",
      "Placement: 360×220 at 240, 180 · color 5",
    ],
  });
  const update = validateGeneratedOperation({ type: "component-card.update", id: "card-existing", patch: { title: "Updated", fields: { value: "80%" } } }, context());
  assert.deepEqual(describeGeneratedOperation(update), {
    title: "Update component card",
    summary: "Update canonical card card-existing across all placements",
    details: ["Title: Updated", "Value: 80%"],
  });
  assert.equal(Object.getPrototypeOf(describeGeneratedOperation(create)), Object.prototype);
});

test("placement recovery drops the applied prefix and retains the untouched suffix", () => {
  const create = validateGeneratedOperation(metricCreate(), context());
  const suffix = { type: "theme.set", theme: "calm" };
  assert.deepEqual(recoverGeneratedPlacementFailure(
    [{ type: "theme.set", theme: "warm" }, create, suffix],
    1,
    { id: create.card.id, path: create.card.path, placed: false },
  ), [
    {
      type: "component-card.update",
      id: create.card.id,
      patch: {},
      canvasId: create.canvasId,
      placement: create.placement,
    },
    suffix,
  ]);

  const updateWithPlacement = validateGeneratedOperation({
    type: "component-card.update",
    id: "card-existing",
    patch: { fields: { value: "80%" } },
    canvasId: "other",
    placement: { id: "node-update-recovery", x: 0, y: 0, width: 360, height: 220 },
  }, context());
  assert.deepEqual(recoverGeneratedPlacementFailure(
    [updateWithPlacement, suffix],
    0,
    {
      id: "card-existing",
      path: "cards/existing-metric--card-existing.md",
      updated: true,
      placed: false,
      placement: { canvasId: "other", geometry: updateWithPlacement.placement },
    },
  ), [
    {
      type: "component-card.update",
      id: "card-existing",
      patch: {},
      canvasId: "other",
      placement: updateWithPlacement.placement,
    },
    suffix,
  ]);

  const widget = validateGeneratedOperation(widgetCreate({ canvasId: "other" }), context());
  assert.deepEqual(recoverGeneratedPlacementFailure(
    [widget, suffix],
    0,
    { path: widget.widget.path, title: widget.widget.title, placed: false },
  ), [
    {
      type: "widget.place",
      path: widget.widget.path,
      canvasId: "other",
      placement: widget.placement,
    },
    suffix,
  ]);
});

test("repository reports a recoverable unplaced file when Canvas placement fails", async () => {
  class CanvasFailingVault extends MemoryVault {
    failCanvasWrite = false;
    async write(path, content, options = {}) {
      if (this.failCanvasWrite && path === "canvases/root.canvas") throw new Error("simulated Canvas failure");
      return super.write(path, content, options);
    }
  }
  const vault = new CanvasFailingVault();
  await vault.write("canvases/root.canvas", JSON.stringify({ nodes: [], edges: [] }));
  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileComponentCardRepository({ vault, catalog, canvasPathFromId: () => "canvases/root.canvas" });
  vault.failCanvasWrite = true;

  let failure;
  try {
    await repository.createCard({
      id: "card-recoverable",
      title: "Recoverable",
      recipe: "metric",
      fields: { value: "1" },
      canvasId: "root",
      geometry: { id: "node-recoverable", x: 0, y: 0, width: 360, height: 220 },
    });
  } catch (error) {
    failure = error;
  }

  assert.equal(failure?.code, "CARD_PLACEMENT_FAILED");
  assert.deepEqual(failure?.details?.recoverable, {
    id: "card-recoverable",
    path: componentCardPath("Recoverable", "card-recoverable"),
    placed: false,
  });
  assert.equal(await vault.exists(failure.details.recoverable.path), true);
  assert.equal(catalog.getById("card-recoverable").placements.length, 0);
  assert.deepEqual(catalog.cards().map((card) => card.id), ["card-recoverable"]);
  assert.deepEqual(JSON.parse(await vault.read("canvases/root.canvas")).nodes, []);
  vault.failCanvasWrite = false;
  await repository.addPlacement("card-recoverable", "root", {
    id: "node-recoverable",
    x: 0,
    y: 0,
    width: 360,
    height: 220,
  });
  assert.equal((await vault.list("cards/")).length, 1);
  assert.equal(catalog.getById("card-recoverable").placements.length, 1);
});

test("repository reports a durable update and retries only its failed placement", async () => {
  class UpdatePlacementFailingVault extends MemoryVault {
    failCanvasWrite = false;
    async write(path, content, options = {}) {
      if (this.failCanvasWrite && path === "canvases/root.canvas") throw new Error("simulated update placement failure");
      return super.write(path, content, options);
    }
  }
  const vault = new UpdatePlacementFailingVault();
  await vault.write("canvases/root.canvas", JSON.stringify({ nodes: [], edges: [] }));
  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileComponentCardRepository({ vault, catalog, canvasPathFromId: () => "canvases/root.canvas" });
  const created = await repository.createCard({
    id: "card-updated-recoverable",
    title: "Updated recoverable",
    recipe: "metric",
    fields: { value: "1" },
  });
  const geometry = { id: "node-updated-recoverable", x: 10, y: 20, width: 360, height: 220, color: "5" };
  vault.failCanvasWrite = true;

  let failure;
  try {
    await repository.updateCardAndPlace(created.id, { value: "2" }, "root", geometry);
  } catch (error) {
    failure = error;
  }

  assert.equal(failure?.code, "CARD_PLACEMENT_FAILED");
  assert.deepEqual(failure?.details?.recoverable, {
    id: created.id,
    path: created.path,
    updated: true,
    placed: false,
    placement: { canvasId: "root", geometry },
  });
  assert.equal(parseComponentCard(await vault.read(created.path)).value, "2");
  assert.equal(catalog.getById(created.id).placements.length, 0);
  assert.equal((await vault.list("cards/")).length, 1);

  vault.failCanvasWrite = false;
  await repository.updateCardAndPlace(created.id, {}, "root", geometry);
  assert.equal(parseComponentCard(await vault.read(created.path)).value, "2");
  assert.equal(catalog.getById(created.id).placements.length, 1);
  assert.equal((await vault.list("cards/")).length, 1);
});
