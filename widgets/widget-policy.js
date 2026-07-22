export const MAX_WIDGET_SOURCE_BYTES = 128 * 1024;
export const MAX_WIDGET_STATIC_ELEMENTS = 500;
export const MAX_WIDGET_SCRIPT_BYTES = 64 * 1024;
export const MAX_WIDGET_STYLE_BYTES = 64 * 1024;
export const MAX_ACTIVE_WIDGETS = 6;
export const WIDGET_MESSAGES_PER_SECOND = 30;
export const WIDGET_MESSAGE_BURST = 60;

const FORBIDDEN_ELEMENTS = new Set([
  "a",
  "area",
  "base",
  "embed",
  "form",
  "frame",
  "frameset",
  "iframe",
  "noembed",
  "noframes",
  "noscript",
  "object",
  "plaintext",
  "xmp",
]);
const RAW_TEXT_ELEMENTS = new Set(["script", "style", "textarea", "title"]);
const RESOURCE_ATTRIBUTES = Object.freeze({
  audio: ["src"],
  feimage: ["href", "xlink:href"],
  image: ["href", "xlink:href"],
  img: ["src", "srcset"],
  input: ["src"],
  source: ["src", "srcset"],
  script: ["href", "xlink:href"],
  track: ["src"],
  use: ["href", "xlink:href"],
  video: ["poster", "src"],
});
const NETWORK_API_PATTERNS = Object.freeze([
  /\bfetch\s*\(/i,
  /\bnew\s+XMLHttpRequest\b/i,
  /\bnew\s+WebSocket\b/i,
  /\bnew\s+EventSource\b/i,
  /\bnew\s+(?:Shared)?Worker\b/i,
  /\bnavigator\s*\.\s*sendBeacon\s*\(/i,
  /\bnavigator\s*\.\s*serviceWorker\b/i,
  /\bimportScripts\s*\(/i,
]);
const JAVASCRIPT_ANIMATION_API_PATTERNS = Object.freeze([
  /\brequestAnimationFrame\b/,
  /\banimate\b/,
  /\bAnimation\b/,
  /\bKeyframeEffect\b/,
]);
const MODULE_RESOURCE_PATTERNS = Object.freeze([
  /\bimport\s*(?:\(|[\s{*])/i,
  /\bexport\s+(?:[^;]*?\s+from\s*)?["']/i,
]);
const textEncoder = new TextEncoder();

function byteLength(value) {
  return textEncoder.encode(value).byteLength;
}

function findTagEnd(source, start) {
  let quote = "";
  for (let index = start; index < source.length; index += 1) {
    const character = source[index];
    if (quote) {
      if (character === quote) quote = "";
      continue;
    }
    if (character === "\"" || character === "'") quote = character;
    else if (character === ">") return index;
  }
  throw new TypeError("Widget source contains an unterminated tag");
}

function parseAttributes(tagSource, nameEnd) {
  const attributes = new Map();
  let index = nameEnd;
  const end = tagSource.length - 1;
  while (index < end) {
    while (index < end && /\s|\//.test(tagSource[index])) index += 1;
    if (index >= end) break;
    const nameStart = index;
    while (index < end && !/[\s=/>]/.test(tagSource[index])) index += 1;
    if (index === nameStart) throw new TypeError("Widget source contains a malformed attribute");
    const name = tagSource.slice(nameStart, index).toLowerCase();
    while (index < end && /\s/.test(tagSource[index])) index += 1;
    let value = "";
    if (tagSource[index] === "=") {
      index += 1;
      while (index < end && /\s/.test(tagSource[index])) index += 1;
      const quote = tagSource[index];
      if (quote === "\"" || quote === "'") {
        index += 1;
        const valueStart = index;
        while (index < end && tagSource[index] !== quote) index += 1;
        if (index >= end) throw new TypeError("Widget source contains an unterminated attribute");
        value = tagSource.slice(valueStart, index);
        index += 1;
      } else {
        const valueStart = index;
        while (index < end && !/[\s>]/.test(tagSource[index])) index += 1;
        value = tagSource.slice(valueStart, index);
      }
    }
    if (attributes.has(name)) throw new TypeError(`Widget source contains duplicate ${name} attributes`);
    attributes.set(name, decodeHtmlReferences(value));
  }
  return attributes;
}

function decodeHtmlReferences(value) {
  const named = {
    amp: "&",
    apos: "'",
    colon: ":",
    emsp: "\u2003",
    ensp: "\u2002",
    gt: ">",
    lt: "<",
    nbsp: "\u00a0",
    newline: "\n",
    quot: "\"",
    tab: "\t",
    thinsp: "\u2009",
  };
  return value
    .replace(/&#x([\da-f]+);?/gi, (_, digits) => decodeCodePoint(Number.parseInt(digits, 16)))
    .replace(/&#(\d+);?/g, (_, digits) => decodeCodePoint(Number(digits)))
    .replace(/&([a-z]+);/gi, (reference, name) => named[name.toLowerCase()] ?? reference);
}

function decodeCodePoint(value) {
  if (!Number.isInteger(value) || value <= 0 || value > 0x10ffff || (value >= 0xd800 && value <= 0xdfff)) return "\ufffd";
  return String.fromCodePoint(value);
}

function normalizedScheme(value) {
  return value.trim().replace(/[\u0000-\u0020]/g, "").toLowerCase();
}

function assertAllowedEmbeddedUrl(value, label) {
  const normalized = normalizedScheme(value);
  if (!normalized || normalized.startsWith("data:") || normalized.startsWith("blob:") || normalized.startsWith("#")) return;
  throw new TypeError(`Widget source contains a forbidden external resource URL in ${label}`);
}

function assertAllowedSrcset(value, label) {
  let index = 0;
  while (index < value.length) {
    while (index < value.length && /[\t\n\f\r ,]/.test(value[index])) index += 1;
    if (index >= value.length) return;
    const urlStart = index;
    while (index < value.length && !/[\t\n\f\r ]/.test(value[index])) index += 1;
    let url = value.slice(urlStart, index);
    while (!url.toLowerCase().startsWith("data:") && url.endsWith(",")) url = url.slice(0, -1);
    assertAllowedEmbeddedUrl(url, label);
    while (index < value.length && /[\t\n\f\r ]/.test(value[index])) index += 1;
    let parentheses = 0;
    while (index < value.length) {
      if (value[index] === "(") parentheses += 1;
      else if (value[index] === ")") parentheses = Math.max(0, parentheses - 1);
      else if (value[index] === "," && parentheses === 0) {
        index += 1;
        break;
      }
      index += 1;
    }
  }
}

function assertAttributePolicy(tagName, attributes) {
  for (const [name, value] of attributes) {
    if (normalizedScheme(value).startsWith("javascript:")) {
      throw new TypeError(`Widget source contains a forbidden javascript URL in ${name}`);
    }
  }
  if (tagName === "meta" && attributes.get("http-equiv")?.trim().toLowerCase() === "refresh") {
    throw new TypeError("Widget source contains forbidden meta refresh");
  }
  if (tagName === "script" && ["src", "href", "xlink:href"].some((name) => attributes.has(name))) {
    throw new TypeError("Widget source contains a forbidden external script resource");
  }
  if (tagName === "link" && attributes.has("href")) {
    throw new TypeError("Widget source contains a forbidden external link resource");
  }
  for (const name of RESOURCE_ATTRIBUTES[tagName] || []) {
    if (!attributes.has(name)) continue;
    if (name === "srcset") assertAllowedSrcset(attributes.get(name), `${tagName}.${name}`);
    else assertAllowedEmbeddedUrl(attributes.get(name), `${tagName}.${name}`);
  }
  if (attributes.has("style")) assertCssPolicy(attributes.get("style"));
}

function assertCssPolicy(css) {
  const normalized = decodeCssEscapes(stripCssComments(css));
  if (/@import\b/i.test(normalized)) throw new TypeError("Widget source contains a forbidden CSS import resource");
  if (/@font-face\b/i.test(normalized)) throw new TypeError("Widget source contains a forbidden font resource");
  const urlPattern = /url\(\s*(?:(["'])(.*?)\1|([^\s)]*))\s*\)/gi;
  let match;
  while ((match = urlPattern.exec(normalized))) assertAllowedEmbeddedUrl(match[2] ?? match[3] ?? "", "CSS url()");
}

function stripCssComments(css) {
  if (/\/\*[\s\S]*$/.test(css.replace(/\/\*[\s\S]*?\*\//g, ""))) throw new TypeError("Widget source contains an unterminated CSS comment");
  return css.replace(/\/\*[\s\S]*?\*\//g, "");
}

function maskCssStringsAndComments(css) {
  const source = stripCssComments(css);
  let masked = "";
  for (let index = 0; index < source.length;) {
    const quote = source[index];
    if (quote !== "\"" && quote !== "'") {
      masked += quote;
      index += 1;
      continue;
    }
    masked += " ";
    index += 1;
    while (index < source.length) {
      if (source[index] === "\\") {
        index += 2;
      } else if (source[index] === quote) {
        index += 1;
        break;
      } else {
        if (/[\n\r\f]/.test(source[index])) masked += source[index];
        index += 1;
      }
    }
  }
  return masked;
}

function decodeCssEscapes(css) {
  return css
    .replace(/\\([\da-f]{1,6})[\t\n\f\r ]?/gi, (_, digits) => decodeCodePoint(Number.parseInt(digits, 16)))
    .replace(/\\\r\n|\\[\n\f\r]/g, "")
    .replace(/\\(.)/gs, "$1");
}

function assertNoAmbiguousRegexCommentDelimiters(script) {
  for (let index = 0; index < script.length; index += 1) {
    if (script[index] === "\\" && script[index + 1] === "/") {
      const following = script[index + 2];
      if (following === "/" || (following === "\\" && (script[index + 3] === "/" || script[index + 3] === "*"))) {
        throw new TypeError("Widget script contains an ambiguous regular expression comment delimiter");
      }
    }
    if (script[index] !== "[") continue;
    for (let classIndex = index + 1; classIndex < script.length && !/[\n\r\u2028\u2029]/.test(script[classIndex]); classIndex += 1) {
      if (script[classIndex] === "\\") {
        classIndex += 1;
        continue;
      }
      if (script[classIndex] === "]") break;
      if (script[classIndex] === "/" && (script[classIndex + 1] === "/" || script[classIndex + 1] === "*")) {
        throw new TypeError("Widget script contains an ambiguous regular expression comment delimiter");
      }
    }
  }
}

function assertScriptPolicy(script) {
  assertNoAmbiguousRegexCommentDelimiters(script);
  if (MODULE_RESOURCE_PATTERNS.some((pattern) => pattern.test(script))) {
    throw new TypeError("Widget source contains a forbidden module resource");
  }
  if (NETWORK_API_PATTERNS.some((pattern) => pattern.test(script))) {
    throw new TypeError("Widget source contains a forbidden network or worker resource API");
  }
}

function analyzeJavaScript(script) {
  const strings = [];

  const addString = (value) => {
    const token = `__STRING_${strings.length}__`;
    strings.push(value);
    return token;
  };

  const scanQuoted = (start, quote) => {
    let value = "";
    let index = start;
    while (index < script.length) {
      const character = script[index];
      if (character === "\\") {
        value += character;
        if (index + 1 < script.length) value += script[index + 1];
        index += 2;
      } else if (character === quote) {
        return { index: index + 1, value };
      } else {
        value += character;
        index += 1;
      }
    }
    return { index, value };
  };

  const canStartRegex = (code) => {
    const prefix = code.trimEnd();
    if (!prefix) return true;
    if (/[\[({=,:;!?&|+\-*%^~<>}]$/.test(prefix)) return true;
    if (!prefix.endsWith(")")) return false;
    let depth = 0;
    for (let index = prefix.length - 1; index >= 0; index -= 1) {
      if (prefix[index] === ")") depth += 1;
      else if (prefix[index] === "(") {
        depth -= 1;
        if (depth === 0) return /\b(?:for|if|while|with)$/.test(prefix.slice(0, index).trimEnd());
      }
    }
    return false;
  };

  const scanRegex = (start) => {
    let inCharacterClass = false;
    let index = start + 1;
    while (index < script.length) {
      const character = script[index];
      if (/[\n\r\u2028\u2029]/.test(character)) throw new TypeError("Widget script contains an ambiguous regular expression literal");
      if (character === "\\") {
        index += 2;
        continue;
      }
      if (character === "[") inCharacterClass = true;
      else if (character === "]") inCharacterClass = false;
      else if (character === "/" && !inCharacterClass) {
        index += 1;
        while (index < script.length && /[a-z]/i.test(script[index])) index += 1;
        return index;
      }
      index += 1;
    }
    throw new TypeError("Widget script contains an unterminated regular expression literal");
  };

  const scanCode = (start, stopAtExpressionEnd = false) => {
    let code = "";
    let braces = 0;
    let index = start;
    while (index < script.length) {
      const character = script[index];
      if (character === "/" && !script.startsWith("//", index) && !script.startsWith("/*", index) && canStartRegex(code)) {
        index = scanRegex(index);
        code += "__REGEX__";
        continue;
      }
      if (script.startsWith("//", index)) {
        const lineEnd = /[\n\r\u2028\u2029]/.exec(script.slice(index + 2));
        if (!lineEnd) return { code: `${code} `, index: script.length };
        index += lineEnd.index + 2;
        code += " ";
        continue;
      }
      if (script.startsWith("/*", index)) {
        const end = script.indexOf("*/", index + 2);
        code += " ";
        index = end < 0 ? script.length : end + 2;
        continue;
      }
      if (character === "\"" || character === "'") {
        const quoted = scanQuoted(index + 1, character);
        code += addString(quoted.value);
        index = quoted.index;
        continue;
      }
      if (character === "`") {
        const template = scanTemplate(index + 1);
        code += template.code;
        index = template.index;
        continue;
      }
      if (stopAtExpressionEnd && character === "}") {
        if (braces === 0) return { code, index: index + 1 };
        braces -= 1;
      } else if (stopAtExpressionEnd && character === "{") {
        braces += 1;
      }
      code += character;
      index += 1;
    }
    return { code, index };
  };

  const scanTemplate = (start) => {
    let code = "";
    let literal = "";
    let index = start;
    while (index < script.length) {
      const character = script[index];
      if (character === "\\") {
        literal += character;
        if (index + 1 < script.length) literal += script[index + 1];
        index += 2;
      } else if (character === "`") {
        if (literal) code += addString(literal);
        return { code, index: index + 1 };
      } else if (character === "$" && script[index + 1] === "{") {
        if (literal) {
          code += addString(literal);
          literal = "";
        }
        const expression = scanCode(index + 2, true);
        code += ` ${expression.code} `;
        index = expression.index;
      } else {
        literal += character;
        index += 1;
      }
    }
    if (literal) code += addString(literal);
    return { code, index };
  };

  const result = scanCode(0);
  return { code: result.code, strings };
}

function hasReducedMotionHandling(styles, scripts) {
  const normalizedStyles = decodeCssEscapes(maskCssStringsAndComments(styles));
  if (/@media[^{]*\(\s*prefers-reduced-motion\s*:\s*reduce\s*\)/i.test(normalizedStyles)) return true;
  const slashFreeBlocks = [];
  let block = [];
  for (const line of scripts.split(/\r\n|[\n\r\u2028\u2029]/)) {
    if (line.includes("/")) {
      if (block.length) slashFreeBlocks.push(block.join("\n"));
      block = [];
    } else {
      block.push(line);
    }
  }
  if (block.length) slashFreeBlocks.push(block.join("\n"));
  return slashFreeBlocks.some(hasJavaScriptReducedMotionEvidence);
}

function truncateClassicScriptComment(line) {
  let quote = "";
  for (let index = 0; index < line.length; index += 1) {
    const character = line[index];
    if (quote) {
      if (character === "\\") index += 1;
      else if (character === quote) quote = "";
      continue;
    }
    if (character === "\"" || character === "'" || character === "`") {
      quote = character;
      continue;
    }
    if (line.startsWith("<!--", index) || line.startsWith("-->", index)) return line.slice(0, index);
  }
  return line;
}

function hasJavaScriptReducedMotionEvidence(script) {
  const evidenceSource = script
    .split("\n")
    .map((line) => (/^[\t ]*#!/.test(line) ? "" : truncateClassicScriptComment(line)))
    .join("\n");
  const { code, strings } = analyzeJavaScript(evidenceSource);
  const matchMedia = /\bmatchMedia\s*\(\s*__STRING_(\d+)__/gi;
  let match;
  while ((match = matchMedia.exec(code))) {
    if (/\(\s*prefers-reduced-motion\s*:\s*reduce\s*\)/i.test(strings[Number(match[1])] || "")) return true;
  }
  return hasProtocolReducedMotionEvidence(code, strings);
}

function hasProtocolReducedMotionEvidence(code, strings) {
  const receiverSource = String.raw`[A-Za-z_$][\w$]*(?:\s*\.\s*[A-Za-z_$][\w$]*)*`;
  const destructured = [];
  const destructuring = new RegExp(String.raw`\{\s*reducedMotion(?:\s*:\s*([A-Za-z_$][\w$]*))?\s*\}\s*=\s*(${receiverSource})\s*\.\s*payload\b`, "g");
  let destructuringMatch;
  while ((destructuringMatch = destructuring.exec(code))) {
    destructured.push({
      name: destructuringMatch[1] || "reducedMotion",
      receiver: normalizeReceiver(destructuringMatch[2]),
    });
  }
  for (let index = 0; index < strings.length; index += 1) {
    if (strings[index] !== "orbit.widget.preferences.v1") continue;
    const token = `__STRING_${index}__`;
    const escapedToken = token.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const ifCondition = /\bif\s*\(([^)]{0,600})\)/g;
    let conditional;
    while ((conditional = ifCondition.exec(code))) {
      const condition = conditional[1];
      const receiver = typeComparisonReceiver(condition, receiverSource, escapedToken);
      if (!receiver) continue;
      const payloadAccess = new RegExp(`${receiverExpression(receiver)}\\s*\\.\\s*payload\\s*\\.\\s*reducedMotion\\b`);
      if (payloadAccess.test(condition)) return true;
      if (destructured.some((entry) => entry.receiver === receiver && new RegExp(`\\b${entry.name}\\b`).test(condition))) return true;
    }
    const switchStatement = new RegExp(String.raw`\bswitch\s*\(\s*(${receiverSource})\s*\.\s*type\s*\)`, "g");
    let switchMatch;
    while ((switchMatch = switchStatement.exec(code))) {
      const receiver = normalizeReceiver(switchMatch[1]);
      const caseBody = exactSwitchCaseBody(code, switchMatch.index + switchMatch[0].length, token);
      if (caseBody && new RegExp(`${receiverExpression(receiver)}\\s*\\.\\s*payload\\s*\\.\\s*reducedMotion\\b`).test(caseBody)) return true;
    }
  }
  return false;
}

function exactSwitchCaseBody(code, switchHeaderEnd, token) {
  const open = code.indexOf("{", switchHeaderEnd);
  if (open < 0) return "";
  let switchDepth = 1;
  let close = open + 1;
  for (; close < code.length && switchDepth > 0; close += 1) {
    if (code[close] === "{") switchDepth += 1;
    else if (code[close] === "}") switchDepth -= 1;
  }
  if (switchDepth !== 0) return "";
  const body = code.slice(open + 1, close - 1);
  let caseStart = -1;
  let depth = 0;
  for (let index = 0; index < body.length; index += 1) {
    if (body[index] === "{") {
      depth += 1;
      continue;
    }
    if (body[index] === "}") {
      depth = Math.max(0, depth - 1);
      continue;
    }
    if (depth !== 0) continue;
    const isCase = body.startsWith("case", index) && !/[\w$]/.test(body[index - 1] || "") && !/[\w$]/.test(body[index + 4] || "");
    const isDefault = body.startsWith("default", index) && !/[\w$]/.test(body[index - 1] || "") && !/[\w$]/.test(body[index + 7] || "");
    if (!isCase && !isDefault) continue;
    if (caseStart >= 0) return body.slice(caseStart, index);
    const colon = body.indexOf(":", index + (isCase ? 4 : 7));
    if (colon < 0) return "";
    if (isCase && body.slice(index + 4, colon).trim() === token) caseStart = colon + 1;
    index = colon;
  }
  return caseStart >= 0 ? body.slice(caseStart) : "";
}

function typeComparisonReceiver(condition, receiverSource, escapedToken) {
  const direct = new RegExp(`(${receiverSource})\\s*\\.\\s*type\\s*={2,3}\\s*${escapedToken}`).exec(condition);
  if (direct) return normalizeReceiver(direct[1]);
  const reversed = new RegExp(`${escapedToken}\\s*={2,3}\\s*(${receiverSource})\\s*\\.\\s*type`).exec(condition);
  return reversed ? normalizeReceiver(reversed[1]) : "";
}

function normalizeReceiver(receiver) {
  return receiver.replace(/\s/g, "");
}

function receiverExpression(receiver) {
  return receiver.split(".").map((part) => part.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")).join("\\s*\\.\\s*");
}

function scanSource(source) {
  const rawText = { script: [], style: [], textarea: [], title: [] };
  let staticElementCount = 0;
  let templateDepth = 0;
  let cursor = 0;
  while (cursor < source.length) {
    const tagStart = source.indexOf("<", cursor);
    if (tagStart < 0) break;
    if (source.startsWith("<!--", tagStart)) {
      const commentEnd = source.indexOf("-->", tagStart + 4);
      if (commentEnd < 0) throw new TypeError("Widget source contains an unterminated comment");
      cursor = commentEnd + 3;
      continue;
    }
    if (source[tagStart + 1] === "!" || source[tagStart + 1] === "?") {
      cursor = findTagEnd(source, tagStart + 1) + 1;
      continue;
    }
    if (source[tagStart + 1] === "/") {
      const closeMatch = /^[A-Za-z][A-Za-z0-9:-]*/.exec(source.slice(tagStart + 2));
      const closeEnd = findTagEnd(source, tagStart + 2);
      if (closeMatch?.[0].toLowerCase() === "template") {
        if (templateDepth === 0) throw new TypeError("Widget source contains an unmatched </template>");
        templateDepth -= 1;
      }
      cursor = closeEnd + 1;
      continue;
    }
    const nameMatch = /^[A-Za-z][A-Za-z0-9:-]*/.exec(source.slice(tagStart + 1));
    if (!nameMatch) {
      cursor = tagStart + 1;
      continue;
    }
    const tagName = nameMatch[0].toLowerCase();
    const tagEnd = findTagEnd(source, tagStart + 1 + nameMatch[0].length);
    const attributes = parseAttributes(source.slice(tagStart, tagEnd + 1), 1 + nameMatch[0].length);
    staticElementCount += 1;
    if (staticElementCount > MAX_WIDGET_STATIC_ELEMENTS) {
      throw new TypeError(`Widget source exceeds ${MAX_WIDGET_STATIC_ELEMENTS} static elements`);
    }
    if (FORBIDDEN_ELEMENTS.has(tagName)) throw new TypeError(`Widget source contains forbidden parser-state absorber <${tagName}>`);
    assertAttributePolicy(tagName, attributes);
    cursor = tagEnd + 1;
    if (tagName === "template") templateDepth += 1;
    if (!RAW_TEXT_ELEMENTS.has(tagName)) continue;
    const closePattern = new RegExp(`</${tagName}[\\t\\n\\f\\r ]*>`, "ig");
    closePattern.lastIndex = cursor;
    const closeMatch = closePattern.exec(source);
    if (!closeMatch) throw new TypeError(`Widget source contains an unterminated <${tagName}> element`);
    rawText[tagName].push(source.slice(cursor, closeMatch.index));
    cursor = closeMatch.index + closeMatch[0].length;
  }
  if (templateDepth !== 0) throw new TypeError("Widget source contains an unclosed <template> element");
  return { rawText, staticElementCount };
}

export function validateWidgetSource(source) {
  if (typeof source !== "string") throw new TypeError("Widget source must be a string");
  if (byteLength(source) > MAX_WIDGET_SOURCE_BYTES) throw new TypeError("Widget source exceeds 128 KiB of UTF-8 bytes");
  const { rawText, staticElementCount } = scanSource(source);
  const title = decodeHtmlReferences(rawText.title[0] || "").trim();
  if (!title) throw new TypeError("Widget source requires a non-empty title");
  const scripts = rawText.script.join("");
  const styles = rawText.style.join("");
  const scriptBytes = byteLength(scripts);
  const styleBytes = byteLength(styles);
  if (scriptBytes > MAX_WIDGET_SCRIPT_BYTES) throw new TypeError("Widget source script text exceeds 64 KiB of UTF-8 bytes");
  if (styleBytes > MAX_WIDGET_STYLE_BYTES) throw new TypeError("Widget source style text exceeds 64 KiB of UTF-8 bytes");
  assertScriptPolicy(scripts);
  assertCssPolicy(styles);
  const normalizedStyles = decodeCssEscapes(stripCssComments(styles));
  const { code: executableScript } = analyzeJavaScript(scripts);
  const hasRawAnimationApi = JAVASCRIPT_ANIMATION_API_PATTERNS.some((pattern) => pattern.test(scripts));
  const hasAnimation = hasRawAnimationApi
    || /\banimation(?:-name)?\s*:/i.test(normalizedStyles)
    || /\brequestAnimationFrame\s*\(/i.test(executableScript);
  if (hasAnimation && !hasReducedMotionHandling(styles, scripts)) throw new TypeError("Widget animation requires reduced motion handling");
  return { title, source, staticElementCount, scriptBytes, styleBytes };
}
