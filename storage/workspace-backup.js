// Version-2 whole-space backup/restore (Phase 4, ADR-0001, plan §15).
//
// A version-2 .orbit.json bundle contains the metadata-only sidecar plus the raw
// logical files (canvases, entities, widgets) — never a SQLite snapshot, which is
// rebuildable. Export preserves file bytes and validates every canvas; import
// validates the envelope, every path, every canvas, and detects duplicate or
// malformed entities in staging before any destructive switch (plan §15.2/§15.3).

import { isCanvas } from "./canvas-validate.js";
import { assertSafePath, caseFoldKey } from "./vault-path.js";
import { SchemaError, ParseError, PathError } from "./vault-errors.js";
import { mediaTypeFor } from "./vault-store.js";
import { SIDECAR_PATH, SIDECAR_FORMAT, SIDECAR_VERSION, parseSidecar } from "./workspace-vault.js";
import { buildSourceRecord } from "./life-indexer.js";

export const BACKUP_FORMAT = SIDECAR_FORMAT; // "orbit-workspace"
export const BACKUP_VERSION = SIDECAR_VERSION; // 2

// Export the whole vault to a version-2 bundle. Reads raw file text (byte
// preservation, plan §15.2), validates every .canvas, sorts paths
// deterministically, rejects duplicate normalized paths, and reports unreadable
// files instead of dropping them silently. The sidecar is carried in `workspace`,
// not duplicated inside `files`.
export async function exportBundle(vault, { exportedAt = new Date().toISOString() } = {}) {
  let sidecarText;
  try { sidecarText = await vault.read(SIDECAR_PATH); }
  catch (_) { throw new SchemaError("Vault has no workspace sidecar to export", { code: "SIDECAR_MISSING" }); }
  const workspace = parseSidecar(sidecarText);

  const metas = (await vault.list("")).sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));
  const seen = new Set();
  const files = [];
  const diagnostics = [];
  for (const meta of metas) {
    if (meta.path === SIDECAR_PATH) continue; // represented by `workspace`
    const fold = caseFoldKey(meta.path);
    if (seen.has(fold)) throw new PathError(`Duplicate normalized path in vault: ${meta.path}`, { code: "PATH_CASE_COLLISION" });
    seen.add(fold);
    let text;
    try { text = await vault.read(meta.path); }
    catch (_) { diagnostics.push({ path: meta.path, code: "FILE_UNREADABLE", message: `Unreadable file: ${meta.path}` }); continue; }
    if (meta.path.endsWith(".canvas")) {
      let parsed;
      try { parsed = JSON.parse(text); }
      catch (_) { throw new ParseError(`Canvas is not valid JSON: ${meta.path}`, { code: "CANVAS_PARSE" }); }
      if (!isCanvas(parsed)) throw new SchemaError(`Canvas failed validation before export: ${meta.path}`, { code: "CANVAS_INVALID" });
    }
    files.push({ path: meta.path, mediaType: meta.mediaType || mediaTypeFor(meta.path), text });
  }
  return { bundle: { format: BACKUP_FORMAT, version: BACKUP_VERSION, exportedAt, workspace, files }, diagnostics };
}

export function serializeBundle(bundle) {
  return JSON.stringify(bundle, null, 2) + "\n";
}

// Parse + validate the envelope only (no vault access, no writes).
export function parseBundle(text) {
  let data;
  try { data = JSON.parse(text); }
  catch (err) { throw new ParseError(`Bundle is not valid JSON: ${err.message}`, { code: "BUNDLE_PARSE" }); }
  if (!data || typeof data !== "object" || Array.isArray(data)) throw new SchemaError("Bundle is not an object", { code: "BUNDLE_SCHEMA" });
  if (data.format !== BACKUP_FORMAT) throw new SchemaError(`Unexpected bundle format: ${data.format}`, { code: "BUNDLE_FORMAT" });
  if (data.version === 1) throw new SchemaError("Version-1 bundles import via the profile migration (Phase 6)", { code: "BUNDLE_VERSION_1" });
  if (data.version !== BACKUP_VERSION) throw new SchemaError(`Unsupported bundle version: ${data.version}`, { code: "BUNDLE_VERSION" });
  if (!data.workspace || typeof data.workspace !== "object") throw new SchemaError("Bundle missing workspace", { code: "BUNDLE_SCHEMA" });
  if (!Array.isArray(data.files)) throw new SchemaError("Bundle missing files array", { code: "BUNDLE_SCHEMA" });
  return data;
}

// Full staging validation: envelope, sidecar, every path (safe + unique), every
// canvas, and entity duplicate-ID / malformed-entity detection (plan §15.3).
// Throws on any hard error; returns soft diagnostics (missing references,
// malformed/duplicate entities) for the caller to summarize. No vault writes.
export async function validateBundle(data) {
  const sidecar = parseSidecar(JSON.stringify(data.workspace));
  const seen = new Set();
  const paths = new Set();
  const diagnostics = [];
  const entityIds = new Map(); // orbitId -> [paths]

  for (const file of data.files) {
    if (!file || typeof file.path !== "string" || typeof file.text !== "string") {
      throw new SchemaError("Bundle file entry is malformed", { code: "BUNDLE_SCHEMA" });
    }
    const p = assertSafePath(file.path); // throws PathError on an unsafe path
    const fold = caseFoldKey(p);
    if (seen.has(fold)) throw new PathError(`Duplicate path in bundle: ${p}`, { code: "PATH_CASE_COLLISION" });
    seen.add(fold);
    paths.add(p);

    const { record } = await buildSourceRecord(p, file.text, { mediaType: file.mediaType });
    if (p.endsWith(".canvas") && record.parseStatus === "error") {
      throw new SchemaError(`Canvas failed validation: ${p}`, { code: "CANVAS_INVALID" });
    }
    if (record.parseStatus === "error") {
      diagnostics.push({ path: p, code: "ENTITY_MALFORMED", message: `Malformed entity: ${record.parseError}` });
    } else if (record.entityId) {
      if (!entityIds.has(record.entityId)) entityIds.set(record.entityId, []);
      entityIds.get(record.entityId).push(p);
    }
  }

  for (const [id, dupePaths] of entityIds) {
    if (dupePaths.length > 1) {
      diagnostics.push({ path: dupePaths[0], code: "DUPLICATE_ID", message: `Duplicate orbit-id ${id}`, details: { orbitId: id, paths: dupePaths } });
    }
  }
  for (const record of Object.values(sidecar.canvases)) {
    if (!paths.has(record.path)) diagnostics.push({ path: record.path, code: "CANVAS_MISSING", message: `Sidecar references missing canvas: ${record.path}` });
  }
  return { sidecar, diagnostics, fileCount: data.files.length, entityCount: entityIds.size };
}

// Restore a validated bundle into a staging vault (plan §15.3). Files are written
// first and the sidecar last so an interrupted restore never advances hierarchy.
// The caller rebuilds the index from staging and switches the active workspace
// only after validation + indexing succeed.
export async function importBundle(vault, bundleText) {
  const data = parseBundle(bundleText);
  const summary = await validateBundle(data); // throws before any write on hard errors
  for (const file of data.files) {
    await vault.write(assertSafePath(file.path), file.text, { mediaType: file.mediaType || mediaTypeFor(file.path) });
  }
  await vault.write(SIDECAR_PATH, JSON.stringify(summary.sidecar, null, 2) + "\n", { mediaType: "application/json" });
  return summary;
}
