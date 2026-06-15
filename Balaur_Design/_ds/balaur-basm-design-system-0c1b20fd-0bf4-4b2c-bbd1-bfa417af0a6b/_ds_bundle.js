/* @ds-bundle: {"format":3,"namespace":"BalaurBasmDesignSystem_0c1b20","components":[{"name":"Avatar","sourcePath":"components/chat/Avatar.jsx"},{"name":"ChatBar","sourcePath":"components/chat/ChatBar.jsx"},{"name":"ChatMessage","sourcePath":"components/chat/ChatMessage.jsx"},{"name":"DialogueChoices","sourcePath":"components/chat/DialogueChoices.jsx"},{"name":"MessageDraft","sourcePath":"components/chat/MessageDraft.jsx"},{"name":"ToolRow","sourcePath":"components/chat/ToolRow.jsx"},{"name":"Button","sourcePath":"components/core/Button.jsx"},{"name":"Card","sourcePath":"components/core/Card.jsx"},{"name":"FolkBand","sourcePath":"components/core/FolkBand.jsx"},{"name":"Pips","sourcePath":"components/core/Pips.jsx"},{"name":"Stitch","sourcePath":"components/core/Stitch.jsx"},{"name":"Tag","sourcePath":"components/core/Tag.jsx"},{"name":"KnowledgeCard","sourcePath":"components/knowledge/KnowledgeCard.jsx"},{"name":"TaskCard","sourcePath":"components/knowledge/TaskCard.jsx"},{"name":"Tabs","sourcePath":"components/navigation/Tabs.jsx"},{"name":"Topbar","sourcePath":"components/navigation/Topbar.jsx"},{"name":"BalaurApp","sourcePath":"ui_kits/balaur/BalaurApp.jsx"},{"name":"ChatScreen","sourcePath":"ui_kits/balaur/ChatScreen.jsx"},{"name":"LifeScreen","sourcePath":"ui_kits/balaur/LifeScreen.jsx"},{"name":"MemoryScreen","sourcePath":"ui_kits/balaur/MemoryScreen.jsx"},{"name":"ProfileScreen","sourcePath":"ui_kits/balaur/ProfileScreen.jsx"},{"name":"TasksScreen","sourcePath":"ui_kits/balaur/TasksScreen.jsx"}],"sourceHashes":{"components/chat/Avatar.jsx":"4e3ed7295a3c","components/chat/ChatBar.jsx":"7ac1b6179db4","components/chat/ChatMessage.jsx":"d6d1e114c33f","components/chat/DialogueChoices.jsx":"8990294f41e1","components/chat/MessageDraft.jsx":"0a1bb1d3145e","components/chat/ToolRow.jsx":"0d2c85ef10f4","components/core/Button.jsx":"5320b72eb303","components/core/Card.jsx":"7b36b9f2e5a2","components/core/FolkBand.jsx":"50c52197457b","components/core/Pips.jsx":"c0ca9db89459","components/core/Stitch.jsx":"4980047b57b0","components/core/Tag.jsx":"e03cea5c14b9","components/knowledge/KnowledgeCard.jsx":"e3218071bb47","components/knowledge/TaskCard.jsx":"f31da62289ce","components/navigation/Tabs.jsx":"d027da04e185","components/navigation/Topbar.jsx":"c535b17f2917","ui_kits/balaur/BalaurApp.jsx":"cc71f64a9b7e","ui_kits/balaur/ChatScreen.jsx":"d4b6454f0ce6","ui_kits/balaur/LifeScreen.jsx":"cf1268c11773","ui_kits/balaur/MemoryScreen.jsx":"4f06214f3a37","ui_kits/balaur/ProfileScreen.jsx":"9e9ed8133e0f","ui_kits/balaur/TasksScreen.jsx":"c35f63954a00"},"inlinedExternals":[],"unexposedExports":[{"name":"matchChoice","sourcePath":"components/chat/MessageDraft.jsx"}]} */

(() => {

const __ds_ns = (window.BalaurBasmDesignSystem_0c1b20 = window.BalaurBasmDesignSystem_0c1b20 || {});

const __ds_scope = {};

(__ds_ns.__errors = __ds_ns.__errors || []);

// components/chat/Avatar.jsx
try { (() => {
/** Basm avatar — borderless static pixel PNG, right-facing profile.
    Activity (thinking/working) is shown by the basm-glow CSS breathing
    teal glow, never frame animation. */
function Avatar({
  src,
  kind = 'balaur',
  size = 54,
  state = 'idle',
  alt = ''
}) {
  return /*#__PURE__*/React.createElement("span", {
    className: 'balaur-avatar balaur-avatar-' + kind,
    "data-kind": kind,
    "data-state": state,
    style: {
      '--avatar-size': size + 'px'
    },
    "aria-hidden": alt ? undefined : true
  }, /*#__PURE__*/React.createElement("img", {
    src: src,
    alt: alt,
    decoding: "async"
  }));
}
Object.assign(__ds_scope, { Avatar });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/chat/Avatar.jsx", error: String((e && e.message) || e) }); }

// components/chat/ChatBar.jsx
try { (() => {
/** Basm composer — the fixed wood-ledge input box, shared across every
    surface that takes the owner's words (chat, task capture). The `above`
    slot holds ledge-level chrome like the model switcher; `sendLabel` and
    `placeholder` let each surface speak in its own voice while the input
    stays identical. Enter sends, Shift+Enter breaks. */
function ChatBar({
  placeholder = 'Speak; I am listening.',
  sendLabel = 'Send',
  above,
  onSend,
  fixed = true,
  rows = 2
}) {
  const ref = React.useRef(null);
  function submit(e) {
    if (e) e.preventDefault();
    const v = ref.current ? ref.current.value.trim() : '';
    if (!v) return;
    if (onSend) onSend(v);
    if (ref.current) ref.current.value = '';
  }
  function onKeyDown(e) {
    if (e.key === 'Enter' && !e.shiftKey && !e.altKey && !e.ctrlKey && !e.metaKey) {
      submit(e);
    }
  }
  return /*#__PURE__*/React.createElement("div", {
    className: 'chatbar' + (fixed ? '' : ' chatbar-static'),
    style: fixed ? undefined : {
      position: 'static'
    }
  }, above ? above : null, /*#__PURE__*/React.createElement("form", {
    className: "chat-form",
    onSubmit: submit
  }, /*#__PURE__*/React.createElement("textarea", {
    ref: ref,
    name: "message",
    placeholder: placeholder,
    autoComplete: "off",
    rows: rows,
    onKeyDown: onKeyDown
  }), /*#__PURE__*/React.createElement("button", {
    className: "btn btn-primary",
    type: "submit"
  }, sendLabel)));
}
Object.assign(__ds_scope, { ChatBar });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/chat/ChatBar.jsx", error: String((e && e.message) || e) }); }

// components/chat/ChatMessage.jsx
try { (() => {
/* RPG typewriter — reveals text per character; instant under
   prefers-reduced-motion. The parent panel click skips to the end. */
function TypewriterText({
  text,
  speed = 14,
  skipSignal
}) {
  const reduced = typeof window !== 'undefined' && window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  const [n, setN] = React.useState(reduced ? text.length : 0);
  React.useEffect(() => {
    if (skipSignal) {
      setN(text.length);
      return;
    }
    if (n >= text.length) return;
    const t = window.setTimeout(() => setN(n + 1), speed);
    return () => window.clearTimeout(t);
  }, [n, text, speed, skipSignal]);
  const typing = n < text.length;
  return /*#__PURE__*/React.createElement("span", null, text.slice(0, n), typing ? /*#__PURE__*/React.createElement("span", {
    className: "type-cursor",
    "aria-hidden": "true"
  }, "\u258E") : null);
}

/** Basm chat message — RPG speech panel with nameplate tab.
    role 'user' (indigo nameplate) | 'balaur' (gold) | 'tool' (teal, no avatar).
    pending shows the thinking dots; reveal types the line out (click to skip). */
function ChatMessage({
  role = 'balaur',
  who,
  avatarSrc,
  pending,
  origin,
  reveal,
  children
}) {
  if (role === 'tool') {
    return /*#__PURE__*/React.createElement("div", {
      className: "msg msg-tool"
    }, /*#__PURE__*/React.createElement("div", {
      className: "who"
    }, who), /*#__PURE__*/React.createElement("div", {
      className: "body"
    }, children));
  }
  const cls = ['msg', role === 'user' ? 'msg-user' : 'msg-balaur', pending ? 'msg-pending' : '', 'msg-with-avatar'].filter(Boolean).join(' ');
  const label = who || (role === 'user' ? 'You' : 'Balaur');
  const [skipped, setSkipped] = React.useState(false);
  const useReveal = reveal && !pending && typeof children === 'string';
  return /*#__PURE__*/React.createElement("div", {
    className: cls
  }, /*#__PURE__*/React.createElement("figure", {
    className: "portrait"
  }, /*#__PURE__*/React.createElement(__ds_scope.Avatar, {
    src: avatarSrc,
    kind: role === 'user' ? 'soul' : 'balaur',
    state: pending ? 'thinking' : 'idle'
  }), /*#__PURE__*/React.createElement("figcaption", {
    className: "who"
  }, label, origin ? ' · ' + origin : '')), /*#__PURE__*/React.createElement("div", {
    className: "msg-main",
    onClick: useReveal && !skipped ? () => setSkipped(true) : undefined,
    style: useReveal && !skipped ? {
      cursor: 'pointer'
    } : undefined
  }, /*#__PURE__*/React.createElement("div", {
    className: "body"
  }, pending ? /*#__PURE__*/React.createElement("span", {
    className: "thinking thinking-dots"
  }, "thinking") : useReveal ? /*#__PURE__*/React.createElement(TypewriterText, {
    text: children,
    skipSignal: skipped
  }) : children)));
}
Object.assign(__ds_scope, { ChatMessage });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/chat/ChatMessage.jsx", error: String((e && e.message) || e) }); }

// components/chat/DialogueChoices.jsx
try { (() => {
/** RPG dialogue choices — the owner's reply as a multiple-choice prompt.
    Mirrors the owner's message row: ornate parchment panel at full column
    width, framed soul portrait at the right. Numbered; keys 1–9 pick,
    ↑/↓ move, Enter speaks. */
function DialogueChoices({
  kicker = 'Your word',
  who = 'You',
  avatarSrc,
  choices = [],
  onPick
}) {
  const panelRef = React.useRef(null);
  function pick(c, i) {
    if (c.onPick) c.onPick(c, i);else if (onPick) onPick(c, i);
  }
  React.useEffect(() => {
    function onKey(e) {
      const t = e.target;
      if (t && (t.tagName === 'TEXTAREA' || t.tagName === 'INPUT' || t.tagName === 'SELECT' || t.isContentEditable)) return;
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      const digit = parseInt(e.key, 10);
      if (digit >= 1 && digit <= choices.length) {
        e.preventDefault();
        pick(choices[digit - 1], digit - 1);
        return;
      }
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        const btns = panelRef.current ? Array.prototype.slice.call(panelRef.current.querySelectorAll('.choice')) : [];
        if (!btns.length) return;
        e.preventDefault();
        const at = btns.indexOf(document.activeElement);
        const next = e.key === 'ArrowDown' ? at < 0 ? 0 : (at + 1) % btns.length : at < 0 ? btns.length - 1 : (at - 1 + btns.length) % btns.length;
        btns[next].focus();
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  });
  return /*#__PURE__*/React.createElement("div", {
    className: "msg choices",
    role: "group",
    "aria-label": kicker
  }, /*#__PURE__*/React.createElement("div", {
    className: "choices-panel",
    ref: panelRef
  }, /*#__PURE__*/React.createElement("div", {
    className: "choices-kicker"
  }, kicker), choices.map((c, i) => /*#__PURE__*/React.createElement("button", {
    key: i,
    type: "button",
    className: "choice",
    onClick: () => pick(c, i)
  }, /*#__PURE__*/React.createElement("span", {
    className: "choice-key",
    "aria-hidden": "true"
  }, i + 1), /*#__PURE__*/React.createElement("span", {
    className: "choice-label"
  }, c.label), c.hint ? /*#__PURE__*/React.createElement("span", {
    className: "choice-hint"
  }, c.hint) : null))), avatarSrc ? /*#__PURE__*/React.createElement("figure", {
    className: "portrait"
  }, /*#__PURE__*/React.createElement(__ds_scope.Avatar, {
    src: avatarSrc,
    kind: "soul"
  }), /*#__PURE__*/React.createElement("figcaption", {
    className: "who"
  }, who)) : null);
}
Object.assign(__ds_scope, { DialogueChoices });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/chat/DialogueChoices.jsx", error: String((e && e.message) || e) }); }

// components/chat/MessageDraft.jsx
try { (() => {
/** Message draft — the composer rendered as an unsent message at the end
    of the conversation: dashed parchment panel in the owner's position
    (portrait right), which "becomes" the next spoken message on send.
    Enter sends, Shift+Enter breaks. */
function MessageDraft({
  who = 'You',
  avatarSrc,
  placeholder = 'Speak; I am listening.',
  sendLabel = 'Send',
  onSend
}) {
  const ref = React.useRef(null);
  function autosize() {
    const el = ref.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = Math.min(el.scrollHeight, window.innerHeight * 0.28) + 'px';
  }
  function submit(e) {
    if (e) e.preventDefault();
    const v = ref.current ? ref.current.value.trim() : '';
    if (!v) return;
    if (onSend) onSend(v);
    if (ref.current) {
      ref.current.value = '';
      autosize();
      ref.current.focus();
    }
  }
  return /*#__PURE__*/React.createElement("form", {
    className: "msg msg-user msg-draft",
    onSubmit: submit
  }, /*#__PURE__*/React.createElement("div", {
    className: "msg-main"
  }, /*#__PURE__*/React.createElement("textarea", {
    ref: ref,
    name: "message",
    placeholder: placeholder,
    autoComplete: "off",
    rows: 2,
    onInput: autosize,
    onKeyDown: e => {
      if (e.key === 'Enter' && !e.shiftKey && !e.altKey && !e.ctrlKey && !e.metaKey) submit(e);
    }
  }), /*#__PURE__*/React.createElement("div", {
    className: "msg-draft-foot"
  }, /*#__PURE__*/React.createElement("span", {
    className: "msg-draft-hint"
  }, "unsent \xB7 enter speaks"), /*#__PURE__*/React.createElement("button", {
    className: "btn btn-primary btn-sm",
    type: "submit"
  }, sendLabel))), /*#__PURE__*/React.createElement("figure", {
    className: "portrait"
  }, /*#__PURE__*/React.createElement(__ds_scope.Avatar, {
    src: avatarSrc,
    kind: "soul"
  }), /*#__PURE__*/React.createElement("figcaption", {
    className: "who"
  }, who)));
}

/** Match typed input against active dialogue choices: the number (1–9),
    the spoken label, the mono hint, or the kind — punctuation-insensitive.
    Returns the choice and its index, or null. */
function matchChoice(text, choices) {
  if (!choices || !choices.length) return null;
  const norm = s => String(s || '').toLowerCase().replace(/[.,!?'"’]/g, '').trim();
  const t = norm(text);
  if (!t) return null;
  if (/^[1-9]$/.test(t)) {
    const i = parseInt(t, 10) - 1;
    return i < choices.length ? {
      choice: choices[i],
      index: i
    } : null;
  }
  for (let i = 0; i < choices.length; i++) {
    const c = choices[i];
    if (norm(c.label) === t || norm(c.hint) === t || norm(c.kind) === t) {
      return {
        choice: c,
        index: i
      };
    }
  }
  return null;
}
Object.assign(__ds_scope, { MessageDraft, matchChoice });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/chat/MessageDraft.jsx", error: String((e && e.message) || e) }); }

// components/chat/ToolRow.jsx
try { (() => {
/** Tool/OS-access event row — bordered, mono-typeset, visible audit trail.
    Prefers a pixel-art icon (assets/icons/) over the legacy text glyph;
    the icon is bare imagery inside the row — the row's border is the frame. */
function ToolRow({
  tool,
  icon,
  glyph = '⌥',
  children
}) {
  return /*#__PURE__*/React.createElement("div", {
    className: "msg msg-tool"
  }, /*#__PURE__*/React.createElement("div", {
    className: "who"
  }, icon ? /*#__PURE__*/React.createElement("img", {
    className: "tool-icon",
    src: icon,
    alt: "",
    decoding: "async"
  }) : /*#__PURE__*/React.createElement("span", {
    className: "tool-icon",
    "aria-hidden": "true"
  }, glyph), "tool \xB7 ", tool, /*#__PURE__*/React.createElement("span", {
    style: {
      opacity: .6
    }
  }, " \u2014 ")), /*#__PURE__*/React.createElement("div", {
    className: "body"
  }, children));
}
Object.assign(__ds_scope, { ToolRow });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/chat/ToolRow.jsx", error: String((e && e.message) || e) }); }

// components/core/Button.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/** Basm button — beveled 16-bit slab; hover brightens, press sinks 3px
    and inverts the bevel (the press feels physical). */
function Button({
  variant = 'primary',
  size,
  href,
  children,
  ...rest
}) {
  const cls = ['btn', variant === 'ghost' ? 'btn-ghost' : variant === 'wood' ? 'btn-wood' : 'btn-primary', size === 'sm' ? 'btn-sm' : ''].filter(Boolean).join(' ');
  if (href) {
    return /*#__PURE__*/React.createElement("a", _extends({
      className: cls,
      href: href
    }, rest), children);
  }
  return /*#__PURE__*/React.createElement("button", _extends({
    type: "button",
    className: cls
  }, rest), children);
}
Object.assign(__ds_scope, { Button });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/core/Button.jsx", error: String((e && e.message) || e) }); }

// components/core/Card.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/** Basm card — surface, 2px outline, 3px radius, gold pixel notch top-right.
    The notch comes from the .card::after CSS rule. */
function Card({
  children,
  style,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("div", _extends({
    className: "card",
    style: style
  }, rest), children);
}
Object.assign(__ds_scope, { Card });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/core/Card.jsx", error: String((e && e.message) || e) }); }

// components/core/FolkBand.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/** Folk band — horizontal carpet stripe (folkred/gold/teal/ember-deep at 135°)
    with 2px fg borders. Use sparingly in dense app UI. */
function FolkBand({
  style,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("div", _extends({
    className: "folk-band",
    style: style
  }, rest));
}
Object.assign(__ds_scope, { FolkBand });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/core/FolkBand.jsx", error: String((e && e.message) || e) }); }

// components/core/Pips.jsx
try { (() => {
/** Importance pips — five 7px squares, gold-filled to level.
    The context-budget dial: importance ≥ 4 means always injected. */
function Pips({
  level = 0,
  max = 5,
  title
}) {
  const items = [];
  for (let i = 0; i < max; i++) {
    items.push(/*#__PURE__*/React.createElement("i", {
      key: i,
      className: i < level ? 'pip pip-on' : 'pip'
    }));
  }
  return /*#__PURE__*/React.createElement("span", {
    className: "kcard-pips",
    title: title || 'importance ' + level + '/' + max
  }, items);
}
Object.assign(__ds_scope, { Pips });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/core/Pips.jsx", error: String((e && e.message) || e) }); }

// components/core/Stitch.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/** Stitch line — 2px dashed folk separator between sections. */
function Stitch({
  style,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("div", _extends({
    className: "stitch",
    style: style
  }, rest));
}
Object.assign(__ds_scope, { Stitch });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/core/Stitch.jsx", error: String((e && e.message) || e) }); }

// components/core/Tag.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/** Basm tag — small mono chip with teal ▪ prefix (from CSS ::before). */
function Tag({
  children,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("span", _extends({
    className: "tag"
  }, rest), children);
}
Object.assign(__ds_scope, { Tag });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/core/Tag.jsx", error: String((e && e.message) || e) }); }

// components/knowledge/KnowledgeCard.jsx
try { (() => {
/** Knowledge card (memory or skill) — the growth surface: Balaur proposes,
    the owner decides. Proposed pops (gold border + hard shadow + ember notch,
    Approve as the only primary). Active is calm with Archive ghost + used ×N.
    Archived is dashed and dimmed with Restore. */
function KnowledgeCard({
  kind = 'memory',
  status = 'active',
  title,
  body,
  whenToUse,
  importance,
  usedCount,
  onApprove,
  onDismiss,
  onArchive,
  onRestore
}) {
  return /*#__PURE__*/React.createElement("article", {
    className: 'kcard kcard-' + status
  }, /*#__PURE__*/React.createElement("header", {
    className: "kcard-head"
  }, /*#__PURE__*/React.createElement("span", {
    className: "kcard-kind"
  }, "\u25AA ", kind), typeof importance === 'number' ? /*#__PURE__*/React.createElement(__ds_scope.Pips, {
    level: importance
  }) : null), title ? /*#__PURE__*/React.createElement("h3", {
    className: "kcard-title"
  }, title) : null, body ? /*#__PURE__*/React.createElement("p", {
    className: "kcard-body"
  }, body) : null, whenToUse ? /*#__PURE__*/React.createElement("p", {
    className: "kcard-when"
  }, "recall: ", whenToUse) : null, /*#__PURE__*/React.createElement("footer", {
    className: "kcard-actions"
  }, status === 'proposed' ? /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement(__ds_scope.Button, {
    size: "sm",
    onClick: onApprove
  }, "Approve"), /*#__PURE__*/React.createElement(__ds_scope.Button, {
    variant: "ghost",
    size: "sm",
    onClick: onDismiss
  }, "Dismiss")) : null, status === 'active' ? /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement(__ds_scope.Button, {
    variant: "ghost",
    size: "sm",
    onClick: onArchive
  }, "Archive"), usedCount ? /*#__PURE__*/React.createElement("span", {
    className: "kcard-meta"
  }, "used \xD7", usedCount) : null) : null, status === 'archived' ? /*#__PURE__*/React.createElement(__ds_scope.Button, {
    variant: "ghost",
    size: "sm",
    onClick: onRestore
  }, "Restore") : null));
}
Object.assign(__ds_scope, { KnowledgeCard });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/knowledge/KnowledgeCard.jsx", error: String((e && e.message) || e) }); }

// components/knowledge/TaskCard.jsx
try { (() => {
/** Task card — operational action card for chat embeds and the /tasks page.
    Open tasks get Done (primary), Snooze, Drop; closed tasks show status. */
function TaskCard({
  title,
  status = 'open',
  dueLine,
  overdue,
  recurLine,
  notes,
  onDone,
  onSnooze,
  onDrop
}) {
  return /*#__PURE__*/React.createElement("article", {
    className: 'kcard tcard tcard-' + status
  }, /*#__PURE__*/React.createElement("header", {
    className: "kcard-head"
  }, /*#__PURE__*/React.createElement("span", {
    className: "kcard-kind"
  }, "\u25AA task"), recurLine ? /*#__PURE__*/React.createElement(__ds_scope.Tag, null, recurLine) : null), /*#__PURE__*/React.createElement("h3", {
    className: "kcard-title"
  }, title), dueLine ? /*#__PURE__*/React.createElement("p", {
    className: 'tcard-due' + (overdue ? ' tcard-overdue' : '')
  }, dueLine) : null, notes ? /*#__PURE__*/React.createElement("details", {
    className: "kcard-edit"
  }, /*#__PURE__*/React.createElement("summary", null, "Notes"), /*#__PURE__*/React.createElement("p", {
    className: "kcard-body"
  }, notes)) : null, /*#__PURE__*/React.createElement("footer", {
    className: "kcard-actions"
  }, status === 'open' ? /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement(__ds_scope.Button, {
    size: "sm",
    onClick: onDone
  }, "Done"), /*#__PURE__*/React.createElement("span", {
    className: "tcard-snooze"
  }, /*#__PURE__*/React.createElement("select", {
    "aria-label": "Snooze until",
    defaultValue: "1h"
  }, /*#__PURE__*/React.createElement("option", {
    value: "1h"
  }, "+1 hour"), /*#__PURE__*/React.createElement("option", {
    value: "tonight"
  }, "tonight"), /*#__PURE__*/React.createElement("option", {
    value: "tomorrow"
  }, "tomorrow")), /*#__PURE__*/React.createElement(__ds_scope.Button, {
    variant: "ghost",
    size: "sm",
    onClick: onSnooze
  }, "Snooze")), /*#__PURE__*/React.createElement(__ds_scope.Button, {
    variant: "ghost",
    size: "sm",
    onClick: onDrop
  }, "Drop")) : /*#__PURE__*/React.createElement("span", {
    className: "kcard-meta"
  }, status)));
}
Object.assign(__ds_scope, { TaskCard });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/knowledge/TaskCard.jsx", error: String((e && e.message) || e) }); }

// components/navigation/Tabs.jsx
try { (() => {
/** Basm tabs — k-tab pills; active is gold-filled with dark text. */
function Tabs({
  items = [],
  active,
  onSelect
}) {
  return /*#__PURE__*/React.createElement("nav", {
    className: "k-tabs"
  }, items.map(it => /*#__PURE__*/React.createElement("a", {
    key: it,
    className: 'k-tab' + (it === active ? ' k-tab-active' : ''),
    href: "#",
    onClick: e => {
      e.preventDefault();
      if (onSelect) onSelect(it);
    }
  }, it)));
}
Object.assign(__ds_scope, { Tabs });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/navigation/Tabs.jsx", error: String((e && e.message) || e) }); }

// components/navigation/Topbar.jsx
try { (() => {
/** Basm topbar — sticky translucent bar with Silkscreen gold nameplate,
    borderless 34px crest, mono uppercase nav, theme toggle. */
function Topbar({
  brand = 'Balaur',
  crestSrc,
  links = [],
  active,
  onNavigate,
  onToggleTheme,
  themeGlyph = '◑'
}) {
  return /*#__PURE__*/React.createElement("header", {
    className: "topbar"
  }, /*#__PURE__*/React.createElement("a", {
    className: "brand",
    href: "#",
    onClick: e => {
      e.preventDefault();
      if (onNavigate) onNavigate('home');
    }
  }, crestSrc ? /*#__PURE__*/React.createElement("img", {
    className: "crest",
    src: crestSrc,
    alt: "",
    decoding: "async"
  }) : null, brand), /*#__PURE__*/React.createElement("nav", null, links.map(l => /*#__PURE__*/React.createElement("a", {
    key: l,
    href: "#",
    style: l === active ? {
      color: 'var(--gold)'
    } : undefined,
    onClick: e => {
      e.preventDefault();
      if (onNavigate) onNavigate(l);
    }
  }, l))), /*#__PURE__*/React.createElement("button", {
    className: "theme-toggle",
    type: "button",
    title: "Toggle light/dark mode",
    "aria-label": "Toggle light/dark mode",
    onClick: onToggleTheme
  }, themeGlyph));
}
Object.assign(__ds_scope, { Topbar });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/navigation/Topbar.jsx", error: String((e && e.message) || e) }); }

// ui_kits/balaur/ChatScreen.jsx
try { (() => {
/* Reading-speed estimate for sequencing after a typewriter reveal. */
function revealMs(text) {
  return Math.min(text.length * 14 + 500, 3200);
}

/** Home page — the master conversation as RPG dialogue. Every decision
    (task confirmation, memory approval, the evening nudge) is answered
    through dialogue choices; the picked line becomes the owner's reply.
    New Balaur lines type out (click to skip). */
function ChatScreen({
  soulSrc,
  headSrc,
  ownerName = 'You',
  assetBase = '../../assets'
}) {
  const I = n => assetBase + '/icons/' + n + '.png';
  const [messages, setMessages] = React.useState([{
    id: 1,
    role: 'balaur',
    text: 'I am here. The hearth is lit and your words stay on this box. What shall we weigh today?'
  }, {
    id: 2,
    role: 'user',
    text: 'Remind me to water the tomatoes every two days, evenings.'
  }, {
    id: 3,
    role: 'balaur',
    text: 'Every second evening at 18:00, then. Shall I hold you to it?'
  }, {
    id: 4,
    role: 'choices',
    kind: 'confirmTask'
  }]);
  const [pending, setPending] = React.useState(false);
  const nextId = React.useRef(5);
  const timers = React.useRef([]);
  React.useEffect(() => () => timers.current.forEach(window.clearTimeout), []);
  function push(msg, delay = 0) {
    const t = window.setTimeout(() => {
      setMessages(m => m.concat([Object.assign({
        id: nextId.current++
      }, msg)]));
    }, delay);
    timers.current.push(t);
    return delay;
  }
  function speak(choice) {
    setMessages(m => m.filter(x => x.role !== 'choices'));
    push({
      role: 'user',
      text: choice.label
    });
  }
  const CHOICES = {
    confirmTask: [{
      label: 'Hold me to it.',
      hint: 'add recurring task',
      kind: 'hold'
    }, {
      label: 'Just this once.',
      hint: 'add one-time task',
      kind: 'once'
    }, {
      label: 'Let it be.',
      hint: 'cancel',
      kind: 'cancel'
    }],
    memory: [{
      label: 'Keep it.',
      hint: 'approve memory',
      kind: 'approve'
    }, {
      label: 'Not this.',
      hint: 'dismiss',
      kind: 'dismiss'
    }, {
      label: 'Ask me another time.',
      hint: 'later',
      kind: 'later'
    }],
    nudge: [{
      label: 'It is done.',
      hint: 'mark done',
      kind: 'done'
    }, {
      label: 'At nightfall.',
      hint: 'snooze · 21:00',
      kind: 'tonight'
    }, {
      label: 'Tomorrow, I swear it.',
      hint: 'snooze · tomorrow',
      kind: 'tomorrow'
    }, {
      label: 'Let this one go.',
      hint: 'drop task',
      kind: 'drop'
    }]
  };
  function askMemory(after) {
    const q = 'You speak of the garden often. Shall I remember it as yours to tend — tomatoes and peppers, watered at dusk?';
    push({
      role: 'balaur',
      text: q,
      reveal: true
    }, after);
    push({
      role: 'choices',
      kind: 'memory'
    }, after + revealMs(q));
  }
  function sendNudge(after) {
    const q = 'The evening comes, and the tomatoes thirst. Will you tend them now?';
    push({
      role: 'balaur',
      text: q,
      reveal: true,
      origin: 'nudge'
    }, after);
    push({
      role: 'choices',
      kind: 'nudge'
    }, after + revealMs(q));
  }
  function onConfirmTask(choice) {
    speak(choice);
    if (choice.kind === 'cancel') {
      push({
        role: 'balaur',
        text: 'Then I shall not speak of it again.',
        reveal: true
      }, 500);
      return;
    }
    const once = choice.kind === 'once';
    push({
      role: 'tool',
      tool: 'task_add',
      icon: 'quill',
      text: once ? 'added task: water the tomatoes · today 18:00' : 'added task: water the tomatoes · every 2 days 18:00'
    }, 500);
    push({
      role: 'task',
      once
    }, 680);
    const line = once ? 'Once, then. Today at 18:00 — I shall speak of it when the hour comes.' : 'It is written. I shall nudge you every second evening; the first falls today at 18:00.';
    push({
      role: 'balaur',
      text: line,
      reveal: true
    }, 950);
    askMemory(950 + revealMs(line) + 600);
  }
  function onMemory(choice) {
    speak(choice);
    if (choice.kind === 'approve') {
      push({
        role: 'tool',
        tool: 'memory_keep',
        icon: 'tome',
        text: 'memory kept: "Garden — tomatoes & peppers, watered at dusk" · importance 3'
      }, 500);
      const line = 'Then it is written. The garden is part of your story now.';
      push({
        role: 'balaur',
        text: line,
        reveal: true
      }, 800);
      sendNudge(800 + revealMs(line) + 900);
    } else if (choice.kind === 'dismiss') {
      push({
        role: 'balaur',
        text: 'Then it passes from me, unkept.',
        reveal: true
      }, 500);
      sendNudge(2400);
    } else {
      push({
        role: 'balaur',
        text: 'As you wish. I shall ask another day.',
        reveal: true
      }, 500);
      sendNudge(2400);
    }
  }
  function onNudge(choice) {
    speak(choice);
    if (choice.kind === 'done') {
      push({
        role: 'tool',
        tool: 'task_done',
        icon: 'check',
        text: 'logged: water the tomatoes · done 18:42 → entries'
      }, 500);
      push({
        role: 'balaur',
        text: 'Well tended. The streak holds — four evenings now.',
        reveal: true
      }, 800);
    } else if (choice.kind === 'tonight') {
      push({
        role: 'tool',
        tool: 'task_snooze',
        icon: 'hourglass',
        text: 'snoozed: water the tomatoes · until 21:00'
      }, 500);
      push({
        role: 'balaur',
        text: 'At nightfall, then. I shall ask once more, and only once.',
        reveal: true
      }, 800);
    } else if (choice.kind === 'tomorrow') {
      push({
        role: 'tool',
        tool: 'task_snooze',
        icon: 'hourglass',
        text: 'snoozed: water the tomatoes · until tomorrow 18:00'
      }, 500);
      push({
        role: 'balaur',
        text: 'Sworn and noted. Tomorrow at dusk.',
        reveal: true
      }, 800);
    } else {
      push({
        role: 'tool',
        tool: 'task_drop',
        icon: 'rune_x',
        text: 'dropped: water the tomatoes'
      }, 500);
      push({
        role: 'balaur',
        text: 'Dropped. No grudge is kept.',
        reveal: true
      }, 800);
    }
  }
  const onPick = {
    confirmTask: onConfirmTask,
    memory: onMemory,
    nudge: onNudge
  };

  /* The one input: if dialogue choices are open, typed words can answer
     them ("1", "keep it", "approve memory"); otherwise it is free speech. */
  function handleDraft(text) {
    const open = messages.find(m => m.role === 'choices');
    if (open) {
      const hit = __ds_scope.matchChoice(text, CHOICES[open.kind]);
      if (hit) {
        onPick[open.kind](hit.choice);
        return;
      }
    }
    send(text);
  }
  function send(text) {
    push({
      role: 'user',
      text
    });
    setPending(true);
    const t = window.setTimeout(() => {
      setPending(false);
      push({
        role: 'balaur',
        text: 'I shall weigh the matter. It is kept on this box, and nowhere else.',
        reveal: true
      });
    }, 1400);
    timers.current.push(t);
  }
  const openChoices = messages.find(m => m.role === 'choices');
  React.useEffect(() => {
    window.scrollTo(0, document.body.scrollHeight);
  }, [messages, pending]);
  return /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("main", null, /*#__PURE__*/React.createElement("div", {
    className: "recap-zone"
  }, /*#__PURE__*/React.createElement("p", {
    className: "recap-hint"
  }, /*#__PURE__*/React.createElement("img", {
    src: I('orb'),
    alt: "",
    style: {
      width: 15,
      height: 15,
      imageRendering: 'pixelated',
      verticalAlign: '-3px',
      marginRight: 6
    }
  }), "further back\u2026")), /*#__PURE__*/React.createElement("section", {
    className: "chat",
    id: "chat",
    "aria-live": "polite",
    style: {
      paddingBottom: 48
    }
  }, messages.map(m => {
    if (m.role === 'tool') {
      return /*#__PURE__*/React.createElement(__ds_scope.ToolRow, {
        key: m.id,
        tool: m.tool,
        icon: m.icon ? I(m.icon) : undefined,
        glyph: m.glyph
      }, m.text);
    }
    if (m.role === 'task') {
      return /*#__PURE__*/React.createElement("div", {
        key: m.id,
        className: "k-inline"
      }, /*#__PURE__*/React.createElement(__ds_scope.TaskCard, {
        title: "Water the tomatoes",
        dueLine: "due today 18:00",
        recurLine: m.once ? undefined : 'every 2 days'
      }));
    }
    if (m.role === 'choices') {
      return null; /* rendered at the composer position below */
    }
    return /*#__PURE__*/React.createElement(__ds_scope.ChatMessage, {
      key: m.id,
      role: m.role,
      who: m.role === 'user' ? ownerName : 'Balaur',
      avatarSrc: m.role === 'user' ? soulSrc : headSrc,
      origin: m.origin,
      reveal: m.reveal
    }, m.text);
  }), pending ? /*#__PURE__*/React.createElement(__ds_scope.ChatMessage, {
    role: "balaur",
    avatarSrc: headSrc,
    pending: true
  }) : null, openChoices ?
  /*#__PURE__*/
  /* Balaur awaits your word — the choices take the composer's
     place, and the draft returns once you have spoken. */
  React.createElement(__ds_scope.DialogueChoices, {
    kicker: "Your word",
    who: ownerName,
    avatarSrc: soulSrc,
    choices: CHOICES[openChoices.kind],
    onPick: onPick[openChoices.kind]
  }) : /*#__PURE__*/React.createElement(__ds_scope.MessageDraft, {
    who: ownerName,
    avatarSrc: soulSrc,
    onSend: handleDraft
  }))));
}
Object.assign(__ds_scope, { ChatScreen });
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/balaur/ChatScreen.jsx", error: String((e && e.message) || e) }); }

// ui_kits/balaur/LifeScreen.jsx
try { (() => {
/* Light parse of a typed "log a measure" line into a life entry.
   Numeric kinds (kind 42, weight 81.2kg) get a value; text kinds keep prose. */
function logEntry(text) {
  const clean = text.trim().replace(/\s+/g, ' ');
  const num = clean.match(/(-?\d+(?:\.\d+)?)\s*([a-zA-Z%]+)?/);
  const kindWord = clean.match(/^([a-zA-Z][\w-]*)/);
  const kind = kindWord ? kindWord[1].toLowerCase() : 'note';
  if (num) {
    return {
      kind,
      numeric: true,
      value: num[1],
      unit: num[2] || ''
    };
  }
  return {
    kind,
    numeric: false,
    text: clean
  };
}

/** /life — owner-defined life log: habit streaks, numeric kinds with
    sparklines, text kinds with recent lines. Mirrors what is actually
    logged. Owner logs a measure through the shared composer. */
function LifeScreen({
  assetBase = '../../assets'
}) {
  const weight = '78,4 76,10 74,18 70,14 66,20 62,16 58,24 54,20 50,26 46,22';
  const [entries, setEntries] = React.useState([]);
  const [note, setNote] = React.useState(null);
  function log(text) {
    const e = logEntry(text);
    setEntries(es => [Object.assign({
      id: Date.now()
    }, e)].concat(es).slice(0, 6));
    setNote(e);
  }
  return /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("main", null, /*#__PURE__*/React.createElement("section", {
    className: "k-section"
  }, /*#__PURE__*/React.createElement("h2", {
    className: "k-heading"
  }, /*#__PURE__*/React.createElement("img", {
    className: "h-icon",
    src: assetBase + '/icons/flame.png',
    alt: ""
  }), " Habits"), /*#__PURE__*/React.createElement("div", {
    className: "habit-strip"
  }, /*#__PURE__*/React.createElement(__ds_scope.Tag, null, "morning pages \xB7 streak 9"), /*#__PURE__*/React.createElement(__ds_scope.Tag, null, "watering \xB7 streak 4"), /*#__PURE__*/React.createElement(__ds_scope.Tag, null, "evening walk \xB7 streak 2"))), /*#__PURE__*/React.createElement("div", {
    className: "stitch"
  }), note ? /*#__PURE__*/React.createElement("div", {
    className: "capture-note",
    role: "status"
  }, /*#__PURE__*/React.createElement("img", {
    className: "capture-note-icon",
    src: assetBase + '/icons/quill.png',
    alt: ""
  }), /*#__PURE__*/React.createElement("span", null, "Logged to ", /*#__PURE__*/React.createElement("b", null, note.kind), note.numeric ? /*#__PURE__*/React.createElement("span", null, " \u2014 ", /*#__PURE__*/React.createElement("b", null, note.value, note.unit)) : null, ". It joins the life log.")) : null, entries.length ? /*#__PURE__*/React.createElement("section", {
    className: "k-section",
    style: {
      paddingTop: 8
    }
  }, /*#__PURE__*/React.createElement("h2", {
    className: "k-heading k-heading-muted"
  }, "Just logged"), /*#__PURE__*/React.createElement("div", {
    className: "k-grid"
  }, entries.map(e => /*#__PURE__*/React.createElement("article", {
    className: "kcard task-fresh",
    key: e.id
  }, /*#__PURE__*/React.createElement("header", {
    className: "kcard-head"
  }, /*#__PURE__*/React.createElement("span", {
    className: "kcard-kind"
  }, "\u25AA ", e.kind), /*#__PURE__*/React.createElement("span", {
    className: "kcard-meta"
  }, "just now")), e.numeric ? /*#__PURE__*/React.createElement("p", {
    className: "life-stat"
  }, e.value, " ", /*#__PURE__*/React.createElement("span", {
    className: "life-unit"
  }, e.unit)) : /*#__PURE__*/React.createElement("p", {
    className: "kcard-body"
  }, e.text))))) : null, /*#__PURE__*/React.createElement("section", {
    className: "k-section"
  }, /*#__PURE__*/React.createElement("h2", {
    className: "k-heading"
  }, "Tracked"), /*#__PURE__*/React.createElement("p", {
    className: "k-sub"
  }, "What appears here is what you log \u2014 the kinds are yours to invent."), /*#__PURE__*/React.createElement("div", {
    className: "k-grid"
  }, /*#__PURE__*/React.createElement("article", {
    className: "kcard"
  }, /*#__PURE__*/React.createElement("header", {
    className: "kcard-head"
  }, /*#__PURE__*/React.createElement("span", {
    className: "kcard-kind"
  }, "\u25AA weight"), /*#__PURE__*/React.createElement("span", {
    className: "kcard-meta"
  }, "24 entries")), /*#__PURE__*/React.createElement("p", {
    className: "life-stat"
  }, "81.2 ", /*#__PURE__*/React.createElement("span", {
    className: "life-unit"
  }, "kg"), " ", /*#__PURE__*/React.createElement("span", {
    className: "life-lastat"
  }, "\xB7 this morning")), /*#__PURE__*/React.createElement("svg", {
    className: "spark",
    viewBox: "0 0 240 48",
    width: "240",
    height: "48",
    role: "img",
    "aria-label": "weight trend"
  }, /*#__PURE__*/React.createElement("polyline", {
    points: weight,
    fill: "none"
  }), /*#__PURE__*/React.createElement("circle", {
    cx: "46",
    cy: "22",
    r: "3"
  })), /*#__PURE__*/React.createElement("p", {
    className: "life-change"
  }, "\u22120.4 kg over two weeks")), /*#__PURE__*/React.createElement("article", {
    className: "kcard"
  }, /*#__PURE__*/React.createElement("header", {
    className: "kcard-head"
  }, /*#__PURE__*/React.createElement("span", {
    className: "kcard-kind"
  }, "\u25AA mood"), /*#__PURE__*/React.createElement("span", {
    className: "kcard-meta"
  }, "31 entries")), /*#__PURE__*/React.createElement("ul", {
    className: "life-lines"
  }, /*#__PURE__*/React.createElement("li", null, "steady \u2014 good work morning, slow afternoon"), /*#__PURE__*/React.createElement("li", null, "bright \u2014 long walk by the river"), /*#__PURE__*/React.createElement("li", null, "tired \u2014 late night reading"))), /*#__PURE__*/React.createElement("article", {
    className: "kcard"
  }, /*#__PURE__*/React.createElement("header", {
    className: "kcard-head"
  }, /*#__PURE__*/React.createElement("span", {
    className: "kcard-kind"
  }, "\u25AA practice"), /*#__PURE__*/React.createElement("span", {
    className: "kcard-meta"
  }, "12 entries")), /*#__PURE__*/React.createElement("p", {
    className: "life-stat"
  }, "35 ", /*#__PURE__*/React.createElement("span", {
    className: "life-unit"
  }, "min"), " ", /*#__PURE__*/React.createElement("span", {
    className: "life-lastat"
  }, "\xB7 yesterday")), /*#__PURE__*/React.createElement("svg", {
    className: "spark",
    viewBox: "0 0 240 48",
    width: "240",
    height: "48",
    role: "img",
    "aria-label": "practice trend"
  }, /*#__PURE__*/React.createElement("polyline", {
    points: "8,30 36,26 64,34 92,18 120,22 148,14 176,20 204,10 232,16",
    fill: "none"
  }), /*#__PURE__*/React.createElement("circle", {
    cx: "232",
    cy: "16",
    r: "3"
  }))))), /*#__PURE__*/React.createElement("div", {
    style: {
      height: 130
    }
  })), /*#__PURE__*/React.createElement(__ds_scope.ChatBar, {
    placeholder: "Log a measure. (\u201Cweight 81.2kg\u201D \xB7 \u201Cmood bright, long walk\u201D)",
    sendLabel: "Log",
    onSend: log
  }));
}
Object.assign(__ds_scope, { LifeScreen });
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/balaur/LifeScreen.jsx", error: String((e && e.message) || e) }); }

// ui_kits/balaur/MemoryScreen.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/* Light parse of a typed "keep this" line into a proposed memory card.
   Mirrors the chat memory_propose tool, surfaced on this page. */
function propose(text) {
  const lower = text.toLowerCase();
  let kind = 'fact';
  if (/\bprefer|\blike|\balways|\bnever|\brather\b/.test(lower)) kind = 'preference';else if (/\bbirthday|\bsister|\bbrother|\bmother|\bfather|\bfriend|\bwife|\bhusband|\bdaughter|\bson\b/.test(lower)) kind = 'person';else if (/\bproject|\bgarden|\bbuild|\bwriting|\bbook\b/.test(lower)) kind = 'project';
  const clean = text.trim().replace(/\s+/g, ' ');
  const title = clean.length > 52 ? clean.slice(0, 49).trimEnd() + '…' : clean;
  return {
    kind,
    title: title.charAt(0).toUpperCase() + title.slice(1),
    body: clean,
    importance: 2
  };
}

/** /memory (and /skills) — proposed → active → archived lifecycle.
    Balaur proposes, the owner decides. Owner can also offer something to
    keep through the shared composer; it lands as a proposed card. */
function MemoryScreen({
  kind = 'memory',
  assetBase = '../../assets'
}) {
  const nextId = React.useRef(100);
  const [cards, setCards] = React.useState([{
    id: 1,
    status: 'proposed',
    kind: 'preference',
    title: 'Prefers Markdown exports',
    body: 'Asked twice for notes as .md files.',
    whenToUse: 'when exporting or saving notes',
    importance: 3
  }, {
    id: 2,
    status: 'active',
    kind: 'person',
    title: 'Ana — sister',
    body: 'Birthday May 3rd. Lives in Cluj.',
    whenToUse: 'birthdays, family plans',
    importance: 4,
    usedCount: 6
  }, {
    id: 3,
    status: 'active',
    kind: 'project',
    title: 'Garden — tomatoes, peppers',
    body: 'Watering every 2 days at 18:00; seedlings planted late April.',
    importance: 3,
    usedCount: 11
  }, {
    id: 4,
    status: 'active',
    kind: 'fact',
    title: 'Box hostname is hearth.local',
    importance: 2,
    usedCount: 2
  }, {
    id: 5,
    status: 'archived',
    kind: 'fact',
    title: 'Old apartment Wi-Fi password'
  }]);
  const [cat, setCat] = React.useState('all');
  const [note, setNote] = React.useState(null);
  function move(id, status) {
    setCards(cs => cs.map(c => c.id === id ? Object.assign({}, c, {
      status
    }) : c));
  }
  function offer(text) {
    const m = propose(text);
    const id = nextId.current++;
    setCards(cs => [Object.assign({
      id,
      status: 'proposed',
      fresh: true
    }, m)].concat(cs));
    setNote(m);
    window.setTimeout(() => setCards(cs => cs.map(x => x.id === id ? Object.assign({}, x, {
      fresh: false
    }) : x)), 1600);
  }
  const proposed = cards.filter(c => c.status === 'proposed');
  const active = cards.filter(c => c.status === 'active' && (cat === 'all' || c.kind === cat));
  const archived = cards.filter(c => c.status === 'archived');
  return /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("main", null, /*#__PURE__*/React.createElement("h1", null, /*#__PURE__*/React.createElement("img", {
    className: "h-icon",
    src: assetBase + '/icons/' + (kind === 'skill' ? 'key' : 'tome') + '.png',
    alt: ""
  }), ' ', kind === 'skill' ? 'Skills' : 'Memory'), note ? /*#__PURE__*/React.createElement("div", {
    className: "capture-note",
    role: "status"
  }, /*#__PURE__*/React.createElement("img", {
    className: "capture-note-icon",
    src: assetBase + '/icons/quill.png',
    alt: ""
  }), /*#__PURE__*/React.createElement("span", null, "I shall weigh it. ", /*#__PURE__*/React.createElement("b", null, "\u201C", note.title, "\u201D"), " waits among ", /*#__PURE__*/React.createElement("b", null, "Awaiting your word"), " \u2014 keep it, and it is yours.")) : null, proposed.length ? /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("section", {
    className: "k-section"
  }, /*#__PURE__*/React.createElement("h2", {
    className: "k-heading k-heading-proposed"
  }, "Awaiting your word ", /*#__PURE__*/React.createElement("span", {
    className: "k-count"
  }, proposed.length)), /*#__PURE__*/React.createElement("p", {
    className: "k-sub"
  }, "Balaur proposed these. Nothing becomes memory without your approval."), /*#__PURE__*/React.createElement("div", {
    className: "k-grid"
  }, proposed.map(c => /*#__PURE__*/React.createElement("div", {
    key: c.id,
    className: c.fresh ? 'task-fresh' : undefined
  }, /*#__PURE__*/React.createElement(__ds_scope.KnowledgeCard, _extends({}, c, {
    onApprove: () => move(c.id, 'active'),
    onDismiss: () => setCards(cs => cs.filter(x => x.id !== c.id))
  })))))), /*#__PURE__*/React.createElement(__ds_scope.Stitch, null)) : null, /*#__PURE__*/React.createElement("section", {
    className: "k-section"
  }, /*#__PURE__*/React.createElement("h2", {
    className: "k-heading"
  }, "Active ", /*#__PURE__*/React.createElement("span", {
    className: "k-count"
  }, active.length)), /*#__PURE__*/React.createElement("div", {
    className: "k-controls"
  }, /*#__PURE__*/React.createElement("input", {
    className: "k-search",
    type: "search",
    placeholder: "Search memory\u2026",
    autoComplete: "off"
  }), /*#__PURE__*/React.createElement(__ds_scope.Tabs, {
    items: ['all', 'fact', 'preference', 'person', 'project'],
    active: cat,
    onSelect: setCat
  })), /*#__PURE__*/React.createElement("div", {
    className: "k-grid"
  }, active.map(c => /*#__PURE__*/React.createElement(__ds_scope.KnowledgeCard, _extends({
    key: c.id
  }, c, {
    onArchive: () => move(c.id, 'archived')
  }))))), archived.length ? /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement(__ds_scope.Stitch, null), /*#__PURE__*/React.createElement("section", {
    className: "k-section"
  }, /*#__PURE__*/React.createElement("h2", {
    className: "k-heading k-heading-muted"
  }, "Archived ", /*#__PURE__*/React.createElement("span", {
    className: "k-count"
  }, archived.length)), /*#__PURE__*/React.createElement("div", {
    className: "k-grid k-grid-muted"
  }, archived.map(c => /*#__PURE__*/React.createElement(__ds_scope.KnowledgeCard, _extends({
    key: c.id
  }, c, {
    onRestore: () => move(c.id, 'active')
  })))))) : null, /*#__PURE__*/React.createElement("div", {
    style: {
      height: 130
    }
  })), /*#__PURE__*/React.createElement(__ds_scope.ChatBar, {
    placeholder: "Tell me what to keep. (\u201CAna prefers tea over coffee\u201D)",
    sendLabel: "Offer",
    onSend: offer
  }));
}
Object.assign(__ds_scope, { MemoryScreen });
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/balaur/MemoryScreen.jsx", error: String((e && e.message) || e) }); }

// ui_kits/balaur/ProfileScreen.jsx
try { (() => {
const SOULS = [['soul-01', 'Him'], ['soul-02', 'Her'], ['soul-03', 'Elder'], ['soul-04', 'Youth'], ['soul-05', 'Maker'], ['soul-06', 'Cyclops'], ['soul-07', 'Gnome'], ['soul-08', 'Ogre'], ['soul-09', 'Strigoi'], ['soul-10', 'Zmeu'], ['soul-11', 'Iele'], ['soul-12', 'Muma'], ['soul-13', 'Căpcăun'], ['soul-14', 'Solomonar'], ['soul-15', 'Vâlvă'], ['soul-16', 'Pricolici']];
const HEADS = [['balaur-01', 'Wise'], ['balaur-02', 'Ancient'], ['balaur-03', 'Guardian'], ['balaur-04', 'Scholar'], ['balaur-05', 'Wild'], ['balaur-06', 'Storm'], ['balaur-07', 'Night'], ['balaur-08', 'Young'], ['balaur-09', 'Ember'], ['balaur-10', 'Frost'], ['balaur-11', 'Healer'], ['balaur-12', 'Trickster'], ['balaur-13', 'Dreamer'], ['balaur-14', 'Forest'], ['balaur-15', 'Dawn'], ['balaur-16', 'Sage']];

/** /profile — identity, soul avatar picker (16), companion head picker (16). */
function ProfileScreen({
  assetBase,
  soul,
  onSoul,
  head,
  onHead,
  ownerName,
  onName
}) {
  const [draft, setDraft] = React.useState(ownerName);
  const [saved, setSaved] = React.useState(false);
  function save(e) {
    e.preventDefault();
    onName(draft.trim() || 'You');
    setSaved(true);
  }
  return /*#__PURE__*/React.createElement("main", {
    className: "profile-page"
  }, /*#__PURE__*/React.createElement("h1", null, "Your Profile"), /*#__PURE__*/React.createElement("p", {
    className: "profile-lede"
  }, "How Balaur sees you \u2014 and how you see Balaur."), /*#__PURE__*/React.createElement("div", {
    className: "stitch"
  }), /*#__PURE__*/React.createElement("div", {
    style: {
      height: 20
    }
  }), /*#__PURE__*/React.createElement("article", {
    className: "profile-card"
  }, /*#__PURE__*/React.createElement("h2", {
    className: "profile-card-title"
  }, "Identity"), /*#__PURE__*/React.createElement("p", {
    className: "profile-hint"
  }, "The name Balaur uses in the chat label when you speak."), /*#__PURE__*/React.createElement("form", {
    className: "profile-name-form",
    onSubmit: save
  }, /*#__PURE__*/React.createElement("label", {
    htmlFor: "display_name"
  }, "Your name"), /*#__PURE__*/React.createElement("div", {
    className: "profile-name-row"
  }, /*#__PURE__*/React.createElement("input", {
    id: "display_name",
    name: "display_name",
    type: "text",
    value: draft,
    autoComplete: "off",
    maxLength: 60,
    placeholder: "How should Balaur call you?",
    onChange: e => {
      setDraft(e.target.value);
      setSaved(false);
    }
  }), /*#__PURE__*/React.createElement(__ds_scope.Button, {
    onClick: save
  }, "Save")), saved ? /*#__PURE__*/React.createElement("p", {
    className: "profile-saved"
  }, "\u25C8 Saved.") : null)), /*#__PURE__*/React.createElement(AvatarPickerCard, {
    title: "Your avatar",
    hint: "Who appears in chat as you. 16 portraits from the Basm world.",
    options: SOULS,
    value: soul,
    onPick: onSoul,
    assetBase: assetBase
  }), /*#__PURE__*/React.createElement(AvatarPickerCard, {
    title: "Companion head",
    hint: "Which Balaur personality you meet in chat. 16 heads, one companion.",
    options: HEADS,
    value: head,
    onPick: onHead,
    assetBase: assetBase
  }));
}
function AvatarPickerCard({
  title,
  hint,
  options,
  value,
  onPick,
  assetBase
}) {
  return /*#__PURE__*/React.createElement("article", {
    className: "profile-card"
  }, /*#__PURE__*/React.createElement("h2", {
    className: "profile-card-title"
  }, title), /*#__PURE__*/React.createElement("p", {
    className: "profile-hint"
  }, hint), /*#__PURE__*/React.createElement("div", {
    className: "avatar-choice-list profile-avatar-grid",
    style: {
      gridTemplateColumns: 'repeat(8, 1fr)'
    }
  }, options.map(([key, label]) => /*#__PURE__*/React.createElement("button", {
    key: key,
    type: "button",
    className: 'avatar-choice profile-avatar-btn' + (key === value ? ' avatar-choice-active' : ''),
    "aria-current": key === value ? 'true' : undefined,
    onClick: () => onPick(key)
  }, /*#__PURE__*/React.createElement("img", {
    className: "px",
    src: assetBase + '/avatars/' + key + '.png',
    alt: "",
    decoding: "async"
  }), /*#__PURE__*/React.createElement("span", null, label)))));
}
Object.assign(__ds_scope, { ProfileScreen });
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/balaur/ProfileScreen.jsx", error: String((e && e.message) || e) }); }

// ui_kits/balaur/TasksScreen.jsx
try { (() => {
/* Light in-world parse of a typed task line into bucket + due/recur hints.
   The same capture the chat's task_add tool performs, surfaced on this page. */
function capture(text) {
  const lower = text.toLowerCase();
  let recurLine,
    dueLine,
    bucket = 'Someday';
  const every = lower.match(/every[^,.;]*/);
  if (every) {
    recurLine = every[0].trim();
    bucket = 'Upcoming';
  }
  if (/\btoday\b|\btonight\b|\bthis evening\b/.test(lower)) {
    dueLine = 'due today';
    bucket = 'Today';
  } else if (/\btomorrow\b/.test(lower)) {
    dueLine = 'due tomorrow';
    bucket = 'Upcoming';
  }
  const time = lower.match(/\b([01]?\d|2[0-3]):[0-5]\d\b/);
  if (time) {
    dueLine = (dueLine ? dueLine + ' ' : 'due ') + time[0];
  }
  const title = text.trim().replace(/\s+/g, ' ');
  return {
    title: title.charAt(0).toUpperCase() + title.slice(1),
    dueLine,
    recurLine,
    bucket
  };
}

/** /tasks — the operational view of what was captured from chat. Adding a
    task uses the very same wood-ledge composer as the conversation: type a
    line, Balaur keeps it on the book. List (buckets), calendar, timeline. */
function TasksScreen({
  assetBase = '../../assets'
}) {
  const [view, setView] = React.useState('list');
  const [kept, setKept] = React.useState(null);
  const nextId = React.useRef(100);
  const [tasks, setTasks] = React.useState([{
    id: 1,
    title: 'Renew passport',
    dueLine: 'was due Tuesday',
    overdue: true,
    status: 'open',
    bucket: 'Overdue'
  }, {
    id: 2,
    title: 'Water the tomatoes',
    dueLine: 'due today 18:00',
    recurLine: 'every 2 days',
    status: 'open',
    bucket: 'Today'
  }, {
    id: 3,
    title: 'Call the notary about the deed',
    dueLine: 'due today',
    status: 'open',
    bucket: 'Today',
    notes: 'Ask about the missing signature page.'
  }, {
    id: 4,
    title: 'Morning pages',
    dueLine: 'due tomorrow 07:00',
    recurLine: 'every mon–fri 07:00',
    status: 'open',
    bucket: 'Upcoming'
  }, {
    id: 5,
    title: 'Learn the Solomonar chant',
    status: 'open',
    bucket: 'Someday'
  }, {
    id: 6,
    title: 'Back up pb_data to the vault disk',
    status: 'done',
    bucket: 'Done recently'
  }]);
  function transition(id, to) {
    setTasks(ts => ts.map(t => t.id === id ? Object.assign({}, t, {
      status: to,
      bucket: to === 'open' ? t.bucket : 'Done recently'
    }) : t));
  }
  function captureTask(text) {
    const t = capture(text);
    const id = nextId.current++;
    setTasks(ts => [Object.assign({
      id,
      status: 'open',
      fresh: true
    }, t)].concat(ts));
    setKept(t);
    window.setTimeout(() => setTasks(ts => ts.map(x => x.id === id ? Object.assign({}, x, {
      fresh: false
    }) : x)), 1600);
  }
  const buckets = ['Overdue', 'Today', 'Upcoming', 'Someday'];
  return /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("main", null, /*#__PURE__*/React.createElement("h1", null, /*#__PURE__*/React.createElement("img", {
    className: "h-icon",
    src: assetBase + '/icons/scroll.png',
    alt: ""
  }), " Tasks"), kept ? /*#__PURE__*/React.createElement("div", {
    className: "capture-note",
    role: "status"
  }, /*#__PURE__*/React.createElement("img", {
    className: "capture-note-icon",
    src: assetBase + '/icons/quill.png',
    alt: ""
  }), /*#__PURE__*/React.createElement("span", null, "Kept. ", /*#__PURE__*/React.createElement("b", null, "\u201C", kept.title, "\u201D"), " now waits on the book", kept.recurLine ? ' — ' + kept.recurLine : kept.dueLine ? ' — ' + kept.dueLine : ', without a date', ".")) : null, /*#__PURE__*/React.createElement("nav", {
    className: "k-tabs t-views"
  }, ['list', 'calendar', 'timeline'].map(v => /*#__PURE__*/React.createElement("a", {
    key: v,
    className: 'k-tab' + (v === view ? ' k-tab-active' : ''),
    href: "#",
    onClick: e => {
      e.preventDefault();
      setView(v);
    }
  }, v))), view === 'list' ? /*#__PURE__*/React.createElement("div", null, buckets.map(b => {
    const items = tasks.filter(t => t.bucket === b && t.status === 'open');
    if (!items.length) return null;
    return /*#__PURE__*/React.createElement("section", {
      className: "k-section",
      key: b
    }, /*#__PURE__*/React.createElement("h2", {
      className: 'k-heading' + (b === 'Overdue' ? ' k-heading-proposed' : '')
    }, b, " ", /*#__PURE__*/React.createElement("span", {
      className: "k-count"
    }, items.length)), b === 'Someday' ? /*#__PURE__*/React.createElement("p", {
      className: "k-sub"
    }, "No date yet \u2014 they wait without nagging.") : null, /*#__PURE__*/React.createElement("div", {
      className: "k-grid"
    }, items.map(t => /*#__PURE__*/React.createElement("div", {
      key: t.id,
      className: t.fresh ? 'task-fresh' : undefined
    }, /*#__PURE__*/React.createElement(__ds_scope.TaskCard, {
      title: t.title,
      dueLine: t.dueLine,
      overdue: t.overdue,
      recurLine: t.recurLine,
      notes: t.notes,
      status: t.status,
      onDone: () => transition(t.id, 'done'),
      onSnooze: () => transition(t.id, 'open'),
      onDrop: () => transition(t.id, 'dropped')
    })))));
  }), /*#__PURE__*/React.createElement(__ds_scope.Stitch, null), /*#__PURE__*/React.createElement("section", {
    className: "k-section"
  }, /*#__PURE__*/React.createElement("h2", {
    className: "k-heading k-heading-muted"
  }, "Done recently"), /*#__PURE__*/React.createElement("div", {
    className: "k-grid k-grid-muted"
  }, tasks.filter(t => t.status !== 'open').map(t => /*#__PURE__*/React.createElement(__ds_scope.TaskCard, {
    key: t.id,
    title: t.title,
    status: t.status === 'dropped' ? 'dropped' : 'done'
  }))))) : null, view === 'calendar' ? /*#__PURE__*/React.createElement(TasksCalendar, null) : null, view === 'timeline' ? /*#__PURE__*/React.createElement(TasksTimeline, null) : null, /*#__PURE__*/React.createElement("div", {
    style: {
      height: 130
    }
  })), /*#__PURE__*/React.createElement(__ds_scope.ChatBar, {
    placeholder: "Name a task; I shall keep it. (\u201Cwater the tomatoes every 2 days, evenings\u201D)",
    sendLabel: "Keep",
    onSend: captureTask
  }));
}
function TasksCalendar() {
  const weeks = [[1, 2, 3, 4, 5, 6, 7], [8, 9, 10, 11, 12, 13, 14], [15, 16, 17, 18, 19, 20, 21], [22, 23, 24, 25, 26, 27, 28], [29, 30, 1, 2, 3, 4, 5]];
  const items = {
    12: ['⟳ 18:00 water the tomatoes'],
    15: ['10:00 notary'],
    16: ['⟳ 07:00 morning pages']
  };
  return /*#__PURE__*/React.createElement("section", {
    className: "k-section"
  }, /*#__PURE__*/React.createElement("div", {
    className: "cal-nav"
  }, /*#__PURE__*/React.createElement("a", {
    className: "btn btn-ghost btn-sm",
    href: "#",
    onClick: e => e.preventDefault()
  }, "\u25C2 prev"), /*#__PURE__*/React.createElement("h2", {
    className: "cal-label"
  }, "June 2026"), /*#__PURE__*/React.createElement("a", {
    className: "btn btn-ghost btn-sm",
    href: "#",
    onClick: e => e.preventDefault()
  }, "next \u25B8")), /*#__PURE__*/React.createElement("table", {
    className: "cal-table"
  }, /*#__PURE__*/React.createElement("thead", null, /*#__PURE__*/React.createElement("tr", null, ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'].map(d => /*#__PURE__*/React.createElement("th", {
    key: d,
    scope: "col"
  }, d)))), /*#__PURE__*/React.createElement("tbody", null, weeks.map((w, wi) => /*#__PURE__*/React.createElement("tr", {
    key: wi
  }, w.map((d, di) => {
    const out = wi === 4 && di > 1;
    const today = d === 12 && wi === 1;
    return /*#__PURE__*/React.createElement("td", {
      key: di,
      className: 'cal-cell' + (out ? ' cal-out' : '') + (today ? ' cal-today' : '')
    }, /*#__PURE__*/React.createElement("span", {
      className: "cal-daynum"
    }, d), (!out && items[d] ? items[d] : []).map((it, ii) => /*#__PURE__*/React.createElement("span", {
      className: "cal-item",
      key: ii
    }, it)));
  }))))), /*#__PURE__*/React.createElement("p", {
    className: "k-sub"
  }, "Recurring tasks are projected from their rules \u2014 the calendar shows the plan, the list holds the actions."));
}
function TasksTimeline() {
  const days = [{
    label: 'Today · Fri Jun 12',
    today: true,
    items: [{
      time: '18:00',
      title: 'water the tomatoes',
      recur: true
    }]
  }, {
    label: 'Sat Jun 13',
    items: []
  }, {
    label: 'Sun Jun 14',
    items: [{
      time: '18:00',
      title: 'water the tomatoes',
      recur: true
    }]
  }, {
    label: 'Mon Jun 15',
    items: [{
      time: '07:00',
      title: 'morning pages',
      recur: true
    }, {
      time: '10:00',
      title: 'call the notary about the deed'
    }]
  }];
  return /*#__PURE__*/React.createElement("section", {
    className: "k-section tl"
  }, days.map(d => /*#__PURE__*/React.createElement("div", {
    key: d.label,
    className: 'tl-day' + (d.today ? ' tl-today' : '')
  }, /*#__PURE__*/React.createElement("h3", {
    className: "tl-label"
  }, d.label), d.items.length ? /*#__PURE__*/React.createElement("ul", {
    className: "tl-items"
  }, d.items.map((it, i) => /*#__PURE__*/React.createElement("li", {
    className: "tl-item",
    key: i
  }, /*#__PURE__*/React.createElement("span", {
    className: "tl-time"
  }, it.time), " ", it.title, it.recur ? /*#__PURE__*/React.createElement("span", null, " ", /*#__PURE__*/React.createElement("span", {
    className: "tag"
  }, "\u27F3")) : null))) : /*#__PURE__*/React.createElement("p", {
    className: "tl-quiet"
  }, "quiet"))), /*#__PURE__*/React.createElement("p", {
    className: "k-sub"
  }, "The next two weeks, rules unrolled \u2014 the forward mirror of the telescope above the chat."));
}
Object.assign(__ds_scope, { TasksScreen });
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/balaur/TasksScreen.jsx", error: String((e && e.message) || e) }); }

// ui_kits/balaur/BalaurApp.jsx
try { (() => {
/** Interactive recreation of the Balaur web app (alexradunet/balaur).
    Pages: Chat (home), Tasks, Life, Memory, Profile. Skills and Heads are
    not recreated in this kit — they follow the Memory page pattern. */
function BalaurApp({
  assetBase = '../../assets'
}) {
  const [page, setPage] = React.useState('home');
  const [light, setLight] = React.useState(false);
  const [soul, setSoul] = React.useState('soul-01');
  const [head, setHead] = React.useState('balaur-01');
  const [ownerName, setOwnerName] = React.useState('Alex');
  React.useEffect(() => {
    document.documentElement.classList.toggle('light', light);
    document.documentElement.classList.toggle('dark', !light);
  }, [light]);
  const soulSrc = assetBase + '/avatars/' + soul + '.png';
  const headSrc = assetBase + '/avatars/' + head + '.png';
  let screen = null;
  if (page === 'home') {
    screen = /*#__PURE__*/React.createElement(__ds_scope.ChatScreen, {
      soulSrc: soulSrc,
      headSrc: headSrc,
      ownerName: ownerName,
      assetBase: assetBase
    });
  } else if (page === 'Tasks') {
    screen = /*#__PURE__*/React.createElement(__ds_scope.TasksScreen, {
      assetBase: assetBase
    });
  } else if (page === 'Memory' || page === 'Skills') {
    screen = /*#__PURE__*/React.createElement(__ds_scope.MemoryScreen, {
      kind: page === 'Skills' ? 'skill' : 'memory',
      assetBase: assetBase
    });
  } else if (page === 'Life') {
    screen = /*#__PURE__*/React.createElement(__ds_scope.LifeScreen, {
      assetBase: assetBase
    });
  } else if (page === 'Profile') {
    screen = /*#__PURE__*/React.createElement(__ds_scope.ProfileScreen, {
      assetBase: assetBase,
      soul: soul,
      onSoul: setSoul,
      head: head,
      onHead: setHead,
      ownerName: ownerName,
      onName: setOwnerName
    });
  } else {
    screen = /*#__PURE__*/React.createElement("main", null, /*#__PURE__*/React.createElement("p", {
      className: "k-empty",
      style: {
        padding: '40px 0'
      }
    }, page, " is not recreated in this kit \u2014 it follows the Memory page pattern in the product."));
  }
  return /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement(__ds_scope.Topbar, {
    crestSrc: assetBase + '/crest.png',
    links: ['Tasks', 'Life', 'Memory', 'Skills', 'Profile', 'Heads'],
    active: page,
    onNavigate: setPage,
    onToggleTheme: () => setLight(!light),
    themeGlyph: light ? '◑' : '☼'
  }), screen);
}
Object.assign(__ds_scope, { BalaurApp });
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/balaur/BalaurApp.jsx", error: String((e && e.message) || e) }); }

__ds_ns.Avatar = __ds_scope.Avatar;

__ds_ns.ChatBar = __ds_scope.ChatBar;

__ds_ns.ChatMessage = __ds_scope.ChatMessage;

__ds_ns.DialogueChoices = __ds_scope.DialogueChoices;

__ds_ns.MessageDraft = __ds_scope.MessageDraft;

__ds_ns.ToolRow = __ds_scope.ToolRow;

__ds_ns.Button = __ds_scope.Button;

__ds_ns.Card = __ds_scope.Card;

__ds_ns.FolkBand = __ds_scope.FolkBand;

__ds_ns.Pips = __ds_scope.Pips;

__ds_ns.Stitch = __ds_scope.Stitch;

__ds_ns.Tag = __ds_scope.Tag;

__ds_ns.KnowledgeCard = __ds_scope.KnowledgeCard;

__ds_ns.TaskCard = __ds_scope.TaskCard;

__ds_ns.Tabs = __ds_scope.Tabs;

__ds_ns.Topbar = __ds_scope.Topbar;

__ds_ns.BalaurApp = __ds_scope.BalaurApp;

__ds_ns.ChatScreen = __ds_scope.ChatScreen;

__ds_ns.LifeScreen = __ds_scope.LifeScreen;

__ds_ns.MemoryScreen = __ds_scope.MemoryScreen;

__ds_ns.ProfileScreen = __ds_scope.ProfileScreen;

__ds_ns.TasksScreen = __ds_scope.TasksScreen;

})();
