import {
  collectKnownFields,
  patchFields,
  replaceBody,
  serializeFrontmatter,
  splitFrontmatter,
} from "./frontmatter.js";
import { byteLength } from "./vault-path.js";
import { ParseError, SchemaError } from "./vault-errors.js";

export const COMPONENT_CARD_SPEC = Object.freeze({
  maxTitleCodePoints: 160,
  maxFieldCodePoints: 160,
  maxUnitCodePoints: 32,
  maxBodyBytes: 32 * 1024,
  maxFileBytes: 64 * 1024,
});

const BASE_FIELDS = {
  "orbit-schema": "number",
  "orbit-type": "enum",
  "orbit-id": "string",
  title: "string",
  recipe: "enum",
  label: "string",
  progress: "number",
  trend: "enum",
  maximum: "number",
  unit: "string",
  tone: "enum",
};
const ORDER = ["orbit-schema", "orbit-type", "orbit-id", "title", "recipe", "value", "label", "progress", "trend", "maximum", "unit", "tone"];
const RECIPES = new Set(["metric", "progress", "callout", "list", "timeline"]);
const TRENDS = new Set(["up", "down", "flat"]);
const TONES = new Set(["info", "success", "warning", "danger"]);
const RECIPE_FIELDS = Object.freeze({
  metric: new Set(["value", "label", "progress", "trend"]),
  progress: new Set(["value", "maximum", "unit"]),
  callout: new Set(["tone"]),
  list: new Set(),
  timeline: new Set(),
});
const RECIPE_FIELD_NAMES = Object.freeze(["value", "label", "progress", "trend", "maximum", "unit", "tone"]);
const IDENTITY_SPEC = Object.freeze({ fields: Object.freeze({ "orbit-type": "enum", "orbit-id": "string" }) });
const ID_RE = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$/;
const PROBE_SPEC = Object.freeze({ fields: Object.freeze({ recipe: "enum" }) });

function specForRecipe(recipe) {
  return { fields: { ...BASE_FIELDS, value: recipe === "progress" ? "number" : "string" } };
}

export const ComponentCardCodec = Object.freeze({ specForRecipe, order: Object.freeze(ORDER) });

function codePoints(value) {
  return [...value].length;
}

function fail(message, code, details = null) {
  throw new SchemaError(message, { code, details });
}

function validateString(value, name, maximum, { required = false, nonEmpty = false } = {}) {
  if (value === undefined || value === null) {
    if (required) fail(`Missing required component-card field: ${name}`, "CARD_FIELD_REQUIRED", { key: name });
    return null;
  }
  if (typeof value !== "string" || (nonEmpty && value.trim().length === 0) || codePoints(value) > maximum) {
    fail(`Invalid component-card field: ${name}`, name === "title" ? "CARD_TITLE_INVALID" : "CARD_FIELD_INVALID", { key: name });
  }
  return value;
}

function validateCommon(card) {
  if (card.schema !== 1) {
    if (typeof card.schema === "number" && card.schema > 1) fail(`Unsupported orbit-schema ${card.schema} (read-only)`, "SCHEMA_NEWER");
    if (card.schema === undefined || card.schema === null) fail("Missing orbit-schema", "SCHEMA_MISSING");
    fail(`Unsupported orbit-schema ${card.schema}`, "SCHEMA_UNSUPPORTED");
  }
  if (card.type === undefined || card.type === null) fail("Missing required component-card field: orbit-type", "CARD_FIELD_REQUIRED", { key: "orbit-type" });
  if (card.type !== "component-card") fail(`Expected orbit-type "component-card", got "${card.type}"`, "CARD_TYPE_MISMATCH");
  if (card.id === undefined || card.id === null) fail("Missing required component-card field: orbit-id", "CARD_FIELD_REQUIRED", { key: "orbit-id" });
  if (typeof card.id !== "string" || !ID_RE.test(card.id)) fail("Invalid component-card orbit-id", "CARD_ID_INVALID");
  validateString(card.title, "title", COMPONENT_CARD_SPEC.maxTitleCodePoints, { required: true, nonEmpty: true });
  if (card.recipe === undefined || card.recipe === null) fail("Missing required component-card field: recipe", "CARD_FIELD_REQUIRED", { key: "recipe" });
  if (!RECIPES.has(card.recipe)) fail(`Invalid component-card recipe: ${card.recipe}`, "CARD_RECIPE_INVALID");
  if (typeof card.body !== "string") fail("Component-card body must be a string", "CARD_BODY_INVALID");
  if (byteLength(card.body) > COMPONENT_CARD_SPEC.maxBodyBytes) fail("Component-card body exceeds 32 KiB", "CARD_BODY_TOO_LARGE");
}

function validateRecipe(card) {
  const allowed = RECIPE_FIELDS[card.recipe];
  for (const key of RECIPE_FIELD_NAMES) {
    if (card[key] !== null && card[key] !== undefined && !allowed.has(key)) {
      fail(`Field ${key} does not belong to the ${card.recipe} recipe`, "CARD_FIELD_INVALID", { key, recipe: card.recipe });
    }
  }

  if (card.recipe === "metric") {
    validateString(card.value, "value", COMPONENT_CARD_SPEC.maxFieldCodePoints, { required: true });
    validateString(card.label, "label", COMPONENT_CARD_SPEC.maxFieldCodePoints);
    if (card.progress !== null && card.progress !== undefined && (typeof card.progress !== "number" || !Number.isFinite(card.progress) || card.progress < 0 || card.progress > 1)) {
      fail("Metric progress must be finite from 0 through 1", "CARD_FIELD_INVALID", { key: "progress" });
    }
    if (card.trend !== null && card.trend !== undefined && !TRENDS.has(card.trend)) fail("Invalid metric trend", "CARD_FIELD_INVALID", { key: "trend" });
  } else if (card.recipe === "progress") {
    if (typeof card.value !== "number") fail("Missing required component-card field: value", "CARD_FIELD_REQUIRED", { key: "value" });
    if (typeof card.maximum !== "number") fail("Missing required component-card field: maximum", "CARD_FIELD_REQUIRED", { key: "maximum" });
    if (!Number.isFinite(card.value) || card.value < 0) fail("Progress value must be finite and non-negative", "CARD_FIELD_INVALID", { key: "value" });
    if (!Number.isFinite(card.maximum) || card.maximum <= 0) fail("Progress maximum must be finite and positive", "CARD_FIELD_INVALID", { key: "maximum" });
    if (card.value > card.maximum) fail("Progress value must not exceed maximum", "CARD_FIELD_INVALID", { key: "value" });
    validateString(card.unit, "unit", COMPONENT_CARD_SPEC.maxUnitCodePoints);
  } else if (card.recipe === "callout" && card.tone !== null && card.tone !== undefined && !TONES.has(card.tone)) {
    fail("Invalid callout tone", "CARD_FIELD_INVALID", { key: "tone" });
  }
}

function validateCard(card) {
  validateCommon(card);
  validateRecipe(card);
  return card;
}

function bodyFromFrontmatter(fm) {
  let body = fm.lines.slice(fm.closeIdx + 1).join("");
  if (body.startsWith(fm.term)) body = body.slice(fm.term.length);
  return body;
}

function fromRaw(raw, body, options = {}) {
  return {
    schema: raw["orbit-schema"],
    type: raw["orbit-type"],
    id: raw["orbit-id"],
    title: raw.title,
    recipe: raw.recipe,
    value: raw.value ?? null,
    label: raw.label ?? null,
    progress: raw.progress ?? null,
    trend: raw.trend ?? null,
    maximum: raw.maximum ?? null,
    unit: raw.unit ?? null,
    tone: raw.tone ?? null,
    body,
    path: options.path ?? null,
    ...(options.hash === undefined ? {} : { hash: options.hash }),
  };
}

export function inspectComponentCardIdentity(text) {
  const fm = splitFrontmatter(String(text));
  if (!fm) throw new ParseError("Missing or unterminated frontmatter", { code: "FM_NO_DELIMITER" });
  const raw = collectKnownFields(fm.lines.slice(fm.openIdx + 1, fm.closeIdx), IDENTITY_SPEC);
  return {
    type: raw["orbit-type"] ?? null,
    id: raw["orbit-id"] ?? null,
  };
}

export function parseComponentCard(text, options = {}) {
  const source = String(text);
  if (byteLength(source) > COMPONENT_CARD_SPEC.maxFileBytes) fail("Component-card file exceeds 64 KiB", "CARD_FILE_TOO_LARGE");
  const fm = splitFrontmatter(source);
  if (!fm) throw new ParseError("Missing or unterminated frontmatter", { code: "FM_NO_DELIMITER" });
  const fmLines = fm.lines.slice(fm.openIdx + 1, fm.closeIdx);
  const probe = collectKnownFields(fmLines, PROBE_SPEC);
  const raw = collectKnownFields(fmLines, specForRecipe(probe.recipe));
  const card = fromRaw(raw, bodyFromFrontmatter(fm), options);
  return validateCard(card);
}

export function serializeComponentCard(input = {}) {
  const card = validateCard({
    schema: 1,
    type: "component-card",
    id: input.id ?? input.orbitId,
    title: input.title,
    recipe: input.recipe,
    value: input.value ?? null,
    label: input.label ?? null,
    progress: input.progress ?? null,
    trend: input.trend ?? null,
    maximum: input.maximum ?? null,
    unit: input.unit ?? null,
    tone: input.tone ?? null,
    body: input.body ?? "",
  });
  const fields = {
    "orbit-schema": card.schema,
    "orbit-type": card.type,
    "orbit-id": card.id,
    title: card.title,
    recipe: card.recipe,
    value: card.value,
    label: card.label,
    progress: card.progress,
    trend: card.trend,
    maximum: card.maximum,
    unit: card.unit,
    tone: card.tone,
  };
  const text = serializeFrontmatter(fields, specForRecipe(card.recipe), ORDER) + (card.body ? `\n${card.body}` : "");
  if (byteLength(text) > COMPONENT_CARD_SPEC.maxFileBytes) fail("Component-card file exceeds 64 KiB", "CARD_FILE_TOO_LARGE");
  return text;
}

const PATCH_KEYS = new Set(["title", "recipe", "value", "label", "progress", "trend", "maximum", "unit", "tone", "body"]);

export function patchComponentCard(text, patch = {}) {
  const current = parseComponentCard(text);
  const keys = Object.keys(patch);
  for (const key of keys) {
    if (key === "id" || key === "orbitId") fail("Component-card orbit-id is immutable", "CARD_ID_IMMUTABLE");
    if (!PATCH_KEYS.has(key)) fail(`Unknown component-card field: ${key}`, "CARD_UNKNOWN_FIELD", { key });
  }
  const candidate = { ...current, ...patch };
  validateCard(candidate);
  const frontmatterPatch = {};
  for (const key of keys) if (key !== "body") frontmatterPatch[key] = patch[key];
  let next = Object.keys(frontmatterPatch).length
    ? patchFields(text, frontmatterPatch, specForRecipe(candidate.recipe), { readSpec: specForRecipe(current.recipe) })
    : text;
  if (Object.hasOwn(patch, "body")) next = replaceBody(next, patch.body);
  if (byteLength(next) > COMPONENT_CARD_SPEC.maxFileBytes) fail("Component-card file exceeds 64 KiB", "CARD_FILE_TOO_LARGE");
  parseComponentCard(next);
  return next;
}
