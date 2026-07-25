// File-canonical note repository (project 003, ADR-0004).
//
// A Note is a path-identified canonical notes/*.md file with no mandatory
// frontmatter and no balaur-id; its identity is its path. This is a thin
// PATH-KEYED repository that combines the task/journal canonical-file-first
// write discipline (expected-hash preconditions, reindex-after-write) with the
// component-card path-scan placement resolution. Note placements are NOT tracked
// by the disposable placements index (notes have no balaur-id and notes/ is not an
// entity directory), so placements are resolved by scanning canvas documents.
//
// replacePlacement(path, fromCanvasId, nodeId, toCanvasId, geometry) is the
// path-generic drain primitive consumed by project 002: remove the placement on
// the source canvas and add a placement for the SAME path on the target, leaving
// the canonical file and its identity unchanged.

import { isCanvas } from "./canvas-validate.js";
import { caseFoldKey, normalizePath, slugify } from "./vault-path.js";
import { ConflictError, SchemaError } from "./vault-errors.js";
import { NOTE_KIND_MARKERS, noteTitleFromBody } from "./note-catalog.js";

const DEFAULT_GEOMETRY = Object.freeze({ x: 40, y: 40, width: 380, height: 220 });

function randomToken() {
  const c = globalThis.crypto;
  if (c?.randomUUID) return c.randomUUID().replace(/-/g, "").slice(0, 12);
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
}

export class FileNoteRepository {
  constructor({ vault, index, indexer, catalog, canvasPathFromId, now = () => new Date().toISOString() }) {
    if (!vault || !catalog) throw new TypeError("FileNoteRepository requires vault and catalog");
    this.vault = vault;
    this.index = index;
    this.indexer = indexer;
    this.catalog = catalog;
    this.canvasPathFromId = canvasPathFromId || (() => null);
    this.now = now;
  }

  // Collision-safe note path: notes/<slug>.md, then -2, -3, … by case-fold key.
  // Collision handling runs only at creation (notes are never renamed in v1).
  allocatePath(title) {
    const base = slugify(title);
    const taken = new Set(this.catalog.notes().map((note) => caseFoldKey(note.path)));
    let candidate = `notes/${base}.md`;
    let suffix = 2;
    while (taken.has(caseFoldKey(candidate))) {
      candidate = `notes/${base}-${suffix}.md`;
      suffix += 1;
    }
    return normalizePath(candidate);
  }

  _noteContent(kind, body) {
    const marker = kind ? NOTE_KIND_MARKERS[kind] : null;
    if (!marker) return body;
    return body ? `${marker}\n${body}` : `${marker}\n`;
  }

  // Create the canonical note file, index it as an untyped source record, and
  // optionally place it on a canvas. The optional explicit `path` override lets
  // the graph starter seed notes at exact paths (the task repository supports
  // input.path the same way).
  async createNote(input = {}) {
    const body = String(input.body ?? "");
    const title = String(input.title ?? "").trim() || noteTitleFromBody(body) || "";
    const content = this._noteContent(input.kind ?? null, body);
    const path = input.path || this.allocatePath(title);
    await this.vault.write(path, content, { expectedHash: null });
    await this.indexer.indexFile(path, content, {});
    await this.catalog.reconcile([path]);
    const placement = input.canvasId ? await this.addPlacement(path, input.canvasId, { ...input.geometry, color: input.color }) : null;
    return { path, note: this.catalog.getByPath(path), placement };
  }

  // Add a standard file-node placement for a note path on a canvas. Cloned from
  // FileJournalRepository.addPlacement, but path-keyed and geometry-validated.
  async addPlacement(path, canvasId, geometry = {}) {
    const noteStat = await this.vault.stat(path);
    if (!noteStat) throw new SchemaError(`Note not found: ${path}`, { code: "NOTE_NOT_FOUND" });
    const canvasPath = this.canvasPathFromId(canvasId);
    if (!canvasPath) throw new SchemaError(`No canvas path for id: ${canvasId}`, { code: "CANVAS_NOT_FOUND" });
    const stat = await this.vault.stat(canvasPath);
    if (!stat) throw new SchemaError(`Canvas not found: ${canvasPath}`, { code: "CANVAS_NOT_FOUND" });
    let doc;
    try { doc = JSON.parse(await this.vault.read(canvasPath)); }
    catch (err) { throw new SchemaError(`Invalid canvas document: ${canvasPath}`, { code: "CANVAS_INVALID", cause: err }); }
    if (!isCanvas(doc)) throw new SchemaError(`Invalid canvas document: ${canvasPath}`, { code: "CANVAS_INVALID" });
    const g = { ...DEFAULT_GEOMETRY, ...geometry };
    if (![g.x, g.y, g.width, g.height].every(Number.isInteger) || g.width <= 0 || g.height <= 0) {
      throw new SchemaError("Placement geometry must use integers with positive dimensions", { code: "CANVAS_GEOMETRY_INVALID" });
    }
    const nodeId = geometry.id || `node-${randomToken()}`;
    if (doc.nodes.some((n) => n.id === nodeId) || doc.edges.some((e) => e.id === nodeId)) throw new SchemaError(`Canvas id already exists: ${nodeId}`, { code: "CANVAS_ID_DUPLICATE" });
    const node = { id: nodeId, type: "file", file: path, x: g.x, y: g.y, width: g.width, height: g.height };
    if (geometry.color !== undefined) node.color = geometry.color;
    doc.nodes.push(node);
    if (!isCanvas(doc)) throw new SchemaError(`Invalid canvas document: ${canvasPath}`, { code: "CANVAS_INVALID" });
    const content = JSON.stringify(doc, null, 2) + "\n";
    await this.vault.write(canvasPath, content, { expectedHash: stat.hash });
    await this.indexer.indexFile(canvasPath, content, {});
    await this.catalog.reconcile([canvasPath]);
    return { canvasId, nodeId, canvasPath, path };
  }

  // Remove one placement (the file node) and its incident edges by canvas path,
  // without touching the note file. Returns true only when a node was removed.
  async _removePlacementAtPath(canvasPath, nodeId) {
    const stat = await this.vault.stat(canvasPath);
    if (!stat) throw new SchemaError(`Canvas not found: ${canvasPath}`, { code: "CANVAS_NOT_FOUND" });
    let doc;
    try { doc = JSON.parse(await this.vault.read(canvasPath)); }
    catch (err) { throw new SchemaError(`Invalid canvas document: ${canvasPath}`, { code: "CANVAS_INVALID", cause: err }); }
    if (!isCanvas(doc)) throw new SchemaError(`Invalid canvas document: ${canvasPath}`, { code: "CANVAS_INVALID" });
    const before = doc.nodes.length;
    doc.nodes = doc.nodes.filter((n) => n.id !== nodeId);
    doc.edges = (doc.edges || []).filter((e) => e.fromNode !== nodeId && e.toNode !== nodeId);
    if (doc.nodes.length === before) return false;
    const content = JSON.stringify(doc, null, 2) + "\n";
    await this.vault.write(canvasPath, content, { expectedHash: stat.hash });
    await this.indexer.indexFile(canvasPath, content, {});
    return true;
  }

  // Remove one placement by (canvasId, nodeId), cloned from
  // FileTaskRepository.removePlacement.
  async removePlacement(canvasId, nodeId) {
    const canvasPath = this.canvasPathFromId(canvasId);
    if (!canvasPath) throw new SchemaError(`No canvas path for id: ${canvasId}`, { code: "CANVAS_NOT_FOUND" });
    const removed = await this._removePlacementAtPath(canvasPath, nodeId);
    if (removed) await this.catalog.reconcile([canvasPath]);
    return { removed, canvasId, nodeId };
  }

  // Full-body rewrite under an expected-hash precondition. A note has no
  // frontmatter, so this is trivially preservation-first. The expected hash is
  // checked against the catalog's last-known hash so a write behind the
  // repository's back is rejected as a conflict (component-card updateCard
  // pattern); the write itself uses the current stat hash.
  async updateNote(path, body) {
    const stat = await this.vault.stat(path);
    if (!stat) throw new SchemaError(`Note not found: ${path}`, { code: "NOTE_NOT_FOUND" });
    const known = this.catalog.getByPath(path);
    if (known && known.hash !== stat.hash) {
      await this.catalog.reconcile([path]);
      throw new ConflictError(`Note changed before update: ${path}`, { code: "WRITE_CONFLICT" });
    }
    const content = String(body);
    await this.vault.write(path, content, { expectedHash: stat.hash });
    await this.indexer.indexFile(path, content, {});
    await this.catalog.reconcile([path]);
    return this.catalog.getByPath(path);
  }

  // Delete the note everywhere: resolve placements by scanning canvas documents
  // (the placements index does NOT track notes), remove each, then remove the
  // canonical file and reindex.
  async deleteNote(path) {
    const stat = await this.vault.stat(path);
    if (!stat) throw new SchemaError(`Note not found: ${path}`, { code: "NOTE_NOT_FOUND" });
    const placements = this.catalog.getByPath(path)?.placements || [];
    const changed = [];
    let removedPlacements = 0;
    for (const placement of placements) {
      if (await this._removePlacementAtPath(placement.canvasPath, placement.nodeId)) {
        removedPlacements += 1;
        changed.push(placement.canvasPath);
      }
    }
    await this.vault.remove(path, { expectedHash: stat.hash });
    await this.indexer.removeFile(path);
    await this.catalog.reconcile([path, ...changed]);
    return { path, removedPlacements };
  }

  // Drain primitive (consumed by project 002): remove the placement on the source
  // canvas and add a placement for the SAME path on the target. Path-generic and
  // path-keyed so notes, tasks, and journals drain uniformly. The source path is
  // preserved, so identity is unchanged.
  async replacePlacement(path, fromCanvasId, nodeId, toCanvasId, geometry = {}) {
    const removed = await this.removePlacement(fromCanvasId, nodeId);
    const added = removed.removed ? await this.addPlacement(path, toCanvasId, geometry) : null;
    return { removed, added, path };
  }
}
