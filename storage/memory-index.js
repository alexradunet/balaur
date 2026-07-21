// In-memory index — reference implementation of the LifeIndexer index port
// (Phase 3). Mirrors the SQL-backed implementation in life-store.js and is used
// by the Node test suite. transaction() snapshots and rolls back on error to
// match SQLite atomicity, so a thrown failure cannot leave a partial projection.

const SEP = "\u0000";

export class MemoryIndex {
  constructor() {
    this._source = new Map();        // path -> source_files record
    this._tasks = new Map();         // id -> task row
    this._habits = new Map();        // id -> habit row
    this._journals = new Map();      // localDate -> journal row
    this._events = new Map();        // id -> calendar-event row
    this._habitEntries = new Map();  // id -> habit event row
    this._placements = new Map();    // canvasId+nodeId -> placement row
    this._diagnostics = new Map();   // path+code -> diagnostic row
    this._state = new Map();         // key -> value
    this._diagId = 0;
  }

  transaction(fn) {
    const snap = {
      source: new Map(this._source), tasks: new Map(this._tasks), habits: new Map(this._habits),
      journals: new Map(this._journals), events: new Map(this._events), habitEntries: new Map(this._habitEntries),
      placements: new Map(this._placements), diagnostics: new Map(this._diagnostics), state: new Map(this._state),
    };
    try {
      return fn();
    } catch (err) {
      this._source = snap.source; this._tasks = snap.tasks; this._habits = snap.habits;
      this._journals = snap.journals; this._events = snap.events; this._habitEntries = snap.habitEntries;
      this._placements = snap.placements; this._diagnostics = snap.diagnostics; this._state = snap.state;
      throw err;
    }
  }

  // --- source_files ----------------------------------------------------------
  upsertSourceFile(rec) { this._source.set(rec.path, { ...rec }); }
  deleteSourceFile(path) { this._source.delete(path); }
  getSourceFile(path) { const r = this._source.get(path); return r ? { ...r } : null; }
  allSourceFiles() { return [...this._source.values()].map((r) => ({ ...r })); }

  // --- typed projections -----------------------------------------------------
  _deleteBySourcePath(map, path) { for (const [k, v] of map) if (v.sourcePath === path) map.delete(k); }

  clearProjectionForPath(path) {
    this._deleteBySourcePath(this._tasks, path);
    this._deleteBySourcePath(this._habits, path);
    this._deleteBySourcePath(this._journals, path);
    this._deleteBySourcePath(this._events, path);
    this._deleteBySourcePath(this._habitEntries, path);
  }

  insertEntityProjection(entityType, row) {
    if (entityType === "task") this._tasks.set(row.id, { ...row });
    else if (entityType === "habit") this._habits.set(row.id, { ...row });
    else if (entityType === "journal") this._journals.set(row.localDate, { ...row });
    else if (entityType === "calendar-event") this._events.set(row.id, { ...row });
  }

  insertHabitEntries(rows) { for (const row of rows) this._habitEntries.set(row.id, { ...row }); }

  // --- placements ------------------------------------------------------------
  clearAllPlacements() { this._placements.clear(); }
  replaceCanvasPlacements(canvasId, rows) {
    for (const [k, v] of this._placements) if (v.canvasId === canvasId) this._placements.delete(k);
    for (const row of rows) this._placements.set(`${row.canvasId}${SEP}${row.nodeId}`, { ...row });
  }
  placementsForEntity(entityId) { return [...this._placements.values()].filter((p) => p.entityId === entityId).map((p) => ({ ...p })); }
  removePlacementsForEntity(entityId) { for (const [k, v] of this._placements) if (v.entityId === entityId) this._placements.delete(k); }
  allPlacements() { return [...this._placements.values()].map((p) => ({ ...p })); }

  // --- diagnostics -----------------------------------------------------------
  recordDiagnostic(d) {
    const key = `${d.sourcePath ?? ""}${SEP}${d.errorCode}`;
    const now = new Date().toISOString();
    const existing = this._diagnostics.get(key);
    if (existing) {
      this._diagnostics.set(key, { ...existing, message: d.message, detailsJson: d.detailsJson ?? null, lastSeenAt: now });
    } else {
      this._diagnostics.set(key, { id: ++this._diagId, sourcePath: d.sourcePath ?? null, errorCode: d.errorCode, message: d.message, detailsJson: d.detailsJson ?? null, firstSeenAt: now, lastSeenAt: now });
    }
  }
  clearDiagnostics(path) { for (const [k, v] of this._diagnostics) if (v.sourcePath === path) this._diagnostics.delete(k); }
  allDiagnostics() { return [...this._diagnostics.values()].map((d) => ({ ...d })); }

  // --- index_state -----------------------------------------------------------
  setIndexState(key, value) { this._state.set(key, value); }
  getIndexState(key) { return this._state.has(key) ? this._state.get(key) : null; }

  // --- full reset (cold rebuild) --------------------------------------------
  clearAll() {
    this._source.clear(); this._tasks.clear(); this._habits.clear(); this._journals.clear();
    this._events.clear(); this._habitEntries.clear(); this._placements.clear();
    this._diagnostics.clear(); this._state.clear();
  }

  // --- readers (verification) ------------------------------------------------
  allTasks() { return [...this._tasks.values()].map((r) => ({ ...r })); }
  taskById(id) { const r = this._tasks.get(id); return r ? { ...r } : null; }
  allHabits() { return [...this._habits.values()].map((r) => ({ ...r })); }
  allJournals() { return [...this._journals.values()].map((r) => ({ ...r })); }
  allEvents() { return [...this._events.values()].map((r) => ({ ...r })); }
  allHabitEntries() { return [...this._habitEntries.values()].map((r) => ({ ...r })); }
}
