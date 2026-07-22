import { validateWidgetSource } from "./widget-policy.js";

export const WIDGET_CSP = "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data: blob:; media-src data: blob:; font-src 'none'; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'";

const DIAGNOSTIC_BOUNDARY_SOURCE = `
(() => {
  const report = (level, value) => {
    const message = value instanceof Error ? value.message : String(value ?? "Unknown widget error");
    globalThis.__balaurReportDiagnostic?.({ level, message: message.slice(0, 4096) });
  };
  addEventListener("error", (event) => report("error", event.error || event.message));
  addEventListener("unhandledrejection", (event) => report("error", event.reason));
})();`;

export function buildWidgetDocument(source, { bootstrapSource } = {}) {
  validateWidgetSource(source);
  if (typeof bootstrapSource !== "string") throw new TypeError("Widget bootstrap source must be a string");
  if (/<\/script/i.test(bootstrapSource)) throw new TypeError("Widget bootstrap source must not close its trusted script element");
  return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta data-balaur-csp http-equiv="Content-Security-Policy" content="${WIDGET_CSP}">
<meta name="referrer" content="no-referrer">
<script data-balaur-bootstrap>${bootstrapSource}</script>
</head>
<body>
${source}
<script data-balaur-diagnostic-boundary>${DIAGNOSTIC_BOUNDARY_SOURCE}</script>
</body>
</html>`;
}
