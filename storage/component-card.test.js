import { test } from "node:test";
import assert from "node:assert/strict";

import {
  COMPONENT_CARD_SPEC,
  parseComponentCard,
  patchComponentCard,
  serializeComponentCard,
} from "./component-card-codec.js";
import { ComponentCardCatalog } from "./component-card-catalog.js";
import { FileComponentCardRepository } from "./component-card-repository.js";
import { MemoryVault } from "./memory-vault.js";
import { ConflictError, SchemaError } from "./vault-errors.js";
import { componentCardPath } from "./vault-path.js";

const canvas = (nodes = [], edges = []) => JSON.stringify({ nodes, edges }, null, 2) + "\n";
const fileNode = (id, file, x = 0) => ({ id, type: "file", file, x, y: 0, width: 360, height: 220 });
const baseCard = (overrides = {}) => ({
  id: "card-a1b2c3",
  title: "Weekly focus",
  recipe: "metric",
  value: "72%",
  label: "Deep-work target",
  progress: 0.72,
  trend: "up",
  body: "Up from 64% last week.",
  ...overrides,
});

function assertCode(code) {
  return (error) => error instanceof SchemaError && error.code === code;
}

test("codec serializes and parses every approved recipe", () => {
  const recipes = [
    baseCard(),
    baseCard({ id: "card-progress", recipe: "progress", value: 7, maximum: 10, unit: "hours", label: undefined, progress: undefined, trend: undefined }),
    baseCard({ id: "card-callout", recipe: "callout", tone: "warning", value: undefined, label: undefined, progress: undefined, trend: undefined }),
    baseCard({ id: "card-list", recipe: "list", value: undefined, label: undefined, progress: undefined, trend: undefined, body: "- One\n- Two" }),
    baseCard({ id: "card-timeline", recipe: "timeline", value: undefined, label: undefined, progress: undefined, trend: undefined, body: "## 2026-07-22\nStarted" }),
  ];

  for (const input of recipes) {
    const text = serializeComponentCard(input);
    const parsed = parseComponentCard(text, { path: `cards/${input.id}.md` });
    assert.equal(parsed.schema, 1);
    assert.equal(parsed.type, "component-card");
    assert.equal(parsed.id, input.id);
    assert.equal(parsed.title, input.title);
    assert.equal(parsed.recipe, input.recipe);
    assert.equal(parsed.body, input.body);
    assert.equal(parsed.path, `cards/${input.id}.md`);
    for (const key of ["value", "label", "progress", "trend", "maximum", "unit", "tone"]) {
      assert.equal(parsed[key], input[key] ?? null, `${input.recipe}.${key}`);
    }
  }
});

test("codec enforces common Unicode and byte limits", () => {
  assert.equal(parseComponentCard(serializeComponentCard(baseCard({ title: "😀".repeat(160) }))).title.length, 320);
  assert.throws(() => serializeComponentCard(baseCard({ title: "😀".repeat(161) })), assertCode("CARD_TITLE_INVALID"));
  assert.throws(() => serializeComponentCard(baseCard({ title: "" })), assertCode("CARD_TITLE_INVALID"));
  assert.throws(() => serializeComponentCard(baseCard({ id: "../unsafe" })), assertCode("CARD_ID_INVALID"));
  assert.throws(() => serializeComponentCard(baseCard({ body: "a".repeat(COMPONENT_CARD_SPEC.maxBodyBytes + 1) })), assertCode("CARD_BODY_TOO_LARGE"));
  assert.doesNotThrow(() => serializeComponentCard(baseCard({ body: "a".repeat(COMPONENT_CARD_SPEC.maxBodyBytes) })));

  const oversized = serializeComponentCard(baseCard()).replace("\n---\n\nUp", `\npadding: "${"x".repeat(COMPONENT_CARD_SPEC.maxFileBytes)}"\n---\n\nUp`);
  assert.throws(() => parseComponentCard(oversized), assertCode("CARD_FILE_TOO_LARGE"));
});

test("codec enforces every recipe field contract", () => {
  for (const key of ["value", "label"]) {
    assert.throws(() => serializeComponentCard(baseCard({ [key]: "x".repeat(161) })), assertCode("CARD_FIELD_INVALID"));
  }
  for (const progress of [-0.01, 1.01, Infinity, "0.5"]) {
    assert.throws(() => serializeComponentCard(baseCard({ progress })), assertCode("CARD_FIELD_INVALID"));
  }
  assert.throws(() => serializeComponentCard(baseCard({ trend: "sideways" })), assertCode("CARD_FIELD_INVALID"));
  assert.throws(() => serializeComponentCard(baseCard({ recipe: "metric", value: undefined })), assertCode("CARD_FIELD_REQUIRED"));

  const progress = (patch) => baseCard({ recipe: "progress", value: 5, maximum: 10, label: undefined, progress: undefined, trend: undefined, ...patch });
  for (const bad of [{ value: -1 }, { value: 11 }, { value: NaN }, { maximum: 0 }, { maximum: Infinity }, { unit: "x".repeat(33) }]) {
    assert.throws(() => serializeComponentCard(progress(bad)), assertCode("CARD_FIELD_INVALID"));
  }
  assert.throws(() => serializeComponentCard(progress({ maximum: undefined })), assertCode("CARD_FIELD_REQUIRED"));

  assert.throws(() => serializeComponentCard(baseCard({ recipe: "callout", value: undefined, label: undefined, progress: undefined, trend: undefined, tone: "loud" })), assertCode("CARD_FIELD_INVALID"));
  assert.throws(() => serializeComponentCard(baseCard({ recipe: "unknown" })), assertCode("CARD_RECIPE_INVALID"));
});

test("codec rejects fields that do not belong to the selected recipe", () => {
  const invalid = [
    baseCard({ maximum: 10 }),
    baseCard({ recipe: "progress", value: 5, maximum: 10, label: undefined, progress: undefined, trend: "up" }),
    baseCard({ recipe: "callout", value: "unexpected", label: undefined, progress: undefined, trend: undefined }),
    baseCard({ recipe: "list", value: undefined, label: undefined, progress: undefined, trend: undefined, tone: "info" }),
    baseCard({ recipe: "timeline", value: undefined, label: undefined, progress: undefined, trend: undefined, unit: "days" }),
  ];

  for (const card of invalid) {
    assert.throws(() => serializeComponentCard(card), assertCode("CARD_FIELD_INVALID"), card.recipe);
  }
});

test("codec rejects malformed, missing, duplicate, and wrong-type frontmatter", () => {
  assert.throws(() => parseComponentCard("# no frontmatter"), (error) => error.code === "FM_NO_DELIMITER");
  assert.throws(() => parseComponentCard("---\norbit-schema: 1\norbit-type: component-card\n---\n"), assertCode("CARD_FIELD_REQUIRED"));
  assert.throws(() => parseComponentCard(serializeComponentCard(baseCard()).replace("recipe: metric\n", "recipe: metric\nrecipe: list\n")), (error) => error.code === "FM_DUPLICATE_KEY");
  assert.throws(() => parseComponentCard(serializeComponentCard(baseCard()).replace("orbit-type: component-card", "orbit-type: task")), assertCode("CARD_TYPE_MISMATCH"));
  assert.throws(() => parseComponentCard(serializeComponentCard(baseCard()).replace("orbit-schema: 1", "orbit-schema: 2")), assertCode("SCHEMA_NEWER"));
});

test("patching preserves unknown keys, comments, ordering, BOM, CRLF, and body bytes", () => {
  const original = "\uFEFF---\r\n# owner note\r\norbit-schema: 1\r\norbit-type: component-card\r\norbit-id: \"card-preserve\"\r\ncustom-key: keep\r\ntitle: \"Before\"\r\nrecipe: callout\r\ntone: info\r\n---\r\n\r\nBody  \r\nbytes\r\n";
  const patched = patchComponentCard(original, { title: "After", tone: "success" });
  assert.ok(patched.startsWith("\uFEFF---\r\n# owner note\r\n"));
  assert.ok(patched.includes("custom-key: keep\r\n"));
  assert.ok(patched.includes('title: "After"\r\n'));
  assert.ok(patched.endsWith("\r\nBody  \r\nbytes\r\n"));
  assert.equal(parseComponentCard(patched).tone, "success");

  const bodyPatched = patchComponentCard(patched, { body: "Replacement\r\n" });
  assert.equal(parseComponentCard(bodyPatched).body, "Replacement\r\n");
  assert.throws(() => patchComponentCard(original, { id: "card-other" }), assertCode("CARD_ID_IMMUTABLE"));
  assert.throws(() => patchComponentCard(original, { arbitrary: true }), assertCode("CARD_UNKNOWN_FIELD"));
});

test("patching transitions metric values to progress numbers without rewriting preserved bytes", () => {
  const original = "\uFEFF---\r\norbit-schema: 1\r\norbit-type: component-card\r\norbit-id: \"card-transition\"\r\ncustom-key: keep\r\ntitle: \"Transition\"\r\nrecipe: metric\r\nvalue: \"72%\"\r\nlabel: \"Target\"\r\nprogress: 0.72\r\ntrend: up\r\n---\r\n\r\nBody  \r\n";
  const patched = patchComponentCard(original, {
    recipe: "progress",
    value: 7,
    maximum: 10,
    unit: "hours",
    label: null,
    progress: null,
    trend: null,
  });

  const card = parseComponentCard(patched);
  assert.equal(card.recipe, "progress");
  assert.equal(card.value, 7);
  assert.equal(card.maximum, 10);
  assert.ok(patched.startsWith("\uFEFF---\r\n"));
  assert.ok(patched.includes("custom-key: keep\r\n"));
  assert.ok(patched.endsWith("\r\nBody  \r\n"));
});

test("patching transitions progress numbers to metric values without rewriting preserved bytes", () => {
  const original = "\uFEFF---\r\norbit-schema: 1\r\norbit-type: component-card\r\norbit-id: \"card-transition-back\"\r\ncustom-key: keep\r\ntitle: \"Transition\"\r\nrecipe: progress\r\nvalue: 7\r\nmaximum: 10\r\nunit: \"hours\"\r\n---\r\n\r\nBody  \r\n";
  const patched = patchComponentCard(original, {
    recipe: "metric",
    value: "72%",
    label: "Target",
    progress: 0.72,
    trend: "up",
    maximum: null,
    unit: null,
  });

  const card = parseComponentCard(patched);
  assert.equal(card.recipe, "metric");
  assert.equal(card.value, "72%");
  assert.equal(card.progress, 0.72);
  assert.ok(patched.startsWith("\uFEFF---\r\n"));
  assert.ok(patched.includes("custom-key: keep\r\n"));
  assert.ok(patched.endsWith("\r\nBody  \r\n"));
});

test("componentCardPath generates a safe stable cards path", () => {
  assert.equal(componentCardPath("Weekly: Focus?", "card-a1b2c3"), "cards/weekly-focus--a1b2c3.md");
  assert.match(componentCardPath("😀".repeat(300), "card-safe-123456"), /^cards\/.+--123456\.md$/);
});

test("catalog rebuild reports malformed, duplicate, missing, and orphaned cards", async () => {
  const vault = new MemoryVault();
  const placedPath = "cards/placed.md";
  const duplicatePath = "cards/duplicate.md";
  const orphanPath = "cards/orphan.md";
  await vault.write(placedPath, serializeComponentCard(baseCard()));
  await vault.write(duplicatePath, serializeComponentCard(baseCard({ title: "Duplicate" })));
  await vault.write(orphanPath, serializeComponentCard(baseCard({ id: "card-orphan", title: "Orphan" })));
  await vault.write("cards/broken.md", "---\norbit-schema: 1\norbit-type: component-card\n---\n");
  await vault.write("canvases/root.canvas", canvas([
    fileNode("placed", placedPath),
    fileNode("malformed", "cards/broken.md", 400),
    fileNode("missing", "cards/missing.md", 800),
  ]));

  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();

  assert.equal(catalog.getByPath(placedPath).placements.length, 1);
  assert.equal(catalog.getById("card-a1b2c3"), null, "duplicate identity is not arbitrarily resolved");
  const codes = catalog.diagnostics().map((diagnostic) => diagnostic.code);
  for (const code of ["CARD_MALFORMED", "DUPLICATE_ID", "CARD_FILE_MISSING", "CARD_ORPHANED"]) assert.ok(codes.includes(code), code);
  assert.ok(catalog.diagnostics().find((diagnostic) => diagnostic.code === "DUPLICATE_ID").details.paths.includes(duplicatePath));
});

test("catalog suppresses a valid card when a malformed card claims the same orbit-id", async () => {
  const vault = new MemoryVault();
  const validPath = "cards/valid.md";
  const malformedPath = "cards/malformed.md";
  await vault.write(validPath, serializeComponentCard(baseCard({ id: "card-shared" })));
  await vault.write(
    malformedPath,
    serializeComponentCard(baseCard({ id: "card-shared", recipe: "progress", value: 5, maximum: 10, label: undefined, progress: undefined, trend: undefined }))
      .replace("value: 5", "value: 11"),
  );
  await vault.write("canvases/root.canvas", canvas([fileNode("valid", validPath)]));

  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();

  assert.equal(catalog.getById("card-shared"), null);
  const duplicate = catalog.diagnostics().find((item) => item.code === "DUPLICATE_ID");
  assert.deepEqual(duplicate.details.paths, [malformedPath, validPath]);
  assert.ok(catalog.diagnostics().some((item) => item.path === malformedPath && item.code === "CARD_MALFORMED"));
});

test("catalog reconcile handles create, modify, move, and removal paths", async () => {
  const vault = new MemoryVault();
  const oldPath = "cards/old.md";
  const newPath = "cards/new.md";
  await vault.write(oldPath, serializeComponentCard(baseCard({ id: "card-move", title: "Old" })));
  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();
  const oldHash = catalog.getByPath(oldPath).hash;

  await vault.move(oldPath, newPath, { expectedHash: oldHash });
  await catalog.reconcile([oldPath, newPath]);
  assert.equal(catalog.getByPath(oldPath), null);
  assert.equal(catalog.getById("card-move").path, newPath);

  const hash = catalog.getByPath(newPath).hash;
  await vault.write(newPath, patchComponentCard(await vault.read(newPath), { title: "New" }), { expectedHash: hash });
  await catalog.reconcile([newPath]);
  assert.equal(catalog.getByPath(newPath).title, "New");

  await vault.remove(newPath, { expectedHash: catalog.getByPath(newPath).hash });
  await catalog.reconcile([newPath]);
  assert.equal(catalog.getById("card-move"), null);
});

test("repository writes canonical files first, moves on title update, and honors conflicts", async () => {
  const vault = new MemoryVault();
  await vault.write("canvases/root.canvas", canvas());
  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileComponentCardRepository({
    vault,
    catalog,
    canvasPathFromId: (id) => id === "root" ? "canvases/root.canvas" : null,
    now: () => "2026-07-22T00:00:00.000Z",
  });

  const created = await repository.createCard({ ...baseCard({ id: "card-create" }), canvasId: "root", geometry: { id: "placement-one", x: 20, y: 30, width: 400, height: 240, color: "5" } });
  assert.equal(created.path, componentCardPath("Weekly focus", "card-create"));
  assert.equal(parseComponentCard(await vault.read(created.path)).id, "card-create");
  assert.deepEqual(JSON.parse(await vault.read("canvases/root.canvas")).nodes, [fileNode("placement-one", created.path, 20)].map((node) => ({ ...node, y: 30, width: 400, height: 240, color: "5" })));
  const duplicatePath = componentCardPath("Duplicate identity", "card-create");
  await assert.rejects(
    () => repository.createCard(baseCard({ id: "card-create", title: "Duplicate identity" })),
    assertCode("CARD_ID_DUPLICATE"),
  );
  assert.equal(await vault.exists(duplicatePath), false);

  const updated = await repository.updateCard("card-create", { title: "Renamed card", progress: 0.8 });
  assert.equal(updated.path, componentCardPath("Renamed card", "card-create"));
  assert.equal(await vault.exists(created.path), false);
  assert.equal(JSON.parse(await vault.read("canvases/root.canvas")).nodes[0].file, updated.path);
  assert.equal(catalog.getById("card-create").progress, 0.8);

  const staleHash = catalog.getById("card-create").hash;
  await vault.write(updated.path, patchComponentCard(await vault.read(updated.path), { label: "External" }), { expectedHash: staleHash });
  await assert.rejects(() => repository.updateCard("card-create", { progress: 0.9 }), (error) => error instanceof ConflictError && error.code === "WRITE_CONFLICT");
  assert.equal(parseComponentCard(await vault.read(updated.path)).progress, 0.8);
});

test("title moves update known placements without being blocked by unrelated malformed canvases", async () => {
  const vault = new MemoryVault();
  await vault.write("canvases/root.canvas", canvas());
  await vault.write("canvases/unrelated.canvas", "{not json");
  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileComponentCardRepository({
    vault,
    catalog,
    canvasPathFromId: (id) => id === "root" ? "canvases/root.canvas" : null,
  });
  const created = await repository.createCard({
    ...baseCard({ id: "card-move-focused" }),
    canvasId: "root",
    geometry: { id: "placement" },
  });

  const updated = await repository.updateCard(created.id, { title: "Focused move" });

  assert.equal(updated.path, componentCardPath("Focused move", created.id));
  assert.equal(JSON.parse(await vault.read("canvases/root.canvas")).nodes[0].file, updated.path);
  assert.equal(await vault.read("canvases/unrelated.canvas"), "{not json");
  assert.ok(catalog.diagnostics().some((item) => item.path === "canvases/unrelated.canvas" && item.code === "CANVAS_MALFORMED"));
});

test("title moves leave the source and placements unchanged when the destination exists", async () => {
  const vault = new MemoryVault();
  await vault.write("canvases/root.canvas", canvas());
  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileComponentCardRepository({
    vault,
    catalog,
    canvasPathFromId: () => "canvases/root.canvas",
  });
  const created = await repository.createCard({
    ...baseCard({ id: "card-destination-conflict" }),
    canvasId: "root",
    geometry: { id: "placement" },
  });
  const destination = componentCardPath("Reserved title", created.id);
  await vault.write(destination, serializeComponentCard(baseCard({
    id: "card-destination-owner",
    title: "Destination owner",
  })));

  await assert.rejects(
    () => repository.updateCard(created.id, { title: "Reserved title" }),
    (error) => error instanceof ConflictError && error.code === "WRITE_CONFLICT",
  );

  assert.equal(parseComponentCard(await vault.read(created.path)).title, "Weekly focus");
  assert.equal(parseComponentCard(await vault.read(destination)).id, "card-destination-owner");
  assert.equal(JSON.parse(await vault.read("canvases/root.canvas")).nodes[0].file, created.path);
  assert.equal(catalog.getById(created.id).path, created.path);
  assert.equal(catalog.getById("card-destination-owner").path, destination);
});

test("title moves retain both files and reconcile partial placements after a Canvas conflict", async () => {
  class CanvasRacingVault extends MemoryVault {
    armed = false;
    fired = false;

    async write(path, content, options = {}) {
      if (this.armed && !this.fired && path === "canvases/b.canvas" && options.expectedHash) {
        this.fired = true;
        const external = (await super.read(path)).replace(/\n$/, " \n");
        await super.write(path, external, { expectedHash: options.expectedHash });
      }
      return super.write(path, content, options);
    }
  }

  const vault = new CanvasRacingVault();
  await vault.write("canvases/a.canvas", canvas());
  await vault.write("canvases/b.canvas", canvas());
  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileComponentCardRepository({
    vault,
    catalog,
    canvasPathFromId: (id) => `canvases/${id}.canvas`,
  });
  const created = await repository.createCard(baseCard({ id: "card-canvas-conflict" }));
  await repository.addPlacement(created.id, "a", { id: "place-a" });
  await repository.addPlacement(created.id, "b", { id: "place-b" });
  const destination = componentCardPath("Partially moved", created.id);
  vault.armed = true;

  await assert.rejects(
    () => repository.updateCard(created.id, { title: "Partially moved" }),
    (error) => error instanceof ConflictError && error.code === "WRITE_CONFLICT",
  );

  assert.equal(await vault.exists(created.path), true);
  assert.equal(await vault.exists(destination), true);
  const placementPaths = [];
  for (const path of ["canvases/a.canvas", "canvases/b.canvas"]) {
    placementPaths.push(JSON.parse(await vault.read(path)).nodes[0].file);
  }
  assert.deepEqual(placementPaths, [destination, created.path]);
  for (const path of placementPaths) assert.equal(await vault.exists(path), true, path);
  assert.equal(catalog.getByPath(created.path).placements.length, 1);
  assert.equal(catalog.getByPath(destination).placements.length, 1);
  assert.equal(catalog.getById(created.id), null);
  assert.ok(catalog.diagnostics().some((item) => item.code === "DUPLICATE_ID" && item.details.orbitId === created.id));
});

test("repository supports multiple strict placements, removes one, then deletes everywhere", async () => {
  const vault = new MemoryVault();
  await vault.write("canvases/a.canvas", canvas());
  await vault.write("canvases/b.canvas", canvas());
  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileComponentCardRepository({ vault, catalog, canvasPathFromId: (id) => `canvases/${id}.canvas` });
  const { id, path } = await repository.createCard(baseCard({ id: "card-places" }));

  await repository.addPlacement(id, "a", { id: "place-a", x: 1, y: 2, width: 300, height: 180, color: "#abcdef" });
  await repository.addPlacement(id, "b", { id: "place-b", x: 3, y: 4, width: 320, height: 200, color: "6" });
  assert.equal(catalog.getById(id).placements.length, 2);
  await assert.rejects(() => repository.addPlacement(id, "a", { id: "bad", width: 0 }), assertCode("CANVAS_GEOMETRY_INVALID"));
  await assert.rejects(() => repository.addPlacement(id, "a", { id: "bad", color: "red" }), assertCode("CANVAS_INVALID"));

  assert.equal((await repository.removePlacement("a", "place-a")).removed, true);
  assert.equal(await vault.exists(path), true);
  assert.equal(catalog.getById(id).placements.length, 1);

  const result = await repository.deleteCard(id);
  assert.equal(result.removedPlacements, 1);
  assert.equal(await vault.exists(path), false);
  assert.deepEqual(JSON.parse(await vault.read("canvases/b.canvas")).nodes, []);
  assert.equal(catalog.getById(id), null);
});

test("delete-everywhere keeps the canonical card when a canvas conflicts", async () => {
  class RacingVault extends MemoryVault {
    armed = false;
    canvasReads = 0;

    async read(path) {
      const content = await super.read(path);
      if (this.armed && path === "canvases/a.canvas" && ++this.canvasReads === 2) {
        const stat = await this.stat(path);
        await this.write(path, content.replace("\n", " \n"), { expectedHash: stat.hash });
      }
      return content;
    }
  }
  const vault = new RacingVault();
  await vault.write("canvases/a.canvas", canvas());
  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileComponentCardRepository({ vault, catalog, canvasPathFromId: () => "canvases/a.canvas" });
  const { id, path } = await repository.createCard({ ...baseCard({ id: "card-delete-conflict" }), canvasId: "a", geometry: { id: "placement" } });
  vault.armed = true;

  await assert.rejects(() => repository.deleteCard(id), (error) => error instanceof ConflictError && error.code === "WRITE_CONFLICT");
  assert.equal(await vault.exists(path), true);
});

test("delete-everywhere uses the catalog's last-known card hash", async () => {
  const vault = new MemoryVault();
  await vault.write("canvases/a.canvas", canvas());
  const catalog = new ComponentCardCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileComponentCardRepository({ vault, catalog, canvasPathFromId: () => "canvases/a.canvas" });
  const { id, path } = await repository.createCard({
    ...baseCard({ id: "card-delete-file-conflict" }),
    canvasId: "a",
    geometry: { id: "placement" },
  });
  const staleHash = catalog.getById(id).hash;
  await vault.write(path, patchComponentCard(await vault.read(path), { label: "External change" }), { expectedHash: staleHash });

  await assert.rejects(() => repository.deleteCard(id), (error) => error instanceof ConflictError && error.code === "WRITE_CONFLICT");

  assert.equal(parseComponentCard(await vault.read(path)).label, "External change");
  assert.equal(JSON.parse(await vault.read("canvases/a.canvas")).nodes[0].id, "placement");
});
