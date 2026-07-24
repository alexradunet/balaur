// Phase 8 tests: file-canonical journal and calendar-event repositories.
// Run: node --test storage/phase8.test.js
import { test } from "node:test";
import assert from "node:assert/strict";

import { MemoryVault } from "./memory-vault.js";
import { MemoryIndex } from "./memory-index.js";
import { LifeIndexer } from "./life-indexer.js";
import { FileJournalRepository, FileEventRepository, journalPath } from "./journal-event-repository.js";
import { isCanvas } from "./canvas-validate.js";

const NOW = "2026-07-21T18:00:00.000Z";

const CANVAS_PATH = "canvases/root.canvas";

function setup() {
  const vault = new MemoryVault();
  const index = new MemoryIndex();
  const indexer = new LifeIndexer({ vault, index });
  const journals = new FileJournalRepository({ vault, index, indexer, canvasPathFromId: (id) => id === "canvas-root" ? CANVAS_PATH : null, now: () => NOW });
  const events = new FileEventRepository({ vault, index, indexer, now: () => NOW });
  return { vault, index, indexer, journals, events };
}

async function seedCanvas(vault) {
  await vault.write(CANVAS_PATH, JSON.stringify({ nodes: [], edges: [] }, null, 2) + "\n");
}

// --- journals ----------------------------------------------------------------

test("createJournal writes a dated journal file and indexes it", async () => {
  const { journals, vault, index } = setup();
  const { localDate, path } = await journals.createJournal({ localDate: "2026-07-21", body: "# Tuesday\n\nJournal text." });
  assert.equal(localDate, "2026-07-21");
  assert.equal(path, "journal/2026/2026-07-21.md");
  const stored = await vault.read(path);
  assert.match(stored, /orbit-type: journal/);
  assert.match(stored, /Journal text\./);
  assert.equal(index.allJournals().length, 1);
  assert.equal(index.allJournals()[0].localDate, "2026-07-21");
});

test("getJournal reads the entry back", async () => {
  const { journals } = setup();
  await journals.createJournal({ localDate: "2026-07-21", body: "# Tuesday\n\nJournal text." });
  const j = await journals.getJournal("2026-07-21");
  assert.equal(j.localDate, "2026-07-21");
  assert.equal(j.body, "# Tuesday\n\nJournal text.");
  assert.equal(j.orbitId, "journal-2026-07-21");
});

test("updateJournal replaces the body, preserving identity and bumping updated-at", async () => {
  const { journals, index } = setup();
  await journals.createJournal({ localDate: "2026-07-21", body: "Old." });
  const updated = await journals.updateJournal("2026-07-21", { body: "New body." });
  assert.equal(updated.body, "New body.");
  assert.equal(updated.localDate, "2026-07-21");
  assert.equal(updated.orbitId, "journal-2026-07-21");
  assert.equal(updated.updatedAt, NOW);
  assert.equal((await journals.getJournal("2026-07-21")).body, "New body.");
  assert.equal(index.allJournals().length, 1);
});

test("updateJournal rejects a missing journal", async () => {
  const { journals } = setup();
  await assert.rejects(() => journals.updateJournal("2026-01-01", { body: "x" }), /Journal not found/);
});

test("journalPath formats the dated path and rejects bad dates", () => {
  assert.equal(journalPath("2026-07-21"), "journal/2026/2026-07-21.md");
  assert.throws(() => journalPath("nope"), /Bad local date/);
});

test("addPlacement places a journal file node on a canvas", async () => {
  const { journals, vault } = setup();
  await seedCanvas(vault);
  await journals.createJournal({ localDate: "2026-07-21", body: "Hello." });
  const result = await journals.addPlacement("2026-07-21", "canvas-root", { id: "j-node" });
  assert.equal(result.canvasId, "canvas-root");
  assert.equal(result.nodeId, "j-node");
  assert.equal(result.path, "journal/2026/2026-07-21.md");
  const doc = JSON.parse(await vault.read(CANVAS_PATH));
  assert.ok(isCanvas(doc), "canvas must remain valid after placement");
  const node = doc.nodes.find((n) => n.id === "j-node");
  assert.ok(node, "file node must exist");
  assert.equal(node.type, "file");
  assert.equal(node.file, "journal/2026/2026-07-21.md");
});

test("addPlacement rejects a duplicate node id with CANVAS_ID_DUPLICATE", async () => {
  const { journals, vault } = setup();
  await seedCanvas(vault);
  await journals.createJournal({ localDate: "2026-07-21", body: "Hello." });
  await journals.addPlacement("2026-07-21", "canvas-root", { id: "dup" });
  await assert.rejects(
    () => journals.addPlacement("2026-07-21", "canvas-root", { id: "dup" }),
    (err) => err.code === "CANVAS_ID_DUPLICATE",
  );
});

test("addPlacement rejects a missing journal with JOURNAL_NOT_FOUND", async () => {
  const { journals, vault } = setup();
  await seedCanvas(vault);
  await assert.rejects(
    () => journals.addPlacement("2099-01-01", "canvas-root", {}),
    (err) => err.code === "JOURNAL_NOT_FOUND",
  );
});

// --- calendar events ---------------------------------------------------------

test("createEvent writes an event file and indexes it", async () => {
  const { events, vault, index } = setup();
  const { id, path } = await events.createEvent({
    id: "event-dentist", title: "Dentist appointment",
    startsAt: "2026-07-25T14:00:00.000Z", endsAt: "2026-07-25T15:00:00.000Z",
    localDate: "2026-07-25", timezone: "Europe/London", body: "Bring imaging.",
  });
  assert.equal(id, "event-dentist");
  assert.match(path, /^events\/dentist-appointment--dentist\.md$|^events\/dentist-appointment--.+\.md$/);
  assert.match(await vault.read(path), /orbit-type: calendar-event/);
  assert.equal(index.allEvents().length, 1);
  assert.equal(index.allEvents()[0].title, "Dentist appointment");
});

test("createEvent requires a title and a start", async () => {
  const { events } = setup();
  await assert.rejects(() => events.createEvent({ title: " ", startsAt: "2026-07-25T14:00:00.000Z" }), /title is required/);
  await assert.rejects(() => events.createEvent({ title: "X" }), /starts-at is required/);
});

test("getEvent reads the event back with timezone and body", async () => {
  const { events } = setup();
  await events.createEvent({ id: "event-dentist", title: "Dentist", startsAt: "2026-07-25T14:00:00.000Z", localDate: "2026-07-25", timezone: "Europe/London", body: "Bring imaging." });
  const ev = await events.getEvent("event-dentist");
  assert.equal(ev.title, "Dentist");
  assert.equal(ev.timezone, "Europe/London");
  assert.equal(ev.startsAt, "2026-07-25T14:00:00.000Z");
  assert.equal(ev.body, "Bring imaging.");
});

test("createEvent derives local-date from starts-at when omitted", async () => {
  const { events } = setup();
  await events.createEvent({ id: "event-x", title: "X", startsAt: "2026-08-01T10:00:00.000Z", timezone: "UTC" });
  assert.equal((await events.getEvent("event-x")).localDate, "2026-08-01");
});

test("createEvent supports all-day events", async () => {
  const { events } = setup();
  await events.createEvent({ id: "event-holiday", title: "Holiday", startsAt: "2026-08-01T00:00:00.000Z", localDate: "2026-08-01", timezone: "Europe/London", allDay: true });
  assert.equal((await events.getEvent("event-holiday")).allDay, true);
});

test("updateEvent patches fields preservation-first", async () => {
  const { events, index } = setup();
  await events.createEvent({ id: "event-dentist", title: "Dentist", startsAt: "2026-07-25T14:00:00.000Z", localDate: "2026-07-25", timezone: "Europe/London" });
  const updated = await events.updateEvent("event-dentist", { startsAt: "2026-07-25T16:00:00.000Z" });
  assert.equal(updated.startsAt, "2026-07-25T16:00:00.000Z");
  assert.equal(updated.title, "Dentist", "untouched field preserved");
  assert.equal(updated.updatedAt, NOW);
  assert.equal(index.allEvents()[0].startsAt, "2026-07-25T16:00:00.000Z");
});

test("updateEvent validates patched event values before writing", async () => {
  const { events, vault } = setup();
  const { path } = await events.createEvent({ id: "event-x", title: "X", startsAt: "2026-07-25T14:00:00.000Z", localDate: "2026-07-25", timezone: "UTC" });
  const before = await vault.read(path);
  await assert.rejects(() => events.updateEvent("event-x", { timezone: "Not/AZone" }), /Invalid IANA timezone/);
  assert.equal(await vault.read(path), before, "invalid timezone must not be written");
});

test("updateEvent rejects unknown fields", async () => {
  const { events } = setup();
  await events.createEvent({ id: "event-x", title: "X", startsAt: "2026-07-25T14:00:00.000Z", localDate: "2026-07-25", timezone: "UTC" });
  await assert.rejects(() => events.updateEvent("event-x", { bogus: 1 }), /Unknown event field/);
});

test("deleteEvent uses the source hash and leaves stale files on conflict", async () => {
  const { events, vault, index } = setup();
  const { id, path } = await events.createEvent({ id: "event-x", title: "X", startsAt: "2026-07-25T14:00:00.000Z", localDate: "2026-07-25", timezone: "UTC" });
  await vault.write(path, (await vault.read(path)).replace(/^title:.*$/m, "title: Changed"));
  await assert.rejects(() => events.deleteEvent(id), /Hash mismatch|WRITE_CONFLICT|conflict/i);
  assert.equal(await vault.exists(path), true);
  assert.equal(index.allEvents().length, 1);
});

test("deleteEvent removes the file and the index entry", async () => {
  const { events, vault, index } = setup();
  const { path } = await events.createEvent({ id: "event-x", title: "X", startsAt: "2026-07-25T14:00:00.000Z", localDate: "2026-07-25", timezone: "UTC" });
  await events.deleteEvent("event-x");
  assert.equal(await vault.exists(path), false);
  assert.equal(index.allEvents().length, 0);
});
