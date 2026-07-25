// Phase 7 tests: file-canonical habit repository (plan §9.4/§13).
// Run: node --test storage/phase7.test.js
import { test } from "node:test";
import assert from "node:assert/strict";

import { MemoryVault } from "./memory-vault.js";
import { MemoryIndex } from "./memory-index.js";
import { LifeIndexer } from "./life-indexer.js";
import { FileHabitRepository, habitLogPath } from "./habit-repository.js";

const NOW = "2026-07-21T18:00:00.000Z";

function setup() {
  const vault = new MemoryVault();
  const index = new MemoryIndex();
  const indexer = new LifeIndexer({ vault, index });
  const repo = new FileHabitRepository({ vault, index, indexer, now: () => NOW });
  return { vault, index, indexer, repo };
}

test("createHabit writes a canonical habit file and indexes it", async () => {
  const { repo, vault, index } = setup();
  const { id, path } = await repo.createHabit({ id: "habit-morning-walk", title: "Morning walk", frequency: "weekly", weekdays: [1, 2, 3, 4, 5], target: 3, unit: "times" });
  assert.equal(id, "habit-morning-walk");
  assert.match(path, /^habits\/morning-walk--morningwalk\.md$|^habits\/morning-walk--.+\.md$/);
  const stored = await vault.read(path);
  assert.match(stored, /balaur-type: habit/);
  assert.match(stored, /balaur-id: "?habit-morning-walk"?/);
  assert.equal(index.allHabits().length, 1);
  assert.equal(index.allHabits()[0].title, "Morning walk");
});

test("createHabit requires a title", async () => {
  const { repo } = setup();
  await assert.rejects(() => repo.createHabit({ title: "  " }), /title is required/);
});

test("getHabit reads the definition back", async () => {
  const { repo } = setup();
  await repo.createHabit({ id: "habit-walk", title: "Walk", frequency: "weekly", weekdays: [1, 3, 5] });
  const habit = await repo.getHabit("habit-walk");
  assert.equal(habit.frequency, "weekly");
  assert.deepEqual(habit.weekdays, [1, 3, 5]);
});

test("updateHabit patches definition fields preservation-first", async () => {
  const { repo, index } = setup();
  await repo.createHabit({ id: "habit-walk", title: "Walk", frequency: "weekly", target: 3 });
  const updated = await repo.updateHabit("habit-walk", { frequency: "daily", target: 1 });
  assert.equal(updated.frequency, "daily");
  assert.equal(updated.target, 1);
  assert.equal(updated.title, "Walk", "untouched field preserved");
  assert.equal(updated.updatedAt, NOW);
  assert.equal(index.allHabits()[0].frequency, "daily");
});

test("updateHabit validates domain patches before writing", async () => {
  const { repo, vault } = setup();
  const { path } = await repo.createHabit({ id: "habit-walk", title: "Walk" });
  const before = await vault.read(path);
  await assert.rejects(() => repo.updateHabit("habit-walk", { frequency: "yearly" }), /Invalid habit frequency/);
  assert.equal(await vault.read(path), before, "invalid habit frequency must not be written");
});

test("updateHabit body replacement preserves unknown frontmatter and CRLF", async () => {
  const { repo, vault, indexer } = setup();
  const { path } = await repo.createHabit({ id: "habit-crlf", title: "Walk" });
  const original = await vault.read(path);
  const external = original.replace(/\n/g, "\r\n").replace("---\r\n", "---\r\ncustom: keep\r\n");
  await vault.write(path, external);
  await indexer.indexFile(path, external, {});
  await repo.updateHabit("habit-crlf", { body: "new\nbody" });
  const updated = await vault.read(path);
  assert.match(updated, /custom: keep\r\n/);
  assert.match(updated, /new\r\nbody/);

});

test("updateHabit rejects unknown fields", async () => {
  const { repo } = setup();
  await repo.createHabit({ id: "habit-walk", title: "Walk" });
  await assert.rejects(() => repo.updateHabit("habit-walk", { bogus: 1 }), /Unknown habit field/);
});

test("archiveHabit sets archived-at", async () => {
  const { repo } = setup();
  await repo.createHabit({ id: "habit-walk", title: "Walk" });
  const archived = await repo.archiveHabit("habit-walk");
  assert.equal(archived.archivedAt, NOW);
});

test("checkIn creates a per-day habit-log file and indexes the event", async () => {
  const { repo, vault, index } = setup();
  await repo.createHabit({ id: "habit-walk", title: "Walk" });
  const result = await repo.checkIn("habit-walk", { localDate: "2026-07-21", status: "done", value: 1 });
  assert.equal(result.logPath, "habits/habit-logs/2026/2026-07-21.md");
  const log = await vault.read(result.logPath);
  assert.match(log, /balaur-type: habit-log/);
  assert.match(log, /balaur:habit-entry/);
  assert.match(log, /habit=habit-walk/);
  assert.match(log, /status=done/);
  const entries = index.allHabitEntries();
  assert.equal(entries.length, 1);
  assert.equal(entries[0].habitId, "habit-walk");
  assert.equal(entries[0].localDate, "2026-07-21");
  assert.equal(entries[0].status, "done");
  assert.equal(entries[0].value, 1);
});

test("checkIn preserves trailing bytes and CRLF when appending", async () => {
  const { repo, vault, indexer } = setup();
  await repo.createHabit({ id: "habit-walk", title: "Walk" });
  const first = await repo.checkIn("habit-walk", { localDate: "2026-07-21" });
  const original = await vault.read(first.logPath);
  const external = original.replace(/\n/g, "\r\n").replace(/\r\n$/, "  \r\n");
  await vault.write(first.logPath, external);
  await indexer.indexFile(first.logPath, external, {});
  const second = await repo.checkIn("habit-walk", { localDate: "2026-07-21", value: 2 });
  const final = await vault.read(first.logPath);
  assert.ok(final.startsWith(external), "existing bytes must remain untouched");
  assert.match(final, new RegExp(`balaur:habit-entry id=${second.entryId}[^\\r\\n]*\\r\\n`));
});

test("checkIn appends to the same day's log, preserving history (not overwriting)", async () => {
  const { repo, index } = setup();
  await repo.createHabit({ id: "habit-walk", title: "Walk" });
  await repo.checkIn("habit-walk", { localDate: "2026-07-21", value: 1 });
  await repo.checkIn("habit-walk", { localDate: "2026-07-21", value: 2 });
  const entries = index.allHabitEntries().filter((e) => e.habitId === "habit-walk");
  assert.equal(entries.length, 2, "both check-ins are preserved as distinct events");
  assert.deepEqual(entries.map((e) => e.value).sort(), [1, 2]);
});

test("multiple habits share one per-day log file", async () => {
  const { repo, vault, index } = setup();
  await repo.createHabit({ id: "habit-a", title: "A" });
  await repo.createHabit({ id: "habit-b", title: "B" });
  await repo.checkIn("habit-a", { localDate: "2026-07-21" });
  await repo.checkIn("habit-b", { localDate: "2026-07-21" });
  const log = await vault.read("habits/habit-logs/2026/2026-07-21.md");
  assert.match(log, /habit=habit-a/);
  assert.match(log, /habit=habit-b/);
  assert.equal(index.allHabitEntries().length, 2);
});

test("check-ins on different days create separate preserved log files", async () => {
  const { repo, vault } = setup();
  await repo.createHabit({ id: "habit-walk", title: "Walk" });
  await repo.checkIn("habit-walk", { localDate: "2026-07-21" });
  await repo.checkIn("habit-walk", { localDate: "2026-07-22" });
  assert.equal(await vault.exists("habits/habit-logs/2026/2026-07-21.md"), true);
  assert.equal(await vault.exists("habits/habit-logs/2026/2026-07-22.md"), true);
});

test("entriesFor returns a habit's events newest-first", async () => {
  const { repo } = setup();
  await repo.createHabit({ id: "habit-walk", title: "Walk" });
  await repo.checkIn("habit-walk", { localDate: "2026-07-21" });
  await repo.checkIn("habit-walk", { localDate: "2026-07-22" });
  const entries = repo.entriesFor("habit-walk");
  assert.equal(entries.length, 2);
  assert.equal(entries[0].localDate, "2026-07-22");
  assert.equal(entries[1].localDate, "2026-07-21");
});

test("checkIn requires an existing habit", async () => {
  const { repo } = setup();
  await assert.rejects(() => repo.checkIn("habit-ghost", { localDate: "2026-07-21" }), /Habit not found/);
});

test("habitLogPath formats the dated path and rejects bad dates", () => {
  assert.equal(habitLogPath("2026-07-21"), "habits/habit-logs/2026/2026-07-21.md");
  assert.equal(habitLogPath("2025-12-31"), "habits/habit-logs/2025/2025-12-31.md");
  assert.throws(() => habitLogPath("07/21/2026"), /Bad local date/);
  assert.throws(() => habitLogPath(""), /Bad local date/);
});
