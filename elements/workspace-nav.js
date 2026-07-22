function element(name, className = "") {
  const node = document.createElement(name);
  if (className) node.className = className;
  return node;
}

export class BalaurWorkspaceNavElement extends HTMLElement {
  static observedAttributes = ["mode"];

  #activeId = null;
  #canvases = [];
  #controller;
  #entries = new Map();
  #trail = [];

  connectedCallback() {
    this.#controller?.abort();
    this.#controller = new AbortController();
    this.addEventListener("click", event => this.#handleClick(event), { signal: this.#controller.signal });
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
    const button = element("button", mode === "canvases" ? "nav-item canvas-list-item" : "");
    button.type = "button";
    button.dataset.canvasId = id;
    if (mode === "canvases") {
      const icon = element("span");
      icon.setAttribute("aria-hidden", "true");
      const title = element("b");
      const count = element("em");
      button.append(icon, title, count);
      return { button, icon, title, count };
    }
    return { button };
  }

  #render() {
    if (!this.isConnected) return;
    const mode = this.#mode();
    const items = mode === "trail" ? this.#trail : this.#canvases;
    const nextIds = new Set();
    this.querySelectorAll("[data-nav-separator]").forEach(separator => separator.remove());
    const desired = [];
    items.forEach((item, index) => {
      const id = typeof item?.id === "string" ? item.id : "";
      if (!id || nextIds.has(id)) return;
      nextIds.add(id);
      let refs = this.#entries.get(id);
      if (!refs) {
        refs = this.#createEntry(id, mode);
        this.#entries.set(id, refs);
      }
      if (mode === "trail") {
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
      } else {
        refs.button.classList.toggle("active", id === this.#activeId);
        if (id === this.#activeId) refs.button.setAttribute("aria-current", "page");
        else refs.button.removeAttribute("aria-current");
        refs.button.style.setProperty("--canvas-depth", String(Number.isFinite(item.depth) ? item.depth : 0));
        refs.icon.textContent = String(item.icon || "↳");
        refs.title.textContent = String(item.title || "Untitled canvas");
        refs.count.textContent = String(Number.isFinite(item.count) ? item.count : 0);
        desired.push(refs.button);
      }
    });
    for (const [id, refs] of this.#entries) {
      if (nextIds.has(id)) continue;
      refs.button.remove();
      this.#entries.delete(id);
    }
    let cursor = this.firstChild;
    for (const node of desired) {
      if (node !== cursor) this.insertBefore(node, cursor);
      cursor = node.nextSibling;
    }
  }

  #handleClick(event) {
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
}
