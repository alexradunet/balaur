import { COMPONENT_CARD_SPEC, serializeComponentCard } from "../storage/component-card-codec.js";
import { assertSafePath, byteLength, componentCardPath } from "../storage/vault-path.js";
import { validateWidgetSource } from "../widgets/widget-policy.js";

export const GENERATED_OPERATION_SPEC = Object.freeze({
  maxOperationBytes: 160 * 1024,
  maxBodyBytes: COMPONENT_CARD_SPEC.maxBodyBytes,
  maxCoordinateMagnitude: 1_000_000,
  maxDimension: 10_000,
});

const TYPES = new Set(["component-card.create", "component-card.update", "widget.create", "widget.place"]);
const CARD_KEYS = new Set(["id", "path", "title", "recipe", "fields", "body"]);
const WIDGET_KEYS = new Set(["path", "title", "source"]);
const PATCH_KEYS = new Set(["title", "recipe", "fields", "body"]);
const FIELD_KEYS = Object.freeze(["value", "label", "progress", "trend", "maximum", "unit", "tone"]);
const FIELD_KEY_SET = new Set(FIELD_KEYS);
const RECIPE_FIELDS = Object.freeze({
  metric: new Set(["value", "label", "progress", "trend"]),
  progress: new Set(["value", "maximum", "unit"]),
  callout: new Set(["tone"]),
  list: new Set(),
  timeline: new Set(),
});
const PLACEMENT_KEYS = new Set(["id", "x", "y", "width", "height", "color"]);
const ID_RE = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$/;
const COLOR_RE = /^(?:[1-6]|#[0-9a-f]{6})$/i;

function plainObject(value, label) {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new TypeError(`${label} must be a plain data object`);
  const prototype = Object.getPrototypeOf(value);
  if ((prototype !== Object.prototype && prototype !== null) || Object.getOwnPropertySymbols(value).length) {
    throw new TypeError(`${label} must be a plain data object`);
  }
  return value;
}

export function assertPlainDataTree(value, label = "Generated operation") {
  if (value === null || typeof value === "string" || typeof value === "boolean") return value;
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value !== "object") throw new TypeError(`${label} must contain only plain JSON data`);
  if (Array.isArray(value)) {
    if (Object.getPrototypeOf(value) !== Array.prototype || Object.getOwnPropertySymbols(value).length) throw new TypeError(`${label} must contain only plain JSON data`);
    for (let index = 0; index < value.length; index += 1) if (!Object.hasOwn(value, index)) throw new TypeError(`${label} must contain only plain JSON data`);
    const descriptors = Object.getOwnPropertyDescriptors(value);
    for (const [key, descriptor] of Object.entries(descriptors)) {
      if (key === "length") continue;
      if (!/^(?:0|[1-9]\d*)$/.test(key) || Number(key) >= value.length || !("value" in descriptor) || !descriptor.enumerable) throw new TypeError(`${label} must contain only plain JSON data`);
      assertPlainDataTree(descriptor.value, `${label}[${key}]`);
    }
    return value;
  }
  plainObject(value, label);
  for (const [key, descriptor] of Object.entries(Object.getOwnPropertyDescriptors(value))) {
    if (!("value" in descriptor) || !descriptor.enumerable) throw new TypeError(`${label} must contain only plain JSON data`);
    assertPlainDataTree(descriptor.value, `${label}.${key}`);
  }
  return value;
}

function assertKnownKeys(value, allowed, label) {
  for (const key of Object.keys(value)) if (!allowed.has(key)) throw new TypeError(`Unknown ${label} field: ${key}`);
}

function serializedBytes(value) {
  let serialized;
  try {
    serialized = JSON.stringify(value);
  } catch {
    throw new TypeError("Generated operation is malformed");
  }
  if (typeof serialized !== "string") throw new TypeError("Generated operation is malformed");
  return byteLength(serialized);
}

function has(contextValue, value) {
  if (contextValue instanceof Set || contextValue instanceof Map) return contextValue.has(value);
  if (Array.isArray(contextValue)) return contextValue.includes(value);
  return Boolean(contextValue && Object.hasOwn(contextValue, value));
}

function cardFromContext(cards, id) {
  if (cards instanceof Map) return cards.get(id) || null;
  if (Array.isArray(cards)) return cards.find((card) => card?.id === id) || null;
  return cards && typeof cards === "object" ? cards[id] || null : null;
}

function validateId(value, label) {
  if (typeof value !== "string" || !ID_RE.test(value)) throw new TypeError(`Invalid ${label}`);
  return value;
}

function validateCardPath(value) {
  if (typeof value !== "string") throw new TypeError("Component-card path must be a string");
  const path = assertSafePath(value);
  if (!path.startsWith("cards/")) throw new TypeError("Component-card path must be under cards/");
  if (!path.endsWith(".md")) throw new TypeError("Component-card path must end in .md");
  return path;
}

function validateWidgetPath(value) {
  if (typeof value !== "string") throw new TypeError("Widget path must be a string");
  const path = assertSafePath(value);
  if (!path.startsWith("widgets/")) throw new TypeError("Widget path must be under widgets/");
  if (!path.endsWith(".html")) throw new TypeError("Widget path must end in .html");
  return path;
}

function validateFields(value, label) {
  const fields = plainObject(value ?? {}, label);
  assertKnownKeys(fields, FIELD_KEY_SET, label);
  return Object.fromEntries(FIELD_KEYS.filter((key) => Object.hasOwn(fields, key)).map((key) => [key, fields[key]]));
}

function validateCanvasId(value, context) {
  const canvasId = value ?? context.canvasId;
  if (typeof canvasId !== "string" || !canvasId) throw new TypeError("A target canvas is required");
  if (context.canvasIds && !has(context.canvasIds, canvasId)) throw new TypeError(`Unknown target canvas: ${canvasId}`);
  return canvasId;
}

function validatePlacement(value, context, { required }) {
  if (value === undefined || value === null) {
    if (required) throw new TypeError("A component-card placement is required");
    return null;
  }
  const placement = plainObject(value, "Component-card placement");
  assertKnownKeys(placement, PLACEMENT_KEYS, "placement");
  const { maxCoordinateMagnitude, maxDimension } = GENERATED_OPERATION_SPEC;
  for (const key of ["x", "y"]) {
    if (!Number.isInteger(placement[key]) || Math.abs(placement[key]) > maxCoordinateMagnitude) {
      throw new TypeError(`Placement geometry ${key} must be an integer from ${-maxCoordinateMagnitude} through ${maxCoordinateMagnitude}`);
    }
  }
  for (const key of ["width", "height"]) {
    if (!Number.isInteger(placement[key]) || placement[key] < 1 || placement[key] > maxDimension) {
      throw new TypeError(`Placement geometry ${key} must be an integer from 1 through ${maxDimension}`);
    }
  }
  const normalized = {
    ...(Object.hasOwn(placement, "id") ? { id: validateId(placement.id, "placement id") } : {}),
    x: placement.x,
    y: placement.y,
    width: placement.width,
    height: placement.height,
  };
  if (normalized.id && context.nodeIds && has(context.nodeIds, normalized.id)) throw new TypeError(`Canvas id already exists: ${normalized.id}`);
  if (Object.hasOwn(placement, "color")) {
    if (typeof placement.color !== "string" || !COLOR_RE.test(placement.color)) throw new TypeError("Placement color must be a standard JSON Canvas color");
    normalized.color = placement.color.startsWith("#") ? placement.color.toLowerCase() : placement.color;
  }
  return normalized;
}

function cardCandidate(card, fallbackId) {
  return {
    id: card.id ?? fallbackId,
    title: card.title,
    recipe: card.recipe,
    ...card.fields,
    body: card.body,
  };
}

function validateCreate(operation, context) {
  assertKnownKeys(operation, new Set(["type", "card", "canvasId", "placement"]), "operation");
  const input = plainObject(operation.card, "Component card");
  assertKnownKeys(input, CARD_KEYS, "card");
  const card = {
    ...(Object.hasOwn(input, "id") ? { id: validateId(input.id, "component-card orbit-id") } : {}),
    ...(Object.hasOwn(input, "path") ? { path: validateCardPath(input.path) } : {}),
    title: input.title,
    recipe: input.recipe,
    fields: validateFields(input.fields, "Component-card fields"),
    body: input.body ?? "",
  };
  if (card.id && ((context.cardIds && has(context.cardIds, card.id)) || cardFromContext(context.cards, card.id))) {
    throw new TypeError(`Component-card orbit-id already exists: ${card.id}`);
  }
  serializeComponentCard(cardCandidate(card, "generated-card"));
  if (card.id && !card.path) card.path = componentCardPath(card.title, card.id);
  return {
    type: operation.type,
    card,
    canvasId: validateCanvasId(operation.canvasId, context),
    placement: validatePlacement(operation.placement, context, { required: true }),
  };
}

function validateWidgetCreate(operation, context) {
  assertKnownKeys(operation, new Set(["type", "widget", "canvasId", "placement"]), "operation");
  const input = plainObject(operation.widget, "Widget");
  assertKnownKeys(input, WIDGET_KEYS, "widget");
  const validated = validateWidgetSource(input.source);
  if (typeof input.title !== "string" || input.title.trim() !== validated.title) {
    throw new TypeError("Widget proposal title must exactly match the validated source title");
  }
  return {
    type: operation.type,
    widget: {
      path: validateWidgetPath(input.path),
      title: validated.title,
      source: validated.source,
    },
    canvasId: validateCanvasId(operation.canvasId, context),
    placement: validatePlacement(operation.placement, context, { required: true }),
  };
}

function validateWidgetPlacement(operation, context) {
  assertKnownKeys(operation, new Set(["type", "path", "canvasId", "placement"]), "operation");
  const path = validateWidgetPath(operation.path);
  if (context.widgetPaths && !has(context.widgetPaths, path)) throw new TypeError(`Unknown widget: ${path}`);
  return {
    type: operation.type,
    path,
    canvasId: validateCanvasId(operation.canvasId, context),
    placement: validatePlacement(operation.placement, context, { required: true }),
  };
}

function validateUpdate(operation, context) {
  assertKnownKeys(operation, new Set(["type", "id", "patch", "canvasId", "placement"]), "operation");
  const id = validateId(operation.id, "component-card orbit-id");
  const current = cardFromContext(context.cards, id);
  if (!current) throw new TypeError(`Unknown component card: ${id}`);
  const input = plainObject(operation.patch ?? {}, "Component-card patch");
  assertKnownKeys(input, PATCH_KEYS, "patch");
  const suppliedFields = Object.hasOwn(input, "fields") ? validateFields(input.fields, "Component-card patch fields") : {};
  const recipe = input.recipe ?? current.recipe;
  const clearedFields = {};
  if (recipe !== current.recipe && RECIPE_FIELDS[recipe]) {
    for (const key of FIELD_KEYS) {
      if (current[key] !== null && current[key] !== undefined && !RECIPE_FIELDS[recipe].has(key)) clearedFields[key] = null;
    }
  }
  const fields = Object.fromEntries(FIELD_KEYS
    .filter((key) => Object.hasOwn(suppliedFields, key) || Object.hasOwn(clearedFields, key))
    .map((key) => [key, Object.hasOwn(suppliedFields, key) ? suppliedFields[key] : clearedFields[key]]));
  const includesFields = Object.hasOwn(input, "fields") || Object.keys(clearedFields).length > 0;
  const patch = {
    ...(Object.hasOwn(input, "title") ? { title: input.title } : {}),
    ...(Object.hasOwn(input, "recipe") ? { recipe: input.recipe } : {}),
    ...(includesFields ? { fields } : {}),
    ...(Object.hasOwn(input, "body") ? { body: input.body } : {}),
  };
  const placement = validatePlacement(operation.placement, context, { required: false });
  if (!Object.keys(patch).length && !placement) throw new TypeError("Component-card update must change fields or add a placement");
  const currentFields = Object.fromEntries(FIELD_KEYS.map((key) => [
    key,
    recipe === current.recipe ? current[key] ?? null : null,
  ]));
  const candidate = {
    id,
    title: patch.title ?? current.title,
    recipe,
    fields: { ...currentFields, ...fields },
    body: patch.body ?? current.body ?? "",
  };
  serializeComponentCard(cardCandidate(candidate, id));
  return {
    type: operation.type,
    id,
    patch,
    ...(placement ? { canvasId: validateCanvasId(operation.canvasId, context) } : {}),
    ...(placement ? { placement } : {}),
  };
}

export function validateGeneratedOperation(operation, context = {}) {
  assertPlainDataTree(operation);
  if (serializedBytes(operation) > GENERATED_OPERATION_SPEC.maxOperationBytes) throw new TypeError("Generated operation is too large");
  plainObject(operation, "Generated operation");
  if (!TYPES.has(operation.type)) throw new TypeError(`Unsupported generated operation: ${String(operation.type)}`);
  if (operation.type === "component-card.create") return validateCreate(operation, context);
  if (operation.type === "component-card.update") return validateUpdate(operation, context);
  if (operation.type === "widget.place") return validateWidgetPlacement(operation, context);
  return validateWidgetCreate(operation, context);
}

function printable(value) {
  return value === null ? "clear" : String(value);
}

function fieldDetails(fields = {}) {
  const labels = { value: "Value", label: "Label", progress: "Progress", trend: "Trend", maximum: "Maximum", unit: "Unit", tone: "Tone" };
  return FIELD_KEYS.filter((key) => Object.hasOwn(fields, key)).map((key) => {
    const value = key === "progress" && typeof fields[key] === "number" ? `${Math.round(fields[key] * 10000) / 100}%` : printable(fields[key]);
    return `${labels[key]}: ${value}`;
  });
}

function placementDetail(placement) {
  return `Placement: ${placement.width}×${placement.height} at ${placement.x}, ${placement.y}${placement.color ? ` · color ${placement.color}` : ""}`;
}

export function describeGeneratedOperation(operation) {
  const value = plainObject(operation, "Generated operation");
  if (value.type === "component-card.create") {
    const details = [
      ...(value.card.id ? [`ID: ${value.card.id}`] : []),
      ...(value.card.path ? [`File: ${value.card.path}`] : []),
      `Target canvas: ${value.canvasId}`,
      `Recipe: ${value.card.recipe}`,
      ...fieldDetails(value.card.fields),
      placementDetail(value.placement),
    ];
    return {
      title: `Create ${value.card.recipe} card`,
      summary: `Create “${value.card.title}” on canvas ${value.canvasId}`,
      details,
    };
  }
  if (value.type === "widget.create") {
    const validated = validateWidgetSource(value.widget.source);
    return {
      title: "Create live widget",
      summary: `Create “${value.widget.title}” on canvas ${value.canvasId}`,
      details: [
        `File: ${value.widget.path}`,
        `Source: ${byteLength(value.widget.source)} UTF-8 bytes`,
        "Capabilities: sandboxed scripts, inline styles, data/blob images and media",
        "Unavailable: host data/mutation, storage, network, forms, popups, navigation, workers, nested frames",
        "Self-navigation attempts pause the widget; hard request suppression is not claimed",
        placementDetail(value.placement),
      ],
      source: validated.source,
    };
  }
  if (value.type === "widget.place") {
    return {
      title: "Place saved widget",
      summary: `Place ${value.path} on canvas ${value.canvasId}`,
      details: [`File: ${value.path}`, `Target canvas: ${value.canvasId}`, placementDetail(value.placement)],
    };
  }
  if (value.type === "component-card.update") {
    const details = [
      ...(value.placement ? [`Target canvas: ${value.canvasId}`] : []),
      ...(Object.hasOwn(value.patch, "title") ? [`Title: ${printable(value.patch.title)}`] : []),
      ...(Object.hasOwn(value.patch, "recipe") ? [`Recipe: ${printable(value.patch.recipe)}`] : []),
      ...fieldDetails(value.patch.fields),
      ...(Object.hasOwn(value.patch, "body") ? [`Body: ${byteLength(value.patch.body)} UTF-8 bytes`] : []),
      ...(value.placement ? [placementDetail(value.placement)] : []),
    ];
    return value.placement
      ? { title: "Update component card", summary: `Update card ${value.id} and add a placement on canvas ${value.canvasId}`, details }
      : { title: "Update component card", summary: `Update canonical card ${value.id} across all placements`, details };
  }
  throw new TypeError(`Unsupported generated operation: ${String(value.type)}`);
}

export function recoverGeneratedPlacementFailure(operations, failedIndex, recoverable) {
  assertPlainDataTree(operations, "Generated operation plan");
  assertPlainDataTree(recoverable, "Recoverable generated file");
  if (!Number.isInteger(failedIndex) || failedIndex < 0 || failedIndex >= operations.length) throw new TypeError("Invalid failed operation index");
  const failed = operations[failedIndex];
  if (failed?.type === "widget.create") {
    if (!failed.placement || failed.widget?.path !== recoverable?.path) {
      throw new TypeError("Recoverable placement does not match the failed widget operation");
    }
    return [{
      type: "widget.place",
      path: recoverable.path,
      canvasId: failed.canvasId,
      placement: failed.placement,
    }, ...operations.slice(failedIndex + 1)];
  }
  const failedId=failed?.type==="component-card.create"?failed.card?.id:failed?.type==="component-card.update"?failed.id:null;
  if (!failedId || failedId !== recoverable?.id || !failed.placement) {
    throw new TypeError("Recoverable placement does not match the failed component-card operation");
  }
  return [
    {
      type: "component-card.update",
      id: recoverable.id,
      patch: {},
      canvasId: failed.canvasId,
      placement: failed.placement,
    },
    ...operations.slice(failedIndex + 1),
  ];
}
