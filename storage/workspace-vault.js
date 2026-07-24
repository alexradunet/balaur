// Canonical workspace persistence on a VaultStore (Phase 4, ADR-0001, plan §6/§14).
//
// Ownership after the file-canonical migration:
//   .orbit/workspace.json   metadata-only sidecar (hierarchy, cameras, JD index)
//   canvases/<name>.canvas  one independently-valid JSON Canvas document each
//
// The sidecar never embeds full documents; each canvas document lives at its own
// logical path and is validated on read and write. This module is platform-neutral
// and asynchronous so it runs against MemoryVault (tests) and IndexedDbVault
// (browser). It is the tested foundation the Phase 4b app.js startup refactor
// builds on; localStorage is consulted only once as a legacy-profile migration source.

import { isCanvas } from "./canvas-validate.js";
import { assertSafePath, caseFoldKey } from "./vault-path.js";
import { SchemaError, ParseError } from "./vault-errors.js";

export const SIDECAR_PATH = ".orbit/workspace.json";
export const SIDECAR_FORMAT = "orbit-workspace";
export const SIDECAR_VERSION = 2; // sidecar FILE format version (file-canonical)
export const ROOT_CANVAS_PATH = "canvases/root.canvas";
export const CANVAS_MEDIA_TYPE = "application/jsoncanvas+json";
export const SIDECAR_MEDIA_TYPE = "application/json";
const DEFAULT_CAMERA = { x: 80, y: 55, zoom: 0.78 };

// Logical .canvas path for a canvas record. The root is always
// canvases/root.canvas; other canvases use their stored path or a derived
// canvases/<id>.canvas. The result is strictly validated so a corrupt sidecar
// cannot escape the vault root (plan §7.3, Phase 4 portal path validation).
export function canvasPathFor(record, rootId) {
  if (record.id === rootId) return assertSafePath(ROOT_CANVAS_PATH);
  return assertSafePath(record.path || `canvases/${record.id}.canvas`);
}

// Build the metadata-only sidecar object from an in-memory workspace. Canvas
// documents are intentionally omitted; each lives at canvasPathFor(record).
// Transient/machine-local state (selection, dialogs, runtime AI status) is never
// part of the workspace object and therefore never exported (plan §6.1).
export function toSidecar(workspace) {
  const rootId = workspace.rootId;
  const canvases = {};
  for (const record of Object.values(workspace.canvases || {})) {
    const entry = {
      id: record.id,
      title: record.title,
      path: canvasPathFor(record, rootId),
      parentId: record.parentId ?? null,
      portalNodeId: record.portalNodeId ?? null,
      camera: record.camera || { ...DEFAULT_CAMERA },
    };
    if (record.icon) entry.icon = record.icon;
    if (record.kind === "hub" || record.kind === "project") entry.kind = record.kind;
    canvases[record.id] = entry;
  }
  return {
    format: SIDECAR_FORMAT,
    version: SIDECAR_VERSION,
    rootId,
    activeId: workspace.activeId || rootId,
    canvases,
  };
}

export function sidecarToJSON(workspace) {
  return JSON.stringify(toSidecar(workspace), null, 2) + "\n";
}

export function canvasToJSON(document) {
  return JSON.stringify(document, null, 2) + "\n";
}

// Parse and validate a sidecar document. Throws ParseError/SchemaError/PathError
// on any structural problem so callers can surface a diagnostic instead of
// trusting a corrupt sidecar.
export function parseSidecar(text) {
  let data;
  try { data = JSON.parse(text); }
  catch (err) { throw new ParseError(`Sidecar is not valid JSON: ${err.message}`, { code: "SIDECAR_PARSE" }); }
  if (!data || typeof data !== "object" || Array.isArray(data)) throw new SchemaError("Sidecar is not an object", { code: "SIDECAR_SCHEMA" });
  if (data.format !== SIDECAR_FORMAT) throw new SchemaError(`Unexpected sidecar format: ${data.format}`, { code: "SIDECAR_FORMAT" });
  if (data.version !== SIDECAR_VERSION) throw new SchemaError(`Unsupported sidecar version: ${data.version}`, { code: "SIDECAR_VERSION" });
  if (typeof data.rootId !== "string" || !data.rootId) throw new SchemaError("Sidecar missing rootId", { code: "SIDECAR_SCHEMA" });
  const canvases = data.canvases;
  if (!canvases || typeof canvases !== "object" || Array.isArray(canvases)) throw new SchemaError("Sidecar missing canvases", { code: "SIDECAR_SCHEMA" });
  if (!canvases[data.rootId]) throw new SchemaError("Sidecar rootId has no canvas record", { code: "SIDECAR_SCHEMA" });
  const folds = new Map();
  for (const [key, record] of Object.entries(canvases)) {
    if (!record || typeof record.id !== "string" || !record.id || key !== record.id || typeof record.title !== "string" || typeof record.path !== "string") {
      throw new SchemaError("Sidecar canvas record is malformed or key/id mismatch", { code: "SIDECAR_SCHEMA" });
    }
    // Navigator glyph is cosmetic sidecar metadata: sanitize rather than reject
    // so a hand-edited icon can never make the workspace unloadable.
    if (record.icon !== undefined) {
      const icon = typeof record.icon === "string" ? record.icon.trim() : "";
      if (!icon || [...icon].length > 16) delete record.icon;
      else record.icon = icon;
    }
    // Graph model (ADR-0003): canvas `kind` is optional app metadata. Legacy
    // Johnny Decimal fields are stripped on read so pre-graph sidecars load.
    delete record.jdCode; delete record.jdTitle; delete record.jdKind;
    if (record.kind !== "hub" && record.kind !== "project") delete record.kind;
    const path = assertSafePath(record.path);
    if (!/^canvases\/[^/]+\.canvas$/.test(path)) throw new SchemaError(`Canvas path is outside canvases/: ${path}`, { code: "SIDECAR_CANVAS_PATH" });
    if (record.id === data.rootId && path !== ROOT_CANVAS_PATH) throw new SchemaError("Root canvas must use canvases/root.canvas", { code: "SIDECAR_CANVAS_PATH" });
    const fold = caseFoldKey(path);
    if (folds.has(fold)) throw new SchemaError(`Canvas path collision: ${path}`, { code: "SIDECAR_CANVAS_COLLISION" });
    folds.set(fold, path);
    if (record.id === data.rootId && record.parentId !== null && record.parentId !== undefined) throw new SchemaError("Root canvas cannot have a parent", { code: "SIDECAR_HIERARCHY" });
    if (record.id !== data.rootId && (typeof record.parentId !== "string" || !record.parentId || record.parentId === record.id)) throw new SchemaError(`Invalid parent for canvas: ${record.id}`, { code: "SIDECAR_HIERARCHY" });
    if (record.id !== data.rootId && (typeof record.portalNodeId !== "string" || !record.portalNodeId)) throw new SchemaError(`Nested canvas needs a portal node: ${record.id}`, { code: "SIDECAR_HIERARCHY" });
    if (record.portalNodeId !== null && record.portalNodeId !== undefined && (typeof record.portalNodeId !== "string" || !record.portalNodeId)) throw new SchemaError(`Invalid portal node for canvas: ${record.id}`, { code: "SIDECAR_HIERARCHY" });
  }
  for (const record of Object.values(canvases)) {
    if (record.parentId !== null && record.parentId !== undefined && !canvases[record.parentId]) throw new SchemaError(`Missing parent canvas: ${record.parentId}`, { code: "SIDECAR_HIERARCHY" });
  }
  for (const record of Object.values(canvases)) {
    const seen = new Set([record.id]);
    let parent = record.parentId;
    while (parent !== null && parent !== undefined) {
      if (seen.has(parent)) throw new SchemaError("Canvas hierarchy contains a cycle", { code: "SIDECAR_HIERARCHY" });
      seen.add(parent); parent = canvases[parent]?.parentId ?? null;
    }
  }
  if (typeof data.activeId !== "string" || !canvases[data.activeId]) data.activeId = data.rootId;
  delete data.johnnyDecimal; // legacy JD index removed in ADR-0003
  return data;
}

// Read the sidecar and re-attach each canvas document from its .canvas file.
// Missing or malformed documents are represented as read-only repair placeholders;
// their raw bytes are retained and save() never substitutes or overwrites them.
export async function loadWorkspace(vault) {
  let sidecarText;
  try { sidecarText = await vault.read(SIDECAR_PATH); }
  catch (err) {
    if (err && err.code === "NOT_FOUND") return null;
    throw err;
  }
  const sidecar = parseSidecar(sidecarText);
  const diagnostics = [];
  const canvases = {};
  for (const record of Object.values(sidecar.canvases)) {
    let document = { nodes: [], edges: [] }, rawContent = null, readOnly = false;
    try {
      rawContent = await vault.read(record.path);
      const parsed = JSON.parse(rawContent);
      if (isCanvas(parsed)) document = parsed;
      else { readOnly = true; diagnostics.push({ path: record.path, code: "CANVAS_INVALID", message: `Canvas document failed validation: ${record.path}` }); }
    } catch (err) {
      readOnly = true;
      const code = err && err.code === "NOT_FOUND" ? "CANVAS_MISSING" : "CANVAS_PARSE";
      if (err && err.code !== "NOT_FOUND") { try { rawContent = await vault.read(record.path); } catch (_) {} }
      diagnostics.push({ path: record.path, code, message: `${code === "CANVAS_MISSING" ? "Missing" : "Unreadable"} canvas file: ${record.path}` });
    }
    canvases[record.id] = { ...record, document, rawContent, readOnly, repairRequired: readOnly };
  }
  return {
    // In-memory workspace shape consumed by app.js (its `version: 1` contract),
    // distinct from the sidecar FILE format version (SIDECAR_VERSION).
    workspace: { version: 1, rootId: sidecar.rootId, activeId: sidecar.activeId, canvases },
    diagnostics,
  };
}

export async function hasWorkspace(vault) {
  return vault.exists(SIDECAR_PATH);
}

// Vault-backed workspace store with optimistic-concurrency saves and a serialized
// write queue. expectedHash preconditions detect external edits (plan §13.4); the
// sidecar is written last so an interrupted save never loses hierarchy.
export class WorkspaceStore {
  constructor(vault) {
    this.vault = vault;
    this.hashes = new Map(); // path -> last-known content hash
    this.ownedCanvasPaths = new Set(); // paths owned by the last loaded/saved sidecar
    this._queue = Promise.resolve();
  }

  // Serialize async work so saves/loads never interleave. The chain itself never
  // rejects; callers receive the real per-task promise.
  _enqueue(task) {
    const run = this._queue.then(task, task);
    this._queue = run.catch(() => {});
    return run;
  }

  async load() {
    return this._enqueue(async () => {
      const result = await loadWorkspace(this.vault);
      if (result) {
        await this._refreshHashes();
        this.ownedCanvasPaths = new Set(Object.values(result.workspace.canvases).map((record) => record.path));
      }
      return result;
    });
  }

  // One-time migration of a legacy in-memory workspace (documents inline) to
  // canonical vault files + sidecar. Every file must be new (expectedHash null).
  async migrate(legacyWorkspace) {
    return this._enqueue(() => this._save(legacyWorkspace, true));
  }

  async save(workspace) {
    return this._enqueue(() => this._save(workspace, false));
  }

  async _refreshHashes() {
    this.hashes.clear();
    for (const meta of await this.vault.list("")) this.hashes.set(meta.path, meta.hash);
  }

  async _save(workspace, fresh) {
    const sidecar = toSidecar(workspace);
    // Validate the complete metadata graph before the first write. In
    // particular, this prevents a malformed path from making a canvas write
    // overwrite a life-entity file or from creating a path collision.
    parseSidecar(JSON.stringify(sidecar));
    const states = [];
    for (const record of Object.values(sidecar.canvases)) {
      const state = workspace.canvases[record.id];
      if (state?.readOnly || state?.repairRequired) continue;
      const doc = state?.document;
      if (!isCanvas(doc)) throw new SchemaError(`Refusing to write invalid canvas document for ${record.id}`, { code: "CANVAS_INVALID" });
      states.push({ record, doc });
    }
    const written = [];
    // TODO(deferred, plan S14): compare canonical hashes and write only changed
    // canvas documents; keep this correctness-first whole-space save for v1.
    // 1. Write every canvas document to its canonical path (validated first).
    for (const { record, doc } of states) {
      const expectedHash = fresh ? null : (this.hashes.get(record.path) ?? null);
      const meta = await this.vault.write(record.path, canvasToJSON(doc), { expectedHash, mediaType: CANVAS_MEDIA_TYPE });
      this.hashes.set(record.path, meta.hash);
      written.push({ path: record.path, hash: meta.hash, mediaType: meta.mediaType });
    }
    // 2. Write the sidecar last: hierarchy only advances after documents land.
    const sidecarExpected = fresh ? null : (this.hashes.get(SIDECAR_PATH) ?? null);
    const sidecarMeta = await this.vault.write(SIDECAR_PATH, JSON.stringify(sidecar, null, 2) + "\n", { expectedHash: sidecarExpected, mediaType: SIDECAR_MEDIA_TYPE });
    this.hashes.set(SIDECAR_PATH, sidecarMeta.hash);
    // 3. Remove only canvas files owned by the previous sidecar. Unknown
    // canvases are recovery orphans and must never be destroyed by a stale
    // workspace snapshot. CAS protects a previously owned file edited outside
    // this store.
    const referenced = new Set(Object.values(sidecar.canvases).map((r) => r.path));
    const orphans = [];
    for (const path of this.ownedCanvasPaths) {
      if (referenced.has(path)) continue;
      const expectedHash = this.hashes.get(path);
      if (expectedHash === undefined) continue;
      const current = await this.vault.stat(path);
      if (!current) { this.hashes.delete(path); continue; }
      await this.vault.remove(path, { expectedHash });
      this.hashes.delete(path);
      orphans.push(path);
    }
    this.ownedCanvasPaths = new Set(referenced);
    return { sidecarHash: sidecarMeta.hash, files: written, orphans, revision: this.vault.revision };
  }
}
