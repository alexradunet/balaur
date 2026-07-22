function element(name, className = "") {
  const node = document.createElement(name);
  if (className) node.className = className;
  return node;
}

export class BalaurTaskListElement extends HTMLElement {
  #controller;
  #empty;
  #items = [];
  #ready = false;
  #rows = new Map();

  connectedCallback() {
    this.#ensureDom();
    this.#controller?.abort();
    this.#controller = new AbortController();
    this.addEventListener("click", event => this.#handleClick(event), { signal: this.#controller.signal });
    this.#render();
  }

  disconnectedCallback() {
    this.#controller?.abort();
    this.#controller = undefined;
  }

  get items() {
    return this.#items;
  }

  set items(value) {
    if (value === this.#items) return;
    this.#items = Array.isArray(value) ? value : [];
    this.#ensureDom();
    this.#render();
  }

  get emptyMessage() {
    return this.#empty?.textContent || "";
  }

  set emptyMessage(value) {
    this.#ensureDom();
    this.#empty.textContent = String(value || "");
    this.#renderVisibility();
  }

  #ensureDom() {
    if (this.#ready) return;
    this.#empty = element("div", "today-empty");
    this.replaceChildren(this.#empty);
    this.#ready = true;
  }

  #createRow(id) {
    const row = element("article", "today-task");
    row.dataset.taskId = id;
    const complete = element("button", "task-check");
    complete.type = "button";
    complete.dataset.completeTask = "";
    const open = element("button", "task-copy");
    open.type = "button";
    open.dataset.openTask = "";
    const title = element("b");
    const context = element("small");
    open.append(title, context);
    const dates = element("div", "task-dates");
    row.append(complete, open, dates);
    return { row, complete, open, title, context, dates };
  }

  #patchRow(refs, item) {
    const done = item.status === "done";
    refs.row.classList.toggle("done", done);
    refs.complete.textContent = done ? "✓" : "";
    refs.complete.setAttribute("aria-label", `Complete ${String(item.title || "task")}`);
    refs.title.textContent = String(item.title || "Untitled task");
    refs.context.textContent = String(item.context || "Inbox");
    const dates = [];
    if (item.scheduledOn) dates.push(["", item.scheduledOn, `Plan ${String(item.scheduledOn).slice(5)}`]);
    if (item.dueOn) dates.push(["due", item.dueOn, `Due ${String(item.dueOn).slice(5)}`]);
    while (refs.dates.children.length > dates.length) refs.dates.lastElementChild.remove();
    dates.forEach(([className, dateTime, text], index) => {
      const time = refs.dates.children[index] || refs.dates.appendChild(element("time"));
      time.className = className;
      time.dateTime = String(dateTime);
      time.textContent = text;
    });
  }

  #render() {
    if (!this.#ready) return;
    const nextIds = new Set();
    const order = [];
    for (const item of this.#items) {
      const id = typeof item?.id === "string" ? item.id : "";
      if (!id || nextIds.has(id)) continue;
      nextIds.add(id);
      order.push(id);
      let refs = this.#rows.get(id);
      if (!refs) {
        refs = this.#createRow(id);
        this.#rows.set(id, refs);
      }
      this.#patchRow(refs, item);
      // Order is reconciled below without detaching an already focused row.
    }
    for (const [id, refs] of this.#rows) {
      if (nextIds.has(id)) continue;
      refs.row.remove();
      this.#rows.delete(id);
    }
    let cursor = this.#empty.nextSibling;
    for (const id of order) {
      const row = this.#rows.get(id)?.row;
      if (!row) continue;
      if (row !== cursor) this.insertBefore(row, cursor);
      cursor = row.nextSibling;
    }
    this.#renderVisibility();
  }

  #renderVisibility() {
    const hasRows = this.#rows.size > 0;
    this.#empty.hidden = hasRows;
    if (!hasRows && this.firstElementChild !== this.#empty) this.prepend(this.#empty);
  }

  #handleClick(event) {
    const control = event.target.closest?.("[data-complete-task], [data-open-task]");
    const row = control?.closest?.("[data-task-id]");
    if (!control || !row || !this.contains(row) || control.disabled) return;
    const taskId = row.dataset.taskId;
    if (!taskId || !this.#items.some(item => item?.id === taskId)) return;
    const type = control.hasAttribute("data-complete-task")
      ? "balaur-task-complete"
      : "balaur-task-open";
    this.dispatchEvent(new CustomEvent(type, {
      bubbles: true,
      composed: true,
      detail: { taskId },
    }));
  }
}
