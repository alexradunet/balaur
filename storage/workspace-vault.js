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
// builds on; the running app still uses the legacy localStorage path until then.

import { isCanvas } from "./canvas-validate.js";
import { assertSafePath } from "./vault-path.js";
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
    if (record.jdCode) entry.jdCode = record.jdCode;
    if (record.jdTitle) entry.jdTitle = record.jdTitle;
    canvases[record.id] = entry;
  }
  return {
    format: SIDECAR_FORMAT,
    version: SIDECAR_VERSION,
    rootId,
    activeId: workspace.activeId || rootId,
    johnnyDecimal: workspace.johnnyDecimal || { enabled: false, entries: {} },
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
  for (const record of Object.values(canvases)) {
    if (!record || typeof record.id !== "string" || typeof record.title !== "string" || typeof record.path !== "string") {
      throw new SchemaError("Sidecar canvas record is malformed", { code: "SIDECAR_SCHEMA" });
    }
    assertSafePath(record.path); // throws PathError on an unsafe stored path
  }
  if (typeof data.activeId !== "string" || !canvases[data.activeId]) data.activeId = data.rootId;
  data.johnnyDecimal ||= { enabled: false, entries: {} };
  data.johnnyDecimal.entries ||= {};
  return data;
}

// Read the sidecar and re-attach each canvas document from its .canvas file.
// Returns null when no sidecar exists yet (a fresh vault). A missing or invalid
// canvas file never loses hierarchy: the record is kept with an empty document
// plus a diagnostic (plan §11.5 missing-reference handling, Phase 4 crash
// recovery — "interrupted canvas save recovers without hierarchy loss").
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
    let document = { nodes: [], edges: [] };
    try {
      const parsed = JSON.parse(await vault.read(record.path));
      if (isCanvas(parsed)) document = parsed;
      else diagnostics.push({ path: record.path, code: "CANVAS_INVALID", message: `Canvas document failed validation: ${record.path}` });
    } catch (err) {
      const code = err && err.code === "NOT_FOUND" ? "CANVAS_MISSING" : "CANVAS_PARSE";
      diagnostics.push({ path: record.path, code, message: `${code === "CANVAS_MISSING" ? "Missing" : "Unreadable"} canvas file: ${record.path}` });
    }
    canvases[record.id] = { ...record, document };
  }
  return {
    // In-memory workspace shape consumed by app.js (its `version: 1` contract),
    // distinct from the sidecar FILE format version (SIDECAR_VERSION).
    workspace: { version: 1, rootId: sidecar.rootId, activeId: sidecar.activeId, johnnyDecimal: sidecar.johnnyDecimal, canvases },
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
      if (result) await this._refreshHashes();
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
    const written = [];
    // 1. Write every canvas document to its canonical path (validated first).
    for (const record of Object.values(sidecar.canvases)) {
      const doc = workspace.canvases[record.id]?.document || { nodes: [], edges: [] };
      if (!isCanvas(doc)) throw new SchemaError(`Refusing to write invalid canvas document for ${record.id}`, { code: "CANVAS_INVALID" });
      const expectedHash = fresh ? null : (this.hashes.get(record.path) ?? null);
      const meta = await this.vault.write(record.path, canvasToJSON(doc), { expectedHash, mediaType: CANVAS_MEDIA_TYPE });
      this.hashes.set(record.path, meta.hash);
      written.push({ path: record.path, hash: meta.hash, mediaType: meta.mediaType });
    }
    // 2. Write the sidecar last: hierarchy only advances after documents land.
    const sidecarExpected = fresh ? null : (this.hashes.get(SIDECAR_PATH) ?? null);
    const sidecarMeta = await this.vault.write(SIDECAR_PATH, JSON.stringify(sidecar, null, 2) + "\n", { expectedHash: sidecarExpected, mediaType: SIDECAR_MEDIA_TYPE });
    this.hashes.set(SIDECAR_PATH, sidecarMeta.hash);
    // 3. Remove canvas files no longer referenced by the sidecar (orphans).
    const referenced = new Set(Object.values(sidecar.canvases).map((r) => r.path));
    const orphans = [];
    for (const meta of await this.vault.list("canvases/")) {
      if (!referenced.has(meta.path)) {
        await this.vault.remove(meta.path);
        this.hashes.delete(meta.path);
        orphans.push(meta.path);
      }
    }
    return { sidecarHash: sidecarMeta.hash, files: written, orphans, revision: this.vault.revision };
  }
}
