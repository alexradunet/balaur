import { test } from "node:test";
import assert from "node:assert/strict";

import { WidgetCatalog } from "./widget-catalog.js";
import { FileWidgetRepository } from "./widget-repository.js";
import { MemoryVault } from "./memory-vault.js";

const source = (title = "Focus dial") => `<!doctype html><title>${title}</title><p>Ready</p>`;
const canvas = (nodes = []) => JSON.stringify({ nodes, edges: [] }, null, 2) + "\n";

test("widget catalog validates and caches canonical HTML without render-time reads", async () => {
  const vault = new MemoryVault();
  await vault.write("widgets/focus.html", source());
  await vault.write("widgets/hostile.html", "<title>Hostile</title><img src='https://example.invalid/x.png'>");
  const catalog = new WidgetCatalog({ vault });

  await catalog.rebuild();

  assert.equal(catalog.getByPath("widgets/focus.html").title, "Focus dial");
  assert.equal(catalog.getByPath("widgets/focus.html").source, source());
  assert.equal(catalog.getByPath("widgets/hostile.html"), null);
  assert.match(catalog.getFallbackByPath("widgets/hostile.html").diagnostic, /external|resource|URL/i);
  assert.equal(catalog.getFallbackByPath("widgets/hostile.html").source.includes("example.invalid"), true);
  assert.equal(catalog.widgets().length, 1);
});

test("widget repository writes validated source then places a standard file node", async () => {
  const vault = new MemoryVault();
  await vault.write("canvases/root.canvas", canvas());
  const catalog = new WidgetCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileWidgetRepository({
    vault,
    catalog,
    canvasPathFromId: (id) => id === "root" ? "canvases/root.canvas" : null,
  });

  const created = await repository.createWidget({
    path: "widgets/focus.html",
    title: "Focus dial",
    source: source(),
    canvasId: "root",
    geometry: { id: "widget-node", x: 20, y: 30, width: 420, height: 260, color: "5" },
  });

  assert.equal(created.path, "widgets/focus.html");
  assert.equal(await vault.read(created.path), source());
  assert.deepEqual(JSON.parse(await vault.read("canvases/root.canvas")).nodes, [{
    id: "widget-node",
    type: "file",
    file: "widgets/focus.html",
    x: 20,
    y: 30,
    width: 420,
    height: 260,
    color: "5",
  }]);
  assert.equal(catalog.getByPath(created.path).title, "Focus dial");
});

test("widget repository rejects unsafe paths, source-title mismatches, and duplicates before writes", async () => {
  const vault = new MemoryVault();
  const catalog = new WidgetCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileWidgetRepository({ vault, catalog });

  await assert.rejects(() => repository.createWidget({ path: "../escape.html", title: "Focus dial", source: source() }), /path|traversal/i);
  await assert.rejects(() => repository.createWidget({ path: "notes/focus.html", title: "Focus dial", source: source() }), /widgets\//i);
  await assert.rejects(() => repository.createWidget({ path: "widgets/focus.html", title: "Other", source: source() }), /title/i);
  assert.equal(await vault.exists("widgets/focus.html"), false);

  await repository.createWidget({ path: "widgets/focus.html", title: "Focus dial", source: source() });
  await assert.rejects(() => repository.createWidget({ path: "widgets/focus.html", title: "Focus dial", source: source() }), /already exists|conflict/i);
});

test("widget repository preserves a recoverable file when Canvas placement fails", async () => {
  class FailingVault extends MemoryVault {
    failCanvas = false;
    async write(path, content, options = {}) {
      if (this.failCanvas && path === "canvases/root.canvas") throw new Error("simulated Canvas failure");
      return super.write(path, content, options);
    }
  }
  const vault = new FailingVault();
  await vault.write("canvases/root.canvas", canvas());
  const catalog = new WidgetCatalog({ vault });
  await catalog.rebuild();
  const repository = new FileWidgetRepository({ vault, catalog, canvasPathFromId: () => "canvases/root.canvas" });
  vault.failCanvas = true;

  await assert.rejects(() => repository.createWidget({
    path: "widgets/recoverable.html",
    title: "Recoverable",
    source: source("Recoverable"),
    canvasId: "root",
    geometry: { id: "widget-node" },
  }), (error) => error.code === "WIDGET_PLACEMENT_FAILED"
    && error.details?.recoverable?.path === "widgets/recoverable.html");
  assert.equal(await vault.exists("widgets/recoverable.html"), true);
  assert.deepEqual(JSON.parse(await vault.read("canvases/root.canvas")).nodes, []);
});
