#!/usr/bin/env node
import { writeFileSync } from "node:fs";
import { BrowserSession } from "./browser-check.mjs";

const DEFAULT_URL = "http://localhost:4173/";
const DEFAULT_RUNS = 5;

function parseArgs(argv) {
  const options = { url: DEFAULT_URL, runs: DEFAULT_RUNS, label: "benchmark", output: null, functionalOnly: false };
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (/^https?:\/\//.test(argument)) options.url = argument;
    else if (argument === "--runs") options.runs = Number(argv[++index]);
    else if (argument === "--label") options.label = argv[++index];
    else if (argument === "--output") options.output = argv[++index];
    else if (argument === "--functional-only") options.functionalOnly = true;
    else throw new Error(`Unknown argument: ${argument}`);
  }
  if (!Number.isInteger(options.runs) || options.runs < 1) throw new Error("--runs must be a positive integer");
  return options;
}

function percentile(samples, value) {
  const sorted = [...samples].sort((left, right) => left - right);
  return sorted[Math.max(0, Math.ceil(value / 100 * sorted.length) - 1)];
}

function rounded(value) {
  return Math.round(value * 1000) / 1000;
}

async function settle(session) {
  await session.evaluate("new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))");
}

async function replaceDocument(session, nodes) {
  const current = await session.evaluate("window.orbitCanvas.getDocument().nodes.map(node => node.id)");
  for (let index = 0; index < current.length; index += 45) {
    const ids = current.slice(index, index + 45);
    await session.evaluate(`window.orbitCanvas.applyOperations(${JSON.stringify(ids.map(id => ({ type: "node.remove", id })))})`);
  }
  for (let index = 0; index < nodes.length; index += 45) {
    const batch = nodes.slice(index, index + 45).map(node => ({ type: "node.add", node }));
    await session.evaluate(`window.orbitCanvas.applyOperations(${JSON.stringify(batch)})`);
  }
  await settle(session);
}

function fixtureNodes() {
  return [
    { id: "benchmark-a", type: "text", x: 0, y: 0, width: 300, height: 180, color: "1", text: "# Benchmark A\nPointer and focus probe." },
    { id: "benchmark-b", type: "text", x: 420, y: 0, width: 300, height: 180, color: "2", text: "# Benchmark B\nConnection target." },
    { id: "benchmark-widget", type: "file", x: 840, y: 0, width: 480, height: 290, color: "5", file: "widgets/focus-orbit.html" },
  ];
}

function hundredNodes() {
  return Array.from({ length: 100 }, (_, index) => ({
    id: `benchmark-node-${index}`,
    type: "text",
    x: index % 10 * 330,
    y: Math.floor(index / 10) * 210,
    width: 280,
    height: 160,
    color: String(index % 6 + 1),
    text: `# Benchmark ${index + 1}\nStable 100-node render fixture.`,
  }));
}

async function probeBehavior(session) {
  await replaceDocument(session, fixtureNodes());
  await session.evaluate("document.getElementById('fitView').click()");
  await settle(session);
  const details = await session.evaluate(`(async () => {
    const pause = milliseconds => new Promise(resolve => setTimeout(resolve, milliseconds));
    const node = id => document.querySelector('.canvas-node[data-id="' + id + '"]');
    const point = element => { const rect = element.getBoundingClientRect(); return { x: rect.left + rect.width / 2, y: rect.top + Math.min(rect.height / 2, 60) }; };
    const pointer = (target, type, at, pointerId = 71) => target.dispatchEvent(new PointerEvent(type, { bubbles: true, cancelable: true, button: 0, buttons: type === "pointerup" ? 0 : 1, pointerId, clientX: at.x, clientY: at.y }));
    const documentCount = () => window.orbitCanvas.getDocument().nodes.length;
    const details = {};

    let first = node("benchmark-a");
    let at = point(first);
    pointer(first.querySelector('.node-content') || first, "pointerdown", at);
    details.selection = !!node("benchmark-a")?.classList.contains("selected") && document.querySelector('.app-shell')?.classList.contains('inspector-open');

    const beforeDoubleClick = documentCount();
    first = node("benchmark-a");
    first.dispatchEvent(new MouseEvent("dblclick", { bubbles: true, cancelable: true, clientX: at.x, clientY: at.y, detail: 2 }));
    details.doubleClick = documentCount() === beforeDoubleClick;

    document.querySelector('.tool[data-tool="note"]').click();
    const beforeNoteTool = documentCount();
    first = node("benchmark-a");
    at = point(first);
    pointer(first.querySelector('.node-content') || first, "pointerdown", at, 72);
    pointer(window, "pointerup", at, 72);
    details.noteTool = documentCount() === beforeNoteTool;
    document.querySelector('.tool[data-tool="select"]').click();

    const dragBefore = window.orbitCanvas.getDocument().nodes.find(item => item.id === "benchmark-a");
    first = node("benchmark-a");
    at = point(first);
    pointer(first.querySelector('.node-content') || first, "pointerdown", at, 73);
    pointer(window, "pointermove", { x: at.x + 32, y: at.y + 24 }, 73);
    pointer(window, "pointerup", { x: at.x + 32, y: at.y + 24 }, 73);
    const dragAfter = window.orbitCanvas.getDocument().nodes.find(item => item.id === "benchmark-a");
    details.drag = dragAfter.x !== dragBefore.x && dragAfter.y !== dragBefore.y && node("benchmark-a").style.left === dragAfter.x + "px";

    const resizeBefore = { width: dragAfter.width, height: dragAfter.height };
    first = node("benchmark-a");
    const handle = first.querySelector('.resize-handle');
    const handlePoint = point(handle);
    pointer(handle, "pointerdown", handlePoint, 74);
    pointer(window, "pointermove", { x: handlePoint.x + 35, y: handlePoint.y + 28 }, 74);
    pointer(window, "pointerup", { x: handlePoint.x + 35, y: handlePoint.y + 28 }, 74);
    const resizeAfter = window.orbitCanvas.getDocument().nodes.find(item => item.id === "benchmark-a");
    details.resize = resizeAfter.width > resizeBefore.width && resizeAfter.height > resizeBefore.height;

    document.querySelector('.tool[data-tool="connect"]').click();
    const edgesBefore = window.orbitCanvas.getDocument().edges.length;
    first = node("benchmark-a");
    pointer(first.querySelector('.node-content') || first, "pointerdown", point(first), 75);
    const second = node("benchmark-b");
    pointer(second.querySelector('.node-content') || second, "pointerdown", point(second), 76);
    details.connection = window.orbitCanvas.getDocument().edges.length === edgesBefore + 1;
    document.querySelector('.tool[data-tool="select"]').click();

    const inspector = document.getElementById('inspector');
    first = node("benchmark-a");
    pointer(first.querySelector('.node-content') || first, "pointerdown", point(first), 77);
    const inspectorRoot = inspector.shadowRoot || inspector;
    const field = inspectorRoot.querySelector('textarea, input:not([type="number"])');
    if (field) {
      field.focus();
      if (typeof field.setSelectionRange === "function") field.setSelectionRange(2, 5);
      const start = field.selectionStart;
      const end = field.selectionEnd;
      field.dispatchEvent(new InputEvent("input", { bubbles: true, composed: true, inputType: "insertText", data: "" }));
      details.focus = document.activeElement === inspector || document.activeElement === field || inspectorRoot.activeElement === field;
      details.focus = details.focus && field.selectionStart === start && field.selectionEnd === end;
    } else details.focus = false;

    const widget = node("benchmark-widget");
    const host = widget?.querySelector('balaur-widget-frame');
    if (host?.activate) host.activate();
    for (let attempt = 0; attempt < 30 && !host?.shadowRoot?.querySelector('iframe') && !widget?.querySelector('iframe'); attempt += 1) await pause(50);
    const frameBefore = host?.shadowRoot?.querySelector('iframe') || widget?.querySelector('iframe') || null;
    details.iframeReady = !!frameBefore;
    document.querySelector('.nav-item[data-filter="1"]').click();
    document.querySelector('.nav-item[data-filter="all"]').click();
    const widgetAfter = node("benchmark-widget");
    const hostAfter = widgetAfter?.querySelector('balaur-widget-frame');
    const frameAfter = hostAfter?.shadowRoot?.querySelector('iframe') || widgetAfter?.querySelector('iframe') || null;
    details.iframeIdentity = !!frameBefore && frameAfter === frameBefore;

    document.querySelector('.nav-item[data-filter="1"]').click();
    details.filter = node("benchmark-a")?.classList.contains('filtered') === false
      && node("benchmark-b")?.classList.contains('filtered') === true
      && getComputedStyle(node("benchmark-b")).pointerEvents === "none";
    document.querySelector('.nav-item[data-filter="all"]').click();

    const canvas = document.getElementById('canvas');
    const transformBeforePan = document.getElementById('world').style.transform;
    document.querySelector('.tool[data-tool="pan"]').click();
    const canvasRect = canvas.getBoundingClientRect();
    const background = { x: canvasRect.right - 25, y: canvasRect.bottom - 25 };
    pointer(canvas, "pointerdown", background, 79);
    pointer(window, "pointermove", { x: background.x - 40, y: background.y - 25 }, 79);
    pointer(window, "pointerup", { x: background.x - 40, y: background.y - 25 }, 79);
    details.pan = document.getElementById('world').style.transform !== transformBeforePan && !canvas.classList.contains('panning');
    document.querySelector('.tool[data-tool="select"]').click();

    const transformBeforeZoom = document.getElementById('world').style.transform;
    canvas.dispatchEvent(new WheelEvent('wheel', { bubbles: true, cancelable: true, deltaY: -120, clientX: canvasRect.left + canvasRect.width / 2, clientY: canvasRect.top + canvasRect.height / 2 }));
    details.zoom = document.getElementById('world').style.transform !== transformBeforeZoom;

    return details;
  })()`);
  await session.evaluate("document.getElementById('fitView').click()");
  await settle(session);
  const widgetPoints = await session.evaluate(`(() => {
    const widget = document.querySelector('.canvas-node[data-id="benchmark-widget"]');
    const header = widget?.querySelector('.node-kicker');
    const frame = widget?.querySelector('balaur-widget-frame')?.shadowRoot?.querySelector('iframe') || widget?.querySelector('iframe');
    if (!widget || !header || !frame) return null;
    const frameRect = frame.getBoundingClientRect();
    const widgetRect = widget.getBoundingClientRect();
    let selectPoint = null;
    for (let y = widgetRect.top + 8; y < Math.min(frameRect.top, widgetRect.bottom); y += 8) {
      for (let x = widgetRect.left + 8; x < widgetRect.right - 8; x += 12) {
        const hit = document.elementFromPoint(x, y);
        if (hit?.closest?.('.canvas-node') === widget && !hit.closest('button,a,.connection-handle')) {
          selectPoint = { x, y };
          break;
        }
      }
      if (selectPoint) break;
    }
    const model = window.orbitCanvas.getDocument().nodes.find(node => node.id === "benchmark-widget");
    return {
      header: selectPoint,
      viewport: { x: frameRect.left + frameRect.width / 2, y: frameRect.top + frameRect.height / 2 },
      before: { x: model.x, y: model.y },
    };
  })()`);
  if (widgetPoints?.header) {
    await session.click(widgetPoints.header.x, widgetPoints.header.y);
    await session.mouse(widgetPoints.header.x, widgetPoints.header.y, "mousePressed");
    await session.mouse(widgetPoints.viewport.x, widgetPoints.viewport.y, "mouseMoved");
    const shieldProbe = await session.evaluate(`(() => {
      const widget = document.querySelector('.canvas-node[data-id="benchmark-widget"]');
      const shield = widget?.querySelector('.widget-shield');
      const rect = shield?.getBoundingClientRect();
      const hit = rect?.width > 0 && rect?.height > 0 ? document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2) : null;
      return {
        displayed: !!shield && getComputedStyle(shield).display !== "none",
        nonzero: !!rect && rect.width > 0 && rect.height > 0,
        hitShield: hit === shield,
        selected: widget?.classList.contains("selected") === true,
        point: rect ? { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 } : null,
      };
    })()`);
    if (shieldProbe.point) await session.mouse(shieldProbe.point.x + 12, shieldProbe.point.y + 8, "mouseMoved");
    await session.mouse(
      shieldProbe.point?.x ?? widgetPoints.viewport.x,
      shieldProbe.point?.y ?? widgetPoints.viewport.y,
      "mouseReleased",
    );
    const dragProbe = await session.evaluate(`(() => {
      const widget = document.querySelector('.canvas-node[data-id="benchmark-widget"]');
      const model = window.orbitCanvas.getDocument().nodes.find(node => node.id === "benchmark-widget");
      return {
        selected: widget?.classList.contains("selected") === true,
        moved: model.x !== ${widgetPoints.before.x} || model.y !== ${widgetPoints.before.y},
        shieldReleased: getComputedStyle(widget.querySelector('.widget-shield')).display === "none",
      };
    })()`);
    details.iframeShield = shieldProbe.displayed && shieldProbe.nonzero && shieldProbe.hitShield;
    details.iframePointerDrag = shieldProbe.selected && dragProbe.selected && dragProbe.moved && dragProbe.shieldReleased;
  } else {
    details.iframeShield = false;
    details.iframePointerDrag = false;
  }
  delete details.iframeReady;
  return details;
}

async function measureHotPaths(session) {
  await replaceDocument(session, hundredNodes());
  await session.evaluate("document.getElementById('fitView').click()");
  await settle(session);
  return session.evaluate(`(() => {
    const elapsed = action => { const started = performance.now(); action(); return performance.now() - started; };
    const pointer = (target, type, at, pointerId) => target.dispatchEvent(new PointerEvent(type, { bubbles: true, cancelable: true, button: 0, buttons: type === "pointerup" ? 0 : 1, pointerId, clientX: at.x, clientY: at.y }));
    const point = element => { const rect = element.getBoundingClientRect(); return { x: rect.left + rect.width / 2, y: rect.top + Math.min(rect.height / 2, 40) }; };
    const canvas = document.getElementById('canvas');

    const rerender = elapsed(() => document.querySelector('.nav-item[data-filter="1"]').click());
    let target = [...document.querySelectorAll('.canvas-node:not(.filtered)')].find(element => {
      const rect = element.getBoundingClientRect();
      const canvasRect = canvas.getBoundingClientRect();
      return rect.right > canvasRect.left && rect.left < canvasRect.right && rect.bottom > canvasRect.top && rect.top < canvasRect.bottom;
    }) || document.querySelector('.canvas-node:not(.filtered)');
    let at = point(target);
    const select = elapsed(() => pointer(target.querySelector('.node-content') || target, 'pointerdown', at, 91));
    pointer(window, 'pointerup', at, 91);

    target = document.querySelector('.canvas-node.selected') || document.querySelector('.canvas-node:not(.filtered)');
    at = point(target);
    const drag = elapsed(() => {
      pointer(target.querySelector('.node-content') || target, 'pointerdown', at, 92);
      pointer(window, 'pointermove', { x: at.x + 12, y: at.y + 9 }, 92);
      pointer(window, 'pointerup', { x: at.x + 12, y: at.y + 9 }, 92);
    });

    document.querySelector('.tool[data-tool="pan"]').click();
    const canvasRect = canvas.getBoundingClientRect();
    const background = { x: canvasRect.right - 20, y: canvasRect.bottom - 20 };
    const pan = elapsed(() => {
      pointer(canvas, 'pointerdown', background, 93);
      pointer(window, 'pointermove', { x: background.x - 12, y: background.y - 8 }, 93);
      pointer(window, 'pointerup', { x: background.x - 12, y: background.y - 8 }, 93);
    });
    document.querySelector('.tool[data-tool="select"]').click();

    const zoom = elapsed(() => canvas.dispatchEvent(new WheelEvent('wheel', { bubbles: true, cancelable: true, deltaY: -80, clientX: canvasRect.left + canvasRect.width / 2, clientY: canvasRect.top + canvasRect.height / 2 })));
    return { rerender100: rerender, select, drag, pan, zoom, domNodes: document.querySelectorAll('.canvas-node').length, documentNodes: window.orbitCanvas.getDocument().nodes.length };
  })()`);
}

async function runOnce(url, run, functionalOnly) {
  const session = new BrowserSession({ url });
  await session.start();
  try {
    await session.send("Page.addScriptToEvaluateOnNewDocument", { source: `(() => {
      const capture = () => {
        if (window.__canvasInitialReadyAt == null && document.querySelectorAll('.canvas-node').length > 0) window.__canvasInitialReadyAt = performance.now();
      };
      new MutationObserver(capture).observe(document, { childList: true, subtree: true });
    })();` });
    await session.navigate();
    await session.waitFor("window.orbitCanvas && window.__canvasInitialReadyAt != null && document.querySelectorAll('.canvas-node').length === window.orbitCanvas.getDocument().nodes.length", 20000);
    await settle(session);
    const initialRender = await session.evaluate("window.__canvasInitialReadyAt");
    const userAgent = await session.evaluate("navigator.userAgent");
    const functional = await probeBehavior(session);
    const measured = functionalOnly ? null : await measureHotPaths(session);
    if (measured && (measured.domNodes !== 100 || measured.documentNodes !== 100)) throw new Error(`Run ${run}: expected 100 nodes, got DOM ${measured.domNodes}, document ${measured.documentNodes}`);
    const metrics = measured ? {
      initialRender: rounded(initialRender),
      rerender100: rounded(measured.rerender100),
      select: rounded(measured.select),
      drag: rounded(measured.drag),
      pan: rounded(measured.pan),
      zoom: rounded(measured.zoom),
    } : null;
    return {
      run,
      metrics,
      functional,
      consoleErrors: session.consoleErrors,
      failedRequests: session.failedRequests.length,
      userAgent,
    };
  } finally {
    await session.close();
  }
}

const options = parseArgs(process.argv.slice(2));
const runs = [];
for (let run = 1; run <= options.runs; run += 1) runs.push(await runOnce(options.url, run, options.functionalOnly));
const metricNames = ["initialRender", "rerender100", "select", "drag", "pan", "zoom"];
const metrics = options.functionalOnly ? null : Object.fromEntries(metricNames.map(name => {
  const samples = runs.map(run => run.metrics[name]);
  return [name, { unit: "ms", samples, p50: rounded(percentile(samples, 50)), p95: rounded(percentile(samples, 95)) }];
}));
const functionalNames = Object.keys(runs[0].functional);
const functional = Object.fromEntries(functionalNames.map(name => [name, runs.every(run => run.functional[name] === true)]));
const result = {
  schemaVersion: 1,
  label: options.label,
  url: options.url,
  freshRuns: options.runs,
  percentileMethod: "nearest-rank",
  environment: { userAgent: runs[0].userAgent },
  metrics,
  functional,
  gates: {
    allFunctional: Object.values(functional).every(Boolean),
    noConsoleErrors: runs.every(run => run.consoleErrors.length === 0),
    noFailedRequests: runs.every(run => run.failedRequests === 0),
  },
  runs,
};
const output = JSON.stringify(result, null, 2);
if (options.output) writeFileSync(options.output, output + "\n");
console.log(output);
if (!result.gates.allFunctional || !result.gates.noConsoleErrors || !result.gates.noFailedRequests) process.exitCode = 1;
