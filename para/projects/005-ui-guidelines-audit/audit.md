---
phase: plan-audit
status: done
project: 005-ui-guidelines-audit
date: 2026-07-26
guidelines-source: https://raw.githubusercontent.com/vercel-labs/web-interface-guidelines/main/command.md
---

# UI audit against Vercel Web Interface Guidelines

Scope: `index.html`, `styles/*.css`, `elements/*.js`, `app.js` (UI surface),
`widgets/focus-orbit.html`, `vendor/pixel-loom/fonts.css`. Guidelines fetched
fresh at audit time. Verdict: strong baseline (reduced-motion system,
forced-colors, `inert` panels, menu keyboard nav, sandboxed widgets, no
`transition: all`, no zoom-blocking viewport). Gaps cluster in accessible
names, live regions, focus visibility, and pointer-only controls.

## Do-now batch (ticket 01)

Mechanical accessibility and copy fixes, all in files already touched by the
audit. No behavior change, no storage layer, no cache-semantics change.

- index.html:87-91 — tool buttons (select/pan/connect/note) title-only, need aria-label
- index.html:112 — #zoomLabel, #fitView need aria-label
- index.html:56 — #newGroup needs aria-label
- index.html:114 — .minimap is a <div> with onclick (app.js `$("#minimap").onclick=fitView`); make it a <button> so Enter/Space work
- index.html:45 — #saveState async text ("Saving…", "Save failed") not announced; add role="status"
- index.html:75 — #lifeIndexStatus same; add role="status"
- index.html:156 — #aiMessages chat stream silent to screen readers; add aria-live="polite"
- index.html:165 — #aiPrompt textarea has no label; add aria-label
- index.html:133 — #journalBody textarea has no label; add aria-label
- index.html:176,193,203 — dialog header glyphs (✓ ✎ ✦) decorative; add aria-hidden="true"
- index.html:207 — #aiModel is an identifier; add spellcheck="false"
- index.html:213 — straight apostrophe in "this browser's"; use curly ’
- styles/shell.css — #canvasTitle sets outline: 0 with no :focus-visible replacement
- styles/canvas.css — .canvas sets outline: 0 but is tabindex="0"; add inset :focus-visible ring
- styles/canvas.css — .minimap becomes a button; needs padding: 0 and cursor: pointer
- app.js — #toggleAIKey aria-label stays "Show API key" after toggling to Hide

## Later batch (needs a separate ticket)

- scroll containment: overscroll-behavior: contain on .ai-messages, .settings-dialog, .add-menu-panel, .canvas-icon-panel
- motion.css: .app-shell/.topbar animate grid-template-columns and .familiar-control animates right; move to transform-based slides
- app.js:1715,1751 — scrollIntoView({behavior:"smooth"}) ignores prefers-reduced-motion
- hardcoded dates: app.js task footer raw ISO "Plan 2026-07-22"; elements/task-list.js:80-81 slice(5) MM-DD; use Intl.DateTimeFormat
- global: touch-action: manipulation on interactive controls; -webkit-tap-highlight-color; env(safe-area-inset-*) for the installable PWA
- tokens: font-variant-numeric: tabular-nums for zoom label, sidebar counts, .today-stats, .task-dates time
- headings: text-wrap: balance on .today-head h1, .vault-landing-wordmark
- index.html: preload critical fonts (Newsreader 600, Work Sans 400)
- app.js: no beforeunload guard while a save is pending (comment at boot acknowledges async writes)
- .resize-handle is pointer-only; needs a keyboard resize path (design decision)
- URL does not reflect active canvas/view/filter; no deep links (design decision)
- button copy is sentence case; guidelines want Title Case (style decision, apply everywhere or nowhere)

## Pass (no action)

styles/foundation.css (focus ring, color-scheme, forced-colors, prefers-contrast);
styles/motion.css reduced-motion collapse and explicit transition properties;
elements/add-menu.js (popover, arrows/Home/End/Escape, focus restore);
elements/workspace-nav.js (aria-current, hidden separators); elements/inspector.js
(wrapped labels, color aria-labels); elements/widget-frame.js (sandbox="allow-scripts",
iframe title, role=status); widgets/focus-orbit.html (reduced motion); no
user-scalable=no; theme-color meta; native <dialog>; inert + aria-hidden AI panel;
toast role=status; menu semantics; autocomplete="new-password" on the key input;
confirm() on every delete-everywhere path; escapeHTML/safeURL everywhere;
Intl dates in Today/journal; font-display: swap on all faces.
