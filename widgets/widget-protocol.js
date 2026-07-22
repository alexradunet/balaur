export const HOST_TO_WIDGET = "host-to-widget";
export const WIDGET_TO_HOST = "widget-to-host";
export const WIDGET_PROTOCOL_VERSION = 1;
export const MAX_WIDGET_MESSAGE_BYTES = 64 * 1024;

const HOST_MESSAGE_TYPES = new Set([
  "orbit.widget.pause.v1",
  "orbit.widget.preferences.v1",
  "orbit.widget.theme.v1",
  "orbit.widget.visibility.v1",
]);
const WIDGET_MESSAGE_TYPES = new Set([
  "orbit.widget.diagnostic.v1",
  "orbit.widget.heartbeat.v1",
  "orbit.widget.ready.v1",
  "orbit.widget.resize.v1",
  "orbit.widget.status.v1",
]);
const THEME_TOKEN_KEYS = Object.freeze([
  "surface",
  "surfaceRaised",
  "content",
  "contentMuted",
  "paper",
  "ink",
  "primary",
  "focus",
  "danger",
  "radius",
  "fontBody",
  "fontMono",
]);
const textEncoder = new TextEncoder();

function clonePlainData(value, label, ancestors = new WeakSet()) {
  if (value === null || typeof value === "string" || typeof value === "boolean") return value;
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (!value || typeof value !== "object") throw new TypeError(`${label} must contain only plain JSON data`);
  if (ancestors.has(value)) throw new TypeError(`${label} must contain only plain JSON data and cannot be cyclic`);
  ancestors.add(value);
  try {
    if (Array.isArray(value)) {
      if (Object.getPrototypeOf(value) !== Array.prototype || Object.getOwnPropertySymbols(value).length) {
        throw new TypeError(`${label} must contain only plain JSON data`);
      }
      const descriptors = Object.getOwnPropertyDescriptors(value);
      for (let index = 0; index < value.length; index += 1) {
        if (!Object.hasOwn(value, index)) throw new TypeError(`${label} must contain only plain JSON data`);
      }
      const clone = [];
      for (const [key, descriptor] of Object.entries(descriptors)) {
        if (key === "length") continue;
        if (!/^(?:0|[1-9]\d*)$/.test(key) || Number(key) >= value.length || !("value" in descriptor) || !descriptor.enumerable) {
          throw new TypeError(`${label} must contain only plain JSON data`);
        }
        clone.push(clonePlainData(descriptor.value, `${label}[${key}]`, ancestors));
      }
      return clone;
    }
    const prototype = Object.getPrototypeOf(value);
    if ((prototype !== Object.prototype && prototype !== null) || Object.getOwnPropertySymbols(value).length) {
      throw new TypeError(`${label} must contain only plain JSON data`);
    }
    const clone = {};
    for (const [key, descriptor] of Object.entries(Object.getOwnPropertyDescriptors(value))) {
      if (!("value" in descriptor) || !descriptor.enumerable) {
        throw new TypeError(`${label} must contain only plain JSON data`);
      }
      Object.defineProperty(clone, key, {
        value: clonePlainData(descriptor.value, `${label}.${key}`, ancestors),
        enumerable: true,
        configurable: true,
        writable: true,
      });
    }
    return clone;
  } finally {
    ancestors.delete(value);
  }
}

function assertKnownKeys(value, allowed, label) {
  for (const key of Object.keys(value)) {
    if (!allowed.has(key)) throw new TypeError(`Unknown ${label} field: ${key}`);
  }
}

function assertExactKeys(value, keys, label) {
  const allowed = new Set(keys);
  assertKnownKeys(value, allowed, label);
  for (const key of keys) if (!Object.hasOwn(value, key)) throw new TypeError(`${label} requires ${key}`);
}

function assertBoolean(value, label) {
  if (typeof value !== "boolean") throw new TypeError(`${label} must be a boolean`);
}

function assertString(value, label, { allowEmpty = true, maxLength = Number.POSITIVE_INFINITY } = {}) {
  if (typeof value !== "string" || (!allowEmpty && !value.trim()) || value.length > maxLength) {
    throw new TypeError(`${label} must be ${allowEmpty ? "a" : "a non-empty"} bounded string`);
  }
}

function validateTheme(payload) {
  assertExactKeys(payload, ["tokens"], "theme payload");
  const tokens = payload.tokens;
  if (!tokens || typeof tokens !== "object" || Array.isArray(tokens)) throw new TypeError("theme tokens must be an object");
  assertExactKeys(tokens, THEME_TOKEN_KEYS, "theme tokens");
  for (const key of THEME_TOKEN_KEYS) assertString(tokens[key], `theme token ${key}`, { allowEmpty: false });
}

function validatePreferences(payload) {
  assertExactKeys(payload, ["reducedMotion", "reducedTransparency", "contrast"], "preferences payload");
  assertBoolean(payload.reducedMotion, "reducedMotion");
  assertBoolean(payload.reducedTransparency, "reducedTransparency");
  if (!["no-preference", "more", "less", "custom"].includes(payload.contrast)) {
    throw new TypeError("contrast must be no-preference, more, less, or custom");
  }
}

function validateVisibility(payload) {
  assertExactKeys(payload, ["visible"], "visibility payload");
  assertBoolean(payload.visible, "visible");
}

function validatePause(payload) {
  assertKnownKeys(payload, new Set(["reason"]), "pause payload");
  if (Object.hasOwn(payload, "reason")) assertString(payload.reason, "pause reason", { maxLength: 1024 });
}

function validateEmpty(payload, label) {
  assertExactKeys(payload, [], label);
}

function validateStatus(payload) {
  assertExactKeys(payload, ["message"], "status payload");
  assertString(payload.message, "status message");
}

function validateResize(payload) {
  assertExactKeys(payload, ["width", "height"], "resize payload");
  for (const key of ["width", "height"]) {
    if (!Number.isInteger(payload[key]) || payload[key] < 1 || payload[key] > 10_000) {
      throw new TypeError(`resize ${key} must be an integer from 1 through 10000`);
    }
  }
}

function validateDiagnostic(payload) {
  assertExactKeys(payload, ["level", "message"], "diagnostic payload");
  if (!["info", "warning", "error"].includes(payload.level)) throw new TypeError("diagnostic level must be info, warning, or error");
  assertString(payload.message, "diagnostic message", { maxLength: 4096 });
}

const PAYLOAD_VALIDATORS = Object.freeze({
  "orbit.widget.theme.v1": validateTheme,
  "orbit.widget.preferences.v1": validatePreferences,
  "orbit.widget.visibility.v1": validateVisibility,
  "orbit.widget.pause.v1": validatePause,
  "orbit.widget.ready.v1": (payload) => validateEmpty(payload, "ready payload"),
  "orbit.widget.status.v1": validateStatus,
  "orbit.widget.resize.v1": validateResize,
  "orbit.widget.heartbeat.v1": (payload) => validateEmpty(payload, "heartbeat payload"),
  "orbit.widget.diagnostic.v1": validateDiagnostic,
});

export function validateWidgetMessage(direction, value) {
  if (direction !== HOST_TO_WIDGET && direction !== WIDGET_TO_HOST) throw new TypeError("Unknown widget message direction");
  const message = clonePlainData(value, "Widget message");
  if (!message || typeof message !== "object" || Array.isArray(message)) throw new TypeError("Widget message must be a plain data object");
  assertExactKeys(message, ["type", "version", "payload"], "widget message");
  if (message.version !== WIDGET_PROTOCOL_VERSION) throw new TypeError(`Widget message version must be ${WIDGET_PROTOCOL_VERSION}`);
  if (typeof message.type !== "string" || !Object.hasOwn(PAYLOAD_VALIDATORS, message.type)) throw new TypeError("Unknown widget message type");
  const allowedTypes = direction === HOST_TO_WIDGET ? HOST_MESSAGE_TYPES : WIDGET_MESSAGE_TYPES;
  if (!allowedTypes.has(message.type)) throw new TypeError(`Widget message type is not allowed in the ${direction} direction`);
  if (!message.payload || typeof message.payload !== "object" || Array.isArray(message.payload)) {
    throw new TypeError("Widget message payload must be a plain data object");
  }
  PAYLOAD_VALIDATORS[message.type](message.payload);
  const messageBytes = textEncoder.encode(JSON.stringify(message)).byteLength;
  if (messageBytes > MAX_WIDGET_MESSAGE_BYTES) throw new TypeError("Widget message exceeds 64 KiB of UTF-8 serialized bytes");
  return message;
}
