import { inspectComponentCardIdentity, parseComponentCard } from "./component-card-codec.js";
import { splitFrontmatter } from "./frontmatter.js";
import { isCanvas } from "./canvas-validate.js";
import { caseFoldKey } from "./vault-path.js";

function isCardPath(path) {
  return path.startsWith("cards/") && path.endsWith(".md");
}

function diagnostic(path, code, message, details = null) {
  return Object.freeze({ path, code, message, ...(details ? { details: Object.freeze(details) } : {}) });
}

function freezeCard(card, placements) {
  return Object.freeze({
    ...card,
    placements: Object.freeze(placements.map((placement) => Object.freeze({ ...placement }))),
  });
}

function recoverScalar(value) {
  const source = String(value || "").trim();
  if (source.startsWith("\"")) {
    try { return JSON.parse(source); } catch {}
  }
  if (source.startsWith("'") && source.endsWith("'")) return source.slice(1, -1).replace(/''/g, "'");
  return source;
}

function recoverFallback(text, path, claimedId, message) {
  const source = String(text || "");
  const frontmatter = splitFrontmatter(source);
  let title = path.split("/").at(-1)?.replace(/\.md$/i, "") || "Component card";
  let body = source;
  if (frontmatter) {
    const titleLine = frontmatter.lines
      .slice(frontmatter.openIdx + 1, frontmatter.closeIdx)
      .find(line => /^\s*title\s*:/.test(line));
    if (titleLine) title = recoverScalar(titleLine.replace(/^\s*title\s*:\s*/, "")) || title;
    body = frontmatter.lines.slice(frontmatter.closeIdx + 1).join("");
    if (body.startsWith(frontmatter.term)) body = body.slice(frontmatter.term.length);
  }
  return Object.freeze({
    id: typeof claimedId === "string" && claimedId ? claimedId : null,
    title,
    recipe: null,
    body,
    path,
    diagnostic: message,
  });
}

export class ComponentCardCatalog {
  constructor({ vault }) {
    if (!vault) throw new TypeError("ComponentCardCatalog requires a vault");
    this.vault = vault;
    this._byPath = new Map();
    this._byId = new Map();
    this._fallbackByPath = new Map();
    this._diagnostics = [];
  }

  async rebuild() {
    const [cardFiles, canvasFiles] = await Promise.all([this.vault.list("cards/"), this.vault.list("canvases/")]);
    const parsedByPath = new Map();
    const diagnostics = [];
    const fallbackByPath = new Map();
    const folds = new Map();

    const claimsById = new Map();
    for (const meta of cardFiles) {
      if (!isCardPath(meta.path)) continue;
      const fold = caseFoldKey(meta.path);
      if (folds.has(fold)) {
        const paths = [folds.get(fold), meta.path];
        diagnostics.push(diagnostic(paths[0], "PATH_CASE_COLLISION", `Case-folded component-card path collision: ${paths.join(", ")}`, { paths }));
      } else {
        folds.set(fold, meta.path);
      }
      let claimedId = null;
      let text = "";
      try {
        text = await this.vault.read(meta.path);
        try {
          claimedId = inspectComponentCardIdentity(text).id;
        } catch {
          // The full parser below supplies the useful malformed-file diagnostic.
        }
        if (typeof claimedId === "string" && claimedId) {
          if (!claimsById.has(claimedId)) claimsById.set(claimedId, []);
          claimsById.get(claimedId).push(meta.path);
        }
        const card = parseComponentCard(text, { path: meta.path, hash: meta.hash });
        parsedByPath.set(meta.path, { card, placements: [] });
      } catch (error) {
        const message = `Malformed component card: ${error.code || error.message}`;
        diagnostics.push(diagnostic(meta.path, "CARD_MALFORMED", message, {
          errorCode: error.code || null,
          orbitId: claimedId,
        }));
        fallbackByPath.set(meta.path, recoverFallback(text, meta.path, claimedId, message));
      }
    }

    const referenced = new Set();
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
        if (node.type !== "file" || !isCardPath(node.file)) continue;
        referenced.add(node.file);
        const entry = parsedByPath.get(node.file);
        if (!entry) {
          if (!cardFiles.some((file) => file.path === node.file)) {
            diagnostics.push(diagnostic(node.file, "CARD_FILE_MISSING", `Canvas references missing component card: ${node.file}`, { canvasPath: meta.path, nodeId: node.id }));
          }
          continue;
        }
        entry.placements.push({ canvasPath: meta.path, nodeId: node.id });
      }
    }

    for (const [path, entry] of parsedByPath) {
      if (!referenced.has(path)) diagnostics.push(diagnostic(path, "CARD_ORPHANED", `Component card has no canvas placement: ${path}`));
    }

    const duplicateIds = new Set();
    for (const [id, paths] of claimsById) {
      if (paths.length < 2) continue;
      paths.sort();
      duplicateIds.add(id);
      diagnostics.push(diagnostic(paths[0], "DUPLICATE_ID", `Duplicate orbit-id "${id}" across ${paths.length} component cards`, { orbitId: id, paths: Object.freeze([...paths]) }));
    }

    const byPath = new Map();
    const byId = new Map();
    for (const [path, entry] of parsedByPath) {
      const card = freezeCard(entry.card, entry.placements);
      byPath.set(path, card);
      if (!duplicateIds.has(card.id)) byId.set(card.id, card);
    }
    this._byPath = byPath;
    this._byId = byId;
    this._fallbackByPath = fallbackByPath;
    this._diagnostics = diagnostics;
    return { cardCount: byPath.size, diagnostics: this.diagnostics() };
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

  getById(id) {
    return this._byId.get(id) || null;
  }

  cards() {
    return [...this._byId.values()];
  }

  diagnostics() {
    return [...this._diagnostics];
  }
}
