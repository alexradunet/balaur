// Phase 1 test suite — run with: node --test storage/phase1.test.js
// Uses only the built-in Node test runner and assert (no package install).

import { test } from "node:test";
import assert from "node:assert/strict";

import { contentHash, shortId } from "./content-hash.js";
import {
  normalizePath, assertSafePath, slugify, entityPath, byteLength,
  caseFoldKey, samePathFold, MAX_COMPONENT_BYTES,
} from "./vault-path.js";
import {
  splitFrontmatter, hasFrontmatter, readFields, patchFields,
  parseValue, serializeValue, serializeFrontmatter,
} from "./frontmatter.js";
import {
  TaskCodec, HabitCodec, serializeTask, parseTask, serializeHabit, parseHabit,
  serializeJournal, parseJournal, serializeCalendarEvent, parseCalendarEvent,
  serializeHabitLog, parseHabitLog, parseEntity, parseHabitEntries, serializeHabitEntry,
} from "./entity-codec.js";
import { PathError, ParseError, SchemaError } from "./vault-errors.js";

const plain = (o) => ({ ...o });
const INST = "2026-07-21T18:00:00.000Z";

const sampleTask = {
  orbitId: "task-a1b2c3", title: "Finish quarterly review", status: "next",
  priority: 1, scheduledOn: "2026-07-22", dueOn: "2026-07-25",
  completedAt: null, estimateMinutes: 45, recurrence: null,
  createdAt: INST, updatedAt: INST,
  body: "Collect the outstanding numbers and prepare the summary.",
};

// --- content hashing ---------------------------------------------------------

test("contentHash is deterministic, content-sensitive, and hex-shaped", async () => {
  const a = await contentHash("hello");
  const b = await contentHash("hello");
  const c = await contentHash("hello!");
  assert.equal(a, b);
  assert.notEqual(a, c);
  assert.match(a, /^[0-9a-f]{64}$/);
});

test("shortId derives a stable 6-char suffix", () => {
  assert.equal(shortId("task-a1b2c3"), "a1b2c3");
  assert.equal(shortId("x"), "x");
  assert.equal(shortId("---"), "000000"); // no alphanumerics -> fallback
});

// --- path normalization ------------------------------------------------------

test("normalizePath converts separators and preserves legal spaces", () => {
  assert.equal(normalizePath("a\\b"), "a/b");
  assert.equal(normalizePath("tasks/My Task.md"), "tasks/My Task.md");
});

test("normalizePath rejects traversal, absolute paths, and schemes", () => {
  assert.throws(() => normalizePath("../etc"), PathError);
  assert.throws(() => normalizePath("a/../../b"), PathError);
  assert.throws(() => normalizePath("a/./b"), PathError);
  assert.throws(() => normalizePath("/abs"), PathError);
  assert.throws(() => normalizePath("file:x"), PathError);
  assert.throws(() => normalizePath("https://x/y"), PathError);
});

test("normalizePath rewrites Windows device names; assertSafePath rejects them", () => {
  assert.equal(normalizePath("CON"), "_CON");
  assert.equal(normalizePath("dir/NUL.txt"), "dir/_NUL.txt");
  assert.throws(() => assertSafePath("CON"), PathError);
  assert.throws(() => assertSafePath("dir/com1"), PathError);
});

test("assertSafePath accepts clean paths and rejects unsafe ones", () => {
  assert.equal(assertSafePath("tasks/finish--a1b2c3.md"), "tasks/finish--a1b2c3.md");
  assert.throws(() => assertSafePath("tasks/../x"), PathError);
  assert.throws(() => assertSafePath("bad:name.md"), PathError);
  assert.throws(() => assertSafePath("trail."), PathError);
  assert.throws(() => assertSafePath("trail "), PathError);
  assert.throws(() => assertSafePath("a\u0000b.md"), PathError);
});

test("byteLength counts UTF-8 bytes, not JS string length", () => {
  assert.equal(byteLength("café"), 5); // é is 2 bytes
  assert.equal(byteLength("abc"), 3);
});

test("component byte length is bounded", () => {
  const long = "a".repeat(MAX_COMPONENT_BYTES + 1);
  assert.throws(() => normalizePath(`tasks/${long}.md`), PathError);
});

test("case-fold and Unicode-normalization collisions are detected", () => {
  assert.ok(samePathFold("Tasks/A.md", "tasks/a.md"));
  // NFD (e + combining acute) vs NFC (é) fold to the same key.
  assert.equal(caseFoldKey("cafe\u0301.md"), caseFoldKey("caf\u00e9.md"));
});

test("slugify and entityPath produce the documented layout", () => {
  assert.equal(slugify("Finish Quarterly Review!"), "finish-quarterly-review");
  assert.equal(slugify("  --weird--  "), "weird");
  assert.equal(slugify(""), "untitled");
  assert.equal(entityPath("tasks", "Finish quarterly review", "task-a1b2c3"),
    "tasks/finish-quarterly-review--a1b2c3.md");
});

// --- constrained value grammar ----------------------------------------------

test("parseValue/serializeValue round-trip primitives", () => {
  assert.equal(parseValue('"a: b # c"', "string"), "a: b # c");
  assert.equal(parseValue("42", "number"), 42);
  assert.equal(parseValue("-3.5", "number"), -3.5);
  assert.equal(parseValue("true", "boolean"), true);
  assert.equal(parseValue("null", "string"), null);
  assert.equal(parseValue("next", "enum"), "next");
  assert.equal(parseValue('"2026-07-22"', "date"), "2026-07-22");
  assert.deepEqual(parseValue("[1, 2, 3]", "number[]"), [1, 2, 3]);
  assert.deepEqual(parseValue('["a", "b"]', "string[]"), ["a", "b"]);
  assert.equal(serializeValue("a: b", "string"), '"a: b"');
  assert.equal(serializeValue(null, "number"), "null");
  assert.equal(serializeValue([1, 2], "number[]"), "[1, 2]");
});

test("parseValue rejects malformed scalars", () => {
  assert.throws(() => parseValue("not-quoted", "string"), ParseError);
  assert.throws(() => parseValue("NaN", "number"), ParseError);
  assert.throws(() => parseValue("Infinity", "number"), ParseError);
  assert.throws(() => parseValue("2026-13-99", "date"), ParseError);
  assert.throws(() => parseValue("Next", "enum"), ParseError); // uppercase not allowed
  assert.throws(() => parseValue("maybe", "boolean"), ParseError);
});

// --- frontmatter splitting & preservation ------------------------------------

test("splitFrontmatter finds the block and reports delimiters", () => {
  const fm = splitFrontmatter("---\na: 1\n---\nbody\n");
  assert.ok(fm);
  assert.equal(fm.openIdx, 0);
  assert.equal(fm.closeIdx, 2);
  assert.ok(hasFrontmatter("---\n---\n"));
  assert.equal(hasFrontmatter("no frontmatter"), false);
  assert.equal(hasFrontmatter("---\nunterminated\n"), false);
});

test("patching one field changes no unrelated byte range", () => {
  const original = [
    "---", "orbit-schema: 1", "orbit-type: task", 'orbit-id: "task-abc123"',
    'title: "Write: a report #1"', "status: next", "priority: 2",
    "custom-unknown: keep me", "  nested: also kept", "---", "",
    "Body line one", "Body line two", "",
  ].join("\n");
  const patched = patchFields(original, { status: "done" }, TaskCodec.spec);
  assert.notEqual(patched, original);
  assert.ok(patched.includes("status: done"));
  // Every line identical except the status line.
  const a = original.split("\n"), b = patched.split("\n");
  assert.equal(a.length, b.length);
  for (let i = 0; i < a.length; i++) {
    if (a[i].startsWith("status:")) assert.equal(b[i], "status: done");
    else assert.equal(b[i], a[i], `line ${i} changed unexpectedly`);
  }
});

test("patching preserves BOM and CRLF line endings", () => {
  const bomDoc = "\uFEFF---\r\norbit-schema: 1\r\norbit-type: task\r\norbit-id: \"t1\"\r\ntitle: \"x\"\r\nstatus: next\r\ncreated-at: \"2026-01-01T00:00:00.000Z\"\r\nupdated-at: \"2026-01-01T00:00:00.000Z\"\r\n---\r\n\r\nBody\r\n";
  const patched = patchFields(bomDoc, { status: "done" }, TaskCodec.spec);
  assert.ok(patched.startsWith("\uFEFF"), "BOM retained");
  assert.ok(patched.includes("status: done\r\n"), "patched line keeps CRLF");
  assert.ok(patched.includes('title: "x"\r\n'), "untouched line keeps CRLF");
});

test("patching inserts a missing known key before the closing delimiter", () => {
  const doc = "---\norbit-schema: 1\norbit-type: task\norbit-id: \"t1\"\ntitle: \"x\"\nstatus: next\ncreated-at: \"2026-01-01T00:00:00.000Z\"\nupdated-at: \"2026-01-01T00:00:00.000Z\"\n---\n\nBody\n";
  const patched = patchFields(doc, { priority: 3 }, TaskCodec.spec);
  assert.ok(patched.includes("priority: 3\n"));
  assert.ok(patched.indexOf("priority: 3") < patched.lastIndexOf("---"));
  assert.ok(patched.includes("Body"));
});

test("duplicate known keys are rejected on read and on patch", () => {
  const dup = "---\norbit-schema: 1\norbit-type: task\nstatus: next\nstatus: done\n---\n";
  assert.throws(() => readFields(dup, TaskCodec.spec), ParseError);
  assert.throws(() => patchFields(dup, { status: "done" }, TaskCodec.spec), ParseError);
});

test("patching refuses unknown fields and missing frontmatter", () => {
  const doc = serializeTask(sampleTask);
  assert.throws(() => patchFields(doc, { bogus: "x" }, TaskCodec.spec), ParseError);
  const noFm = "just text, no frontmatter";
  assert.throws(() => patchFields(noFm, { status: "done" }, TaskCodec.spec), ParseError);
  assert.equal(noFm, "just text, no frontmatter"); // original untouched
});

test("external edits: reordered keys, requoted strings, and block arrays still parse", () => {
  const reordered = "---\norbit-schema: 1\norbit-type: task\nstatus: next\norbit-id: \"t1\"\ntitle: \"x\"\ncreated-at: \"2026-01-01T00:00:00.000Z\"\nupdated-at: \"2026-01-01T00:00:00.000Z\"\n---\n";
  assert.equal(parseTask(reordered).status, "next");

  const requoted = serializeTask({ ...sampleTask, title: 'She said "hi" — café' });
  assert.equal(parseTask(requoted).title, 'She said "hi" — café');

  const blockArr = "---\norbit-schema: 1\norbit-type: habit\norbit-id: \"h1\"\ntitle: \"Walk\"\nfrequency: weekly\nweekdays:\n  - 1\n  - 2\n  - 3\ntarget: 1\nunit: \"walk\"\ncreated-at: \"2026-01-01T00:00:00.000Z\"\nupdated-at: \"2026-01-01T00:00:00.000Z\"\n---\n";
  assert.deepEqual(parseHabit(blockArr).weekdays, [1, 2, 3]);
});

// --- entity codecs -----------------------------------------------------------

test("task round-trips through serialize/parse", () => {
  const parsed = parseTask(serializeTask(sampleTask));
  assert.deepEqual(plain(parsed), plain(sampleTask));
});

test("task body preserves newlines and unicode", () => {
  const t = { ...sampleTask, body: "line1\nline2\n\nline4 — ünïcode" };
  assert.equal(parseTask(serializeTask(t)).body, t.body);
});

test("habit, journal, and calendar-event round-trip", () => {
  const habit = {
    orbitId: "habit-morning-walk", title: "Morning walk", frequency: "weekly",
    weekdays: [1, 2, 3, 4, 5], target: 1, unit: "walk", archivedAt: null,
    createdAt: INST, updatedAt: INST, body: "Walk before opening work applications.",
  };
  assert.deepEqual(plain(parseHabit(serializeHabit(habit))), plain(habit));

  const journal = {
    orbitId: "journal-2026-07-21", localDate: "2026-07-21",
    createdAt: INST, updatedAt: INST, body: "# Tuesday, July 21\n\nJournal text.",
  };
  assert.deepEqual(plain(parseJournal(serializeJournal(journal))), plain(journal));

  const event = {
    orbitId: "event-dentist-m4n5o6", title: "Dentist appointment",
    startsAt: "2026-07-24T09:00:00+03:00", endsAt: "2026-07-24T10:00:00+03:00",
    localDate: "2026-07-24", timezone: "Europe/Bucharest", allDay: false, source: "orbit",
    createdAt: INST, updatedAt: INST, body: "Bring the previous imaging report.",
  };
  assert.deepEqual(plain(parseCalendarEvent(serializeCalendarEvent(event))), plain(event));
});

test("habit-log carries local-date and parses check-in event markers", () => {
  const entry = { id: "habit-entry-r4s5t6", habit: "habit-morning-walk", status: "done", value: 1, at: INST };
  const marker = serializeHabitEntry(entry);
  const log = { localDate: "2026-07-21", body: `# Habit check-ins\n\n- [x] Morning walk\n  ${marker}\n` };
  const text = serializeHabitLog(log);
  const parsed = parseHabitLog(text);
  assert.equal(parsed.localDate, "2026-07-21");
  const entries = parseHabitEntries(parsed.body);
  assert.equal(entries.length, 1);
  assert.deepEqual(plain(entries[0]), plain(entry));
});

test("parseEntity dispatches on orbit-type", () => {
  const entity = parseEntity(serializeTask(sampleTask));
  assert.equal(entity.type, "task");
  assert.equal(entity.title, sampleTask.title);
  assert.throws(() => parseEntity("---\norbit-schema: 1\norbit-type: bogus\n---\n"), ParseError);
});

test("newer orbit-schema is read-only diagnostic; missing schema is an error", () => {
  const newer = "---\norbit-schema: 2\norbit-type: task\norbit-id: \"t1\"\ntitle: \"x\"\nstatus: next\ncreated-at: \"2026-01-01T00:00:00.000Z\"\nupdated-at: \"2026-01-01T00:00:00.000Z\"\n---\n";
  assert.throws(() => parseTask(newer), (e) => e instanceof SchemaError && e.code === "SCHEMA_NEWER");
  const missing = "---\norbit-type: task\norbit-id: \"t1\"\ntitle: \"x\"\nstatus: next\ncreated-at: \"2026-01-01T00:00:00.000Z\"\nupdated-at: \"2026-01-01T00:00:00.000Z\"\n---\n";
  assert.throws(() => parseTask(missing), (e) => e instanceof SchemaError && e.code === "SCHEMA_MISSING");
});

test("serializeFrontmatter emits keys in stable order", () => {
  const fm = serializeFrontmatter(
    { "orbit-schema": 1, "orbit-type": "task", "orbit-id": "t1" },
    { fields: { "orbit-schema": "number", "orbit-type": "enum", "orbit-id": "string" } },
    ["orbit-schema", "orbit-type", "orbit-id"],
  );
  assert.equal(fm, "---\norbit-schema: 1\norbit-type: task\norbit-id: \"t1\"\n---\n");
});
