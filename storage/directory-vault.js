// File System Access VaultStore adapter — browser default (ADR-0004).
//
// BROWSER-ONLY: the File System Access API is unavailable in Node, so this
// module is verified by `node --check` and the browser contract suite (the
// browser-check `contract` subcommand over an OPFS handle), not by the Node
// test runner. Its external behavior mirrors the fully-tested MemoryVault; its
// disk semantics (real mtime, case-fold-by-listing, non-atomic writes guarded
// by expectedHash) mirror FsVault.
//
// The picked folder's plain files ARE the vault: there is no content cache, so
// every read/stat/list goes to disk and external edits (an editor, Syncthing)
// are visible on demand. Writes are non-atomic (createWritable); the
// expectedHash precondition is the conflict guard. DOMExceptions are mapped to
// typed vault errors at every adapter boundary.

import { VaultStore, mediaTypeFor } from "./vault-store.js";
import { contentHash } from "./content-hash.js";
import { byteLength, assertSafePath, caseFoldKey } from "./vault-path.js";
import { ConflictError, PathError, StorageError, VaultError } from "./vault-errors.js";

export class DirectoryVault extends VaultStore {
  constructor(handle) {
    super();
    if (!handle) {
      throw new StorageError("DirectoryVault requires a FileSystemDirectoryHandle; the handle is missing", { code: "STORAGE_UNAVAILABLE" });
    }
    if (handle.kind !== "directory") {
      throw new StorageError(`DirectoryVault requires a directory handle; got kind "${handle.kind}", expected "directory"`, { code: "STORAGE_UNAVAILABLE" });
    }
    this._handle = handle;
    this._revision = 0;
    this._journal = []; // { revision, path, operation, hash, oldPath? }
  }

  get revision() { return this._revision; }

  _bump(path, operation, hash, oldPath) {
    this._revision += 1;
    const entry = { revision: this._revision, path, operation, hash };
    if (oldPath !== undefined) entry.oldPath = oldPath;
    this._journal.push(entry);
    return this._revision;
  }

  // Optimistic concurrency — the exact MemoryVault table:
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

  // Case-fold collision by listing (FsVault style): the folder may live on a
  // case-insensitive filesystem, so a create must not shadow an existing path
  // that differs only by case fold.
  async _checkFoldCollision(path) {
    const fold = caseFoldKey(path);
    for (const meta of await this.list("")) {
      if (caseFoldKey(meta.path) === fold && meta.path !== path) {
        throw new PathError(`Case-fold collision: "${path}" vs "${meta.path}"`, { code: "PATH_CASE_COLLISION" });
      }
    }
  }

  // Map a DOMException to a typed vault error. Errors that are already vault
  // errors pass through unchanged. NotFoundError maps to NOT_FOUND here (the
  // read/boundary interpretation); _stat special-cases it to null for the
  // stat/exists interpretation before this is reached.
  _wrap(error, message) {
    if (error instanceof VaultError) return error;
    const details = { name: error?.name, message: error?.message };
    switch (error?.name) {
      case "NotFoundError":
        return new VaultError(`${message}: not found`, { code: "NOT_FOUND", details });
      case "TypeMismatchError":
        return new PathError(`${message}: a directory occupies a file path or vice versa`, { code: "PATH_COMPONENT", details });
      // NotAllowedError, SecurityError (permission denied or revoked mid-session),
      // InvalidStateError, and any other unexpected DOMException:
      default:
        return new StorageError(`${message}: storage unavailable`, { code: "STORAGE_UNAVAILABLE", details });
    }
  }

  // Resolve the parent directory handle for a path, walking every segment except
  // the last with getDirectoryHandle(segment, { create }). Raw DOMExceptions
  // propagate so callers can decide whether NotFoundError is null/false or a
  // NOT_FOUND error.
  async _dirFor(path, { create } = {}) {
    const clean = assertSafePath(path);
    const parts = clean.split("/");
    const name = parts.pop();
    let dir = this._handle;
    for (const segment of parts) {
      dir = await dir.getDirectoryHandle(segment, { create: !!create });
    }
    return { dir, name };
  }

  async _fileHandle(path, { create } = {}) {
    const { dir, name } = await this._dirFor(path, { create });
    return dir.getFileHandle(name, { create: !!create });
  }

  // Resolve the File for a path (no create). Raw DOMExceptions propagate.
  async _file(path) {
    const handle = await this._fileHandle(path, {});
    return handle.getFile();
  }

  // Non-atomic stream write shared by write() and move(). createWritable is not
  // atomic-rename; the expectedHash precondition is the conflict guard (ADR-0004).
  // move() calls this directly rather than delegating to write(), preserving the
  // single-bump invariant.
  async _streamWrite(fileHandle, text) {
    const writable = await fileHandle.createWritable();
    await writable.write(text);
    await writable.close();
  }

  // Internal meta builder shared by stat/exists/write/move. A missing file (or a
  // missing parent directory) yields null; modifiedAt is the real disk mtime.
  async _stat(path) {
    const p = assertSafePath(path);
    let file;
    try {
      file = await this._file(p);
    } catch (error) {
      if (error?.name === "NotFoundError") return null;
      throw this._wrap(error, `Cannot stat ${p}`);
    }
    const text = await file.text();
    return {
      path: p,
      mediaType: mediaTypeFor(p),
      size: byteLength(text),
      hash: await contentHash(text),
      modifiedAt: new Date(file.lastModified).toISOString(),
      revision: this._revision,
    };
  }

  async exists(path) {
    return (await this._stat(path)) !== null;
  }

  async stat(path) {
    return this._stat(path);
  }

  async read(path) {
    const p = assertSafePath(path);
    let file;
    try {
      file = await this._file(p);
    } catch (error) {
      throw this._wrap(error, `Cannot read ${p}`);
    }
    return file.text();
  }

  // Recursive walk from the root handle; skip nothing (.balaur, canvases,
  // entities, widgets, and foreign files are all listed).
  async _walk(dir, rel, out) {
    for await (const entry of dir.values()) {
      const relPath = rel ? `${rel}/${entry.name}` : entry.name;
      if (entry.kind === "directory") {
        await this._walk(entry, relPath, out);
      } else if (entry.kind === "file") {
        const meta = await this._stat(relPath);
        if (meta) out.push(meta);
      }
    }
  }

  async list(prefix = "") {
    const out = [];
    await this._walk(this._handle, "", out);
    return out
      .filter((meta) => meta.path.startsWith(prefix))
      .sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));
  }

  async write(path, content, options = {}) {
    const p = assertSafePath(path);
    const existing = await this._stat(p);
    this._checkPrecondition(p, existing, options.expectedHash);
    if (!existing) await this._checkFoldCollision(p);
    const text = String(content);
    try {
      const handle = await this._fileHandle(p, { create: true });
      await this._streamWrite(handle, text);
    } catch (error) {
      throw this._wrap(error, `Cannot write ${p}`);
    }
    const hash = await contentHash(text);
    const revision = this._bump(p, existing ? "modify" : "create", hash);
    return { path: p, mediaType: options.mediaType || mediaTypeFor(p), size: byteLength(text), hash, modifiedAt: new Date().toISOString(), revision };
  }

  async remove(path, options = {}) {
    const p = assertSafePath(path);
    const existing = await this._stat(p);
    if (!existing) throw new VaultError(`Not found: ${p}`, { code: "NOT_FOUND" });
    this._checkPrecondition(p, existing, options.expectedHash);
    try {
      const handle = await this._fileHandle(p, {});
      await handle.remove();
    } catch (error) {
      throw this._wrap(error, `Cannot remove ${p}`);
    }
    this._bump(p, "remove", existing.hash);
    return true;
  }

  // Mirrors MemoryVault.move ordering so a failed destination write never deletes
  // the source: read source, destination-exists conflict, precondition, fold
  // check, destination write via the low-level _streamWrite (never delegate to
  // this.write — that would journal a spurious create and bump twice), a single
  // move bump, then remove the source.
  async move(from, to, options = {}) {
    const f = assertSafePath(from);
    const t = assertSafePath(to);
    const existing = await this._stat(f);
    if (!existing) throw new VaultError(`Not found: ${f}`, { code: "NOT_FOUND" });
    if (await this._stat(t)) throw new ConflictError(`Destination exists: ${t}`, { code: "WRITE_CONFLICT" });
    this._checkPrecondition(f, existing, options.expectedHash);
    await this._checkFoldCollision(t);
    const text = await this.read(f);
    try {
      const handle = await this._fileHandle(t, { create: true });
      await this._streamWrite(handle, text);
    } catch (error) {
      throw this._wrap(error, `Cannot write ${t}`);
    }
    const hash = await contentHash(text);
    const revision = this._bump(t, "move", hash, f);
    try {
      const sourceHandle = await this._fileHandle(f, {});
      await sourceHandle.remove();
    } catch (error) {
      throw this._wrap(error, `Cannot remove ${f}`);
    }
    return { path: t, mediaType: existing.mediaType, size: byteLength(text), hash, modifiedAt: new Date().toISOString(), revision };
  }

  // Changed-path reconciliation support (LifeIndexer.reconcileWarm). Synchronous,
  // identical to MemoryVault/FsVault.
  changesSince(revision) {
    return this._journal.filter((e) => e.revision > revision);
  }

  async snapshot() {
    const metas = await this.list("");
    const files = [];
    for (const meta of metas) {
      files.push({ path: meta.path, mediaType: meta.mediaType, text: await this.read(meta.path) });
    }
    files.sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));
    return { format: "balaur-vault-snapshot", revision: this._revision, files };
  }

  // Replaces the entire file tree of the picked folder (empty directories may
  // remain; harmless). Only reachable through explicitly confirmed destructive
  // flows (whole-space import, Reset starter), which validate into a staging
  // vault first.
  async restore(snapshot) {
    const prepared = [];
    const seen = new Map();
    for (const file of snapshot?.files || []) {
      const p = assertSafePath(file.path);
      const fold = caseFoldKey(p);
      if (seen.has(fold)) throw new PathError(`Case-fold collision: "${p}" vs "${seen.get(fold)}"`, { code: "PATH_CASE_COLLISION" });
      seen.set(fold, p);
      prepared.push({ path: p, text: file.text, mediaType: file.mediaType });
    }
    // Remove every file currently in the tree.
    for (const meta of await this.list("")) {
      try {
        const handle = await this._fileHandle(meta.path, {});
        await handle.remove();
      } catch (error) {
        throw this._wrap(error, `Cannot remove ${meta.path}`);
      }
    }
    this._journal = [];
    this._revision = 0;
    // The tree is empty, so expectedHash: null (must-not-exist) always passes;
    // the per-write fold check is redundant but harmless.
    for (const file of prepared) {
      await this.write(file.path, file.text, { expectedHash: null, mediaType: file.mediaType });
    }
    return { revision: this._revision, count: prepared.length };
  }
}
