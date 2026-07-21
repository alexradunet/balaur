// IndexedDB VaultStore adapter — browser default (Phase 2, plan §10.2).
//
// BROWSER-ONLY: IndexedDB is unavailable in Node, so this module is verified by
// `node --check` and by the browser smoke test (AGENTS.md §13/§16), not by the
// Node test runner. Its logic mirrors the fully-tested MemoryVault.
//
// Stores (plan §10.2):
//   files    path -> { path, mediaType, content, size, hash, modifiedAt, revision }
//   changes  revision (autoIncrement) -> { revision, path, operation, hash, oldPath? }
//   folds    fold (case-fold key) -> { fold, path }   (case-fold collision detection)
//   settings key -> { key, value }                    (adapter-local, non-portable)
//
// The Service Worker must never cache these records; IndexedDB is user data.

import { VaultStore, mediaTypeFor } from "./vault-store.js";
import { contentHash } from "./content-hash.js";
import { byteLength, assertSafePath, caseFoldKey } from "./vault-path.js";
import { ConflictError, PathError, StorageError, VaultError } from "./vault-errors.js";

function reqP(request) {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

function txDone(tx) {
  return new Promise((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onabort = () => reject(tx.error || new Error("Vault transaction aborted"));
    tx.onerror = () => reject(tx.error);
  });
}

export class IndexedDbVault extends VaultStore {
  constructor(name = "orbit-vault") {
    super();
    this.name = name;
    this._dbp = null;
    this._revision = 0;
  }

  get revision() { return this._revision; }

  _open() {
    if (this._dbp) return this._dbp;
    this._dbp = new Promise((resolve, reject) => {
      if (typeof indexedDB === "undefined") {
        reject(new StorageError("IndexedDB is unavailable", { code: "STORAGE_UNAVAILABLE" }));
        return;
      }
      const req = indexedDB.open(this.name, 1);
      req.onupgradeneeded = () => {
        const db = req.result;
        if (!db.objectStoreNames.contains("files")) db.createObjectStore("files", { keyPath: "path" });
        if (!db.objectStoreNames.contains("changes")) db.createObjectStore("changes", { keyPath: "revision", autoIncrement: true });
        if (!db.objectStoreNames.contains("folds")) db.createObjectStore("folds", { keyPath: "fold" });
        if (!db.objectStoreNames.contains("settings")) db.createObjectStore("settings", { keyPath: "key" });
      };
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(new StorageError("Could not open vault database", { code: "STORAGE_UNAVAILABLE", cause: req.error }));
      req.onblocked = () => reject(new StorageError("Vault database open blocked", { code: "STORAGE_UNAVAILABLE" }));
    });
    return this._dbp;
  }

  _checkPrecondition(path, existing, expectedHash) {
    if (expectedHash === undefined) return;
    if (expectedHash === null) {
      if (existing) throw new ConflictError(`Expected "${path}" to not exist`, { code: "WRITE_CONFLICT" });
      return;
    }
    if (!existing) throw new ConflictError(`Expected existing content for "${path}"`, { code: "WRITE_CONFLICT" });
    if (existing.hash !== expectedHash) {
      throw new ConflictError(`Hash mismatch for "${path}"`, { code: "WRITE_CONFLICT", details: { expected: expectedHash, actual: existing.hash } });
    }
  }

  _meta(record) {
    const { content, ...meta } = record;
    return meta;
  }

  async currentRevision() {
    const db = await this._open();
    const tx = db.transaction("changes", "readonly");
    const store = tx.objectStore("changes");
    return new Promise((resolve, reject) => {
      let last = 0;
      const cursorReq = store.openCursor(null, "prev");
      cursorReq.onsuccess = () => {
        const cursor = cursorReq.result;
        if (cursor) { resolve(cursor.key); return; }
        resolve(last);
      };
      cursorReq.onerror = () => reject(cursorReq.error);
    });
  }

  async exists(path) {
    const p = assertSafePath(path);
    const db = await this._open();
    const record = await reqP(db.transaction("files").objectStore("files").get(p));
    return record !== undefined;
  }

  async stat(path) {
    const p = assertSafePath(path);
    const db = await this._open();
    const record = await reqP(db.transaction("files").objectStore("files").get(p));
    return record ? this._meta(record) : null;
  }

  async read(path) {
    const p = assertSafePath(path);
    const db = await this._open();
    const record = await reqP(db.transaction("files").objectStore("files").get(p));
    if (!record) throw new VaultError(`Not found: ${p}`, { code: "NOT_FOUND" });
    return record.content;
  }

  async list(prefix = "") {
    const db = await this._open();
    const all = await reqP(db.transaction("files").objectStore("files").getAll());
    return all
      .filter((r) => r.path.startsWith(prefix))
      .map((r) => this._meta(r))
      .sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));
  }

  async write(path, content, options = {}) {
    const p = assertSafePath(path);
    const text = String(content);
    const hash = await contentHash(text); // hash outside the tx (async work)
    const db = await this._open();
    const tx = db.transaction(["files", "changes", "folds"], "readwrite");
    const files = tx.objectStore("files");
    const changes = tx.objectStore("changes");
    const folds = tx.objectStore("folds");
    const current = await reqP(files.get(p));
    this._checkPrecondition(p, current, options.expectedHash);
    if (!current) {
      const fk = caseFoldKey(p);
      const other = await reqP(folds.get(fk));
      if (other && other.path !== p) throw new PathError(`Case-fold collision: "${p}" vs "${other.path}"`, { code: "PATH_CASE_COLLISION" });
      folds.put({ fold: fk, path: p });
    }
    const revision = await reqP(changes.add({ path: p, operation: current ? "modify" : "create", hash }));
    const record = { path: p, mediaType: options.mediaType || mediaTypeFor(p), content: text, size: byteLength(text), hash, modifiedAt: new Date().toISOString(), revision };
    files.put(record);
    await txDone(tx);
    this._revision = revision;
    this.emit({ type: current ? "modify" : "create", path: p, hash });
    return this._meta(record);
  }

  async remove(path, options = {}) {
    const p = assertSafePath(path);
    const db = await this._open();
    const tx = db.transaction(["files", "changes", "folds"], "readwrite");
    const files = tx.objectStore("files");
    const current = await reqP(files.get(p));
    if (!current) throw new VaultError(`Not found: ${p}`, { code: "NOT_FOUND" });
    this._checkPrecondition(p, current, options.expectedHash);
    files.delete(p);
    tx.objectStore("folds").delete(caseFoldKey(p));
    const revision = await reqP(tx.objectStore("changes").add({ path: p, operation: "remove", hash: current.hash }));
    await txDone(tx);
    this._revision = revision;
    this.emit({ type: "remove", path: p, hash: current.hash });
    return true;
  }

  async move(from, to, options = {}) {
    const f = assertSafePath(from);
    const t = assertSafePath(to);
    const db = await this._open();
    const tx = db.transaction(["files", "changes", "folds"], "readwrite");
    const files = tx.objectStore("files");
    const folds = tx.objectStore("folds");
    const current = await reqP(files.get(f));
    if (!current) throw new VaultError(`Not found: ${f}`, { code: "NOT_FOUND" });
    if (await reqP(files.get(t))) throw new ConflictError(`Destination exists: ${t}`, { code: "WRITE_CONFLICT" });
    this._checkPrecondition(f, current, options.expectedHash);
    const fk = caseFoldKey(t);
    const other = await reqP(folds.get(fk));
    if (other && other.path !== t) throw new PathError(`Case-fold collision: "${t}" vs "${other.path}"`, { code: "PATH_CASE_COLLISION" });
    files.delete(f);
    folds.delete(caseFoldKey(f));
    folds.put({ fold: fk, path: t });
    const revision = await reqP(tx.objectStore("changes").add({ path: t, operation: "move", hash: current.hash, oldPath: f }));
    const record = { ...current, path: t, revision };
    files.put(record);
    await txDone(tx);
    this._revision = revision;
    this.emit({ type: "move", path: t, oldPath: f, hash: current.hash });
    return this._meta(record);
  }

  async changesSince(revision) {
    const db = await this._open();
    const tx = db.transaction("changes", "readonly");
    const store = tx.objectStore("changes");
    const range = typeof IDBKeyRange !== "undefined" ? IDBKeyRange.lowerBound(revision, true) : null;
    return new Promise((resolve, reject) => {
      const out = [];
      const cursorReq = store.openCursor(range);
      cursorReq.onsuccess = () => {
        const cursor = cursorReq.result;
        if (cursor) { out.push(cursor.value); cursor.continue(); }
        else resolve(out);
      };
      cursorReq.onerror = () => reject(cursorReq.error);
    });
  }

  async snapshot() {
    const db = await this._open();
    const all = await reqP(db.transaction("files").objectStore("files").getAll());
    const files = all
      .map((r) => ({ path: r.path, mediaType: r.mediaType, text: r.content }))
      .sort((a, b) => (a.path < b.path ? -1 : 1));
    return { format: "orbit-vault-snapshot", revision: await this.currentRevision(), files };
  }

  async restore(snapshot) {
    const db = await this._open();
    const tx = db.transaction(["files", "changes", "folds"], "readwrite");
    const files = tx.objectStore("files");
    const changes = tx.objectStore("changes");
    const folds = tx.objectStore("folds");
    files.clear();
    changes.clear();
    folds.clear();
    let count = 0;
    let revision = 0;
    for (const file of snapshot?.files || []) {
      const p = assertSafePath(file.path);
      const text = String(file.text);
      const hash = await contentHash(text);
      folds.put({ fold: caseFoldKey(p), path: p });
      revision = await reqP(changes.add({ path: p, operation: "create", hash }));
      files.put({ path: p, mediaType: file.mediaType || mediaTypeFor(p), content: text, size: byteLength(text), hash, modifiedAt: new Date().toISOString(), revision });
      count++;
    }
    await txDone(tx);
    this._revision = revision;
    this.emit({ type: "restore", path: "", hash: null });
    return { revision, count };
  }

  async getSetting(key) {
    const db = await this._open();
    const record = await reqP(db.transaction("settings").objectStore("settings").get(key));
    return record ? record.value : undefined;
  }

  async setSetting(key, value) {
    const db = await this._open();
    const tx = db.transaction("settings", "readwrite");
    tx.objectStore("settings").put({ key, value });
    await txDone(tx);
  }
}
