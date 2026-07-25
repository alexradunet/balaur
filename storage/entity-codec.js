// Canonical entity codecs (Phase 1, plan §9).
//
// Each entity is a Markdown file: constrained frontmatter (balaur-schema 1) plus
// an ordinary Markdown body. Field keys are kebab-case in files and camelCase in
// application objects. Parsing validates required fields, enums, dates, and the
// balaur-schema version; unknown frontmatter is preserved by the codec layer.

import { splitFrontmatter, collectKnownFields, serializeFrontmatter, isValidInstant, isValidIanaTimezone } from "./frontmatter.js";
import { ParseError, SchemaError } from "./vault-errors.js";

export const SUPPORTED_SCHEMA = 1;

function kebabToCamel(k) { return k.replace(/-([a-z])/g, (_, c) => c.toUpperCase()); }

function validateSchema(schema) {
  if (schema === undefined || schema === null) throw new SchemaError("Missing balaur-schema", { code: "SCHEMA_MISSING" });
  if (typeof schema !== "number" || !Number.isFinite(schema)) throw new SchemaError("Invalid balaur-schema", { code: "SCHEMA_INVALID" });
  if (schema > SUPPORTED_SCHEMA) throw new SchemaError(`Unsupported balaur-schema ${schema} (read-only)`, { code: "SCHEMA_NEWER" });
  if (schema !== SUPPORTED_SCHEMA) throw new SchemaError(`Unsupported balaur-schema ${schema}`, { code: "SCHEMA_UNSUPPORTED" });
}

const TASK_STATUSES = new Set(["inbox", "next", "scheduled", "waiting", "done", "cancelled"]);
const HABIT_FREQUENCIES = new Set(["daily", "weekly", "monthly"]);
const HABIT_ENTRY_STATUSES = new Set(["done", "skipped", "missed"]);

function validateDomain(type, out) {
  if (type === "task" && !TASK_STATUSES.has(out.status)) {
    throw new SchemaError(`Invalid task status: ${out.status}`, { code: "TASK_BAD_STATUS" });
  }
  if (type === "habit") {
    if (!HABIT_FREQUENCIES.has(out.frequency)) throw new SchemaError(`Invalid habit frequency: ${out.frequency}`, { code: "HABIT_BAD_FREQUENCY" });
    if (out.weekdays !== null && (!Array.isArray(out.weekdays) || out.weekdays.some((d) => !Number.isInteger(d) || d < 1 || d > 7) || new Set(out.weekdays).size !== out.weekdays.length)) {
      throw new SchemaError("Habit weekdays must be unique integers from 1 through 7", { code: "HABIT_BAD_WEEKDAY" });
    }
  }
  if (type === "calendar-event" && !isValidIanaTimezone(out.timezone)) {
    throw new SchemaError(`Invalid IANA timezone: ${out.timezone}`, { code: "EVENT_BAD_TIMEZONE" });
  }
}

function validateEntityValues(type, entity) {
  validateDomain(type, entity);
  return entity;
}

function makeCodec({ type, fields, order, required }) {
  const spec = { fields };

  function serialize(entity) {
    const fmFields = Object.create(null);
    for (const key of order) {
      const camel = kebabToCamel(key);
      let value = entity[camel];
      if (key === "balaur-schema") value = SUPPORTED_SCHEMA;
      else if (key === "balaur-type") value = type;
      else if (value === undefined) {
        if (required.includes(key)) {
          throw new ParseError(`Missing required field: ${key}`, { code: "ENTITY_MISSING_FIELD", details: { key } });
        }
        value = null;
      }
      fmFields[key] = value;
    }
    validateEntityValues(type, Object.fromEntries(order.filter((key) => !["balaur-schema", "balaur-type"].includes(key)).map((key) => [kebabToCamel(key), fmFields[key]])));
    const fm = serializeFrontmatter(fmFields, spec, order);
    const body = entity.body ?? "";
    return fm + (body ? "\n" + body : "");
  }

  function parse(text) {
    const fm = splitFrontmatter(text);
    if (!fm) throw new ParseError("Missing or unterminated frontmatter", { code: "FM_NO_DELIMITER" });
    const raw = collectKnownFields(fm.lines.slice(fm.openIdx + 1, fm.closeIdx), spec);
    validateSchema(raw["balaur-schema"]);
    if (raw["balaur-type"] !== type) {
      throw new ParseError(`Expected balaur-type "${type}", got "${raw["balaur-type"]}"`, { code: "ENTITY_TYPE_MISMATCH" });
    }
    for (const key of required) {
      if (key === "balaur-schema" || key === "balaur-type") continue;
      if (raw[key] === undefined || raw[key] === null) {
        throw new ParseError(`Missing required field: ${key}`, { code: "ENTITY_MISSING_FIELD", details: { key } });
      }
    }
    const out = Object.create(null);
    for (const key of order) {
      if (key === "balaur-schema" || key === "balaur-type") continue;
      out[kebabToCamel(key)] = raw[key] === undefined ? null : raw[key];
    }
    let body = fm.lines.slice(fm.closeIdx + 1).join("");
    if (body.startsWith(fm.term)) body = body.slice(fm.term.length); // drop one separator without changing CRLF fidelity
    out.body = body;
    validateEntityValues(type, out);
    return out;
  }

  return { type, spec, order, required, serialize, parse };
}

// --- field specifications ----------------------------------------------------

export const TaskCodec = makeCodec({
  type: "task",
  fields: {
    "balaur-schema": "number", "balaur-type": "enum", "balaur-id": "string", "title": "string",
    "status": "enum", "priority": "number", "scheduled-on": "date", "due-on": "date",
    "completed-at": "instant", "estimate-minutes": "number", "recurrence": "string",
    "created-at": "instant", "updated-at": "instant",
  },
  order: ["balaur-schema", "balaur-type", "balaur-id", "title", "status", "priority",
    "scheduled-on", "due-on", "completed-at", "estimate-minutes", "recurrence",
    "created-at", "updated-at"],
  required: ["balaur-id", "title", "status", "created-at", "updated-at"],
});

export const HabitCodec = makeCodec({
  type: "habit",
  fields: {
    "balaur-schema": "number", "balaur-type": "enum", "balaur-id": "string", "title": "string",
    "frequency": "enum", "weekdays": "number[]", "target": "number", "unit": "string",
    "archived-at": "instant", "created-at": "instant", "updated-at": "instant",
  },
  order: ["balaur-schema", "balaur-type", "balaur-id", "title", "frequency", "weekdays",
    "target", "unit", "archived-at", "created-at", "updated-at"],
  required: ["balaur-id", "title", "frequency", "created-at", "updated-at"],
});

export const HabitLogCodec = makeCodec({
  type: "habit-log",
  fields: { "balaur-schema": "number", "balaur-type": "enum", "local-date": "date" },
  order: ["balaur-schema", "balaur-type", "local-date"],
  required: ["local-date"],
});

export const JournalCodec = makeCodec({
  type: "journal",
  fields: {
    "balaur-schema": "number", "balaur-type": "enum", "balaur-id": "string", "local-date": "date",
    "created-at": "instant", "updated-at": "instant",
  },
  order: ["balaur-schema", "balaur-type", "balaur-id", "local-date", "created-at", "updated-at"],
  required: ["balaur-id", "local-date", "created-at", "updated-at"],
});

export const CalendarEventCodec = makeCodec({
  type: "calendar-event",
  fields: {
    "balaur-schema": "number", "balaur-type": "enum", "balaur-id": "string", "title": "string",
    "starts-at": "instant", "ends-at": "instant", "local-date": "date", "timezone": "string",
    "all-day": "boolean", "source": "enum", "created-at": "instant", "updated-at": "instant",
  },
  order: ["balaur-schema", "balaur-type", "balaur-id", "title", "starts-at", "ends-at",
    "local-date", "timezone", "all-day", "source", "created-at", "updated-at"],
  required: ["balaur-id", "title", "starts-at", "local-date", "timezone", "created-at", "updated-at"],
});

export const ENTITY_CODECS = {
  "task": TaskCodec,
  "habit": HabitCodec,
  "habit-log": HabitLogCodec,
  "journal": JournalCodec,
  "calendar-event": CalendarEventCodec,
};

// Convenience wrappers.
export const serializeTask = (t) => TaskCodec.serialize(t);
export const parseTask = (text) => TaskCodec.parse(text);
export const serializeHabit = (h) => HabitCodec.serialize(h);
export const parseHabit = (text) => HabitCodec.parse(text);
export const serializeHabitLog = (h) => HabitLogCodec.serialize(h);
export const parseHabitLog = (text) => HabitLogCodec.parse(text);
export const serializeJournal = (j) => JournalCodec.serialize(j);
export const parseJournal = (text) => JournalCodec.parse(text);
export const serializeCalendarEvent = (e) => CalendarEventCodec.serialize(e);
export const parseCalendarEvent = (text) => CalendarEventCodec.parse(text);

// Dispatch on balaur-type. Returns { type, ...fields, body }.
export function parseEntity(text) {
  const fm = splitFrontmatter(text);
  if (!fm) throw new ParseError("Missing or unterminated frontmatter", { code: "FM_NO_DELIMITER" });
  const probe = collectKnownFields(fm.lines.slice(fm.openIdx + 1, fm.closeIdx), { fields: { "balaur-type": "enum" } });
  const type = probe["balaur-type"];
  const codec = ENTITY_CODECS[type];
  if (!codec) throw new ParseError(`Unknown balaur-type: ${type}`, { code: "ENTITY_UNKNOWN_TYPE" });
  return { type, ...codec.parse(text) };
}

// --- habit-log check-in event markers (plan §9.4) ----------------------------
// Constrained token grammar; no arbitrary user text inside the marker.

const HABIT_ENTRY_START_RE = /<!--\s*balaur:habit-entry\b/gi;
const ENTRY_TOKEN_RE = /^[a-z0-9][a-z0-9.:+-]*$/i;

export function parseHabitEntries(body) {
  const source = String(body);
  const entries = [];
  // Inspect every marker start rather than merely checking whether one complete
  // marker exists. A valid marker followed by an unterminated marker invalidates
  // the complete source file.
  for (const start of source.matchAll(HABIT_ENTRY_START_RE)) {
    const end = source.indexOf("-->", start.index);
    if (end < 0) throw new ParseError("Unterminated habit-entry marker", { code: "HABIT_ENTRY_INVALID" });
    const marker = source.slice(start.index, end + 3);
    const m = /^<!--\s*balaur:habit-entry\s+([^>]*?)\s*-->$/i.exec(marker);
    if (!m) throw new ParseError("Malformed habit-entry marker", { code: "HABIT_ENTRY_INVALID" });
    const attrs = Object.create(null);
    const tokens = m[1].trim() ? m[1].trim().split(/\s+/) : [];
    for (const tok of tokens) {
      const eq = tok.indexOf("=");
      if (eq <= 0 || eq === tok.length - 1) throw new ParseError(`Malformed habit-entry attribute: ${tok}`, { code: "HABIT_ENTRY_INVALID" });
      const key = tok.slice(0, eq);
      if (!(key in { id: 1, habit: 1, status: 1, value: 1, at: 1 })) throw new ParseError(`Unknown habit-entry attribute: ${key}`, { code: "HABIT_ENTRY_INVALID" });
      if (Object.hasOwn(attrs, key)) throw new ParseError(`Duplicate habit-entry attribute: ${key}`, { code: "HABIT_ENTRY_INVALID" });
      attrs[key] = tok.slice(eq + 1);
    }
    for (const key of ["id", "habit", "status", "value", "at"]) if (!(key in attrs)) throw new ParseError(`Missing habit-entry attribute: ${key}`, { code: "HABIT_ENTRY_INVALID" });
    if (!ENTRY_TOKEN_RE.test(attrs.id) || !ENTRY_TOKEN_RE.test(attrs.habit) || !HABIT_ENTRY_STATUSES.has(attrs.status) || !isValidInstant(attrs.at)) {
      throw new ParseError("Invalid habit-entry marker", { code: "HABIT_ENTRY_INVALID" });
    }
    const value = Number(attrs.value);
    if (!Number.isFinite(value)) throw new ParseError(`Bad habit-entry value: ${attrs.value}`, { code: "HABIT_ENTRY_INVALID" });
    entries.push({ id: attrs.id, habit: attrs.habit, status: attrs.status, value, at: attrs.at });
  }
  return entries;
}

export function serializeHabitEntry(entry) {
  const token = (label, v) => {
    if (!ENTRY_TOKEN_RE.test(String(v))) throw new ParseError(`Bad habit-entry ${label}: ${v}`, { code: "FM_BAD_ENUM" });
    return v;
  };
  if (!HABIT_ENTRY_STATUSES.has(String(entry.status))) throw new ParseError(`Bad habit-entry status: ${entry.status}`, { code: "HABIT_ENTRY_INVALID" });
  if (!isValidInstant(entry.at)) throw new ParseError(`Bad habit-entry instant: ${entry.at}`, { code: "HABIT_ENTRY_INVALID" });
  const value = Number(entry.value);
  if (!Number.isFinite(value)) throw new ParseError(`Bad habit-entry value: ${entry.value}`, { code: "FM_BAD_NUMBER" });
  return `<!-- balaur:habit-entry id=${token("id", entry.id)} habit=${token("habit", entry.habit)} status=${token("status", entry.status)} value=${value} at=${token("at", entry.at)} -->`;
}
