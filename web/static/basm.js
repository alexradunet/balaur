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

(function watchChatbar() {
  const bar = document.getElementById('chatbar');
  if (!bar) return;
  basmSyncChatbarSpace();
  if (!window.ResizeObserver) return;
  new ResizeObserver(basmSyncChatbarSpace).observe(bar);
})();

window.addEventListener('resize', basmSyncChatbarSpace);

// ── Chatbar re-watch after HTMX OOB swap (chatbar is replaced) ────
document.body.addEventListener('htmx:oobAfterSwap', (e) => {
  if (e.detail?.target?.id === 'chatbar') {
    basmSyncChatbarSpace();
    // Re-attach observer to the new element
    const bar = document.getElementById('chatbar');
    if (bar && window.ResizeObserver) {
      new ResizeObserver(basmSyncChatbarSpace).observe(bar);
    }
  }
});

// ── Native <dialog> for model modal ───────────────────────────────
// When HTMX swaps content into #model-modal (a <dialog> element),
// open it. The server renders just the inner content; the dialog
// wrapper lives in home.html and persists.

document.body.addEventListener('htmx:afterSwap', (e) => {
  const el = e.detail?.target;
  if (el && el.id === 'model-modal' && el.tagName === 'DIALOG') {
    if (!el.open) el.showModal();
  }
});

// ── Avatar state via HTMX lifecycle ───────────────────────────────
// Balaur avatar in the *pending* message already animates via the
// .balaur-avatar-live class added by balaurPrepareChat in home.html.
// This handler drives any *persistent* avatar elements elsewhere on
// the page (future topbar companion display, head sidebars, etc.)

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

// ── Knowledge tab active state (instant feedback before HTMX) ─────
// Removes the need for inline selectTab() in knowledge.html.
document.body.addEventListener('click', (e) => {
  const tab = e.target.closest('.k-tab[hx-get]');
  if (!tab) return;
  tab.closest('.k-tabs, nav')
    ?.querySelectorAll('.k-tab')
    .forEach(t => t.classList.remove('k-tab-active'));
  tab.classList.add('k-tab-active');
});
