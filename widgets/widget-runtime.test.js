import { test } from "node:test";
import assert from "node:assert/strict";

import {
  MAX_ACTIVE_WIDGETS,
  MAX_WIDGET_SCRIPT_BYTES,
  MAX_WIDGET_SOURCE_BYTES,
  MAX_WIDGET_STATIC_ELEMENTS,
  MAX_WIDGET_STYLE_BYTES,
  WIDGET_MESSAGE_BURST,
  WIDGET_MESSAGES_PER_SECOND,
  validateWidgetSource,
} from "./widget-policy.js";
import { WIDGET_CSP, buildWidgetDocument } from "./widget-envelope.js";
import {
  HOST_TO_WIDGET,
  MAX_WIDGET_MESSAGE_BYTES,
  WIDGET_PROTOCOL_VERSION,
  WIDGET_TO_HOST,
  validateWidgetMessage,
} from "./widget-protocol.js";

const titled = (body = "<p>Ready</p>") => `<title>Test widget</title>${body}`;
const byteLength = (value) => new TextEncoder().encode(value).byteLength;

function sourceAtBytes(limit) {
  const prefix = titled("<!--");
  const suffix = "-->";
  return `${prefix}${"x".repeat(limit - byteLength(prefix) - byteLength(suffix))}${suffix}`;
}

function messageAtBytes(direction, makeMessage, limit) {
  let low = 0;
  let high = limit;
  while (low <= high) {
    const middle = Math.floor((low + high) / 2);
    const message = makeMessage("x".repeat(middle));
    const bytes = byteLength(JSON.stringify(message));
    if (bytes === limit) return message;
    if (bytes < limit) low = middle + 1;
    else high = middle - 1;
  }
  throw new Error(`Could not construct ${direction} message at ${limit} bytes`);
}

const validHostMessages = [
  {
    type: "orbit.widget.theme.v1",
    version: 1,
    payload: {
      tokens: {
        surface: "#24150c",
        surfaceRaised: "#2e1a0e",
        content: "#f1e7d4",
        contentMuted: "#cfc1aa",
        paper: "#d7c48f",
        ink: "#2a2015",
        primary: "#f2c14e",
        focus: "#5ed0bd",
        danger: "#a65745",
        radius: "4px",
        fontBody: "Work Sans, system-ui, sans-serif",
        fontMono: "JetBrains Mono, ui-monospace, monospace",
      },
    },
  },
  {
    type: "orbit.widget.preferences.v1",
    version: 1,
    payload: { reducedMotion: true, reducedTransparency: false, contrast: "more" },
  },
  { type: "orbit.widget.visibility.v1", version: 1, payload: { visible: true } },
  { type: "orbit.widget.pause.v1", version: 1, payload: { reason: "offscreen" } },
];

const validWidgetMessages = [
  { type: "orbit.widget.ready.v1", version: 1, payload: {} },
  { type: "orbit.widget.status.v1", version: 1, payload: { message: "Rendering" } },
  { type: "orbit.widget.resize.v1", version: 1, payload: { width: 640, height: 360 } },
  { type: "orbit.widget.heartbeat.v1", version: 1, payload: {} },
  { type: "orbit.widget.diagnostic.v1", version: 1, payload: { level: "warning", message: "WebGL fallback active" } },
];

test("exports the approved widget limits exactly", () => {
  assert.equal(MAX_WIDGET_SOURCE_BYTES, 128 * 1024);
  assert.equal(MAX_WIDGET_STATIC_ELEMENTS, 500);
  assert.equal(MAX_WIDGET_SCRIPT_BYTES, 64 * 1024);
  assert.equal(MAX_WIDGET_STYLE_BYTES, 64 * 1024);
  assert.equal(MAX_WIDGET_MESSAGE_BYTES, 64 * 1024);
  assert.equal(MAX_ACTIVE_WIDGETS, 6);
  assert.equal(WIDGET_MESSAGES_PER_SECOND, 30);
  assert.equal(WIDGET_MESSAGE_BURST, 60);
  assert.equal(WIDGET_PROTOCOL_VERSION, 1);
});

test("accepts native custom elements, declarative Shadow DOM, and allowed embedded resources", () => {
  const source = `<!doctype html>
    <title>Shadow dial &amp; controls</title>
    <style>
      orbit-dial { display: block; background-image: url(data:image/svg+xml,%3Csvg%3E%3C/svg%3E); }
      @media (prefers-reduced-motion: reduce) { orbit-dial { animation: none; } }
    </style>
    <orbit-dial>
      <template shadowrootmode="open"><canvas width="320" height="180"></canvas></template>
      <img alt="" src="blob:https://example.invalid/asset">
      <video src="data:video/mp4;base64,AA=="></video>
    </orbit-dial>
    <script>
      customElements.define("orbit-dial", class extends HTMLElement {
        connectedCallback() { if (!this.shadowRoot) this.attachShadow({ mode: "closed" }); }
      });
    </script>`;
  const result = validateWidgetSource(source);
  assert.equal(result.title, "Shadow dial & controls");
  assert.equal(result.source, source);
  assert.equal(result.staticElementCount, 8);
  assert.ok(result.scriptBytes > 0);
  assert.ok(result.styleBytes > 0);
});

test("enforces UTF-8 source bytes at the exact boundary", () => {
  const exact = sourceAtBytes(MAX_WIDGET_SOURCE_BYTES);
  assert.equal(byteLength(exact), MAX_WIDGET_SOURCE_BYTES);
  assert.equal(validateWidgetSource(exact).source, exact);
  assert.throws(() => validateWidgetSource(`${exact}é`), /128 KiB|source.*bytes/i);
  assert.throws(() => validateWidgetSource(new String(titled())), /string/i);
});

test("enforces static element count at the exact boundary without counting raw script text", () => {
  const exact = `<title>Counted</title>${"<i></i>".repeat(MAX_WIDGET_STATIC_ELEMENTS - 2)}<script>const sample = "<b></b>";</script>`;
  assert.equal(validateWidgetSource(exact).staticElementCount, MAX_WIDGET_STATIC_ELEMENTS);
  assert.throws(() => validateWidgetSource(`${exact}<i></i>`), /500|element/i);
});

test("enforces aggregate script and style UTF-8 bytes at exact boundaries", () => {
  const exactScript = titled(`<script>${" ".repeat(MAX_WIDGET_SCRIPT_BYTES)}</script>`);
  const exactStyle = titled(`<style>${" ".repeat(MAX_WIDGET_STYLE_BYTES)}</style>`);
  assert.equal(validateWidgetSource(exactScript).scriptBytes, MAX_WIDGET_SCRIPT_BYTES);
  assert.equal(validateWidgetSource(exactStyle).styleBytes, MAX_WIDGET_STYLE_BYTES);
  assert.throws(() => validateWidgetSource(titled(`<script>${" ".repeat(MAX_WIDGET_SCRIPT_BYTES + 1)}</script>`)), /script.*64 KiB|script.*bytes/i);
  assert.throws(() => validateWidgetSource(titled(`<style>${" ".repeat(MAX_WIDGET_STYLE_BYTES + 1)}</style>`)), /style.*64 KiB|style.*bytes/i);
});

test("requires a non-empty document or widget title", () => {
  for (const source of ["<p>Untitled</p>", "<title>  </title><p>Untitled</p>", "<title></title>"]) {
    assert.throws(() => validateWidgetSource(source), /title/i);
  }
});

test("rejects every forbidden element and meta refresh case-insensitively", () => {
  const cases = [
    "<base href='data:text/plain,ok'>",
    "<a href='#local'>link</a>",
    "<area href='#local'>",
    "<iframe srcdoc='<p>nested</p>'></iframe>",
    "<frame>",
    "<object></object>",
    "<embed>",
    "<form></form>",
    "<META HTTP-EQUIV='Refresh' content='0;url=data:text/html,bye'>",
  ];
  for (const fragment of cases) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /forbidden|refresh/i, fragment);
  }
});

test("rejects external and executable URL attributes across resource channels", () => {
  const cases = [
    "<script src='https://example.invalid/a.js'></script>",
    "<link rel='stylesheet' href='/theme.css'>",
    "<link rel='modulepreload' href='module.js'>",
    "<img src='https://example.invalid/a.png'>",
    "<img srcset='data:image/png;base64,AA== 1x, /remote.png 2x'>",
    "<picture><source srcset='//example.invalid/a.webp'></picture>",
    "<video src='/movie.mp4'></video>",
    "<video poster='poster.png'></video>",
    "<audio src='movie.mp3'></audio>",
    "<track src='captions.vtt'>",
    "<input type='image' src='/submit.png'>",
    "<svg><image href='/image.svg'></image></svg>",
    "<svg><image xlink:href='/image.svg'></image></svg>",
    "<svg><use href='/sprite.svg#icon'></use></svg>",
    "<button formaction='javascript:alert(1)'>Run</button>",
  ];
  for (const fragment of cases) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /URL|resource|javascript|forbidden/i, fragment);
  }
});

test("rejects CSS URL, import, and font channels except data/blob image resources", () => {
  const cases = [
    "<style>@import '/theme.css';</style>",
    "<style>.x { background: url(https://example.invalid/x.png); }</style>",
    "<p style=\"background:url('/x.png')\"></p>",
    "<style>@font-face { font-family: x; src: url(data:font/woff2;base64,AA==); }</style>",
  ];
  for (const fragment of cases) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /URL|resource|font|import/i, fragment);
  }
  assert.doesNotThrow(() => validateWidgetSource(titled("<style>.x{background:url(blob:https://example.invalid/id)}</style>")));
});

test("rejects module and runtime network/resource APIs", () => {
  const cases = [
    "<script type='module'>import x from './x.js';</script>",
    "<script type='module'>export { x } from './x.js';</script>",
    "<script>import('./x.js')</script>",
    "<script>fetch('/secret')</script>",
    "<script>new XMLHttpRequest()</script>",
    "<script>new WebSocket('wss://example.invalid')</script>",
    "<script>new EventSource('/events')</script>",
    "<script>new Worker('worker.js')</script>",
    "<script>new SharedWorker('worker.js')</script>",
    "<script>navigator.sendBeacon('/collect')</script>",
    "<script>navigator.serviceWorker.register('/sw.js')</script>",
    "<script>importScripts('/worker.js')</script>",
  ];
  for (const fragment of cases) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /module|network|resource|worker/i, fragment);
  }
});

test("requires reduced-motion handling when animation is declared or scheduled", () => {
  assert.throws(() => validateWidgetSource(titled("<style>.x{animation: spin 1s infinite}</style>")), /reduced motion/i);
  assert.throws(() => validateWidgetSource(titled("<script>requestAnimationFrame(draw)</script>")), /reduced motion/i);
  assert.doesNotThrow(() => validateWidgetSource(titled("<style>.x{animation:spin 1s}@media (prefers-reduced-motion: reduce){.x{animation:none}}</style>")));
  assert.doesNotThrow(() => validateWidgetSource(titled("<script>const reduce = matchMedia('(prefers-reduced-motion: reduce)'); requestAnimationFrame(draw)</script>")));
});

test("builds trusted CSP, bootstrap, generated source, and diagnostics in strict order", () => {
  const source = titled("<orbit-safe>generated-marker</orbit-safe>");
  const documentSource = buildWidgetDocument(source, { bootstrapSource: "globalThis.bootstrapMarker = true;" });
  const cspIndex = documentSource.indexOf("data-balaur-csp");
  const bootstrapIndex = documentSource.indexOf("data-balaur-bootstrap");
  const generatedIndex = documentSource.indexOf("generated-marker");
  const diagnosticIndex = documentSource.indexOf("data-balaur-diagnostic-boundary");
  assert.ok(documentSource.startsWith("<!doctype html>"));
  assert.ok(cspIndex > 0 && cspIndex < bootstrapIndex);
  assert.ok(bootstrapIndex < generatedIndex);
  assert.ok(generatedIndex < diagnosticIndex);
  assert.ok(documentSource.includes(`content="${WIDGET_CSP}"`));
  assert.equal(WIDGET_CSP, "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data: blob:; media-src data: blob:; font-src 'none'; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'");
  assert.equal(documentSource.includes("'unsafe-eval'"), false);
});

test("the envelope validates generated source and rejects script-closing bootstrap text", () => {
  assert.throws(() => buildWidgetDocument("<p>Untitled</p>", { bootstrapSource: "" }), /title/i);
  assert.throws(() => buildWidgetDocument(titled(), { bootstrapSource: "</script><p>escape</p>" }), /bootstrap/i);
  assert.throws(() => buildWidgetDocument(titled(), {}), /bootstrap/i);
});

test("accepts every closed protocol message in only its allowed direction and clones it", () => {
  for (const [direction, messages] of [[HOST_TO_WIDGET, validHostMessages], [WIDGET_TO_HOST, validWidgetMessages]]) {
    for (const message of messages) {
      const normalized = validateWidgetMessage(direction, message);
      assert.deepEqual(normalized, message);
      assert.notEqual(normalized, message);
      assert.notEqual(normalized.payload, message.payload);
      const otherDirection = direction === HOST_TO_WIDGET ? WIDGET_TO_HOST : HOST_TO_WIDGET;
      assert.throws(() => validateWidgetMessage(otherDirection, message), /direction|message type/i);
    }
  }
});

test("rejects unknown directions, wrapper fields, message types, and versions", () => {
  const ready = validWidgetMessages[0];
  assert.throws(() => validateWidgetMessage("sideways", ready), /direction/i);
  assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, { ...ready, extra: true }), /unknown|field/i);
  assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, { ...ready, type: "orbit.widget.mutate.v1" }), /message type/i);
  assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, { ...ready, version: 2 }), /version/i);
  assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, { ...ready, version: "1" }), /version/i);
});

test("enforces closed payload schemas and bounded resize/text values", () => {
  assert.throws(() => validateWidgetMessage(HOST_TO_WIDGET, { ...validHostMessages[2], payload: { visible: true, path: "tasks/private.md" } }), /unknown|payload/i);
  assert.throws(() => validateWidgetMessage(HOST_TO_WIDGET, { ...validHostMessages[1], payload: { ...validHostMessages[1].payload, reducedMotion: "yes" } }), /reducedMotion/i);
  assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, { ...validWidgetMessages[2], payload: { width: 0, height: 100 } }), /width/i);
  assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, { ...validWidgetMessages[2], payload: { width: 100, height: 10_001 } }), /height/i);
  assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, { ...validWidgetMessages[4], payload: { level: "fatal", message: "boom" } }), /level/i);
  assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, { ...validWidgetMessages[4], payload: { level: "error", message: "x".repeat(4097) } }), /message/i);
});

test("rejects prototype-bearing, accessor, sparse, cyclic, and toJSON-bearing data without invoking it", () => {
  let invoked = false;
  const payloads = [
    Object.assign(Object.create({ inherited: true }), {}),
    new (class Payload {})(),
    Object.defineProperty({}, "message", { enumerable: true, get() { invoked = true; return "no"; } }),
    Object.assign([], { 1: "hole" }),
  ];
  for (const payload of payloads) {
    assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, { type: "orbit.widget.status.v1", version: 1, payload }), /plain JSON data|plain data/i);
  }
  const cyclic = {};
  cyclic.self = cyclic;
  assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, { type: "orbit.widget.status.v1", version: 1, payload: cyclic }), /plain JSON data|cyclic/i);
  const value = { type: "orbit.widget.ready.v1", version: 1, payload: {} };
  value.payload.toJSON = () => { invoked = true; return {}; };
  assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, value), /plain JSON data|plain data/i);
  assert.equal(invoked, false);
});

test("measures closed protocol messages at the exact UTF-8 byte boundary", () => {
  const exact = messageAtBytes(WIDGET_TO_HOST, (message) => ({
    type: "orbit.widget.status.v1",
    version: 1,
    payload: { message },
  }), MAX_WIDGET_MESSAGE_BYTES);
  assert.equal(byteLength(JSON.stringify(exact)), MAX_WIDGET_MESSAGE_BYTES);
  assert.deepEqual(validateWidgetMessage(WIDGET_TO_HOST, exact), exact);
  exact.payload.message += "é";
  assert.throws(() => validateWidgetMessage(WIDGET_TO_HOST, exact), /64 KiB|message.*bytes/i);
});

test("uses HTML raw-text parsing rules for self-closing syntax and exact end-tag names", () => {
  const source = "<title/>Raw title</title><style/>.x{color:red}</style><script/>const marker = \"</scripture>\";</script>";
  const result = validateWidgetSource(source);
  assert.equal(result.title, "Raw title");
  assert.equal(result.styleBytes, byteLength(".x{color:red}"));
  assert.equal(result.scriptBytes, byteLength("const marker = \"</scripture>\";"));
  assert.equal(result.staticElementCount, 3);
});

test("decodes character references before title and URL policy checks", () => {
  assert.throws(() => validateWidgetSource("<title>&#x20;&#32;&#x09;</title>"), /title/i);
  assert.throws(() => validateWidgetSource("<title>&nbsp;&Tab;</title>"), /title/i);
  assert.throws(() => validateWidgetSource(titled("<meta http-equiv='r&#x65;fresh' content='0'>")), /refresh/i);
  assert.throws(() => validateWidgetSource(titled("<button formaction='&#x6a;ava&#x73;cript&#x3a;alert(1)'>x</button>")), /javascript/i);
  assert.throws(() => validateWidgetSource(titled("<button formaction='java&#10;script:alert(1)'>x</button>")), /javascript/i);
});

test("parses srcset data commas and zero-space candidate separators conservatively", () => {
  assert.doesNotThrow(() => validateWidgetSource(titled("<img srcset='data:image/png;base64,AA== 1x'>")));
  assert.throws(
    () => validateWidgetSource(titled("<img srcset='data:image/png;base64,AA== 1x,/remote.png 2x'>")),
    /URL|resource|srcset/i,
  );
  assert.throws(
    () => validateWidgetSource(titled("<source srcset='blob:https://example.invalid/id 1x,https://example.invalid/x 2x'>")),
    /URL|resource|srcset/i,
  );
});

test("normalizes CSS escapes before checking URL, import, and font channels", () => {
  const cases = [
    "<style>@\\69mport '/theme.css';</style>",
    "<style>.x{background:u\\72l('/x.png')}</style>",
    "<style>@font-\\66 ace{font-family:x;src:url(data:font/woff2;base64,AA==)}</style>",
    "<p style=\"background:u\\000072l('/x.png')\"></p>",
  ];
  for (const fragment of cases) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /URL|resource|font|import/i, fragment);
  }
});

test("requires real reduced-motion evidence rather than comment or string decoys", () => {
  assert.throws(
    () => validateWidgetSource(titled("<style>.x{animation:spin 1s}/* @media (prefers-reduced-motion: reduce){} */</style>")),
    /reduced motion/i,
  );
  assert.throws(
    () => validateWidgetSource(titled("<script>const decoy = \"matchMedia('(prefers-reduced-motion: reduce)')\"; requestAnimationFrame(draw);</script>")),
    /reduced motion/i,
  );
  assert.throws(
    () => validateWidgetSource(titled("<script>const typeDecoy = \"orbit.widget.preferences.v1\"; const reducedMotion = false; requestAnimationFrame(draw);</script>")),
    /reduced motion/i,
  );
  assert.doesNotThrow(() => validateWidgetSource(titled("<script>const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches; requestAnimationFrame(draw);</script>")));
  assert.doesNotThrow(() => validateWidgetSource(titled(`<script>
    addEventListener("message", (event) => {
      if (event.data.type === "orbit.widget.preferences.v1" && event.data.payload.reducedMotion) cancelAnimationFrame(frame);
    });
    requestAnimationFrame(draw);
  </script>`)));
});

test("rejects parser-state absorbers and unclosed inert or raw-text containers", () => {
  for (const fragment of ["<plaintext>absorbed", "<xmp>absorbed", "<noembed>absorbed", "<noframes>absorbed", "<noscript>absorbed"]) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /forbidden|absorber/i, fragment);
  }
  for (const fragment of ["<template><p>unclosed</p>", "<textarea>unclosed", "<script>unclosed", "<style>unclosed"]) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /unterminated|unclosed/i, fragment);
  }
  assert.doesNotThrow(() => validateWidgetSource(titled("<template shadowrootmode='open'><p>safe</p></template><textarea>notes</textarea>")));
});

test("preserves own __proto__ data for schema rejection and normalizes null prototypes", () => {
  const message = Object.assign(Object.create(null), {
    type: "orbit.widget.status.v1",
    version: 1,
    payload: Object.assign(Object.create(null), { message: "ready" }),
  });
  const normalized = validateWidgetMessage(WIDGET_TO_HOST, message);
  assert.equal(Object.getPrototypeOf(normalized), Object.prototype);
  assert.equal(Object.getPrototypeOf(normalized.payload), Object.prototype);

  const pollutedPayload = { message: "ready" };
  Object.defineProperty(pollutedPayload, "__proto__", { value: { canonicalData: true }, enumerable: true });
  assert.throws(
    () => validateWidgetMessage(WIDGET_TO_HOST, { type: "orbit.widget.status.v1", version: 1, payload: pollutedPayload }),
    /unknown|field|plain JSON data/i,
  );
  assert.equal(Object.getPrototypeOf(pollutedPayload), Object.prototype);
});

test("enforces message bytes for oversized nested payload data", () => {
  const message = structuredClone(validHostMessages[0]);
  message.payload.tokens.fontBody = "x".repeat(MAX_WIDGET_MESSAGE_BYTES);
  assert.throws(() => validateWidgetMessage(HOST_TO_WIDGET, message), /64 KiB|message.*bytes/i);
});

test("rejects frameset source before envelope construction", () => {
  const source = titled("<FrAmEsEt></FrAmEsEt>");
  assert.throws(() => validateWidgetSource(source), /frameset|forbidden|absorber/i);
  assert.throws(() => buildWidgetDocument(source, { bootstrapSource: "" }), /frameset|forbidden|absorber/i);
});

test("recognizes every JavaScript line terminator and executable template expressions", () => {
  for (const terminator of ["\r", "\u2028", "\u2029"]) {
    const source = titled(`<script>// matchMedia('(prefers-reduced-motion: reduce)')${terminator}requestAnimationFrame(draw);</script>`);
    assert.throws(() => validateWidgetSource(source), /reduced motion/i);
  }
  assert.throws(
    () => validateWidgetSource(titled("<script>const rendered = `${requestAnimationFrame(draw)}`;</script>")),
    /reduced motion/i,
  );
  assert.throws(
    () => validateWidgetSource(titled("<script>const inert = `requestAnimationFrame(draw)`;</script>")),
    /reduced motion/i,
  );
});

test("rejects mixed-case SVG script and filter-image resource URLs", () => {
  const cases = [
    "<svg><ScRiPt HrEf='/module.js'></ScRiPt></svg>",
    "<svg><script XLINK:href='https://example.invalid/module.js'></script></svg>",
    "<svg><FeImage HREF='/texture.png'></FeImage></svg>",
    "<svg><feimage xlink:HREF='https://example.invalid/texture.png'></feimage></svg>",
  ];
  for (const fragment of cases) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /URL|resource|script/i, fragment);
  }
  assert.doesNotThrow(() => validateWidgetSource(titled("<svg><feImage href='data:image/png;base64,AA=='></feImage></svg>")));
});

test("rejects ambiguous JavaScript regex literals before comment stripping", () => {
  const cases = [
    String.raw`<script>const lineCommentPattern = /\/\//; requestAnimationFrame(draw);</script>`,
    String.raw`<script>const blockCommentPattern = /\/\*/; requestAnimationFrame(draw);</script>`,
    String.raw`<script>const escapedClass = /[/*\\]]+/gi; requestAnimationFrame(draw);</script>`,
  ];
  for (const fragment of cases) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /ambiguous.*regular expression|comment delimiter/i, fragment);
  }
  assert.throws(
    () => validateWidgetSource(titled("<script>const safePattern = /[a-z/]+/g; requestAnimationFrame(draw);</script>")),
    /reduced motion/i,
  );
});

test("uses the regex lexical goal at statement starts after control headers and blocks", () => {
  const cases = [
    String.raw`<script>if (enabled) /\/\//.test(input); requestAnimationFrame(draw);</script>`,
    String.raw`<script>if (enabled) { drawStatic(); } /\/\*/.test(input); requestAnimationFrame(draw);</script>`,
    String.raw`<script>if (enabled) { drawStatic(); } else /\/\//.test(input); requestAnimationFrame(draw);</script>`,
  ];
  for (const fragment of cases) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /ambiguous.*regular expression|comment delimiter/i, fragment);
  }
});

test("rejects ambiguous regex comment-delimiter sequences independent of lexical context", () => {
  const cases = [
    String.raw`<script type="module">export default /\/\//;</script>`,
    String.raw`<script>const matcher = new /\/\//.constructor();</script>`,
    String.raw`<script>class Matcher extends /\/\//.constructor {}</script>`,
  ];
  for (const fragment of cases) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /ambiguous.*regular expression|comment delimiter/i, fragment);
  }
  assert.doesNotThrow(() => validateWidgetSource(titled("<script>const ratio = 10 / 2; const safe = /[a-z/]+/i;</script>")));
});

test("rejects comment delimiters in regex-style character classes after pattern prefixes", () => {
  const cases = [
    String.raw`<script type="module">export default /a[/*]/; requestAnimationFrame(draw);</script>`,
    String.raw`<script>const Matcher = new /a[//]/.constructor(); requestAnimationFrame(draw);</script>`,
  ];
  for (const fragment of cases) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /ambiguous.*regular expression|comment delimiter/i, fragment);
  }
  assert.doesNotThrow(() => validateWidgetSource(titled("<script>const ratio = total / count; const safe = /a[b/c*]+/i;</script>")));
});

test("applies raw animation-token policy before JavaScript lexical stripping", () => {
  const rawTokenCases = [
    "<script>// requestAnimationFrame(draw)</script>",
    "<script>const note = \"requestAnimationFrame(draw)\";</script>",
    "<script>const note = 'element.animate(keyframes)';</script>",
    "<script>const note = `new Animation(effect, document.timeline)`;</script>",
    "<script>const note = /new KeyframeEffect/;</script>",
    "<script>function schedule() { return /'/, requestAnimationFrame(draw); }</script>",
    "<script>const Matcher = new /`/.constructor(); requestAnimationFrame(draw);</script>",
  ];
  for (const fragment of rawTokenCases) {
    assert.throws(() => validateWidgetSource(titled(fragment)), /reduced motion/i, fragment);
  }
  assert.throws(
    () => validateWidgetSource(titled("<script>const decoy = \"matchMedia('(prefers-reduced-motion: reduce)')\"; // requestAnimationFrame(draw)</script>")),
    /reduced motion/i,
  );
  assert.doesNotThrow(() => validateWidgetSource(titled(`<script>
    const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
    const policyProbe = "requestAnimationFrame(draw)";
  </script>`)));
});

test("covers every supported raw JavaScript animation API token", () => {
  const tokens = [
    "requestAnimationFrame(draw)",
    "element.animate(keyframes)",
    "new Animation(effect, document.timeline)",
    "new KeyframeEffect(element, keyframes)",
  ];
  for (const token of tokens) {
    assert.throws(() => validateWidgetSource(titled(`<script>/* ${token} */</script>`)), /reduced motion/i, token);
  }
});

test("rejects regex-forged reduced-motion evidence and accepts slash-free handlers", () => {
  const forgedMatchMedia = String.raw`<script>
    function evidence() { return /matchMedia\('(prefers-reduced-motion: reduce)'\)/; }
    requestAnimationFrame(draw);
  </script>`;
  const forgedProtocol = String.raw`<script>
    function evidence() { return /type === "orbit.widget.preferences.v1" && payload.reducedMotion/; }
    requestAnimationFrame(draw);
  </script>`;
  assert.throws(() => validateWidgetSource(titled(forgedMatchMedia)), /reduced motion/i);
  assert.throws(() => validateWidgetSource(titled(forgedProtocol)), /reduced motion/i);

  assert.doesNotThrow(() => validateWidgetSource(titled(`<script>
    const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
    requestAnimationFrame(draw);
  </script>`)));
  assert.doesNotThrow(() => validateWidgetSource(titled(`<script>
    addEventListener("message", event => {
      if (event.data.type === "orbit.widget.preferences.v1" && event.data.payload.reducedMotion) cancelAnimationFrame(frame);
    });
    requestAnimationFrame(draw);
  </script>`)));
});

test("requires an actual reduce media query outside CSS strings and comments", () => {
  const stringDecoy = `<style>
    .spinner { animation: spin 1s infinite; content: "@media (prefers-reduced-motion: reduce)"; }
  </style>`;
  const noPreference = `<style>
    .spinner { animation: spin 1s infinite; }
    @media (prefers-reduced-motion: no-preference) { .spinner { opacity: 1; } }
  </style>`;
  assert.throws(() => validateWidgetSource(titled(stringDecoy)), /reduced motion/i);
  assert.throws(() => validateWidgetSource(titled(noPreference)), /reduced motion/i);
  assert.doesNotThrow(() => validateWidgetSource(titled(`<style>
    .spinner { animation: spin 1s infinite; }
    @media (prefers-reduced-motion: reduce) { .spinner { animation: none; } }
  </style>`)));
});

test("requires matchMedia reduce and ignores classic HTML comments and hashbang evidence", () => {
  assert.throws(() => validateWidgetSource(titled(`<script>
    const moving = matchMedia("(prefers-reduced-motion: no-preference)").matches;
    requestAnimationFrame(draw);
  </script>`)), /reduced motion/i);
  for (const comment of [
    `<!-- matchMedia("(prefers-reduced-motion: reduce)")`,
    `--> matchMedia("(prefers-reduced-motion: reduce)")`,
    `#! matchMedia("(prefers-reduced-motion: reduce)")`,
  ]) {
    assert.throws(() => validateWidgetSource(titled(`<script>${comment}
      requestAnimationFrame(draw);
    </script>`)), /reduced motion/i, comment);
  }
  assert.doesNotThrow(() => validateWidgetSource(titled(`<script>
    const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
    requestAnimationFrame(draw);
  </script>`)));
});

test("ignores mid-line classic-script comment evidence outside strings", () => {
  for (const comment of [
    `const marker = true; <!-- matchMedia("(prefers-reduced-motion: reduce)")`,
    `const marker = true; --> matchMedia("(prefers-reduced-motion: reduce)")`,
  ]) {
    assert.throws(() => validateWidgetSource(titled(`<script>${comment}
      requestAnimationFrame(draw);
    </script>`)), /reduced motion/i, comment);
  }
  assert.doesNotThrow(() => validateWidgetSource(titled(`<script>
    const marker = "<!--";
    const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
    requestAnimationFrame(draw);
  </script>`)));
});

test("requires structural protocol reduced-motion handling rather than nearby identifiers", () => {
  const looseComparison = `<script>
    const matchingType = event.data.type === "orbit.widget.preferences.v1";
    const reducedMotion = false;
    requestAnimationFrame(draw);
  </script>`;
  assert.throws(() => validateWidgetSource(titled(looseComparison)), /reduced motion/i);

  assert.doesNotThrow(() => validateWidgetSource(titled(`<script>
    addEventListener("message", event => {
      const { reducedMotion } = event.data.payload;
      if (event.data.type === "orbit.widget.preferences.v1" && reducedMotion) cancelAnimationFrame(frame);
    });
    requestAnimationFrame(draw);
  </script>`)));
  assert.doesNotThrow(() => validateWidgetSource(titled(`<script>
    switch (event.data.type) {
      case "orbit.widget.preferences.v1":
        if (event.data.payload.reducedMotion) cancelAnimationFrame(frame);
    }
    requestAnimationFrame(draw);
  </script>`)));
});

test("detects animate tokens across member whitespace and comments", () => {
  for (const call of [
    "body . animate(keyframes)",
    "body . /* visual transition */ animate(keyframes)",
  ]) {
    assert.throws(() => validateWidgetSource(titled(`<script>${call}</script>`)), /reduced motion/i, call);
  }
  assert.doesNotThrow(() => validateWidgetSource(titled(`<script>
    const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
    body . animate(keyframes);
  </script>`)));
});

test("requires protocol type and reducedMotion payload to use the same receiver", () => {
  const cases = [
    `<script>
      if (event.data.type === "orbit.widget.preferences.v1" && unrelated.payload.reducedMotion) cancelAnimationFrame(frame);
      requestAnimationFrame(draw);
    </script>`,
    `<script>
      const { reducedMotion } = unrelated.payload;
      if (event.data.type === "orbit.widget.preferences.v1" && reducedMotion) cancelAnimationFrame(frame);
      requestAnimationFrame(draw);
    </script>`,
    `<script>
      switch (event.data.type) {
        case "orbit.widget.preferences.v1":
          if (unrelated.payload.reducedMotion) cancelAnimationFrame(frame);
      }
      requestAnimationFrame(draw);
    </script>`,
  ];
  for (const source of cases) assert.throws(() => validateWidgetSource(titled(source)), /reduced motion/i);
  assert.doesNotThrow(() => validateWidgetSource(titled(`<script>
    if (message.type === "orbit.widget.preferences.v1" && message.payload.reducedMotion) cancelAnimationFrame(frame);
    requestAnimationFrame(draw);
  </script>`)));
});

test("scopes switch protocol evidence to the exact preferences case body", () => {
  const wrongCase = `<script>
    switch (message.type) {
      case "orbit.widget.preferences.v1":
        break;
      case "orbit.widget.other.v1":
        if (message.payload.reducedMotion) cancelAnimationFrame(frame);
    }
    requestAnimationFrame(draw);
  </script>`;
  const afterSwitch = `<script>
    switch (message.type) {
      case "orbit.widget.preferences.v1":
        break;
    }
    if (message.payload.reducedMotion) cancelAnimationFrame(frame);
    requestAnimationFrame(draw);
  </script>`;
  assert.throws(() => validateWidgetSource(titled(wrongCase)), /reduced motion/i);
  assert.throws(() => validateWidgetSource(titled(afterSwitch)), /reduced motion/i);
  assert.doesNotThrow(() => validateWidgetSource(titled(`<script>
    switch (message.type) {
      case "orbit.widget.preferences.v1":
        if (message.payload.reducedMotion) cancelAnimationFrame(frame);
        break;
      default:
        break;
    }
    requestAnimationFrame(draw);
  </script>`)));
});
