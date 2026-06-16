# 05 — Port the Settings focus body (Profile + Models) to gomponents

> **Read `plans/ui-domain-pages/README.md` first** (shared recipe, conventions,
> conflict map, verification). **Exemplar**:
> `internal/feature/lifecards/lifelogfocus.go` + `internal/web/{cards.go,focus.go}`.
> Stamped against `884d692`.

## ⚠️ STATUS: BLOCKED on the Ollama→Kronk migration — do not start until it lands

`/focus/settings` is a two-section body (Profile + Models) rendered by **one**
shared `settings_body` define (`web/templates/settings-focus.html`). The **Models
section is being rewritten right now** by the parallel `improve/ollama-to-kronk`
branch: it removes the OpenAI-provider routes/handlers and introduces gomponents
`internal/feature/modelcards/{ModelCard,Panel}` + a `modelsPageData.ModelsHTML`
field that `settings_body` injects. Touching the Models section — or even the
shared `settings_body` that holds both sections — now would collide with that
work.

**Profile alone is not cleanly separable:** the Profile and Models branches share
`settings_body`, so porting Profile-only still means editing a file the Kronk
branch edits. Therefore: **wait until `improve/ollama-to-kronk` merges to the base
branch** (the Models gomponents `modelcards.Panel` will then exist on the base),
then do the full settings port below in one pass. Pick up the clean bodies
(01–04) first; revisit this last.

> When you start, re-confirm the block is cleared: `git log --oneline -5
> origin/<base-branch> -- internal/feature/modelcards web/templates/models.html`
> — if `modelcards.Panel` and the Kronk model handlers are on the base branch,
> you're unblocked.

## Context / why

Finish the redesign: port the settings focus body to a gomponents component so
`/focus/settings` matches the rest of the Domain Navigator (the lifelog/quests/…
bodies were ported the same way). Profile is a Balaur-domain surface (clean);
Models is the Kronk model UI (port via the components the Kronk branch built).

## Current state (read these)

`web/templates/settings-focus.html` — `settings_body`:
`<div class="settings-layout"><nav class="settings-nav">` two
`<a class="settings-nav-link[ settings-nav-active]" href="/focus/settings?section=profile|models" data-on:click__prevent="@get('/focus/settings?section=…')">Profile|Models</a>`
`</nav><div class="settings-content">{if eq .Section "models"}{models_panel .Models}{else}{profile_identity_card .Profile}{profile_soul_section .Profile}{profile_balaur_section .Profile}{end}</div></div>`.

`web/templates/profile.html` — three fragment defines, each its own
re-render target after its `@post`:

- `profile_identity_card` (`#identity-card`): `<article class="profile-card"
  id="identity-card"><h2 class="profile-card-title">Identity</h2><p
  class="profile-hint">…</p><form class="profile-name-form"
  data-on:submit__prevent="@post('/ui/profile/name', {contentType:'form'})"><label
  for="display_name">Your name</label><div class="profile-name-row"><input
  id="display_name" name="display_name" type="text" value="…" placeholder="…"
  autocomplete="off" maxlength="60"><button class="btn btn-primary"
  type="submit">Save</button></div>{if .SavedName}<p class="profile-saved">◈
  Saved.</p>{end}</form></article>`.
- `profile_soul_section` (`#soul-section`): `<article class="profile-card"
  id="soul-section">…<div class="avatar-choice-list profile-avatar-grid">{range
  .AvatarOptions}<form data-on:submit__prevent="@post('/ui/profile/soul-avatar',
  {contentType:'form'})"><input type="hidden" name="soul_avatar" value="{Key}"><button
  class="avatar-choice profile-avatar-btn[ avatar-choice-active]" type="submit"{if
  .Active} aria-current="true" disabled{end}><img class="px" src="{URL}" alt=""
  decoding="async"><span>{Label}</span></button></form>{end}</div></article>`.
- `profile_balaur_section` (`#balaur-section`): identical shape, `@post('/ui/profile/balaur-avatar')`, hidden `balaur_avatar`, ranges `.BalaurOptions`.

Handlers (`internal/web/profile.go`, read for line numbers): `buildProfileData`
(view `profileData{OwnerName string; SavedName bool; AvatarOptions, BalaurOptions
[]AvatarOption}`, `AvatarOption{Key, Label, URL string; Active bool}`), `saveName`
(POST `/ui/profile/name` → `#identity-card` **outer**), `setSoulAvatarFromProfile`
(POST `/ui/profile/soul-avatar` → `#soul-section` **outer**), `setBalaurAvatarPref`
(POST `/ui/profile/balaur-avatar` → `#balaur-section` **outer**). The settings
shell is `settingsFocusHTML(params)` → `settingsData{Section string; Profile
profileData; Models modelsPageData}` → `settings_body`. The Models section is
`modelcards.Panel` (Kronk branch) via `modelsPageData.ModelsHTML`.

The settings card lives in `internal/feature/settingscards/`.

## Action contract — preserve byte-for-byte

| Trigger | Endpoint | Fields | SSE target & mode |
|---|---|---|---|
| nav tab | `@get('/focus/settings?section=profile\|models')` (`__prevent` + href) | `section` query | `#main` **inner** (focusPage) |
| save name | `@post('/ui/profile/name', {contentType:'form'})` | `display_name` (≤60) | `#identity-card` **outer** |
| pick soul avatar | `@post('/ui/profile/soul-avatar', {contentType:'form'})` | hidden `soul_avatar`={Key} | `#soul-section` **outer** |
| pick head avatar | `@post('/ui/profile/balaur-avatar', {contentType:'form'})` | hidden `balaur_avatar`={Key} | `#balaur-section` **outer** |

Load-bearing ids: `#identity-card`, `#soul-section`, `#balaur-section`. Classes:
`settings-layout`, `settings-nav`, `settings-nav-link`, `settings-nav-active`,
`settings-content`, `profile-card`, `profile-card-title`, `profile-hint`,
`profile-name-form`, `profile-name-row`, `profile-saved`, `avatar-choice-list`,
`profile-avatar-grid`, `avatar-choice`, `profile-avatar-btn`,
`avatar-choice-active`, `px`. The avatar grid is a **form-per-button** with a
hidden input — hand-emit it (do **not** swap in `ui.Avatar`, which emits the
`balaur-avatar` structure, not this `avatar-choice` button).

## Scope (when unblocked)

**In scope:** `internal/feature/settingscards/` (new `settingsfocus.go` with the
Profile components + a `SettingsFocus` body that composes the nav + Profile
components + the Models panel) + the `registerSettings` size dispatch,
`internal/web/profile.go` (point `saveName`/`setSoul…`/`setBalaur…` at the Profile
components), `internal/web/settings.go`/`models.go` (have `settingsFocusHTML`
render `SettingsFocus`; the Models panel is `modelcards.Panel`), `focus.go` (drop
`case "settings"`), `web/templates/{settings-focus.html,profile.html}` (retire
defines once unused), storybook.

**Out of scope:** `modelcards.*` internals (built by Kronk — reuse `modelcards.Panel`),
the `/ui/model/*` handlers, README conflict files.

## Steps (when unblocked)

1. `internal/feature/settingscards/settingsfocus.go`: port the three Profile
   defines to components `ProfileIdentityCard(view)`, `ProfileSoulSection(view)`,
   `ProfileBalaurSection(view)` (view-models mirroring `profileData` /
   `AvatarOption`), preserving every id/class + the `@post` contracts above.
2. `SettingsFocus(section string, profile …, modelsPanel g.Node) g.Node` — port
   `settings_body`: the `settings-layout` + `settings-nav` (the two tabs with
   `settings-nav-active` on the current section + the `@get` nav) + the
   `settings-content` dispatch (models → `modelsPanel` (a `modelcards.Panel`);
   profile → the three Profile components).
3. `registerSettings`: `ui.Focus` → `SettingsFocus(…)` for the section in
   `params["section"]` (default profile), else the existing tile.
4. `internal/web/profile.go`: `saveName`/`setSoulAvatarFromProfile`/
   `setBalaurAvatarPref` render the new Profile components for their `#identity-card`/
   `#soul-section`/`#balaur-section` **outer** patches (so initial + re-render match).
5. `internal/web/focus.go`: delete `case "settings": return h.settingsFocusHTML(params)`.
6. `settingsFocusHTML`/`modelsPanel` wiring: render `SettingsFocus` with the
   `modelcards.Panel` for the Models section. Keep the `models-panel` SSE target
   the Kronk model handlers patch.
7. Retire the `settings_body`/`profile_*` defines after the handlers no longer
   execute them; grep `*_test.go` first (`TestFocusSettingsProfile`/`…Models`,
   `TestModelsPageAndCleanChatbarRender` reference `settings_body`/`models_panel`
   — keep them green via the ported body; leave any direct-`ExecuteTemplate` test
   as dead code + TODO).
8. Storybook `settingsfocusStory()` (Profile variant + Models variant) +
   register; + `settingsfocus_test.go` asserting the Profile contract
   (`#identity-card`, the three `@post` endpoints, `avatar-choice-active`) and the
   nav tabs.

## Done criteria (when unblocked)

- `CGO_ENABLED=0 go build ./...` → 0; targeted + storybook tests ok; `gofmt -l`
  empty; `git diff --check` clean.
- `internal/web/focus_test.go::TestFocusSettingsProfile` and
  `TestFocusSettingsModels` still pass (`#identity-card`, `settings-nav`,
  `#models-panel`).
- Live: `curl -s 127.0.0.1:PORT/focus/settings` (profile) contains `#identity-card`,
  `#soul-section`, `#balaur-section`, the three `@post('/ui/profile/…'`, the nav,
  topbar + `id="dock"`; `?section=models` renders the `modelcards.Panel`. Save a
  name and pick an avatar manually.

## Maintenance / escape hatches

- If, when you start, `modelcards.Panel` is **not** on the base branch yet, the
  block is not cleared — STOP and wait (or do only the Profile components +
  handlers, leaving `settings_body`'s models branch untouched, and explicitly
  note the half-port). Do not edit `models.html`/`modelsPageData`.
- Avatar grid stays a form-per-button (the click posts + re-renders the whole
  section); don't convert it to a single form or `ui.Avatar`.
- README conflict files → STOP.
</content>
