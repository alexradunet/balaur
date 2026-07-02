# Plan 255: Surface export and encrypted backup in the Settings UI

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/feature/settingscards internal/web/web.go internal/web/settings_backup.go internal/web/settings_backup_test.go internal/cards/cards.go internal/feature/storybook/stories_cards.go internal/feature/storybook/story.go internal/self/knowledge.md README.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: none
- **Category**: direction
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

Balaur's sovereignty pillar — the one-way Johnny Decimal Markdown mirror and
the passphrase-encrypted off-box backup — ships and is leak-tested, but ONLY
as CLI verbs (`balaur export`, `balaur export --encrypt`) with the passphrase
in an environment variable. `PRODUCT.md:81-82` defines the owner test as
"own it — their conversation, tasks, journal, and life-log are all theirs in
SQLite, and they have opened the file at least once to prove it", and
`PRODUCT.md:87-88` states "The closing test of the north star, still ahead of
us: a non-technical owner gets from nothing to a working companion **without a
terminal**." Today that owner cannot make a backup at all. This plan adds a
"Backup" section to the existing Settings focus view with two owner-clickable
actions: write the Markdown mirror, and write a passphrase-encrypted backup
archive — both backed by the already-shipped `internal/export` functions, both
audited, with the passphrase typed once into a form and never stored, logged,
or echoed back. Restore stays CLI-only (deliberately deferred — see
Maintenance notes).

## Current state

### The export/backup capability that already exists (reuse, do not modify)

- `internal/export/export.go` — the mirror writer. Signature at
  `internal/export/export.go:109`:

  ```go
  func ExportMirror(app core.App, destDir string) ([]string, error) {
  ```

  It writes one Markdown file per active owner-authored node under `destDir`,
  commits the tree to a local git history (skipped cleanly when git is
  absent), and returns the sorted relative file paths written.

- `internal/export/encrypt.go` — the encrypted envelope.
  `internal/export/encrypt.go:57`:

  ```go
  func EncryptDir(srcDir, destFile, passphrase string) error {
      if passphrase == "" {
          return fmt.Errorf("export: empty passphrase")
      }
  ```

  and `internal/export/encrypt.go:106`:

  ```go
  func DecryptDir(srcFile, destDir, passphrase string) error {
  ```

  `EncryptDir` writes a single file to `destFile` (the parent directory must
  exist — the handler creates it). There is no enforced archive filename
  convention anywhere in the repo (the CLI takes an arbitrary `--archive`
  path); this plan picks `<data dir>/backup/balaur-backup-<yyyymmdd-hhmmss>.enc`.

- `internal/cli/export.go` — the CLI-only surface this plan mirrors into the
  UI. The passphrase env var at `internal/cli/export.go:17`:

  ```go
  const passphraseEnv = "BALAUR_EXPORT_PASSPHRASE"
  ```

  The temp-dir + wipe shape and the warning the UI copy must carry, at
  `internal/cli/export.go:68-75`:

  ```go
  	fmt.Fprintln(cmd.ErrOrStderr(),
  		"WARNING: if you lose this passphrase, this backup is UNRECOVERABLE — there is no recovery, no escrow, no cloud.")

  	tmp, err := os.MkdirTemp("", "balaur-export-*")
  	if err != nil {
  		return nil, fmt.Errorf("export: temp dir: %w", err)
  	}
  	defer os.RemoveAll(tmp)
  ```

  and the plain-mirror default dest at `internal/cli/export.go:43-46`:

  ```go
  		dest := out
  		if dest == "" {
  			dest = filepath.Join(app.DataDir(), "export")
  		}
  		files, err := export.ExportMirror(app, dest)
  ```

### The settings surface to extend

- `internal/feature/settingscards/settingsfocus.go` — aggregate dispatcher.
  The view-model at `settingsfocus.go:21-28`:

  ```go
  type SettingsFocusView struct {
  	Section      string               // "profile" | "models" | "heads" | "nudges" | "capabilities"
  	Profile      ProfileView          // used when Section == "profile"
  	Models       modelcards.PanelView // used when Section == "models"
  	Heads        headscards.HeadsView // used when Section == "heads"
  	Nudge        NudgeView            // used when Section == "nudges"
  	Capabilities CapabilitiesView     // used when Section == "capabilities"
  }
  ```

  The section allow-list in `BuildSettingsFocus` at `settingsfocus.go:34-39`:

  ```go
  	switch section {
  	case "models", "heads", "nudges", "capabilities":
  		// known sections
  	default:
  		section = "profile"
  	}
  ```

  The render dispatch in `SettingsFocus` at `settingsfocus.go:65-79`:

  ```go
  	switch v.Section {
  	case "models":
  		content = modelcards.Panel(v.Models)
  	case "heads":
  		content = headscards.HeadsCard(v.Heads)
  	case "nudges":
  		content = NudgeSection(v.Nudge)
  	case "capabilities":
  		content = CapabilitiesSection(v.Capabilities)
  	default:
  ```

- `internal/feature/settingscards/settingsfocus_nudges.go` — the exemplar for
  a per-section file (view-model struct + render func in one file, Datastar
  form posts). Its form idiom at `settingsfocus_nudges.go:38-40`:

  ```go
  	post := func(url string) g.Node {
  		return data.On("submit", "@post('"+url+"', {contentType:'form'})", data.ModifierPrevent)
  	}
  ```

  and its section root at `settingsfocus_nudges.go:57`:

  ```go
  	return h.Article(h.Class("profile-card"), h.ID("nudge-section"),
  ```

- `internal/feature/settingscards/settingsfocus_capabilities.go` — the closest
  exemplar of a POST-backed settings control with a secret-ish input.
  `MessengerGatewaySection` at `settingsfocus_capabilities.go:158-173`:

  ```go
  		h.Form(
  			h.Class("profile-name-form"),
  			data.On("submit", "@post('/ui/settings/messenger-token', {contentType:'form'})", data.ModifierPrevent),
  			h.Label(h.For("messenger_token"), g.Text("Token")),
  			h.Div(h.Class("profile-name-row"),
  				h.Input(
  					h.ID("messenger_token"),
  					h.Name("messenger_token"),
  					h.Type("text"),
  					h.Value(v.MessengerToken),
  					h.Placeholder("set a secret token…"),
  					g.Attr("autocomplete", "off"),
  				),
  				h.Button(h.Class("btn btn-primary"), h.Type("submit"), g.Text("Save")),
  			),
  		),
  ```

- `internal/feature/settingscards/settings.go` — the settings tile with the
  owner-clickable section links, at `settings.go:27-32`:

  ```go
  		h.Ul(h.Class("ucard-stats"),
  			h.Li(h.A(h.Href("/ui/show/settings?section=profile"), g.Attr("data-on:click__prevent", "@get('/ui/show/settings?section=profile'); basmOpenPanel()"), g.Text("Profile"))),
  			h.Li(h.A(h.Href("/ui/show/settings?section=models"), g.Attr("data-on:click__prevent", "@get('/ui/show/settings?section=models'); basmOpenPanel()"), g.Text("Models & APIs"))),
  			h.Li(h.A(h.Href("/ui/show/settings?section=heads"), g.Attr("data-on:click__prevent", "@get('/ui/show/settings?section=heads'); basmOpenPanel()"), g.Text("Heads"))),
  			h.Li(h.A(h.Href("/ui/show/settings?section=appearance"), g.Attr("data-on:click__prevent", "@get('/ui/show/settings?section=appearance'); basmOpenPanel()"), g.Text("Appearance"))),
  		),
  ```

- `internal/cards/cards.go` — the typed card-spec registry. The settings
  section enum at `cards.go:242` (inside the `Type: "settings"` spec starting
  at `cards.go:235`):

  ```go
  				{Name: "section", Enum: []string{"profile", "models", "heads", "appearance", "capabilities", "nudges"}, Doc: "settings section (default profile)"},
  ```

  `cards.Validate` (at `cards.go:366`) REJECTS enum values not in this list
  with a 400 — so `/ui/show/settings?section=backup` fails until `"backup"`
  is added here.

### The web gateway pattern to follow

- `internal/web/messenger_settings.go` — the whole file is the handler
  exemplar (`messenger_settings.go:16-30`):

  ```go
  func (h *handlers) saveMessengerToken(e *core.RequestEvent) error {
  	token := strings.TrimSpace(e.Request.FormValue("messenger_token"))
  	// Never log the token value — it is a secret.
  	if err := store.SetOwnerSetting(h.app, "messenger_token", token); err != nil {
  		return e.InternalServerError("saving messenger token", err)
  	}
  	view := settingscards.BuildCapabilities(h.app)
  	var b strings.Builder
  	if err := settingscards.MessengerGatewaySection(view).Render(&b); err != nil {
  		return e.InternalServerError("rendering messenger gateway section", err)
  	}
  	sse := datastar.NewSSE(e.Response, e.Request)
  	patchOuterHTML(sse, "messenger-gateway-section", b.String())
  	return nil
  }
  ```

- `internal/web/web.go` — the route block to extend, at `web.go:226-227`:

  ```go
  	// Settings writes: capabilities / messenger token.
  	se.Router.POST("/ui/settings/messenger-token", h.saveMessengerToken)
  ```

- `internal/web/toast.go:15-20` — the toast helper:

  ```go
  func emitToast(sse *datastar.ServerSentEventGenerator, tone, msg string) {
  	_ = sse.PatchElements(
  		renderNodeHTML(ui.Toast(ui.ToastProps{Tone: tone}, g.Text(msg))),
  		datastar.WithSelectorID("toast-region"), datastar.WithModeAppend(),
  	)
  }
  ```

  Usage exemplar: `internal/web/tasks.go:92` — `emitToast(sse, "success", "Marked done.")`.

- `internal/store/audit.go:14` — the audit helper (call strictly AFTER the
  successful write, never before):

  ```go
  func Audit(app core.App, actor, action, target string, allowed bool, detail map[string]any) {
  ```

### Test helpers that exist

- `internal/web/handlers_test.go:29` — `func newWebApp(t testing.TB) *tests.TestApp`:
  builds a temp-dir PocketBase test app with the web routes bound on
  `OnServe` and all feature card renderers registered.
- `internal/web/review_test.go:54-72` — `serveReviewRoute(t, app, url)`: builds
  the real router via `apis.NewRouter` + `app.OnServe().Trigger` + `BuildMux`,
  then serves one `httptest` POST (nil body). Model the new form-post helper
  on it (add a form body + headers).
- `internal/web/knowledge_test.go:46-49` — the Datastar form-post headers:

  ```go
  var sseHeaders = map[string]string{
  	"Content-Type": "application/x-www-form-urlencoded",
  	"Accept":       "text/event-stream",
  }
  ```

- `internal/cli/restore_test.go:16-20` — how to seed an exportable node:

  ```go
  	app := storetest.NewApp(t)
  	if _, err := nodes.Create(app, "note", "Restore Note", "Body text.",
  		nodes.StatusActive, nil); err != nil {
  		t.Fatalf("create: %v", err)
  	}
  ```

  (import `"github.com/alexradunet/balaur/internal/nodes"`; in `internal/web`
  tests use `newWebApp(t)` instead of `storetest.NewApp`).
- `internal/feature/settingscards/settingsfocus_test.go:14-16` — component
  tests render via `uitest.Render` and assert substrings (see
  `TestProfileIdentityCardContract` at `settingsfocus_test.go:20-45` for the
  contract-test style, including the note that gomponents escapes `'` to
  `&#39;` in attribute values).

### Storybook facts

- Stories are plain funcs returning `Story`, registered in the ordered list in
  `internal/feature/storybook/story.go:60-133`; `capabilitiesStory()` is
  registered at `story.go:124`. The Settings-group section stories
  (`nudgesectionStory` at `stories_cards.go:102`, `capabilitiesStory` at
  `stories_cards.go:126`) live in `internal/feature/storybook/stories_cards.go`.
- The coverage test (`internal/feature/storybook/coverage_test.go:45`,
  `TestEveryRegisteredCardHasAStory`) maps registered CARD TYPES to story ids.
  This plan adds no new card type, so it stays green automatically — but the
  repo rule (AGENTS.md: "add or update its story in the same change") still
  requires a story for the new section, and the `settingsfocusStory` props
  must stay truthful.
- `settingsfocusStory` at `stories_cards.go:734` documents the sections; its
  `Section` prop line at `stories_cards.go:765`:

  ```go
  			{"Section", "string", `"profile"`, `Active section: "profile", "models", "heads", "nudges", or "capabilities". Controls which content renders.`},
  ```

### Docs to update

- `internal/self/knowledge.md:321-331` describes `balaur export` and
  `--encrypt` (ending "…owner-supplied passphrase via
  `BALAUR_EXPORT_PASSPHRASE`, no escrow, no cloud: lose the passphrase and the
  backup is unrecoverable."). It currently says nothing about a web surface
  for export/backup.
- `README.md:143`:

  ```
  - **Your record, portable:** `balaur export` writes a one-way Johnny Decimal Markdown mirror of your active nodes (plan 194); `--encrypt` produces a passphrase-protected archive (plan 195).
  ```

### Tour anchors near the edited lines (`.tours/` is a maintained artifact)

- `internal/web/web.go` is anchored at lines 1, 64, and 138 (tours
  `00-orientation`, `07-the-web-gateway`). The new routes are inserted after
  line 227 — all anchors are ABOVE the insertion, so no tour fix is needed.
- `internal/cards/cards.go` is anchored at lines 26 and 366
  (`08-hateoas-cards-and-boards`; line 366 is `func Validate`). The enum edit
  at `cards.go:242` MUST stay a single-line edit (append `"backup"` inside
  the existing slice literal, no new lines) so the `:366` anchor does not
  shift. If you cannot keep it single-line, fix the tour anchor in the same
  commit.

### Repo conventions that apply (quoted from AGENTS.md)

- "Errors are values: wrap with `fmt.Errorf("doing x: %w", err)`, return
  early, no panics in library code."
- "Structured logging only: `app.Logger()` (a `*slog.Logger`) with key/value
  pairs. No `log.Printf`/`fmt.Print*` in service code."
- "Audit strictly AFTER the successful write, never before — the audit log
  must not record a mutation that did not persist."
- "gomponents: alias the html package as `h "maragu.dev/gomponents/html"` …
  User/model text renders through escaping `g.Text`; `g.Raw` is for
  already-trusted, already-rendered HTML only."
- "Sanitize errors and tool output so they do not leak private paths, tokens,
  or vault content unnecessarily."
- Passphrase handling rules for this plan: the passphrase is read from the
  form value only; it is never logged, never persisted (no owner_settings
  key), never placed in any view-model field, and never echoed back into the
  re-rendered fragment or a toast. Do not `TrimSpace` it (passphrases are
  literal); only reject the empty string.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0 |
| Targeted tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/<pkg>/ -run <Name> -count=1` | ok |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tours lint | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | ok |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always set
`TMPDIR=$HOME/.cache/go-tmp` on test commands.)

## Suggested executor toolkit

- If the `ui-development` skill is available, invoke it before Steps 1 and 6 —
  it covers the Hearthwood component layers, the storybook workflow, and the
  Datastar `@post`/SSE contract this plan relies on.
- If the `go-standards` skill is available, apply it while writing the
  handlers in Step 4.
- Optional final check: the `run-balaur` skill (or `/verify`) to click the two
  new buttons in the real UI at `http://127.0.0.1:8090/`.

## Scope

**In scope** (the only files you should modify or create):

- `internal/feature/settingscards/settingsfocus_backup.go` (create)
- `internal/feature/settingscards/settingsfocus_backup_test.go` (create)
- `internal/feature/settingscards/settingsfocus.go` (add the section)
- `internal/feature/settingscards/settings.go` (add the tile link)
- `internal/cards/cards.go` (one-line enum edit at line 242)
- `internal/web/settings_backup.go` (create)
- `internal/web/settings_backup_test.go` (create)
- `internal/web/web.go` (two route lines after line 227)
- `internal/feature/storybook/stories_cards.go` (new story + settingsfocus props line)
- `internal/feature/storybook/story.go` (register the story)
- `internal/self/knowledge.md` (one sentence)
- `README.md` (extend one bullet)
- `plans/README.md` (status row only, when done)

**Out of scope** (do NOT touch, even though they look related):

- `internal/export/*` and `internal/cli/*` — the export/encrypt/restore
  internals and CLI verbs are shipped and leak-tested; this plan only calls
  them.
- Any restore/decrypt UI — needs file upload + overwrite semantics; deferred.
- Scheduling/automatic backups — owner-initiated only in this plan.
- `internal/web/messenger_settings.go` and the messenger token control — the
  exemplar, not the subject.
- `internal/ui/*` atoms and `basm.css` — reuse existing classes
  (`profile-card`, `profile-hint`, `profile-name-form`, `profile-name-row`,
  `btn btn-primary`, `kcard-actions`); no new CSS.

## Git workflow

- The executor runs in an isolated git worktree branched from `origin/main`;
  branch name `advisor/255-backup-settings-ui`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/`chore`),
  e.g. `feat(settings): backup & export section in the settings UI`.
- Commit per logical unit with explicit pathspecs (the main checkout is shared
  by parallel agents — stage only your own files, never `git add -A`).
- NEVER push; the reviewer merges.

## Steps

### Step 1: Create the Backup section component

Create `internal/feature/settingscards/settingsfocus_backup.go`, modeled on
`settingsfocus_nudges.go` (header comment style, imports, form idiom). Target
shape (adjust copy wording only if it collides with existing storybook
do/don'ts — the SEMANTICS are load-bearing):

```go
// settingsfocus_backup.go — the Backup settings section: write the Markdown
// mirror and the passphrase-encrypted backup archive without a terminal.
// UI face of the `balaur export` / `balaur export --encrypt` CLI verbs.
package settingscards

import (
	"fmt"

	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"
)

// BackupView is the view-model for the backup section. It carries only the
// RESULTS of the last action — never the passphrase (which is read from the
// form once, used, and discarded; it must not appear in any view-model field).
type BackupView struct {
	MirrorDone  bool   // last action wrote the Markdown mirror
	MirrorFiles int    // files written by that mirror run
	MirrorDest  string // where the mirror was written
	ArchivePath string // path of the encrypted archive just written; empty otherwise
	Error       string // owner-safe error line; empty when the last action succeeded
}

// BackupSection renders the backup controls (#backup-section): the Markdown
// mirror button and the encrypted-backup form. Re-render target after POST
// /ui/settings/export and /ui/settings/backup (outer patch #backup-section).
func BackupSection(v BackupView) g.Node {
	out := []g.Node{
		h.H2(h.Class("profile-card-title"), g.Text("Backup & export")),
		h.P(h.Class("profile-hint"), g.Text("Your record, portable: write a readable Markdown mirror of your active notes, memories, and journal — or wrap it in an encrypted archive for off-box backup. Nothing leaves this machine.")),
	}
	if v.Error != "" {
		out = append(out, h.P(h.Class("profile-hint"), g.Text("Error: "+v.Error)))
	}

	// Plain mirror.
	out = append(out,
		h.Form(
			data.On("submit", "@post('/ui/settings/export', {contentType:'form'})", data.ModifierPrevent),
			h.Div(h.Class("kcard-actions"),
				h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"), g.Text("Write Markdown mirror")),
			),
		),
	)
	if v.MirrorDone {
		out = append(out, h.P(h.Class("profile-hint"),
			g.Text(fmt.Sprintf("Wrote %d files to %s.", v.MirrorFiles, v.MirrorDest))))
	}

	// Encrypted backup. The passphrase input is type=password + autocomplete=off
	// and is NEVER echoed back — BackupView has no passphrase field by design.
	out = append(out,
		h.Form(
			h.Class("profile-name-form"),
			data.On("submit", "@post('/ui/settings/backup', {contentType:'form'})", data.ModifierPrevent),
			h.Label(h.For("backup_passphrase"), g.Text("Passphrase")),
			h.Div(h.Class("profile-name-row"),
				h.Input(
					h.ID("backup_passphrase"),
					h.Name("passphrase"),
					h.Type("password"),
					h.Placeholder("choose a strong passphrase…"),
					g.Attr("autocomplete", "off"),
				),
				h.Button(h.Class("btn btn-primary"), h.Type("submit"), g.Text("Write encrypted backup")),
			),
		),
		h.P(h.Class("profile-hint"), g.Text("If you lose this passphrase, this backup is UNRECOVERABLE — there is no recovery, no escrow, no cloud.")),
	)
	if v.ArchivePath != "" {
		out = append(out, h.P(h.Class("profile-hint"), g.Text("Encrypted backup written to "+v.ArchivePath+". Restore with: balaur restore --archive <path> --out <dir>.")))
	}

	return h.Article(h.Class("profile-card"), h.ID("backup-section"), g.Group(out))
}
```

The warning sentence intentionally mirrors the CLI warning at
`internal/cli/export.go:69`.

**Verify**: `CGO_ENABLED=0 go build ./internal/feature/settingscards/` → exit 0.

### Step 2: Wire the "backup" section into the dispatcher, the card enum, and the tile

1. `internal/feature/settingscards/settingsfocus.go`:
   - `SettingsFocusView` (line 21-28): add field `Backup BackupView // used when Section == "backup"`
     and extend the `Section` comment to
     `// "profile" | "models" | "heads" | "nudges" | "capabilities" | "backup"`.
   - `BuildSettingsFocus` allow-list (line 35): change to
     `case "models", "heads", "nudges", "capabilities", "backup":`.
     No build case is needed — the zero `BackupView` IS the idle state (no
     data fetch; KISS).
   - `SettingsFocus` render switch (lines 65-79): add
     `case "backup": content = BackupSection(v.Backup)`.
2. `internal/cards/cards.go:242`: append `"backup"` to the enum slice —
   single-line edit, the line becomes:

   ```go
   				{Name: "section", Enum: []string{"profile", "models", "heads", "appearance", "capabilities", "nudges", "backup"}, Doc: "settings section (default profile)"},
   ```

   Confirm the file's total line count is unchanged (`git diff --stat internal/cards/cards.go`
   shows `1 insertion(+), 1 deletion(-)`) so the tour anchor at `cards.go:366`
   does not shift.
3. `internal/feature/settingscards/settings.go:27-32`: add a fifth `h.Li(...)`
   link matching the existing four exactly, with `section=backup` and label
   `Backup`:

   ```go
   			h.Li(h.A(h.Href("/ui/show/settings?section=backup"), g.Attr("data-on:click__prevent", "@get('/ui/show/settings?section=backup'); basmOpenPanel()"), g.Text("Backup"))),
   ```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0, and
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/feature/settingscards/ ./internal/cards/ -count=1` → ok.

### Step 3: Component contract test

Create `internal/feature/settingscards/settingsfocus_backup_test.go` in
package `settingscards_test`, modeled on `TestProfileIdentityCardContract`
(`settingsfocus_test.go:20-45`, rendering via the existing
`renderNode(t, n)` helper in that file). Tests:

1. `TestBackupSectionContract` — render `settingscards.BackupSection(settingscards.BackupView{})`
   and assert it contains ALL of:
   - `id="backup-section"`
   - `class="profile-card"`
   - `data-on:submit__prevent="@post(&#39;/ui/settings/export&#39;, {contentType:&#39;form&#39;})"`
   - `data-on:submit__prevent="@post(&#39;/ui/settings/backup&#39;, {contentType:&#39;form&#39;})"`
   - `name="passphrase"`
   - `type="password"`
   - `autocomplete="off"`
   - `UNRECOVERABLE`

   (gomponents escapes `'` → `&#39;` in attribute values.)
2. `TestBackupSectionResults` — render with
   `BackupView{MirrorDone: true, MirrorFiles: 3, MirrorDest: "/data/export", ArchivePath: "/data/backup/balaur-backup-x.enc"}`
   and assert `Wrote 3 files to /data/export.` and
   `balaur restore` both appear; render with `BackupView{Error: "boom"}` and
   assert `Error: boom` appears.

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/feature/settingscards/ -run TestBackupSection -count=1` → ok, 2 tests pass.

### Step 4: Web handlers + routes

Create `internal/web/settings_backup.go`, modeled on
`internal/web/messenger_settings.go`. Target shape:

```go
package web

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/export"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/store"
)

// settings_backup.go — the web face of `balaur export` (plans 194/195): the
// settings Backup section's two POSTs. Restore stays CLI-only. The passphrase
// is read from the form once, used, and discarded — never logged, never
// persisted, never echoed back (BackupView has no passphrase field).

// patchBackupSection re-renders #backup-section with the given result view.
func (h *handlers) patchBackupSection(e *core.RequestEvent, view settingscards.BackupView, toastTone, toastMsg string) error {
	var b strings.Builder
	if err := settingscards.BackupSection(view).Render(&b); err != nil {
		return e.InternalServerError("rendering backup section", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	patchOuterHTML(sse, "backup-section", b.String())
	if toastMsg != "" {
		emitToast(sse, toastTone, toastMsg)
	}
	return nil
}

// exportMirrorNow handles POST /ui/settings/export — writes the Markdown
// mirror to <data dir>/export (same default as the CLI verb) and re-renders
// the section with the destination + file count.
func (h *handlers) exportMirrorNow(e *core.RequestEvent) error {
	dest := filepath.Join(h.app.DataDir(), "export")
	files, err := export.ExportMirror(h.app, dest)
	if err != nil {
		h.app.Logger().Warn("export mirror from settings", "err", err)
		return h.patchBackupSection(e, settingscards.BackupView{Error: "could not write the mirror — see the server log"}, "", "")
	}
	// Audit strictly AFTER the successful write.
	store.Audit(h.app, "owner", "export.mirror", dest, true, map[string]any{"files": len(files)})
	view := settingscards.BackupView{MirrorDone: true, MirrorFiles: len(files), MirrorDest: dest}
	return h.patchBackupSection(e, view, "success", "Markdown mirror written.")
}

// backupEncryptNow handles POST /ui/settings/backup — renders the mirror into
// a wiped temp dir (same shape as the CLI's runEncrypt), encrypts it into
// <data dir>/backup/balaur-backup-<stamp>.enc, and shows the archive path.
// The passphrase comes from the form value only; it is not trimmed (only the
// empty string is rejected) and never logged.
func (h *handlers) backupEncryptNow(e *core.RequestEvent) error {
	passphrase := e.Request.FormValue("passphrase")
	if passphrase == "" {
		return e.BadRequestError("a passphrase is required", nil)
	}

	tmp, err := os.MkdirTemp("", "balaur-export-*")
	if err != nil {
		return e.InternalServerError("creating export temp dir", err)
	}
	defer os.RemoveAll(tmp)

	if _, err := export.ExportMirror(h.app, tmp); err != nil {
		h.app.Logger().Warn("export mirror for backup", "err", err)
		return h.patchBackupSection(e, settingscards.BackupView{Error: "could not render the mirror — see the server log"}, "", "")
	}

	backupDir := filepath.Join(h.app.DataDir(), "backup")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return e.InternalServerError("creating backup dir", err)
	}
	archive := filepath.Join(backupDir, "balaur-backup-"+time.Now().Format("20060102-150405")+".enc")
	if err := export.EncryptDir(tmp, archive, passphrase); err != nil {
		h.app.Logger().Warn("encrypting backup", "err", err)
		return h.patchBackupSection(e, settingscards.BackupView{Error: "could not encrypt the backup — see the server log"}, "", "")
	}
	// Audit strictly AFTER the successful write. No passphrase material anywhere.
	store.Audit(h.app, "owner", "export.encrypt", archive, true, nil)
	return h.patchBackupSection(e, settingscards.BackupView{ArchivePath: archive}, "success", "Encrypted backup written.")
}
```

Register both routes in `internal/web/web.go` directly after line 227
(`se.Router.POST("/ui/settings/messenger-token", h.saveMessengerToken)`):

```go
	// Settings → Backup: owner-clickable export mirror + encrypted backup
	// (web face of `balaur export` / `--encrypt`; restore stays CLI-only).
	se.Router.POST("/ui/settings/export", h.exportMirrorNow)
	se.Router.POST("/ui/settings/backup", h.backupEncryptNow)
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0, then `go vet ./...` → exit 0.

### Step 5: Handler tests

Create `internal/web/settings_backup_test.go` (package `web`). Add a local
form-post helper modeled on `serveReviewRoute` (`review_test.go:54-72`) but
with a form body and the `sseHeaders` header pair
(`knowledge_test.go:46-49` — `Content-Type: application/x-www-form-urlencoded`,
`Accept: text/event-stream`):

```go
func serveSettingsForm(t *testing.T, app core.App, url, form string) *httptest.ResponseRecorder {
	// as serveReviewRoute, but: httptest.NewRequest("POST", url, strings.NewReader(form))
	// + req.Header.Set for both sseHeaders entries.
}
```

Tests (seed an exportable node first where noted, using
`nodes.Create(app, "note", "Backup Note", "Body text.", nodes.StatusActive, nil)`
as in `internal/cli/restore_test.go:17-18`; audit assertions use
`store.ListAudit(app, "<action>", "", 10)` as in
`internal/web/life_drop_test.go:31-37`):

1. `TestSettingsExportMirror` — `newWebApp(t)`, seed one node, POST
   `/ui/settings/export` with empty form → status 200; the response body
   contains `backup-section` and `Wrote 1 files to`; at least one `.md` file
   exists under `filepath.Join(app.DataDir(), "export")` (walk with
   `filepath.WalkDir`, skip the `.git` dir); `store.ListAudit(app, "export.mirror", "", 10)`
   returns ≥ 1 row whose `target` is the export dir.
2. `TestSettingsBackupEmptyPassphrase` — POST `/ui/settings/backup` with body
   `passphrase=` → status 400; `filepath.Join(app.DataDir(), "backup")` does
   not exist (`os.Stat` → `os.IsNotExist`); `store.ListAudit(app, "export.encrypt", "", 10)`
   returns 0 rows.
3. `TestSettingsBackupRoundTrip` — seed one node; POST `/ui/settings/backup`
   with body `passphrase=correct+horse+battery+staple` → status 200; exactly
   one file matching `balaur-backup-*.enc` exists under
   `<DataDir>/backup/`; `export.DecryptDir(archive, filepath.Join(t.TempDir(), "restored"), "correct horse battery staple")`
   returns nil and the restored dir contains ≥ 1 `.md` file (round-trip
   proof); `store.ListAudit(app, "export.encrypt", "", 10)` returns ≥ 1 row.
4. `TestSettingsBackupNoPassphraseLeak` — reusing the round-trip app/response
   (or a fresh POST): the HTTP response body does NOT contain
   `correct horse battery staple`; `json.Marshal` of every
   `export.encrypt` audit record does NOT contain
   `correct horse battery staple` (assert on the raw marshaled bytes).

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run TestSettings -count=1` → ok, all new tests pass.

### Step 6: Storybook story

1. In `internal/feature/storybook/stories_cards.go`, add
   `backupsectionStory()` directly after `capabilitiesStory()` (which ends at
   `stories_cards.go:159`), modeled on `nudgesectionStory`
   (`stories_cards.go:102-122`):

   ```go
   // backupsectionStory documents the Backup & export settings section — the
   // web face of `balaur export` / `--encrypt`. Restore stays CLI-only.
   func backupsectionStory() Story {
   	return Story{
   		ID: "backupsection", Group: "Settings", Title: "Backup & export", Wide: true, OnDark: true,
   		Blurb: "Owner-clickable sovereignty: write the Johnny Decimal Markdown mirror, or wrap it in a passphrase-encrypted archive — no terminal, nothing leaves the box. The passphrase is typed once, used once, and never stored, logged, or echoed back. Lose it and the backup is unrecoverable: no recovery, no escrow, no cloud.",
   		Variants: []Variant{
   			{"idle", settingscards.BackupSection(settingscards.BackupView{})},
   			{"mirror written", settingscards.BackupSection(settingscards.BackupView{MirrorDone: true, MirrorFiles: 42, MirrorDest: "/home/owner/.balaur/pb_data/export"})},
   			{"backup written", settingscards.BackupSection(settingscards.BackupView{ArchivePath: "/home/owner/.balaur/pb_data/backup/balaur-backup-20260701-093000.enc"})},
   			{"error", settingscards.BackupSection(settingscards.BackupView{Error: "could not write the mirror — see the server log"})},
   		},
   		Props: []Prop{
   			{"MirrorDone", "bool", "false", "Last action wrote the Markdown mirror; shows the files/dest line."},
   			{"MirrorFiles", "int", "0", "Files written by that mirror run."},
   			{"MirrorDest", "string", "—", "Where the mirror was written (<data dir>/export)."},
   			{"ArchivePath", "string", "—", "Path of the encrypted archive just written; empty otherwise."},
   			{"Error", "string", "—", "Owner-safe error line; raw errors stay in the server log."},
   		},
   		Dos: []string{
   			"Carry the unrecoverable-passphrase warning verbatim next to the form.",
   			"Show the real on-disk paths — the owner should know exactly where their data is.",
   		},
   		Donts: []string{
   			"Echo, store, log, or audit the passphrase — the view-model has no passphrase field by design.",
   			"Offer restore here — decrypting an archive is CLI-only until upload/overwrite semantics exist.",
   		},
   	}
   }
   ```

2. Register it in `internal/feature/storybook/story.go` by adding
   `backupsectionStory(),` after `capabilitiesStory(),` (line 124).
3. Update the `settingsfocusStory` `Section` prop line
   (`stories_cards.go:765`) to include `"backup"`:

   ```go
   			{"Section", "string", `"profile"`, `Active section: "profile", "models", "heads", "nudges", "capabilities", or "backup". Controls which content renders.`},
   ```

   and in the story's Blurb (`stories_cards.go:759`) change the tab list
   `(Profile / Models / Heads / Nudges / Capabilities)` to
   `(Profile / Models / Heads / Nudges / Capabilities / Backup)`.

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/feature/storybook/ -count=1` → ok
(this also proves `TestEveryRegisteredCardHasAStory` and the story registry
stayed consistent).

### Step 7: Docs — self-knowledge and README

1. `internal/self/knowledge.md`: after the sentence ending "…lose the
   passphrase and the backup is unrecoverable." (line ~331, before
   "`balaur restore --archive <path>`"), insert this sentence:

   > Both verbs are also one click away in the web Settings → Backup section
   > (POST /ui/settings/export writes the mirror to `<data dir>/export`; POST
   > /ui/settings/backup encrypts it to `<data dir>/backup/`, passphrase typed
   > into the form, used once, never stored or logged; both actions are
   > audited as `export.mirror`/`export.encrypt`; restore stays CLI-only).

2. `README.md:143`: extend the bullet so it ends:

   ```
   …a passphrase-protected archive (plan 195). Both are also one click away in the web UI under Settings → Backup (restore stays CLI-only).
   ```

**Verify**:
`grep -n "Settings → Backup" internal/self/knowledge.md README.md` → one match
in each file.

### Step 8: Full gates

Run, in order:

1. `gofmt -l .` → empty output
2. `go vet ./...` → exit 0
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0
4. `CGO_ENABLED=0 go build ./...` → exit 0
5. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
6. `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
   (web.go and cards.go carry tour anchors — see "Current state")
7. `git diff --check` → no output

**Verify**: all seven commands succeed as stated.

## Test plan

- **New component tests** in
  `internal/feature/settingscards/settingsfocus_backup_test.go` (Step 3):
  contract (ids, both `@post` targets, password input, `autocomplete="off"`,
  the UNRECOVERABLE warning) and result rendering (mirror line, archive line,
  error line). Model after `TestProfileIdentityCardContract`
  (`settingsfocus_test.go:20-45`).
- **New handler tests** in `internal/web/settings_backup_test.go` (Step 5):
  1. happy-path mirror: 200, `.md` files on disk under `<DataDir>/export`,
     `export.mirror` audit row exists;
  2. empty passphrase: 400, no `<DataDir>/backup` dir, no `export.encrypt`
     audit row;
  3. encrypted round-trip: 200, one `balaur-backup-*.enc` archive,
     `export.DecryptDir` with the same passphrase succeeds and yields `.md`
     files;
  4. leak check: passphrase string absent from the HTTP response body and
     from the marshaled audit records.
  Model after `internal/web/messenger_settings_test.go` (scenario style) and
  `internal/web/life_drop_test.go` (route-serve + audit assertion style);
  never a real model — `newWebApp` already fakes everything needed.
- **Storybook**: `go test ./internal/feature/storybook/` green after the new
  story registers (Step 6).
- **Verification**: `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` →
  exit 0, including 6 new tests (2 component + 4 handler).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → exit 0, no output
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run TestSettings -count=1` → ok (4 new tests)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/feature/settingscards/ -run TestBackupSection -count=1` → ok (2 new tests)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
- [ ] `grep -n '"backup"' internal/cards/cards.go` → exactly one match, on the settings `section` enum line
- [ ] `grep -rn "backupsectionStory" internal/feature/storybook/story.go` → one match (registered)
- [ ] `grep -n "Settings → Backup" internal/self/knowledge.md README.md` → one match in each
- [ ] `grep -rn "passphrase" internal/feature/settingscards/settingsfocus_backup.go | grep -i "value("` → no matches (the input never carries a value)
- [ ] `git status --porcelain` shows changes ONLY in the in-scope files listed
      under Scope (plus `plans/README.md`)
- [ ] `plans/README.md` status row for 255 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows any in-scope file changed since `077318a` and the
  "Current state" excerpts no longer match the live code (in particular: the
  `SettingsFocusView` section list, the `cards.go:242` enum line, or the
  `web.go:227` route line).
- The single-line enum edit in `internal/cards/cards.go` cannot stay
  single-line, AND you cannot fix the `.tours/08-hateoas-cards-and-boards.tour`
  anchor at `internal/cards/cards.go:366` in the same commit.
- `TestSettingsBackupRoundTrip` fails because `export.DecryptDir` cannot open
  the archive the handler wrote — that means the handler's EncryptDir call is
  wrong; do not modify `internal/export` to compensate.
- The synchronous export/encrypt path takes longer than ~30 seconds against
  the seeded dev data (run `balaur seed` on a scratch `--dir` to check if in
  doubt) — ship nothing async; report that an async follow-up plan is needed
  and whether the synchronous v1 is still acceptable.
- Any step appears to require touching `internal/export/*`, `internal/cli/*`,
  `internal/ui/*`, or `basm.css`.
- A step's verification fails twice after a reasonable fix attempt.
- You find an existing storybook or settings convention that contradicts this
  plan's component shape (e.g. a mandatory story field or section-registration
  step this plan does not mention).

## Maintenance notes

- **Restore stays CLI-only, deliberately.** A UI restore needs file upload +
  explicit overwrite semantics (`balaur restore` refuses a non-empty `--out`);
  bolting that onto this section would grow scope and risk. The section's
  archive-written line points the owner at the CLI verb instead.
- **No scheduling.** Backups are owner-initiated clicks; automatic/periodic
  backup is a separate decision (where to put archives, rotation, passphrase
  reuse) and was explicitly excluded here.
- **Same-second overwrite**: two encrypted backups within one second share a
  filename (`20060102-150405` stamp) and the second overwrites the first.
  Accepted for v1 (human click cadence); revisit only if automation lands.
- **Synchronous request**: the export runs inside the POST. If the owner's
  data grows to where a mirror takes many seconds, this needs the async
  pattern the model-download flow uses — a reviewer should check the timing
  note in the STOP conditions was honored.
- **Archives live inside the data dir** (`pb_data/backup/`), which is already
  git-ignored and treated as secret; nothing new to ignore. The point of the
  encrypted archive is that the OWNER copies it off-box.
- **What a reviewer should scrutinize**: the passphrase discipline (form →
  `EncryptDir` → gone; no view-model field, no slog call, no audit detail),
  audit-after-write ordering, and that the error paths return owner-safe copy
  while the raw error goes only to `app.Logger()`.
