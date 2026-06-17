# Plan 092: Settings is summoned as per-section, nav-free artifacts (no in-artifact tabs)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md`.
>
> **Drift check (run first)**:
> `git diff --stat 766b7aa..HEAD -- internal/feature/settingscards/ internal/web/home.go internal/web/handlers_test.go internal/web/assets/static/basm.css internal/feature/storybook/stories_cards.go internal/feature/storybook/stories_settings.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.
>
> **Sandbox note**: in a TLS-intercepting sandbox (Hyperagent), Go commands
> need the GOPROXY shim — see `docs/hyperagent-sandbox.md`. GOSUMDB stays on.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW–MED
- **Depends on**: none (independent of 093 and 094; all three can land in any order)
- **Category**: direction (UX) / tech-debt
- **Planned at**: commit `766b7aa`, 2026-06-17

## Why this matters

When the owner summons **Settings** into the chat (sidebar click or the agent's
`card_show`), it renders the full `settings-layout` with a `settings-nav` tab
bar (Profile / Models / Heads / Appearance). That tab bar is *navigation inside
an artifact that already lives inside the chat* — two navigation systems
competing, which the owner has flagged as confusing. The decision (confirmed
2026-06-17): **navigation lives in the sidebar, not in the artifact.** Each
settings section becomes its own small, nav-free artifact, summoned from a
**Settings** sub-menu in the sidebar.

The enabling fact: **after plan 089 retired the `/focus/{type}` pages,
`ui.Focus` is rendered in exactly one place — the in-chat artifact**
(`cardFocusHTML` / `uicardBody` in `internal/web/cards.go`). So dropping the
settings nav wrapper changes nothing but the artifact. And `BuildSettingsFocus`
**already builds one section at a time** from the `section` param, and the
settings tile **already links** to `?section=…` — so this is a renderer change
plus a sidebar change, **not new card types**.

## Current state

Files and their roles:

- `internal/feature/settingscards/settingsfocus.go` — builds + renders the
  settings focus body. `SettingsFocus` (the thing to denav) is at lines 340–387.
- `internal/feature/settingscards/settings.go` — `registerSettings`
  (lines 32–44) wires the card into the `ui` registry; `SettingsCard` (the
  tile, lines 16–30) already links to `?section=…`.
- `internal/web/home.go` — `domainSidebar()` (lines 43–85) builds the product
  sidebar; the single `Settings` item is at line 71.
- `internal/web/cards.go` — `cardFocusHTML` / `uicardBody` (lines 128–159): the
  ONLY caller of `ui.Focus` (the in-chat artifact). Read its comment; do not
  change this file.
- `internal/web/handlers_test.go` — asserts the settings artifact markup
  (lines ~118–130). Must be updated.
- `internal/web/assets/static/basm.css` — `.settings-layout` (line 1867),
  `.settings-nav` (1879), `.settings-nav-link` (1888), `.settings-nav-active`
  (1907). Orphaned after this plan.
- `internal/feature/storybook/stories_cards.go` — renders the card focuses for
  the storybook (grep `SettingsFocus` to find the settings story).

### `SettingsFocus` today (`settingsfocus.go:340-387`) — the nav to remove

```go
// SettingsFocus renders the full settings focus body. Ports {{define
// "settings_body"}} ... : the settings nav (Profile / Models / Heads /
// Appearance tabs) and the section content.
func SettingsFocus(v SettingsFocusView) g.Node {
	var content g.Node
	switch v.Section {
	case "models":
		content = modelcards.Panel(v.Models)
	case "heads":
		content = headscards.HeadsCard(v.Heads)
	case "appearance":
		content = AppearanceSection()
	default:
		content = g.Group([]g.Node{
			ProfileIdentityCard(v.Profile),
			ProfileSoulSection(v.Profile),
			ProfileBalaurSection(v.Profile),
		})
	}

	return Div(Class("settings-layout"),
		Nav(
			Class("settings-nav"),
			g.Attr("aria-label", "Settings sections"),
			settingsNavLink(v.Section, "profile", "Profile"),
			settingsNavLink(v.Section, "models", "Models"),
			settingsNavLink(v.Section, "heads", "Heads"),
			settingsNavLink(v.Section, "appearance", "Appearance"),
		),
		Div(Class("settings-content"), content),
	)
}

// settingsNavLink renders one settings-nav tab. ...
func settingsNavLink(active, section, label string) g.Node {
	cls := "settings-nav-link"
	if active == section {
		cls += " settings-nav-active"
	}
	href := "/ui/show/settings?section=" + section
	return A(
		Class(cls),
		Href(href),
		data.On("click", "@get('"+href+"')", data.ModifierPrevent),
		g.Text(label),
	)
}
```

The `switch v.Section` above is exactly the per-section content we keep; only
the `Div(Class("settings-layout"), Nav(...settings-nav...), ...)` wrapper and
`settingsNavLink` are removed.

### `domainSidebar()` today (`home.go:47-85`) — the sidebar to extend

```go
func domainSidebar() shell.SidebarProps {
	item := func(label, typ, icon string) shell.SidebarItem {
		href := "/ui/show/" + typ
		return shell.SidebarItem{
			Label:  label,
			Href:   href,
			Icon:   icon,
			Action: "@get('" + href + "')",
		}
	}
	return shell.SidebarProps{
		Brand: g.Group([]g.Node{ /* crest + name */ }),
		Sections: []shell.SidebarSection{
			{Label: "Domains", Items: []shell.SidebarItem{
				item("Quests", "quests", "scroll"),
				item("Knowledge", "memory", "tome"),
				item("Life", "lifelog", "orb"),
				item("Journal", "journal", "quill"),
				item("Heads", "heads", "shield"),
				item("Settings", "settings", "key"),
			}},
		},
		Footer: g.Group([]g.Node{ /* theme toggle + Home link */ }),
	}
}
```

`shell.SidebarSection{Label, Items}` and `shell.SidebarItem{Label, Href, Icon,
Action, …}` are defined in `internal/ui/shell/sidebar.go`. A second section is
rendered as a second labelled group (label + count + rule) — see
`Sidebar` in that file. There is no collapsible-disclosure mechanism; a
labelled section is the right primitive (a `▾` disclosure is a deferred
enhancement, see Maintenance notes).

### Repo conventions to match

- gomponents components live in feature packages; render with `.Render(w)`.
  Match the import style already in `settingsfocus.go` (`g "maragu.dev/gomponents"`,
  dot-import `. "maragu.dev/gomponents/html"`, `data "maragu.dev/gomponents-datastar"`).
- The sidebar item Action is a Datastar `@get('…')` expression; the `Href` is
  the no-JS fallback. `/ui/show/{type}` honors query params (see `uiShow` in
  `internal/web/show.go` → `queryToMap(e.Request.URL.Query())`), so
  `/ui/show/settings?section=models` is valid for both `@get` and the href.
- Tests use PocketBase `tests.ApiScenario` (see `internal/web/handlers_test.go`).
  `ExpectedContent` / `NotExpectedContent` are substring checks against the
  response body; Datastar attribute single-quotes render HTML-escaped as
  `&#39;` (see the existing `@post(&#39;/ui/profile/name&#39;` assertion).

## Commands you will need

| Purpose   | Command                                            | Expected on success |
|-----------|----------------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                     | exit 0              |
| Vet       | `go vet ./...`                                     | exit 0              |
| Test (pkg)| `go test ./internal/web/... ./internal/feature/...`| all pass            |
| Test (all)| `go test ./...`                                    | all pass            |
| Format    | `gofmt -l internal/`                               | no output           |
| Diff check| `git diff --check`                                 | no output           |

## Scope

**In scope** (the only files you should modify):
- `internal/feature/settingscards/settingsfocus.go` — denav `SettingsFocus`, delete `settingsNavLink`.
- `internal/feature/settingscards/settings_test.go` and/or `settingsfocus_test.go` — update assertions (whichever asserts the nav/layout; grep them).
- `internal/web/home.go` — sidebar: split into Domains + Settings sub-menu.
- `internal/web/handlers_test.go` — update the settings-artifact assertions.
- `internal/web/assets/static/basm.css` — remove the orphaned `.settings-nav*` / `.settings-layout` rules; optionally add a minimal `.settings-section` container.
- `internal/feature/storybook/stories_cards.go` (and `stories_settings.go` only if it renders `SettingsFocus`) — update the settings story to the nav-free render.

**Out of scope** (do NOT touch, even though they look related):
- `internal/web/cards.go` — the `ui.Focus` dispatch is correct; leave it.
- `internal/web/show.go` — the injection door is correct.
- `internal/web/profile.go`, `internal/feature/modelcards/`,
  `internal/feature/headscards/` — the per-section content + its re-render
  handlers are reused unchanged. The profile write handlers patch
  `#identity-card` / `#soul-section` / `#balaur-section`, which live *inside*
  the section content and are unaffected by removing the outer nav.
- `BuildSettingsFocus` — it already does per-section building; do not change it.
- Plans 093 / 094 surfaces (day, quests, the artifact cap).

## Git workflow

- Branch: `improve/092-settings-per-section-artifacts`.
- Commit per step or per logical unit; conventional-commit style, e.g.
  `refactor(web): settings artifact renders one nav-free section`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Make `SettingsFocus` render one section, nav-free

In `settingsfocus.go`, change `SettingsFocus` so it returns the per-section
`content` it already computes, wrapped in a single light container — and drop
the `settings-layout` wrapper and the `settings-nav` `Nav(...)`. Target shape:

```go
func SettingsFocus(v SettingsFocusView) g.Node {
	var content g.Node
	switch v.Section {
	case "models":
		content = modelcards.Panel(v.Models)
	case "heads":
		content = headscards.HeadsCard(v.Heads)
	case "appearance":
		content = AppearanceSection()
	default:
		content = g.Group([]g.Node{
			ProfileIdentityCard(v.Profile),
			ProfileSoulSection(v.Profile),
			ProfileBalaurSection(v.Profile),
		})
	}
	// One nav-free section. Navigation lives in the sidebar (plan 092);
	// each settings section is summoned as its own artifact.
	return Div(Class("settings-section"), content)
}
```

Then **delete `settingsNavLink`** (lines ~375–387) — it is now unused. Confirm
the `data` import is still used elsewhere in the file (it is — the profile
forms use `data.On`); do not remove it.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 2: Split the sidebar into Domains + a Settings sub-menu

In `home.go` `domainSidebar()`, add a helper for section links and replace the
single `Settings` domain item with a dedicated **Settings** section. Drop the
top-level `Heads` and `Settings` items from `Domains` (Heads moves under
Settings, matching the owner's chosen IA). Target shape:

```go
	item := func(label, typ, icon string) shell.SidebarItem {
		href := "/ui/show/" + typ
		return shell.SidebarItem{Label: label, Href: href, Icon: icon, Action: "@get('" + href + "')"}
	}
	// sect summons one settings section as its own nav-free artifact.
	sect := func(label, section string) shell.SidebarItem {
		href := "/ui/show/settings?section=" + section
		return shell.SidebarItem{Label: label, Href: href, Action: "@get('" + href + "')"}
	}
	return shell.SidebarProps{
		Brand: /* unchanged */,
		Sections: []shell.SidebarSection{
			{Label: "Domains", Items: []shell.SidebarItem{
				item("Quests", "quests", "scroll"),
				item("Knowledge", "memory", "tome"),
				item("Life", "lifelog", "orb"),
				item("Journal", "journal", "quill"),
			}},
			{Label: "Settings", Items: []shell.SidebarItem{
				sect("Profile", "profile"),
				sect("Appearance", "appearance"),
				sect("Models", "models"),
				sect("Heads", "heads"),
			}},
		},
		Footer: /* unchanged */,
	}
```

(Settings items are intentionally icon-less to read as sub-items; the existing
icon stems are scroll/tome/orb/quill/shield/key only — do not invent stems.)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Remove the orphaned settings-nav CSS

In `basm.css`, delete the now-unused rules: `.settings-layout` (and its
`@media` block at ~1873), `.settings-nav`, `.settings-nav-link`,
`.settings-nav-link:hover`, `.settings-nav-active` (lines ~1867–1910). If the
profile/section content needs a wrapper rule, add a single minimal
`.settings-section { display: flex; flex-direction: column; gap: var(--space-4); }`
near the deleted block (only if visual spacing regresses — otherwise omit; the
inner cards already space themselves).

**Verify**: `grep -rn "settings-nav\|settings-layout" internal/` → returns
nothing (no Go reference, no CSS rule). `grep -rn "settingsNavLink" internal/`
→ nothing.

### Step 4: Update the storybook settings story

In `stories_cards.go` (grep `SettingsFocus`), the settings story renders the
focus body. After Step 1 it renders one nav-free section — update the story's
description/notes to say so and drop any "tabs/nav" do-or-don't copy. If the
story passed a fixture with no `Section`, it now shows the Profile section
(identity + avatars) with no tab bar — that is correct; keep it.

**Verify**: `go test ./internal/feature/storybook/...` → all pass.

### Step 5: Update the settings tests

1) `internal/web/handlers_test.go` (~lines 118–130): the `/ui/show/settings`
artifact test currently asserts `class="settings-layout"`. Change it to assert
the **nav-free profile section** and to **reject** the nav:

```go
ExpectedContent: []string{
	`id="identity-card"`,               // the Profile section's identity form
	`@post(&#39;/ui/profile/name&#39;`, // a working write form is present
},
NotExpectedContent: []string{
	`settings-layout`, `settings-nav`,  // no in-artifact navigation (plan 092)
},
```

Add a second sub-test that `/ui/show/settings?section=models` renders the
models panel without the nav. Pick a stable substring from `modelcards.Panel`
markup (read it; e.g. a class like `model-panel` or a heading) and assert it
plus `NotExpectedContent: []string{"settings-nav"}`.

2) `internal/feature/settingscards/settings_test.go` / `settingsfocus_test.go`:
grep for `settings-layout` / `settings-nav` / `settingsNavLink` and update any
assertion to the nav-free shape (assert `settings-section` or the section's own
markup; reject `settings-nav`). Do not weaken coverage — keep asserting the
profile identity form / models panel content.

**Verify**: `go test ./internal/web/... ./internal/feature/...` → all pass.

### Step 6: Full gates

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `gofmt -l internal/` → no output
- `git diff --check` → no output

## Test plan

- Update `internal/web/handlers_test.go`: settings artifact asserts the
  profile section + write form, **rejects** `settings-layout`/`settings-nav`;
  new sub-test for `?section=models`.
- Update `internal/feature/settingscards/*_test.go` to the nav-free render.
- Pattern to follow: the existing `tests.ApiScenario` blocks in
  `handlers_test.go` (note `ExpectedContent` + `NotExpectedContent`).
- Verification: `go test ./...` → all pass, including the updated settings
  tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l internal/` prints nothing
- [ ] `grep -rn "settings-nav\|settings-layout\|settingsNavLink" internal/` returns nothing
- [ ] `internal/web/home.go` has a `Settings` sidebar section with Profile/Appearance/Models/Heads (`grep -n 'Label: "Settings"' internal/web/home.go`)
- [ ] `git status` shows only in-scope files modified
- [ ] `plans/readme.md` status row for 092 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The `SettingsFocus` / `domainSidebar` code at the locations above does not
  match the excerpts (the codebase drifted since `766b7aa`).
- `ui.Focus` turns out to be rendered somewhere OTHER than
  `cardFocusHTML`/`uicardBody` (grep `ui.Focus` across `internal/`; if a page
  handler or board still uses it, removing the nav would break that surface —
  STOP, the 089 retirement assumption is false).
- Removing `settingsNavLink` makes the `data` import unused (it should not —
  the profile forms use `data.On`; if the build complains, STOP and report).
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- The `settings` card now renders ONE section per artifact, chosen by the
  `section` param (default `profile`). The agent's `card_show` with
  `{type:"settings", params:{section:"models"}}` and the sidebar both reach it.
- Persisted history: old `settings` artifact rows (pre-092, no `section`) now
  re-render as the Profile section on reload — acceptable (history re-renders
  live from the registry, documented behavior).
- Deferred enhancement: a true collapsible `▾` disclosure for the Settings
  section (the owner's mock showed one). The labelled section is the KISS
  version; add disclosure JS later if desired.
- Reviewer should scrutinize: that the profile/models re-render handlers
  (`#identity-card`, `#soul-section`, `#balaur-section`, the models panel
  patch ids) still match — they patch inside the section content, so removing
  the outer nav must not have changed those ids.
- Sibling plans 093 (denav day & quests) and 094 (cap active artifacts) apply
  the same "no nav in the artifact" principle to other surfaces and are
  independent of this one.
