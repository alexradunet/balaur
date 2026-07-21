// In-memory VaultStore adapter (Phase 2, plan §10.5).
//
// Deterministic and dependency-free, for codec/indexer/migration/conflict and
// crash-recovery tests. Supports injected failures after each operation boundary
// so tests can prove a failure cannot leave a partially updated record.

import { VaultStore, mediaTypeFor } from "./vault-store.js";
import { contentHash } from "./content-hash.js";
import { byteLength, assertSafePath, caseFoldKey } from "./vault-path.js";
import { ConflictError, PathError, VaultError } from "./vault-errors.js";

export class MemoryVault extends VaultStore {
  constructor() {
    super();
    this._files = new Map();   // path -> full record (includes content)
    this._folds = new Map();   // case-fold key -> path (collision detection)
    this._journal = [];        // { revision, path, operation, hash, oldPath? }
    this._revision = 0;
    this._fail = null;         // { op, error } one-shot injected failure
  }

  get revision() { return this._revision; }

  // Inject a one-shot failure for the next call to `op` (write/remove/move/restore).
  failNext(op, error) {
    this._fail = { op, error: error || new VaultError("Injected failure", { code: "INJECTED" }) };
  }

  _maybeFail(op) {
    if (this._fail && this._fail.op === op) {
      const err = this._fail.error;
      this._fail = null;
      throw err; // thrown before any mutation -> state stays consistent
    }
  }

  _bump(path, operation, hash, oldPath) {
    this._revision += 1;
    const entry = { revision: this._revision, path, operation, hash };
    if (oldPath !== undefined) entry.oldPath = oldPath;
    this._journal.push(entry);
    return this._revision;
  }

  // Optimistic concurrency (plan §10.1):
  //   expectedHash === undefined -> no precondition
  //   expectedHash === null      -> require the file does NOT exist (create)
  //   expectedHash === "<hash>"  -> require existing content with that hash
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

  _checkFoldCollision(path) {
    const fk = caseFoldKey(path);
    const other = this._folds.get(fk);
    if (other && other !== path) {
      throw new PathError(`Case-fold collision: "${path}" vs "${other}"`, { code: "PATH_CASE_COLLISION" });
    }
    this._folds.set(fk, path);
  }

  _meta(record) {
    const { content, ...meta } = record;
    return meta;
  }

  async exists(path) {
    return this._files.has(assertSafePath(path));
  }

  async stat(path) {
    const record = this._files.get(assertSafePath(path));
    return record ? this._meta(record) : null;
  }

  async read(path) {
    const p = assertSafePath(path);
    const record = this._files.get(p);
    if (!record) throw new VaultError(`Not found: ${p}`, { code: "NOT_FOUND" });
    return record.content;
  }

  async list(prefix = "") {
    const out = [];
    for (const [p, record] of this._files) {
      if (p.startsWith(prefix)) out.push(this._meta(record));
    }
    out.sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));
    return out;
  }

  async write(path, content, options = {}) {
    this._maybeFail("write");
    const p = assertSafePath(path);
    const existing = this._files.get(p);
    this._checkPrecondition(p, existing, options.expectedHash);
    if (!existing) this._checkFoldCollision(p);
    const text = String(content);
    const hash = await contentHash(text);
    const revision = this._bump(p, existing ? "modify" : "create", hash);
    const record = {
      path: p,
      mediaType: options.mediaType || mediaTypeFor(p),
      content: text,
      size: byteLength(text),
      hash,
      modifiedAt: new Date().toISOString(),
      revision,
    };
    this._files.set(p, record);
    this.emit({ type: existing ? "modify" : "create", path: p, hash });
    return this._meta(record);
  }

  async remove(path, options = {}) {
    this._maybeFail("remove");
    const p = assertSafePath(path);
    const existing = this._files.get(p);
    if (!existing) throw new VaultError(`Not found: ${p}`, { code: "NOT_FOUND" });
    this._checkPrecondition(p, existing, options.expectedHash);
    this._files.delete(p);
    this._folds.delete(caseFoldKey(p));
    this._bump(p, "remove", existing.hash);
    this.emit({ type: "remove", path: p, hash: existing.hash });
    return true;
  }

  async move(from, to, options = {}) {
    this._maybeFail("move");
    const f = assertSafePath(from);
    const t = assertSafePath(to);
    const existing = this._files.get(f);
    if (!existing) throw new VaultError(`Not found: ${f}`, { code: "NOT_FOUND" });
    if (this._files.has(t)) throw new ConflictError(`Destination exists: ${t}`, { code: "WRITE_CONFLICT" });
    this._checkPrecondition(f, existing, options.expectedHash);
    this._checkFoldCollision(t);
    this._files.delete(f);
    this._folds.delete(caseFoldKey(f));
    const revision = this._bump(t, "move", existing.hash, f);
    const record = { ...existing, path: t, revision };
    this._files.set(t, record);
    this.emit({ type: "move", path: t, oldPath: f, hash: existing.hash });
    return this._meta(record);
  }

  // Changed-path reconciliation support (plan §12.2).
  changesSince(revision) {
    return this._journal.filter((e) => e.revision > revision);
  }

  async snapshot() {
    const files = [...this._files.values()]
      .map((r) => ({ path: r.path, mediaType: r.mediaType, text: r.content }))
      .sort((a, b) => (a.path < b.path ? -1 : 1));
    return { format: "orbit-vault-snapshot", revision: this._revision, files };
  }

  async restore(snapshot) {
    this._maybeFail("restore");
    this._files.clear();
    this._folds.clear();
    this._journal = [];
    this._revision = 0;
    for (const file of snapshot?.files || []) {
      const p = assertSafePath(file.path);
      this._checkFoldCollision(p);
      const text = String(file.text);
      const hash = await contentHash(text);
      const revision = this._bump(p, "create", hash);
      this._files.set(p, {
        path: p,
        mediaType: file.mediaType || mediaTypeFor(p),
        content: text,
        size: byteLength(text),
        hash,
        modifiedAt: new Date().toISOString(),
        revision,
      });
    }
    this.emit({ type: "restore", path: "", hash: null });
    return { revision: this._revision, count: this._files.size };
  }
}
