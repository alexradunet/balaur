export class BalaurDialogFrameElement extends HTMLElement {
  constructor() {
    super();
    const root = this.attachShadow({ mode: "open" });
    const style = document.createElement("style");
    style.textContent = ":host{display:block;min-inline-size:0}:host([hidden]){display:none!important}.dialog-frame{display:contents}";
    const frame = document.createElement("div");
    frame.className = "dialog-frame";
    frame.part = "frame";
    const before = document.createElement("slot");
    before.name = "before";
    const content = document.createElement("slot");
    const after = document.createElement("slot");
    after.name = "after";
    frame.append(before, content, after);
    root.append(style, frame);
  }
}
