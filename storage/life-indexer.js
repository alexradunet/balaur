// LifeIndexer — projects canonical vault files into a queryable index (Phase 3).
//
// The indexer depends on two injected ports: a VaultStore (files) and an index
// (source_files / typed projections / placements / diagnostics / state). It never
// imports the SQLite store directly, so its logic is Node-testable against
// MemoryVault + MemoryIndex, and the same class drives SqliteLifeStore in the
// browser (plan §11.4 per-file replacement, §12 index lifecycle).
//
// Parsing/validation happens OUTSIDE any index transaction; each file's
// projection is then applied atomically (plan §11.4).

import { contentHash } from "./content-hash.js";
import { byteLength } from "./vault-path.js";
import { mediaTypeFor } from "./vault-store.js";
import { splitFrontmatter, collectKnownFields } from "./frontmatter.js";
import { ENTITY_CODECS, parseHabitEntries } from "./entity-codec.js";

// Entity directory -> orbit-type (plan §6 layout).
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

export function isCanvasDoc(doc) {
  return !!doc && typeof doc === "object" && Array.isArray(doc.nodes) && Array.isArray(doc.edges);
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
  return { kind: "entity", parsed: { type, ...ENTITY_CODECS[type].parse(content) } };
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
      if (!isCanvasDoc(doc)) throw new Error("Not a valid JSON Canvas document");
      return { record, parsed: { type: "canvas", doc } };
    } catch (err) {
      record.parseStatus = "error";
      record.parseError = `${err.code || "CANVAS_INVALID"}: ${err.message}`;
      return { record, parsed: null };
    }
  }

  if (path.endsWith(".md")) {
    try {
      const { kind, parsed } = parseMdEntity(content);
      if (kind === "untyped") return { record, parsed: null };
      record.entityType = parsed.type;
      record.entityId = parsed.orbitId || null;
      return { record, parsed };
    } catch (err) {
      record.parseStatus = "error";
      record.parseError = `${err.code || "PARSE"}: ${err.message}`;
      record.entityType = entityTypeFromPath(path);
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

  // Index one file: parse outside the transaction, then apply atomically.
  async indexFile(path, content, meta = {}) {
    const { record, parsed } = await buildSourceRecord(path, content, meta);
    this.index.transaction(() => {
      this.index.clearProjectionForPath(path);
      this.index.clearDiagnostics(path);
      this.index.upsertSourceFile(stripInternal(record));
      if (record.parseStatus === "error") {
        this.index.recordDiagnostic({ sourcePath: path, errorCode: "PARSE_ERROR", message: record.parseError, detailsJson: null });
      } else {
        this._applyProjection(record, parsed);
      }
    });
    return stripInternal(record);
  }

  removeFile(path) {
    const prior = this.index.getSourceFile(path);
    this.index.transaction(() => {
      this.index.clearProjectionForPath(path);
      this.index.deleteSourceFile(path);
      this.index.clearDiagnostics(path);
      // Plan §11.5: removing an entity drops its placements and flags canvas
      // nodes that still reference the now-missing file.
      if (prior && prior.entityId) {
        const dangling = this.index.placementsForEntity(prior.entityId);
        this.index.removePlacementsForEntity(prior.entityId);
        for (const p of dangling) {
          this.index.recordDiagnostic({
            sourcePath: path, errorCode: "MISSING_REFERENCE",
            message: `Canvas "${p.canvasId}" node "${p.nodeId}" references a deleted entity file`,
            detailsJson: JSON.stringify(p),
          });
        }
      }
    });
  }

  // Full cold rebuild (plan §12.1). Idempotent.
  async rebuild({ onProgress } = {}) {
    const files = await this.vault.list();
    const records = [];
    let n = 0;
    for (const f of files) {
      const content = await this.vault.read(f.path);
      const { record, parsed } = await buildSourceRecord(f.path, content, { mediaType: f.mediaType, modifiedAt: f.modifiedAt });
      record._parsed = parsed;
      records.push(record);
      if (onProgress) onProgress({ phase: "parse", done: ++n, total: files.length });
    }

    this.index.transaction(() => this.index.clearAll());

    let m = 0;
    for (const record of records) {
      this.index.transaction(() => {
        this.index.upsertSourceFile(stripInternal(record));
        if (record.parseStatus === "error") {
          this.index.recordDiagnostic({ sourcePath: record.path, errorCode: "PARSE_ERROR", message: record.parseError, detailsJson: null });
        } else {
          this._applyProjection(record, record._parsed);
        }
      });
      if (onProgress) onProgress({ phase: "index", done: ++m, total: records.length });
    }

    for (const d of detectDuplicateIds(records)) this.index.recordDiagnostic(d);
    await this._rebuildAllPlacements();

    this.index.setIndexState("indexedRevision", String(this.vault.revision));
    this.index.setIndexState("generation", String(Date.now()));
    return this.stats();
  }

  // Warm reconciliation from a vault revision (plan §12.2).
  async reconcileWarm(fromRevision) {
    const changes = await this.vault.changesSince(fromRevision);
    const byPath = new Map();
    for (const c of changes) byPath.set(c.path, c); // coalesce: last operation per path wins
    let canvasChanged = false;
    for (const [path, change] of byPath) {
      if (path.endsWith(".canvas")) canvasChanged = true;
      if (change.operation === "remove") { this.removeFile(path); continue; }
      let content;
      try { content = await this.vault.read(path); }
      catch { this.removeFile(path); continue; }
      await this.indexFile(path, content, {});
    }
    if (canvasChanged) await this._rebuildAllPlacements();
    this.index.setIndexState("indexedRevision", String(this.vault.revision));
    return this.stats();
  }

  async _rebuildAllPlacements() {
    const pathToEntity = new Map();
    for (const rec of this.index.allSourceFiles()) {
      if (rec.entityType && rec.entityType !== "canvas") {
        pathToEntity.set(rec.path, { entityId: rec.entityId, entityType: rec.entityType });
      }
    }
    this.index.transaction(() => this.index.clearAllPlacements());
    const canvasFiles = (await this.vault.list()).filter((f) => f.path.endsWith(".canvas"));
    for (const f of canvasFiles) {
      const content = await this.vault.read(f.path);
      let doc;
      try { doc = JSON.parse(content); } catch { continue; }
      if (!isCanvasDoc(doc)) continue;
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
      sourceFiles: this.index.allSourceFiles().length,
      tasks: this.index.allTasks().length,
      habits: this.index.allHabits().length,
      placements: this.index.allPlacements().length,
      diagnostics: this.index.allDiagnostics().length,
      indexedRevision: this.index.getIndexState("indexedRevision"),
      generation: this.index.getIndexState("generation"),
    };
  }
}
