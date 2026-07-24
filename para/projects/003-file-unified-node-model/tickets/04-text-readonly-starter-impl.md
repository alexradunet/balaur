---
phase: implement
status: done
project: 003-file-unified-node-model
ticket: 04
date: 2026-07-24
plan: plan.md
commit: a88ee7b
branch: 003-ticket-04
---

# Implementation: Ticket 04 — text read-only + greenfield file-backed starter (plan Steps 9-10)

Completes the ADR-0004 hard cut. `text` is now read-only interop in the UI (Balaur
never authors or mutates a text node, but imported/external text nodes still render),
and the graph starter is regenerated as a fully file-backed space so a fresh install
demonstrates the unified model. Scope is `app.js` only.

## Steps completed

- [x] Step 9 (text read-only, inspector) — removed the text-editing textarea branch
  `else if(item.type==="text")fields.push({key:"text",...})` and replaced it with a
  read-only note: `else if(item.type==="text")notes.push({text:"Imported text node ·
  read-only. Balaur authors file-backed notes (ADR-0004)."});` (now `app.js:1299`).
  The `text` render branch in `renderNodes` stays (`app.js:922`, `markdownToHTML(node.text)`),
  so imported documents remain visible. Verified: `grep -n 'item.type==="text"' app.js`
  shows only the read-only note branch plus the render/title/summary reads.
- [x] Step 9 (remove dead `item.text=` mutation) — deleted the now-dead
  `aiTitle`/`aiPrompt` mutation of `item.text` in `applyInspectorField`; those fields
  are handled by the `scope==="note"` branch from Ticket 02 (an AI operator is a note
  file-node). The dangling `else if` became `if`. `buildAICardText` stays referenced
  via `noteBodyFromInspector` (`app.js:605`). Verified: `grep -n 'item.text=' app.js`
  returns NO matches.
- [x] Step 10 (path consts) — defined `const STARTER_NOTES={...}` beside
  `STARTER_TASK_PATH` (`app.js:51-62`) with ten slugged `notes/*.md` paths matching the
  `slugify` convention (e.g. `homeGuide:"notes/start-here-your-life-as-a-graph.md"`,
  `inboxTrip:"notes/autumn-city-break-idea.md"`, `archivePortfolio:"notes/portfolio-refresh-completed.md"`).
- [x] Step 10 (starter rewrite) — in `createGraphStarterWorkspace()`, replaced every
  inline `text` node (`home-guide`, `inbox-guide`, `inbox-trip`, `projects-guide`,
  `cb-note`, `wiki-guide`, `wiki-budget`, `wiki-subscriptions`, `archive-guide`,
  `archive-portfolio`) with a standard `file` node referencing its `STARTER_NOTES`
  path. Every node `id`, geometry (`x/y/width/height`), and `color` is preserved,
  including the dormant `#6c757d` (`DORMANT_NODE_COLOR`) on `archive-portfolio`. Every
  labelled edge is preserved unchanged: `e-inbox-filed` (`filed-to`, color 6),
  `e-cb-partof` (`part-of`, color 6), `e-wiki-relates` (`relates-to`, color 5). The task
  (`cb-task` → `STARTER_TASK_PATH`) and the journal (`wiki-journal` → `journalFile`) are
  unchanged. Only node storage changed from `type:"text"` to `type:"file", file:<path>`.
- [x] Step 10 (seeding) — in `seedGraphStarterEntities()`, seeded each starter note file
  at the exact path its `file` node references via `noteRepository.createNote({path, body, kind})`
  (the explicit `path` override from Ticket 01), wrapped in per-seed try/catch so
  re-seeding is idempotent (`createNote` writes with `expectedHash:null`, which throws
  `WRITE_CONFLICT` when the file already exists — same pattern as the task seed). The
  inbox capture (`inboxTrip`) carries `kind:"inbox"`; the reference pages (`wikiBudget`,
  `wikiSubscriptions`, `archivePortfolio`) carry `kind:"reference"`; the guides and
  `cbNote` carry no marker. The repository prepends the inert marker via `_noteContent`,
  so the seeded bytes match the previous inline text exactly (`NOTE_KIND_MARKERS` in
  `note-catalog.js` equals app.js `NOTE_MARKERS`). The legacy `demoCanvas` (`app.js:24-42`)
  is untouched.

## Files changed

- `app.js` — added `STARTER_NOTES` path consts; converted the ten starter text nodes to
  file nodes; added idempotent note seeding in `seedGraphStarterEntities`; replaced the
  inspector text textarea with a read-only note; removed the dead `item.text=` mutation
  in `applyInspectorField`. 45 insertions, 13 deletions.

No other file touched. `storage/life-indexer.js`, `storage/memory-index.js`,
`storage/canvas-validate.js`, `storage/note-catalog.js`, `storage/note-repository.js`,
and the task/journal/habit/component-card/widget repositories are byte-for-byte unchanged
(`git diff --stat` against them is empty).

## Verification results

```
$ node --check app.js                      # exit 0
$ git diff --check HEAD~1 HEAD             # exit 0 (no whitespace errors)
$ git status --porcelain                   # clean after commit (only ` M app.js` before)
$ git show --stat HEAD                     # app.js | 58 +++…---  (1 file, 45+/13-)

$ grep -n 'item.text=' app.js              # NO matches (exit 1)
$ grep -n 'type:"text"' app.js             # only demoCanvas lines 30-35
$ grep -c 'file:STARTER_NOTES\.' app.js    # 10 (one per converted node)
$ grep -c 'path:STARTER_NOTES\.' app.js    # 10 (one per seeded note)

$ node --test storage/phase1.test.js storage/phase5.test.js storage/phase8.test.js \
    storage/phase-query.test.js storage/note-repository.test.js
ℹ tests 91
ℹ pass 91
ℹ fail 0
```

Edges preserved (verified by grep): `e-inbox-filed` (app.js:81), `e-cb-partof`
(app.js:91), `e-wiki-relates` (app.js:99), each with its label and color intact.
`demoCanvas` unchanged: `git diff` shows no edits to any `n-focus`/`n-project`/…/`n-reading`
node or the `demoCanvas` binding.

## Browser-pending behaviors

These ticket-04 behaviors are NOT covered by the Node seam (app.js UI) and remain
browser-pending; they were not fabricated as Node tests:

- A selected foreign `text` node shows the read-only note in the inspector (no Markdown
  textarea) while its content still renders on the canvas.
- A true fresh install (`createGraphStarterWorkspace`) builds Home + four hub canvases
  whose guide/example cards render from the seeded `notes/*.md` files; the inbox capture
  and reference pages carry their inert markers; the dormant archive card keeps `#6c757d`.
- Re-running `seedGraphStarterEntities` is idempotent (no duplicate-note errors surface).

Verify via the `browser-check` skill smoke suite when a browser is available.

## Issues encountered

None. Plan Steps 9-10 matched the live code (line numbers had shifted after Tickets 01-03,
but every referenced construct was located by content and matched the excerpts). No STOP
condition occurred; no out-of-scope file was touched; no ambiguity required guessing.
