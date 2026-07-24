---
phase: implement
status: done
project: 003-file-unified-node-model
ticket: 05
date: 2026-07-24
plan: plan.md
commit: 45db054
branch: 003-ticket-05
---

# Implementation: Ticket 05 — docs + verification gate (plan Steps 11-12)

Prose and verification only; no source changed. Documentation now matches the
shipped file-unified model (ADR-0004, tickets 01-04 on the integration branch),
and AGENTS.md §13 publishes the real suite total confirmed by the gate.

## Steps completed

- [x] Step 11 (docs): updated AGENTS.md §4.2, §3 repository map, §13; docs/life-data.md; docs/architecture.md; docs/generative-canvas.md — verified: `git diff --name-only` shows only those four files.
- [x] Step 12.1 (syntax): `node --check app.js storage/note-catalog.js storage/note-repository.js storage/note-repository.test.js` — verified: all four print OK, exit 0.
- [x] Step 12.2 (suite): ran the exact AGENTS.md §13 command including `storage/note-repository.test.js` — verified: `tests 196 / pass 196 / fail 0`; published count bumped 172 → 196 to match.
- [x] Step 12.3 (whitespace): `git diff --check` — verified: exit 0, no output.
- [x] Step 12.4 (browser smoke): served on 4173, ran `browser-check.mjs smoke --offline` with system chromium — 12/13 PASS, 1 stale-assertion FAIL (see Issues). Direct probe confirms the behavior is correct. Server killed afterward.

## Files changed

- `AGENTS.md` — §4.2: notes/inbox/reference and AI operators are path-identified `notes/*.md` placed by standard `file` nodes, marker in the body, `text` is read-only interop (ADR-0004). §3 map: added `storage/note-catalog.js` and `storage/note-repository.js` with the `notes/` layout. §13: added `storage/note-repository.test.js` to the `node --test` command and bumped the count to **196** (prior 172 + 24 note-repository tests). §10 (review follow-up, commit `45db054`): rewrote the last stale "text node" sentence so AI operators are standard `file` nodes referencing `notes/*.md` whose body carries the `<!-- orbit:ai-card -->` marker, with edge-derived inputs, and their output is a file-backed note on the reserved `AI output` edge (ADR-0004); security substance kept (allowlisted operations, debouncing, stable output-node reuse, queued reruns, cycle detection, file-node inputs resolve canonical file bodies).
- `docs/life-data.md` — added `notes/*.md` to the ownership list; added a `### Notes: notes/*.md` contract subsection (path identity, no mandatory frontmatter/`orbit-id`, kind as inert body marker, indexer treats as valid untyped Markdown, placements resolved by canvas scan); node-typing section now states notes are file-backed and `text` is read-only interop, and lists the `orbit:ai-card` marker.
- `docs/architecture.md` — ownership tree adds `notes/*.md`; projection paragraph adds `NoteCatalog`; startup step 5 adds the note catalog; repository list adds `FileNoteRepository` (path-identified notes, placements, drain primitive) and `NoteCatalog` (ADR-0004).
- `docs/generative-canvas.md` — superseded the four text-node statements: the AI one-shot note and the AI operator/output are standard `file` nodes referencing `notes/*.md`, the compatibility marker lives in the file body, interop clients see ordinary `file` nodes; security boundary stated unchanged.

## Verification results

Syntax (Step 12.1):

```
app.js OK
note-catalog OK
note-repository OK
note-repository.test OK
```

Full suite (Step 12.2), exact command now in AGENTS.md §13:

```
node --test \
  storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js \
  storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js \
  storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js \
  storage/phase10.test.js storage/phase-query.test.js \
  storage/note-repository.test.js

ℹ tests 196
ℹ pass 196
ℹ fail 0
```

The note-repository suite alone is 24 tests (172 + 24 = 196). Published count matches.

Whitespace (Step 12.3): `git diff --check` exit 0, no output.

Grep guards (Done criteria):

```
$ grep -n 'type:"text"' app.js
30-35: all inside the legacy demoCanvas (lines 24-42); no interactive authoring path.
$ grep -n 'item.text=' app.js
(no matches; exit 1)
```

The `type==="text"` read paths (nodeTitle/nodeSummary/AI-body resolution, the
read-only inspector note at app.js:1299, a task-count heuristic) are render/interop
only and author nothing.

Out-of-scope files unmodified: `git diff --stat` for this commit lists only
`AGENTS.md`, `docs/architecture.md`, `docs/generative-canvas.md`, `docs/life-data.md`.
`storage/life-indexer.js`, `storage/memory-index.js`, `storage/canvas-validate.js`,
and the task/journal/habit/component-card/widget repositories are untouched.

Regression pin present (storage/note-repository.test.js:250-275): asserts a note
placement is a valid `file` node on the canvas yet absent from `index.allPlacements()`
and `index.placementsForEntity()`, and that `replacePlacement` (drain) and
`deleteNote` still resolve the placement by canvas scan.

Browser smoke (Step 12.4), `browser-check.mjs smoke --offline` on system chromium:

```
PASS  boot: no uncaught console errors
PASS  boot: no failed asset requests
PASS  render: DOM cards match document nodes  [5/5]
PASS  index: canonical files indexed  [Files · 19 indexed]
PASS  select: card selected + inspector open
PASS  select: corner-bracket frame, no circles
PASS  create: portal dblclick navigates without creating nodes  [19 -> 19]
PASS  create: portal probe restores original canvas
PASS  create: note tool on card creates nothing  [6 -> 6]
FAIL  create: dblclick on background creates a note  [6 -> 7]
PASS  export: document is valid JSON Canvas
PASS  persist: reload keeps title and node count  [7 -> 7]
PASS  offline: shell renders from cache
```

Direct probe of the "failed" check (eval, fresh profile) confirms the behavior is
correct and the assertion is stale:

```
{ "before": 5, "after": 6, "lastType": "file",
  "lastFile": "notes/new-thought.md", "lastText": null, "isNoteFile": true }
```

## Browser-pending behaviors (labeled, not claimed verified by Node)

- File-backed starter renders every document node: smoke PASS (5/5 DOM cards match
  document nodes; 19 files indexed). Browser-verified this session.
- Double-click on empty background creates a file-backed note: confirmed by direct
  probe (`notes/new-thought.md` placed by a `file` node). Browser-verified this session.
- Editing a note writes the `.md`: NOT exercised by the smoke suite; remains
  browser-pending.
- Imported `text` node renders read-only (no edit textarea): NOT exercised by the
  smoke suite; remains browser-pending. The inspector read-only note exists at
  app.js:1299 (code path present, browser behavior pending).
- Offline shell from cache: smoke PASS. Browser-verified this session.

## Issues encountered

The smoke suite reports 12/13. The single FAIL is a stale assertion in the
browser-check skill, not a regression in the implementation. The check
(`browser-check.mjs:373`) asserts `info.count === before + 1 && info.lastText.includes("New thought")`
with `lastText = last?.text ?? ""`. The node count does increase (6 -> 7), but the
created node is now a file-backed `file` node with no `.text` property (its content
lives in `notes/new-thought.md`), so `lastText` is empty and the second conjunct
fails. The direct probe above shows the created node is `{type:"file", file:"notes/new-thought.md"}`,
exactly the ADR-0004 behavior the ticket asks to confirm.

Fixing the assertion (check `last.type === "file" && last.file.startsWith("notes/")`
instead of `last.text`) is a change to `.pi/skills/browser-check/scripts/browser-check.mjs`,
which is outside this ticket's hard scope (documentation files only). Flagged here for
a follow-up; the skill is local agent tooling, not deployed shell. No source file was
modified to make the gate pass.

The http.server started for the smoke run was killed afterward (port 4173 refuses
connections; no `http.server` process remains).

## Review follow-up

Standards review returned one [hard] finding: AGENTS.md §10 (line 248) still read
"AI operators remain standard text nodes…", contradicting the shipped `isAICard`
(requires `node.type==="file"` referencing `notes/*.md`), §4.2, ADR-0004, and
CONTEXT.md. Fixed in follow-up commit `45db054` (AGENTS.md only, 1+/1-). Re-verified:
`grep -n "text node" AGENTS.md` returns nothing (exit 1); `git diff --check` exit 0;
the §13 published count line is intact at **196** (no source/test changed, so the
suite total is unchanged); `git status` showed only AGENTS.md before the commit.

## Domain flags

None. The reconciled `CONTEXT.md` glossary (Note, Placement, Drain, Inbox note,
Reference page, AI operator, Text node (interop)) already matches the shipped model
and the documentation updated here; no contradictions found.
