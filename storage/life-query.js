// App-facing query facade over MemoryIndex. It never owns data; all rows are
// rebuilt by LifeIndexer from canonical vault files.
import { isValidInstant } from "./frontmatter.js";
import { SchemaError } from "./vault-errors.js";

const OPEN = new Set(["done", "cancelled"]);
const copyTask = (row) => ({ ...row, recurrence: row.recurrenceJson == null ? null : JSON.parse(row.recurrenceJson) });
const copyHabit = (row) => ({ ...row, weekdays: row.weekdaysJson == null ? [] : JSON.parse(row.weekdaysJson) });
const compare = (a, b) => a < b ? -1 : a > b ? 1 : 0;

export class LifeQuery {
  constructor(index) { this.index = index; }
  openTasks({ localDate = null, statuses = null, includeOverdue = true } = {}) {
    const allowed = statuses ? new Set(statuses) : null;
    return this.index.allTasks().map((row, order) => ({ row, order })).filter(({ row }) => !OPEN.has(row.status) && (!allowed || allowed.has(row.status)) && (!localDate || row.scheduledOn === localDate || (includeOverdue && row.dueOn && row.dueOn <= localDate))).sort((a, b) => {
      const ap = a.row.priority == null ? Number.POSITIVE_INFINITY : a.row.priority, bp = b.row.priority == null ? Number.POSITIVE_INFINITY : b.row.priority;
      return compare(ap, bp) || compare(a.row.dueOn || "9999-99-99", b.row.dueOn || "9999-99-99") || compare(a.row.scheduledOn || "9999-99-99", b.row.scheduledOn || "9999-99-99") || compare(a.row.title, b.row.title) || compare(a.row.id, b.row.id) || a.order - b.order;
    }).map(({ row }) => copyTask(row));
  }
  today(options = {}) { return this.openTasks(options); }
  tasksByStatus(status) { const statuses = Array.isArray(status) ? new Set(status) : new Set([status]); return this.index.allTasks().filter((row) => statuses.has(row.status)).map(copyTask).sort((a, b) => compare(a.title, b.title) || compare(a.id, b.id)); }
  tasks(status) { return status === undefined ? this.index.allTasks().map(copyTask) : this.tasksByStatus(status); }
  habits({ throughDate = null, timezone = "UTC" } = {}) {
    const entries = this.index.allHabitEntries();
    const result = this.index.allHabits().map(copyHabit).map((habit) => {
      const byDate = new Map(entries.filter((e) => e.habitId === habit.id && (!throughDate || e.localDate <= throughDate)).sort((a, b) => compare(a.occurredAt || "", b.occurredAt || "")).map((e) => [e.localDate, e]));
      const daily = [...byDate.values()].sort((a, b) => compare(b.localDate, a.localDate));
      const latest = daily[0] || null;
      let streak = 0, date = throughDate || (latest ? latest.localDate : null);
      while (date) {
        const event = byDate.get(date);
        if (!event || event.status !== "done") break;
        streak++;
        const d = new Date(`${date}T00:00:00Z`); d.setUTCDate(d.getUTCDate() - 1); date = `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}-${String(d.getUTCDate()).padStart(2, "0")}`;
      }
      return { ...habit, latestDailyState: latest ? { localDate: latest.localDate, status: latest.status, value: latest.value, occurredAt: latest.occurredAt } : null, streak };
    });
    return result;
  }
  journalForDate(localDate) { const row = this.index.allJournals().find((j) => j.localDate === localDate); return row ? { ...row } : null; }
  eventsInRange(from, to) {
    const instant = (value, label) => {
      if (value == null || value === "") return null;
      if (!isValidInstant(value) || !Number.isFinite(Date.parse(value))) throw new SchemaError(`Invalid ${label} instant: ${value}`, { code: "BAD_INSTANT" });
      return Date.parse(value);
    };
    const fromMs = instant(from, "from"), toMs = instant(to, "to");
    if (fromMs !== null && toMs !== null && fromMs > toMs) throw new SchemaError("Event range start is after its end", { code: "BAD_RANGE" });
    return this.index.allEvents().map((e) => ({ row: e, ms: instant(e.startsAt, "event") })).filter(({ ms }) => (fromMs === null || ms >= fromMs) && (toMs === null || ms < toMs)).map(({ row }) => ({ ...row, allDay: Boolean(row.allDay) })).sort((a, b) => Date.parse(a.startsAt) - Date.parse(b.startsAt) || compare(a.id, b.id));
  }
}
