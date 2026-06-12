# Plan 022: Consolidate Profile, Skills, and Models under a /settings page with a sidebar

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 9fd16ac..HEAD -- internal/web web/templates web/static/basm.css`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW–MED (pure UI restructure; all state-changing endpoints stay
  unchanged, so the blast radius is page rendering and navigation)
- **Depends on**: none
- **Category**: direction (owner-requested feature)
- **Planned at**: commit `9fd16ac`, 2026-06-12

## Why this matters

The owner asked for one Settings page with a sidebar of categories instead of
three scattered top-level pages: **Profile**, **Skills**, and **Model
configuration**. Today `/profile`, `/skills`, and `/models` are separate pages
each linked from the topbar. After this plan, `GET /settings/{section}`
renders each of those three concerns inside a shared shell with a left
sidebar, the old URLs 302-redirect to their new homes, and the topbar carries
a single "Settings" link. Plans 023 (GGUF download manager) and 024 (OpenAI
provider manager) build new capability inside the Models section this plan
creates — land this first.

## Current state

Balaur is a single Go binary: PocketBase embedded, server-rendered
`html/template` + HTMX, **no SPA framework, no Node build step** (AGENTS.md).
Templates live in `web/templates/*.html`, are parsed once at startup
(`internal/web/web.go:146`), and every `{{define}}` shares one global
namespace across all files. CSS is `web/static/basm.css`; the design system
(DESIGN.md, "Basm") requires components to reference CSS custom properties
(`var(--gold)`), never hand-picked hexes.

Relevant files:

- `internal/web/web.go` — route registration (`Register`) and the `handlers`
  struct. Routes today (web.go:159-196, excerpt):

  ```go
  se.Router.GET("/", h.home)
  se.Router.GET("/models", h.modelsPage)
  ...
  se.Router.GET("/skills", h.skillsPage)
  ...
  // Profile page and its sub-actions.
  se.Router.GET("/profile", h.profilePage)
  se.Router.POST("/ui/profile/name", h.saveName)
  se.Router.POST("/ui/profile/soul-avatar", h.setSoulAvatarFromProfile)
  se.Router.POST("/ui/profile/balaur-avatar", h.setBalaurAvatarPref)
  ```

- `internal/web/profile.go` — `profilePage` renders `profile.html` with
  `buildProfileData(false)`; the three POST fragment handlers re-render
  `profile_identity_card`, `profile_soul_section`, `profile_balaur_section`.
- `internal/web/models.go` — `modelsPage` renders `models.html` with
  `modelsData()`; the panel fragment is `models_panel`, re-rendered by
  `modelsPanel(e, msg)` and targeted by HTMX as `#models-panel`.
- `internal/web/knowledge.go` — `skillsPage` (knowledge.go:40-53) builds a
  `map[string]any` with keys `Title`, `Kind` ("skills"), `Proposed`,
  `Active`, `Archived`, `Query`, and renders `knowledge.html`. `memoryPage`
  uses the same template with `Kind: "memories"` plus `Categories`.
- `web/templates/profile.html` — full page (`<!DOCTYPE html>` … `</html>`)
  followed by three `{{define}}` fragments:
  `profile_identity_card` (id `identity-card`), `profile_soul_section`
  (id `soul-section`), `profile_balaur_section` (id `balaur-section`).
  Each fragment is an HTMX swap target for its own POST endpoint.
- `web/templates/models.html` — full page wrapper plus
  `{{define "models_panel"}}` (`<div id="models-panel">…`), which contains
  the model-choice grid and the "Add OpenAI-compatible API" form. All HTMX
  forms inside it target `#models-panel` with `hx-swap="outerHTML"` and post
  a hidden `target=models` field.
- `web/templates/knowledge.html` — full page; its `<main>` body renders
  Proposed / Active (with `.k-search` live search posting to
  `/ui/knowledge/{{.Kind}}/grid`) / Archived sections. The category `k-tabs`
  nav only renders when `.Categories` is non-empty — skills passes no
  categories, so skills shows search only.
- `web/templates/layout.html` — `{{define "topbar"}}` nav (layout.html:23-31):

  ```html
  <nav>
    <a href="/tasks">Tasks</a>
    <a href="/life">Life</a>
    <a href="/memory">Memory</a>
    <a href="/skills">Skills</a>
    <a href="/profile">Profile</a>
    <a href="/heads">Heads</a>
    <a href="/_/" target="_blank" rel="noopener noreferrer">Engine room</a>
  </nav>
  ```

- `web/static/basm.css` — tokens and components. The pattern to model the
  sidebar's look on is `.k-tab` (basm.css:618-638): mono font, uppercase,
  `var(--muted)` text, `var(--gold)` active background. `main` is styled at
  basm.css:125.
- Redirect convention: `internal/web/day.go:51` —
  `return e.Redirect(http.StatusFound, "/day/"+today.Format(dayLayout))`.
- Test harness: `internal/web/handlers_test.go` — `tests.ApiScenario` with
  `TestAppFactory: newWebApp`; `TestHandlerHomePage` is the structural
  pattern. `internal/web/templates_test.go` exercises template rendering.

Design constraints from DESIGN.md to honor: Basm voice ("warm, wise,
plain-spoken … no exclamation marks, no hype, no emoji in product UI");
templates must reference CSS tokens; the PocketBase admin stays "the
superuser engine room", never the product surface.

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Tests     | `go test ./internal/web/...`     | ok                  |
| All tests | `go test ./...`                  | all packages ok     |
| Vet       | `go vet ./...`                   | exit 0              |
| Format    | `gofmt -l internal web`          | no output           |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope** (the only files you should modify or create):
- `internal/web/web.go` (routes)
- `internal/web/settings.go` (create — settings page handler)
- `internal/web/profile.go`, `internal/web/models.go`,
  `internal/web/knowledge.go` (page handlers become redirects; data builders
  get reused)
- `web/templates/settings.html` (create — shell + sidebar + sections)
- `web/templates/profile.html`, `web/templates/models.html`,
  `web/templates/knowledge.html` (drop dead page wrappers / factor body)
- `web/templates/layout.html` (topbar nav)
- `web/static/basm.css` (settings layout styles, appended at the end)
- `internal/web/handlers_test.go`, `internal/web/templates_test.go` (tests)

**Out of scope** (do NOT touch, even though they look related):
- Any `POST /ui/...` endpoint path or its form contract — the chatbar model
  picker on the home page posts to the same endpoints; changing paths breaks
  it.
- `internal/store`, `internal/turn`, `migrations/` — no data or schema change
  is needed here.
- The `/memory` page beyond the shared-body factoring in Step 3 — memory
  stays a standalone topbar page by owner decision.
- The chat modal flow (`/ui/model/missing`, `/ui/model/download`) — plan 023
  rewrites it; leave it exactly as-is.
- Download manager / provider management features — plans 023 and 024.

## Git workflow

- Branch: `advisor/022-settings-shell`
- Commit style: conventional commits, e.g.
  `feat(web): settings page with sidebar; fold profile/skills/models under it`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Create the settings handler

Create `internal/web/settings.go`:

- `type settingsData struct { Title, Section string; Profile profileData;
  Models modelsPageData; Skills map[string]any }` — only the active
  section's field is populated.
- `func (h *handlers) settingsPage(e *core.RequestEvent) error` reads the
  `{section}` path value (`e.Request.PathValue("section")`):
  - `"profile"` → `data.Profile = h.buildProfileData(false)`
  - `"models"` → `data.Models, err = h.modelsData()` (propagate err as
    `e.InternalServerError("loading models", err)`)
  - `"skills"` → reuse the body of the current `skillsPage`: extract its
    data-building into `func (h *handlers) skillsData(q string)
    map[string]any` in `internal/web/knowledge.go` and call it here with
    `e.Request.URL.Query().Get("q")`
  - any other value → `return e.Redirect(http.StatusFound,
    "/settings/profile")`
  - then `return h.render(e, "settings.html", data)` with
    `Title: "Settings"` and `Section` set.
- `func (h *handlers) settingsRoot(e *core.RequestEvent) error` →
  `return e.Redirect(http.StatusFound, "/settings/profile")`.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0 (template not referenced
yet; handler compiles).

### Step 2: Create the settings template

Create `web/templates/settings.html`: a full page (model it on
`profile.html`'s wrapper: `page_head`, `topbar`) whose `<main>` is a
two-column settings layout:

```html
<main class="settings-page">
  <h1>Settings</h1>
  <div class="settings-layout">
    <nav class="settings-nav" aria-label="Settings sections">
      <a class="settings-nav-link{{if eq .Section "profile"}} settings-nav-active{{end}}" href="/settings/profile">Profile</a>
      <a class="settings-nav-link{{if eq .Section "skills"}} settings-nav-active{{end}}" href="/settings/skills">Skills</a>
      <a class="settings-nav-link{{if eq .Section "models"}} settings-nav-active{{end}}" href="/settings/models">Models</a>
    </nav>
    <div class="settings-content">
      {{if eq .Section "profile"}}
        {{template "profile_identity_card" .Profile}}
        {{template "profile_soul_section" .Profile}}
        {{template "profile_balaur_section" .Profile}}
      {{else if eq .Section "skills"}}
        {{template "knowledge_body" .Skills}}
      {{else if eq .Section "models"}}
        {{template "models_panel" .Models}}
      {{end}}
    </div>
  </div>
</main>
```

Sidebar links are plain `<a href>` full-page loads — KISS, no HTMX needed
for top-level navigation. Copy tone: plain nouns only, per DESIGN.md.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0. (Template parse errors
surface at first `Register` in tests — checked in Step 6.)

### Step 3: Factor the knowledge page body into a shared define

In `web/templates/knowledge.html`, move everything between `<main>` and
`</main>` (the `h1` excluded — keep `<h1>{{.Title}}</h1>` in the page) into
`{{define "knowledge_body"}}…{{end}}` at the bottom of the same file, and
replace the moved markup with `{{template "knowledge_body" .}}`. The memory
page (`/memory`) must render byte-identically except for this refactor.
Do not change any `hx-get` URL inside it — the skills search fragment
endpoint `/ui/knowledge/skills/grid` keeps working inside the settings shell
because the swap target `#k-active-grid` is part of `knowledge_body`.

Note: the `k-tabs` `href="/{{.Kind}}?...”` fallback links only render when
`.Categories` is non-empty — true only for memory, so no skills URL inside
the body needs changing.

**Verify**: `go test ./internal/web/...` → ok (existing template tests parse
all templates).

### Step 4: Convert old pages to redirects; trim dead wrappers

- `internal/web/profile.go`: `profilePage` body becomes
  `return e.Redirect(http.StatusFound, "/settings/profile")`. Keep
  `buildProfileData` and all three POST handlers unchanged.
- `internal/web/models.go`: `modelsPage` body becomes
  `return e.Redirect(http.StatusFound, "/settings/models")`. Keep
  `modelsData`, `modelsPanel`, and every other handler unchanged.
- `internal/web/knowledge.go`: `skillsPage` body becomes
  `return e.Redirect(http.StatusFound, "/settings/skills")` (preserve the
  `q` query param if present: redirect to
  `"/settings/skills?q=" + url.QueryEscape(q)` only when `q != ""`).
  `memoryPage` stays a full page.
- `web/templates/profile.html`: delete the full-page wrapper (everything
  before the first `{{define}}`), keeping the three fragment defines. The
  file becomes defines-only; add a one-line top comment saying these
  fragments render inside `settings.html` and as HTMX swap responses.
- `web/templates/models.html`: same treatment — keep only
  `{{define "models_panel"}}`, plus a heading comment.
- In `internal/web/web.go` `Register`, add:

  ```go
  se.Router.GET("/settings", h.settingsRoot)
  se.Router.GET("/settings/{section}", h.settingsPage)
  ```

  Keep the existing `/profile`, `/models`, `/skills` GET registrations
  (they now serve redirects — old bookmarks keep working).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0;
`grep -c "DOCTYPE" web/templates/profile.html web/templates/models.html` →
`0` for both.

### Step 5: Topbar + CSS

- `web/templates/layout.html` topbar nav: remove the `/skills` and
  `/profile` links; add `<a href="/settings">Settings</a>` where the Profile
  link was (final order: Tasks, Life, Memory, Heads, Settings, Engine room —
  move Heads up so Settings sits next to Engine room).
- Append to `web/static/basm.css` a `/* ── Settings ── */` section:
  `.settings-layout` (CSS grid: `grid-template-columns: 180px 1fr; gap: 24px;`
  collapsing to one column under `@media (max-width: 700px)`),
  `.settings-nav` (vertical flex, `position: sticky; top: 24px;
  align-self: start;`), `.settings-nav-link` and `.settings-nav-active`
  copying the `.k-tab` / `.k-tab-active` recipe (basm.css:619-637) — mono
  font, uppercase, `var(--muted)`, active = `var(--gold)` background with
  `var(--gold-deep)` border and `#1a0e07` text exactly as `.k-tab-active`
  does. Use only existing custom properties; no new hexes besides the
  `#1a0e07` already used by `.k-tab-active`.

**Verify**: `grep -n "settings-nav-active" web/static/basm.css` → match;
`grep -c 'href="/profile"' web/templates/layout.html` → `0`.

### Step 6: Tests

In `internal/web/handlers_test.go`, add a `TestSettingsPages` modeled on
`TestHandlerHomePage` (`tests.ApiScenario`, `TestAppFactory: newWebApp`):

- `GET /settings` → 302 (use `ExpectedStatus: 302`)
- `GET /settings/profile` → 200, `ExpectedContent` includes
  `"identity-card"` and `"settings-nav"`
- `GET /settings/models` → 200, contains `"models-panel"`
- `GET /settings/skills` → 200, contains `"k-active-grid"`
- `GET /settings/bogus` → 302
- `GET /profile` → 302; `GET /models` → 302; `GET /skills` → 302
- `GET /memory` → 200 still contains `"k-active-grid"` (the Step 3 refactor
  didn't break memory)

**Verify**: `go test ./internal/web/...` → ok, new test included.

## Test plan

Covered by Step 6. Structural pattern: `TestHandlerHomePage` in
`internal/web/handlers_test.go`. Cases: each section renders (200 + marker
content), root and unknown sections redirect, the three legacy URLs
redirect, memory page unaffected.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0; `gofmt -l internal web` prints nothing
- [ ] `go test ./...` → all packages ok, including `TestSettingsPages`
- [ ] `grep -rn 'href="/profile"\|href="/skills"' web/templates/layout.html`
      → no matches; `grep -n 'href="/settings"' web/templates/layout.html`
      → one match
- [ ] `grep -c "DOCTYPE" web/templates/profile.html web/templates/models.html`
      → 0 and 0
- [ ] No files outside the in-scope list modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `internal/web` or `web/templates` changed since
  `9fd16ac` and the excerpts above no longer match.
- The fragment defines (`profile_identity_card`, `models_panel`,
  `knowledge_body`) collide with an existing define name in another template
  file (templates share one namespace; `template.Must` will panic in tests).
- Any existing handler test fails after Step 4 in a way that isn't a literal
  expectation on the old page markup (markup-expectation updates are fine;
  behavioral failures are not).
- You find a consumer of `GET /models`, `/profile`, or `/skills` other than
  a browser link (e.g. something in `internal/cli` or `internal/tools`
  fetching those pages) — grep first: `grep -rn '"/models"\|"/profile"\|"/skills"' internal/ --include="*.go"`.

## Maintenance notes

- Plans 023 and 024 add new blocks inside the Models section
  (`models_panel`) and new routes under `/ui/model/...` — they assume this
  plan's structure. Land 022 → 023 → 024.
- `internal/web/models.go` was already flagged (first improve cycle) as a
  decomposition candidate; this plan deliberately leaves it alone. If it
  keeps growing in 023/024, split page-vs-fragment handlers then.
- Reviewer focus: confirm no `POST /ui/...` path changed, and that the home
  chatbar model picker (`chat_bar` template) still works — it shares the
  select/save endpoints with `models_panel`.
- Deferred: keeping `/profile` etc. as permanent redirects (301) — left at
  302 so changing the layout again stays cheap.
