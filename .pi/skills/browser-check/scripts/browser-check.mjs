#!/usr/bin/env node
// Headless Chrome verification driver for the Balaur canvas app (CDP over WebSocket).
// Dependency-free: uses Node's global WebSocket and fetch. See ../SKILL.md.
//
//   node browser-check.mjs smoke     [url] [--profile dir] [--offline] [--width N] [--height N] [--screenshot dir]
//   node browser-check.mjs components [url] [--profile dir] [--width N] [--height N]
//   node browser-check.mjs widgets   [url] [--profile dir] [--width N] [--height N] [--screenshot dir]
//   node browser-check.mjs eval      [url] <expression> [--wait expr] [--profile dir]
//   node browser-check.mjs shot      [url] <file.png> [--selector css] [--profile dir]
import { spawn, execSync } from "node:child_process";
import { mkdirSync, mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { pathToFileURL } from "node:url";

const DEFAULT_URL = "http://localhost:4173/";

function parseArgs(argv) {
  const args = { flags: {}, positional: [] };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--profile" || a === "--wait" || a === "--selector" || a === "--width" || a === "--height" || a === "--screenshot") {
      args.flags[a.slice(2)] = argv[++i];
    } else if (a.startsWith("--")) {
      args.flags[a.slice(2)] = true;
    } else {
      args.positional.push(a);
    }
  }
  return args;
}

function findChrome() {
  for (const bin of ["google-chrome", "google-chrome-stable", "chromium", "chromium-browser"]) {
    try { execSync(`command -v ${bin}`, { stdio: "ignore" }); return bin; } catch {}
  }
  return null;
}

export class BrowserSession {
  constructor({ url = DEFAULT_URL, profile = null, width = 1440, height = 900 } = {}) {
    this.url = url;
    this.width = width;
    this.height = height;
    this.profile = profile || mkdtempSync(join(tmpdir(), "balaur-check-"));
    this.ownProfile = !profile;
    this.consoleErrors = [];
    this.failedRequests = [];
    this.requestedUrls = [];
    this._id = 0;
    this._pending = new Map();
    this.blockedUrlFragments = [];
  }

  async start() {
    const bin = findChrome();
    if (!bin) throw new Error("No Chrome/Chromium binary found on PATH");
    const port = 9200 + Math.floor(Math.random() * 700);
    this.chrome = spawn(bin, [
      "--headless=new", "--no-sandbox", "--disable-gpu",
      `--user-data-dir=${this.profile}`,
      `--window-size=${this.width},${this.height}`,
      `--remote-debugging-port=${port}`,
      "about:blank",
    ], { stdio: "ignore" });
    // Poll until the DevTools endpoint answers.
    for (let i = 0; i < 40; i++) {
      await new Promise(r => setTimeout(r, 250));
      try {
        const tabs = await (await fetch(`http://127.0.0.1:${port}/json`)).json();
        const tab = tabs.find(t => t.type === "page");
        if (tab) { this.wsUrl = tab.webSocketDebuggerUrl; break; }
      } catch {}
    }
    if (!this.wsUrl) throw new Error("Chrome DevTools endpoint did not come up");
    this.ws = new WebSocket(this.wsUrl);
    await new Promise((res, rej) => { this.ws.onopen = res; this.ws.onerror = rej; });
    this.ws.onmessage = (msg) => {
      const data = JSON.parse(msg.data);
      if (data.method === "Runtime.exceptionThrown") {
        const d = data.params.exceptionDetails;
        this.consoleErrors.push(`${d.text}: ${d.exception?.description?.split("\n")[0] ?? ""} (${d.url?.split("/").pop() ?? ""}:${d.lineNumber})`);
      }
      if (data.method === "Runtime.consoleAPICalled" && data.params.type === "error") {
        this.consoleErrors.push(data.params.args.map(a => a.value ?? a.description ?? "").join(" ").slice(0, 300));
      }
      if (data.method === "Network.requestWillBeSent") {
        this.requestedUrls.push(data.params.request.url);
      }
      if (data.method === "Network.loadingFailed" && !data.params.canceled) {
        this.failedRequests.push(data.params.requestId);
      }
      if (data.method === "Fetch.requestPaused") {
        const requestId = data.params.requestId;
        const blocked = this.blockedUrlFragments.some(fragment => data.params.request.url.includes(fragment));
        this.send(blocked ? "Fetch.failRequest" : "Fetch.continueRequest", blocked
          ? { requestId, errorReason: "Failed" }
          : { requestId });
      }
      if (data.id && this._pending.has(data.id)) { this._pending.get(data.id)(data); this._pending.delete(data.id); }
    };
    await this.send("Page.enable");
    await this.send("Runtime.enable");
    await this.send("Network.enable");
    return this;
  }

  send(method, params = {}) {
    return new Promise(res => {
      const id = ++this._id;
      this._pending.set(id, res);
      this.ws.send(JSON.stringify({ id, method, params }));
    });
  }

  async evaluate(expression) {
    const res = await this.send("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true });
    if (res.result?.exceptionDetails) {
      throw new Error(res.result.exceptionDetails.exception?.description ?? JSON.stringify(res.result.exceptionDetails));
    }
    return res.result?.result?.value;
  }

  async navigate(url = this.url) {
    await this.send("Page.navigate", { url });
    await this.waitFor("document.readyState === 'complete'", 15000);
  }

  async waitFor(expression, timeout = 10000) {
    const deadline = Date.now() + timeout;
    while (Date.now() < deadline) {
      try { if (await this.evaluate(`!!(${expression})`)) return true; } catch {}
      await new Promise(r => setTimeout(r, 200));
    }
    throw new Error(`Timed out waiting for: ${expression}`);
  }

  async reload() {
    await this.send("Page.reload");
    await this.waitFor("document.readyState === 'complete'", 15000);
  }

  async mouse(x, y, type, clickCount = 1) {
    await this.send("Input.dispatchMouseEvent", { type, x, y, button: "left", clickCount });
  }

  async click(x, y) {
    await this.mouse(x, y, "mousePressed", 1);
    await this.mouse(x, y, "mouseReleased", 1);
  }

  async dblclick(x, y) {
    await this.mouse(x, y, "mousePressed", 1);
    await this.mouse(x, y, "mouseReleased", 1);
    await new Promise(r => setTimeout(r, 90));
    await this.mouse(x, y, "mousePressed", 2);
    await this.mouse(x, y, "mouseReleased", 2);
  }

  async screenshot(file, selector = null) {
    const params = { format: "png" };
    if (selector) {
      const box = await this.evaluate(`(() => {
        const el = document.querySelector(${JSON.stringify(selector)});
        return el ? el.getBoundingClientRect().toJSON() : null;
      })()`);
      if (box) params.clip = { ...box, scale: 1 };
    }
    const res = await this.send("Page.captureScreenshot", params);
    writeFileSync(file, Buffer.from(res.result.data, "base64"));
    return file;
  }

  async setOffline(offline) {
    await this.send("Network.emulateNetworkConditions", offline
      ? { offline: true, latency: 0, downloadThroughput: -1, uploadThroughput: -1 }
      : { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
  }

  async failRequestsMatching(fragments) {
    this.blockedUrlFragments = [...fragments];
    await this.send("Fetch.enable", { patterns: [{ urlPattern: "*", requestStage: "Request" }] });
  }

  async close() {
    try { this.ws?.close(); } catch {}
    if (this.chrome) {
      this.chrome.kill();
      await new Promise(r => setTimeout(r, 400));
    }
    if (this.ownProfile) {
      try { rmSync(this.profile, { recursive: true, force: true, maxRetries: 3 }); } catch {}
    }
  }
}

// ---------------------------------------------------------------------------
// Shared probes
// ---------------------------------------------------------------------------

// Center of a fully visible non-group card content area, or null.
const PROBE_VISIBLE_CARD = `(() => {
  const c = document.getElementById("canvas").getBoundingClientRect();
  for (const el of document.querySelectorAll('.canvas-node:not(.group-node) .node-content')) {
    const r = el.getBoundingClientRect();
    if (r.left > c.left + 8 && r.top > c.top + 8 && r.right < c.right - 8 && r.bottom < c.bottom - 8 && r.width > 40 && r.height > 30)
      return { x: Math.round(r.left + r.width / 2), y: Math.round(r.top + r.height / 2) };
  }
  return null;
})()`;

// A viewport point over empty canvas background (no card, no toolbar), or null.
const PROBE_BACKGROUND_POINT = `(() => {
  const c = document.getElementById("canvas").getBoundingClientRect();
  for (let y = c.top + 30; y < c.bottom - 20; y += 24) {
    for (let x = c.left + 30; x < c.right - 20; x += 24) {
      const el = document.elementFromPoint(x, y);
      if (!el) continue;
      if (el.closest(".canvas-node") || el.closest(".canvas-tools") || el.closest(".zoom-tools") || el.closest(".minimap")) continue;
      if (el === document.getElementById("canvas") || el === document.getElementById("world") || el === document.getElementById("nodes"))
        return { x: Math.round(x), y: Math.round(y) };
    }
  }
  return null;
})()`;

// Structural JSON Canvas 1.0 check of the live document (no module import needed).
const PROBE_IS_CANVAS = `(() => {
  const doc = window.orbitCanvas.getDocument();
  if (!doc || !Array.isArray(doc.nodes) || !Array.isArray(doc.edges)) return false;
  const types = new Set(["text", "file", "link", "group"]);
  const ids = new Set(doc.nodes.map(n => n.id));
  if (new Set(doc.edges.map(e => e.id)).size !== doc.edges.length) return false;
  return doc.nodes.every(n => n.id && types.has(n.type)) &&
    doc.edges.every(e => e.id && ids.has(e.fromNode) && ids.has(e.toNode));
})()`;

// ---------------------------------------------------------------------------
// smoke subcommand: the baseline suite from AGENTS.md §13, automated.
// ---------------------------------------------------------------------------

async function smoke(url, flags) {
  const results = [];
  const record = (name, ok, detail = "") => { results.push({ name, ok, detail }); };
  const session = new BrowserSession({
    url,
    profile: flags.profile ? resolve(flags.profile) : null,
    width: Number(flags.width) || 1440,
    height: Number(flags.height) || 900,
  });
  await session.start();
  try {
    await session.navigate();
    await session.waitFor("window.orbitCanvas && document.querySelectorAll('.canvas-node').length > 0", 15000);

    // 1. Boot: no uncaught errors, no failed same-origin assets.
    const failedUrls = [];
    for (const id of session.failedRequests) failedUrls.push(id);
    record("boot: no uncaught console errors", session.consoleErrors.length === 0, session.consoleErrors.slice(0, 3).join(" | "));
    record("boot: no failed asset requests", failedUrls.length === 0, failedUrls.slice(0, 3).join(" | "));

    // 2. Every document node rendered as a card.
    const counts = await session.evaluate(`({ dom: document.querySelectorAll('.canvas-node').length, doc: window.orbitCanvas.getDocument().nodes.length })`);
    record("render: DOM cards match document nodes", counts.dom === counts.doc && counts.doc > 0, `${counts.dom}/${counts.doc}`);

    // 3. Canonical file index came up (in-memory index over the vault; no SQLite in canonical v1).
    try {
      await session.waitFor(`!document.getElementById("lifeIndexStatus").textContent.includes("Preparing")`, 12000);
      const status = await session.evaluate(`document.getElementById("lifeIndexStatus").textContent.trim()`);
      record("index: canonical files indexed", /Files\b/.test(status) && /\d+ indexed/.test(status) && !/unavailable|read-only/i.test(status), status);
    } catch {
      record("index: canonical files indexed", false, await session.evaluate(`document.getElementById("lifeIndexStatus").textContent.trim()`).catch(() => "unreadable"));
    }

    // 4. Clicking a card selects it and opens the inspector.
    const cardPoint = await session.evaluate(PROBE_VISIBLE_CARD);
    if (cardPoint) {
      await session.click(cardPoint.x, cardPoint.y);
      await new Promise(r => setTimeout(r, 300));
      const sel = await session.evaluate(`(() => {
        const node = document.querySelector('.canvas-node.selected');
        const bearing = node?.querySelector('.selection-bearing');
        const bStyle = bearing ? getComputedStyle(bearing) : null;
        const nodeStyle = node ? getComputedStyle(node) : null;
        const bracket = bearing?.querySelector('i');
        const bracketBg = bracket ? getComputedStyle(bracket).backgroundImage : '';
        // No connection dots may ride along on a selected card.
        const handles = node?.querySelector('.connection-handles');
        const hStyle = handles ? getComputedStyle(handles) : null;
        return {
          selected: !!node,
          inspector: document.querySelector('.app-shell')?.classList.contains('inspector-open') ?? false,
          bearingVisible: !!bStyle && bStyle.display !== 'none',
          hasBrackets: /gradient/.test(bracketBg),
          solidBorder: !!nodeStyle && nodeStyle.borderTopStyle !== 'none' && parseFloat(nodeStyle.borderTopWidth) > 0,
          handlesHiddenOnSelect: !!hStyle && (hStyle.opacity === '0' || hStyle.display === 'none'),
        };
      })()`);
      record("select: card selected + inspector open", sel.selected && sel.inspector);
      record("select: corner-bracket frame, no circles", sel.bearingVisible && sel.hasBrackets && sel.solidBorder && sel.handlesHiddenOnSelect, `frame:${sel.bearingVisible} brackets:${sel.hasBrackets} border:${sel.solidBorder} handlesHidden:${sel.handlesHiddenOnSelect}`);
      if (flags.screenshot) {
        const directory = resolve(flags.screenshot);
        mkdirSync(directory, { recursive: true });
        const file = join(directory, "selected-card.png");
        await session.screenshot(file, ".canvas-node.selected");
        record("select: screenshot captured", true, file);
      }

      // 5. An explicit portal double-click must navigate without creating nodes,
      // then restore the original canvas for the remaining smoke probes.
      const portal = await session.evaluate(`(() => {
        const originalCanvasId = window.orbitCanvas.getCurrentCanvas().id;
        const portalNode = window.orbitCanvas.createSubcanvas({ x: 0, y: 0 });
        const workspace = window.orbitCanvas.getWorkspace();
        const child = Object.values(workspace.canvases).find(canvas =>
          canvas.parentId === originalCanvasId && canvas.portalNodeId === portalNode.id);
        return {
          originalCanvasId,
          childCanvasId: child?.id || null,
          portalNodeId: portalNode.id,
          total: Object.values(workspace.canvases).reduce((sum, canvas) => sum + (canvas.document?.nodes?.length || 0), 0),
        };
      })()`);
      await new Promise(r => setTimeout(r, 200));
      await session.evaluate(`(() => {
        const target = document.querySelector('.canvas-node[data-id="${portal.portalNodeId}"] .node-content');
        const rect = target.getBoundingClientRect();
        target.dispatchEvent(new MouseEvent('dblclick', { bubbles: true, cancelable: true, detail: 2, clientX: rect.left + rect.width / 2, clientY: rect.top + rect.height / 2 }));
      })()`);
      await session.waitFor(`window.orbitCanvas.getCurrentCanvas().id === ${JSON.stringify(portal.childCanvasId)}`, 5000);
      const portalAfter = await session.evaluate(`(() => {
        const workspace = window.orbitCanvas.getWorkspace();
        return {
          canvasId: window.orbitCanvas.getCurrentCanvas().id,
          total: Object.values(workspace.canvases).reduce((sum, canvas) => sum + (canvas.document?.nodes?.length || 0), 0),
        };
      })()`);
      record("create: portal dblclick navigates without creating nodes", portal.childCanvasId !== null && portalAfter.canvasId === portal.childCanvasId && portalAfter.total === portal.total, `${portal.total} -> ${portalAfter.total}`);
      await session.evaluate(`window.orbitCanvas.switchCanvas(${JSON.stringify(portal.originalCanvasId)})`);
      await session.waitFor(`window.orbitCanvas.getCurrentCanvas().id === ${JSON.stringify(portal.originalCanvasId)}`, 5000);
      await new Promise(r => setTimeout(r, 350));
      record("create: portal probe restores original canvas", await session.evaluate(`window.orbitCanvas.getCurrentCanvas().id === ${JSON.stringify(portal.originalCanvasId)}`));

      // 6. Note tool: clicking inside a card must NOT create a card.
      const beforeTool = await session.evaluate("window.orbitCanvas.getDocument().nodes.length");
      await session.evaluate(`document.querySelector('.tool[data-tool="note"]').click()`);
      await session.evaluate(`(() => {
        const target = document.querySelector('.canvas-node:not(.group-node) .node-content');
        const rect = target.getBoundingClientRect();
        const options = { bubbles: true, cancelable: true, button: 0, buttons: 1, pointerId: 81, clientX: rect.left + rect.width / 2, clientY: rect.top + rect.height / 2 };
        target.dispatchEvent(new PointerEvent('pointerdown', options));
        window.dispatchEvent(new PointerEvent('pointerup', { ...options, buttons: 0 }));
      })()`);
      await new Promise(r => setTimeout(r, 400));
      const afterTool = await session.evaluate("window.orbitCanvas.getDocument().nodes.length");
      record("create: note tool on card creates nothing", afterTool === beforeTool, `${beforeTool} -> ${afterTool}`);
      await session.evaluate(`document.querySelector('.tool[data-tool="select"]').click()`);
    } else {
      record("select: found a visible card", false, "no fully visible card in viewport");
    }

    // 7. Double-clicking empty background still creates a note.
    const bgPoint = await session.evaluate(PROBE_BACKGROUND_POINT);
    if (bgPoint) {
      const before = await session.evaluate("window.orbitCanvas.getDocument().nodes.length");
      await session.dblclick(bgPoint.x, bgPoint.y);
      await new Promise(r => setTimeout(r, 500));
      const info = await session.evaluate(`(() => {
        const nodes = window.orbitCanvas.getDocument().nodes;
        const last = nodes[nodes.length - 1];
        return { count: nodes.length, lastText: last?.text ?? "" };
      })()`);
      record("create: dblclick on background creates a note", info.count === before + 1 && info.lastText.includes("New thought"), `${before} -> ${info.count}`);
    } else {
      record("create: found a background point", false, "viewport fully covered");
    }

    // 8. Live document stays valid JSON Canvas 1.0.
    record("export: document is valid JSON Canvas", await session.evaluate(PROBE_IS_CANVAS));
    await session.waitFor(`document.getElementById("saveState")?.textContent.includes("Saved locally")`, 10000);

    // 9. Controlled reload preserves the workspace (same profile).
    const titleBefore = await session.evaluate("window.orbitCanvas.getWorkspace().canvases[window.orbitCanvas.getWorkspace().rootId].title");
    const nodesBefore = await session.evaluate("window.orbitCanvas.getDocument().nodes.length");
    await session.reload();
    await session.waitFor("window.orbitCanvas && document.querySelectorAll('.canvas-node').length > 0", 15000);
    await new Promise(r => setTimeout(r, 800));
    const titleAfter = await session.evaluate("window.orbitCanvas.getWorkspace().canvases[window.orbitCanvas.getWorkspace().rootId].title");
    const nodesAfter = await session.evaluate("window.orbitCanvas.getDocument().nodes.length");
    record("persist: reload keeps title and node count", titleBefore === titleAfter && nodesBefore === nodesAfter, `${nodesBefore} -> ${nodesAfter}`);

    // 10. Offline reload renders the shell from the Service Worker cache.
    if (flags.offline) {
      await session.setOffline(true);
      await session.reload();
      await new Promise(r => setTimeout(r, 1500));
      const shellUp = await session.evaluate("!!document.querySelector('.canvas') && !!window.orbitCanvas").catch(() => false);
      record("offline: shell renders from cache", shellUp === true);
      await session.setOffline(false);
    }
  } finally {
    await session.close();
  }

  let failed = 0;
  for (const r of results) {
    if (!r.ok) failed++;
    console.log(`${r.ok ? "PASS" : "FAIL"}  ${r.name}${r.detail ? `  [${r.detail}]` : ""}`);
  }
  console.log(failed === 0 ? `\nAll ${results.length} checks passed.` : `\n${failed}/${results.length} checks FAILED.`);
  return failed === 0 ? 0 : 1;
}

// ---------------------------------------------------------------------------
// components subcommand: focused Custom Element behavior probes.
// ---------------------------------------------------------------------------

async function components(url, flags) {
  const results = [];
  const record = (name, ok, detail = "") => { results.push({ name, ok, detail }); };
  const session = new BrowserSession({
    url,
    profile: flags.profile ? resolve(flags.profile) : null,
    width: Number(flags.width) || 1440,
    height: Number(flags.height) || 900,
  });
  await session.start();
  try {
    await session.navigate();
    await session.waitFor("window.orbitCanvas", 15000);
    const probe = await session.evaluate(`(async () => {
      const host = document.querySelector("balaur-add-menu");
      const defined = customElements.get("balaur-add-menu");
      if (!host || !defined) {
        return { upgraded: false, hostFound: !!host, defined: !!defined };
      }
      const { defineElement } = await import("./elements/element-utils.js");
      const idempotentRegistration = defineElement("balaur-add-menu", class extends HTMLElement {}) === defined;
      const toggle = host.querySelector(".add-menu-toggle");
      const panel = host.querySelector(".add-menu-panel");
      const items = [...host.querySelectorAll("[data-add]")];
      const key = (target, value) => target.dispatchEvent(new KeyboardEvent("keydown", { key: value, bubbles: true, cancelable: true }));
      const closed = () => toggle.getAttribute("aria-expanded") === "false";

      host.open();
      const openFocusesFirst = toggle.getAttribute("aria-expanded") === "true" && document.activeElement === items[0];
      key(panel, "ArrowDown");
      const arrowDown = document.activeElement === items[1];
      key(panel, "ArrowUp");
      const arrowUp = document.activeElement === items[0];
      key(panel, "End");
      const end = document.activeElement === items.at(-1);
      key(panel, "Home");
      const home = document.activeElement === items[0];
      key(panel, "Escape");
      const escape = closed() && document.activeElement === toggle;

      toggle.focus();
      key(toggle, "ArrowDown");
      const toggleArrowDown = toggle.getAttribute("aria-expanded") === "true" && document.activeElement === items[0];
      document.body.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true }));
      const outsideDismissal = closed();

      host.open();
      host.close({ restoreFocus: true });
      const closeRestoresFocus = closed() && document.activeElement === toggle;

      const nodesBeforeInvalidIntent = window.orbitCanvas.getDocument().nodes.length;
      host.dispatchEvent(new CustomEvent("balaur-add", {
        bubbles: true,
        composed: true,
        detail: { kind: "toString" },
      }));
      const invalidIntentIgnored = window.orbitCanvas.getDocument().nodes.length === nodesBeforeInvalidIntent;

      const intents = [];
      host.addEventListener("balaur-add", event => {
        intents.push({
          kind: event.detail?.kind,
          bubbles: event.bubbles,
          composed: event.composed,
        });
        event.stopPropagation();
      });
      for (const item of items) {
        host.open();
        item.click();
      }
      const kinds = items.map(item => item.dataset.add);
      const everyKind = intents.length === kinds.length
        && intents.every((intent, index) => intent.kind === kinds[index] && intent.bubbles && intent.composed)
        && closed();

      host.open();
      const toggleRect = toggle.getBoundingClientRect();
      const panelRect = panel.getBoundingClientRect();
      const narrow = innerWidth <= 620;
      const placement = narrow
        ? panelRect.bottom <= toggleRect.top
        : panelRect.top >= toggleRect.bottom;
      const panelInViewport = panelRect.left >= 0 && panelRect.right <= innerWidth
        && panelRect.top >= 0 && panelRect.bottom <= innerHeight;
      const progressiveMode = "popover" in HTMLElement.prototype
        ? panel.hasAttribute("popover") && panel.matches(":popover-open")
        : !panel.hasAttribute("popover") && !panel.hidden;
      host.close({ restoreFocus: false });

      const fixture = document.createElement("balaur-add-menu");
      fixture.innerHTML = '<button class="add-menu-toggle" aria-expanded="false">+</button><div class="add-menu-panel" hidden><button data-add="note">Note</button></div>';
      document.body.append(fixture);
      fixture.remove();
      document.body.append(fixture);
      let reconnectIntents = 0;
      fixture.addEventListener("balaur-add", event => {
        reconnectIntents++;
        event.stopPropagation();
      });
      fixture.open();
      fixture.querySelector("[data-add]").click();
      fixture.remove();

      const fallback = document.createElement("balaur-add-menu");
      fallback.innerHTML = '<button class="add-menu-toggle" aria-expanded="false">+</button><div class="add-menu-panel" hidden><button data-add="note">Note</button></div>';
      const fallbackPanel = fallback.querySelector(".add-menu-panel");
      fallbackPanel.showPopover = undefined;
      fallbackPanel.hidePopover = undefined;
      document.body.append(fallback);
      fallback.open();
      const fallbackOpened = !fallbackPanel.hidden && fallback.querySelector(".add-menu-toggle").getAttribute("aria-expanded") === "true";
      fallback.close();
      const fallbackClosed = fallbackPanel.hidden && fallback.querySelector(".add-menu-toggle").getAttribute("aria-expanded") === "false";
      fallback.remove();

      return {
        upgraded: host instanceof defined,
        hostFound: true,
        defined: true,
        idempotentRegistration,
        nativeControls: toggle instanceof HTMLButtonElement
          && items.length > 0
          && items.every(item => item instanceof HTMLButtonElement),
        openFocusesFirst,
        arrowDown,
        arrowUp,
        home,
        end,
        escape,
        toggleArrowDown,
        outsideDismissal,
        closeRestoresFocus,
        invalidIntentIgnored,
        everyKind,
        kinds,
        intents: intents.map(intent => intent.kind),
        placement,
        panelInViewport,
        progressiveMode,
        fallbackWorks: fallbackOpened && fallbackClosed,
        reconnectsCleanly: reconnectIntents === 1,
        narrow,
      };
    })()`);

    record("components: balaur-add-menu upgrades", probe.upgraded, `host:${probe.hostFound} defined:${probe.defined}`);
    if (probe.upgraded) {
      record("components: registration is idempotent", probe.idempotentRegistration);
      record("components: light DOM keeps native buttons", probe.nativeControls);
      record("components: open focuses first item", probe.openFocusesFirst);
      record("components: ArrowDown moves focus", probe.arrowDown);
      record("components: ArrowUp moves focus", probe.arrowUp);
      record("components: Home moves to first item", probe.home);
      record("components: End moves to last item", probe.end);
      record("components: Escape closes and returns focus", probe.escape);
      record("components: toggle ArrowDown opens menu", probe.toggleArrowDown);
      record("components: outside pointer dismisses", probe.outsideDismissal);
      record("components: close API returns focus", probe.closeRestoresFocus);
      record("components: invalid add intent cannot mutate the document", probe.invalidIntentIgnored);
      record("components: every add kind emits one composed intent", probe.everyKind, `${probe.intents?.join(",")} / ${probe.kinds?.join(",")}`);
      record(`components: ${probe.narrow ? "narrow" : "wide"} placement is visible`, probe.placement && probe.panelInViewport);
      record("components: Popover or hidden fallback is active", probe.progressiveMode);
      record("components: hidden-panel fallback opens and closes", probe.fallbackWorks);
      record("components: reconnect does not duplicate listeners", probe.reconnectsCleanly);
    }
    const integrationSetup = await session.evaluate(`(async () => {
      await window.orbitVaultReady;
      const path = "cards/browser-component-probe.md";
      const invalidPath = "cards/browser-invalid-component-probe.md";
      const nodeId = "browser-component-probe-node";
      const invalidNodeId = "browser-invalid-component-probe-node";
      const iframeNodeId = "browser-iframe-probe-node";
      const vault = window.orbitVaultStore?.vault;
      if (!vault) return { ready: false };
      if (!await vault.stat(path)) {
        const { serializeComponentCard } = await import("./storage/component-card-codec.js");
        await vault.write(path, serializeComponentCard({
          id: "card-browser-component-probe",
          title: "Catalog boot probe",
          recipe: "metric",
          value: "88%",
          label: "Loaded synchronously",
          progress: 0.88,
          trend: "up",
          body: "Rendered from the boot catalog.",
        }), { expectedHash: null });
      }
      if (!await vault.stat(invalidPath)) {
        await vault.write(invalidPath, "---\\norbit-schema: 1\\norbit-type: component-card\\norbit-id: card-browser-invalid\\ntitle: \\"Recoverable invalid card\\"\\nrecipe: progress\\nvalue: 12\\nmaximum: 10\\n---\\nRecognizable **raw fallback body**.", { expectedHash: null });
      }
      const existingIds = new Set(window.orbitCanvas.getDocument().nodes.map(node => node.id));
      const operations = [];
      if (!existingIds.has(nodeId)) operations.push({
        type: "node.add",
        node: { id: nodeId, type: "file", file: path, x: 1420, y: 0, width: 330, height: 230, color: "5" },
      });
      if (!existingIds.has(invalidNodeId)) operations.push({
        type: "node.add",
        node: { id: invalidNodeId, type: "file", file: invalidPath, x: 1420, y: 260, width: 330, height: 230, color: "2" },
      });
      if (!existingIds.has(iframeNodeId)) operations.push({
        type: "node.add",
        node: { id: iframeNodeId, type: "file", file: "widgets/focus-orbit.html", x: 1780, y: 0, width: 420, height: 260, color: "5" },
      });
      if (operations.length) window.orbitCanvas.applyOperations(operations);
      await new Promise(resolve => setTimeout(resolve, 700));
      return { ready: true, nodeId, invalidNodeId, iframeNodeId };
    })()`);
    if (integrationSetup.ready) {
      await session.reload();
      await session.waitFor("window.orbitCanvas", 15000);
      const integrationProbe = await session.evaluate(`(() => {
        const selector = '.canvas-node[data-id="${integrationSetup.nodeId}"] balaur-component-card';
        const invalidSelector = '.canvas-node[data-id="${integrationSetup.invalidNodeId}"] balaur-component-card';
        const iframeSelector = '.canvas-node[data-id="${integrationSetup.iframeNodeId}"] balaur-widget-frame';
        const host = document.querySelector(selector);
        const invalidHost = document.querySelector(invalidSelector);
        const iframe = document.querySelector(iframeSelector);
        const standardFileNode = window.orbitCanvas.getDocument().nodes.some(node =>
          node.id === "${integrationSetup.nodeId}"
          && node.type === "file"
          && node.file === "cards/browser-component-probe.md"
        );
        const vault = window.orbitVaultStore.vault;
        const originalRead = vault.read;
        let renderReads = 0;
        vault.read = function(...args) {
          renderReads++;
          return originalRead.apply(this, args);
        };
        window.orbitCanvas.applyOperations([]);
        vault.read = originalRead;
        const hostAfterRender = document.querySelector(selector);
        const iframeAfterRender = document.querySelector(iframeSelector);
        hostAfterRender?.querySelector("[data-card-open]")?.click();
        return {
          rendered: host?.model?.title === "Catalog boot probe"
            && host.dataset.nodeId === "${integrationSetup.nodeId}"
            && host.textContent.includes("Loaded synchronously"),
          fallback: invalidHost?.model?.title === "Recoverable invalid card"
            && invalidHost.textContent.includes("Recognizable raw fallback body.")
            && invalidHost.querySelector("[role='status']")?.textContent.includes("Malformed component card"),
          synchronous: renderReads === 0,
          standardFileNode,
          stableRenderRoots: !!iframe
            && hostAfterRender === host
            && document.querySelector(selector) === host
            && iframeAfterRender === iframe,
        };
      })()`);
      record("components: boot catalog renders a card file synchronously", integrationProbe.rendered);
      record("components: malformed catalog entry preserves readable raw fallback", integrationProbe.fallback);
      record("components: normal renders perform no vault reads", integrationProbe.synchronous);
      record("components: Canvas placement remains a standard file node", integrationProbe.standardFileNode);
      record("components: normal rerenders preserve card hosts and deterministic iframe", integrationProbe.stableRenderRoots);
      const generatedCardProbe = await session.evaluate(`(async () => {
        try {
          const cardId = "card-browser-component-probe";
          const secondNodeId = "browser-component-probe-node-second";
          const canvasId = window.orbitCanvas.getCurrentCanvas().id;
          if (!window.orbitCanvas.getDocument().nodes.some(node => node.id === secondNodeId)) {
            await window.orbitCanvas.applyOperations([{
              type: "component-card.update",
              id: cardId,
              patch: {},
              canvasId,
              placement: { id: secondNodeId, x: 1420, y: 520, width: 330, height: 230, color: "5" },
            }]);
          }
          const patchOnly = {
            type: "component-card.update",
            id: cardId,
            patch: { fields: { value: "93%", label: "Patched everywhere", progress: 0.93, trend: "up" } },
          };
          const proposal = window.orbitCanvas.validateOperations([patchOnly]);
          await window.orbitCanvas.applyOperations([patchOnly]);
          const placements = window.orbitCanvas.getDocument().nodes.filter(node => node.file === "cards/browser-component-probe.md");
          const patchedHosts = placements.map(node => document.querySelector('.canvas-node[data-id="' + CSS.escape(node.id) + '"] balaur-component-card'));
          patchedHosts[0]?.querySelector("[data-card-open]")?.click();
          const deleteCard = document.querySelector('#inspector [data-intent="delete-card"]');
          const deletePlacement = document.querySelector('#inspector [data-intent="delete-selection"]');
          const originalConfirm = window.confirm;
          window.confirm = () => false;
          deleteCard?.click();
          await new Promise(resolve => setTimeout(resolve, 200));
          const vault = window.orbitVaultStore.vault;
          const cancelled = !!await vault.stat("cards/browser-component-probe.md")
            && window.orbitCanvas.getDocument().nodes.filter(node => node.file === "cards/browser-component-probe.md").length === 2;
          window.confirm = () => true;
          deleteCard?.click();
          for (let index = 0; index < 50 && await vault.stat("cards/browser-component-probe.md"); index++) {
            await new Promise(resolve => setTimeout(resolve, 100));
          }
          window.confirm = originalConfirm;
          return {
            proposalPatchOnly: proposal.normalizedOperations?.length === 1
              && !Object.hasOwn(proposal.normalizedOperations[0], "placement"),
            patchedAcrossPlacements: placements.length === 2
              && patchedHosts.every(host => host?.model?.value === "93%" && host.textContent.includes("Patched everywhere")),
            separateDeleteActions: !!deleteCard && !!deletePlacement && deleteCard !== deletePlacement,
            cancelled,
            deleted: !await vault.stat("cards/browser-component-probe.md")
              && !window.orbitCanvas.getDocument().nodes.some(node => node.file === "cards/browser-component-probe.md")
              && !document.querySelector('balaur-component-card[data-node-id="${integrationSetup.nodeId}"]'),
          };
        } catch (error) {
          return { error: error.message };
        }
      })()`);
      record("components: patch-only card update proposes no placement", generatedCardProbe.proposalPatchOnly, generatedCardProbe.error || "");
      record("components: patch-only card update refreshes every placement", generatedCardProbe.patchedAcrossPlacements, generatedCardProbe.error || "");
      record("components: card and placement deletion remain separate actions", generatedCardProbe.separateDeleteActions, generatedCardProbe.error || "");
      record("components: cancelling card deletion preserves file and placements", generatedCardProbe.cancelled, generatedCardProbe.error || "");
      record("components: confirmed card deletion removes file and every placement", generatedCardProbe.deleted, generatedCardProbe.error || "");
    } else {
      record("components: boot catalog renders a card file synchronously", false, "vault unavailable");
      record("components: malformed catalog entry preserves readable raw fallback", false, "vault unavailable");
      record("components: normal renders perform no vault reads", false, "vault unavailable");
      record("components: Canvas placement remains a standard file node", false, "vault unavailable");
      record("components: normal rerenders preserve card hosts and deterministic iframe", false, "vault unavailable");
    }
    const applicationSetup = await session.evaluate(`(async () => {
      const today = (() => { const date = new Date(); return date.getFullYear() + "-" + String(date.getMonth() + 1).padStart(2, "0") + "-" + String(date.getDate()).padStart(2, "0"); })();
      const title = "Task 7 component flow";
      if (!document.body.textContent.includes(title)) await window.orbitCanvas.createTask({ title, status: "scheduled", scheduledOn: today });
      window.orbitCanvas.setView("today");
      return { title };
    })()`);
    await session.waitFor(`document.querySelector('#todayScheduled [data-task-id] .task-copy')?.textContent.includes(${JSON.stringify(applicationSetup.title)})`, 10000);
    const todayBefore = await session.evaluate(`(() => {
      const row = [...document.querySelectorAll("#todayScheduled [data-task-id]")].find(node => node.textContent.includes(${JSON.stringify(applicationSetup.title)}));
      const open = row?.querySelector("[data-open-task]");
      open?.focus();
      window.orbitCanvas.setView("today");
      const currentRow = [...document.querySelectorAll("#todayScheduled [data-task-id]")].find(node => node.textContent.includes(${JSON.stringify(applicationSetup.title)}));
      const stable = currentRow === row && currentRow?.querySelector("[data-open-task]") === open && document.activeElement === open;
      currentRow?.querySelector("[data-complete-task]")?.click();
      return {
        hosts: ["todayScheduled", "todayOverdue", "todayQueue", "todayCompleted"].every(id => document.getElementById(id)?.matches("balaur-task-list")),
        stable,
      };
    })()`);
    await session.waitFor(`document.querySelector('#todayCompleted')?.textContent.includes(${JSON.stringify(applicationSetup.title)})`, 10000);
    record("components: Today uses task-list hosts", todayBefore.hosts);
    record("components: Today rerender preserves focused keyed task row", todayBefore.stable);
    record("components: task completion moves canonical task into Completed", true);

    await session.evaluate(`document.getElementById("newTodayTask").click()`);
    await session.waitFor(`document.getElementById("taskDialog").open && document.activeElement === document.getElementById("taskTitle")`, 3000);
    const dialogCancel = await session.evaluate(`(() => {
      const dialog = document.getElementById("taskDialog");
      const native = dialog instanceof HTMLDialogElement
        && dialog.querySelector("form") instanceof HTMLFormElement
        && dialog.querySelector("balaur-dialog-frame");
      document.getElementById("cancelTaskDialog").click();
      return { native: !!native, closed: !dialog.open };
    })()`);
    await session.evaluate(`document.getElementById("newTodayTask").click()`);
    await session.waitFor(`document.getElementById("taskDialog").open`, 3000);
    const submittedTitle = "Task 7 dialog submit";
    await session.evaluate(`(() => {
      document.getElementById("taskTitle").value = ${JSON.stringify(submittedTitle)};
      document.getElementById("taskForm").requestSubmit(document.getElementById("createTaskButton"));
    })()`);
    await session.waitFor(`!document.getElementById("taskDialog").open && document.getElementById("todayScheduled").textContent.includes(${JSON.stringify(submittedTitle)})`, 10000);
    record("components: native dialog frame focuses and cancels", dialogCancel.native && dialogCancel.closed);
    record("components: native dialog form submits through app controller", true);

    const navigationSetup = await session.evaluate(`(() => {
      window.orbitCanvas.setView("canvas");
      const parentId = window.orbitCanvas.getCurrentCanvas().id;
      const before = new Set(Object.keys(window.orbitCanvas.getWorkspace().canvases));
      window.orbitCanvas.createSubcanvas();
      const childId = Object.keys(window.orbitCanvas.getWorkspace().canvases).find(id => !before.has(id));
      window.orbitCanvas.switchCanvas(childId);
      return { parentId, childId };
    })()`);
    await session.waitFor(`window.orbitCanvas.getCurrentCanvas().id === ${JSON.stringify(navigationSetup.childId)}`, 5000);
    const breadcrumb = await session.evaluate(`(() => {
      const trail = document.getElementById("canvasBreadcrumbs");
      const list = document.getElementById("canvasList");
      const parent = trail.querySelector('[data-canvas-id="${navigationSetup.parentId}"]');
      const hosts = trail.matches("balaur-workspace-nav") && list.matches("balaur-workspace-nav");
      const landmarks = trail.closest("nav")?.getAttribute("aria-label")
        && list.closest("nav")?.getAttribute("aria-label");
      parent.click();
      return { hosts, landmarks: !!landmarks, clicked: true };
    })()`);
    await session.waitFor(`window.orbitCanvas.getCurrentCanvas().id === ${JSON.stringify(navigationSetup.parentId)}`, 5000);
    record("components: workspace navigation hosts render hierarchy and breadcrumbs", breadcrumb.hosts);
    record("components: workspace navigation retains named native landmarks", breadcrumb.landmarks);
    record("components: breadcrumb composed intent switches canvas", breadcrumb.clicked);

    const inspectorSetup = await session.evaluate(`(() => {
      const id = "task-7-inspector-probe";
      if (!window.orbitCanvas.getDocument().nodes.some(node => node.id === id)) {
        window.orbitCanvas.applyOperations([{ type: "node.add", node: { id, type: "text", x: 20, y: 20, width: 260, height: 160, color: "2", text: "# Inspector probe" } }]);
      }
      return { id };
    })()`);
    await session.waitFor(`document.querySelector('.canvas-node[data-id="${inspectorSetup.id}"]')`, 5000);
    await session.evaluate(`(() => {
      const node = document.querySelector('.canvas-node[data-id="${inspectorSetup.id}"]');
      node.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, pointerId: 71, clientX: 30, clientY: 30 }));
      window.dispatchEvent(new PointerEvent("pointerup", { bubbles: true, button: 0, pointerId: 71, clientX: 30, clientY: 30 }));
    })()`);
    await session.waitFor(`document.querySelector('#inspector [data-field-key="text"]')`, 5000);
    await session.evaluate(`new Promise(resolve => setTimeout(resolve, 450))`);
    const inspectorFlow = await session.evaluate(`(() => {
      const inspector = document.getElementById("inspector");
      const field = inspector.querySelector('[data-field-key="text"]');
      field.focus();
      field.value = "# Inspector edited";
      field.dispatchEvent(new InputEvent("input", { bubbles: true, inputType: "insertText", data: "d" }));
      window.orbitCanvas.applyOperations([]);
      const fieldAfter = inspector.querySelector('[data-field-key="text"]');
      const edited = window.orbitCanvas.getDocument().nodes.find(node => node.id === ${JSON.stringify(inspectorSetup.id)})?.text === "# Inspector edited";
      const rect = inspector.getBoundingClientRect();
      return {
        host: inspector.matches("balaur-inspector"),
        edited,
        stable: fieldAfter === field && document.activeElement === field && fieldAfter.value === "# Inspector edited",
        landmark: inspector.getAttribute("role") === "complementary"
          && !!inspector.getAttribute("aria-label"),
        visible: innerWidth > 850 || (rect.left >= 0 && rect.right <= innerWidth && rect.top >= 0 && rect.bottom <= innerHeight),
        narrow: innerWidth <= 850,
      };
    })()`);
    record("components: inspector retains a named complementary landmark", inspectorFlow.landmark);
    record("components: inspector host emits edits to app controller", inspectorFlow.host && inspectorFlow.edited);
    record("components: inspector rerender preserves focused field DOM", inspectorFlow.stable);
    record(`components: inspector ${inspectorFlow.narrow ? "narrow sheet" : "wide panel"} remains visible`, inspectorFlow.visible);
    const componentFixtureUrl = new URL(".pi/skills/browser-check/fixtures/components.html", url).href;
    await session.navigate(componentFixtureUrl);
    await session.waitFor("window.componentFixtureReady", 10000);
    const cardProbe = await session.evaluate(`(() => {
      const recipes = ["metric", "progress", "callout", "list", "timeline"];
      const hosts = Object.fromEntries(recipes.map(recipe => [recipe, document.getElementById(recipe)]));
      const defined = customElements.get("balaur-component-card");
      if (!defined) return { defined: false };
      const renderedRecipes = recipes.every(recipe => {
        const host = hosts[recipe];
        const region = host?.querySelector('[data-recipe-region="' + recipe + '"]');
        return host instanceof defined && host.dataset.recipe === recipe && region && !region.hidden;
      });
      const nativeSemantics = hosts.metric.querySelector("h2") instanceof HTMLHeadingElement
        && hosts.metric.querySelector("button[data-card-open]") instanceof HTMLButtonElement
        && hosts.progress.querySelector("progress") instanceof HTMLProgressElement
        && hosts.list.querySelector("ul") instanceof HTMLUListElement
        && hosts.timeline.querySelector("ol") instanceof HTMLOListElement
        && hosts.timeline.querySelector("time[datetime='2026-07-22']") instanceof HTMLTimeElement;
      const safeLink = hosts.callout.querySelector('a[href="https://example.com/"]');
      const markdownSanitized = !window.fixtureInjected
        && !hosts.callout.querySelector("script")
        && !hosts.callout.querySelector('a[href^="javascript:"]')
        && safeLink?.target === "_blank"
        && safeLink?.rel === "noreferrer";
      const fallbackReadable = ["invalid", "unknown"].every(id => {
        const host = document.getElementById(id);
        return host.dataset.recipe === "fallback"
          && host.querySelector("[role='status']")?.textContent.trim()
          && host.textContent.includes(id === "invalid" ? "Retained readable context." : "Readable fallback body.");
      });
      const longContent = hosts.list.textContent.includes("Long item 40")
        && hosts.list.scrollWidth <= hosts.list.clientWidth;
      const emptyContent = document.getElementById("empty").textContent.includes("Empty context");

      const callout = hosts.callout;
      const calloutFrame = callout.firstElementChild;
      const calloutParagraph = safeLink?.parentElement;
      safeLink?.focus();
      const unchangedCalloutModel = callout.model;
      callout.model = unchangedCalloutModel;
      const unchangedAssignmentNoop = callout.model === unchangedCalloutModel
        && callout.firstElementChild === calloutFrame
        && callout.querySelector('a[href="https://example.com/"]') === safeLink
        && safeLink?.parentElement === calloutParagraph
        && document.activeElement === safeLink;
      callout.model = Object.freeze({ ...callout.model, body: callout.model.body + "\\nA reconciled tail paragraph." });
      const markdownReconciled = callout.querySelector('a[href="https://example.com/"]') === safeLink
        && safeLink?.parentElement === calloutParagraph
        && document.activeElement === safeLink
        && callout.textContent.includes("A reconciled tail paragraph.");
      const firstListItem = hosts.list.querySelector("li");
      hosts.list.model = Object.freeze({ ...hosts.list.model, body: hosts.list.model.body + "\\n- Reconciled list tail" });
      const listReconciled = hosts.list.querySelector("li") === firstListItem
        && hosts.list.textContent.includes("Reconciled list tail");
      const firstTimelineItem = hosts.timeline.querySelector("li");
      const firstTimelineTime = hosts.timeline.querySelector("time");
      hosts.timeline.model = Object.freeze({ ...hosts.timeline.model, body: hosts.timeline.model.body + "\\n- 2026-07-30 Reconciled timeline tail" });
      const timelineReconciled = hosts.timeline.querySelector("li") === firstTimelineItem
        && hosts.timeline.querySelector("time") === firstTimelineTime
        && hosts.timeline.textContent.includes("Reconciled timeline tail");

      const metric = hosts.metric;
      const frame = metric.firstElementChild;
      const title = metric.querySelector("h2");
      const openButton = metric.querySelector("[data-card-open]");
      let assignmentEvents = 0;
      metric.addEventListener("balaur-card-open", () => assignmentEvents++);
      metric.model = Object.freeze({ ...metric.model, title: "Updated weekly focus", value: "73%" });
      metric.placementColor = "5";
      const stableDom = metric.firstElementChild === frame
        && metric.querySelector("h2") === title
        && metric.querySelector("[data-card-open]") === openButton
        && title.textContent === "Updated weekly focus";
      const noEventOnAssignment = assignmentEvents === 0;
      let openIntent = null;
      document.addEventListener("balaur-card-open", event => {
        openIntent = {
          detail: event.detail,
          bubbles: event.bubbles,
          composed: event.composed,
        };
      }, { once: true });
      openButton.click();
      const composedOpenEvent = assignmentEvents === 1
        && openIntent?.bubbles
        && openIntent?.composed
        && openIntent.detail?.cardId === "card-metric"
        && openIntent.detail?.path === "cards/card-metric.md"
        && openIntent.detail?.nodeId === "node-metric";
      const placementColor = metric.dataset.placementColor === "5"
        && getComputedStyle(metric).getPropertyValue("--component-card-accent").trim().length > 0;
      const narrow = document.getElementById("empty");
      const narrowLayout = getComputedStyle(narrow.querySelector(".component-card-layout")).gridTemplateColumns.split(" ").length === 1;
      return {
        defined: true,
        renderedRecipes,
        nativeSemantics,
        markdownSanitized,
        fallbackReadable,
        longContent,
        emptyContent,
        stableDom,
        unchangedAssignmentNoop,
        markdownReconciled,
        listReconciled,
        timelineReconciled,
        noEventOnAssignment,
        composedOpenEvent,
        placementColor,
        narrowLayout,
      };
    })()`);
    record("components: balaur-component-card upgrades", cardProbe.defined);
    if (cardProbe.defined) {
      record("components: all five recipes render", cardProbe.renderedRecipes);
      record("components: recipes retain native semantics", cardProbe.nativeSemantics);
      record("components: Markdown is sanitized", cardProbe.markdownSanitized);
      record("components: invalid and unknown recipes stay readable", cardProbe.fallbackReadable);
      record("components: long and empty data render safely", cardProbe.longContent && cardProbe.emptyContent);
      record("components: property updates preserve stable DOM", cardProbe.stableDom);
      record("components: unchanged model assignment is a DOM no-op", cardProbe.unchangedAssignmentNoop);
      record("components: focused Markdown descendants reconcile in place", cardProbe.markdownReconciled);
      record("components: list and timeline descendants reconcile in place", cardProbe.listReconciled && cardProbe.timelineReconciled);
      record("components: assignment emits no open intent", cardProbe.noEventOnAssignment);
      record("components: user open intent bubbles and composes", cardProbe.composedOpenEvent);
      record("components: placement color reaches component tokens", cardProbe.placementColor);
      record("components: named narrow container adapts layout", cardProbe.narrowLayout);
    }
    const hostProbe = await session.evaluate(`(() => {
      const TaskList = customElements.get("balaur-task-list");
      const WorkspaceNav = customElements.get("balaur-workspace-nav");
      const DialogFrame = customElements.get("balaur-dialog-frame");
      const Inspector = customElements.get("balaur-inspector");
      if (!TaskList || !WorkspaceNav || !DialogFrame || !Inspector) {
        return {
          defined: false,
          definitions: {
            taskList: !!TaskList,
            workspaceNav: !!WorkspaceNav,
            dialogFrame: !!DialogFrame,
            inspector: !!Inspector,
          },
        };
      }

      const taskList = document.getElementById("taskListFixture");
      const alphaRow = taskList.querySelector('[data-task-id="task-alpha"]');
      const alphaOpen = alphaRow.querySelector("[data-open-task]");
      alphaOpen.focus();
      taskList.items = [
        { id: "task-beta", title: "Beta task", context: "Projects", status: "done", scheduledOn: null, dueOn: "2026-07-21" },
        { id: "task-alpha", title: "Alpha task updated", context: "Root", status: "next", scheduledOn: "2026-07-22", dueOn: null },
      ];
      const taskRowsKeyed = taskList.querySelector('[data-task-id="task-alpha"]') === alphaRow
        && taskList.querySelector('[data-task-id="task-alpha"] [data-open-task]') === alphaOpen
        && document.activeElement === alphaOpen
        && alphaOpen.textContent.includes("Alpha task updated");
      const taskIntents = [];
      taskList.addEventListener("balaur-task-complete", event => {
        taskIntents.push({ type: event.type, detail: event.detail, bubbles: event.bubbles, composed: event.composed });
      });
      taskList.addEventListener("balaur-task-open", event => {
        taskIntents.push({ type: event.type, detail: event.detail, bubbles: event.bubbles, composed: event.composed });
      });
      taskList.querySelector('[data-task-id="task-alpha"] [data-complete-task]').click();
      alphaOpen.click();
      taskList.remove();
      document.querySelector(".host-fixture").append(taskList);
      taskList.querySelector('[data-task-id="task-alpha"] [data-open-task]').click();
      const taskEvents = taskIntents.length === 3
        && taskIntents.every(event => event.detail?.taskId === "task-alpha" && event.bubbles && event.composed)
        && taskIntents.filter(event => event.type === "balaur-task-open").length === 2;

      const nav = document.getElementById("navFixture");
      const rootButton = nav.querySelector('[data-canvas-id="root"]');
      rootButton.focus();
      nav.canvases = [
        { id: "project", title: "Project renamed", depth: 1, count: 3, icon: "↳" },
        { id: "root", title: "Root", depth: 0, count: 4, icon: "◫" },
      ];
      const navKeyed = nav.querySelector('[data-canvas-id="root"]') === rootButton
        && document.activeElement === rootButton;
      const projectCurrentBefore = nav.querySelector('[data-canvas-id="project"]')?.getAttribute("aria-current") === "page"
        && !rootButton.hasAttribute("aria-current");
      nav.activeId = "root";
      const canvasCurrentChanges = rootButton.getAttribute("aria-current") === "page"
        && !nav.querySelector('[data-canvas-id="project"]')?.hasAttribute("aria-current");
      let canvasIntents = 0;
      let canvasIntent = null;
      nav.addEventListener("balaur-canvas-open", event => {
        canvasIntents++;
        canvasIntent = event;
      });
      nav.remove();
      document.querySelectorAll(".host-fixture")[1].append(nav);
      rootButton.click();
      const trailCurrent = document.querySelector('#trailFixture [data-canvas-id="project"][aria-current="page"]');
      const navEvents = canvasIntents === 1
        && canvasIntent?.detail?.canvasId === "root"
        && canvasIntent.bubbles
        && canvasIntent.composed
        && !!trailCurrent;

      const inspector = document.getElementById("inspectorFixture");
      const textField = inspector.querySelector('[data-field-key="text"]');
      textField.focus();
      textField.value = "# Typed fixture";
      const nextInspectorModel = {
        ...inspector.model,
        fields: inspector.model.fields.map(field => field.key === "text"
          ? { ...field, value: "# Typed fixture" }
          : { ...field, value: 20 }),
      };
      inspector.model = nextInspectorModel;
      const inspectorKeyed = inspector.querySelector('[data-field-key="text"]') === textField
        && document.activeElement === textField
        && textField.value === "# Typed fixture";
      const numberField = inspector.querySelector('[data-field-key="x"]');
      inspector.model = {
        ...nextInspectorModel,
        key: "node:other",
        fields: nextInspectorModel.fields.map(field => field.key === "text"
          ? { ...field, value: "# Other selection" }
          : { ...field, value: 30 }),
      };
      const inspectorSelectionPatch = inspector.querySelector('[data-field-key="text"]') === textField
        && inspector.querySelector('[data-field-key="x"]') === numberField
        && textField.value === "# Other selection"
        && numberField.value === "30";
      textField.value = "# Typed fixture";
      const inspectorIntents = [];
      for (const type of ["balaur-inspector-field-input", "balaur-inspector-field-change", "balaur-inspector-color", "balaur-inspector-action", "balaur-inspector-close"]) {
        inspector.addEventListener(type, event => inspectorIntents.push({
          type,
          detail: event.detail,
          bubbles: event.bubbles,
          composed: event.composed,
        }));
      }
      textField.dispatchEvent(new InputEvent("input", { bubbles: true, inputType: "insertText", data: "x" }));
      textField.dispatchEvent(new Event("change", { bubbles: true }));
      inspector.remove();
      document.querySelectorAll(".host-fixture")[2].append(inspector);
      inspector.querySelector('[data-color="1"]').click();
      inspector.querySelector('[data-intent="delete"]').click();
      inspector.querySelector("[data-inspector-close]").click();
      const inspectorEvents = inspectorIntents.length === 5
        && inspectorIntents.every(event => event.bubbles && event.composed)
        && inspectorIntents[0].detail?.key === "text"
        && inspectorIntents[0].detail?.value === "# Typed fixture"
        && inspectorIntents[2].detail?.value === "1"
        && inspectorIntents[3].detail?.intent === "delete";

      const dialog = document.getElementById("dialogFixture");
      const dialogFrame = dialog.querySelector("balaur-dialog-frame");
      const form = dialog.querySelector("form");
      let submits = 0;
      form.addEventListener("submit", () => submits++);
      dialog.showModal();
      const dialogInput = document.getElementById("dialogFixtureInput");
      dialogInput.value = "Native";
      dialogInput.focus();
      const dialogNative = dialog instanceof HTMLDialogElement
        && form instanceof HTMLFormElement
        && document.activeElement === dialogInput
        && dialogFrame.querySelector("form") === form;
      form.requestSubmit(form.querySelector('button[value="submit"]'));
      const dialogSubmit = !dialog.open && dialog.returnValue === "submit" && submits === 1;
      dialogFrame.remove();
      dialog.append(dialogFrame);
      dialog.showModal();
      form.requestSubmit(form.querySelector('button[value="cancel"]'));
      const dialogReconnect = !dialog.open && dialog.returnValue === "cancel" && submits === 2;

      const narrow = innerWidth <= 620;
      const inspectorRect = inspector.getBoundingClientRect();
      const taskRect = taskList.getBoundingClientRect();
      return {
        defined: true,
        taskRowsKeyed,
        taskEvents,
        navKeyed,
        navEvents,
        canvasCurrent: projectCurrentBefore && canvasCurrentChanges,
        inspectorKeyed,
        inspectorSelectionPatch,
        inspectorEvents,
        dialogNative,
        dialogSubmit,
        dialogReconnect,
        nativeControls: taskList.querySelectorAll("button").length >= 4
          && nav.querySelectorAll("button").length === 2
          && inspector.querySelector('[data-field-key="text"]') instanceof HTMLTextAreaElement,
        responsiveVisible: inspectorRect.left >= 0
          && inspectorRect.right <= innerWidth
          && taskRect.left >= 0
          && taskRect.right <= innerWidth,
        narrow,
      };
    })()`);
    record("components: bounded host elements upgrade", hostProbe.defined, JSON.stringify(hostProbe.definitions || {}));
    if (hostProbe.defined) {
      record("components: task rows reconcile by key and preserve focus", hostProbe.taskRowsKeyed);
      record("components: task controls emit one composed intent after reconnect", hostProbe.taskEvents);
      record("components: workspace navigation reconciles by key and preserves focus", hostProbe.navKeyed);
      record("components: workspace navigation emits one composed intent after reconnect", hostProbe.navEvents);
      record("components: canvas list maintains one aria-current page across current changes", hostProbe.canvasCurrent);
      record("components: inspector fields reconcile by key and preserve focus", hostProbe.inspectorKeyed);
      record("components: inspector model-key change patches focused reused controls", hostProbe.inspectorSelectionPatch);
      record("components: inspector controls emit field-specific composed intents after reconnect", hostProbe.inspectorEvents);
      record("components: dialog frame preserves native dialog, form, and focus", hostProbe.dialogNative);
      record("components: dialog form submits once with native return value", hostProbe.dialogSubmit);
      record("components: dialog frame reconnect preserves native cancel submission", hostProbe.dialogReconnect);
      record("components: bounded hosts retain native controls", hostProbe.nativeControls);
      record(`components: bounded hosts remain visible at ${hostProbe.narrow ? "narrow" : "wide"} width`, hostProbe.responsiveVisible);
    }
    if (flags.screenshot) {
      const directory = resolve(flags.screenshot);
      mkdirSync(directory, { recursive: true });
      const file = join(directory, `component-cards-${session.width}x${session.height}.png`);
      await session.screenshot(file);
      record("components: fixture screenshot captured", true, file);
    }
    await session.send("Emulation.setEmulatedMedia", {
      features: [
        { name: "forced-colors", value: "active" },
        { name: "prefers-reduced-motion", value: "reduce" },
      ],
    });
    const mediaProbe = await session.evaluate(`(() => {
      const host = document.getElementById("metric");
      const frame = host?.firstElementChild;
      const style = frame ? getComputedStyle(frame) : null;
      return {
        forced: matchMedia("(forced-colors: active)").matches
          && style?.borderStyle !== "none"
          && style?.boxShadow === "none",
        reduced: matchMedia("(prefers-reduced-motion: reduce)").matches
          && style?.transitionDuration === "0s"
          && style?.animationName === "none",
      };
    })()`);
    record("components: forced colors retain a visible boundary", mediaProbe.forced);
    record("components: reduced motion disables transitions", mediaProbe.reduced);
    record("components: no uncaught console errors", session.consoleErrors.length === 0, session.consoleErrors.slice(0, 3).join(" | "));
    record("components: no failed asset requests", session.failedRequests.length === 0, session.failedRequests.slice(0, 3).join(" | "));
  } finally {
    await session.close();
  }

  const failureSession = new BrowserSession({ url, width: Number(flags.width) || 1440, height: Number(flags.height) || 900 });
  try {
    await failureSession.start();
    await failureSession.failRequestsMatching(["/elements/register.js"]);
    await failureSession.navigate();
    await failureSession.waitFor("window.orbitCanvas && window.orbitVaultStore", 15000);
    const failureSetup = await failureSession.evaluate(`(async () => {
      const wait = milliseconds => new Promise(resolve => setTimeout(resolve, milliseconds));
      const ids = {
        note: "component-registration-failure-save",
        card: "component-registration-fallback-card",
        widget: "component-registration-fallback-widget",
      };
      const definitionsAbsent = [
        "balaur-add-menu",
        "balaur-component-card",
        "balaur-dialog-frame",
        "balaur-inspector",
        "balaur-task-list",
        "balaur-widget-frame",
        "balaur-workspace-nav",
      ].every(name => !customElements.get(name));

      const initialCanvasId = window.orbitCanvas.getCurrentCanvas().id;
      const alternateCanvas = [...document.querySelectorAll("#canvasList [data-fallback-canvas-id]")]
        .find(button => button.dataset.fallbackCanvasId !== initialCanvasId);
      alternateCanvas?.click();
      await wait(100);
      const navigationWorks = Boolean(alternateCanvas)
        && window.orbitCanvas.getCurrentCanvas().id === alternateCanvas.dataset.fallbackCanvasId
        && document.querySelectorAll('#canvasList [aria-current="page"]').length === 1;

      document.querySelector('[data-app-view="today"]')?.click();
      await wait(100);
      const taskRows = [...document.querySelectorAll("balaur-task-list [data-fallback-task-id]")];
      const taskOpen = document.querySelector("balaur-task-list [data-fallback-task-open]");
      taskOpen?.click();
      await wait(100);
      const todayWorks = taskRows.length > 0
        && Boolean(taskOpen)
        && !document.querySelector("#canvas").hidden;

      const beforeAdd = window.orbitCanvas.getDocument().nodes.length;
      document.querySelector("#addMenuToggle")?.click();
      const addPanelOpened = !document.querySelector("#addMenu")?.hidden;
      document.querySelector('balaur-add-menu [data-add="note"]')?.click();
      await wait(100);
      const nativeAddWorks = addPanelOpened
        && window.orbitCanvas.getDocument().nodes.length === beforeAdd + 1;

      const vault = window.orbitVaultStore.vault;
      const cardPath = "cards/registration-fallback.md";
      const widgetPath = "widgets/registration-fallback.html";
      if (!(await vault.stat(cardPath))) {
        await vault.write(cardPath, [
          "---",
          "orbit-schema: 1",
          "orbit-type: component-card",
          'orbit-id: "card-registration-fallback"',
          'title: "Readable fallback card"',
          "recipe: callout",
          "tone: info",
          "---",
          "Canonical fallback body.",
          "",
        ].join("\\n"), { expectedHash: null, mediaType: "text/markdown" });
      }
      if (!(await vault.stat(widgetPath))) {
        await vault.write(widgetPath, "<!doctype html><title>Inactive fallback widget</title><p>Reviewed source stays inactive.</p>\\n", {
          expectedHash: null,
          mediaType: "text/html",
        });
      }
      const canvasDocument = window.orbitCanvas.getDocument();
      const operations = [];
      if (!canvasDocument.nodes.some(node => node.id === ids.note)) {
        operations.push({
          type: "node.add",
          node: { id: ids.note, type: "text", text: "# Registration fallback", x: 20, y: 20, width: 260, height: 120 },
        });
      }
      if (!canvasDocument.nodes.some(node => node.id === ids.card)) {
        operations.push({
          type: "node.add",
          node: { id: ids.card, type: "file", file: cardPath, x: 300, y: 20, width: 320, height: 190 },
        });
      }
      if (!canvasDocument.nodes.some(node => node.id === ids.widget)) {
        operations.push({
          type: "node.add",
          node: { id: ids.widget, type: "file", file: widgetPath, x: 640, y: 20, width: 320, height: 190 },
        });
      }
      if (operations.length) await window.orbitCanvas.applyOperations(operations);
      await wait(100);
      const card = document.querySelector('.canvas-node[data-id="' + ids.card + '"]');
      const widget = document.querySelector('.canvas-node[data-id="' + ids.widget + '"]');
      const cardFallback = card?.querySelector("[data-fallback-component-card]");
      const widgetFallback = widget?.querySelector("[data-fallback-widget]");
      const cardReadable = cardFallback?.textContent.includes("registration-fallback.md")
        && cardFallback.textContent.includes("unavailable");
      const widgetReadable = widgetFallback?.textContent.includes("registration-fallback.html")
        && widgetFallback.textContent.includes("not executed")
        && !widget.querySelector("iframe, balaur-widget-frame");
      const target = cardFallback || card?.querySelector(".node-content");
      if (target) {
        const box = target.getBoundingClientRect();
        target.dispatchEvent(new PointerEvent("pointerdown", {
          bubbles: true,
          button: 0,
          pointerId: 1,
          clientX: box.left + Math.min(20, box.width / 2),
          clientY: box.top + Math.min(20, box.height / 2),
        }));
        window.dispatchEvent(new PointerEvent("pointerup", { bubbles: true, pointerId: 1 }));
      }
      await wait(50);
      const inspector = document.querySelector("#inspector");
      const inspectorReadable = inspector?.matches("[data-fallback-inspector]")
        && inspector.textContent.includes("File node")
        && inspector.querySelectorAll("[data-field-key]").length > 0
        && inspector.querySelectorAll("[data-intent]").length > 0;
      const cardNode = window.orbitCanvas.getDocument().nodes.find(node => node.id === ids.card);
      const targetColor = cardNode?.color === "1" ? "2" : "1";
      inspector?.querySelector('[data-color="' + targetColor + '"]')?.click();
      await wait(50);
      const inspectorActionWorks = window.orbitCanvas.getDocument().nodes
        .some(node => node.id === ids.card && node.color === targetColor);
      return {
        definitionsAbsent,
        navigationWorks,
        todayWorks,
        nativeAddWorks,
        cardReadable,
        widgetReadable,
        inspectorReadable,
        inspectorActionWorks,
        ids,
      };
    })()`);
    await failureSession.waitFor(`document.getElementById("saveState")?.textContent.includes("Saved locally")`, 10000);
    await failureSession.reload();
    await failureSession.waitFor("window.orbitCanvas && window.orbitVaultStore", 15000);
    await failureSession.waitFor(`window.orbitCanvas?.getDocument().nodes.some(node => node.id === "component-registration-failure-save")`, 15000);
    const persisted = await failureSession.evaluate(`(() => {
      const ids = ${JSON.stringify({
        note: "component-registration-failure-save",
        card: "component-registration-fallback-card",
        widget: "component-registration-fallback-widget",
      })};
      const documentModel = window.orbitCanvas.getDocument();
      return {
        rendered: document.querySelectorAll(".canvas-node").length === documentModel.nodes.length,
        persisted: Object.values(ids).every(id => documentModel.nodes.some(node => node.id === id)),
      };
    })()`);
    const fallbackWorks = failureSetup.definitionsAbsent
      && failureSetup.navigationWorks
      && failureSetup.todayWorks
      && failureSetup.nativeAddWorks
      && failureSetup.cardReadable
      && failureSetup.widgetReadable
      && failureSetup.inspectorReadable
      && failureSetup.inspectorActionWorks;
    record("components: registration failure leaves meaningful native application fallbacks", fallbackWorks, JSON.stringify({ failureSetup, persisted }));
    record("components: registration failure still boots, renders, and saves canonical data", persisted.rendered && persisted.persisted);
  } catch (error) {
    record("components: registration failure leaves meaningful native application fallbacks", false, error.message);
    record("components: registration failure still boots, renders, and saves canonical data", false, error.message);
  } finally {
    await failureSession.close();
  }

  let failed = 0;
  for (const result of results) {
    if (!result.ok) failed++;
    console.log(`${result.ok ? "PASS" : "FAIL"}  ${result.name}${result.detail ? `  [${result.detail}]` : ""}`);
  }
  console.log(failed === 0 ? `\nAll ${results.length} checks passed.` : `\n${failed}/${results.length} checks FAILED.`);
  return failed === 0 ? 0 : 1;
}

// ---------------------------------------------------------------------------
// widgets subcommand: sandbox lifecycle, policy, hostile, and app integration.
// ---------------------------------------------------------------------------

async function widgets(url, flags) {
  const results = [];
  const record = (name, ok, detail = "") => { results.push({ name, ok, detail }); };
  const basicSource = `<!doctype html><title>Browser widget</title><p>Ready</p><script>
addEventListener("balaur-widget-ready", () => balaurWidget.post({type:"orbit.widget.status.v1",version:1,payload:{message:"source-ready"}}));
addEventListener("balaur-widget-message", event => {
  const message = event.detail;
  if (message.type === "orbit.widget.theme.v1") balaurWidget.post({type:"orbit.widget.status.v1",version:1,payload:{message:"theme:" + message.payload.tokens.primary}});
  if (message.type === "orbit.widget.preferences.v1") balaurWidget.post({type:"orbit.widget.status.v1",version:1,payload:{message:"preferences:" + message.payload.reducedMotion}});
  if (message.type === "orbit.widget.visibility.v1") balaurWidget.post({type:"orbit.widget.status.v1",version:1,payload:{message:"visibility:" + message.payload.visible}});
});<\/script>`;
  const hostileSource = `<!doctype html><title>Hostile probe</title><p>Probe</p><script>
addEventListener("balaur-widget-ready", async () => {
  const blocked = {};
  const violations = new Set();
  addEventListener("securitypolicyviolation", event => violations.add(event.effectiveDirective));
  try { void parent.document.body; blocked.parent = false; } catch { blocked.parent = true; }
  try { void localStorage.length; blocked.storage = false; } catch { blocked.storage = true; }
  try { void document.cookie; blocked.cookie = false; } catch { blocked.cookie = true; }
  try { await globalThis[["fe","tch"].flat().join("")]("https:" + "//example.invalid/fetch"); blocked.fetch = false; } catch { blocked.fetch = true; }
  blocked.image = await new Promise(resolve => { const image = new Image(); image.onload = () => resolve(false); image.onerror = () => resolve(true); image.src = "https:" + "//example.invalid/image.png"; });
  blocked.font = await new Promise(resolve => {
    const style = document.createElement("style");
    style.textContent = "@font-face{font-family:probe;src:url(https:" + "//example.invalid/font.woff2)}";
    document.head.append(style);
    document.fonts.load("12px probe").then(() => resolve(false), () => resolve(true));
    setTimeout(() => resolve(true), 500);
  });
  try { const form = document.createElement("form"); form.action = "https:" + "//example.invalid/form"; form.target = "_top"; document.body.append(form); form.submit(); blocked.form = false; } catch { blocked.form = true; }
  try { blocked.popup = globalThis.open("data:text/html,popup") === null; } catch { blocked.popup = true; }
  try { top.location = "data:text/html,top"; blocked.top = false; } catch { blocked.top = true; }
  blocked.worker = await new Promise(resolve => {
    try {
      const worker = new globalThis[["Work","er"].join("")]("data:text/javascript,0");
      worker.onmessage = () => resolve(false);
      worker.onerror = () => resolve(true);
      setTimeout(() => { worker.terminate(); resolve(violations.has("worker-src")); }, 500);
    } catch { resolve(true); }
  });
  const frame = document.createElement("iframe");
  frame.src = "data:text/html,nested";
  document.body.append(frame);
  await new Promise(resolve => setTimeout(resolve, 500));
  blocked.form ||= violations.has("form-action");
  blocked.frame = violations.has("frame-src");
  balaurWidget.post({type:"orbit.widget.status.v1",version:1,payload:{message:JSON.stringify(blocked)}});
});<\/script>`;
  const navigationSource = `<!doctype html><title>Navigation probe</title><script>
addEventListener("balaur-widget-ready", () => { location.href = "data:text/html,navigated"; });
<\/script>`;
  const floodSource = `<!doctype html><title>Flood probe</title><script>
addEventListener("balaur-widget-ready", () => {
  for (let index = 0; index < 80; index++) balaurWidget.post({type:"orbit.widget.status.v1",version:1,payload:{message:"flood-" + index}});
});<\/script>`;
  const invalidMessageSource = `<!doctype html><title>Schema probe</title><script>
addEventListener("balaur-widget-ready", () => balaurWidget.post({type:"orbit.widget.mutate.v1",version:1,payload:{canonical:true}}));
<\/script>`;
  const oversizedMessageSource = `<!doctype html><title>Size probe</title><script>
addEventListener("balaur-widget-ready", () => balaurWidget.post({type:"orbit.widget.status.v1",version:1,payload:{message:"x".repeat(70 * 1024)}}));
<\/script>`;
  const quietSource = `<!doctype html><title>Quiet probe</title><script>
const original = MessagePort.prototype.postMessage;
MessagePort.prototype.postMessage = function(message, ...rest) {
  if (message?.type === "orbit.widget.heartbeat.v1") return;
  return original.call(this, message, ...rest);
};
<\/script>`;
  const session = new BrowserSession({
    url,
    profile: flags.profile ? resolve(flags.profile) : null,
    width: Number(flags.width) || 1440,
    height: Number(flags.height) || 900,
  });
  await session.start();
  try {
    const fixtureUrl = new URL(".pi/skills/browser-check/fixtures/widgets.html", url).href;
    await session.navigate(fixtureUrl);
    await session.waitFor("window.widgetFixtureReady", 5000);
    const defined = await session.evaluate("!!customElements.get('balaur-widget-frame')");
    record("widgets: balaur-widget-frame upgrades", defined);
    if (defined) {
      const setup = await session.evaluate(`(() => {
        const root = document.getElementById("fixture");
        const host = document.createElement("balaur-widget-frame");
        host.path = "widgets/browser-widget.html";
        host.title = "Browser widget";
        host.source = ${JSON.stringify(basicSource)};
        host.themeSnapshot = {surface:"#24150c",surfaceRaised:"#2e1a0e",content:"#f1e7d4",contentMuted:"#cfc1aa",paper:"#d7c48f",ink:"#2a2015",primary:"#f2c14e",focus:"#5ed0bd",danger:"#a65745",radius:"4px",fontBody:"Work Sans, system-ui, sans-serif",fontMono:"JetBrains Mono, ui-monospace, monospace"};
        host.preferences = {reducedMotion:true,reducedTransparency:false,contrast:"more"};
        root.append(host);
        const before = {iframe: !!host.shadowRoot.querySelector("iframe"), state: host.dataset.state};
        const created = [];
        const revoked = [];
        const originalCreate = URL.createObjectURL;
        const originalRevoke = URL.revokeObjectURL;
        URL.createObjectURL = value => { const result = originalCreate.call(URL, value); created.push(result); return result; };
        URL.revokeObjectURL = value => { revoked.push(value); return originalRevoke.call(URL, value); };
        window.fixtureWidget = host;
        window.fixtureUrls = {created, revoked, restore(){ URL.createObjectURL = originalCreate; URL.revokeObjectURL = originalRevoke; }};
        host.activate();
        return before;
      })()`);
      record("widgets: source assignment is explicitly inactive", !setup.iframe && setup.state === "inactive");
      await session.waitFor("fixtureWidget.dataset.state === 'ready'", 5000);
      const activeProbe = await session.evaluate(`(() => {
        const host = fixtureWidget;
        const frame = host.shadowRoot.querySelector("iframe");
        const buttons = [...host.shadowRoot.querySelectorAll("button")];
        const statusBeforeFake = host.status;
        dispatchEvent(new MessageEvent("message", {data:{type:"orbit.widget.status.v1",version:1,payload:{message:"global-forgery"}}}));
        return {
          attributes: frame?.getAttribute("sandbox") === "allow-scripts"
            && frame?.getAttribute("referrerpolicy") === "no-referrer"
            && frame?.getAttribute("loading") === "lazy"
            && frame?.hasAttribute("allow") && frame?.getAttribute("allow") === "",
          opaqueBlob: frame?.src.startsWith("blob:") && !frame.src.includes(host.path),
          nativeControls: buttons.length === 3 && buttons.every(button => button instanceof HTMLButtonElement),
          fakeIgnored: host.status === statusBeforeFake,
          active: host.active === true,
        };
      })()`);
      record("widgets: iframe attributes are exact and least-privilege", activeProbe.attributes);
      record("widgets: runtime uses an opaque blob document, never a vault path", activeProbe.opaqueBlob);
      record("widgets: pause reload and source controls are native buttons", activeProbe.nativeControls);
      record("widgets: communication ignores the global message bus", activeProbe.fakeIgnored);
      record("widgets: explicit activation reaches ready state", activeProbe.active);

      const propagationProbe = await session.evaluate(`(() => {
        const parent=fixtureWidget.parentElement,sourceButton=fixtureWidget.shadowRoot.querySelector('[data-action="source"]');
        let pointerdowns=0,clicks=0;
        parent.addEventListener("pointerdown",()=>pointerdowns++,{once:true});
        parent.addEventListener("click",()=>clicks++,{once:true});
        sourceButton.dispatchEvent(new PointerEvent("pointerdown",{bubbles:true,composed:true}));
        sourceButton.click();
        return {pointerdowns,clicks};
      })()`);
      record("widgets: shadow actions cannot start canvas pointer or click handling", propagationProbe.pointerdowns===0&&propagationProbe.clicks===0);

      await session.evaluate(`fixtureWidget.themeSnapshot = {...fixtureWidget.themeSnapshot, primary:"#00ff99"}`);
      await session.waitFor("fixtureWidget.status === 'theme:#00ff99'", 3000);
      record("widgets: theme snapshots project over the private port", true);
      await session.evaluate(`fixtureWidget.preferences = {reducedMotion:false,reducedTransparency:true,contrast:"custom"}`);
      await session.waitFor("fixtureWidget.status === 'preferences:false'", 3000);
      record("widgets: preferences project over the private port", true);
      await session.evaluate(`fixtureWidget.visibility = false`);
      await session.waitFor("fixtureWidget.status === 'visibility:false'", 3000);
      record("widgets: visibility projects over the private port", true);

      const pauseProbe = await session.evaluate(`(() => {
        const oldUrl = fixtureUrls.created.at(-1);
        fixtureWidget.pause();
        return {
          paused: fixtureWidget.dataset.state === "paused" && !fixtureWidget.shadowRoot.querySelector("iframe"),
          revoked: fixtureUrls.revoked.includes(oldUrl),
          count: fixtureUrls.created.length,
        };
      })()`);
      await new Promise(resolve => setTimeout(resolve, 300));
      const stayedPaused = await session.evaluate("fixtureWidget.dataset.state === 'paused' && fixtureUrls.created.length === " + pauseProbe.count);
      record("widgets: pause destroys context and revokes its object URL", pauseProbe.paused && pauseProbe.revoked);
      record("widgets: pause never auto-restarts", stayedPaused);
      await session.evaluate("fixtureWidget.reload()");
      await session.waitFor("fixtureWidget.dataset.state === 'ready' && fixtureUrls.created.length === 2", 5000);
      record("widgets: reload creates a fresh document and channel", await session.evaluate("fixtureUrls.created[0] !== fixtureUrls.created[1]"));

      const identityProbe = await session.evaluate(`(async () => {
        const host=fixtureWidget,checks=[];
        for(const mutate of [
          () => { host.path="widgets/identity-changed.html"; },
          () => { host.source=${JSON.stringify(basicSource.replace("<p>Ready</p>","<p>Changed</p>"))}; },
          () => { host.title="Renamed widget"; },
        ]){
          const oldUrl=fixtureUrls.created.at(-1),before=fixtureUrls.created.length;
          mutate();await new Promise(resolve=>setTimeout(resolve,150));
          checks.push(!host.active&&host.dataset.state==="inactive"&&!host.shadowRoot.querySelector("iframe")&&fixtureUrls.revoked.includes(oldUrl)&&fixtureUrls.created.length===before);
          if(host.title==="Renamed widget")host.source=host.source.replace("<title>Browser widget</title>","<title>Renamed widget</title>");
          host.activate();await new Promise(resolve=>setTimeout(resolve,350));
          checks.push(host.active);
        }
        const beforeDisconnect=fixtureUrls.created.length,disconnectUrl=fixtureUrls.created.at(-1);
        host.remove();document.getElementById("fixture").append(host);await new Promise(resolve=>setTimeout(resolve,200));
        return {
          identity:checks.every(Boolean),
          disconnect:!host.active&&host.dataset.state==="inactive"&&!host.shadowRoot.querySelector("iframe")&&/^Inactive/.test(host.status)&&fixtureUrls.revoked.includes(disconnectUrl)&&fixtureUrls.created.length===beforeDisconnect,
        };
      })()`);
      record("widgets: active source path and title identity changes require explicit Run", identityProbe.identity);
      record("widgets: disconnect and reconnect reset to consistent inactive state", identityProbe.disconnect);

      const policyProbe = await session.evaluate(`(() => {
        const host = document.createElement("balaur-widget-frame");
        host.path = "widgets/invalid.html";
        host.title = "Invalid";
        host.source = "<title>Invalid</title><img src='https://example.invalid/x.png'>";
        document.getElementById("fixture").append(host);
        host.activate();
        return {state:host.dataset.state,status:host.status,iframe:!!host.shadowRoot.querySelector("iframe")};
      })()`);
      record("widgets: invalid source never executes", policyProbe.state === "error" && !policyProbe.iframe && /resource|URL/i.test(policyProbe.status));

      const repairProbe = await session.evaluate(`(() => {
        const host=document.createElement("balaur-widget-frame"),raw="<title>Repair me</title><img src='https://example.invalid/x.png'>";
        host.path="widgets/malformed.html";host.title="Repair me";host.source=raw;host.diagnostic="Malformed widget: forbidden external resource URL";
        let reviewed=null;host.addEventListener("balaur-widget-view-source",event=>reviewed=event.detail,{once:true});
        document.getElementById("fixture").append(host);host.activate();host.shadowRoot.querySelector('[data-action="source"]').click();
        return {state:host.dataset.state,status:host.status,active:host.active,iframe:!!host.shadowRoot.querySelector("iframe"),reviewed,path:host.path,raw};
      })()`);
      record("widgets: malformed catalog state is non-executing and reviewable", repairProbe.state==="repair"&&!repairProbe.active&&!repairProbe.iframe&&repairProbe.status.includes(repairProbe.path)&&repairProbe.status.includes("Malformed widget")&&repairProbe.reviewed?.source===repairProbe.raw);

      const capProbe = await session.evaluate(`(async () => {
        fixtureWidget.pause();
        const hosts = [];
        for (let index = 0; index < 7; index++) {
          const host = document.createElement("balaur-widget-frame");
          host.path = "widgets/cap-" + index + ".html";
          host.title = "Browser widget";
          host.source = ${JSON.stringify(basicSource)};
          document.getElementById("fixture").append(host);
          hosts.push(host);
          host.activate();
        }
        await new Promise(resolve => setTimeout(resolve, 700));
        const firstSix = hosts.slice(0, 6).every(host => host.active);
        const seventhInactive = !hosts[6].active && hosts[6].dataset.state === "inactive";
        hosts[0].pause();
        const noAuto = !hosts[6].active;
        hosts[6].activate();
        await new Promise(resolve => setTimeout(resolve, 500));
        const explicitRecovery = hosts[6].active;
        hosts.forEach(host => host.pause());
        return {firstSix,seventhInactive,noAuto,explicitRecovery};
      })()`);
      record("widgets: six-active cap requires an explicit retry", Object.values(capProbe).every(Boolean));

      async function probeFailure(source, expectedState, label, timeout = 5000) {
        await session.evaluate(`(() => {
          const host = document.createElement("balaur-widget-frame");
          host.path = "widgets/probe.html";
          host.title = ${JSON.stringify(label)};
          host.source = ${JSON.stringify(source)};
          document.getElementById("fixture").append(host);
          window.currentProbe = host;
          host.activate();
        })()`);
        await session.waitFor(`currentProbe.dataset.state === ${JSON.stringify(expectedState)}`, timeout);
        const value = await session.evaluate("({state:currentProbe.dataset.state,status:currentProbe.status,active:currentProbe.active})");
        await session.evaluate("currentProbe.pause(); currentProbe.remove()");
        return value;
      }

      const hostile = await session.evaluate(`(async () => {
        const host = document.createElement("balaur-widget-frame");
        host.path = "widgets/hostile.html";
        host.title = "Hostile probe";
        host.source = ${JSON.stringify(hostileSource)};
        document.getElementById("fixture").append(host);
        host.activate();
        const deadline = Date.now() + 5000;
        while (Date.now() < deadline && !host.status.startsWith("{")) await new Promise(resolve => setTimeout(resolve, 50));
        let blocked = {};
        try { blocked = JSON.parse(host.status); } catch {}
        host.pause();
        return blocked;
      })()`);
      const hostileNames = ["parent","storage","cookie","fetch","image","font","form","popup","top","worker","frame"];
      const escapedRequest=session.requestedUrls.find(requestUrl=>/^https?:\/\/example\.invalid\//.test(requestUrl));
      const fixtureStayedPut=await session.evaluate("location.href")===fixtureUrl,statusCaptured=Object.keys(hostile).length===hostileNames.length;
      const effectiveHostile={...hostile,form:hostile.form===true||hostile.form===false&&!escapedRequest&&fixtureStayedPut&&statusCaptured};
      record("widgets: every hostile capability probe is blocked", hostileNames.every(name => effectiveHostile[name] === true)&&!escapedRequest&&fixtureStayedPut&&statusCaptured, `${JSON.stringify(hostile)} effective-form:${effectiveHostile.form} requests:${escapedRequest||"none"} location:${fixtureStayedPut?"fixture":"escaped"} status:${statusCaptured}`);

      const schema = await probeFailure(invalidMessageSource, "error", "Schema probe");
      record("widgets: invalid inbound schema closes the private port", !schema.active && /schema|message|allowed|unknown/i.test(schema.status));
      const size = await probeFailure(oversizedMessageSource, "error", "Size probe");
      record("widgets: oversized inbound messages close the private port", !size.active && /64 KiB|size|bytes/i.test(size.status));
      const noisy = await probeFailure(floodSource, "noisy", "Flood probe");
      record("widgets: burst rate enforcement closes noisy widgets", !noisy.active);
      const navigated = await probeFailure(navigationSource, "navigated", "Navigation probe");
      record("widgets: self-navigation pauses with the documented diagnostic", !navigated.active && /self-navigation|navigat/i.test(navigated.status));
      const quiet = await probeFailure(quietSource, "unresponsive", "Quiet probe", 18000);
      record("widgets: three missed heartbeats stop without auto-restart", !quiet.active && /heartbeat|unresponsive/i.test(quiet.status));

      await session.evaluate("fixtureUrls.restore()");
      if (flags.screenshot) {
        const directory = resolve(flags.screenshot);
        mkdirSync(directory, { recursive: true });
        const file = join(directory, `widget-frame-${session.width}x${session.height}.png`);
        await session.screenshot(file);
        record("widgets: fixture screenshot captured", true, file);
      }
    }
    record("widgets: no uncaught host errors", session.consoleErrors.length === 0, session.consoleErrors.slice(0, 3).join(" | "));
  } finally {
    await session.close();
  }
  let failed = 0;
  for (const result of results) {
    if (!result.ok) failed++;
    console.log(`${result.ok ? "PASS" : "FAIL"}  ${result.name}${result.detail ? `  [${result.detail}]` : ""}`);
  }
  console.log(failed === 0 ? `\nAll ${results.length} checks passed.` : `\n${failed}/${results.length} checks FAILED.`);
  return failed === 0 ? 0 : 1;
}

// ---------------------------------------------------------------------------
// CLI
// ---------------------------------------------------------------------------

if (import.meta.url === pathToFileURL(process.argv[1] || "").href) {
  const args = parseArgs(process.argv.slice(3));
  const command = process.argv[2] || "smoke";
  const url = args.positional[0] && /^https?:\/\//.test(args.positional[0]) ? args.positional[0] : DEFAULT_URL;

  try {
    if (command === "smoke") {
      process.exit(await smoke(url, args.flags));
    } else if (command === "components") {
      process.exit(await components(url, args.flags));
    } else if (command === "widgets") {
      process.exit(await widgets(url, args.flags));
    } else if (command === "eval") {
      const expression = args.positional.find(a => a !== url) ?? args.positional[0];
      if (!expression) throw new Error("Usage: browser-check.mjs eval [url] <expression> [--wait expr]");
      const session = new BrowserSession({ url, profile: args.flags.profile ? resolve(args.flags.profile) : null });
      await session.start();
      try {
        await session.navigate();
        if (args.flags.wait) await session.waitFor(args.flags.wait, 15000);
        console.log(JSON.stringify(await session.evaluate(expression), null, 2));
        if (session.consoleErrors.length) console.error("console errors:", session.consoleErrors);
      } finally { await session.close(); }
    } else if (command === "shot") {
      const file = args.positional.find(a => a !== url && a.endsWith(".png")) ?? "shot.png";
      const session = new BrowserSession({ url, profile: args.flags.profile ? resolve(args.flags.profile) : null });
      await session.start();
      try {
        await session.navigate();
        await session.waitFor("window.orbitCanvas && document.querySelectorAll('.canvas-node').length > 0", 15000);
        await new Promise(r => setTimeout(r, 600));
        console.log("wrote", await session.screenshot(file, args.flags.selector ?? null));
      } finally { await session.close(); }
    } else {
      console.error("Unknown command. Use: smoke | components | widgets | eval | shot");
      process.exit(2);
    }
  } catch (error) {
    console.error("browser-check failed:", error.message);
    process.exit(1);
  }
}
