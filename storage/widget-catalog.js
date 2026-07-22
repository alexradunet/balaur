import { isCanvas } from "./canvas-validate.js";
import { caseFoldKey } from "./vault-path.js";
import { validateWidgetSource } from "../widgets/widget-policy.js";

export function isWidgetPath(path) {
  return typeof path === "string" && path.startsWith("widgets/") && path.endsWith(".html");
}

function diagnostic(path, code, message, details = null) {
  return Object.freeze({ path, code, message, ...(details ? { details: Object.freeze(details) } : {}) });
}

function freezeWidget(validated, path, hash, placements) {
  return Object.freeze({
    ...validated,
    path,
    hash,
    placements: Object.freeze(placements.map((placement) => Object.freeze({ ...placement }))),
  });
}

function fallbackWidget(path, source, message) {
  const titleMatch = String(source).match(/<title(?:\s[^>]*)?>([\s\S]*?)<\/title\s*>/i);
  const title = titleMatch?.[1]?.replace(/<[^>]*>/g, "").trim()
    || path.split("/").at(-1)?.replace(/\.html$/i, "")
    || "Live widget";
  return Object.freeze({ path, title, source: String(source), diagnostic: message });
}

export class WidgetCatalog {
  constructor({ vault }) {
    if (!vault) throw new TypeError("WidgetCatalog requires a vault");
    this.vault = vault;
    this._byPath = new Map();
    this._fallbackByPath = new Map();
    this._diagnostics = [];
  }

  async rebuild() {
    const [widgetFiles, canvasFiles] = await Promise.all([
      this.vault.list("widgets/"),
      this.vault.list("canvases/"),
    ]);
    const parsed = new Map();
    const fallbackByPath = new Map();
    const diagnostics = [];
    const folds = new Map();

    for (const meta of widgetFiles) {
      if (!isWidgetPath(meta.path)) continue;
      const fold = caseFoldKey(meta.path);
      if (folds.has(fold)) {
        const paths = [folds.get(fold), meta.path];
        diagnostics.push(diagnostic(paths[0], "WIDGET_PATH_CASE_COLLISION", `Case-folded widget path collision: ${paths.join(", ")}`, { paths }));
      } else {
        folds.set(fold, meta.path);
      }
      let source = "";
      try {
        source = await this.vault.read(meta.path);
        parsed.set(meta.path, { validated: validateWidgetSource(source), hash: meta.hash, placements: [] });
      } catch (error) {
        const message = `Malformed widget: ${error.code || error.message}`;
        diagnostics.push(diagnostic(meta.path, "WIDGET_MALFORMED", message, { errorCode: error.code || null }));
        fallbackByPath.set(meta.path, fallbackWidget(meta.path, source, message));
      }
    }

    const referenced = new Set();
    for (const meta of canvasFiles) {
      if (!meta.path.endsWith(".canvas")) continue;
      let document;
      try {
        document = JSON.parse(await this.vault.read(meta.path));
      } catch (error) {
        diagnostics.push(diagnostic(meta.path, "CANVAS_MALFORMED", `Malformed canvas: ${error.message}`));
        continue;
      }
      if (!isCanvas(document)) {
        diagnostics.push(diagnostic(meta.path, "CANVAS_MALFORMED", "Malformed canvas: strict JSON Canvas validation failed"));
        continue;
      }
      for (const node of document.nodes) {
        if (node.type !== "file" || !isWidgetPath(node.file)) continue;
        referenced.add(node.file);
        const entry = parsed.get(node.file);
        if (entry) entry.placements.push({ canvasPath: meta.path, nodeId: node.id });
        else if (!widgetFiles.some((file) => file.path === node.file)) {
          diagnostics.push(diagnostic(node.file, "WIDGET_FILE_MISSING", `Canvas references missing widget: ${node.file}`, { canvasPath: meta.path, nodeId: node.id }));
        }
      }
    }

    const byPath = new Map();
    for (const [path, entry] of parsed) {
      byPath.set(path, freezeWidget(entry.validated, path, entry.hash, entry.placements));
      if (!referenced.has(path)) diagnostics.push(diagnostic(path, "WIDGET_ORPHANED", `Widget has no canvas placement: ${path}`));
    }
    this._byPath = byPath;
    this._fallbackByPath = fallbackByPath;
    this._diagnostics = diagnostics;
    return { widgetCount: byPath.size, diagnostics: this.diagnostics() };
  }

  async reconcile(_paths = []) {
    return this.rebuild();
  }

  getByPath(path) {
    return this._byPath.get(path) || null;
  }

  getFallbackByPath(path) {
    return this._fallbackByPath.get(path) || null;
  }

  widgets() {
    return [...this._byPath.values()];
  }

  diagnostics() {
    return [...this._diagnostics];
  }
}
