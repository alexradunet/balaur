import { buildWidgetDocument } from "../widgets/widget-envelope.js";
import {
  MAX_ACTIVE_WIDGETS,
  WIDGET_MESSAGE_BURST,
  WIDGET_MESSAGES_PER_SECOND,
  validateWidgetSource,
} from "../widgets/widget-policy.js";
import {
  HOST_TO_WIDGET,
  WIDGET_PROTOCOL_VERSION,
  WIDGET_TO_HOST,
  validateWidgetMessage,
} from "../widgets/widget-protocol.js";

const activeWidgets = new Set();
const HEARTBEAT_INTERVAL_MS = 5_000;
const HEARTBEAT_MISSES = 3;
const INACTIVE_STATUS = "Inactive — choose Run to execute reviewed source.";

const BOOTSTRAP_SOURCE = `(() => {
  let port = null;
  const post = message => { if (port) port.postMessage(message); };
  Object.defineProperty(globalThis, "balaurWidget", { value: Object.freeze({ post }), configurable: false, writable: false });
  globalThis.__balaurReportDiagnostic = payload => post({ type: "orbit.widget.diagnostic.v1", version: 1, payload });
  addEventListener("message", function initialize(event) {
    if (port || event.data?.type !== "orbit.widget.init.v1" || event.data?.version !== 1 || event.ports.length !== 1) return;
    port = event.ports[0];
    port.onmessage = messageEvent => {
      const message = messageEvent.data;
      if (!message || message.version !== 1 || typeof message.type !== "string" || !message.payload || typeof message.payload !== "object") return;
      if (message.type === "orbit.widget.theme.v1") {
        for (const [key, value] of Object.entries(message.payload.tokens)) document.documentElement.style.setProperty("--balaur-" + key, value);
      }
      if (message.type === "orbit.widget.pause.v1") dispatchEvent(new CustomEvent("balaur-widget-pause", { detail: message.payload }));
      dispatchEvent(new CustomEvent("balaur-widget-message", { detail: message }));
    };
    port.start();
    post({ type: "orbit.widget.ready.v1", version: 1, payload: {} });
    dispatchEvent(new Event("balaur-widget-ready"));
    setInterval(() => post({ type: "orbit.widget.heartbeat.v1", version: 1, payload: {} }), 5000);
  }, { once: true });
})();`;

function button(label, action) {
  const value = document.createElement("button");
  value.type = "button";
  value.textContent = label;
  value.dataset.action = action;
  return value;
}

export class BalaurWidgetFrameElement extends HTMLElement {
  #source = "";
  #path = "";
  #title = "Live widget";
  #diagnostic = "";
  #themeSnapshot = null;
  #preferences = { reducedMotion: false, reducedTransparency: false, contrast: "no-preference" };
  #visibility = true;
  #iframe = null;
  #port = null;
  #objectUrl = null;
  #heartbeatTimer = null;
  #lastHeartbeat = 0;
  #loadCount = 0;
  #tokens = WIDGET_MESSAGE_BURST;
  #tokenTime = 0;
  #observer = null;
  #intersecting = true;
  #visibilityHandler = () => { this.visibility = this.#intersecting && !document.hidden; };
  #status = INACTIVE_STATUS;
  #refs = null;

  constructor() {
    super();
    const root = this.attachShadow({ mode: "open" });
    const frame = document.createElement("article");
    frame.className = "widget-host";
    const header = document.createElement("header");
    const title = document.createElement("strong");
    title.part = "title";
    const status = document.createElement("span");
    status.part = "status";
    status.setAttribute("role", "status");
    header.append(title, status);
    const viewport = document.createElement("div");
    viewport.className = "widget-viewport";
    viewport.part = "body";
    const actions = document.createElement("footer");
    actions.part = "actions";
    actions.append(button("Run", "activate"), button("Reload", "reload"), button("View source", "source"));
    const style = document.createElement("style");
    style.textContent = `
      :host{display:block;min-inline-size:0;block-size:100%;color:var(--content-primary,#21170f)}:host([hidden]){display:none}
      .widget-host{display:grid;grid-template-rows:auto minmax(8rem,1fr) auto;block-size:100%;min-block-size:12rem;background:var(--surface-raised,#fff);border:1px solid var(--border-subtle,#9b8b7a);border-radius:var(--radius-md,8px);overflow:hidden}
      header,footer{display:flex;align-items:center;gap:.5rem;padding:.45rem .6rem;background:var(--surface-sunken,#eee8de)}header{justify-content:space-between}header span{font-size:.72rem;color:var(--content-muted,#655b50);text-align:end}footer{justify-content:flex-end;flex-wrap:wrap}
      button{font:inherit;border:1px solid currentColor;border-radius:4px;background:transparent;color:inherit;padding:.25rem .55rem;cursor:pointer}button:focus-visible{outline:2px solid var(--focus-ring,#087f70);outline-offset:2px}
      .widget-viewport{position:relative;min-block-size:0;background:var(--surface-canvas,#f5f1ea)}iframe{display:block;border:0;inline-size:100%;block-size:100%;min-block-size:8rem}
      :host([data-state="error"]) .widget-host,:host([data-state="repair"]) .widget-host,:host([data-state="noisy"]) .widget-host,:host([data-state="unresponsive"]) .widget-host,:host([data-state="navigated"]) .widget-host{border-color:var(--status-danger,#a65745)}
      @media(forced-colors:active){.widget-host{border:1px solid CanvasText}button{border:1px solid ButtonText}}
      @media(prefers-reduced-motion:reduce){*{transition:none!important;animation:none!important}}
    `;
    root.append(style, frame);
    frame.append(header, viewport, actions);
    this.#refs = { title, status, viewport, actions };
    actions.addEventListener("pointerdown",event=>event.stopPropagation());
    actions.addEventListener("click", (event) => {
      event.stopPropagation();
      const action = event.target.closest("button")?.dataset.action;
      if (action === "activate") {
        if (this.active) this.pause(); else this.activate();
        this.dispatchEvent(new CustomEvent("balaur-widget-pause", { bubbles: true, composed: true, detail: { path: this.#path, active: this.active } }));
      } else if (action === "reload") {
        this.reload();
        this.dispatchEvent(new CustomEvent("balaur-widget-reload", { bubbles: true, composed: true, detail: { path: this.#path } }));
      } else if (action === "source") {
        this.dispatchEvent(new CustomEvent("balaur-widget-view-source", { bubbles: true, composed: true, detail: { path: this.#path, title: this.#title, source: this.#source } }));
      }
    });
    this.#refs.status.textContent = this.#status;
  }

  connectedCallback() {
    this.#refs.title.textContent = this.#title;
    if (this.#diagnostic) this.#setState("repair", `${this.#path || "Widget source"} — ${this.#diagnostic}`);
    else if (!this.dataset.state || this.dataset.state !== "inactive") this.#setState("inactive", INACTIVE_STATUS);
    this.#observer = new IntersectionObserver(([entry]) => { this.#intersecting=entry.isIntersecting;this.#visibilityHandler(); });
    this.#observer.observe(this);
    document.addEventListener("visibilitychange",this.#visibilityHandler);
  }

  disconnectedCallback() {
    this.#observer?.disconnect();
    this.#observer = null;
    document.removeEventListener("visibilitychange",this.#visibilityHandler);
    this.#cleanup();
    this.#setState("inactive", INACTIVE_STATUS);
  }

  get source() { return this.#source; }
  set source(value) { const next=typeof value === "string" ? value : "";if(next===this.#source)return;this.#invalidateIdentity();this.#source=next; }
  get path() { return this.#path; }
  set path(value) { const next=typeof value === "string" ? value : "";if(next===this.#path)return;this.#invalidateIdentity();this.#path=next; }
  get title() { return this.#title; }
  set title(value) { const next=typeof value === "string" && value.trim() ? value.trim() : "Live widget";if(next===this.#title)return;this.#invalidateIdentity();this.#title=next;if(this.#refs)this.#refs.title.textContent=next; }
  get diagnostic() { return this.#diagnostic; }
  set diagnostic(value) {
    const next=typeof value === "string" ? value.trim() : "";
    if(next===this.#diagnostic)return;
    if(this.#iframe||this.#port||activeWidgets.has(this))this.#cleanup();
    this.#diagnostic=next;
    if(next)this.#setState("repair",`${this.#path || "Widget source"} — ${next}`);
    else if(this.dataset.state==="repair")this.#setState("inactive",INACTIVE_STATUS);
  }
  get themeSnapshot() { return this.#themeSnapshot; }
  set themeSnapshot(value) { this.#themeSnapshot = value && typeof value === "object" ? { ...value } : null; this.#sendTheme(); }
  get preferences() { return { ...this.#preferences }; }
  set preferences(value) { this.#preferences = value && typeof value === "object" ? { ...value } : this.#preferences; this.#send("orbit.widget.preferences.v1", this.#preferences); }
  get visibility() { return this.#visibility; }
  set visibility(value) { this.#visibility = Boolean(value); this.#send("orbit.widget.visibility.v1", { visible: this.#visibility }); }
  get active() { return activeWidgets.has(this) && Boolean(this.#iframe); }
  get status() { return this.#status; }

  activate() {
    if (this.active) return true;
    if(this.#diagnostic){this.#setState("repair",`${this.#path || "Widget source"} — ${this.#diagnostic}`);return false;}
    let validated;
    try {
      validated = validateWidgetSource(this.#source);
      if (this.#title !== validated.title) throw new TypeError("Widget title does not match the validated source title");
    } catch (error) {
      this.#setState("error", error.message);
      return false;
    }
    if (activeWidgets.size >= MAX_ACTIVE_WIDGETS) {
      this.#setState("inactive", `Inactive — ${MAX_ACTIVE_WIDGETS} widgets are already active. Pause one, then choose Run again.`);
      return false;
    }
    activeWidgets.add(this);
    this.#tokens = WIDGET_MESSAGE_BURST;
    this.#tokenTime = performance.now();
    this.#lastHeartbeat = performance.now();
    this.#loadCount = 0;
    const documentSource = buildWidgetDocument(validated.source, { bootstrapSource: BOOTSTRAP_SOURCE });
    this.#objectUrl = URL.createObjectURL(new Blob([documentSource], { type: "text/html" }));
    const iframe = document.createElement("iframe");
    iframe.setAttribute("sandbox", "allow-scripts");
    iframe.setAttribute("referrerpolicy", "no-referrer");
    iframe.setAttribute("loading", "lazy");
    iframe.setAttribute("allow", "");
    iframe.title = this.#title;
    iframe.addEventListener("load", () => this.#loaded(iframe));
    iframe.src = this.#objectUrl;
    this.#iframe = iframe;
    this.#refs.viewport.replaceChildren(iframe);
    this.#setState("starting", "Starting reviewed widget…");
    this.#syncAction();
    return true;
  }

  pause(reason = "user") {
    if (this.#port) this.#send("orbit.widget.pause.v1", { reason });
    this.#cleanup();
    this.#setState("paused", "Paused — source remains available for review or reload.");
  }

  reload() {
    this.#cleanup();
    return this.activate();
  }

  #invalidateIdentity() {
    if(!this.#iframe&&!this.#port&&!activeWidgets.has(this))return;
    this.#cleanup();
    this.#setState("inactive","Inactive — widget source identity changed. Review it, then choose Run.");
  }

  #loaded(iframe) {
    if (iframe !== this.#iframe) return;
    this.#loadCount += 1;
    if (this.#loadCount > 1) {
      this.#stop("navigated", "Self-navigation detected. The widget was paused; hard request suppression is not claimed.");
      return;
    }
    const channel = new MessageChannel();
    this.#port = channel.port1;
    channel.port1.onmessage = (event) => this.#receive(event.data);
    channel.port1.onmessageerror = () => this.#stop("error", "Widget message could not be decoded.");
    channel.port1.start();
    iframe.contentWindow.postMessage({ type: "orbit.widget.init.v1", version: WIDGET_PROTOCOL_VERSION, payload: {} }, "*", [channel.port2]);
    this.#heartbeatTimer = setInterval(() => {
      if (performance.now() - this.#lastHeartbeat >= HEARTBEAT_INTERVAL_MS * HEARTBEAT_MISSES - 250) {
        this.#stop("unresponsive", "Widget is unresponsive after three missed heartbeats. It was not restarted.");
      }
    }, HEARTBEAT_INTERVAL_MS);
  }

  #receive(value) {
    const now = performance.now();
    this.#tokens = Math.min(WIDGET_MESSAGE_BURST, this.#tokens + ((now - this.#tokenTime) / 1000) * WIDGET_MESSAGES_PER_SECOND);
    this.#tokenTime = now;
    if (this.#tokens < 1) {
      this.#stop("noisy", "Widget exceeded the 30 messages/second rate with burst 60. It was not restarted.");
      return;
    }
    this.#tokens -= 1;
    let message;
    try { message = validateWidgetMessage(WIDGET_TO_HOST, value); }
    catch (error) { this.#stop("error", `Widget message schema rejected: ${error.message}`); return; }
    if (message.type === "orbit.widget.ready.v1") {
      this.#lastHeartbeat = now;
      this.#setState("ready", "Ready");
      this.#sendTheme();
      this.#send("orbit.widget.preferences.v1", this.#preferences);
      this.#send("orbit.widget.visibility.v1", { visible: this.#visibility && !document.hidden });
    } else if (message.type === "orbit.widget.heartbeat.v1") this.#lastHeartbeat = now;
    else if (message.type === "orbit.widget.status.v1") this.#setState("ready", message.payload.message);
    else if (message.type === "orbit.widget.diagnostic.v1") this.#setState("ready", `${message.payload.level}: ${message.payload.message}`);
    else if (message.type === "orbit.widget.resize.v1" && this.#iframe) this.#iframe.style.minHeight = `${message.payload.height}px`;
  }

  #sendTheme() {
    if (this.#themeSnapshot) this.#send("orbit.widget.theme.v1", { tokens: this.#themeSnapshot });
  }

  #send(type, payload) {
    if (!this.#port) return false;
    try {
      this.#port.postMessage(validateWidgetMessage(HOST_TO_WIDGET, { type, version: WIDGET_PROTOCOL_VERSION, payload }));
      return true;
    } catch (error) {
      this.#stop("error", `Host widget message rejected: ${error.message}`);
      return false;
    }
  }

  #stop(state, message) {
    this.#cleanup();
    this.#setState(state, message);
  }

  #cleanup() {
    activeWidgets.delete(this);
    clearInterval(this.#heartbeatTimer);
    this.#heartbeatTimer = null;
    if (this.#port) { this.#port.onmessage = null; this.#port.close(); this.#port = null; }
    if (this.#iframe) { this.#iframe.remove(); this.#iframe = null; }
    if (this.#objectUrl) { URL.revokeObjectURL(this.#objectUrl); this.#objectUrl = null; }
    this.#syncAction();
  }

  #setState(state, message) {
    this.dataset.state = state;
    this.#status = String(message || "");
    this.#refs.status.textContent = this.#status;
    this.#syncAction();
  }

  #syncAction() {
    const run = this.#refs?.actions.querySelector('[data-action="activate"]');
    if (run) run.textContent = this.active ? "Pause" : "Run";
  }
}
