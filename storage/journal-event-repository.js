// File-canonical journal and calendar-event repositories (Phase 8, ADR-0001).
//
// Journals are one canonical Markdown file per local date (journal/YYYY/YYYY-MM-DD.md).
// Calendar events are canonical Markdown files under events/. Both are projections
// over the life layer: writes go to the canonical file first, then reindex through
// the LifeIndexer. Platform-neutral and asynchronous — tested against MemoryVault
// + MemoryIndex (storage/phase8.test.js).

import { serializeJournal, parseJournal, serializeCalendarEvent, parseCalendarEvent, CalendarEventCodec, JournalCodec } from "./entity-codec.js";
import { patchFields, replaceBody, isValidLocalDate, localDateForInstant } from "./frontmatter.js";
import { entityPath } from "./vault-path.js";
import { isCanvas } from "./canvas-validate.js";
import { SchemaError } from "./vault-errors.js";

function assertLocalDate(localDate) {
  if (!isValidLocalDate(String(localDate))) throw new SchemaError(`Bad local date: ${localDate}`, { code: "BAD_LOCAL_DATE" });
  return String(localDate);
}

// Logical path for a day's journal file (plan §6 layout).
export function journalPath(localDate) {
  const d = assertLocalDate(localDate);
  return `journal/${d.slice(0, 4)}/${d}.md`;
}

export class FileJournalRepository {
  constructor({ vault, index, indexer, canvasPathFromId, now = () => new Date().toISOString() }) {
    this.vault = vault;
    this.index = index;
    this.indexer = indexer;
    this.canvasPathFromId = canvasPathFromId || (() => null);
    this.now = now;
  }

  async createJournal({ localDate, body = "", id = null }) {
    const d = assertLocalDate(localDate);
    const ts = this.now();
    const journal = { orbitId: id || `journal-${d}`, localDate: d, createdAt: ts, updatedAt: ts, body };
    const path = journalPath(d);
    const content = serializeJournal(journal);
    await this.vault.write(path, content, { expectedHash: null });
    await this.indexer.indexFile(path, content, {});
    return { localDate: d, path, journal };
  }

  async getJournal(localDate) {
    return parseJournal(await this.vault.read(journalPath(localDate)));
  }

  // Journals are app-owned and round-trip losslessly, so rebuild from the parsed
  // entry with the new body and bump updated-at.
  async updateJournal(localDate, { body } = {}) {
    const path = journalPath(localDate);
    const stat = await this.vault.stat(path);
    if (!stat) throw new SchemaError(`Journal not found: ${localDate}`, { code: "JOURNAL_NOT_FOUND" });
    const existingText = await this.vault.read(path);
    const existing = parseJournal(existingText);
    const patched = patchFields(existingText, { "updated-at": this.now() }, JournalCodec.spec);
    const content = replaceBody(patched, body ?? existing.body);
    const parsed = parseJournal(content);
    await this.vault.write(path, content, { expectedHash: stat.hash });
    await this.indexer.indexFile(path, content, {});
    return parsed;
  }

  async addPlacement(localDate, canvasId, geometry = {}) {
    const path = journalPath(localDate);
    const stat = await this.vault.stat(path);
    if (!stat) throw new SchemaError(`Journal not found: ${localDate}`, { code: "JOURNAL_NOT_FOUND" });
    const canvasPath = this.canvasPathFromId(canvasId);
    if (!canvasPath) throw new SchemaError(`No canvas path for id: ${canvasId}`, { code: "CANVAS_NOT_FOUND" });
    const canvasStat = await this.vault.stat(canvasPath);
    if (!canvasStat) throw new SchemaError(`Canvas not found: ${canvasPath}`, { code: "CANVAS_NOT_FOUND" });
    let doc;
    try { doc = JSON.parse(await this.vault.read(canvasPath)); }
    catch (err) { throw new SchemaError(`Invalid canvas document: ${canvasPath}`, { code: "CANVAS_INVALID", cause: err }); }
    if (!isCanvas(doc)) throw new SchemaError(`Invalid canvas document: ${canvasPath}`, { code: "CANVAS_INVALID" });
    const g = { x: 40, y: 40, width: 320, height: 200, ...geometry };
    const nodeId = geometry.id || `node-${localDate}`;
    if (doc.nodes.some((n) => n.id === nodeId) || doc.edges.some((e) => e.id === nodeId)) throw new SchemaError(`Canvas id already exists: ${nodeId}`, { code: "CANVAS_ID_DUPLICATE" });
    doc.nodes.push({ id: nodeId, type: "file", file: path, x: g.x, y: g.y, width: g.width, height: g.height, ...(g.color ? { color: g.color } : {}) });
    if (!isCanvas(doc)) throw new SchemaError(`Invalid canvas document: ${canvasPath}`, { code: "CANVAS_INVALID" });
    const content = JSON.stringify(doc, null, 2) + "\n";
    await this.vault.write(canvasPath, content, { expectedHash: canvasStat.hash });
    await this.indexer.indexFile(canvasPath, content, {});
    return { canvasId, nodeId, canvasPath, path };
  }
}

const EVENT_PATCH_KEYS = {
  title: "title", startsAt: "starts-at", endsAt: "ends-at",
  localDate: "local-date", timezone: "timezone", allDay: "all-day",
};

function randomToken() {
  const c = globalThis.crypto;
  if (c?.randomUUID) return c.randomUUID().replace(/-/g, "").slice(0, 12);
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
}

export class FileEventRepository {
  constructor({ vault, index, indexer, now = () => new Date().toISOString(), idPrefix = "event" }) {
    this.vault = vault;
    this.index = index;
    this.indexer = indexer;
    this.now = now;
    this.idPrefix = idPrefix;
  }

  _newId() { return `${this.idPrefix}-${randomToken()}`; }

  async createEvent(input = {}) {
    const title = String(input.title ?? "").trim();
    if (!title) throw new SchemaError("Event title is required", { code: "EVENT_TITLE_REQUIRED" });
    if (!input.startsAt) throw new SchemaError("Event starts-at is required", { code: "EVENT_STARTS_REQUIRED" });
    const id = input.id || this._newId();
    const ts = this.now();
    const event = {
      orbitId: id, title,
      startsAt: input.startsAt,
      endsAt: input.endsAt ?? null,
      timezone: input.timezone || "UTC",
      localDate: assertLocalDate(input.localDate || localDateForInstant(input.startsAt, input.timezone || "UTC")),
      allDay: input.allDay ?? false,
      source: input.source || "orbit",
      createdAt: ts, updatedAt: ts,
      body: input.body || "",
    };
    const path = input.path || entityPath("events", title, id, "md");
    const content = serializeCalendarEvent(event);
    await this.vault.write(path, content, { expectedHash: null });
    await this.indexer.indexFile(path, content, {});
    return { id, path, event };
  }

  _sourceFor(id) {
    const row = this.index.allEvents().find((e) => e.id === id);
    if (!row) throw new SchemaError(`Event not found: ${id}`, { code: "EVENT_NOT_FOUND" });
    return { path: row.sourcePath, hash: row.sourceHash };
  }

  async getEvent(id) {
    const { path } = this._sourceFor(id);
    return parseCalendarEvent(await this.vault.read(path));
  }

  async updateEvent(id, patch = {}) {
    const { path, hash } = this._sourceFor(id);
    const content = await this.vault.read(path);
    const fmPatch = { "updated-at": this.now() };
    for (const [key, value] of Object.entries(patch)) {
      if (key === "body") continue;
      const fmKey = EVENT_PATCH_KEYS[key];
      if (!fmKey) throw new SchemaError(`Unknown event field: ${key}`, { code: "EVENT_UNKNOWN_FIELD" });
      fmPatch[fmKey] = value;
    }
    let next = patchFields(content, fmPatch, CalendarEventCodec.spec);
    if ("body" in patch) next = replaceBody(next, patch.body);
    const parsed = parseCalendarEvent(next);
    await this.vault.write(path, next, { expectedHash: hash });
    await this.indexer.indexFile(path, next, {});
    return parsed;
  }

  async deleteEvent(id) {
    const { path, hash } = this._sourceFor(id);
    await this.vault.remove(path, { expectedHash: hash });
    await this.indexer.removeFile(path);
    return { id, path };
  }
}
