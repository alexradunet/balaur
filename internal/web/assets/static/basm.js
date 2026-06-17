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
    btn.setAttribute('aria-pressed', isLight ? 'true' : 'false');
  });
}

document.addEventListener('DOMContentLoaded', basmUpdateThemeButtons);

// ── Theme palette (hearthwood / forest / dungeon) ──────────────────
// Orthogonal to light/dark mode (basmToggleTheme). The <head> no-flash
// script applies the saved palette before paint. The palette is chosen in
// Settings → Appearance and the storybook sidebar footer; both call
// basmSetPalette directly (the topbar cycler was retired).

// Set a specific palette directly.
window.basmSetPalette = function (name) {
  var order = ['hearthwood', 'forest', 'dungeon'];
  if (order.indexOf(name) < 0) name = 'hearthwood';
  var d = document.documentElement;
  d.classList.remove('theme-hearthwood', 'theme-forest', 'theme-dungeon');
  d.classList.add('theme-' + name);
  localStorage.setItem('basm-palette', name);
  basmUpdatePaletteButtons();
};

function basmUpdatePaletteButtons() {
  var order = ['hearthwood', 'forest', 'dungeon'];
  var d = document.documentElement;
  var cur = order.find(function (t) { return d.classList.contains('theme-' + t); }) || 'hearthwood';
  // Storybook sidebar footer buttons. The Settings → Appearance buttons mark
  // their active state in pure CSS off the <html> palette class, so they need
  // no sync here.
  document.querySelectorAll('.sb-theme-btn').forEach(function (btn) {
    btn.classList.toggle('is-active', btn.getAttribute('data-theme') === cur);
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

// ── Cap active artifacts (plan 094) ────────────────────────────────
// Keep at most ACTIVE_ARTIFACT_CAP artifacts expanded; older ones collapse to
// their static "shown earlier" chip. Runs on load and on every #chat mutation
// (covers sidebar injects, agent card_show, and clusters across all fragments).
var ACTIVE_ARTIFACT_CAP = 3;
function balaurCapArtifacts() {
  var chat = document.getElementById('chat');
  if (!chat) return;
  var arts = chat.querySelectorAll('.artifact');
  var cutoff = arts.length - ACTIVE_ARTIFACT_CAP;
  arts.forEach(function (el, i) {
    el.classList.toggle('artifact--collapsed', i < cutoff);
  });
}

// Datastar appends/morphs #chat directly (no htmx swap events), so watch the
// chat node and keep the latest message in view.
document.addEventListener('DOMContentLoaded', () => {
  const chat = document.getElementById('chat');
  if (!chat) return;
  balaurScrollToLatest();
  balaurCapArtifacts();
  new MutationObserver(() => { balaurCapArtifacts(); balaurScrollToLatest(); })
    .observe(chat, { childList: true, subtree: true });
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

// ── Product topbar off-canvas nav (accessible) ────────────────────
// Separate from basmToggleNav (storybook). Closed drawer is inert (untabbable);
// open moves focus in, traps Tab, closes on Escape/backdrop click, restores focus.
// Two drawer mechanisms now coexist intentionally — see Maintenance notes in plan 078.
(function () {
  function drawer()   { return document.getElementById('topnav-drawer'); }
  function backdrop() { return document.querySelector('.topnav-backdrop'); }
  function burger()   { return document.querySelector('.topnav-burger'); }
  function focusables(d) { return d.querySelectorAll('a[href], button:not([disabled])'); }
  var lastFocus = null;

  // Closed drawer must not participate in tab order.
  function setClosedInert() { var d = drawer(); if (d && !d.classList.contains('is-open')) d.inert = true; }
  document.addEventListener('DOMContentLoaded', setClosedInert);

  window.basmToggleTopnav = function () {
    var d = drawer(); if (!d) return;
    var open = !d.classList.contains('is-open');
    d.classList.toggle('is-open', open);
    var bk = backdrop(); if (bk) bk.classList.toggle('is-open', open);
    d.inert = !open;
    d.setAttribute('aria-hidden', open ? 'false' : 'true');
    var bg = burger(); if (bg) bg.setAttribute('aria-expanded', open ? 'true' : 'false');
    if (open) {
      lastFocus = document.activeElement;
      var f = focusables(d); if (f.length) f[0].focus();
    } else if (lastFocus && lastFocus.focus) {
      lastFocus.focus();
    }
  };

  // Tab trap inside the open drawer.
  document.addEventListener('keydown', function (e) {
    var d = drawer(); if (!d || !d.classList.contains('is-open')) return;
    if (e.key === 'Escape') { e.preventDefault(); window.basmToggleTopnav(); return; }
    if (e.key !== 'Tab') return;
    var f = focusables(d); if (!f.length) return;
    var first = f[0], last = f[f.length - 1];
    if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
    else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
  });

  // Navigating via a drawer link dismisses the drawer.
  document.addEventListener('click', function (e) {
    var d = drawer();
    if (d && d.classList.contains('is-open') && e.target.closest('#topnav-drawer a')) {
      window.basmToggleTopnav();
    }
  });
})();

// ── Sidebar scroll persistence ─────────────────────────────────────
// Each storybook nav is a full page load, which would reset the sidebar rail to
// the top. Persist its scroll position per tab so clicking an item near the
// bottom of the list keeps the rail where it was.
(function () {
  var KEY = 'basm-side-scroll';
  function rail() { return document.querySelector('.sb-side'); }
  function save() { var r = rail(); if (r) { try { sessionStorage.setItem(KEY, String(r.scrollTop)); } catch (e) {} } }
  window.addEventListener('pagehide', save);
  document.addEventListener('DOMContentLoaded', function () {
    var r = rail(); if (!r) return;
    try { var v = sessionStorage.getItem(KEY); if (v) r.scrollTop = parseInt(v, 10) || 0; } catch (e) {}
    // belt-and-suspenders: also save synchronously on a nav-item click.
    r.addEventListener('click', function (e) { if (e.target.closest && e.target.closest('.sb-nav-item')) save(); });
  });
})();
