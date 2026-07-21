// Typed errors for the file-canonical vault (Phase 1, ADR-0001).
// Every error carries a stable machine-readable `code` for index diagnostics
// (plan §11.2 index_diagnostics, §21) and optional structured `details`.

export class VaultError extends Error {
  constructor(message, { code = "VAULT_ERROR", details = null, cause = null } = {}) {
    super(message);
    this.name = new.target.name;
    this.code = code;
    this.details = details;
    if (cause) this.cause = cause;
  }
}

// Path traversal, absolute paths, schemes, reserved names, overlong components,
// case-fold/Unicode collisions (plan §7.3).
export class PathError extends VaultError {
  constructor(message, opts = {}) { super(message, { code: "PATH_INVALID", ...opts }); }
}

// Malformed frontmatter, duplicate known keys, missing delimiters,
// unsupported value grammar (plan §8).
export class ParseError extends VaultError {
  constructor(message, opts = {}) { super(message, { code: "PARSE_INVALID", ...opts }); }
}

// Missing or unsupported orbit-schema (plan §8.4).
export class SchemaError extends VaultError {
  constructor(message, opts = {}) { super(message, { code: "SCHEMA_UNSUPPORTED", ...opts }); }
}

// Optimistic-write hash mismatch (plan §10.1, §13.4).
export class ConflictError extends VaultError {
  constructor(message, opts = {}) { super(message, { code: "WRITE_CONFLICT", ...opts }); }
}

// Quota exhaustion or unavailable storage backend (plan §10.2).
export class StorageError extends VaultError {
  constructor(message, opts = {}) { super(message, { code: "STORAGE_UNAVAILABLE", ...opts }); }
}
