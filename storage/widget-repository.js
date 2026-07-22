import { isCanvas } from "./canvas-validate.js";
import { assertSafePath } from "./vault-path.js";
import { SchemaError } from "./vault-errors.js";
import { validateWidgetSource } from "../widgets/widget-policy.js";
import { isWidgetPath } from "./widget-catalog.js";

const DEFAULT_GEOMETRY = Object.freeze({ x: 40, y: 40, width: 420, height: 260 });
const COLOR_RE = /^(?:[1-6]|#[0-9a-f]{6})$/i;

function randomToken() {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID().replace(/-/g, "").slice(0, 12);
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
}

function schemaError(message, code, details = null) {
  return new SchemaError(message, { code, details });
}

export function validateWidgetPath(path) {
  const safe = assertSafePath(path);
  if (!isWidgetPath(safe)) throw schemaError(`Widget path must be under widgets/ and end in .html: ${safe}`, "WIDGET_PATH_INVALID");
  return safe;
}

export class FileWidgetRepository {
  constructor({ vault, catalog, canvasPathFromId } = {}) {
    if (!vault || !catalog) throw new TypeError("FileWidgetRepository requires vault and catalog");
    this.vault = vault;
    this.catalog = catalog;
    this.canvasPathFromId = canvasPathFromId || (() => null);
  }

  async createWidget(input = {}) {
    const path = validateWidgetPath(input.path);
    const validated = validateWidgetSource(input.source);
    if (typeof input.title !== "string" || input.title.trim() !== validated.title) {
      throw schemaError("Widget proposal title must exactly match the validated source title", "WIDGET_TITLE_MISMATCH");
    }
    await this.catalog.rebuild();
    if (this.catalog.getByPath(path) || this.catalog.getFallbackByPath(path) || await this.vault.stat(path)) {
      throw schemaError(`Widget already exists: ${path}`, "WIDGET_PATH_DUPLICATE");
    }
    await this.vault.write(path, validated.source, { expectedHash: null, mediaType: "text/html" });
    await this.catalog.reconcile([path]);
    let placement = null;
    if (input.canvasId) {
      try {
        placement = await this.addPlacement(path, input.canvasId, input.geometry);
      } catch (error) {
        throw schemaError(`Widget was saved but could not be placed: ${path}`, "WIDGET_PLACEMENT_FAILED", {
          recoverable: { path, title: validated.title, placed: false },
          cause: error.message,
          causeCode: error.code || null,
        });
      }
    }
    return { path, widget: this.catalog.getByPath(path), placement };
  }

  async addPlacement(path, canvasId, geometry = {}) {
    const widget = this.catalog.getByPath(validateWidgetPath(path));
    if (!widget) throw schemaError(`Widget not found: ${path}`, "WIDGET_NOT_FOUND");
    const canvasPath = this.canvasPathFromId(canvasId);
    if (!canvasPath) throw schemaError(`No canvas path for id: ${canvasId}`, "CANVAS_NOT_FOUND");
    const safeCanvasPath = assertSafePath(canvasPath);
    const stat = await this.vault.stat(safeCanvasPath);
    if (!stat) throw schemaError(`Canvas not found: ${safeCanvasPath}`, "CANVAS_NOT_FOUND");
    let document;
    try {
      document = JSON.parse(await this.vault.read(safeCanvasPath));
    } catch (error) {
      throw schemaError(`Invalid canvas document: ${safeCanvasPath}`, "CANVAS_INVALID", { cause: error.message });
    }
    if (!isCanvas(document)) throw schemaError(`Invalid canvas document: ${safeCanvasPath}`, "CANVAS_INVALID");
    const values = { ...DEFAULT_GEOMETRY, ...geometry };
    if (![values.x, values.y, values.width, values.height].every(Number.isInteger)
      || values.width < 1 || values.height < 1) {
      throw schemaError("Placement geometry must use integers with positive dimensions", "CANVAS_GEOMETRY_INVALID");
    }
    if (geometry.color !== undefined && (typeof geometry.color !== "string" || !COLOR_RE.test(geometry.color))) {
      throw schemaError("Placement color must be a standard JSON Canvas color", "CANVAS_INVALID");
    }
    const nodeId = geometry.id || `node-${randomToken()}`;
    if (document.nodes.some((node) => node.id === nodeId) || document.edges.some((edge) => edge.id === nodeId)) {
      throw schemaError(`Canvas id already exists: ${nodeId}`, "CANVAS_ID_DUPLICATE");
    }
    const node = {
      id: nodeId,
      type: "file",
      file: widget.path,
      x: values.x,
      y: values.y,
      width: values.width,
      height: values.height,
      ...(geometry.color !== undefined ? { color: geometry.color } : {}),
    };
    document.nodes.push(node);
    if (!isCanvas(document)) throw schemaError(`Invalid resulting canvas: ${safeCanvasPath}`, "CANVAS_INVALID");
    await this.vault.write(safeCanvasPath, JSON.stringify(document, null, 2) + "\n", { expectedHash: stat.hash });
    await this.catalog.reconcile([safeCanvasPath]);
    return { canvasId, canvasPath: safeCanvasPath, nodeId };
  }
}
