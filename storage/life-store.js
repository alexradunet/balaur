import sqlite3InitModule from "../vendor/sqlite/sqlite3.mjs";

const SCHEMA_VERSION = 1;
const TABLES = ["canvases", "canvas_nodes", "tasks", "habits", "habit_entries", "journal_entries", "calendar_events", "activity_log"];

const now = () => new Date().toISOString();
const asJSON = value => value == null ? null : JSON.stringify(value);
const fromJSON = value => { try { return value == null ? null : JSON.parse(value); } catch (_) { return null; } };
const contentHash = value => {
  let hash = 2166136261;
  for (const char of String(value || "")) { hash ^= char.charCodeAt(0); hash = Math.imul(hash, 16777619); }
  return (hash >>> 0).toString(16).padStart(8, "0");
};
const canvasNodeTitle = node => {
  if (node.type === "text") return node.text.match(/^#{1,2}\s+(.+)$/m)?.[1] || "Text note";
  if (node.type === "group") return node.label || "Group";
  if (node.type === "file") return node.file?.split("/").pop() || "File";
  if (node.type === "link") { try { return new URL(node.url).hostname; } catch (_) { return "Link"; } }
  return node.id;
};

class SqliteLifeStore {
  constructor(sqlite3, db) {
    this.sqlite3 = sqlite3;
    this.db = db;
    this.backend = "sqlite-wasm-kvvfs-localStorage";
  }

  migrate() {
    this.db.exec("PRAGMA foreign_keys=ON;");
    const version = Number(this.db.selectValue("PRAGMA user_version") || 0);
    if (version > SCHEMA_VERSION) throw new Error(`Life database schema ${version} is newer than this Balaur build`);
    if (version < 1) this.db.transaction(() => {
      this.db.exec(`
        CREATE TABLE IF NOT EXISTS canvases (
          id TEXT PRIMARY KEY, title TEXT NOT NULL, path TEXT, parent_id TEXT, jd_code TEXT, updated_at TEXT NOT NULL
        );
        CREATE TABLE IF NOT EXISTS canvas_nodes (
          canvas_id TEXT NOT NULL, node_id TEXT NOT NULL, node_type TEXT NOT NULL,
          title TEXT, content_hash TEXT, updated_at TEXT NOT NULL,
          PRIMARY KEY (canvas_id, node_id)
        );
        CREATE TABLE IF NOT EXISTS tasks (
          id TEXT PRIMARY KEY, canvas_id TEXT NOT NULL, node_id TEXT, block_key TEXT NOT NULL DEFAULT '',
          title TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'inbox'
            CHECK(status IN ('inbox','next','scheduled','waiting','done','cancelled')),
          priority INTEGER, scheduled_on TEXT, due_on TEXT, completed_at TEXT,
          estimate_minutes INTEGER, recurrence_json TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
          UNIQUE(canvas_id, node_id, block_key)
        );
        CREATE INDEX IF NOT EXISTS tasks_schedule_idx ON tasks(scheduled_on, status);
        CREATE INDEX IF NOT EXISTS tasks_due_idx ON tasks(due_on, status);
        CREATE TABLE IF NOT EXISTS habits (
          id TEXT PRIMARY KEY, canvas_id TEXT, node_id TEXT, title TEXT NOT NULL,
          schedule_json TEXT NOT NULL, target REAL, unit TEXT, archived_at TEXT,
          created_at TEXT NOT NULL, updated_at TEXT NOT NULL
        );
        CREATE TABLE IF NOT EXISTS habit_entries (
          habit_id TEXT NOT NULL, local_date TEXT NOT NULL,
          status TEXT NOT NULL CHECK(status IN ('done','skipped','missed')),
          value REAL, completed_at TEXT, note TEXT,
          PRIMARY KEY (habit_id, local_date)
        );
        CREATE INDEX IF NOT EXISTS habit_entries_date_idx ON habit_entries(local_date);
        CREATE TABLE IF NOT EXISTS journal_entries (
          local_date TEXT PRIMARY KEY, canvas_id TEXT NOT NULL, node_id TEXT,
          created_at TEXT NOT NULL, updated_at TEXT NOT NULL
        );
        CREATE TABLE IF NOT EXISTS calendar_events (
          id TEXT PRIMARY KEY, title TEXT NOT NULL, starts_at TEXT, ends_at TEXT,
          local_date TEXT, timezone TEXT, all_day INTEGER NOT NULL DEFAULT 0,
          canvas_id TEXT, node_id TEXT, source TEXT NOT NULL DEFAULT 'orbit',
          created_at TEXT NOT NULL, updated_at TEXT NOT NULL
        );
        CREATE INDEX IF NOT EXISTS calendar_events_date_idx ON calendar_events(local_date, starts_at);
        CREATE TABLE IF NOT EXISTS activity_log (
          id INTEGER PRIMARY KEY AUTOINCREMENT, event_type TEXT NOT NULL,
          entity_id TEXT, occurred_at TEXT NOT NULL, payload_json TEXT
        );
        PRAGMA user_version=1;
      `);
    });
    return this;
  }

  query(sql, bind = []) { return (bind.length ? this.db.selectObjects(sql, bind) : this.db.selectObjects(sql)).map(row => ({ ...row })); }
  value(sql, bind = []) { return bind.length ? this.db.selectValue(sql, bind) : this.db.selectValue(sql); }
  run(sql, bind = []) { if (bind.length) this.db.exec({ sql, bind }); else this.db.exec(sql); return this; }
  transaction(callback) { return this.db.transaction(callback); }

  writeCanvasRecord(record, timestamp = now()) {
    this.run(`INSERT INTO canvases(id,title,path,parent_id,jd_code,updated_at) VALUES(?,?,?,?,?,?)
      ON CONFLICT(id) DO UPDATE SET title=excluded.title,path=excluded.path,parent_id=excluded.parent_id,jd_code=excluded.jd_code,updated_at=excluded.updated_at`,
      [record.id, record.title, record.path || null, record.parentId || null, record.jdCode || null, timestamp]);
    this.run("DELETE FROM canvas_nodes WHERE canvas_id=?", [record.id]);
    for (const node of record.document?.nodes || []) {
      const content = node.type === "text" ? node.text : node.type === "link" ? node.url : node.type === "file" ? node.file : node.label;
      this.run("INSERT INTO canvas_nodes(canvas_id,node_id,node_type,title,content_hash,updated_at) VALUES(?,?,?,?,?,?)",
        [record.id, node.id, node.type, canvasNodeTitle(node), contentHash(content), timestamp]);
    }
  }
  syncCanvasRecord(record) { this.transaction(() => this.writeCanvasRecord(record)); return this.stats(); }
  syncWorkspaceIndex(workspace) {
    const timestamp = now();
    this.transaction(() => {
      this.run("DELETE FROM canvas_nodes"); this.run("DELETE FROM canvases");
      for (const record of Object.values(workspace.canvases || {})) this.writeCanvasRecord(record, timestamp);
    });
    return this.stats();
  }

  upsertTask(task) {
    const timestamp = now(), createdAt = task.createdAt || timestamp;
    this.run(`INSERT INTO tasks(id,canvas_id,node_id,block_key,title,status,priority,scheduled_on,due_on,completed_at,estimate_minutes,recurrence_json,created_at,updated_at)
      VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
      ON CONFLICT(id) DO UPDATE SET canvas_id=excluded.canvas_id,node_id=excluded.node_id,block_key=excluded.block_key,title=excluded.title,status=excluded.status,priority=excluded.priority,scheduled_on=excluded.scheduled_on,due_on=excluded.due_on,completed_at=excluded.completed_at,estimate_minutes=excluded.estimate_minutes,recurrence_json=excluded.recurrence_json,updated_at=excluded.updated_at`,
      [task.id, task.canvasId, task.nodeId || null, task.blockKey || "", task.title, task.status || "inbox", task.priority ?? null, task.scheduledOn || null, task.dueOn || null, task.completedAt || null, task.estimateMinutes ?? null, asJSON(task.recurrence), createdAt, timestamp]);
    this.log("task.upserted", task.id, task); return this.task(task.id);
  }
  task(id) { const row = this.db.selectObject("SELECT * FROM tasks WHERE id=?", [id]); return row ? this.mapTask(row) : null; }
  tasks({ status, scheduledOn, dueBefore, canvasId } = {}) {
    const where = [], bind = [];
    if (status) { where.push("status=?"); bind.push(status); }
    if (scheduledOn) { where.push("scheduled_on=?"); bind.push(scheduledOn); }
    if (dueBefore) { where.push("due_on<=?"); bind.push(dueBefore); }
    if (canvasId) { where.push("canvas_id=?"); bind.push(canvasId); }
    return this.query(`SELECT * FROM tasks${where.length ? ` WHERE ${where.join(" AND ")}` : ""} ORDER BY COALESCE(scheduled_on,due_on,'9999-12-31'), COALESCE(priority,99), created_at`, bind).map(row => this.mapTask(row));
  }
  mapTask(row) { return { id:row.id,canvasId:row.canvas_id,nodeId:row.node_id,blockKey:row.block_key,title:row.title,status:row.status,priority:row.priority,scheduledOn:row.scheduled_on,dueOn:row.due_on,completedAt:row.completed_at,estimateMinutes:row.estimate_minutes,recurrence:fromJSON(row.recurrence_json),createdAt:row.created_at,updatedAt:row.updated_at }; }
  updateTask(id, patch = {}) {
    const columns={title:"title",status:"status",priority:"priority",scheduledOn:"scheduled_on",dueOn:"due_on",completedAt:"completed_at",estimateMinutes:"estimate_minutes",recurrence:"recurrence_json"},sets=[],bind=[];
    for(const [key,column] of Object.entries(columns))if(Object.hasOwn(patch,key)){sets.push(`${column}=?`);bind.push(key==="recurrence"?asJSON(patch[key]):patch[key]??null);}
    if(!sets.length)return this.task(id);sets.push("updated_at=?");bind.push(now(),id);this.run(`UPDATE tasks SET ${sets.join(",")} WHERE id=?`,bind);this.log("task.updated",id,patch);return this.task(id);
  }
  completeTask(id, completedAt = now()) { this.run("UPDATE tasks SET status='done',completed_at=?,updated_at=? WHERE id=?", [completedAt, now(), id]); this.log("task.completed", id); return this.task(id); }
  deleteTask(id){this.run("DELETE FROM tasks WHERE id=?",[id]);this.log("task.deleted",id);return true;}

  upsertHabit(habit) {
    const timestamp = now();
    this.run(`INSERT INTO habits(id,canvas_id,node_id,title,schedule_json,target,unit,archived_at,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)
      ON CONFLICT(id) DO UPDATE SET canvas_id=excluded.canvas_id,node_id=excluded.node_id,title=excluded.title,schedule_json=excluded.schedule_json,target=excluded.target,unit=excluded.unit,archived_at=excluded.archived_at,updated_at=excluded.updated_at`,
      [habit.id, habit.canvasId || null, habit.nodeId || null, habit.title, asJSON(habit.schedule || {}), habit.target ?? null, habit.unit || null, habit.archivedAt || null, habit.createdAt || timestamp, timestamp]);
    this.log("habit.upserted", habit.id, habit); return habit;
  }
  habits() { return this.query("SELECT * FROM habits WHERE archived_at IS NULL ORDER BY title").map(row => ({ ...row, schedule: fromJSON(row.schedule_json) })); }
  recordHabit(entry) {
    this.run(`INSERT INTO habit_entries(habit_id,local_date,status,value,completed_at,note) VALUES(?,?,?,?,?,?)
      ON CONFLICT(habit_id,local_date) DO UPDATE SET status=excluded.status,value=excluded.value,completed_at=excluded.completed_at,note=excluded.note`,
      [entry.habitId, entry.localDate, entry.status || "done", entry.value ?? null, entry.completedAt || now(), entry.note || null]);
    this.log("habit.checked", entry.habitId, entry); return entry;
  }
  habitEntries(start, end) { return this.query("SELECT * FROM habit_entries WHERE local_date BETWEEN ? AND ? ORDER BY local_date", [start, end]); }

  upsertJournal(entry) {
    const timestamp = now();
    this.run(`INSERT INTO journal_entries(local_date,canvas_id,node_id,created_at,updated_at) VALUES(?,?,?,?,?)
      ON CONFLICT(local_date) DO UPDATE SET canvas_id=excluded.canvas_id,node_id=excluded.node_id,updated_at=excluded.updated_at`,
      [entry.localDate, entry.canvasId, entry.nodeId || null, entry.createdAt || timestamp, timestamp]);
    this.log("journal.upserted", entry.localDate, entry); return entry;
  }
  journal(localDate) { return this.db.selectObject("SELECT * FROM journal_entries WHERE local_date=?", [localDate]) || null; }

  upsertCalendarEvent(event) {
    const timestamp = now();
    this.run(`INSERT INTO calendar_events(id,title,starts_at,ends_at,local_date,timezone,all_day,canvas_id,node_id,source,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
      ON CONFLICT(id) DO UPDATE SET title=excluded.title,starts_at=excluded.starts_at,ends_at=excluded.ends_at,local_date=excluded.local_date,timezone=excluded.timezone,all_day=excluded.all_day,canvas_id=excluded.canvas_id,node_id=excluded.node_id,source=excluded.source,updated_at=excluded.updated_at`,
      [event.id, event.title, event.startsAt || null, event.endsAt || null, event.localDate || null, event.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone, event.allDay ? 1 : 0, event.canvasId || null, event.nodeId || null, event.source || "orbit", event.createdAt || timestamp, timestamp]);
    this.log("calendar.upserted", event.id, event); return event;
  }
  calendarEvents(start, end) { return this.query("SELECT * FROM calendar_events WHERE (local_date BETWEEN ? AND ?) OR (starts_at>=? AND starts_at<?) ORDER BY COALESCE(starts_at,local_date)", [start, end, start, `${end}T23:59:59.999Z`]); }

  log(eventType, entityId = null, payload = null) { this.run("INSERT INTO activity_log(event_type,entity_id,occurred_at,payload_json) VALUES(?,?,?,?)", [eventType, entityId, now(), asJSON(payload)]); }
  stats() { return { backend: this.backend, sqliteVersion: this.sqlite3.version.libVersion, schemaVersion: Number(this.value("PRAGMA user_version")), canvases: Number(this.value("SELECT count(*) FROM canvases")), nodes: Number(this.value("SELECT count(*) FROM canvas_nodes")), tasks: Number(this.value("SELECT count(*) FROM tasks")), habits: Number(this.value("SELECT count(*) FROM habits")), events: Number(this.value("SELECT count(*) FROM calendar_events")) }; }

  exportSnapshot() {
    const data = { schemaVersion: SCHEMA_VERSION };
    for (const table of TABLES) data[table] = this.query(`SELECT * FROM ${table}`);
    return data;
  }
  importSnapshot(snapshot) {
    if (!snapshot || Number(snapshot.schemaVersion) > SCHEMA_VERSION) throw new Error("Unsupported life-data snapshot");
    const columns = Object.create(null);
    for (const table of TABLES) columns[table] = new Set(this.query(`PRAGMA table_info(${table})`).map(row => row.name));
    this.transaction(() => {
      for (const table of [...TABLES].reverse()) this.run(`DELETE FROM ${table}`);
      for (const table of TABLES) for (const row of snapshot[table] || []) {
        const keys = Object.keys(row).filter(key => columns[table].has(key)); if (!keys.length) continue;
        this.run(`INSERT OR REPLACE INTO ${table}(${keys.join(",")}) VALUES(${keys.map(() => "?").join(",")})`, keys.map(key => row[key]));
      }
    });
    return this.stats();
  }

  close() { this.db.close(); }
}

async function initializeLifeStore() {
  const status = document.querySelector("#lifeStoreStatus");
  try {
    const sqlite3 = await sqlite3InitModule({ print: () => {}, printErr: (...args) => console.warn("SQLite:", ...args) });
    const db = new sqlite3.oo1.DB(":localStorage:", "c"), store = new SqliteLifeStore(sqlite3, db).migrate();
    const workspace = window.orbitCanvas?.getWorkspace?.(); if (workspace) store.syncWorkspaceIndex(workspace);
    window.orbitLifeStore = store;
    if (status) { const stats = store.stats(); status.className = "storage-state ready"; status.innerHTML = `<i></i> SQLite ${stats.sqliteVersion} · local`; status.title = "SQLite Wasm using the browser localStorage kvvfs starter backend"; }
    window.dispatchEvent(new CustomEvent("orbit:life-store-ready", { detail: store.stats() }));
    return store;
  } catch (error) {
    console.error("Could not initialize the Balaur life database", error);
    if (status) { status.className = "storage-state error"; status.innerHTML = "<i></i> Life database unavailable"; status.title = error.message; }
    window.dispatchEvent(new CustomEvent("orbit:life-store-error", { detail: error }));
    return null;
  }
}

window.orbitLifeReady = initializeLifeStore();
export { SqliteLifeStore, SCHEMA_VERSION };
