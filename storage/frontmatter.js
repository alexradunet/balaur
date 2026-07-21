// Preservation-first frontmatter codec (Phase 1, plan §8).
//
// Orbit generates YAML-compatible frontmatter but parses only known, flat,
// Orbit-owned keys. Unknown keys, comments, ordering, indentation, line endings,
// BOM, and the Markdown body are preserved byte-for-byte. Patching replaces only
// the exact known property lines being changed (or inserts one missing line
// before the closing delimiter). When a block cannot be patched safely, the
// codec throws rather than overwriting the file.

import { ParseError } from "./vault-errors.js";

const BOM = "\uFEFF";
const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;
const INSTANT_RE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/;
const ENUM_RE = /^[a-z][a-z0-9_-]*$/;
const NUMBER_RE = /^-?(0|[1-9]\d*)(\.\d+)?([eE][+-]?\d+)?$/;
const KEY_RE = /^([A-Za-z0-9_-]+)[ \t]*:(.*)$/;

// --- line handling (terminators preserved) -----------------------------------

function splitLinesKeepEnd(src) {
  const lines = [];
  let start = 0;
  for (let i = 0; i < src.length; i++) {
    if (src[i] === "\n") { lines.push(src.slice(start, i + 1)); start = i + 1; }
  }
  if (start < src.length) lines.push(src.slice(start));
  return lines;
}

function lineContent(line) { return line.replace(/\r?\n$/, ""); }
function lineTerminator(line) { const m = /\r?\n$/.exec(line); return m ? m[0] : ""; }

// Split a document into its frontmatter structure. Returns null when there is no
// valid frontmatter block (no opening `---` on the first line, or no closing).
export function splitFrontmatter(text) {
  let bom = "";
  let src = String(text ?? "");
  if (src.startsWith(BOM)) { bom = BOM; src = src.slice(1); }
  const lines = splitLinesKeepEnd(src);
  if (lines.length === 0) return null;
  if (!/^---[ \t]*$/.test(lineContent(lines[0]))) return null;
  let closeIdx = -1;
  for (let i = 1; i < lines.length; i++) {
    if (/^---[ \t]*$/.test(lineContent(lines[i]))) { closeIdx = i; break; }
  }
  if (closeIdx === -1) return null;
  const term = lineTerminator(lines[0]) || "\n";
  return { bom, lines, openIdx: 0, closeIdx, term };
}

export function hasFrontmatter(text) { return splitFrontmatter(text) !== null; }

// --- constrained value grammar (plan §8.2) -----------------------------------

function parseScalar(raw, kind) {
  const value = String(raw).trim();
  switch (kind) {
    case "string": {
      if (!(value.startsWith('"') && value.endsWith('"'))) {
        throw new ParseError(`Expected a double-quoted string, got: ${raw}`, { code: "FM_BAD_STRING" });
      }
      try { return JSON.parse(value); }
      catch { throw new ParseError(`Invalid quoted string: ${raw}`, { code: "FM_BAD_STRING" }); }
    }
    case "date": {
      let s;
      try { s = JSON.parse(value); } catch { s = null; }
      if (typeof s !== "string" || !DATE_RE.test(s)) {
        throw new ParseError(`Expected a YYYY-MM-DD date, got: ${raw}`, { code: "FM_BAD_DATE" });
      }
      return s;
    }
    case "instant": {
      let s;
      try { s = JSON.parse(value); } catch { s = null; }
      if (typeof s !== "string" || !INSTANT_RE.test(s)) {
        throw new ParseError(`Expected an ISO 8601 instant, got: ${raw}`, { code: "FM_BAD_INSTANT" });
      }
      return s;
    }
    case "number": {
      if (!NUMBER_RE.test(value) || !Number.isFinite(Number(value))) {
        throw new ParseError(`Expected a finite number, got: ${raw}`, { code: "FM_BAD_NUMBER" });
      }
      return Number(value);
    }
    case "boolean": {
      if (value === "true") return true;
      if (value === "false") return false;
      throw new ParseError(`Expected true or false, got: ${raw}`, { code: "FM_BAD_BOOLEAN" });
    }
    case "enum": {
      if (!ENUM_RE.test(value)) {
        throw new ParseError(`Expected a lowercase enum token, got: ${raw}`, { code: "FM_BAD_ENUM" });
      }
      return value;
    }
    default:
      throw new ParseError(`Unknown field kind: ${kind}`, { code: "FM_BAD_KIND" });
  }
}

function splitFlow(inner) {
  const items = [];
  let inStr = false, esc = false, cur = "";
  for (const ch of inner) {
    if (inStr) {
      cur += ch;
      if (esc) esc = false;
      else if (ch === "\\") esc = true;
      else if (ch === '"') inStr = false;
      continue;
    }
    if (ch === '"') { inStr = true; cur += ch; continue; }
    if (ch === "," ) { items.push(cur.trim()); cur = ""; continue; }
    cur += ch;
  }
  if (cur.trim() !== "") items.push(cur.trim());
  return items;
}

function elementKind(kind) { return kind === "number[]" ? "number" : "string"; }

function parseFlowArray(raw, kind) {
  const value = String(raw).trim();
  if (!(value.startsWith("[") && value.endsWith("]"))) {
    throw new ParseError(`Expected a flow array, got: ${raw}`, { code: "FM_BAD_ARRAY" });
  }
  const inner = value.slice(1, -1).trim();
  if (inner === "") return [];
  return splitFlow(inner).map((item) => parseScalar(item, elementKind(kind)));
}

export function parseValue(raw, kind) {
  const value = String(raw).trim();
  if (value === "null") return null;
  if (kind === "number[]" || kind === "string[]") return parseFlowArray(value, kind);
  return parseScalar(value, kind);
}

export function serializeValue(value, kind) {
  if (value === null || value === undefined) return "null";
  switch (kind) {
    case "string":
    case "date":
    case "instant":
      return JSON.stringify(String(value));
    case "number":
      if (!Number.isFinite(value)) throw new ParseError(`Cannot serialize non-finite number`, { code: "FM_BAD_NUMBER" });
      return String(value);
    case "boolean":
      return value ? "true" : "false";
    case "enum":
      if (!ENUM_RE.test(value)) throw new ParseError(`Cannot serialize enum token: ${value}`, { code: "FM_BAD_ENUM" });
      return value;
    case "number[]":
      return `[${value.map((n) => serializeValue(n, "number")).join(", ")}]`;
    case "string[]":
      return `[${value.map((s) => serializeValue(s, "string")).join(", ")}]`;
    default:
      throw new ParseError(`Unknown field kind: ${kind}`, { code: "FM_BAD_KIND" });
  }
}

// --- reading known fields ----------------------------------------------------

// Locate the line range (inclusive) for a top-level key, including any indented
// block-sequence continuation lines. Returns null when absent.
function findKeyRange(lines, key) {
  const re = new RegExp(`^${key.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}[ \\t]*:(.*)$`);
  for (let i = 0; i < lines.length; i++) {
    const m = re.exec(lineContent(lines[i]));
    if (!m) continue;
    const rest = m[1].trim();
    let end = i;
    if (rest === "") {
      let j = i + 1;
      while (j < lines.length) {
        const c = lineContent(lines[j]);
        if (c.trim() !== "" && /^[ \t]/.test(c)) j++;
        else break;
      }
      end = j - 1;
      if (j === i + 1) end = i;
    }
    return { start: i, end, rest };
  }
  return null;
}

// Collect known fields from frontmatter lines. Rejects duplicate known keys.
// Unknown keys and nested structures are ignored (preserved, not evaluated).
export function collectKnownFields(fmLines, spec) {
  const fields = Object.create(null);
  const seen = new Set();
  let i = 0;
  while (i < fmLines.length) {
    const m = KEY_RE.exec(lineContent(fmLines[i]));
    if (!m) { i++; continue; }
    const key = m[1];
    if (!(key in spec.fields)) { i++; continue; }
    if (seen.has(key)) {
      throw new ParseError(`Duplicate known frontmatter key: ${key}`, { code: "FM_DUPLICATE_KEY", details: { key } });
    }
    seen.add(key);
    const kind = spec.fields[key];
    const rest = m[2].trim();
    const isArray = kind === "number[]" || kind === "string[]";
    if (isArray && rest === "") {
      // Block sequence form (external-edit tolerance, plan §8.3 / review S4).
      const items = [];
      let j = i + 1;
      while (j < fmLines.length) {
        const c = lineContent(fmLines[j]);
        const item = /^[ \t]+-[ \t]+(.*)$/.exec(c) || /^[ \t]+-$/.exec(c);
        if (!item) break;
        if (item[1] !== undefined) items.push(item[1].trim());
        j++;
      }
      fields[key] = items.map((it) => (it.startsWith('"') ? parseScalar(it, "string") : parseScalar(it, elementKind(kind))));
      i = j;
    } else {
      fields[key] = parseValue(rest, kind);
      i++;
    }
  }
  return fields;
}

// Read and parse the known Orbit fields from a full document.
export function readFields(text, spec) {
  const fm = splitFrontmatter(text);
  if (!fm) throw new ParseError("Missing or unterminated frontmatter", { code: "FM_NO_DELIMITER" });
  return collectKnownFields(fm.lines.slice(fm.openIdx + 1, fm.closeIdx), spec);
}

// --- patching (preservation-first) -------------------------------------------

// Patch one or more known fields, replacing only their lines. Unknown content,
// comments, ordering, indentation, line endings, BOM, and body are preserved.
// Throws (refusing to write) when the block is absent or a key is duplicated.
export function patchFields(text, patch, spec) {
  const fm = splitFrontmatter(text);
  if (!fm) throw new ParseError("Missing or unterminated frontmatter; refusing to write", { code: "FM_NO_DELIMITER" });
  const fmLines = fm.lines.slice(fm.openIdx + 1, fm.closeIdx);
  collectKnownFields(fmLines, spec); // throws on duplicate known keys

  const next = fmLines.slice();
  for (const [key, value] of Object.entries(patch)) {
    if (!(key in spec.fields)) {
      throw new ParseError(`Refusing to write unknown orbit field: ${key}`, { code: "FM_UNKNOWN_FIELD" });
    }
    const kind = spec.fields[key];
    const line = `${key}: ${serializeValue(value, kind)}`;
    const range = findKeyRange(next, key);
    if (range) {
      const term = lineTerminator(next[range.start]) || fm.term;
      next.splice(range.start, range.end - range.start + 1, line + term);
    } else {
      next.push(line + fm.term); // insert before closing delimiter
    }
  }

  const out = fm.lines.slice(0, fm.openIdx + 1).concat(next, fm.lines.slice(fm.closeIdx));
  return fm.bom + out.join("");
}

// Serialize a fresh frontmatter block in a stable key order (for new files).
export function serializeFrontmatter(fields, spec, order, term = "\n") {
  const lines = [`---${term}`];
  for (const key of order) {
    if (!(key in spec.fields)) throw new ParseError(`Unknown orbit field in order: ${key}`, { code: "FM_UNKNOWN_FIELD" });
    lines.push(`${key}: ${serializeValue(fields[key], spec.fields[key])}${term}`);
  }
  lines.push(`---${term}`);
  return lines.join("");
}
