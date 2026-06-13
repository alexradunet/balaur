# Plan 056: Settings focus — a new settings card (profile + models), retire /settings, /profile, /models (Phase 6)

> **Executor instructions**: Follow this plan step by step. Run every Verify and
> confirm before moving on. On a STOP condition, stop and report. When done,
> update the `056` row in `plans/readme.md`. Execute with
> `superpowers:subagent-driven-development` or `superpowers:executing-plans`.
>
> **Drift check (run first)**: `git diff --stat 3b8a7d1..HEAD -- internal/web web/templates internal/cards`
> Authored at `3b8a7d1` (Phase 5 / plan 055 merged). Spec:
> `docs/superpowers/specs/2026-06-13-card-first-kill-the-pages-design.md`.
> If `internal/web/settings.go`, `internal/web/profile.go`, `internal/web/models.go`,
> `internal/web/cards.go`, `internal/cards/cards.go`, `web/templates/settings.html`,
> `web/templates/profile.html`, `web/templates/models.html`,
> `web/templates/cards.html`, `web/templates/home.html`, or
> `web/templates/layout.html` changed since `3b8a7d1`, compare excerpts; on
> mismatch, STOP.

## Status

- **Priority**: P2 (Phase 6 — the last surface phase)
- **Effort**: L (3 sections; many shared fragment templates; several link
  re-points; retires 4 routes incl. 2 redirects)
- **Risk**: MED (the section fragment templates are shared with live write
  endpoints — move, don't break; the model section has SSE-driven elements)
- **Depends on**: plans/050 (focus seam), plans/052/055 (new-card precedent),
  plans/053 (skills card — settings drops its Skills section) — DONE/merged
- **Category**: direction (card-first "kill the pages", Phase 6 of 8)
- **Planned at**: commit `3b8a7d1`, 2026-06-13

## Why this matters

`/settings/{section}` is the settings shell (Profile / Skills / Models);
`/models` and `/profile` are already just redirects into it. It becomes a new
**`settings` card** whose focus is the shell, with a `section` param. **Skills
leaves settings** — Phase 3 made the skills card focus the skills manager, so the
settings shell keeps only **Profile + Models**, removing the brief duplication
Phase 3 noted. All write endpoints (`/ui/profile/*`, `/ui/model/*`) and the
section fragment templates are reused unchanged.

## Current state

### Focus seam
`focusBodyHTML` (`internal/web/focus.go`) — cases quests/journal/day/memory/skills/
lifelog; default → `cardHTML`. This plan adds a `settings` case.

### The settings shell (`internal/web/settings.go`, `web/templates/settings.html`)
- `settingsPage` (GET `/settings/{section}`, `settings.go:17-38`): builds
  `settingsData{Section, Profile, Models, Skills}` per section and renders
  `settings.html`. **Delete.**
- `settingsRoot` (GET `/settings`, `settings.go:42-44`): redirect to
  `/settings/profile`. **Delete.**
- `settings.html`: a standalone doc — `<h1>` + a sidebar nav (Profile/Skills/
  Models, `href="/settings/{section}"`) + a section dispatch that includes
  `profile_identity_card`+`profile_soul_section`+`profile_balaur_section` /
  `knowledge_body` / `models_panel`. **Move the body to a focus template; drop
  the Skills nav + Skills branch.**

### Redirects into settings (delete both)
- `modelsPage` (GET `/models`, `models.go:131-133`) → `/settings/models`.
- `profilePage` (GET `/profile`, `profile.go:31-33`) → `/settings/profile`.

### Shared section bodies — KEEP (used by settings AND by write-endpoint SSE)
- Profile defines in `profile.html`: `profile_identity_card`,
  `profile_soul_section`, `profile_balaur_section` (re-rendered by `saveName` /
  `setSoulAvatarFromProfile` / `setBalaurAvatarPref`).
- Models defines in `models.html`: `models_panel`, `gguf_progress`, etc.
  (re-rendered by the `/ui/model/*` handlers; `gguf_progress` self-polls
  `/ui/model/gguf/progress`).
- `knowledge_body` (skills) — KEPT, but no longer included by settings (skills is
  the skills card now).

### Data builders — KEEP
`buildProfileData(false)` (`profile.go`), `modelsData()` (`models.go:135`),
`skillsData()` (`knowledge.go` — still used by the skills card focus).

### Write endpoints — KEEP (all of them)
`/ui/profile/{name,soul-avatar,balaur-avatar}`; all `/ui/model/*`
(`web.go:187-199`).

### Links to re-point (break when the pages die)
- topbar `<a href="/settings">Settings</a>` (`layout.html:26`) → `/focus/settings`.
- `home.html`: `model_switcher` "Manage models & APIs →" (`:116`) + "Set up a
  model →" (`:127`) `href="/settings/models"` → `/focus/settings?section=models`;
  "Your avatar & profile →" (`:136`) `href="/profile"` → `/focus/settings?section=profile`.
- `models.html`: two reload links `href="/settings/models"` (`:21,:23`) →
  `/focus/settings?section=models`.
- `cards.html`: skills card title (`:309`) + footer (`:318`) `href="/settings/skills"`
  → `/focus/skills` (skills is its own card now).
- `knowledge-focus.html:3` comment mentions `/settings/skills` — update wording
  (skills focus + the settings card no longer carries skills).

### Routes (`web.go`)
`GET /models` (`:186`), `GET /settings` (`:218`), `GET /settings/{section}`
(`:219`), `GET /profile` (`:222`) — delete. KEEP all `/ui/*` write routes.

### Card registry — add a `settings` spec.

### Tests
`TestSettingsPages` (`handlers_test.go`) exercises `/settings/{profile,skills,models}`
+ the retired-route guards; `TestProviderManager` etc. exercise the model write
endpoints. Read before editing.

## Commands you will need
```bash
go test ./internal/web/... ./internal/cards/...
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
grep -rnE '"/settings|href="/settings|"/profile"|href="/profile|"/models"|href="/models|settingsPage|settingsRoot|modelsPage|profilePage|"settings\.html"' internal web --include='*.go' --include='*.html'
```

## Scope

**In:** a new `settings` card (registry spec + tile + focus = the settings shell,
**Profile + Models** sections with a `section` param); move `settings.html`'s body
to a `settings_body` focus template (drop the Skills nav/branch); delete
`/settings`, `/settings/{section}`, `/models`, `/profile` routes +
`settingsPage`/`settingsRoot`/`modelsPage`/`profilePage`; drop the `Skills` field
from `settingsData`; re-point all links; adapt tests. **KEEP** every write
endpoint + handler, every section fragment define, `buildProfileData`,
`modelsData`, `skillsData`.

**Out:** any change to the write endpoints or the section fragment internals; the
skills card (unchanged); a profile/models "create" beyond what exists; Phase 7
topbar finalization.

## Git workflow
Branch `feature/card-first-kill-pages` (synced to `main` @ `3b8a7d1`). Commit
after each green step. A–C additive; D deletes; E docs.

## Steps

### Step A: move the settings body → `settings_body`, add the focus (additive)

**FIRST move the template:** create `web/templates/settings-focus.html` with a
`{{define "settings_body"}}` holding the sidebar (Profile + Models only — NO
Skills) + the Profile/Models section dispatch, with the nav links firing the
focus route. Base it on `settings.html`'s `<div class="settings-layout">…</div>`:

```html
{{- /* settings-focus.html — the settings shell (Profile + Models) as the
     settings card's focus body. Skills is its own card now. The section
     fragment defines (profile_*, models_panel) live in profile.html/models.html
     and are shared with the /ui/* write endpoints. */ -}}
{{define "settings_body"}}
<div class="settings-layout">
  <nav class="settings-nav" aria-label="Settings sections">
    <a class="settings-nav-link{{if eq .Section "profile"}} settings-nav-active{{end}}"
       href="/focus/settings?section=profile"
       data-on:click__prevent="@get('/focus/settings?section=profile')">Profile</a>
    <a class="settings-nav-link{{if eq .Section "models"}} settings-nav-active{{end}}"
       href="/focus/settings?section=models"
       data-on:click__prevent="@get('/focus/settings?section=models')">Models</a>
  </nav>
  <div class="settings-content">
    {{if eq .Section "models"}}
      {{template "models_panel" .Models}}
    {{else}}
      {{template "profile_identity_card" .Profile}}
      {{template "profile_soul_section" .Profile}}
      {{template "profile_balaur_section" .Profile}}
    {{end}}
  </div>
</div>
{{end}}
```

**Then drop the Skills field** from `settingsData` (`settings.go`) — it is no
longer rendered (only `settingsPage`, which this plan deletes, set it). Leave the
struct with `Title, Section, Profile, Models` (Title may also go if unused after
Step D).

**Add the focus renderer** in `internal/web/settings.go` (add `"html/template"`,
`"strings"` imports as needed):

```go
// settingsFocusHTML renders the settings card's focus body (Profile + Models).
// Was /settings/{section}. The section param defaults to profile; Skills is the
// skills card now.
func (h *handlers) settingsFocusHTML(params map[string]string) template.HTML {
	section := params["section"]
	if section != "models" {
		section = "profile"
	}
	data := settingsData{Section: section}
	switch section {
	case "models":
		m, err := h.modelsData()
		if err != nil {
			h.app.Logger().Warn("settings focus models failed", "err", err)
			return cardErrorStrip("could not load models")
		}
		data.Models = m
	default:
		data.Profile = h.buildProfileData(false)
	}
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "settings_body", data); err != nil {
		h.app.Logger().Warn("settings focus render failed", "err", err)
		return cardErrorStrip("could not open settings")
	}
	return template.HTML(b.String())
}
```

**Add the dispatch** in `internal/web/focus.go` `focusBodyHTML`:

```go
	case "settings":
		return h.settingsFocusHTML(params)
```

**Verify:** `go build ./... && go test ./internal/web/ -run 'TestFocus|TestSettings|TestProfile|TestProvider'` → ok.
**Commit:** `git add web/templates/settings-focus.html internal/web/settings.go internal/web/focus.go && git commit -m "feat(focus): settings card focus = the settings shell (profile + models)"`

### Step B: the `settings` registry spec + tile

**File:** `internal/cards/cards.go` — add to `registry`:

```go
		{
			Type:  "settings",
			Label: "Settings",
			Icon:  "key",
			W:     6,
			H:     24,
			Params: []ParamSpec{
				{Name: "section", Enum: []string{"profile", "models"}, Doc: "settings section (default profile)"},
			},
		},
```

**File:** `internal/cards/cards_test.go` — add `settings` to the all-types /
count / `HasManage` false-list (match the lifelog/day precedent).

**File:** `web/templates/cards.html` — add a light tile (links into the focus):

```html
{{define "ucard_settings"}}
<article class="kcard ucard ucard-settings" id="ucard-settings">
  <header class="kcard-head"><span class="kcard-kind"><img class="tool-icon" src="/static/icons/key.png" alt="">Settings</span></header>
  <ul class="ucard-stats">
    <li><a href="/focus/settings?section=profile">Profile</a></li>
    <li><a href="/focus/settings?section=models">Models &amp; APIs</a></li>
  </ul>
  <footer class="kcard-actions"><a href="/focus/settings">open settings →</a></footer>
</article>
{{end}}
```

**File:** `internal/web/cards.go` — register the renderer + a trivial tile render
(the tile is static links; no data fetch):

```go
	case "settings":
		return h.renderCardSettings(w, params)
```

```go
func (h *handlers) renderCardSettings(w io.Writer, _ map[string]string) error {
	return h.tmpl.ExecuteTemplate(w, "ucard_settings", nil)
}
```

**Verify:** `go build ./... && go test ./internal/web/ ./internal/cards/ -run 'TestUiCard|TestFocus|TestAll|TestHasManage'` → ok.

**Tests (add to `internal/web/focus_test.go`):**

```go
// TestUiCardSettingsTile: the settings tile renders with section links.
func TestUiCardSettingsTile(t *testing.T) {
	s := tests.ApiScenario{
		Name: "GET /ui/cards/settings renders the tile", Method: "GET",
		URL: "/ui/cards/settings", TestAppFactory: newWebApp, ExpectedStatus: 200,
		ExpectedContent: []string{"ucard-settings", `/focus/settings?section=models`},
	}
	s.Test(t)
}

// TestFocusSettingsProfile: /focus/settings renders the profile section by default.
func TestFocusSettingsProfile(t *testing.T) {
	s := tests.ApiScenario{
		Name: "GET /focus/settings → profile section", Method: "GET",
		URL: "/focus/settings", TestAppFactory: newWebApp, ExpectedStatus: 200,
		ExpectedContent: []string{`id="identity-card"`, "settings-nav"},
		NotExpectedContent: []string{">Skills<"},
	}
	s.Test(t)
}

// TestFocusSettingsModels: ?section=models renders the models panel.
func TestFocusSettingsModels(t *testing.T) {
	s := tests.ApiScenario{
		Name: "GET /focus/settings?section=models → models panel", Method: "GET",
		URL: "/focus/settings?section=models", TestAppFactory: newWebApp, ExpectedStatus: 200,
		ExpectedContent: []string{"settings-nav"},
	}
	s.Test(t)
}
```

> Confirm the real ids/markers in `profile_identity_card` (`id="identity-card"`)
> and `models_panel` against the templates; adjust the asserted substrings to what
> actually renders. Drop `NotExpectedContent` if the harness lacks it (use
> `AfterTestFunc`).

**Verify:** `go test ./internal/web/ -run 'TestUiCardSettings|TestFocusSettings' -v` → PASS.
**Commit:** `git add internal/cards web/templates/cards.html internal/web/cards.go internal/web/focus_test.go && git commit -m "feat(cards): new settings card — tile + profile/models focus"`

### Step C: re-point links

- `web/templates/layout.html`: topbar `href="/settings"` → `href="/focus/settings"`.
- `web/templates/home.html`: `:116` + `:127` `href="/settings/models"` →
  `/focus/settings?section=models`; `:136` `href="/profile"` →
  `/focus/settings?section=profile`.
- `web/templates/models.html`: the two `href="/settings/models"` reload links →
  `/focus/settings?section=models`.
- `web/templates/cards.html`: skills card `:309` title + `:318` footer
  `href="/settings/skills"` → `/focus/skills`.
- `web/templates/knowledge-focus.html:3`: update the comment (skills focus +
  settings no longer carries skills).

**Verify:** `go test ./internal/web/ -run 'TestUiCard|TestCard|TestHandlerHomePage|TestModels'` → ok;
`grep -rn 'href="/settings/\|href="/settings"\|href="/profile"\|href="/models"' web/templates` → none.
**Commit:** `git add web/templates && git commit -m "feat: settings/profile/models links point to the settings + skills focuses"`

### Step D: delete the pages

1. **Delete** `web/templates/settings.html` (body now in `settings-focus.html`).
2. **Remove routes** `GET /models`, `GET /settings`, `GET /settings/{section}`,
   `GET /profile` (`web.go`). KEEP every `/ui/*` write route.
3. **Delete handlers** `settingsPage`, `settingsRoot` (`settings.go`),
   `modelsPage` (`models.go`), `profilePage` (`profile.go`). KEEP `modelsData`,
   `buildProfileData`, all `/ui/profile/*` + `/ui/model/*` handlers, the section
   fragment templates, `skillsData`. Remove imports left unused by the deletions
   (e.g. `net/http` in `settings.go` if `settingsRoot` was its only user — check
   `go build`).
4. **Tests:** `TestSettingsPages` — its `/settings/{section}` render cases →
   `/focus/settings?section=…` (asserting the section bodies); its `/settings`,
   `/profile`, `/models` cases → retired-route 302 guards (some may already be
   guards). The skills section case → drop (skills tested via the skills card
   focus, plan 053). Keep the model/profile write-endpoint tests.

**Verify (all must hold):**
```
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
grep -rnE '"/settings|href="/settings|"/profile"|href="/profile|"/models"|href="/models|settingsPage|settingsRoot|modelsPage|profilePage|"settings\.html"' internal web --include='*.go' --include='*.html'
```
The grep returns nothing except retired-route 302 guard tests and the kept
`/focus/settings`/`/settings/models`-free results. `gofmt -l` empty. (Note:
`/ui/model/*` and `/ui/profile/*` are write routes, not matched by the patterns
above — they must remain.)

**Browser check (owner — no display here):** drop a `settings` card on a board →
tile → expand → Profile section (edit name, pick avatars — saves in place); click
Models → the model panel (active model, GGUF download/progress, providers — all
work); the dock's "Manage models" + "Your avatar" links open the settings focus;
topbar Settings opens the focus; `/settings`, `/settings/x`, `/profile`, `/models`
302 to `/boards`; the skills card still opens the skills manager.

**Commit:** `git add -A && git commit -m "feat(settings): retire /settings, /profile, /models into the settings card focus"`

### Step E: docs
Update the `056` row in `plans/readme.md` → DONE. Fix `/settings`/`/profile`/
`/models` refs in `DESIGN.md`, `README.md`, `internal/self/knowledge.md`.

**Commit:** `git add -A && git commit -m "docs: settings/profile/models are the settings card focus now; 056 done"`

## Test plan
- **Focus + tile** (`focus_test.go`): `/ui/cards/settings` renders; `/focus/settings`
  → profile section (`id="identity-card"`, no Skills nav); `?section=models` →
  models panel.
- **Write endpoints intact**: profile name/avatar + model provider/gguf tests pass
  (endpoints + fragment templates unchanged; their SSE patches still target the
  same ids that now live inside the settings focus).
- **Skills unaffected**: the skills card focus (plan 053) still renders; settings
  no longer includes skills.
- **Deletion safety**: Step D grep clean; `go test ./...` green.
- **Browser** (owner): Step D checklist.

## Done criteria
- [ ] `focusBodyHTML` dispatches `settings` → the shell (Profile + Models); a
      `section` param selects the section (default profile); others unchanged.
- [ ] New `settings` card: registry spec + `renderCardSettings` tile +
      `settingsFocusHTML` focus; `settings_body` in exactly one file
      (`settings-focus.html`); Skills NOT in the settings nav/body.
- [ ] `/settings`, `/settings/{section}`, `/models`, `/profile` routes +
      `settingsPage`/`settingsRoot`/`modelsPage`/`profilePage` deleted; ALL
      `/ui/*` write routes + handlers + section fragment templates + `modelsData`/
      `buildProfileData`/`skillsData` KEPT.
- [ ] All links re-pointed (topbar, dock model/profile links, models reload,
      skills card → `/focus/skills`); no `href="/settings…"`/`/profile`/`/models`
      remains.
- [ ] `settingsData.Skills` removed.
- [ ] Step D grep clean; `go test ./...`, vet, `gofmt -l` (empty), CGO-free build
      clean; `git diff --check` clean.
- [ ] `plans/readme.md` 056 → DONE; doc refs fixed.

## STOP conditions
- "redefinition of template settings_body" → defined in two files; keep one.
- A profile/model write endpoint's SSE patch no longer lands (the fragment id it
  targets isn't in the focus) → the moved `settings_body` must include the same
  section fragments (`profile_identity_card` etc. with their ids); restore them.
- Deleting `modelsPage`/`profilePage`/`settingsRoot` orphans an import (`net/http`)
  → remove it; on any other compile error re-check the keep/delete split.
- The skills card focus breaks (it shares `skillsData`/`knowledge_body`) → those
  are KEPT; only the settings *inclusion* of skills is dropped.
- The Step D grep finds a `/settings|/profile|/models` page reference not in
  Current state → STOP, list, re-point or remove.

## Maintenance notes
- The model section keeps its SSE-driven elements (`gguf_progress` self-polling
  `/ui/model/gguf/progress`, provider forms) — they work inside the focus because
  their target ids are unchanged. Opening `/focus/settings?section=models` while a
  download runs shows live progress as before.
- `focusBodyHTML` cases now: quests, journal, day, memory, skills, lifelog,
  settings. Phase 7 (Cleanup) is the finale — topbar/nav, dead code, DESIGN.md.
