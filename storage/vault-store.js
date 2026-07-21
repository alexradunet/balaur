// VaultStore contract (Phase 2, plan §10.1).
//
// A platform-neutral, asynchronous logical vault. Adapters: MemoryVault (tests),
// IndexedDbVault (browser default), and later browser-directory / Tauri. The
// logical layout is identical across all adapters (plan §6).

import { StorageError } from "./vault-errors.js";

// Infer a media type from a path extension (plan §15.1). The path extension
// remains authoritative; this is a convenience default.
export function mediaTypeFor(path) {
  const p = String(path).toLowerCase();
  if (p.endsWith(".canvas")) return "application/jsoncanvas+json";
  if (p.endsWith(".md") || p.endsWith(".markdown")) return "text/markdown";
  if (p.endsWith(".json")) return "application/json";
  if (p.endsWith(".html") || p.endsWith(".htm")) return "text/html";
  return "application/octet-stream";
}

// File metadata record shape (plan §10.1). `read` returns text content; `stat`
// and `list` return this metadata (without content).
//   { path, mediaType, size, hash, modifiedAt, revision }

// Normalized change event (plan §12.3):
//   { type: "create" | "modify" | "move" | "remove" | "restore", path, oldPath?, hash }

export class VaultStore {
  constructor() {
    this._listeners = new Set();
  }

  // Register a change subscriber. Returns an unsubscribe function.
  subscribe(callback) {
    this._listeners.add(callback);
    return () => this._listeners.delete(callback);
  }

  emit(change) {
    for (const cb of [...this._listeners]) {
      try { cb(change); }
      catch (err) { console.warn("vault subscriber error", err); }
    }
  }

  // Monotonic vault revision used for warm reconciliation (plan §12.2).
  get revision() { return 0; }

  // --- abstract adapter surface (all asynchronous) --------------------------
  async list(prefix = "") { throw new StorageError("list() not implemented"); }
  async read(path) { throw new StorageError("read() not implemented"); }
  async write(path, content, options = {}) { throw new StorageError("write() not implemented"); }
  async remove(path, options = {}) { throw new StorageError("remove() not implemented"); }
  async move(from, to, options = {}) { throw new StorageError("move() not implemented"); }
  async stat(path) { throw new StorageError("stat() not implemented"); }
  async exists(path) { throw new StorageError("exists() not implemented"); }
  async snapshot() { throw new StorageError("snapshot() not implemented"); }
  async restore(snapshot) { throw new StorageError("restore() not implemented"); }
}
