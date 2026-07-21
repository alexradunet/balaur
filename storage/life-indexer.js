// LifeIndexer — projects canonical vault files into a queryable index (Phase 3).
//
// The indexer depends on two injected ports: a VaultStore (files) and an index
// (source files / typed projections / placements / diagnostics / state). It
// projects canonical vault files into the disposable in-memory query index.
//
// Parsing/validation happens OUTSIDE any index transaction; each file's
// projection is then applied atomically.

import { contentHash } from "./content-hash.js";
import { byteLength } from "./vault-path.js";
import { mediaTypeFor } from "./vault-store.js";
import { splitFrontmatter, collectKnownFields } from "./frontmatter.js";
import { ENTITY_CODECS, parseHabitEntries } from "./entity-codec.js";
import { isCanvas } from "./canvas-validate.js";

// Entity directory -> orbit-type (canonical vault layout).
const ENTITY_DIR_TO_TYPE = {
  "tasks": "task",
  "habits": "habit",
  "habit-logs": "habit-log",
  "journal": "journal",
  "events": "calendar-event",
};

export function entityTypeFromPath(path) {
  const dir = String(path).split("/")[0];
  return ENTITY_DIR_TO_TYPE[dir] || null;
}

function stripInternal(record) {
  const { _doc, _parsed, ...rest } = record;
  return rest;
}

// Classify and parse a Markdown file: an Orbit entity, an untyped note, or an
// error. Untyped notes are valid (entityType null); only malformed Orbit-marked
// files are errors (plan §14.3).
function parseMdEntity(content) {
  const fm = splitFrontmatter(content);
  if (!fm) return { kind: "untyped", parsed: null };
  const probe = collectKnownFields(fm.lines.slice(fm.openIdx + 1, fm.closeIdx), { fields: { "orbit-type": "enum" } });
  const type = probe["orbit-type"];
  if (!type) return { kind: "untyped", parsed: null };
  if (!ENTITY_CODECS[type]) {
    const err = new Error(`Unknown orbit-type: ${type}`);
    err.code = "ENTITY_UNKNOWN_TYPE";
    throw err;
  }
  const parsed = { type, ...ENTITY_CODECS[type].parse(content) };
  if (type === "habit-log") parseHabitEntries(parsed.body); // malformed marker invalidates the whole source file
  return { kind: "entity", parsed };
}

// Build the source_files record (and parsed payload) for one file.
export async function buildSourceRecord(path, content, meta = {}) {
  const hash = await contentHash(content);
  const record = {
    path,
    mediaType: meta.mediaType || mediaTypeFor(path),
    entityType: null,
    entityId: null,
    contentHash: hash,
    sizeBytes: byteLength(content),
    modifiedAt: meta.modifiedAt || null,
    indexedAt: new Date().toISOString(),
    parseStatus: "ok",
    parseError: null,
  };

  if (path.endsWith(".canvas")) {
    record.entityType = "canvas";
    try {
      const doc = JSON.parse(content);
      if (!isCanvas(doc)) throw new Error("Not a valid JSON Canvas document");
      return { record, parsed: { type: "canvas", doc } };
    } catch (err) {
      record.parseStatus = "error";
      record.parseError = `${err.code || "CANVAS_INVALID"}: ${err.message}`;
      return { record, parsed: null };
    }
  }

  if (path.endsWith(".md")) {
    // Extract identity independently of typed parsing. A malformed duplicate
    // must still suppress a valid sibling rather than becoming the winner.
    let identity = {};
    try {
      const fm = splitFrontmatter(content);
      if (fm) identity = collectKnownFields(fm.lines.slice(fm.openIdx + 1, fm.closeIdx), {
        fields: { "orbit-type": "enum", "orbit-id": "string" },
      });
    } catch (_) { /* the full parser below records the useful diagnostic */ }
    try {
      const { kind, parsed } = parseMdEntity(content);
      if (kind === "untyped") return { record, parsed: null };
      record.entityType = parsed.type;
      record.entityId = parsed.orbitId || identity["orbit-id"] || null;
      return { record, parsed };
    } catch (err) {
      record.parseStatus = "error";
      record.parseError = `${err.code || "PARSE"}: ${err.message}`;
      record.entityType = identity["orbit-type"] || entityTypeFromPath(path);
      record.entityId = identity["orbit-id"] || null;
      return { record, parsed: null };
    }
  }

  return { record, parsed: null }; // widgets and other opaque files
}

// Build the typed projection for a parsed entity (or habit-entry event rows).
export function buildEntityProjection(record, parsed) {
  const path = record.path;
  const hash = record.contentHash;
  switch (record.entityType) {
    case "task":
      return { kind: "entity", entityType: "task", row: {
        id: parsed.orbitId, sourcePath: path, sourceHash: hash, title: parsed.title,
        status: parsed.status, priority: parsed.priority, scheduledOn: parsed.scheduledOn,
        dueOn: parsed.dueOn, completedAt: parsed.completedAt, estimateMinutes: parsed.estimateMinutes,
        recurrenceJson: parsed.recurrence == null ? null : JSON.stringify(parsed.recurrence),
        createdAt: parsed.createdAt, updatedAt: parsed.updatedAt,
      } };
    case "habit":
      return { kind: "entity", entityType: "habit", row: {
        id: parsed.orbitId, sourcePath: path, sourceHash: hash, title: parsed.title,
        frequency: parsed.frequency, weekdaysJson: JSON.stringify(parsed.weekdays || []),
        target: parsed.target, unit: parsed.unit, archivedAt: parsed.archivedAt,
        createdAt: parsed.createdAt, updatedAt: parsed.updatedAt,
      } };
    case "journal":
      return { kind: "entity", entityType: "journal", row: {
        localDate: parsed.localDate, sourcePath: path, sourceHash: hash, orbitId: parsed.orbitId,
        createdAt: parsed.createdAt, updatedAt: parsed.updatedAt,
      } };
    case "calendar-event":
      return { kind: "entity", entityType: "calendar-event", row: {
        id: parsed.orbitId, sourcePath: path, sourceHash: hash, title: parsed.title,
        startsAt: parsed.startsAt, endsAt: parsed.endsAt, localDate: parsed.localDate,
        timezone: parsed.timezone, allDay: parsed.allDay ? 1 : 0, source: parsed.source,
        createdAt: parsed.createdAt, updatedAt: parsed.updatedAt,
      } };
    case "habit-log": {
      const rows = parseHabitEntries(parsed.body).map((e, i) => ({
        id: e.id || `${path}#${i}`, habitId: e.habit, sourcePath: path, sourceHash: hash,
        sourceKey: `${parsed.localDate}:${e.id || i}`, localDate: parsed.localDate,
        status: e.status, value: e.value, occurredAt: e.at, note: null,
      }));
      return { kind: "habit-entries", rows };
    }
    default:
      return null;
  }
}

// Scan a canvas document for entity file-node placements (plan §9.2, §12.1.8).
export function extractCanvasPlacements(canvasId, doc, pathToEntity) {
  const placements = [];
  const missing = [];
  for (const node of doc.nodes || []) {
    if (node.type !== "file" || typeof node.file !== "string") continue;
    if (!entityTypeFromPath(node.file)) continue; // widget/subcanvas/other, not an entity
    const ent = pathToEntity.get(node.file);
    if (!ent) { missing.push({ canvasId, nodeId: node.id, path: node.file }); continue; }
    if (!ent.entityId) continue; // entity file exists but has no orbit-id (e.g. habit-log)
    placements.push({ entityId: ent.entityId, entityType: ent.entityType, sourcePath: node.file, canvasId, nodeId: node.id });
  }
  return { placements, missing };
}

// Detect one orbit-id claimed by more than one file (plan §7.1, §12.1.5).
export function detectDuplicateIds(records) {
  const byId = new Map();
  for (const r of records) {
    if (!r.entityId) continue;
    if (!byId.has(r.entityId)) byId.set(r.entityId, []);
    byId.get(r.entityId).push(r.path);
  }
  const diagnostics = [];
  for (const [id, paths] of byId) {
    if (paths.length > 1) {
      for (const p of paths) {
        diagnostics.push({
          sourcePath: p, errorCode: "DUPLICATE_ID",
          message: `Duplicate orbit-id "${id}" across ${paths.length} files`,
          detailsJson: JSON.stringify({ orbitId: id, paths }),
        });
      }
    }
  }
  return diagnostics;
}

// --- orchestrator ------------------------------------------------------------

export class LifeIndexer {
  constructor({ vault, index, canvasIdFromPath }) {
    this.vault = vault;
    this.index = index;
    this.canvasIdFromPath = canvasIdFromPath || ((path) => String(path).split("/").pop().replace(/\.canvas$/, ""));
  }

  _applyProjection(record, parsed) {
    const proj = buildEntityProjection(record, parsed);
    if (!proj) return;
    if (proj.kind === "entity") this.index.insertEntityProjection(proj.entityType, proj.row);
    else if (proj.kind === "habit-entries") this.index.insertHabitEntries(proj.rows);
  }

  async _recordsWithParsed(replacement = null) {
    const paths = new Set(this.index.allSourceFiles().map((r) => r.path));
    if (replacement) paths.add(replacement.record.path);
    const out = [];
    for (const path of paths) {
      let content;
      if (replacement && path === replacement.record.path) content = replacement.content;
      else { try { content = await this.vault.read(path); } catch (_) { continue; } }
      const built = await buildSourceRecord(path, content, replacement && path === replacement.record.path ? replacement.meta : {});
      out.push({ ...built, record: { ...built.record, _parsed: built.parsed } });
    }
    return out;
  }

  async _reproject(records) {
    const sourceRecords = records.map((r) => r.record || r);
    const duplicatePaths = new Set(detectDuplicateIds(sourceRecords).map((d) => d.sourcePath));
    this.index.transaction(() => {
      this.index.clearAllProjections?.();
      for (const rec of records) this.index.clearProjectionForPath(rec.record.path);
      this.index.clearAllDiagnostics?.();
      for (const rec of records) {
        this.index.upsertSourceFile(stripInternal(rec.record));
        if (rec.record.parseStatus === "error") this.index.recordDiagnostic({ sourcePath: rec.record.path, errorCode: "PARSE_ERROR", message: rec.record.parseError, detailsJson: null });
        else if (!duplicatePaths.has(rec.record.path)) this._applyProjection(rec.record, rec.parsed);
      }
      for (const d of detectDuplicateIds(sourceRecords)) this.index.recordDiagnostic(d);
    });
  }

  // Index one file, then re-evaluate all identity conflicts. This deliberately
  // reprojects the small in-memory index so a new duplicate cannot leave a
  // last-writer winner behind, and removing a duplicate restores its sibling.
  async indexFile(path, content, meta = {}) {
    const built = await buildSourceRecord(path, content, meta);
    await this._reproject(await this._recordsWithParsed({ record: built.record, parsed: built.parsed, content: String(content), meta }));
    await this._rebuildAllPlacements();
    return stripInternal(built.record);
  }

  async removeFile(path) {
    this.index.deleteSourceFile(path);
    const records = await this._recordsWithParsed();
    await this._reproject(records);
    await this._rebuildAllPlacements();
  }

  // Full cold rebuild (plan §12.1). Idempotent.
  async rebuild() {
    const files = await this.vault.list();
    const records = [];
    for (const f of files) {
      const content = await this.vault.read(f.path);
      const { record, parsed } = await buildSourceRecord(f.path, content, { mediaType: f.mediaType, modifiedAt: f.modifiedAt });
      record._parsed = parsed;
      records.push(record);
    }

    this.index.transaction(() => this.index.clearAll());

    const wrapped = records.map((record) => ({ record, parsed: record._parsed }));
    await this._reproject(wrapped);
    await this._rebuildAllPlacements();

    this.index.setIndexState("indexedRevision", String(this.vault.revision));
    this.index.setIndexState("generation", String(Date.now()));
    return this.stats();
  }

  // Warm reconciliation from a vault revision (plan §12.2). Move ancestry is
  // retained while changes are coalesced: a move followed by a modify/remove on
  // the new path still removes the old indexed source.
  async reconcileWarm(fromRevision) {
    const changes = await this.vault.changesSince(fromRevision);
    const aliases = new Map(); // current path -> paths that may still be indexed
    const final = new Map();
    const removed = new Set();
    const canvasPaths = new Set();
    for (const change of changes) {
      const path = change.path;
      if (path.endsWith(".canvas")) canvasPaths.add(path);
      if (change.operation === "move" && change.oldPath) {
        canvasPaths.add(change.oldPath);
        const oldAliases = aliases.get(change.oldPath) || new Set([change.oldPath]);
        aliases.delete(change.oldPath);
        aliases.set(path, new Set([...oldAliases, change.oldPath]));
        for (const oldPath of oldAliases) removed.add(oldPath);
        final.set(path, { ...change, oldPaths: oldAliases });
      } else if (change.operation === "remove") {
        const oldAliases = aliases.get(path) || new Set();
        for (const oldPath of oldAliases) removed.add(oldPath);
        removed.add(path);
        aliases.delete(path);
        final.delete(path);
      } else {
        const oldAliases = aliases.get(path) || new Set();
        final.set(path, { ...change, oldPaths: oldAliases });
      }
    }
    for (const path of removed) {
      if (final.has(path)) continue;
      await this.removeFile(path);
    }
    for (const [path] of final) {
      let content;
      try { content = await this.vault.read(path); }
      catch (_) { await this.removeFile(path); continue; }
      await this.indexFile(path, content, {});
    }
    if (canvasPaths.size) await this._rebuildAllPlacements();
    this.index.setIndexState("indexedRevision", String(this.vault.revision));
    return this.stats();
  }

  // TODO(deferred, plan S14): avoid rewriting/reindexing unchanged canvases and
  // prune the in-memory adapter's change journal when a persistent revision
  // checkpoint is available.
  async _rebuildAllPlacements() {
    const pathToEntity = new Map();
    const byId = new Map();
    for (const rec of this.index.allSourceFiles()) if (rec.entityType && rec.entityType !== "canvas" && rec.entityId) {
      if (!byId.has(rec.entityId)) byId.set(rec.entityId, []);
      byId.get(rec.entityId).push(rec);
    }
    for (const rec of this.index.allSourceFiles()) {
      if (rec.entityType && rec.entityType !== "canvas" && rec.entityId && byId.get(rec.entityId)?.length === 1) {
        pathToEntity.set(rec.path, { entityId: rec.entityId, entityType: rec.entityType });
      }
    }
    this.index.transaction(() => this.index.clearAllPlacements());
    const canvasFiles = (await this.vault.list()).filter((f) => f.path.endsWith(".canvas"));
    for (const f of canvasFiles) {
      const content = await this.vault.read(f.path);
      let doc;
      try { doc = JSON.parse(content); } catch { continue; }
      if (!isCanvas(doc)) continue;
      const canvasId = this.canvasIdFromPath(f.path);
      const { placements, missing } = extractCanvasPlacements(canvasId, doc, pathToEntity);
      this.index.transaction(() => this.index.replaceCanvasPlacements(canvasId, placements));
      for (const miss of missing) {
        this.index.recordDiagnostic({
          sourcePath: miss.path, errorCode: "MISSING_REFERENCE",
          message: `Canvas "${miss.canvasId}" node "${miss.nodeId}" references a missing entity file`,
          detailsJson: JSON.stringify(miss),
        });
      }
    }
  }

  stats() {
    return {
      // The sidecar is workspace metadata, not a source file shown in the
      // life-data count. It may still be retained in the disposable index.
      sourceFiles: this.index.allSourceFiles().filter((record) => record.path !== ".orbit/workspace.json").length,
      tasks: this.index.allTasks().length,
      habits: this.index.allHabits().length,
      placements: this.index.allPlacements().length,
      diagnostics: this.index.allDiagnostics().length,
      indexedRevision: this.index.getIndexState("indexedRevision"),
      generation: this.index.getIndexState("generation"),
    };
  }
}
