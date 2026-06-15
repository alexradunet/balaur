# Balaur — Basm Design System (Hearthwood canon)

**Basm** (Romanian for *fairy tale*) is the design language of **Balaur**, a self-hosted, FOSS personal AI companion that lives on a box you own. The system is pixel-art, 16-bit, rooted in Romanian fairy tales and the balaur (the many-headed fairy-tale dragon): **woven, tactile, storybook, and hand-built — a usable app that reads like a fantasy RPG**.

The promise: **Your personal wise companion that grows with you.**

## Sources

- **GitHub — product:** https://github.com/alexradunet/balaur — Go binary embedding PocketBase, HTMX web UI, `html/template` pages. Art: `web/static/` (crest, logo, 32 avatars). Page markup: `web/templates/`.
- **GitHub — design exploration:** https://github.com/alexradunet/basm-design-system.

> **Canon note:** this project intentionally diverges from the repo's `basm.css` (Forest at Dusk). The **Hearthwood** direction here — tavern oak, candlelit gold, parchment dialogue panels, RPG construction — is the new source of truth. The repo should migrate toward these tokens, not the other way around. The pre-Hearthwood comparison sheet lives at `explorations/RPG Directions.html`.

## Brand essence

- **One companion, one main head.** Balaur has one master life conversation (the main head). Focused work opens a temporary sub-head — a branch — then merges back. Growth is depicted as another Balaur head on the same body, **never** a plugin tile, extension store, or multi-agent swarm. "New branch, new head."
- **Wise companion, quiet guardian.** Warm to you, steady, unbudging toward anything that would misuse what it knows. Privacy is the quiet second beat, never the lead.
- **Local-first, sovereign by construction.** Single binary, SQLite file, your box, your data. The repo is the system.
- **The conversation is an RPG dialogue.** You and Balaur speak across parchment panels with nameplates; decisions are made by choosing a spoken line, not by clicking admin chrome.

Keywords: wise · companion · grows-with-you · remembers · trustworthy · pixel-art · 16-bit · woven · tactile · storybook · tavern · parchment · local-first · private · open-source · self-hosted.

## CONTENT FUNDAMENTALS

**Voice:** a wise old companion who genuinely likes you and would never flatter you. Warm, calm, candid, patient, loyal. Never servile, hypey, cutesy, ominous, or mysterious-for-effect.

- **Register:** warm, wise, plain-spoken. Short, weighted sentences. Concrete nouns: *SQLite file, your box, the chosen model, Markdown*.
- **No exclamation marks. No hype. No emoji in product UI.**
- **Person:** Balaur speaks as "I"; the owner is "you." UI labels address the owner directly: "Awaiting your word.", "Tell Balaur in chat what to keep for you."
- **Casing:** sentence case for headings and body; lowercase for mono meta-labels (`used ×3`, `tool · search`, `recall: …`); UPPERCASE comes from CSS (`text-transform`) on nav, buttons, tags, kickers — never typed by hand.
- **Mythic seasoning, sparingly:** "I shall weigh the matter." · "A new head wakes." · "Then it is written." Plain technical truth is the base for anything that matters: "Your data stays on the box unless you switch models yourself."
- **Dialogue choices speak in the owner's voice.** When Balaur asks for a decision, the options are short spoken lines with the mechanical effect as a small mono hint. Established vocabularies — memory: "Keep it." · "Not this." · "Ask me another time."; task confirmation: "Hold me to it." · "Just this once." · "Let it be."; nudge: "It is done." · "At nightfall." · "Tomorrow, I swear it." · "Let this one go." The decision IS the conversation; never break immersion with admin verbs as the label.
- **Dry warmth in empty states:** "Nothing on the book. Tell Balaur in chat what to keep for you." · "No date yet — they wait without nagging." · "quiet" (timeline day with no items).
- **Name:** *Balaur*. Avoid "the Balaur app" and "Balaur AI."

**Signature lines:** Your personal wise companion that grows with you. · Remembers what matters. · New branch, new head. · Your life is not a product. · The repo is the system. · Woven, not rendered.

**Honesty rule:** copy must match what is actually shipped. Never state roadmap features (vault mirror, embedding recall, encrypted export, multi-human accounts, channel adapters) as shipped.

## VISUAL FOUNDATIONS — Hearthwood

**The room:** a tavern study at night. The page is dark oak; fixed chrome (topbar, chatbar) is beveled wood plank; everything written — speech, cards, journals — is parchment with ink. Candlelit gold and ember are the warm accents; teal is the otherworldly one (thinking, tool glyphs, recall).

**Principles:** Woven, not rendered · Craft over gloss (no glassmorphism, no soft gradients, no blur, no AI sparkle) · Dark-first guardian domain · Pixel-hard 16-bit construction · A usable app first — RPG flavor never costs legibility or density.

- **Materials (the core system):**
  - *Oak page* — `--bg` with 4px pixel-dither grain (`--grain-warm`).
  - *Wood chrome* — `--chrome` + plank lines (`--wood-planks`) + raised bevel (`--bevel-up`); 2px near-black pixel outline (`--outline-2`). Topbar, chatbar, tool rows, tags, tabs, model chips, avatar frames. Helper class: `.wood`.
  - *Parchment* — `--surface` + ink grain (`--grain-ink`), `--ink` text, 2px `--parch-edge` border, paper bevel + 3px hard drop (`--parch-bevel`). Speech panels, knowledge/task cards, calendar cells, journals, profile cards, dialogs. Helper class: `.parch`.
  - *Inset wells* — `--chrome-2` + `--bevel-in` for things carved into the wood: tool rows, recap cards, the chat input, inactive model chips.
- **Color tokens** in `tokens/colors.css` via `light-dark()`. Dark Hearthwood is the identity; light is a daylight study (page lightens, wood and parchment stay). Parchment + ink tokens (`--surface*`, `--ink*`, `--gold-ink`, `--teal-ink`, `--indigo-ink`, `--good-ink`) are constant across modes. Status colors always pair with text labels, never color-only.
- **Type:** unchanged roles — `--font-display` Pixelify Sans (headings, 20px+), `--font-pixel` Silkscreen (nameplate + runes ONLY), `--font-body` Work Sans 17px/1.6, `--font-mono` JetBrains Mono (functional small layer, uppercase via CSS). Self-hosted woff2 in `fonts/`.
- **Shape:** square corners (`--radius: 0`) — RPG panels don't round. 2px borders everywhere; dashed = archived/dropped/draft.
- **Shadows = bevels + hard drops.** No blur anywhere. Raised chrome uses `--bevel-up`; carved wells use `--bevel-in`; parchment uses `--parch-bevel`; interactive slabs add `--drop-hard` (0 3px 0).
- **Press physics:** hover brightens (`filter: brightness(1.04–1.12)`) or warms border to gold; active sinks 3px (`translateY(3px)`) and inverts the bevel — the press is physical. Transitions 80ms crisp.
- **Chat = RPG dialogue.** Two characters face each other across the table: Balaur speaks from the **left**, the owner from the **right** (portrait mirrored via CSS `scaleX(-1)` to face the words — never new art). Portraits are large (96px; 64px on small screens), set in a beveled wood frame with a thin inner keyline — gold for Balaur, indigo for the owner — and the speaker's name on a small wood plate hung beneath. The parchment speech panel carries a stepped pixel tail pointing at its speaker. Tool events are dark inset wood slabs indented to the text column (`--chat-gutter`): the audit trail, visible, never hidden.
- **Decisions = dialogue choices.** A gold-bordered parchment panel (`.choices`, full column width, owner's portrait at the right) listing spoken lines; ▸ ember cursor appears on hover; mono hint shows the mechanical effect; keys 1–9 pick. Reserved for moments Balaur asks for the owner's word. The panel takes the **composer's place** at the end of the conversation while the question is open; the draft input returns once the owner has spoken.
- **The composer is an unsent message.** In chat, the input is a dashed parchment draft panel in the owner's position at the end of the conversation (`MessageDraft`); dashed = draft, solidifying to gold while writing, no speech tail until spoken. Sending appends it as the owner's line. Embedded records (the inline task card) ride the same message column at the same width. Operational pages (Tasks, Memory, Life) keep the fixed wood-ledge `ChatBar` for page-level capture.
- **Ornament budget:** gold pixel **corner brackets** (`.ornate`) mark only panels that matter — proposed knowledge, dialogue choices, dialogs. Everything else gets at most the 7px gold-ink corner notch. At least one restrained folk motif per surface: stitch line, notch, folk band, or brackets — never all at once.
- **Motion:** minimal and diegetic. `basm-glow` breathing teal on the avatar while thinking/working; stepped `thinking-dots`; newly arriving Balaur lines type out RPG-style (ember block cursor, ~14ms/char, click the panel to skip) — history never re-types. No sprite animation. All of it respects `prefers-reduced-motion` (typewriter renders instantly).
- **Imagery:** hi-bit pixel art from the crest's world, `image-rendering: pixelated`, always borderless — frames belong to HTML context (wood frames, parchment panels), never the art.
- **Headings on the page** are `--fg-strong` (Pixelify); headings on parchment are `--ink`. Section headings carry a trailing stitch rule (`.k-heading::after`); the proposed section is gold: "Awaiting your word."

## ICONOGRAPHY

- **Icons are pixel-art PNGs** from `assets/icons/` — drawn on a 12px grid in the crest's palette (amber/gold, teal, parchment, dark outline), shipped borderless on transparency, rendered with `image-rendering: pixelated`, never boxed. Vocabulary: `quill` write/add · `tome` memory · `gem` importance/reward · `hourglass` snooze/recurrence · `lens` search/recall · `orb` recap/insight · `scroll` tasks · `bell` nudge · `shield` OS access/guardian · `flame` hearth/streaks · `key` skills · `check` done · `rune_x` drop/dismiss.
- Sizes: 18px in tool rows (`img.tool-icon`), 26px beside page headings (`.h-icon`). Never scale below 14px — use a text rune instead.
- **Text-level runes stay typographic** (mono, `--teal`/`--teal-ink`): `▪` tag prefix · `▸` dialogue-choice cursor · `⟳` recurring · `◑`/`☼` theme toggle · `◂ ▸` calendar nav. No icon fonts, no SVG sets, no detailed line icons.
- Tool rows prefer the pixel icon (ToolRow `icon` prop); the legacy glyph (`⌥ ✎ ◈ ⌕…`) is the fallback when no icon fits.
- **No emoji, ever, in product UI.**
- All pixel art (crest, logo, avatars, icons) ships borderless — frames belong to HTML context, never the art. New icons must be drawn on the same 12px grid with the same palette; quantize, no anti-aliasing.

## Marks & avatars (`assets/`)

- `assets/crest.png` — the owner's crest: three-headed amber balaur with teal orb and tome; its folk-diamond frame is part of the artwork. Topbar brand (34px) and empty-chat hearth.
- `assets/logo.png` — gold Balaur head medallion; favicon and touch icon.
- `assets/avatars/balaur-01..16.png` — 16 Balaur head personalities (01 Wise default, 02 Ancient, 03 Guardian, 04 Scholar, 05 Wild, 06 Storm, 07 Night, 08 Young, 09 Ember, 10 Frost, 11 Healer, 12 Trickster, 13 Dreamer, 14 Forest, 15 Dawn, 16 Sage).
- `assets/avatars/soul-01..16.png` — 16 owner "soul" portraits (01 Him … 16 Pricolici; see `guidelines/brand-souls.html`).
- Avatar rules: strict side profile facing right (mirror with CSS `scaleX(-1)` only); single static 256×256 PNG; borderless; pixelated; activity via `basm-glow`, never frame animation.

## Index

| Path | What |
|---|---|
| `styles.css` | Global CSS entry — imports everything below |
| `tokens/colors.css` · `tokens/typography.css` · `tokens/layout.css` | Hearthwood tokens: materials, ink, bevels, grain, type, focus |
| `basm/base.css` | Reset, element styles, `.parch` / `.wood` / `.ornate` material helpers |
| `basm/components.css` | Topbar, buttons, cards, motifs, RPG chat (speech panels, nameplates, dialogue choices, message draft), chatbar, dialog |
| `basm/pages.css` | Knowledge cards, recap, tasks/calendar/timeline, life, day, profile, model switcher |
| `fonts/` | Self-hosted woff2: Pixelify Sans, Silkscreen, Work Sans, JetBrains Mono |
| `assets/` | crest, logo medallion, 32 pixel-art avatars, 13 pixel icons (`assets/icons/`) |
| `guidelines/` | Foundation specimen cards (Design System tab) |
| `components/core/` | Button (primary/ghost/wood), Tag, Card, Stitch, FolkBand, Pips |
| `components/chat/` | Avatar, ChatMessage, ToolRow, DialogueChoices, MessageDraft, ChatBar |
| `components/knowledge/` | KnowledgeCard, TaskCard |
| `components/navigation/` | Topbar, Tabs |
| `ui_kits/balaur/` | Interactive recreation: RPG chat with dialogue-choice memory approval, tasks, memory, life, profile |
| `templates/balaur-app/` | Template: the full Balaur web UI (Design Component, mounts the whole app) |
| `templates/hearthwood-page/` | Template: blank page scaffold — wood topbar over a parchment panel |
| `explorations/RPG Directions.html` | The three-direction comparison that chose Hearthwood |
| `SKILL.md` | Agent-skill entry point |

## Quick checklist for any Balaur surface

1. One core voice: wise companion — warm, steady, never flattering. Lead with help/memory/growth; privacy second.
2. Heads = focused branches inside one companion; never separate AIs.
3. Mythic seasoning over plain technical truth. No hype, no emoji, no exclamation marks.
4. Materials honest: oak page, wood chrome, parchment content, ink text. Tokens from `styles.css` (`var(--gold)`), never hand-picked hexes.
5. Square corners, 2px outlines, bevels + hard drops; presses sink 3px. No blur, no soft gradients.
6. Decisions in chat are dialogue choices in the owner's spoken voice; ornate gold brackets only on panels that matter.
7. At least one restrained folk/pixel motif per surface: stitch line, notch, folk band, or brackets.
8. Art ships borderless; frames belong to HTML context. Avatars: 256px PNG, right-facing profile, CSS glow for state.
9. Keep assets local; no third-party CDNs in product UI.
