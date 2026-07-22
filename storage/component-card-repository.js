import { parseComponentCard, patchComponentCard, serializeComponentCard } from "./component-card-codec.js";
import { isCanvas } from "./canvas-validate.js";
import { assertSafePath, componentCardPath } from "./vault-path.js";
import { ConflictError, SchemaError } from "./vault-errors.js";

const DEFAULT_GEOMETRY = Object.freeze({ x: 40, y: 40, width: 360, height: 220 });

function randomToken() {
  const crypto = globalThis.crypto;
  if (crypto?.randomUUID) return crypto.randomUUID().replace(/-/g, "").slice(0, 12);
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
}

function schemaError(message, code, details = null) {
  return new SchemaError(message, { code, details });
}

function validateCardPath(path) {
  const safe = assertSafePath(path);
  if (!safe.startsWith("cards/") || !safe.endsWith(".md")) throw schemaError(`Component-card path must be under cards/: ${safe}`, "CARD_PATH_INVALID");
  return safe;
}

export class FileComponentCardRepository {
  constructor({ vault, catalog, canvasPathFromId, now = () => new Date().toISOString(), idPrefix = "card" }) {
    if (!vault || !catalog) throw new TypeError("FileComponentCardRepository requires vault and catalog");
    this.vault = vault;
    this.catalog = catalog;
    this.canvasPathFromId = canvasPathFromId || (() => null);
    this.now = now;
    this.idPrefix = idPrefix;
  }

  _newId() {
    return `${this.idPrefix}-${randomToken()}`;
  }

  _sourceFor(id) {
    const card = this.catalog.getById(id);
    if (!card) throw schemaError(`Component card not found: ${id}`, "CARD_NOT_FOUND");
    return card;
  }

  async _loadCanvas(canvasPath) {
    const path = assertSafePath(canvasPath);
    const stat = await this.vault.stat(path);
    if (!stat) throw schemaError(`Canvas not found: ${path}`, "CANVAS_NOT_FOUND");
    let document;
    try {
      document = JSON.parse(await this.vault.read(path));
    } catch (error) {
      throw schemaError(`Invalid canvas document: ${path}`, "CANVAS_INVALID", { cause: error.message });
    }
    if (!isCanvas(document)) throw schemaError(`Invalid canvas document: ${path}`, "CANVAS_INVALID");
    return { path, stat, document };
  }

  async _saveCanvas(path, document, expectedHash) {
    if (!isCanvas(document)) throw schemaError(`Invalid canvas document: ${path}`, "CANVAS_INVALID");
    const content = JSON.stringify(document, null, 2) + "\n";
    return this.vault.write(path, content, { expectedHash });
  }

  async createCard(input = {}) {
    const id = input.id || this._newId();
    const fields = input.fields && typeof input.fields === "object" ? input.fields : {};
    const cardInput = { ...input, ...fields, id };
    delete cardInput.fields;
    delete cardInput.canvasId;
    delete cardInput.geometry;
    delete cardInput.path;
    const content = serializeComponentCard(cardInput);
    const parsed = parseComponentCard(content);
    const path = validateCardPath(input.path || componentCardPath(parsed.title, id));
    await this.catalog.rebuild();
    const identityClaimed = this.catalog.getById(id) || this.catalog.diagnostics().some((item) => item.details?.orbitId === id);
    if (identityClaimed) throw schemaError(`Component-card orbit-id already exists: ${id}`, "CARD_ID_DUPLICATE");
    await this.vault.write(path, content, { expectedHash: null });
    await this.catalog.reconcile([path]);
    let placement = null;
    if (input.canvasId) {
      try {
        placement = await this.addPlacement(id, input.canvasId, input.geometry);
      } catch (error) {
        throw schemaError(
          `Component card was saved but could not be placed: ${path}`,
          "CARD_PLACEMENT_FAILED",
          {
            recoverable: { id, path, placed: false },
            cause: error.message,
            causeCode: error.code || null,
          },
        );
      }
    }
    return { id, path, card: this.catalog.getById(id), placement };
  }

  async getCard(id) {
    const source = this._sourceFor(id);
    return parseComponentCard(await this.vault.read(source.path), { path: source.path, hash: source.hash });
  }

  async _rewritePlacementPaths(fromPath, toPath, placements, changedPaths) {
    const canvasPaths = new Set(placements.map((placement) => placement.canvasPath));
    for (const canvasPath of canvasPaths) {
      changedPaths.add(canvasPath);
      const loaded = await this._loadCanvas(canvasPath);
      let dirty = false;
      for (const node of loaded.document.nodes) {
        if (node.type === "file" && node.file === fromPath) {
          node.file = toPath;
          dirty = true;
        }
      }
      if (!dirty) continue;
      await this._saveCanvas(loaded.path, loaded.document, loaded.stat.hash);
    }
  }

  async updateCard(id, patch = {}) {
    const known = this._sourceFor(id);
    const current = await this.vault.read(known.path);
    const stat = await this.vault.stat(known.path);
    if (!stat || stat.hash !== known.hash) {
      await this.catalog.reconcile([known.path]);
      throw new ConflictError(`Component card changed before update: ${known.path}`, { code: "WRITE_CONFLICT" });
    }
    await this.catalog.rebuild();
    const source = this.catalog.getById(id);
    if (!source || source.path !== known.path || source.hash !== known.hash) {
      throw new ConflictError(`Component card changed before update: ${known.path}`, { code: "WRITE_CONFLICT" });
    }

    const next = patchComponentCard(current, patch);
    const parsed = parseComponentCard(next);
    const destination = parsed.title === source.title ? source.path : componentCardPath(parsed.title, id);
    if (destination === source.path) {
      await this.vault.write(source.path, next, { expectedHash: known.hash });
      await this.catalog.reconcile([source.path]);
      return this.catalog.getByPath(source.path);
    }

    const changedPaths = new Set([source.path, destination]);
    try {
      await this.vault.write(destination, next, { expectedHash: null });
      await this._rewritePlacementPaths(source.path, destination, source.placements, changedPaths);
      await this.vault.remove(source.path, { expectedHash: known.hash });
      await this.catalog.reconcile([...changedPaths]);
      return this.catalog.getByPath(destination);
    } catch (error) {
      await this.catalog.reconcile([...changedPaths]);
      throw error;
    }
  }

  async updateCardAndPlace(id, patch = {}, canvasId, geometry = {}) {
    const hasPatch = Object.keys(patch).length > 0;
    const card = hasPatch ? await this.updateCard(id, patch) : this._sourceFor(id);
    try {
      const placement = await this.addPlacement(id, canvasId, geometry);
      return { card: this.catalog.getById(id), placement, updated: hasPatch };
    } catch (error) {
      throw schemaError(
        `${hasPatch ? "Component-card update was saved" : "Component card is saved"} but could not be placed: ${card.path}`,
        "CARD_PLACEMENT_FAILED",
        {
          recoverable: {
            id,
            path: card.path,
            updated: hasPatch,
            placed: false,
            placement: { canvasId, geometry: { ...geometry } },
          },
          cause: error.message,
          causeCode: error.code || null,
        },
      );
    }
  }

  async addPlacement(id, canvasId, geometry = {}) {
    const source = this._sourceFor(id);
    const canvasPath = this.canvasPathFromId(canvasId);
    if (!canvasPath) throw schemaError(`No canvas path for id: ${canvasId}`, "CANVAS_NOT_FOUND");
    const loaded = await this._loadCanvas(canvasPath);
    const values = { ...DEFAULT_GEOMETRY, ...geometry };
    if (![values.x, values.y, values.width, values.height].every(Number.isInteger) || values.width <= 0 || values.height <= 0) {
      throw schemaError("Placement geometry must use integers with positive dimensions", "CANVAS_GEOMETRY_INVALID");
    }
    const nodeId = geometry.id || `node-${randomToken()}`;
    if (loaded.document.nodes.some((node) => node.id === nodeId) || loaded.document.edges.some((edge) => edge.id === nodeId)) {
      throw schemaError(`Canvas id already exists: ${nodeId}`, "CANVAS_ID_DUPLICATE");
    }
    const node = {
      id: nodeId,
      type: "file",
      file: source.path,
      x: values.x,
      y: values.y,
      width: values.width,
      height: values.height,
    };
    if (geometry.color !== undefined) node.color = geometry.color;
    loaded.document.nodes.push(node);
    await this._saveCanvas(loaded.path, loaded.document, loaded.stat.hash);
    await this.catalog.reconcile([loaded.path]);
    return { canvasId, canvasPath: loaded.path, nodeId };
  }

  async _removePlacementAtPath(canvasPath, nodeId) {
    const loaded = await this._loadCanvas(canvasPath);
    const before = loaded.document.nodes.length;
    loaded.document.nodes = loaded.document.nodes.filter((node) => node.id !== nodeId);
    if (loaded.document.nodes.length === before) return false;
    loaded.document.edges = loaded.document.edges.filter((edge) => edge.fromNode !== nodeId && edge.toNode !== nodeId);
    await this._saveCanvas(loaded.path, loaded.document, loaded.stat.hash);
    return true;
  }

  async removePlacement(canvasId, nodeId) {
    const canvasPath = this.canvasPathFromId(canvasId);
    if (!canvasPath) throw schemaError(`No canvas path for id: ${canvasId}`, "CANVAS_NOT_FOUND");
    const removed = await this._removePlacementAtPath(canvasPath, nodeId);
    if (removed) await this.catalog.reconcile([canvasPath]);
    return { removed, canvasId, nodeId };
  }

  async deleteCard(id) {
    const known = this._sourceFor(id);
    const stat = await this.vault.stat(known.path);
    if (!stat || stat.hash !== known.hash) {
      throw new ConflictError(`Component card changed before deletion: ${known.path}`, { code: "WRITE_CONFLICT" });
    }
    await this.catalog.rebuild();
    const source = this.catalog.getById(id);
    if (!source || source.path !== known.path || source.hash !== known.hash) {
      throw new ConflictError(`Component card changed before deletion: ${known.path}`, { code: "WRITE_CONFLICT" });
    }
    const placements = [...source.placements];
    const changed = [];
    for (const placement of placements) {
      if (await this._removePlacementAtPath(placement.canvasPath, placement.nodeId)) changed.push(placement.canvasPath);
    }
    await this.vault.remove(source.path, { expectedHash: known.hash });
    await this.catalog.reconcile([source.path, ...changed]);
    return { id, path: source.path, removedPlacements: changed.length };
  }
}
