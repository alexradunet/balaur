function element(name, className = "") {
  const node = document.createElement(name);
  if (className) node.className = className;
  return node;
}

function fieldValue(control, field) {
  if (field.control === "number") return Math.round(Number(control.value));
  return control.value;
}

export class BalaurInspectorElement extends HTMLElement {
  #actions;
  #colors;
  #controller;
  #editor;
  #empty;
  #fieldRows;
  #fields;
  #form;
  #model = null;
  #renderedModelKey = null;
  #notes;
  #ready = false;
  #title;

  connectedCallback() {
    this.#ensureDom();
    this.#controller?.abort();
    this.#controller = new AbortController();
    const { signal } = this.#controller;
    this.addEventListener("click", event => this.#handleClick(event), { signal });
    this.addEventListener("input", event => this.#handleFieldEvent(event, "input"), { signal });
    this.addEventListener("change", event => this.#handleFieldEvent(event, "change"), { signal });
    this.addEventListener("focusout", event => this.#handleFieldEvent(event, "blur"), { signal });
    this.#render();
  }

  disconnectedCallback() {
    this.#controller?.abort();
    this.#controller = undefined;
  }

  get model() {
    return this.#model;
  }

  set model(value) {
    if (value === this.#model) return;
    this.#model = value && typeof value === "object" ? value : null;
    this.#ensureDom();
    this.#render();
  }

  #ensureDom() {
    if (this.#ready) return;
    this.#empty = element("div", "inspector-empty");
    const emptyIcon = element("span");
    emptyIcon.textContent = "↖";
    const emptyTitle = element("h3");
    emptyTitle.textContent = "Nothing selected";
    const emptyCopy = element("p");
    emptyCopy.textContent = "Select a card or connection to edit its details.";
    this.#empty.append(emptyIcon, emptyTitle, emptyCopy);

    this.#editor = element("div", "inspector-editor");
    const header = element("div", "inspector-head");
    this.#title = element("h3");
    const close = element("button", "close-inspector");
    close.type = "button";
    close.dataset.inspectorClose = "";
    close.setAttribute("aria-label", "Close inspector");
    close.textContent = "×";
    header.append(this.#title, close);
    this.#form = element("form", "inspector-form");
    this.#form.addEventListener("submit", event => event.preventDefault());
    this.#notes = element("div", "inspector-notes");
    this.#colors = element("div", "inspector-colors");
    this.#actions = element("div", "inspector-actions");
    this.#form.append(this.#notes, this.#colors, this.#actions);
    this.#editor.append(header, this.#form);
    this.replaceChildren(this.#empty, this.#editor);
    this.#fields = new Map();
    this.#fieldRows = new Map();
    this.#ready = true;
  }

  #createField(field) {
    const label = element("label", "field");
    label.dataset.field = field.key;
    const caption = element("span");
    let control;
    if (field.control === "textarea") control = element("textarea");
    else if (field.control === "select") control = element("select");
    else {
      control = element("input");
      control.type = ["number", "date", "url"].includes(field.control) ? field.control : "text";
    }
    control.dataset.fieldKey = field.key;
    label.append(caption, control);
    return { label, caption, control, optionsSignature: "" };
  }

  #patchField(refs, field, modelChanged) {
    refs.caption.textContent = String(field.label || field.key);
    refs.control.disabled = Boolean(this.#model?.readonly || field.disabled);
    refs.control.title = refs.control.disabled ? String(this.#model?.readonlyMessage || "Canonical files are read-only until repaired or restored.") : "";
    if (field.control === "select") {
      const options = Array.isArray(field.options) ? field.options : [];
      const signature = JSON.stringify(options.map(option => [option.value, option.label]));
      if (refs.optionsSignature !== signature) {
        refs.control.replaceChildren(...options.map(option => {
          const node = element("option");
          node.value = String(option.value ?? "");
          node.textContent = String(option.label ?? option.value ?? "");
          return node;
        }));
        refs.optionsSignature = signature;
      }
    }
    const value = field.value == null ? "" : String(field.value);
    if ((modelChanged || document.activeElement !== refs.control) && refs.control.value !== value) refs.control.value = value;
  }

  #renderFields(fields, modelChanged) {
    const next = new Set();
    const usedRows = new Set();
    const desiredRoots = [];
    const desiredByRow = new Map();
    for (const field of fields) {
      if (!field || typeof field.key !== "string" || !field.key || next.has(field.key)) continue;
      next.add(field.key);
      let refs = this.#fields.get(field.key);
      const desiredTag = field.control === "textarea" ? "TEXTAREA" : field.control === "select" ? "SELECT" : "INPUT";
      if (!refs || refs.control.tagName !== desiredTag || (desiredTag === "INPUT" && refs.control.type !== (["number", "date", "url"].includes(field.control) ? field.control : "text"))) {
        refs?.label.remove();
        refs = this.#createField(field);
        this.#fields.set(field.key, refs);
      }
      this.#patchField(refs, field, modelChanged);
      if (field.row) {
        const rowKey = String(field.row);
        let row = this.#fieldRows.get(rowKey);
        if (!row) {
          row = element("div", "field-row");
          row.dataset.fieldRow = rowKey;
          this.#fieldRows.set(rowKey, row);
        }
        if (!usedRows.has(rowKey)) {
          usedRows.add(rowKey);
          desiredRoots.push(row);
          desiredByRow.set(rowKey, []);
        }
        desiredByRow.get(rowKey).push(refs.label);
      } else {
        desiredRoots.push(refs.label);
      }
    }
    for (const [rowKey, labels] of desiredByRow) {
      const row = this.#fieldRows.get(rowKey);
      let cursor = row.firstChild;
      for (const label of labels) {
        if (label !== cursor) row.insertBefore(label, cursor);
        cursor = label.nextSibling;
      }
    }
    for (const [key, refs] of this.#fields) {
      if (next.has(key)) continue;
      refs.label.remove();
      this.#fields.delete(key);
    }
    for (const [key, row] of this.#fieldRows) {
      if (usedRows.has(key)) continue;
      row.remove();
      this.#fieldRows.delete(key);
    }
    let cursor = this.#form.firstChild;
    for (const node of desiredRoots) {
      if (node !== cursor) this.#form.insertBefore(node, cursor);
      cursor = node.nextSibling;
    }
  }

  #renderNotes(notes) {
    const desired = Array.isArray(notes) ? notes : [];
    while (this.#notes.children.length > desired.length) this.#notes.lastElementChild.remove();
    desired.forEach((note, index) => {
      const node = this.#notes.children[index] || this.#notes.appendChild(element("div", "field-hint"));
      node.textContent = String(note?.text || note || "");
    });
    this.#notes.hidden = desired.length === 0;
  }

  #renderColors(colors) {
    const desired = Array.isArray(colors) ? colors : [];
    this.#colors.hidden = desired.length === 0;
    if (!desired.length) {
      this.#colors.replaceChildren();
      return;
    }
    let label = this.#colors.querySelector(":scope > span");
    let list = this.#colors.querySelector(":scope > .color-list");
    if (!label || !list) {
      label = element("span");
      label.textContent = "Color preset";
      list = element("div", "color-list");
      this.#colors.replaceChildren(label, list);
    }
    const buttons = new Map([...list.querySelectorAll("button[data-color]")].map(button => [button.dataset.color, button]));
    const next = new Set();
    for (const color of desired) {
      const value = String(color?.value || "");
      if (!value || next.has(value)) continue;
      next.add(value);
      let button = buttons.get(value);
      if (!button) {
        button = element("button", "color-choice");
        button.type = "button";
        button.dataset.color = value;
      }
      button.classList.toggle("active", Boolean(color.active));
      button.style.background = String(color.color || "");
      button.setAttribute("aria-label", `Color ${value}`);
      button.disabled = Boolean(this.#model?.readonly);
      list.append(button);
    }
    for (const [value, button] of buttons) if (!next.has(value)) button.remove();
  }

  #renderActions(actions) {
    const desired = Array.isArray(actions) ? actions : [];
    const buttons = new Map([...this.#actions.querySelectorAll("button[data-intent]")].map(button => [button.dataset.intent, button]));
    const next = new Set();
    for (const action of desired) {
      const intent = String(action?.intent || "");
      if (!intent || next.has(intent)) continue;
      next.add(intent);
      let button = buttons.get(intent);
      if (!button) {
        button = element("button");
        button.type = "button";
        button.dataset.intent = intent;
      }
      button.className = action.danger ? "danger-btn" : String(action.className || "button ghost");
      button.textContent = String(action.label || intent);
      button.disabled = Boolean(this.#model?.readonly && action.requiresWrite !== false);
      this.#actions.append(button);
    }
    for (const [intent, button] of buttons) if (!next.has(intent)) button.remove();
    this.#actions.hidden = next.size === 0;
  }

  #render() {
    if (!this.#ready) return;
    const model = this.#model;
    const modelKey = model ? String(model.key || "") : null;
    const modelChanged = modelKey !== this.#renderedModelKey;
    this.#empty.hidden = Boolean(model);
    this.#editor.hidden = !model;
    if (!model) {
      this.#renderedModelKey = null;
      return;
    }
    this.#title.textContent = String(model.title || "Inspector");
    this.#renderFields(Array.isArray(model.fields) ? model.fields : [], modelChanged);
    this.#renderNotes(model.notes);
    this.#renderColors(model.colors);
    this.#renderActions(model.actions);
    this.#renderedModelKey = modelKey;
  }

  #detailForField(control) {
    const field = this.#model?.fields?.find(candidate => candidate?.key === control.dataset.fieldKey);
    if (!field) return null;
    return {
      key: field.key,
      value: fieldValue(control, field),
      scope: field.scope || "item",
      modelKey: String(this.#model.key || ""),
      taskId: field.taskId || null,
      canvasId: field.canvasId || null,
    };
  }

  #handleFieldEvent(event, phase) {
    const control = event.target.closest?.("[data-field-key]");
    if (!control || !this.contains(control)) return;
    const detail = this.#detailForField(control);
    if (!detail) return;
    this.dispatchEvent(new CustomEvent(`balaur-inspector-field-${phase}`, {
      bubbles: true,
      composed: true,
      detail,
    }));
  }

  #handleClick(event) {
    const close = event.target.closest?.("[data-inspector-close]");
    if (close && this.contains(close)) {
      this.dispatchEvent(new CustomEvent("balaur-inspector-close", { bubbles: true, composed: true }));
      return;
    }
    const color = event.target.closest?.("[data-color]");
    if (color && this.contains(color) && !color.disabled) {
      this.dispatchEvent(new CustomEvent("balaur-inspector-color", {
        bubbles: true,
        composed: true,
        detail: { value: color.dataset.color, modelKey: String(this.#model?.key || "") },
      }));
      return;
    }
    const action = event.target.closest?.("[data-intent]");
    if (!action || !this.contains(action) || action.disabled) return;
    const configured = this.#model?.actions?.find(candidate => candidate?.intent === action.dataset.intent);
    if (!configured) return;
    this.dispatchEvent(new CustomEvent("balaur-inspector-action", {
      bubbles: true,
      composed: true,
      detail: {
        intent: action.dataset.intent,
        modelKey: String(this.#model?.key || ""),
        taskId: configured.taskId || null,
        cardId: configured.cardId || null,
        canvasId: configured.canvasId || null,
      },
    }));
  }
}
