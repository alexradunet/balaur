// File-canonical habit repository (Phase 7, ADR-0001, plan §9.4/§13).
//
// Habit definitions are canonical Markdown files under habits/. Daily check-ins
// are append-only event markers in a per-day habit-log file
// (habits/habit-logs/YYYY/YYYY-MM-DD.md); the latest event per (habit, local-date) is a
// projection, while the daily history is preserved (plan: habits are event logs,
// not recurring tasks). Writes go to the canonical files first, then reindex
// through the LifeIndexer. Platform-neutral and asynchronous — tested against
// MemoryVault + MemoryIndex (storage/phase7.test.js).

import { serializeHabit, parseHabit, HabitCodec, serializeHabitLog, serializeHabitEntry } from "./entity-codec.js";
import { patchFields, replaceBody, localDateForInstant, isValidLocalDate } from "./frontmatter.js";
import { entityPath } from "./vault-path.js";
import { SchemaError } from "./vault-errors.js";

const PATCH_KEYS = {
  title: "title", frequency: "frequency", weekdays: "weekdays",
  target: "target", unit: "unit", archivedAt: "archived-at",
};

function randomToken() {
  const c = globalThis.crypto;
  if (c?.randomUUID) return c.randomUUID().replace(/-/g, "").slice(0, 12);
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
}

// Logical path for a day's habit-log file (plan §6 layout).
export function habitLogPath(localDate) {
  if (!isValidLocalDate(String(localDate))) throw new SchemaError(`Bad local date: ${localDate}`, { code: "BAD_LOCAL_DATE" });
  return `habits/habit-logs/${String(localDate).slice(0, 4)}/${localDate}.md`;
}

export class FileHabitRepository {
  constructor({ vault, index, indexer, now = () => new Date().toISOString(), idPrefix = "habit" }) {
    this.vault = vault;
    this.index = index;
    this.indexer = indexer;
    this.now = now;
    this.idPrefix = idPrefix;
  }

  _newId() { return `${this.idPrefix}-${randomToken()}`; }

  async createHabit(input = {}) {
    const title = String(input.title ?? "").trim();
    if (!title) throw new SchemaError("Habit title is required", { code: "HABIT_TITLE_REQUIRED" });
    const id = input.id || this._newId();
    const ts = this.now();
    const habit = {
      balaurId: id, title,
      frequency: input.frequency || "daily",
      weekdays: input.weekdays || [],
      target: input.target ?? null,
      unit: input.unit ?? null,
      archivedAt: null,
      createdAt: ts, updatedAt: ts,
      body: input.body || "",
    };
    const path = input.path || entityPath("habits", title, id, "md");
    const content = serializeHabit(habit);
    await this.vault.write(path, content, { expectedHash: null });
    await this.indexer.indexFile(path, content, {});
    return { id, path, habit };
  }

  _sourceFor(id) {
    const row = this.index.allHabits().find((h) => h.id === id);
    if (!row) throw new SchemaError(`Habit not found: ${id}`, { code: "HABIT_NOT_FOUND" });
    return { path: row.sourcePath, hash: row.sourceHash };
  }

  async getHabit(id) {
    const { path } = this._sourceFor(id);
    return parseHabit(await this.vault.read(path));
  }

  // Preservation-first update of definition fields; bumps updated-at.
  async updateHabit(id, patch = {}) {
    const { path, hash } = this._sourceFor(id);
    const content = await this.vault.read(path);
    const fmPatch = { "updated-at": this.now() };
    for (const [key, value] of Object.entries(patch)) {
      if (key === "body") continue;
      const fmKey = PATCH_KEYS[key];
      if (!fmKey) throw new SchemaError(`Unknown habit field: ${key}`, { code: "HABIT_UNKNOWN_FIELD" });
      fmPatch[fmKey] = value;
    }
    let next = patchFields(content, fmPatch, HabitCodec.spec);
    if ("body" in patch) next = replaceBody(next, patch.body);
    // Validate the patched definition before writing: patchFields does not
    // enforce the habit frequency/weekday domain.
    const parsed = parseHabit(next);
    await this.vault.write(path, next, { expectedHash: hash });
    await this.indexer.indexFile(path, next, {});
    return parsed;
  }

  async archiveHabit(id) { return this.updateHabit(id, { archivedAt: this.now() }); }

  // Append a check-in event to the day's habit-log (creating it if needed).
  // The marker grammar is constrained: no arbitrary user text (plan §9.4).
  async checkIn(habitId, opts = {}) {
    this._sourceFor(habitId); // habit must exist
    const localDate = opts.localDate || localDateForInstant(opts.at || this.now(), opts.timezone || "UTC");
    const status = opts.status || "done";
    const value = Number(opts.value ?? 1);
    const at = opts.at || this.now();
    const entryId = randomToken();
    const marker = serializeHabitEntry({ id: entryId, habit: habitId, status, value, at });

    const logPath = habitLogPath(localDate);
    const stat = await this.vault.stat(logPath);
    let content;
    if (stat) {
      const existing = await this.vault.read(logPath);
      const eol = existing.includes("\r\n") ? "\r\n" : "\n";
      // Preserve every existing byte, including trailing spaces and the file's
      // line-ending convention. Add only the separator required by the file.
      content = existing.endsWith(eol) ? `${existing}${marker}${eol}` : `${existing}${eol}${marker}${eol}`;
      await this.vault.write(logPath, content, { expectedHash: stat.hash });
    } else {
      content = serializeHabitLog({ localDate, body: `${marker}\n` });
      await this.vault.write(logPath, content, { expectedHash: null });
    }
    await this.indexer.indexFile(logPath, content, {});
    return { habitId, localDate, entryId, logPath, status, value };
  }

  // Projected check-in events for a habit (from the index), newest first.
  entriesFor(habitId) {
    return this.index.allHabitEntries()
      .filter((e) => e.habitId === habitId)
      .sort((a, b) => (a.localDate < b.localDate ? 1 : a.localDate > b.localDate ? -1 : 0));
  }
}
