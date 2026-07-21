// File-canonical task repository (Phase 5, ADR-0001, plan §13/§14.4).
//
// Tasks are canonical Markdown files under tasks/; their workflow state lives in
// preservation-first frontmatter. Placements are standard JSON Canvas `file`
// nodes in .canvas documents (zero or many per task). The repository writes the
// canonical file first, then reindexes through the LifeIndexer; Today/search keep
// querying the index. Platform-neutral and asynchronous — tested against
// MemoryVault + MemoryIndex (storage/phase5.test.js).
//
// Canvas identity coupling: placements are derived from canvas files, so the
// indexer maps canvas path -> id (canvasIdFromPath) while this repository maps
// id -> path (canvasPathFromId). The caller supplies both from the workspace
// sidecar; they must be inverses.

import { serializeTask, parseTask, TaskCodec } from "./entity-codec.js";
import { patchFields, splitFrontmatter } from "./frontmatter.js";
import { entityPath } from "./vault-path.js";
import { isCanvas } from "./canvas-validate.js";
import { SchemaError } from "./vault-errors.js";

// App-facing camelCase patch keys -> frontmatter kebab-case keys (plan §8).
const PATCH_KEYS = {
  title: "title", status: "status", priority: "priority",
  scheduledOn: "scheduled-on", dueOn: "due-on", completedAt: "completed-at",
  estimateMinutes: "estimate-minutes", recurrence: "recurrence",
};

const DEFAULT_GEOMETRY = { x: 40, y: 40, width: 380, height: 220 };

// Replace only the body, preserving the (already patched) frontmatter bytes.
function replaceBody(text, body) {
  const fm = splitFrontmatter(text);
  if (!fm) throw new SchemaError("Cannot replace body: missing frontmatter", { code: "FM_NO_DELIMITER" });
  const head = fm.bom + fm.lines.slice(0, fm.closeIdx + 1).join("");
  return head + (body ? "\n" + body : "");
}

function randomToken() {
  const c = globalThis.crypto;
  if (c?.randomUUID) return c.randomUUID().replace(/-/g, "").slice(0, 12);
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
}

export class FileTaskRepository {
  constructor({ vault, index, indexer, canvasPathFromId, now = () => new Date().toISOString(), idPrefix = "task" }) {
    this.vault = vault;
    this.index = index;
    this.indexer = indexer;
    this.canvasPathFromId = canvasPathFromId || (() => null);
    this.now = now;
    this.idPrefix = idPrefix;
  }

  _newId() { return `${this.idPrefix}-${randomToken()}`; }

  // Create the canonical task file, index it, and optionally place it on a canvas.
  async createTask(input = {}) {
    const title = String(input.title ?? "").trim();
    if (!title) throw new SchemaError("Task title is required", { code: "TASK_TITLE_REQUIRED" });
    const id = input.id || this._newId();
    const ts = this.now();
    const done = input.status === "done";
    const task = {
      orbitId: id, title,
      status: input.status || "next",
      priority: input.priority ?? null,
      scheduledOn: input.scheduledOn || null,
      dueOn: input.dueOn || null,
      completedAt: done ? (input.completedAt || ts) : null,
      estimateMinutes: input.estimateMinutes ?? null,
      recurrence: input.recurrence ?? null,
      createdAt: ts, updatedAt: ts,
      body: input.body || "",
    };
    const path = input.path || entityPath("tasks", title, id, "md");
    const content = serializeTask(task);
    await this.vault.write(path, content, { expectedHash: null });
    await this.indexer.indexFile(path, content, {});
    const placement = input.canvasId ? await this.addPlacement(id, input.canvasId, input.geometry) : null;
    return { id, path, task, placement };
  }

  _sourceFor(id) {
    const row = this.index.taskById(id);
    if (!row) throw new SchemaError(`Task not found: ${id}`, { code: "TASK_NOT_FOUND" });
    return { path: row.sourcePath, hash: row.sourceHash };
  }

  async getTask(id) {
    const { path } = this._sourceFor(id);
    return parseTask(await this.vault.read(path));
  }

  // Preservation-first update: patch only the named frontmatter fields (and/or the
  // body), leaving all other bytes untouched (plan §8.3). Bumps updated-at.
  async updateTask(id, patch = {}) {
    const { path, hash } = this._sourceFor(id);
    const content = await this.vault.read(path);
    const fmPatch = { "updated-at": this.now() };
    for (const [key, value] of Object.entries(patch)) {
      if (key === "body") continue;
      const fmKey = PATCH_KEYS[key];
      if (!fmKey) throw new SchemaError(`Unknown task field: ${key}`, { code: "TASK_UNKNOWN_FIELD" });
      fmPatch[fmKey] = value;
    }
    let next = patchFields(content, fmPatch, TaskCodec.spec);
    if ("body" in patch) next = replaceBody(next, patch.body);
    await this.vault.write(path, next, { expectedHash: hash });
    await this.indexer.indexFile(path, next, {});
    return parseTask(next);
  }

  async completeTask(id) { return this.updateTask(id, { status: "done", completedAt: this.now() }); }
  async reopenTask(id) { return this.updateTask(id, { status: "next", completedAt: null }); }

  // Add a standard file-node placement for a task on a canvas (plan §13.2).
  async addPlacement(id, canvasId, geometry = {}) {
    const { path: taskPath } = this._sourceFor(id);
    const canvasPath = this.canvasPathFromId(canvasId);
    if (!canvasPath) throw new SchemaError(`No canvas path for id: ${canvasId}`, { code: "CANVAS_NOT_FOUND" });
    const stat = await this.vault.stat(canvasPath);
    const doc = stat ? JSON.parse(await this.vault.read(canvasPath)) : { nodes: [], edges: [] };
    if (!isCanvas(doc)) throw new SchemaError(`Invalid canvas document: ${canvasPath}`, { code: "CANVAS_INVALID" });
    const g = { ...DEFAULT_GEOMETRY, ...geometry };
    const node = { id: geometry.id || `node-${randomToken()}`, type: "file", file: taskPath, x: g.x, y: g.y, width: g.width, height: g.height };
    if (geometry.color !== undefined) node.color = geometry.color;
    doc.nodes.push(node);
    const content = JSON.stringify(doc, null, 2) + "\n";
    await this.vault.write(canvasPath, content, { expectedHash: stat ? stat.hash : null });
    await this.indexer.indexFile(canvasPath, content, {});
    await this.indexer.reindexPlacements();
    return { canvasId, nodeId: node.id, canvasPath };
  }

  // Remove one placement (the file node) without touching the task entity (plan §13.3).
  async removePlacement(canvasId, nodeId) {
    const canvasPath = this.canvasPathFromId(canvasId);
    if (!canvasPath) throw new SchemaError(`No canvas path for id: ${canvasId}`, { code: "CANVAS_NOT_FOUND" });
    const stat = await this.vault.stat(canvasPath);
    if (!stat) throw new SchemaError(`Canvas not found: ${canvasPath}`, { code: "CANVAS_NOT_FOUND" });
    const doc = JSON.parse(await this.vault.read(canvasPath));
    const before = doc.nodes.length;
    doc.nodes = doc.nodes.filter((n) => n.id !== nodeId);
    doc.edges = (doc.edges || []).filter((e) => e.fromNode !== nodeId && e.toNode !== nodeId);
    if (doc.nodes.length === before) return { removed: false, canvasId, nodeId };
    const content = JSON.stringify(doc, null, 2) + "\n";
    await this.vault.write(canvasPath, content, { expectedHash: stat.hash });
    await this.indexer.indexFile(canvasPath, content, {});
    await this.indexer.reindexPlacements();
    return { removed: true, canvasId, nodeId };
  }

  // Delete the task everywhere: remove every placement, then the canonical file
  // (plan §13.3 "Delete everywhere" — the only entity-deleting path).
  async deleteTask(id) {
    const { path } = this._sourceFor(id);
    const placements = this.index.placementsForEntity(id);
    for (const p of placements) {
      try { await this.removePlacement(p.canvasId, p.nodeId); }
      catch (_) { /* canvas already gone; file removal below is authoritative */ }
    }
    await this.vault.remove(path);
    this.indexer.removeFile(path);
    return { id, path, removedPlacements: placements.length };
  }
}
