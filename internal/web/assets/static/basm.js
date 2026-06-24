/* basm.js — Balaur UI utilities. Loaded on every page (deferred).
   Keeps home.html's inline script focused on chat-specific behaviour.
   No framework, no build step — just the platform. */

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

// Visible (not filtered-out, menu-open) command rows, top-to-bottom.
// A row hidden by Datastar data-show has inline display:none → offsetParent null.
function balaurCmdVisibleItems(palette) {
  if (!palette) return [];
  return Array.prototype.filter.call(
    palette.querySelectorAll('.cmd-item'),
    function (el) { return el.offsetParent !== null; });
}

window.balaurSubmitOnEnter = function (event) {
  if (event.key !== 'Enter' || event.shiftKey || event.altKey ||
      event.ctrlKey || event.metaKey || event.isComposing) return;
  event.preventDefault();
  var ta = event.currentTarget;
  // Slash-command: act on the highlighted palette row (or the first visible
  // one) instead of sending the "/foo" line to chat.
  if (ta.value.trimStart().startsWith('/')) {
    var palette = ta.closest('.composer') &&
      ta.closest('.composer').querySelector('.cmd-palette');
    var items = balaurCmdVisibleItems(palette);
    var target = items.filter(function (el) {
      return el.classList.contains('is-active');
    })[0] || items[0];
    if (target) target.click();   // triggers the row's data-on:click @get
    return;                       // never post a "/foo" line to chat
  }
  ta.form && ta.form.requestSubmit();
};

// ── Composer /-command menu: ↑/↓ navigate the active row ───────────────
// The active row is DOM state (.cmd-item.is-active), not a Datastar signal:
// "the active row" = "the Nth currently-visible row", a value only the browser
// knows after data-show filtering. Enter (balaurSubmitOnEnter) reads it.
document.addEventListener('keydown', function (e) {
  if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
  var ta = e.target;
  if (!ta || ta.tagName !== 'TEXTAREA' || !ta.closest('.composer')) return;
  var palette = ta.closest('.composer').querySelector('.cmd-palette');
  if (!palette || palette.offsetParent === null) return; // menu not open
  var items = balaurCmdVisibleItems(palette);
  if (!items.length) return;
  e.preventDefault(); // own the arrow: don't move the textarea caret
  var cur = -1;
  for (var i = 0; i < items.length; i++) {
    if (items[i].classList.contains('is-active')) { cur = i; break; }
  }
  var next = e.key === 'ArrowDown'
    ? (cur + 1) % items.length
    : (cur - 1 + items.length) % items.length; // wraps at both ends
  items.forEach(function (el) { el.classList.remove('is-active'); });
  items[next].classList.add('is-active');
  items[next].scrollIntoView({ block: 'nearest' });
});

// As the owner types and the menu re-filters, default the highlight to the top
// match so Enter's target is always visible. Deferred one frame so Datastar's
// data-show has re-evaluated visibility before we read offsetParent.
document.addEventListener('input', function (e) {
  var ta = e.target;
  if (!ta || ta.tagName !== 'TEXTAREA' || !ta.closest('.composer')) return;
  var palette = ta.closest('.composer').querySelector('.cmd-palette');
  if (!palette) return;
  requestAnimationFrame(function () {
    var items = balaurCmdVisibleItems(palette);
    items.forEach(function (el, i) { el.classList.toggle('is-active', i === 0); });
  });
});

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
  new MutationObserver(() => { balaurScrollToLatest(); })
    .observe(chat, { childList: true, subtree: true });
});

// ── Right panel: auto-open the mobile drawer when an artifact is summoned ──
document.addEventListener('DOMContentLoaded', () => {
  var inner = document.getElementById('panel-inner');
  if (!inner) return;
  var isNarrow = function () { return window.matchMedia('(max-width: 720px)').matches; };
  new MutationObserver(function () {
    if (isNarrow() && !inner.querySelector('.panel-empty')) {
      document.documentElement.classList.add('panel-open');
    }
  }).observe(inner, { childList: true, subtree: true });
  // The scrim is a ::after pseudo-element (NOT a clickable DOM node) — so close
  // on any click that lands outside #panel and .sb-side while the drawer is open.
  document.addEventListener('click', function (e) {
    if (document.documentElement.classList.contains('panel-open') &&
        !e.target.closest('#panel') && !e.target.closest('.sb-side') && !e.target.closest('.navrail')) {
      document.documentElement.classList.remove('panel-open');
    }
  });
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') document.documentElement.classList.remove('panel-open');
  });
});

// ── Panel: collapse toggle + drag-to-resize (plan 103) ─────────────
// State lives on <html> and persists server-side (owner_settings), so the
// committed width and collapsed flag survive reload without a flash.
// Both the server render (chatshell.go) and this drag set --w-panel on
// document.documentElement (<html>) — they MUST target the same element so the
// CSS custom property cascade resolves through one owner; .app-shell inherits it.
window.basmTogglePanel = function () {
  var on = document.documentElement.classList.toggle('panel-collapsed');
  var t = document.querySelector('.navrail-toggle');
  if (t) t.setAttribute('aria-expanded', on ? 'false' : 'true');
  fetch('/ui/panel/collapse', {
    method: 'POST', headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: 'on=' + (on ? '1' : '0'),
  });
};

// Reveal the panel live when a nav-rail destination is summoned. The /ui/show
// door already persisted collapsed=0 server-side; this drops the class so the
// card shows on desktop without a reload. On ≤720px the #panel-inner observer
// adds panel-open for the overlay, so this is a desktop-reveal no-op there.
window.basmOpenPanel = function () {
  document.documentElement.classList.remove('panel-collapsed');
  var t = document.querySelector('.navrail-toggle');
  if (t) t.setAttribute('aria-expanded', 'true');
};

// Drag the panel divider to resize. Commits the width to the server on release.
document.addEventListener('pointerdown', function (e) {
  var grip = e.target.closest('.panel-resizer');
  if (!grip) return;
  e.preventDefault();
  grip.setPointerCapture(e.pointerId);
  grip.classList.add('dragging');
  var onMove = function (ev) {
    var w = Math.max(320, Math.min(1100, window.innerWidth - ev.clientX));
    document.documentElement.style.setProperty('--w-panel', w + 'px');
  };
  var onUp = function () {
    grip.classList.remove('dragging');
    document.removeEventListener('pointermove', onMove);
    document.removeEventListener('pointerup', onUp);
    document.removeEventListener('pointercancel', onUp);
    var w = parseInt(document.documentElement.style.getPropertyValue('--w-panel'), 10);
    if (w) fetch('/ui/panel/width', {
      method: 'POST', headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: 'px=' + w,
    });
  };
  document.addEventListener('pointermove', onMove);
  document.addEventListener('pointerup', onUp);
  document.addEventListener('pointercancel', onUp);
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

// Toast auto-dismiss (plan 174 S7): when the server appends a .toast into
// #toast-region, hold ~3s, then animate it out (.toast-out) and remove it.
// Reduced-motion-safe — the exit animation collapses to instant via basm.css.
(function () {
  function basmToastInit() {
    var region = document.getElementById('toast-region');
    if (!region) return;
    new MutationObserver(function (muts) {
      muts.forEach(function (m) {
        m.addedNodes.forEach(function (n) {
          if (n.nodeType !== 1 || !n.classList || !n.classList.contains('toast')) return;
          setTimeout(function () {
            n.classList.add('toast-out');
            setTimeout(function () { n.remove(); }, 200);
          }, 3000);
        });
      });
    }).observe(region, { childList: true });
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', basmToastInit);
  else basmToastInit();
})();
