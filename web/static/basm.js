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

(function watchChatbar() {
  const bar = document.getElementById('chatbar');
  if (!bar) return;
  basmSyncChatbarSpace();
  if (!window.ResizeObserver) return;
  new ResizeObserver(basmSyncChatbarSpace).observe(bar);
})();

window.addEventListener('resize', basmSyncChatbarSpace);

// ── Chatbar re-watch after HTMX OOB swap ──────────────────────────
document.body.addEventListener('htmx:oobAfterSwap', (e) => {
  if (e.detail?.target?.id === 'chatbar') {
    basmSyncChatbarSpace();
    const bar = document.getElementById('chatbar');
    if (bar && window.ResizeObserver) {
      new ResizeObserver(basmSyncChatbarSpace).observe(bar);
    }
  }
});

// ── Native <dialog> for model modal ───────────────────────────────
document.body.addEventListener('htmx:afterSwap', (e) => {
  const el = e.detail?.target;
  if (el && el.id === 'model-modal' && el.tagName === 'DIALOG') {
    if (!el.open) el.showModal();
  }
});

// ── Avatar state via HTMX lifecycle ───────────────────────────────
function basmSetAvatarState(kind, state) {
  document.querySelectorAll(`.balaur-avatar[data-kind="${kind}"]`).forEach(el => {
    el.dataset.state = state;
  });
}

document.body.addEventListener('htmx:beforeRequest', (e) => {
  if (e.target.closest?.('.chat-form')) {
    basmSetAvatarState('balaur', 'thinking');
  }
});
document.body.addEventListener('htmx:afterSettle', () => {
  basmSetAvatarState('balaur', 'idle');
});

// ── Knowledge tab active state ─────────────────────────────────────
document.body.addEventListener('click', (e) => {
  const tab = e.target.closest('.k-tab[hx-get]');
  if (!tab) return;
  tab.closest('.k-tabs, nav')
    ?.querySelectorAll('.k-tab')
    .forEach(t => t.classList.remove('k-tab-active'));
  tab.classList.add('k-tab-active');
});

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
