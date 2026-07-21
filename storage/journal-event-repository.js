// File-canonical journal and calendar-event repositories (Phase 8, ADR-0001).
//
// Journals are one canonical Markdown file per local date (journal/YYYY/YYYY-MM-DD.md).
// Calendar events are canonical Markdown files under events/. Both are projections
// over the life layer: writes go to the canonical file first, then reindex through
// the LifeIndexer. Platform-neutral and asynchronous — tested against MemoryVault
// + MemoryIndex (storage/phase8.test.js).

import { serializeJournal, parseJournal, serializeCalendarEvent, parseCalendarEvent, CalendarEventCodec } from "./entity-codec.js";
import { patchFields } from "./frontmatter.js";
import { entityPath } from "./vault-path.js";
import { SchemaError } from "./vault-errors.js";

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

function assertLocalDate(localDate) {
  if (!DATE_RE.test(String(localDate))) throw new SchemaError(`Bad local date: ${localDate}`, { code: "BAD_LOCAL_DATE" });
  return String(localDate);
}

// Logical path for a day's journal file (plan §6 layout).
export function journalPath(localDate) {
  const d = assertLocalDate(localDate);
  return `journal/${d.slice(0, 4)}/${d}.md`;
}

export class FileJournalRepository {
  constructor({ vault, index, indexer, now = () => new Date().toISOString() }) {
    this.vault = vault;
    this.index = index;
    this.indexer = indexer;
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
    const existing = parseJournal(await this.vault.read(path));
    const updated = { ...existing, body: body ?? existing.body, updatedAt: this.now() };
    const content = serializeJournal(updated);
    await this.vault.write(path, content, { expectedHash: stat.hash });
    await this.indexer.indexFile(path, content, {});
    return updated;
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
      localDate: assertLocalDate(input.localDate || String(input.startsAt).slice(0, 10)),
      timezone: input.timezone || "UTC",
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
    if ("body" in patch) {
      const parsed = parseCalendarEvent(content);
      next = serializeCalendarEvent({ ...parsed, ...patch, orbitId: id, updatedAt: this.now() });
    }
    await this.vault.write(path, next, { expectedHash: hash });
    await this.indexer.indexFile(path, next, {});
    return parseCalendarEvent(next);
  }

  async deleteEvent(id) {
    const { path } = this._sourceFor(id);
    await this.vault.remove(path);
    this.indexer.removeFile(path);
    return { id, path };
  }
}
