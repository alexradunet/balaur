/* basm.js — Balaur UI utilities. Loaded on every page (deferred).
   Keeps home.html's inline script focused on chat-specific behaviour.
   No framework, no build step — just the platform. */

// ── Light/dark theme toggle ────────────────────────────────────────
// The <head> inline script in page_head reads localStorage and applies
// the class before render to prevent FOUC. This file handles the toggle
// interaction and keeps the button glyph in sync.

window.basmToggleTheme = function () {
  const html    = document.documentElement;
  const isLight = html.classList.contains('light');
  html.classList.remove('light', 'dark');
  if (isLight) {
    html.classList.add('dark');
    localStorage.setItem('basm-theme', 'dark');
  } else {
    html.classList.add('light');
    localStorage.setItem('basm-theme', 'light');
  }
  basmUpdateThemeButtons();
};

function basmUpdateThemeButtons() {
  const isLight = document.documentElement.classList.contains('light');
  document.querySelectorAll('.theme-toggle').forEach(btn => {
    btn.textContent = isLight ? '◑' : '☼';
    btn.title       = isLight ? 'Switch to dark mode' : 'Switch to light mode';
  });
}

document.addEventListener('DOMContentLoaded', basmUpdateThemeButtons);

// ── Theme palette cycle (hearthwood → forest → dungeon) ────────────
// Orthogonal to light/dark mode (basmToggleTheme). The <head> no-flash
// script applies the saved palette before paint; this handles the cycle.
window.basmCycleTheme = function () {
  var order = ['hearthwood', 'forest', 'dungeon'];
  var d = document.documentElement;
  var cur = order.find(function (t) { return d.classList.contains('theme-' + t); }) || 'hearthwood';
  var next = order[(order.indexOf(cur) + 1) % order.length];
  d.classList.remove('theme-hearthwood', 'theme-forest', 'theme-dungeon');
  d.classList.add('theme-' + next);
  localStorage.setItem('basm-palette', next);
  basmUpdatePaletteButtons();
};

function basmUpdatePaletteButtons() {
  var order = ['hearthwood', 'forest', 'dungeon'];
  var labels = { hearthwood: 'Hearth', forest: 'Forest', dungeon: 'Dungeon' };
  var d = document.documentElement;
  var cur = order.find(function (t) { return d.classList.contains('theme-' + t); }) || 'hearthwood';
  document.querySelectorAll('.theme-cycle').forEach(function (btn) {
    btn.textContent = labels[cur];
    btn.title = 'Cycle theme (now ' + labels[cur] + ')';
  });
}
document.addEventListener('DOMContentLoaded', basmUpdatePaletteButtons);

// ── Chatbar height → CSS custom property ──────────────────────────
// The chat section uses padding-bottom: var(--chatbar-space) so the
// last message is never hidden behind the fixed bar.

function basmSyncChatbarSpace() {
  const bar = document.getElementById('chatbar');
  if (!bar) return;
  document.documentElement.style.setProperty(
    '--chatbar-space',
    `${bar.offsetHeight + 24}px`
  );
}

// Datastar replaces #chatbar (and #chat-draft) wholesale when the model state
// changes, so re-sync the spacing and re-observe the new node when it appears.
// Tracking the observed node avoids stacking ResizeObservers on every patch.
let basmChatbarObserved = null;
function basmWatchChatbar() {
  const bar = document.getElementById('chatbar');
  if (!bar) return;
  basmSyncChatbarSpace();
  if (bar === basmChatbarObserved || !window.ResizeObserver) return;
  basmChatbarObserved = bar;
  new ResizeObserver(basmSyncChatbarSpace).observe(bar);
}

window.addEventListener('resize', basmSyncChatbarSpace);
document.addEventListener('DOMContentLoaded', () => {
  basmWatchChatbar();
  // The dock is never swapped, but its #chatbar/#chat-draft are patched.
  const dock = document.getElementById('dock') || document.body;
  new MutationObserver(basmWatchChatbar).observe(dock, { childList: true, subtree: true });
});

// The model modal (<dialog>) opens itself: the SSE handler that patches its
// content runs showModal() via ExecuteScript. The knowledge tab active-state
// is driven by Datastar data-class. No htmx lifecycle hooks remain.

// ── Dialogue choices keyboard shortcut (1–9) ───────────────────────
// When no input is focused and no modifier is held, pressing a digit
// clicks the matching choice in the last visible .choices panel.
document.addEventListener('keydown', (e) => {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') return;
  if (e.metaKey || e.ctrlKey || e.altKey) return;
  const n = parseInt(e.key, 10);
  if (!n || n < 1 || n > 9) return;
  const chat = document.getElementById('chat');
  if (!chat) return;
  const panels = chat.querySelectorAll('.choices');
  if (!panels.length) return;
  const last = panels[panels.length - 1];
  const buttons = last.querySelectorAll('.choice');
  const btn = buttons[n - 1];
  if (btn) btn.click();
});

// ── Chat helpers (master chat dock) ────────────────────────────────
// Moved here from home.html so the chat works wherever the dock is
// mounted (home main today, the board sidebar next). Datastar drives the
// stream; these handle scroll, enter-to-send, and the model modal.
window.balaurScrollToLatest = function () {
  const chat = document.getElementById('chat');
  // A docked #chat scrolls itself; the full-width home #chat scrolls the page.
  if (chat && chat.scrollHeight > chat.clientHeight + 4) {
    chat.scrollTop = chat.scrollHeight;
  } else {
    scrollTo(0, document.documentElement.scrollHeight || document.body.scrollHeight);
  }
};

window.balaurSubmitOnEnter = function (event) {
  if (event.key !== 'Enter' || event.shiftKey || event.altKey ||
      event.ctrlKey || event.metaKey || event.isComposing) return;
  event.preventDefault();
  event.currentTarget.form?.requestSubmit();
};

window.balaurCloseModal = function () {
  const d = document.getElementById('model-modal');
  if (d) { d.close(); d.innerHTML = ''; }
};

// Datastar appends/morphs #chat directly (no htmx swap events), so watch the
// chat node and keep the latest message in view.
document.addEventListener('DOMContentLoaded', () => {
  const chat = document.getElementById('chat');
  if (!chat) return;
  balaurScrollToLatest();
  new MutationObserver(balaurScrollToLatest).observe(chat, { childList: true, subtree: true });
});

// ── Dock: full-screen toggle + drag-to-resize the rail ─────────────
// State lives on <html> (applied early by the page_head inline script to avoid
// a flash) and persists in localStorage.
window.basmToggleDockFull = function () {
  const on = document.documentElement.classList.toggle('dock-full');
  localStorage.setItem('basm-dock-full', on ? '1' : '0');
  balaurScrollToLatest();
};

// Drag the left grip to resize the rail. Delegated from document so it survives
// #main swaps (the dock itself is never swapped). Disabled in full-screen mode.
document.addEventListener('pointerdown', (e) => {
  const grip = e.target.closest('.dock-grip');
  if (!grip || document.documentElement.classList.contains('dock-full')) return;
  e.preventDefault();
  grip.setPointerCapture(e.pointerId);
  grip.classList.add('dragging');
  const onMove = (ev) => {
    const w = Math.max(280, Math.min(720, window.innerWidth - ev.clientX));
    document.documentElement.style.setProperty('--sidebar-w', w + 'px');
  };
  const onUp = () => {
    grip.classList.remove('dragging');
    document.removeEventListener('pointermove', onMove);
    document.removeEventListener('pointerup', onUp);
    document.removeEventListener('pointercancel', onUp);
    const w = parseInt(document.documentElement.style.getPropertyValue('--sidebar-w'), 10);
    if (w) localStorage.setItem('basm-dock-w', String(w));
  };
  document.addEventListener('pointermove', onMove);
  document.addEventListener('pointerup', onUp);
  document.addEventListener('pointercancel', onUp);
});

// ── Storybook off-canvas nav drawer ────────────────────────────────
// The sidebar (.sb-side) is fixed off-screen ≤920px; the .sb-topbar burger
// and the .sb-backdrop both toggle it. Closes on backdrop click and on any
// nav-item click (so navigating dismisses the drawer).
window.basmToggleNav = function () {
  var open = document.documentElement.classList.toggle('sb-nav-open');
  document.querySelectorAll('.sb-side, .sb-backdrop').forEach(function (el) { el.classList.toggle('is-open', open); });
  document.querySelectorAll('.sb-burger').forEach(function (b) { b.setAttribute('aria-expanded', open ? 'true' : 'false'); });
};
document.addEventListener('click', function (e) {
  if (e.target.closest('.sb-side .sb-nav-item') && document.documentElement.classList.contains('sb-nav-open')) {
    window.basmToggleNav();
  }
});
