function element(name, className = "") {
  const node = document.createElement(name);
  if (className) node.className = className;
  return node;
}

// Collapse state is shell UI state (same localStorage category as the sidebar
// width), never workspace data: the hierarchy itself lives in the sidecar.
const COLLAPSE_STORAGE_KEY = "balaur-shell-collapsed-canvases";

function loadCollapsed() {
  try {
    const raw = JSON.parse(localStorage.getItem(COLLAPSE_STORAGE_KEY) || "[]");
    return new Set(Array.isArray(raw) ? raw.filter(id => typeof id === "string") : []);
  } catch (_) {
    return new Set();
  }
}

function persistCollapsed(collapsed) {
  try {
    localStorage.setItem(COLLAPSE_STORAGE_KEY, JSON.stringify([...collapsed]));
  } catch (_) {}
}

function depthOf(item) {
  return Number.isFinite(item?.depth) ? item.depth : 0;
}

export class BalaurWorkspaceNavElement extends HTMLElement {
  static observedAttributes = ["mode"];

  #activeId = null;
  #canvases = [];
  #collapsed = loadCollapsed();
  #controller;
  #entries = new Map();
  #trail = [];

  connectedCallback() {
    this.#controller?.abort();
    this.#controller = new AbortController();
    const { signal } = this.#controller;
    this.addEventListener("click", event => this.#handleClick(event), { signal });
    this.addEventListener("keydown", event => this.#handleKeydown(event), { signal });
    this.#render();
  }

  disconnectedCallback() {
    this.#controller?.abort();
    this.#controller = undefined;
  }

  attributeChangedCallback(name, oldValue, newValue) {
    if (name === "mode" && oldValue !== newValue) {
      this.#entries.clear();
      this.replaceChildren();
      this.#render();
    }
  }

  get trail() {
    return this.#trail;
  }

  set trail(value) {
    if (value === this.#trail) return;
    this.#trail = Array.isArray(value) ? value : [];
    this.#render();
  }

  get canvases() {
    return this.#canvases;
  }

  set canvases(value) {
    if (value === this.#canvases) return;
    this.#canvases = Array.isArray(value) ? value : [];
    this.#render();
  }

  get activeId() {
    return this.#activeId;
  }

  set activeId(value) {
    const next = typeof value === "string" ? value : null;
    if (next === this.#activeId) return;
    this.#activeId = next;
    this.#render();
  }

  #mode() {
    return this.getAttribute("mode") === "trail" ? "trail" : "canvases";
  }

  #createEntry(id, mode) {
    if (mode !== "canvases") {
      const button = element("button");
      button.type = "button";
      button.dataset.canvasId = id;
      return { button };
    }
    const row = element("div", "tree-row");
    const toggle = element("button", "tree-toggle");
    toggle.type = "button";
    toggle.dataset.treeToggle = id;
    const caret = element("span");
    caret.setAttribute("aria-hidden", "true");
    caret.textContent = "▸";
    toggle.append(caret);
    const button = element("button", "nav-item canvas-list-item");
    button.type = "button";
    button.dataset.canvasId = id;
    const icon = element("span");
    icon.setAttribute("aria-hidden", "true");
    const title = element("b");
    const count = element("em");
    button.append(icon, title, count);
    row.append(toggle, button);
    return { row, toggle, button, icon, title, count };
  }

  #render() {
    if (!this.isConnected) return;
    if (this.#mode() === "trail") this.#renderTrail();
    else this.#renderCanvases();
  }

  #renderTrail() {
    const items = this.#trail;
    const nextIds = new Set();
    this.querySelectorAll("[data-nav-separator]").forEach(separator => separator.remove());
    const desired = [];
    items.forEach((item, index) => {
      const id = typeof item?.id === "string" ? item.id : "";
      if (!id || nextIds.has(id)) return;
      nextIds.add(id);
      let refs = this.#entries.get(id);
      if (!refs) {
        refs = this.#createEntry(id, "trail");
        this.#entries.set(id, refs);
      }
      refs.button.textContent = String(item.title || "Untitled canvas");
      if (id === this.#activeId) refs.button.setAttribute("aria-current", "page");
      else refs.button.removeAttribute("aria-current");
      desired.push(refs.button);
      if (index < items.length - 1) {
        const separator = element("span");
        separator.setAttribute("aria-hidden", "true");
        separator.dataset.navSeparator = "";
        separator.textContent = "›";
        desired.push(separator);
      }
    });
    this.#reconcile(nextIds, desired);
  }

  // The canvases list arrives flat and pre-order (a child directly follows
  // its parent at depth + 1), so ancestry is recovered with a depth stack.
  #renderCanvases() {
    const items = this.#canvases.filter(item => typeof item?.id === "string" && item.id);
    this.#ensureActiveVisible(items);
    const nextIds = new Set();
    const hiddenIds = new Set();
    const stack = [];
    const desired = [];
    items.forEach((item, index) => {
      const id = item.id;
      if (nextIds.has(id)) return;
      nextIds.add(id);
      const depth = depthOf(item);
      while (stack.length && stack[stack.length - 1].depth >= depth) stack.pop();
      if (stack.some(ancestor => this.#collapsed.has(ancestor.id))) hiddenIds.add(id);
      stack.push({ id, depth });
      const hasChildren = index + 1 < items.length && depthOf(items[index + 1]) > depth;
      let refs = this.#entries.get(id);
      if (!refs) {
        refs = this.#createEntry(id, "canvases");
        this.#entries.set(id, refs);
      }
      refs.row.hidden = hiddenIds.has(id);
      refs.row.style.setProperty("--canvas-depth", String(depth));
      refs.toggle.classList.toggle("leaf", !hasChildren);
      if (hasChildren) {
        const expanded = !this.#collapsed.has(id);
        refs.toggle.setAttribute("aria-expanded", String(expanded));
        refs.toggle.setAttribute("aria-label", `${expanded ? "Collapse" : "Expand"} ${item.title || "Untitled canvas"}`);
      } else {
        refs.toggle.removeAttribute("aria-expanded");
        refs.toggle.removeAttribute("aria-label");
      }
      refs.button.classList.toggle("active", id === this.#activeId);
      if (id === this.#activeId) refs.button.setAttribute("aria-current", "page");
      else refs.button.removeAttribute("aria-current");
      refs.icon.textContent = String(item.icon || "↳");
      refs.title.textContent = String(item.title || "Untitled canvas");
      refs.count.textContent = String(Number.isFinite(item.count) ? item.count : 0);
      desired.push(refs.row);
    });
    this.#reconcile(nextIds, desired);
  }

  #reconcile(nextIds, desired) {
    for (const [id, refs] of this.#entries) {
      if (nextIds.has(id)) continue;
      (refs.row || refs.button).remove();
      this.#entries.delete(id);
    }
    let cursor = this.firstChild;
    for (const node of desired) {
      if (node !== cursor) this.insertBefore(node, cursor);
      cursor = node.nextSibling;
    }
  }

  // Opening a canvas never leaves its row hidden inside a collapsed branch.
  #ensureActiveVisible(items) {
    if (!this.#activeId) return;
    const index = items.findIndex(item => item.id === this.#activeId);
    if (index < 0) return;
    const stack = [];
    for (let i = 0; i < index; i++) {
      const depth = depthOf(items[i]);
      while (stack.length && stack[stack.length - 1].depth >= depth) stack.pop();
      stack.push({ id: items[i].id, depth });
    }
    // The loop stops before the active item, so the stack still carries the
    // deeper subtree of its previous sibling. Pop by the active item's own
    // depth (mirroring #renderCanvases) so only true ancestors remain.
    const activeDepth = depthOf(items[index]);
    while (stack.length && stack[stack.length - 1].depth >= activeDepth) stack.pop();
    let changed = false;
    for (const ancestor of stack) {
      if (this.#collapsed.delete(ancestor.id)) changed = true;
    }
    if (changed) persistCollapsed(this.#collapsed);
  }

  #setCollapsed(id, value) {
    if (value) this.#collapsed.add(id);
    else this.#collapsed.delete(id);
    persistCollapsed(this.#collapsed);
    this.#render();
  }

  #handleClick(event) {
    const toggle = event.target.closest?.("button[data-tree-toggle]");
    if (toggle && this.contains(toggle)) {
      const id = toggle.dataset.treeToggle;
      if (this.#canvases.some(item => item?.id === id)) this.#setCollapsed(id, !this.#collapsed.has(id));
      return;
    }
    const button = event.target.closest?.("button[data-canvas-id]");
    if (!button || !this.contains(button) || button.disabled) return;
    const canvasId = button.dataset.canvasId;
    const items = this.#mode() === "trail" ? this.#trail : this.#canvases;
    if (!canvasId || !items.some(item => item?.id === canvasId)) return;
    this.dispatchEvent(new CustomEvent("balaur-canvas-open", {
      bubbles: true,
      composed: true,
      detail: { canvasId },
    }));
  }

  // ← collapses the branch under the focused row, → re-expands it.
  #handleKeydown(event) {
    if (this.#mode() !== "canvases" || (event.key !== "ArrowLeft" && event.key !== "ArrowRight")) return;
    const button = event.target.closest?.("button[data-canvas-id]");
    if (!button || !this.contains(button)) return;
    const refs = this.#entries.get(button.dataset.canvasId);
    if (!refs || refs.toggle.classList.contains("leaf")) return;
    const collapsed = this.#collapsed.has(button.dataset.canvasId);
    if ((event.key === "ArrowLeft" && !collapsed) || (event.key === "ArrowRight" && collapsed)) {
      event.preventDefault();
      this.#setCollapsed(button.dataset.canvasId, !collapsed);
    }
  }
}
