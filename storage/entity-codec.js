// Canonical entity codecs (Phase 1, plan §9).
//
// Each entity is a Markdown file: constrained frontmatter (orbit-schema 1) plus
// an ordinary Markdown body. Field keys are kebab-case in files and camelCase in
// application objects. Parsing validates required fields, enums, dates, and the
// orbit-schema version; unknown frontmatter is preserved by the codec layer.

import { splitFrontmatter, collectKnownFields, serializeFrontmatter } from "./frontmatter.js";
import { ParseError, SchemaError } from "./vault-errors.js";

export const SUPPORTED_SCHEMA = 1;

function kebabToCamel(k) { return k.replace(/-([a-z])/g, (_, c) => c.toUpperCase()); }

function validateSchema(schema) {
  if (schema === undefined || schema === null) throw new SchemaError("Missing orbit-schema", { code: "SCHEMA_MISSING" });
  if (typeof schema !== "number" || !Number.isFinite(schema)) throw new SchemaError("Invalid orbit-schema", { code: "SCHEMA_INVALID" });
  if (schema > SUPPORTED_SCHEMA) throw new SchemaError(`Unsupported orbit-schema ${schema} (read-only)`, { code: "SCHEMA_NEWER" });
  if (schema !== SUPPORTED_SCHEMA) throw new SchemaError(`Unsupported orbit-schema ${schema}`, { code: "SCHEMA_UNSUPPORTED" });
}

function makeCodec({ type, fields, order, required }) {
  const spec = { fields };

  function serialize(entity) {
    const fmFields = Object.create(null);
    for (const key of order) {
      const camel = kebabToCamel(key);
      let value = entity[camel];
      if (key === "orbit-schema") value = SUPPORTED_SCHEMA;
      else if (key === "orbit-type") value = type;
      else if (value === undefined) {
        if (required.includes(key)) {
          throw new ParseError(`Missing required field: ${key}`, { code: "ENTITY_MISSING_FIELD", details: { key } });
        }
        value = null;
      }
      fmFields[key] = value;
    }
    const fm = serializeFrontmatter(fmFields, spec, order);
    const body = entity.body ?? "";
    return fm + (body ? "\n" + body : "");
  }

  function parse(text) {
    const fm = splitFrontmatter(text);
    if (!fm) throw new ParseError("Missing or unterminated frontmatter", { code: "FM_NO_DELIMITER" });
    const raw = collectKnownFields(fm.lines.slice(fm.openIdx + 1, fm.closeIdx), spec);
    validateSchema(raw["orbit-schema"]);
    if (raw["orbit-type"] !== type) {
      throw new ParseError(`Expected orbit-type "${type}", got "${raw["orbit-type"]}"`, { code: "ENTITY_TYPE_MISMATCH" });
    }
    for (const key of required) {
      if (key === "orbit-schema" || key === "orbit-type") continue;
      if (raw[key] === undefined || raw[key] === null) {
        throw new ParseError(`Missing required field: ${key}`, { code: "ENTITY_MISSING_FIELD", details: { key } });
      }
    }
    const out = Object.create(null);
    for (const key of order) {
      if (key === "orbit-schema" || key === "orbit-type") continue;
      out[kebabToCamel(key)] = raw[key] === undefined ? null : raw[key];
    }
    let body = fm.lines.slice(fm.closeIdx + 1).join("");
    if (body.startsWith("\n")) body = body.slice(1); // drop the single blank separator line
    out.body = body;
    return out;
  }

  return { type, spec, order, required, serialize, parse };
}

// --- field specifications ----------------------------------------------------

export const TaskCodec = makeCodec({
  type: "task",
  fields: {
    "orbit-schema": "number", "orbit-type": "enum", "orbit-id": "string", "title": "string",
    "status": "enum", "priority": "number", "scheduled-on": "date", "due-on": "date",
    "completed-at": "instant", "estimate-minutes": "number", "recurrence": "string",
    "created-at": "instant", "updated-at": "instant",
  },
  order: ["orbit-schema", "orbit-type", "orbit-id", "title", "status", "priority",
    "scheduled-on", "due-on", "completed-at", "estimate-minutes", "recurrence",
    "created-at", "updated-at"],
  required: ["orbit-id", "title", "status", "created-at", "updated-at"],
});

export const HabitCodec = makeCodec({
  type: "habit",
  fields: {
    "orbit-schema": "number", "orbit-type": "enum", "orbit-id": "string", "title": "string",
    "frequency": "enum", "weekdays": "number[]", "target": "number", "unit": "string",
    "archived-at": "instant", "created-at": "instant", "updated-at": "instant",
  },
  order: ["orbit-schema", "orbit-type", "orbit-id", "title", "frequency", "weekdays",
    "target", "unit", "archived-at", "created-at", "updated-at"],
  required: ["orbit-id", "title", "frequency", "created-at", "updated-at"],
});

export const HabitLogCodec = makeCodec({
  type: "habit-log",
  fields: { "orbit-schema": "number", "orbit-type": "enum", "local-date": "date" },
  order: ["orbit-schema", "orbit-type", "local-date"],
  required: ["local-date"],
});

export const JournalCodec = makeCodec({
  type: "journal",
  fields: {
    "orbit-schema": "number", "orbit-type": "enum", "orbit-id": "string", "local-date": "date",
    "created-at": "instant", "updated-at": "instant",
  },
  order: ["orbit-schema", "orbit-type", "orbit-id", "local-date", "created-at", "updated-at"],
  required: ["orbit-id", "local-date", "created-at", "updated-at"],
});

export const CalendarEventCodec = makeCodec({
  type: "calendar-event",
  fields: {
    "orbit-schema": "number", "orbit-type": "enum", "orbit-id": "string", "title": "string",
    "starts-at": "instant", "ends-at": "instant", "local-date": "date", "timezone": "string",
    "all-day": "boolean", "source": "enum", "created-at": "instant", "updated-at": "instant",
  },
  order: ["orbit-schema", "orbit-type", "orbit-id", "title", "starts-at", "ends-at",
    "local-date", "timezone", "all-day", "source", "created-at", "updated-at"],
  required: ["orbit-id", "title", "starts-at", "local-date", "timezone", "created-at", "updated-at"],
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

// Dispatch on orbit-type. Returns { type, ...fields, body }.
export function parseEntity(text) {
  const fm = splitFrontmatter(text);
  if (!fm) throw new ParseError("Missing or unterminated frontmatter", { code: "FM_NO_DELIMITER" });
  const probe = collectKnownFields(fm.lines.slice(fm.openIdx + 1, fm.closeIdx), { fields: { "orbit-type": "enum" } });
  const type = probe["orbit-type"];
  const codec = ENTITY_CODECS[type];
  if (!codec) throw new ParseError(`Unknown orbit-type: ${type}`, { code: "ENTITY_UNKNOWN_TYPE" });
  return { type, ...codec.parse(text) };
}

// --- habit-log check-in event markers (plan §9.4) ----------------------------
// Constrained token grammar; no arbitrary user text inside the marker.

const HABIT_ENTRY_RE = /<!--\s*orbit:habit-entry\s+([^>]*?)\s*-->/g;
const ENTRY_TOKEN_RE = /^[a-z0-9][a-z0-9.:+-]*$/i;

export function parseHabitEntries(body) {
  const entries = [];
  for (const m of String(body).matchAll(HABIT_ENTRY_RE)) {
    const attrs = Object.create(null);
    for (const tok of m[1].trim().split(/\s+/)) {
      const eq = tok.indexOf("=");
      if (eq > 0) attrs[tok.slice(0, eq)] = tok.slice(eq + 1);
    }
    entries.push({
      id: attrs.id ?? null,
      habit: attrs.habit ?? null,
      status: attrs.status ?? null,
      value: attrs.value === undefined ? null : Number(attrs.value),
      at: attrs.at ?? null,
    });
  }
  return entries;
}

export function serializeHabitEntry(entry) {
  const token = (label, v) => {
    if (!ENTRY_TOKEN_RE.test(String(v))) throw new ParseError(`Bad habit-entry ${label}: ${v}`, { code: "FM_BAD_ENUM" });
    return v;
  };
  const value = Number(entry.value);
  if (!Number.isFinite(value)) throw new ParseError(`Bad habit-entry value: ${entry.value}`, { code: "FM_BAD_NUMBER" });
  return `<!-- orbit:habit-entry id=${token("id", entry.id)} habit=${token("habit", entry.habit)} status=${token("status", entry.status)} value=${value} at=${token("at", entry.at)} -->`;
}
