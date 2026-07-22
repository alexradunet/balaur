export class BalaurAddMenuElement extends HTMLElement {
  #controller;
  #panel;
  #supportsPopover = false;
  #toggle;

  connectedCallback() {
    this.#controller?.abort();
    this.#controller = new AbortController();
    this.#toggle = this.querySelector(".add-menu-toggle");
    this.#panel = this.querySelector(".add-menu-panel");
    if (!this.#toggle || !this.#panel) return;

    const { signal } = this.#controller;
    this.#supportsPopover = typeof this.#panel.showPopover === "function"
      && typeof this.#panel.hidePopover === "function";
    if (this.#supportsPopover) {
      this.#panel.hidden = false;
      this.#panel.setAttribute("popover", "manual");
    } else {
      this.#panel.removeAttribute("popover");
      this.#panel.hidden = true;
    }
    this.#toggle.setAttribute("aria-expanded", "false");

    this.#toggle.addEventListener("click", () => {
      if (this.#isOpen()) this.close();
      else this.open();
    }, { signal });
    this.#toggle.addEventListener("keydown", event => this.#handleToggleKeydown(event), { signal });
    this.#panel.addEventListener("keydown", event => this.#handleMenuKeydown(event), { signal });
    this.addEventListener("click", event => this.#handleSelection(event), { signal });
    document.addEventListener("pointerdown", event => {
      if (this.#isOpen() && !event.composedPath().includes(this)) this.close();
    }, { signal });
    window.addEventListener("resize", () => {
      if (this.#supportsPopover && this.#isOpen()) this.#positionPopover();
    }, { signal });
  }

  disconnectedCallback() {
    this.#controller?.abort();
    this.#controller = undefined;
    this.close();
  }

  open() {
    if (!this.#toggle || !this.#panel || this.#isOpen()) return;
    if (this.#supportsPopover) {
      this.#panel.showPopover();
      this.#positionPopover();
    } else {
      this.#panel.hidden = false;
    }
    this.#toggle.setAttribute("aria-expanded", "true");
    this.#items()[0]?.focus();
  }

  close({ restoreFocus = false } = {}) {
    if (!this.#toggle || !this.#panel) return;
    if (this.#supportsPopover) {
      if (this.#panel.matches(":popover-open")) this.#panel.hidePopover();
    } else {
      this.#panel.hidden = true;
    }
    this.#toggle.setAttribute("aria-expanded", "false");
    if (restoreFocus && this.isConnected) this.#toggle.focus();
  }

  #handleMenuKeydown(event) {
    const items = this.#items();
    if (!items.length) return;
    const index = items.indexOf(document.activeElement);
    if (event.key === "Escape") {
      event.preventDefault();
      this.close({ restoreFocus: true });
      return;
    }
    if (event.key === "ArrowDown") {
      event.preventDefault();
      items[(index + 1) % items.length].focus();
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      items[(index - 1 + items.length) % items.length].focus();
    } else if (event.key === "Home") {
      event.preventDefault();
      items[0].focus();
    } else if (event.key === "End") {
      event.preventDefault();
      items.at(-1).focus();
    }
  }

  #handleSelection(event) {
    const item = event.target.closest?.("[data-add]");
    if (!item || !this.contains(item) || item.disabled) return;
    const kind = item.dataset.add;
    if (!kind) return;
    this.close();
    this.dispatchEvent(new CustomEvent("balaur-add", {
      bubbles: true,
      composed: true,
      detail: { kind },
    }));
  }

  #handleToggleKeydown(event) {
    if (event.key === "Escape") {
      this.close({ restoreFocus: true });
    } else if (event.key === "ArrowDown" && !this.#isOpen()) {
      event.preventDefault();
      this.open();
    }
  }

  #isOpen() {
    if (!this.#panel) return false;
    return this.#supportsPopover
      ? this.#panel.matches(":popover-open")
      : !this.#panel.hidden;
  }

  #items() {
    return [...this.querySelectorAll(".add-menu-item, [data-add]")]
      .filter((item, index, items) => items.indexOf(item) === index && !item.disabled);
  }

  #positionPopover() {
    const toggleRect = this.#toggle.getBoundingClientRect();
    const panelRect = this.#panel.getBoundingClientRect();
    const gap = 9;
    const edge = 8;
    const left = Math.min(
      Math.max(edge, toggleRect.right - panelRect.width),
      Math.max(edge, innerWidth - panelRect.width - edge),
    );
    const spaceAbove = toggleRect.top - edge;
    const spaceBelow = innerHeight - toggleRect.bottom - edge;
    const openAbove = spaceAbove >= panelRect.height + gap || spaceAbove > spaceBelow;
    const top = openAbove
      ? Math.max(edge, toggleRect.top - panelRect.height - gap)
      : Math.min(toggleRect.bottom + gap, innerHeight - panelRect.height - edge);
    this.#panel.style.inset = `${Math.max(edge, top)}px auto auto ${left}px`;
  }
}
