// Node filesystem VaultStore adapter. Writes are serialized, path components are
// checked for symlinks, and files are committed through temporary siblings.
import { promises as fsp } from "node:fs";
import nodePath from "node:path";
import { randomBytes } from "node:crypto";
import { VaultStore, mediaTypeFor } from "./vault-store.js";
import { contentHash } from "./content-hash.js";
import { byteLength, assertSafePath, caseFoldKey } from "./vault-path.js";
import { ConflictError, PathError, VaultError } from "./vault-errors.js";

export class FsVault extends VaultStore {
  constructor(root) {
    super(); this.root = nodePath.resolve(root); this._journal = []; this._revision = 0; this._queue = Promise.resolve();
  }
  get revision() { return this._revision; }
  _enqueue(task) { const run = this._queue.then(task, task); this._queue = run.catch(() => {}); return run; }
  _abs(p) { return nodePath.join(this.root, assertSafePath(p)); }
  _bump(path, operation, hash, oldPath) { const revision = ++this._revision; const e = { revision, path, operation, hash }; if (oldPath !== undefined) e.oldPath = oldPath; this._journal.push(e); return revision; }

  async _assertComponents(p, { leafMayMissing = true } = {}) {
    const clean = assertSafePath(p); let current = this.root;
    let rootStat;
    try { rootStat = await fsp.lstat(current); } catch (e) { if (e.code === "ENOENT") { await fsp.mkdir(current, { recursive: true }); rootStat = await fsp.lstat(current); } else throw e; }
    if (rootStat.isSymbolicLink() || !rootStat.isDirectory()) throw new PathError("Vault root is not a real directory", { code: "PATH_SYMLINK" });
    const parts = clean.split("/");
    for (let i = 0; i < parts.length; i++) {
      current = nodePath.join(current, parts[i]);
      try {
        const st = await fsp.lstat(current);
        if (st.isSymbolicLink()) throw new PathError(`Symlinked vault component: ${parts[i]}`, { code: "PATH_SYMLINK" });
        if (i < parts.length - 1 && !st.isDirectory()) throw new PathError(`Vault component is not a directory: ${parts[i]}`, { code: "PATH_COMPONENT" });
      } catch (e) {
        if (e.code === "ENOENT" && leafMayMissing) break;
        throw e;
      }
    }
    return clean;
  }

  async _record(p) {
    const clean = await this._assertComponents(p);
    const abs = this._abs(clean); let content;
    try { content = await fsp.readFile(abs, "utf8"); }
    catch (err) { if (err.code === "ENOENT") return null; throw new VaultError(`Cannot read ${clean}: ${err.message}`, { code: "STORAGE_UNAVAILABLE" }); }
    const st = await fsp.stat(abs);
    return { path: clean, mediaType: mediaTypeFor(clean), content, size: byteLength(content), hash: await contentHash(content), modifiedAt: st.mtime.toISOString(), revision: this._revision };
  }
  async exists(p) { return (await this._record(p)) !== null; }
  async stat(p) { const r = await this._record(p); if (!r) return null; const { content, ...meta } = r; return meta; }
  async read(p) { const r = await this._record(p); if (!r) throw new VaultError(`Not found: ${p}`, { code: "NOT_FOUND" }); return r.content; }

  async _walk(rel, out) {
    const abs = rel ? nodePath.join(this.root, rel) : this.root; let entries;
    try { entries = await fsp.readdir(abs, { withFileTypes: true }); }
    catch (e) { if (e.code === "ENOENT") return; throw e; }
    for (const entry of entries) {
      const relPath = rel ? `${rel}/${entry.name}` : entry.name;
      const st = await fsp.lstat(nodePath.join(this.root, relPath));
      if (st.isSymbolicLink()) throw new PathError(`Symlinked vault component: ${relPath}`, { code: "PATH_SYMLINK" });
      if (st.isDirectory()) await this._walk(relPath, out);
      else if (st.isFile()) { const rec = await this._record(relPath); if (rec) { const { content, ...meta } = rec; out.push(meta); } }
    }
  }
  async list(prefix = "") { if (prefix) await this._assertComponents(String(prefix).replace(/\/$/, ""), { leafMayMissing: true }); const out = []; await this._walk("", out); return out.filter((m) => m.path.startsWith(prefix)).sort((a, b) => a.path.localeCompare(b.path)); }
  async _checkFoldCollision(p) {
    const fold = caseFoldKey(p); for (const meta of await this.list("")) if (caseFoldKey(meta.path) === fold && meta.path !== p) throw new PathError(`Case-fold collision: "${p}" vs "${meta.path}"`, { code: "PATH_CASE_COLLISION" });
  }
  _checkPrecondition(p, existing, expectedHash) { if (expectedHash === undefined) return; if (expectedHash === null && existing) throw new ConflictError(`Expected "${p}" to not exist`, { code: "WRITE_CONFLICT" }); if (expectedHash !== null && (!existing || existing.hash !== expectedHash)) throw new ConflictError(`Hash mismatch for "${p}"`, { code: "WRITE_CONFLICT" }); }
  async _ensureParent(p) { const clean = await this._assertComponents(p); const dir = nodePath.dirname(this._abs(clean)); await fsp.mkdir(dir, { recursive: true }); await this._assertComponents(clean); return clean; }
  async _atomicWrite(p, text, existing, expectedHash) {
    const abs = this._abs(p); const tmp = `${abs}.orbit-tmp-${process.pid}-${randomBytes(6).toString("hex")}`;
    await fsp.writeFile(tmp, text, { encoding: "utf8", flag: "wx" });
    try {
      // A hard link is an O_EXCL-style commit: it never replaces a file that
      // appeared after preflight. For an existing file, re-read the expected
      // hash immediately before the atomic replacement so stale writers fail.
      if (!existing) {
        try { await fsp.link(tmp, abs); }
        catch (error) {
          if (error.code === "EEXIST") throw new ConflictError(`Expected "${p}" to not exist`, { code: "WRITE_CONFLICT" });
          throw error;
        }
        await fsp.unlink(tmp);
      } else {
        const current = await this._record(p);
        if (!current || current.hash !== (expectedHash ?? existing.hash)) throw new ConflictError(`Hash mismatch for "${p}"`, { code: "WRITE_CONFLICT" });
        await fsp.rename(tmp, abs);
      }
    } catch (e) {
      try { await fsp.unlink(tmp); } catch (_) {}
      throw e;
    }
  }
  async _write(p, content, options = {}) {
    const clean = await this._ensureParent(p); const existing = await this._record(clean); this._checkPrecondition(clean, existing, options.expectedHash); if (!existing) await this._checkFoldCollision(clean);
    const text = String(content), hash = await contentHash(text); await this._atomicWrite(clean, text, existing, options.expectedHash); const revision = this._bump(clean, existing ? "modify" : "create", hash); return { path: clean, mediaType: options.mediaType || mediaTypeFor(clean), size: byteLength(text), hash, modifiedAt: new Date().toISOString(), revision };
  }
  async write(p, content, options = {}) { return this._enqueue(() => this._write(p, content, options)); }
  async _remove(p, options = {}) { const clean = await this._assertComponents(p), existing = await this._record(clean); if (!existing) throw new VaultError(`Not found: ${clean}`, { code: "NOT_FOUND" }); this._checkPrecondition(clean, existing, options.expectedHash); await fsp.unlink(this._abs(clean)); this._bump(clean, "remove", existing.hash); return true; }
  async remove(p, options = {}) { return this._enqueue(() => this._remove(p, options)); }
  async _move(from, to, options = {}) {
    const f = await this._assertComponents(from), t = await this._assertComponents(to);
    const existing = await this._record(f);
    if (!existing) throw new VaultError(`Not found: ${f}`, { code: "NOT_FOUND" });
    if (await this._record(t)) throw new ConflictError(`Destination exists: ${t}`, { code: "WRITE_CONFLICT" });
    await this._checkFoldCollision(t); this._checkPrecondition(f, existing, options.expectedHash); await this._ensureParent(t);
    const source = this._abs(f), destination = this._abs(t);
    const current = await this._record(f);
    if (!current || current.hash !== (options.expectedHash ?? existing.hash)) throw new ConflictError(`Hash mismatch for "${f}"`, { code: "WRITE_CONFLICT" });
    // link(2) is no-replace on POSIX; unlike rename it cannot overwrite a
    // destination created by a concurrent/external writer.
    try { await fsp.link(source, destination); }
    catch (error) {
      if (error.code === "EEXIST") throw new ConflictError(`Destination exists: ${t}`, { code: "WRITE_CONFLICT" });
      throw error;
    }
    try { await fsp.unlink(source); }
    catch (error) { try { await fsp.unlink(destination); } catch (_) {} throw error; }
    const revision = this._bump(t, "move", existing.hash, f);
    return { path: t, mediaType: existing.mediaType, size: existing.size, hash: existing.hash, modifiedAt: new Date().toISOString(), revision };
  }
  async move(from, to, options = {}) { return this._enqueue(() => this._move(from, to, options)); }
  changesSince(revision) { return this._journal.filter((e) => e.revision > revision); }
  async snapshot() { const files = (await this.list("")).map((m) => ({ path: m.path, mediaType: m.mediaType, text: null })); for (const f of files) f.text = await this.read(f.path); return { format: "orbit-vault-snapshot", revision: this._revision, files: files.sort((a, b) => a.path.localeCompare(b.path)) }; }
  async restore(snapshot) { return this._enqueue(() => this._restore(snapshot)); }
  async _rename(from, to) { return fsp.rename(from, to); }
  async _restore(snapshot) {
    try { const rootStat = await fsp.lstat(this.root); if (rootStat.isSymbolicLink() || !rootStat.isDirectory()) throw new PathError("Vault root is not a real directory", { code: "PATH_SYMLINK" }); } catch (e) { if (e.code !== "ENOENT") throw e; }
    const prepared = [], seen = new Map();
    for (const file of snapshot?.files || []) { const p = assertSafePath(file.path), fold = caseFoldKey(p); if (seen.has(fold)) throw new PathError(`Case-fold collision: "${p}" vs "${seen.get(fold)}"`, { code: "PATH_CASE_COLLISION" }); seen.set(fold, p); prepared.push({ p, text: String(file.text), hash: await contentHash(String(file.text)), mediaType: file.mediaType || mediaTypeFor(p) }); }
    const staging = `${this.root}.orbit-restore-${process.pid}-${randomBytes(6).toString("hex")}`; await fsp.mkdir(staging, { recursive: true });
    const backup = `${this.root}.orbit-old-${process.pid}-${randomBytes(6).toString("hex")}`;
    let oldRootMoved = false;
    try {
      for (const file of prepared) { const abs = nodePath.join(staging, file.p); await fsp.mkdir(nodePath.dirname(abs), { recursive: true }); await fsp.writeFile(abs, file.text, { flag: "wx" }); }
      try { await this._rename(this.root, backup); oldRootMoved = true; }
      catch (error) { if (error.code !== "ENOENT") throw error; }
      try { await this._rename(staging, this.root); }
      catch (error) {
        // Activation failed after the old root was moved: put the old vault
        // back before surfacing the error. Never leave the canonical root gone.
        if (oldRootMoved) {
          try { await fsp.rm(this.root, { recursive: true, force: true }); } catch (_) {}
          await this._rename(backup, this.root);
          oldRootMoved = false;
        }
        throw error;
      }
      if (oldRootMoved) await fsp.rm(backup, { recursive: true, force: true });
    } catch (e) {
      await fsp.rm(staging, { recursive: true, force: true });
      // If cleanup itself failed before the nested rollback, make one final
      // best-effort recovery attempt while preserving the original error.
      if (oldRootMoved) {
        try { await fsp.rm(this.root, { recursive: true, force: true }); await this._rename(backup, this.root); } catch (_) {}
      }
      throw e;
    }
    this._journal = []; this._revision = 0; for (const file of prepared) this._bump(file.p, "create", file.hash); return { revision: this._revision, count: prepared.length };
  }
}
