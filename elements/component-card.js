const RECIPES = new Set(["metric", "progress", "callout", "list", "timeline"]);
const TRENDS = new Set(["up", "down", "flat"]);
const TONES = new Set(["info", "success", "warning", "danger"]);
const PRESET_COLORS = new Set(["1", "2", "3", "4", "5", "6"]);

function element(name, className = "") {
  const node = document.createElement(name);
  if (className) node.className = className;
  return node;
}

function safeLink(value) {
  try {
    const url = new URL(value, document.baseURI);
    return ["http:", "https:", "mailto:"].includes(url.protocol) ? url.href : null;
  } catch {
    return null;
  }
}

function appendInline(parent, source) {
  const pattern = /(\*\*([^*]+)\*\*|`([^`]+)`|\[([^\]]+)\]\(([^)\s]+)\))/g;
  let offset = 0;
  for (const match of String(source).matchAll(pattern)) {
    parent.append(document.createTextNode(source.slice(offset, match.index)));
    if (match[2] !== undefined) {
      const strong = element("strong");
      strong.textContent = match[2];
      parent.append(strong);
    } else if (match[3] !== undefined) {
      const code = element("code");
      code.textContent = match[3];
      parent.append(code);
    } else {
      const href = safeLink(match[5]);
      if (href) {
        const link = element("a");
        link.href = href;
        link.target = "_blank";
        link.rel = "noreferrer";
        link.textContent = match[4];
        parent.append(link);
      } else {
        parent.append(document.createTextNode(match[4]));
      }
    }
    offset = match.index + match[0].length;
  }
  parent.append(document.createTextNode(source.slice(offset)));
}

function compatibleNode(current, desired) {
  if (!current || current.nodeType !== desired.nodeType) return false;
  return current.nodeType !== Node.ELEMENT_NODE
    || (current.localName === desired.localName && current.namespaceURI === desired.namespaceURI);
}

function reconcileAttributes(current, desired) {
  for (const attribute of [...current.attributes]) {
    if (!desired.hasAttribute(attribute.name)) current.removeAttribute(attribute.name);
  }
  for (const attribute of desired.attributes) {
    if (current.getAttribute(attribute.name) !== attribute.value) current.setAttribute(attribute.name, attribute.value);
  }
}

function reconcileChildren(parent, desiredParent) {
  const desiredChildren = [...desiredParent.childNodes];
  for (let index = 0; index < desiredChildren.length; index++) {
    const desired = desiredChildren[index];
    const current = parent.childNodes[index];
    if (!compatibleNode(current, desired)) {
      if (current) current.replaceWith(desired);
      else parent.append(desired);
      continue;
    }
    if (current.nodeType === Node.TEXT_NODE) {
      if (current.data !== desired.data) current.data = desired.data;
    } else {
      reconcileAttributes(current, desired);
      reconcileChildren(current, desired);
    }
  }
  while (parent.childNodes.length > desiredChildren.length) parent.lastChild.remove();
}

function renderMarkdown(container, source) {
  const fragment = document.createDocumentFragment();
  let list = null;
  for (const rawLine of String(source || "").split(/\r?\n/)) {
    const line = rawLine.trimEnd();
    const item = line.match(/^\s*-\s+(?:\[([ xX])\]\s+)?(.+)$/);
    if (item) {
      if (!list) {
        list = element("ul", "component-card-markdown-list");
        fragment.append(list);
      }
      const row = element("li");
      if (item[1] !== undefined) {
        row.dataset.checked = item[1].toLowerCase() === "x" ? "true" : "false";
        const marker = element("span", "component-card-check");
        marker.setAttribute("aria-hidden", "true");
        row.append(marker);
      }
      appendInline(row, item[2]);
      list.append(row);
      continue;
    }
    list = null;
    if (!line.trim() || /^<!--\s*orbit:/.test(line.trim())) continue;
    const heading = line.match(/^#{1,6}\s+(.+)$/);
    const block = element(heading ? "h3" : "p");
    appendInline(block, heading ? heading[1] : line);
    fragment.append(block);
  }
  reconcileChildren(container, fragment);
}

function renderTimeline(container, source) {
  const list = element("ol", "component-card-timeline-list");
  for (const rawLine of String(source || "").split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line) continue;
    const match = line.match(/^(?:#{1,6}\s+|-\s+)?(\d{4}-\d{2}-\d{2})(?:\s+|$)(.*)$/);
    const item = element("li", match ? "component-card-timeline-entry" : "component-card-timeline-note");
    if (match) {
      const time = element("time");
      time.dateTime = match[1];
      time.textContent = match[1];
      item.append(time);
      if (match[2]) {
        const detail = element("span");
        appendInline(detail, match[2]);
        item.append(detail);
      }
    } else {
      appendInline(item, line.replace(/^[-#]+\s*/, ""));
    }
    list.append(item);
  }
  const fragment = document.createDocumentFragment();
  fragment.append(list);
  reconcileChildren(container, fragment);
}

function diagnosticFor(model) {
  if (!model || typeof model !== "object") return "Component-card data is unavailable.";
  if (typeof model.diagnostic === "string" && model.diagnostic.trim()) return model.diagnostic;
  if (!RECIPES.has(model.recipe)) return `Unknown component-card recipe: ${String(model.recipe || "missing")}.`;
  return "Component-card data is invalid.";
}

function validModel(model) {
  if (!model || typeof model !== "object" || model.schema !== 1 || model.type !== "component-card") return false;
  if (typeof model.id !== "string" || !model.id || typeof model.title !== "string" || !model.title.trim()) return false;
  if (typeof model.body !== "string" || !RECIPES.has(model.recipe)) return false;
  if (model.recipe === "metric") {
    return typeof model.value === "string"
      && (model.progress === null || model.progress === undefined || (Number.isFinite(model.progress) && model.progress >= 0 && model.progress <= 1))
      && (model.trend === null || model.trend === undefined || TRENDS.has(model.trend));
  }
  if (model.recipe === "progress") return Number.isFinite(model.value) && Number.isFinite(model.maximum) && model.value >= 0 && model.maximum > 0 && model.value <= model.maximum;
  if (model.recipe === "callout") return model.tone === null || model.tone === undefined || TONES.has(model.tone);
  return true;
}

export class BalaurComponentCardElement extends HTMLElement {
  #model = null;
  #ready = false;
  #refs = null;

  connectedCallback() {
    this.#ensureDom();
    this.#render();
  }

  get model() {
    return this.#model;
  }

  set model(value) {
    if (value === this.#model) return;
    this.#model = value;
    this.#ensureDom();
    this.#render();
  }

  get placementColor() {
    return this.dataset.placementColor || null;
  }

  set placementColor(value) {
    const color = typeof value === "string" ? value.trim() : "";
    if (PRESET_COLORS.has(color) || /^#[0-9a-f]{6}$/i.test(color)) {
      this.dataset.placementColor = color;
      if (color.startsWith("#")) this.style.setProperty("--component-card-accent", color);
      else this.style.removeProperty("--component-card-accent");
    } else {
      delete this.dataset.placementColor;
      this.style.removeProperty("--component-card-accent");
    }
  }

  #ensureDom() {
    if (this.#ready) return;
    const frame = element("article", "component-card-frame");
    const header = element("header", "component-card-header");
    const kicker = element("p", "component-card-kicker");
    const title = element("h2", "component-card-title");
    header.append(kicker, title);

    const layout = element("div", "component-card-layout");
    const metric = element("section", "component-card-recipe component-card-metric");
    metric.dataset.recipeRegion = "metric";
    const metricValue = element("p", "component-card-metric-value");
    const metricLabel = element("p", "component-card-metric-label");
    const metricTrend = element("span", "component-card-trend");
    const metricProgress = element("progress", "component-card-meter");
    metricProgress.max = 1;
    const metricBody = element("div", "component-card-markdown");
    metric.append(metricValue, metricLabel, metricTrend, metricProgress, metricBody);

    const progressRegion = element("section", "component-card-recipe component-card-progress");
    progressRegion.dataset.recipeRegion = "progress";
    const progressText = element("p", "component-card-progress-value");
    const progressMeter = element("progress", "component-card-meter");
    const progressBody = element("div", "component-card-markdown");
    progressRegion.append(progressText, progressMeter, progressBody);

    const callout = element("aside", "component-card-recipe component-card-callout");
    callout.dataset.recipeRegion = "callout";
    const calloutBody = element("div", "component-card-markdown");
    callout.append(calloutBody);

    const listRegion = element("section", "component-card-recipe component-card-list");
    listRegion.dataset.recipeRegion = "list";
    const listBody = element("div", "component-card-markdown");
    listRegion.append(listBody);

    const timeline = element("section", "component-card-recipe component-card-timeline");
    timeline.dataset.recipeRegion = "timeline";
    const timelineBody = element("div", "component-card-markdown");
    timeline.append(timelineBody);

    const fallback = element("section", "component-card-recipe component-card-fallback");
    fallback.dataset.recipeRegion = "fallback";
    const fallbackBody = element("div", "component-card-markdown");
    const diagnostic = element("p", "component-card-diagnostic");
    diagnostic.setAttribute("role", "status");
    fallback.append(fallbackBody, diagnostic);
    layout.append(metric, progressRegion, callout, listRegion, timeline, fallback);

    const footer = element("footer", "component-card-footer");
    const open = element("button", "component-card-open");
    open.type = "button";
    open.dataset.cardOpen = "";
    open.textContent = "Open card";
    open.addEventListener("pointerdown", event => event.stopPropagation());
    open.addEventListener("click", event => {
      event.stopPropagation();
      const model = this.#model;
      if (!model || typeof model !== "object") return;
      this.dispatchEvent(new CustomEvent("balaur-card-open", {
        bubbles: true,
        composed: true,
        detail: {
          cardId: typeof model.id === "string" ? model.id : null,
          path: typeof model.path === "string" ? model.path : null,
          nodeId: this.dataset.nodeId || null,
        },
      }));
    });
    footer.append(open);
    frame.append(header, layout, footer);
    this.append(frame);
    this.#refs = {
      frame, kicker, title, layout, metric, metricValue, metricLabel, metricTrend, metricProgress, metricBody,
      progressRegion, progressText, progressMeter, progressBody, callout, calloutBody, listRegion, listBody,
      timeline, timelineBody, fallback, fallbackBody, diagnostic, open,
    };
    this.#ready = true;
  }

  #render() {
    if (!this.#ready) return;
    const model = this.#model;
    const valid = validModel(model);
    const recipe = valid ? model.recipe : "fallback";
    this.dataset.recipe = recipe;
    this.#refs.kicker.textContent = recipe === "fallback" ? "COMPONENT FILE" : recipe.toUpperCase();
    this.#refs.title.textContent = typeof model?.title === "string" && model.title.trim() ? model.title : "Component card";
    this.#refs.frame.dataset.tone = valid && model.recipe === "callout" ? (model.tone || "info") : "";
    this.#refs.open.disabled = !model || typeof model !== "object";
    this.#refs.open.setAttribute("aria-label", `Open ${this.#refs.title.textContent}`);
    for (const [name, region] of [["metric", this.#refs.metric], ["progress", this.#refs.progressRegion], ["callout", this.#refs.callout], ["list", this.#refs.listRegion], ["timeline", this.#refs.timeline], ["fallback", this.#refs.fallback]]) {
      region.hidden = name !== recipe;
    }

    if (recipe === "metric") {
      this.#refs.metricValue.textContent = model.value;
      this.#refs.metricLabel.textContent = model.label || "";
      this.#refs.metricLabel.hidden = !model.label;
      this.#refs.metricTrend.textContent = model.trend ? ({ up: "↑ Up", down: "↓ Down", flat: "→ Flat" })[model.trend] : "";
      this.#refs.metricTrend.hidden = !model.trend;
      this.#refs.metricProgress.value = model.progress ?? 0;
      this.#refs.metricProgress.hidden = model.progress === null || model.progress === undefined;
      renderMarkdown(this.#refs.metricBody, model.body);
    } else if (recipe === "progress") {
      const unit = model.unit ? ` ${model.unit}` : "";
      this.#refs.progressText.textContent = `${model.value} of ${model.maximum}${unit}`;
      this.#refs.progressMeter.max = model.maximum;
      this.#refs.progressMeter.value = model.value;
      renderMarkdown(this.#refs.progressBody, model.body);
    } else if (recipe === "callout") {
      renderMarkdown(this.#refs.calloutBody, model.body);
    } else if (recipe === "list") {
      renderMarkdown(this.#refs.listBody, model.body);
    } else if (recipe === "timeline") {
      renderTimeline(this.#refs.timelineBody, model.body);
    } else {
      renderMarkdown(this.#refs.fallbackBody, typeof model?.body === "string" ? model.body : "");
      this.#refs.diagnostic.textContent = diagnosticFor(model);
    }
  }
}
