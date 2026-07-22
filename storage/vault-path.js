// Cross-platform vault path normalization (Phase 1, plan §7.2–§7.3).
//
// One shared normalizer must produce the same logical result on Windows, macOS,
// Linux, Android, iOS, and the browser. Unsafe paths produce PathError
// diagnostics rather than escaping the vault root.

import { PathError } from "./vault-errors.js";
import { shortId } from "./content-hash.js";

export const MAX_COMPONENT_BYTES = 255;
export const MAX_PATH_BYTES = 4096;

// Characters forbidden in a single component on some platform. `/` and `\` are
// separators and are handled by splitting; control chars (incl. null) are banned.
const FORBIDDEN_RE = /[<>:"|?*\u0000-\u001f\u007f]/g;
const SCHEME_RE = /^[a-z][a-z0-9+.-]*:/i;
// Windows device names, bare or followed by an extension (CON, CON.txt, ...).
const DEVICE_RE = /^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\.|$)/i;

export function byteLength(str) {
  return new TextEncoder().encode(String(str)).length;
}

// Case-fold + NFC key used to detect collisions even on case-sensitive hosts.
// Unicode default case folding (not locale-sensitive lower-casing). NFKC folds
// compatibility characters and the explicit mappings cover the notable full
// folds that JavaScript's toLowerCase does not expand (for example sharp s).
const FULL_CASE_FOLDS = new Map([
  ["ß", "ss"], ["ẞ", "ss"], ["ς", "σ"], ["ŉ", "ʼn"], ["İ", "i\u0307"],
  ["ǰ", "j\u030c"], ["ΐ", "ι\u0308\u0301"], ["ΰ", "υ\u0308\u0301"], ["և", "եւ"],
  ["ẖ", "h\u0331"], ["ẗ", "t\u0308"], ["ẘ", "w\u030a"], ["ẙ", "y\u030a"],
  ["ẚ", "aʾ"], ["ỉ", "i\u0309"], ["ỏ", "o\u0309"], ["ử", "ư\u0309"],
]);

export function unicodeCaseFold(value) {
  return [...String(value).normalize("NFKC").toLowerCase()]
    .map((char) => FULL_CASE_FOLDS.get(char) || char).join("")
    .replace(/ς/g, "σ")
    .normalize("NFC");
}

export function caseFoldKey(path) {
  return unicodeCaseFold(String(path).replace(/\\/g, "/"));
}

function trimTrailing(segment) {
  // Windows forbids trailing spaces and periods; trim both.
  return segment.replace(/[ .]+$/, "");
}

function isDeviceName(segment) {
  return DEVICE_RE.test(segment);
}

// Human-readable slug from a title (plan §7.2). Lowercase, punctuation folded to
// dashes. Never empty; falls back to "untitled".
export function slugify(title) {
  let slug = String(title ?? "").normalize("NFC").toLowerCase();
  slug = slug.replace(/[\/\\<>:"|?*\u0000-\u001f\u007f]/g, " ");
  slug = slug.replace(/[^a-z0-9]+/g, "-");
  slug = slug.replace(/-+/g, "-").replace(/^-+|-+$/g, "");
  slug = trimTrailing(slug);
  if (!slug) slug = "untitled";
  // Bound the slug so the whole component stays under the byte limit.
  while (byteLength(slug) > 120) slug = slug.slice(0, slug.length - 1);
  return slug;
}

// Sanitize a single existing component, preserving case. Replaces forbidden
// characters with "-", NFC-normalizes, and trims trailing spaces/periods.
function sanitizeComponent(segment) {
  let out = String(segment).normalize("NFC");
  out = out.replace(FORBIDDEN_RE, "-");
  out = out.replace(/[/\\]/g, "-");
  out = trimTrailing(out);
  return out;
}

function assertComponentBounds(component) {
  if (component === "") {
    throw new PathError("Path component is empty", { code: "PATH_EMPTY_COMPONENT" });
  }
  if (byteLength(component) > MAX_COMPONENT_BYTES) {
    throw new PathError(`Path component exceeds ${MAX_COMPONENT_BYTES} bytes`, { code: "PATH_COMPONENT_TOO_LONG" });
  }
}

// Normalize a vault-relative path for generation. Rejects absolute paths, URL
// schemes, and `.`/`..` components; sanitizes each component; bounds byte length.
export function normalizePath(input) {
  const raw = String(input ?? "");
  if (SCHEME_RE.test(raw)) {
    throw new PathError(`Path has a URL scheme: ${raw}`, { code: "PATH_SCHEME" });
  }
  const slashed = raw.replace(/\\/g, "/");
  if (slashed.startsWith("/")) {
    throw new PathError(`Path is absolute: ${raw}`, { code: "PATH_ABSOLUTE" });
  }
  const parts = slashed.split("/").filter((p) => p !== "");
  if (parts.length === 0) {
    throw new PathError("Path is empty", { code: "PATH_EMPTY" });
  }
  const normalized = [];
  for (const part of parts) {
    if (part === "." || part === "..") {
      throw new PathError(`Path contains a traversal component "${part}"`, { code: "PATH_TRAVERSAL" });
    }
    let component = sanitizeComponent(part);
    if (isDeviceName(component)) component = `_${component}`;
    assertComponentBounds(component);
    normalized.push(component);
  }
  const joined = normalized.join("/");
  if (byteLength(joined) > MAX_PATH_BYTES) {
    throw new PathError(`Path exceeds ${MAX_PATH_BYTES} bytes`, { code: "PATH_TOO_LONG" });
  }
  return joined;
}

// Strict validation for a path that is already expected to be clean (read from a
// canvas file node or the vault). Does not silently rewrite: any unsafe input
// throws so callers can surface a diagnostic. Use before adapter calls.
export function assertSafePath(path) {
  const raw = String(path ?? "");
  if (raw.includes("\\")) throw new PathError(`Path contains a backslash: ${raw}`, { code: "PATH_FORBIDDEN_CHAR" });
  if (SCHEME_RE.test(raw)) throw new PathError(`Path has a URL scheme: ${raw}`, { code: "PATH_SCHEME" });
  const slashed = raw;
  if (slashed.startsWith("/")) throw new PathError(`Path is absolute: ${raw}`, { code: "PATH_ABSOLUTE" });
  if (slashed === "" || slashed.endsWith("/") || slashed.includes("//")) throw new PathError(`Path has an empty component: ${raw}`, { code: "PATH_EMPTY_COMPONENT" });
  const parts = slashed.split("/");
  if (parts.length === 0) throw new PathError("Path is empty", { code: "PATH_EMPTY" });
  for (const part of parts) {
    if (part === "." || part === "..") throw new PathError(`Path contains "${part}"`, { code: "PATH_TRAVERSAL" });
    if (part !== part.normalize("NFC")) throw new PathError(`Path component is not NFC-normalized: ${part}`, { code: "PATH_NOT_NORMALIZED" });
    FORBIDDEN_RE.lastIndex = 0;
    if (FORBIDDEN_RE.test(part)) { FORBIDDEN_RE.lastIndex = 0; throw new PathError(`Path component has forbidden characters: ${part}`, { code: "PATH_FORBIDDEN_CHAR" }); }
    FORBIDDEN_RE.lastIndex = 0;
    if (/[\\]/.test(part)) throw new PathError(`Path component has a backslash: ${part}`, { code: "PATH_FORBIDDEN_CHAR" });
    if (trimTrailing(part) !== part) throw new PathError(`Path component has trailing space/period: ${part}`, { code: "PATH_TRAILING" });
    if (isDeviceName(part)) throw new PathError(`Path component is a reserved device name: ${part}`, { code: "PATH_DEVICE_NAME" });
    assertComponentBounds(part);
  }
  const joined = parts.join("/");
  if (byteLength(joined) > MAX_PATH_BYTES) throw new PathError(`Path exceeds ${MAX_PATH_BYTES} bytes`, { code: "PATH_TOO_LONG" });
  return joined;
}

// Detect a case-folded collision between two normalized paths.
export function samePathFold(a, b) {
  return caseFoldKey(a) === caseFoldKey(b);
}

// Canonical entity path (plan §7.2): <dir>/<readable-slug>--<stable-short-id>.<ext>.
// The full stable id stays in frontmatter; the suffix only keeps names readable.
export function entityPath(dir, title, id, ext = "md") {
  const base = normalizePath(dir);
  const slug = slugify(title);
  const suffix = shortId(id);
  const component = `${slug}--${suffix}.${ext}`;
  assertComponentBounds(component);
  return `${base}/${component}`;
}

// Canonical declarative component-card path. Cards are Markdown data files and
// use the same readable-slug plus stable-id suffix convention as life entities.
export function componentCardPath(title, id) {
  return entityPath("cards", title, id, "md");
}
