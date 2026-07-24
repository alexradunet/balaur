// Disposable synchronous note content projection (project 003, ADR-0004).
//
// A Note is a path-identified canonical notes/*.md file with no mandatory
// frontmatter and no orbit-id; its identity is its path and its kind
// (inbox/reference/ai) is an inert body marker. This catalog preloads note
// bodies and their canvas placements so render and AI-card detection read
// synchronously and never do an async vault read per card (AGENTS.md §5). It is
// rebuilt from the vault and never owns canonical content. Modeled directly on
// widget-catalog.js / component-card-catalog.js.

import { isCanvas } from "./canvas-validate.js";
import { caseFoldKey } from "./vault-path.js";

export function isNotePath(path) {
  return typeof path === "string" && path.startsWith("notes/") && path.endsWith(".md");
}

// Inert body markers (match app.js AI_CARD_MARKER and the inbox/reference
// markers). Single source of truth shared with note-repository.js so the marker
// strings cannot drift between the projection and the writer.
export const NOTE_KIND_MARKERS = Object.freeze({
  inbox: "<!-- orbit:inbox -->",
  reference: "<!-- orbit:reference -->",
  ai: "<!-- orbit:ai-card -->",
});

function diagnostic(path, code, message, details = null) {
  return Object.freeze({ path, code, message, ...(details ? { details: Object.freeze(details) } : {}) });
}

function noteSlug(path) {
  return path.split("/").at(-1).replace(/\.md$/, "") || "note";
}

// Title is derived, not stored: the first `# Heading` line, else null so callers
// can fall back (the catalog falls back to the path slug; the repository falls
// back to a slug for path allocation).
export function noteTitleFromBody(body) {
  const heading = String(body).split("\n").find((line) => /^#\s+/.test(line));
  const title = heading ? heading.replace(/^#\s+/, "").trim() : "";
  return title || null;
}

function deriveKind(body) {
  const text = String(body);
  for (const [kind, marker] of Object.entries(NOTE_KIND_MARKERS)) {
    if (text.includes(marker)) return kind;
  }
  return null;
}

function freezeNote(path, hash, body, placements) {
  return Object.freeze({
    path,
    hash,
    title: noteTitleFromBody(body) || noteSlug(path),
    body: String(body),
    kind: deriveKind(body),
    placements: Object.freeze(placements.map((placement) => Object.freeze({ ...placement }))),
  });
}

function fallbackNote(path, message) {
  return Object.freeze({ path, title: noteSlug(path), body: "", diagnostic: message });
}

export class NoteCatalog {
  constructor({ vault }) {
    if (!vault) throw new TypeError("NoteCatalog requires a vault");
    this.vault = vault;
    this._byPath = new Map();
    this._fallbackByPath = new Map();
    this._diagnostics = [];
  }

  async rebuild() {
    const [noteFiles, canvasFiles] = await Promise.all([
      this.vault.list("notes/"),
      this.vault.list("canvases/"),
    ]);
    const parsed = new Map();
    const fallbackByPath = new Map();
    const diagnostics = [];
    const folds = new Map();

    for (const meta of noteFiles) {
      if (!isNotePath(meta.path)) continue;
      const fold = caseFoldKey(meta.path);
      if (folds.has(fold)) {
        const paths = [folds.get(fold), meta.path];
        diagnostics.push(diagnostic(paths[0], "NOTE_PATH_CASE_COLLISION", `Case-folded note path collision: ${paths.join(", ")}`, { paths }));
      } else {
        folds.set(fold, meta.path);
      }
      try {
        const body = await this.vault.read(meta.path);
        parsed.set(meta.path, { body, hash: meta.hash, placements: [] });
      } catch (error) {
        // A note is ordinary Markdown, so any byte sequence is valid; the only
        // realistic failure is an unreadable file.
        const message = `Unreadable note: ${error.code || error.message}`;
        diagnostics.push(diagnostic(meta.path, "NOTE_UNREADABLE", message, { errorCode: error.code || null }));
        fallbackByPath.set(meta.path, fallbackNote(meta.path, message));
      }
    }

    for (const meta of canvasFiles) {
      if (!meta.path.endsWith(".canvas")) continue;
      let document;
      try {
        document = JSON.parse(await this.vault.read(meta.path));
      } catch (error) {
        diagnostics.push(diagnostic(meta.path, "CANVAS_MALFORMED", `Malformed canvas: ${error.message}`));
        continue;
      }
      if (!isCanvas(document)) {
        diagnostics.push(diagnostic(meta.path, "CANVAS_MALFORMED", "Malformed canvas: strict JSON Canvas validation failed"));
        continue;
      }
      for (const node of document.nodes) {
        if (node.type !== "file" || !isNotePath(node.file)) continue;
        const entry = parsed.get(node.file);
        if (entry) entry.placements.push({ canvasPath: meta.path, nodeId: node.id });
        else if (!noteFiles.some((file) => file.path === node.file)) {
          diagnostics.push(diagnostic(node.file, "NOTE_FILE_MISSING", `Canvas references missing note: ${node.file}`, { canvasPath: meta.path, nodeId: node.id }));
        }
      }
    }

    // A note with zero placements is normal (notes, like tasks, may be
    // unplaced), so there is deliberately no orphan diagnostic here.
    const byPath = new Map();
    for (const [path, entry] of parsed) {
      byPath.set(path, freezeNote(path, entry.hash, entry.body, entry.placements));
    }
    this._byPath = byPath;
    this._fallbackByPath = fallbackByPath;
    this._diagnostics = diagnostics;
    return { noteCount: byPath.size, diagnostics: this.diagnostics() };
  }

  async reconcile(_paths = []) {
    return this.rebuild();
  }

  getByPath(path) {
    return this._byPath.get(path) || null;
  }

  getFallbackByPath(path) {
    return this._fallbackByPath.get(path) || null;
  }

  notes() {
    return [...this._byPath.values()];
  }

  diagnostics() {
    return [...this._diagnostics];
  }
}
