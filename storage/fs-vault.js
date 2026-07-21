// Node filesystem VaultStore adapter (Phase 9, ADR-0001, plan §10/§18).
//
// NODE-ONLY: maps the logical vault onto a real directory tree via
// node:fs/promises. It validates the VaultStore contract against an actual
// filesystem (not just memory), is the reference implementation for the
// browser File-System-Access adapter (same logical layout, same precondition
// semantics), and enables Node tooling (backup/export/migration) over a real
// vault folder. The change journal + revision are session-local (like
// MemoryVault); the filesystem is the durable store and a cold rebuild always
// works. Tested against a temp directory in storage/phase9.test.js.

import { promises as fsp } from "node:fs";
import nodePath from "node:path";
import { VaultStore, mediaTypeFor } from "./vault-store.js";
import { contentHash } from "./content-hash.js";
import { byteLength, assertSafePath } from "./vault-path.js";
import { ConflictError, VaultError } from "./vault-errors.js";

export class FsVault extends VaultStore {
  constructor(root) {
    super();
    this.root = nodePath.resolve(root);
    this._journal = [];
    this._revision = 0;
  }

  get revision() { return this._revision; }

  _abs(p) { return nodePath.join(this.root, assertSafePath(p)); }

  _bump(path, operation, hash, oldPath) {
    this._revision += 1;
    const entry = { revision: this._revision, path, operation, hash };
    if (oldPath !== undefined) entry.oldPath = oldPath;
    this._journal.push(entry);
    return this._revision;
  }

  async _record(p) {
    const abs = this._abs(p);
    let content;
    try { content = await fsp.readFile(abs, "utf8"); }
    catch (err) {
      if (err.code === "ENOENT") return null;
      throw new VaultError(`Cannot read ${p}: ${err.message}`, { code: "STORAGE_UNAVAILABLE" });
    }
    const hash = await contentHash(content);
    let modifiedAt = null;
    try { modifiedAt = (await fsp.stat(abs)).mtime.toISOString(); } catch (_) {}
    return { path: p, mediaType: mediaTypeFor(p), content, size: byteLength(content), hash, modifiedAt, revision: this._revision };
  }

  _checkPrecondition(p, existing, expectedHash) {
    if (expectedHash === undefined) return;
    if (expectedHash === null) {
      if (existing) throw new ConflictError(`Expected "${p}" to not exist`, { code: "WRITE_CONFLICT" });
      return;
    }
    if (!existing) throw new ConflictError(`Expected existing content for "${p}"`, { code: "WRITE_CONFLICT" });
    if (existing.hash !== expectedHash) throw new ConflictError(`Hash mismatch for "${p}"`, { code: "WRITE_CONFLICT", details: { expected: expectedHash, actual: existing.hash } });
  }

  async exists(p) { return (await this._record(p)) !== null; }

  async stat(p) {
    const rec = await this._record(p);
    if (!rec) return null;
    const { content, ...meta } = rec;
    return meta;
  }

  async read(p) {
    const rec = await this._record(p);
    if (!rec) throw new VaultError(`Not found: ${p}`, { code: "NOT_FOUND" });
    return rec.content;
  }

  async _walk(rel, out) {
    const abs = rel ? nodePath.join(this.root, rel) : this.root;
    let entries;
    try { entries = await fsp.readdir(abs, { withFileTypes: true }); }
    catch (err) { if (err.code === "ENOENT") return; throw err; }
    for (const entry of entries) {
      const relPath = rel ? `${rel}/${entry.name}` : entry.name;
      if (entry.isDirectory()) await this._walk(relPath, out);
      else if (entry.isFile()) {
        const rec = await this._record(relPath);
        if (rec) { const { content, ...meta } = rec; out.push(meta); }
      }
    }
  }

  async list(prefix = "") {
    const out = [];
    await this._walk("", out);
    return out.filter((m) => m.path.startsWith(prefix)).sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));
  }

  async write(p, content, options = {}) {
    const abs = this._abs(p);
    const existing = await this._record(p);
    this._checkPrecondition(p, existing, options.expectedHash);
    const text = String(content);
    const hash = await contentHash(text);
    await fsp.mkdir(nodePath.dirname(abs), { recursive: true });
    await fsp.writeFile(abs, text, "utf8");
    const revision = this._bump(p, existing ? "modify" : "create", hash);
    this.emit({ type: existing ? "modify" : "create", path: p, hash });
    return { path: p, mediaType: options.mediaType || mediaTypeFor(p), size: byteLength(text), hash, modifiedAt: new Date().toISOString(), revision };
  }

  async remove(p, options = {}) {
    const abs = this._abs(p);
    const existing = await this._record(p);
    if (!existing) throw new VaultError(`Not found: ${p}`, { code: "NOT_FOUND" });
    this._checkPrecondition(p, existing, options.expectedHash);
    await fsp.unlink(abs);
    this._bump(p, "remove", existing.hash);
    this.emit({ type: "remove", path: p, hash: existing.hash });
    return true;
  }

  async move(from, to, options = {}) {
    const f = this._abs(from);
    const t = this._abs(to);
    const existing = await this._record(from);
    if (!existing) throw new VaultError(`Not found: ${from}`, { code: "NOT_FOUND" });
    if (await this._record(to)) throw new ConflictError(`Destination exists: ${to}`, { code: "WRITE_CONFLICT" });
    this._checkPrecondition(from, existing, options.expectedHash);
    await fsp.mkdir(nodePath.dirname(t), { recursive: true });
    await fsp.rename(f, t);
    const revision = this._bump(to, "move", existing.hash, from);
    this.emit({ type: "move", path: to, oldPath: from, hash: existing.hash });
    return { path: to, mediaType: existing.mediaType, size: existing.size, hash: existing.hash, modifiedAt: new Date().toISOString(), revision };
  }

  changesSince(revision) { return this._journal.filter((e) => e.revision > revision); }

  async snapshot() {
    const files = (await this.list(""))
      .map((m) => ({ path: m.path, mediaType: m.mediaType, text: null }))
      .sort((a, b) => (a.path < b.path ? -1 : 1));
    for (const file of files) file.text = await this.read(file.path);
    return { format: "orbit-vault-snapshot", revision: this._revision, files };
  }

  async restore(snapshot) {
    // Clear existing files, then write the snapshot's files fresh.
    for (const meta of await this.list("")) {
      try { await fsp.unlink(this._abs(meta.path)); } catch (_) {}
    }
    this._journal = [];
    this._revision = 0;
    for (const file of snapshot?.files || []) {
      const p = assertSafePath(file.path);
      const text = String(file.text);
      const hash = await contentHash(text);
      const abs = this._abs(p);
      await fsp.mkdir(nodePath.dirname(abs), { recursive: true });
      await fsp.writeFile(abs, text, "utf8");
      this._bump(p, "create", hash);
    }
    this.emit({ type: "restore", path: "", hash: null });
    return { revision: this._revision, count: (await this.list("")).length };
  }
}
