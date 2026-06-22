# Plan 148: Make the avatar roster a single source of truth

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat ab2c0a9..HEAD -- internal/store/owner_settings.go internal/store/owner_settings_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `ab2c0a9`, 2026-06-22

## Why this matters

`internal/store/owner_settings.go` defines each avatar roster **twice**: once as a
`map[string]string` literal (`soulAvatarMap`, `balaurAvatarMap`) used for key
validation and URL lookups, and again as `[]AvatarEntry` slices returned by
`SoulAvatars()` / `BalaurHeads()` (the documented single source of truth, iterated
by the web option builders). The two copies must be kept in lockstep by hand —
adding or renaming an avatar means editing two places, and a typo or omission in
either silently desynchronizes validation from what the UI offers. This is
~34 lines of duplicated data that should be derived, not maintained twice.

The fix is to delete the two map literals and **derive** the lookup maps from the
slices at package-init time, preserving identical behavior — including the two
legacy soul aliases (`"male"`, `"female"`) that exist in the map but not the slice.
After this lands, the slices are the only place an avatar is declared; everything
else reads from them.

## Current state

The facts the executor needs, inlined:

- `internal/store/owner_settings.go` — the only file that declares avatar data.
  Both maps and both slices live here, plus the functions that consume them.
- `internal/store/owner_settings_test.go` — covers the rosters, the
  validity functions, and (crucially) the legacy aliases. It references the
  package-private `soulAvatarMap` directly, so the map must keep that exact name.

### What the maps are used for (verified — these are the ONLY uses in the repo)

`grep -rn "soulAvatarMap" --include="*.go" .` and the same for `balaurAvatarMap`
return matches **only** in these two files. The maps are consumed by:

1. **Validation** — `ValidSoulAvatarKey` / `ValidBalaurAvatarKey` do
   `_, ok := <map>[key]; return ok`.
2. **URL lookup** — `SoulAvatarURL`, `BalaurAvatarURL`, `BalaurAvatarURLForKey`
   do `if url, ok := <map>[key]; ok { return url }`.
3. **Test** — `owner_settings_test.go:74,77` reads `soulAvatarMap["male"]` and
   `soulAvatarMap["female"]` directly (in-package access).

No production code outside `owner_settings.go` touches either map variable.

### Critical legacy-alias detail (do NOT lose these)

- `soulAvatarMap` contains **two keys that are NOT in the `SoulAvatars()` slice**:
  - `"male"` → `/static/avatars/soul-01.png`
  - `"female"` → `/static/avatars/soul-02.png`
  These are kept as legacy aliases and MUST survive. The derived map must add
  them back on top of the slice-derived entries.
- `balaurAvatarMap` has **no** extra keys — it is exactly the 16 `BalaurHeads()`
  keys, 1:1. No aliases to preserve there.

### Exact current code (HEAD `ab2c0a9`)

`internal/store/owner_settings.go:9-16` — the entry type (the single source of truth):

```go
// AvatarEntry is one selectable avatar: key (stored in owner_settings /
// head records), human label, and served URL. The exported rosters are the
// single source of truth — web option builders iterate these.
type AvatarEntry struct {
	Key   string
	Label string
	URL   string
}
```

`internal/store/owner_settings.go:60-122` — soul section (map literal + consumers):

```go
// ── Soul avatar ────────────────────────────────────────────────────────

// soulAvatarMap maps avatar keys to their static file paths.
// Legacy values "male" and "female" are kept as aliases.
var soulAvatarMap = map[string]string{
	"soul-01": "/static/avatars/soul-01.png", // Him
	"soul-02": "/static/avatars/soul-02.png", // Her
	"soul-03": "/static/avatars/soul-03.png", // Elder
	"soul-04": "/static/avatars/soul-04.png", // Youth
	"soul-05": "/static/avatars/soul-05.png", // Maker
	"soul-06": "/static/avatars/soul-06.png", // Cyclops
	"soul-07": "/static/avatars/soul-07.png", // Gnome
	"soul-08": "/static/avatars/soul-08.png", // Ogre
	"soul-09": "/static/avatars/soul-09.png", // Strigoi
	"soul-10": "/static/avatars/soul-10.png", // Zmeu
	"soul-11": "/static/avatars/soul-11.png", // Iele
	"soul-12": "/static/avatars/soul-12.png", // Muma Pădurii
	"soul-13": "/static/avatars/soul-13.png", // Căpcăun
	"soul-14": "/static/avatars/soul-14.png", // Solomonar
	"soul-15": "/static/avatars/soul-15.png", // Vâlvă
	"soul-16": "/static/avatars/soul-16.png", // Pricolici
	"male":    "/static/avatars/soul-01.png", // legacy alias
	"female":  "/static/avatars/soul-02.png", // legacy alias
}

// ValidSoulAvatarKey reports whether key is a recognised soul avatar.
func ValidSoulAvatarKey(key string) bool {
	_, ok := soulAvatarMap[key]
	return ok
}

// SoulAvatars returns the roster of 16 soul avatars, the single source of truth.
func SoulAvatars() []AvatarEntry {
	return []AvatarEntry{
		// Basm world — human characters
		{"soul-01", "Him", "/static/avatars/soul-01.png"},
		{"soul-02", "Her", "/static/avatars/soul-02.png"},
		{"soul-03", "Elder", "/static/avatars/soul-03.png"},
		{"soul-04", "Youth", "/static/avatars/soul-04.png"},
		{"soul-05", "Maker", "/static/avatars/soul-05.png"},
		{"soul-06", "Cyclops", "/static/avatars/soul-06.png"},
		{"soul-07", "Gnome", "/static/avatars/soul-07.png"},
		{"soul-08", "Ogre", "/static/avatars/soul-08.png"},
		// Romanian mythological creatures
		{"soul-09", "Strigoi", "/static/avatars/soul-09.png"},
		{"soul-10", "Zmeu", "/static/avatars/soul-10.png"},
		{"soul-11", "Iele", "/static/avatars/soul-11.png"},
		{"soul-12", "Muma", "/static/avatars/soul-12.png"},
		{"soul-13", "Căpcăun", "/static/avatars/soul-13.png"},
		{"soul-14", "Solomonar", "/static/avatars/soul-14.png"},
		{"soul-15", "Vâlvă", "/static/avatars/soul-15.png"},
		{"soul-16", "Pricolici", "/static/avatars/soul-16.png"},
	}
}

// SoulAvatarURL resolves the owner's soul avatar preference to a static URL.
func SoulAvatarURL(app core.App) string {
	key := GetOwnerSetting(app, "soul_avatar", "soul-01")
	if url, ok := soulAvatarMap[key]; ok {
		return url
	}
	return "/static/avatars/soul-01.png"
}
```

`internal/store/owner_settings.go:124-200` — balaur section (map literal + consumers):

```go
// ── Balaur head avatar ─────────────────────────────────────────────────

var balaurAvatarMap = map[string]string{
	"balaur-01": "/static/avatars/balaur-01.png", // Wise (default)
	"balaur-02": "/static/avatars/balaur-02.png", // Ancient
	"balaur-03": "/static/avatars/balaur-03.png", // Guardian
	"balaur-04": "/static/avatars/balaur-04.png", // Scholar
	"balaur-05": "/static/avatars/balaur-05.png", // Wild
	"balaur-06": "/static/avatars/balaur-06.png", // Storm
	"balaur-07": "/static/avatars/balaur-07.png", // Night
	"balaur-08": "/static/avatars/balaur-08.png", // Young
	"balaur-09": "/static/avatars/balaur-09.png", // Ember
	"balaur-10": "/static/avatars/balaur-10.png", // Frost
	"balaur-11": "/static/avatars/balaur-11.png", // Healer
	"balaur-12": "/static/avatars/balaur-12.png", // Trickster
	"balaur-13": "/static/avatars/balaur-13.png", // Dreamer
	"balaur-14": "/static/avatars/balaur-14.png", // Forest
	"balaur-15": "/static/avatars/balaur-15.png", // Dawn
	"balaur-16": "/static/avatars/balaur-16.png", // Sage
}

// BalaurHeads returns the roster of 16 Balaur personalities, the single source of truth.
func BalaurHeads() []AvatarEntry {
	return []AvatarEntry{
		{"balaur-01", "Wise", "/static/avatars/balaur-01.png"},
		{"balaur-02", "Ancient", "/static/avatars/balaur-02.png"},
		{"balaur-03", "Guardian", "/static/avatars/balaur-03.png"},
		{"balaur-04", "Scholar", "/static/avatars/balaur-04.png"},
		{"balaur-05", "Wild", "/static/avatars/balaur-05.png"},
		{"balaur-06", "Storm", "/static/avatars/balaur-06.png"},
		{"balaur-07", "Night", "/static/avatars/balaur-07.png"},
		{"balaur-08", "Young", "/static/avatars/balaur-08.png"},
		{"balaur-09", "Ember", "/static/avatars/balaur-09.png"},
		{"balaur-10", "Frost", "/static/avatars/balaur-10.png"},
		{"balaur-11", "Healer", "/static/avatars/balaur-11.png"},
		{"balaur-12", "Trickster", "/static/avatars/balaur-12.png"},
		{"balaur-13", "Dreamer", "/static/avatars/balaur-13.png"},
		{"balaur-14", "Forest", "/static/avatars/balaur-14.png"},
		{"balaur-15", "Dawn", "/static/avatars/balaur-15.png"},
		{"balaur-16", "Sage", "/static/avatars/balaur-16.png"},
	}
}

// ValidBalaurAvatarKey reports whether key is a recognised Balaur head.
func ValidBalaurAvatarKey(key string) bool {
	_, ok := balaurAvatarMap[key]
	return ok
}

// BalaurAvatarURL resolves the owner's chosen Balaur head to a static URL.
func BalaurAvatarURL(app core.App) string {
	key := GetOwnerSetting(app, "balaur_avatar", "balaur-01")
	if url, ok := balaurAvatarMap[key]; ok {
		return url
	}
	return "/static/avatars/balaur-01.png"
}
```

`internal/store/owner_settings.go:190-200` — the by-key resolver (also uses the map):

```go
// ── Balaur head avatar by key ──────────────────────────────────────

// BalaurAvatarURLForKey resolves a Balaur avatar key (balaur-01…balaur-16) to
// a static URL, falling back to the owner's default when the key is empty or
// unknown. Used to render a head's avatar (built-in or custom).
func BalaurAvatarURLForKey(app core.App, key string) string {
	if url, ok := balaurAvatarMap[key]; ok {
		return url
	}
	return BalaurAvatarURL(app)
}
```

### Test that pins the legacy aliases (must keep passing unchanged)

`internal/store/owner_settings_test.go:62-80`:

```go
func TestLegacySoulAvatarAliases(t *testing.T) {
	app := storetest.NewApp(t)
	// Legacy aliases should still resolve
	if SoulAvatarURL(app) != "/static/avatars/soul-01.png" {
		t.Fatal("default soul avatar wrong")
	}
	if !ValidSoulAvatarKey("male") {
		t.Fatal("legacy male alias not valid")
	}
	if !ValidSoulAvatarKey("female") {
		t.Fatal("legacy female alias not valid")
	}
	if url := soulAvatarMap["male"]; url != "/static/avatars/soul-01.png" {
		t.Fatalf("male alias resolves to %q, want soul-01", url)
	}
	if url := soulAvatarMap["female"]; url != "/static/avatars/soul-02.png" {
		t.Fatalf("female alias resolves to %q, want soul-02", url)
	}
}
```

Because this test reads `soulAvatarMap["male"]` / `soulAvatarMap["female"]`
directly (in-package), the derived map MUST keep the exact name `soulAvatarMap`
and MUST contain those two alias keys. Do not touch the test.

### Repo conventions that apply

- Errors are values; no panics in library code. This change has no error paths.
- Prefer modern stdlib (`slices`/`maps`). You do not need them here — a plain
  package-level `var` derived in a tiny build helper is simplest. KISS wins.
- `staticcheck` gates the build, including U1000 (dead code): after deleting the
  map literals, the package must have no unused symbols or imports. The current
  imports are only `fmt` and `github.com/pocketbase/pocketbase/core`; both stay
  used (`fmt.Errorf` in `SetOwnerSetting`, `core.App` throughout). Do NOT add or
  remove imports.

## Commands you will need

| Purpose    | Command                              | Expected on success     |
|------------|--------------------------------------|-------------------------|
| Build      | `CGO_ENABLED=0 go build ./...`        | exit 0                  |
| Vet        | `go vet ./...`                        | exit 0                  |
| Tests(pkg) | `go test ./internal/store/...`        | all pass                |
| Tests(all) | `go test ./...`                       | all pass                |
| Format     | `gofmt -l internal/store/owner_settings.go` | prints nothing    |
| Lint       | `make lint`                           | exit 0                  |
| Diff check | `git diff --check`                    | no whitespace errors    |

(A `PostToolUse` hook runs `gofmt -w` on every edited `.go` file automatically;
the `gofmt -l` gate confirms the result.)

## Scope

**In scope** (the only file you should modify):
- `internal/store/owner_settings.go`

**Out of scope** (do NOT touch):
- `internal/store/owner_settings_test.go` — the tests are the behavior contract
  for this change. They must keep passing **unchanged**. Do not edit them. (In
  particular, do not "fix" the direct `soulAvatarMap[...]` reads — keeping them
  valid is the whole point of preserving the map name.)
- The `SoulAvatars()` / `BalaurHeads()` slice bodies — their entries are the
  source of truth and stay exactly as they are. Do not reorder, relabel, or
  re-key them.
- Any web option builder (`internal/web/models.go`,
  `internal/feature/settingscards/settingsfocus.go`,
  `internal/feature/headscards/heads.go`) — they iterate the slices, which are
  unchanged.

## Git workflow

- Land directly on `main`; no PR gate. Do NOT push or commit unless the operator
  explicitly says to — make the change, run the gates, and report.
- If you make a branch, name it `advisor/148-avatar-roster-single-source`.
- Conventional-commit subject if/when asked to commit, e.g.:
  `refactor(store): derive avatar lookup maps from rosters (plan 148)`

## Steps

### Step 1: Add a private helper that derives a lookup map from a slice

In `internal/store/owner_settings.go`, just **below** the `AvatarEntry` type
definition (after line 16, the closing `}` of the struct), add a small helper:

```go
// avatarMap builds a key→URL lookup from a roster slice. The slices returned
// by SoulAvatars / BalaurHeads are the single source of truth; the package-level
// lookup maps below are derived from them at init so an avatar is declared once.
func avatarMap(entries []AvatarEntry) map[string]string {
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Key] = e.URL
	}
	return m
}
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0 (helper compiles; it is
referenced in the next step, so build it together with Step 2 if the lone helper
trips U1000 — see note below).

> Note: `staticcheck` flags an unused function (U1000). `avatarMap` becomes used
> the moment Step 2 wires it in. Do Steps 1 and 2 as one edit pass, then run the
> build/lint once after Step 2.

### Step 2: Replace the soul map literal with a derived map

Delete the entire `soulAvatarMap` literal block — the comment lines and the
`var soulAvatarMap = map[string]string{ ... }` that currently span lines 62–83
(from `// soulAvatarMap maps avatar keys...` through the closing `}` after the
`"female"` alias line).

Replace it with a derived declaration that re-adds the two legacy aliases on top
of the slice-derived entries:

```go
// soulAvatarMap is the key→URL lookup for soul avatars, derived from the
// SoulAvatars roster (the single source of truth). Legacy values "male" and
// "female" are kept as aliases for owner_settings written before the soul-NN
// keys existed.
var soulAvatarMap = func() map[string]string {
	m := avatarMap(SoulAvatars())
	m["male"] = m["soul-01"]   // legacy alias
	m["female"] = m["soul-02"] // legacy alias
	return m
}()
```

Leave `ValidSoulAvatarKey`, `SoulAvatars`, and `SoulAvatarURL` exactly as they
are — they keep reading `soulAvatarMap`, which now exists as the derived map with
the same name and same contents (16 soul keys + `male` + `female`).

**Verify**: deferred to Step 4 (combined build after all edits).

### Step 3: Replace the balaur map literal with a derived map

Delete the entire `balaurAvatarMap` literal block — the comment line(s) and the
`var balaurAvatarMap = map[string]string{ ... }` currently spanning lines
126–143 (from `var balaurAvatarMap = map[string]string{` through the closing
`}` after the `balaur-16` / `// Sage` line).

Replace it with:

```go
// balaurAvatarMap is the key→URL lookup for Balaur heads, derived from the
// BalaurHeads roster (the single source of truth). No legacy aliases — the
// 16 balaur-NN keys are 1:1 with the roster.
var balaurAvatarMap = avatarMap(BalaurHeads())
```

Leave `BalaurHeads`, `ValidBalaurAvatarKey`, `BalaurAvatarURL`, and
`BalaurAvatarURLForKey` exactly as they are — they keep reading
`balaurAvatarMap`.

> Placement note: package-level `var` initializers can reference functions and
> other package-level vars regardless of source order — Go resolves init
> dependencies automatically. So it does not matter that `BalaurHeads()` is
> textually defined after `balaurAvatarMap`. No reordering needed.

**Verify**: deferred to Step 4.

### Step 4: Build, vet, and run the package tests

Run the full local gate for this package:

```
CGO_ENABLED=0 go build ./...
go vet ./...
gofmt -l internal/store/owner_settings.go
go test ./internal/store/...
```

**Verify**:
- `go build` → exit 0
- `go vet` → exit 0
- `gofmt -l ...` → prints nothing
- `go test ./internal/store/...` → all pass, **including** `TestAvatarRosters`,
  `TestValidAvatarKeysMatchRosters`, and `TestLegacySoulAvatarAliases`
  (the legacy-alias test is the proof the `male`/`female` aliases survived).

### Step 5: Full suite + lint

```
go test ./...
make lint
git diff --check
```

**Verify**:
- `go test ./...` → all pass
- `make lint` → exit 0 (staticcheck reports no U1000 dead symbol; `avatarMap` is
  used by both derived vars, and the literals are gone)
- `git diff --check` → no whitespace errors
- `git status --porcelain` shows only `internal/store/owner_settings.go` modified

## Test plan

No new tests are required — the existing tests in
`internal/store/owner_settings_test.go` already pin every behavior this change
must preserve, and they must pass **unchanged**:

- `TestAvatarRosters` — both slices still return 16 well-formed entries
  (proves the slices, the source of truth, are untouched).
- `TestValidAvatarKeysMatchRosters` — every roster key validates, `"nope"`
  does not (proves the derived maps contain exactly the roster keys and reject
  unknown keys).
- `TestLegacySoulAvatarAliases` — `male`/`female` still validate and resolve to
  soul-01 / soul-02, and the direct `soulAvatarMap["male"]` / `["female"]` reads
  return the right URLs (proves the legacy aliases survived the derivation and
  the map keeps its name).

Verification: `go test ./internal/store/...` → all pass (no test count change;
same tests, same names).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0; `internal/store` tests all pass, including
      `TestLegacySoulAvatarAliases`
- [ ] `make lint` exits 0 (no U1000 dead-code or unused-import failures)
- [ ] `gofmt -l internal/store/owner_settings.go` prints nothing
- [ ] No `map[string]string{` literal for avatars remains:
      `grep -nE 'soul-0[12]":\s*"/static' internal/store/owner_settings.go`
      returns matches ONLY inside `SoulAvatars()` (the slice), not a `map` block.
      (Quicker check: the file should now contain `avatarMap(SoulAvatars())` and
      `avatarMap(BalaurHeads())`, and exactly one occurrence each of
      `var soulAvatarMap` / `var balaurAvatarMap`.)
- [ ] `git status --porcelain` lists only `internal/store/owner_settings.go`
      as modified (no other source file, not the test file)
- [ ] `plans/readme.md` status row for plan 148 updated (unless a reviewer owns
      the index)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `internal/store/owner_settings.go` or its test changed
  since `ab2c0a9`, and the live code no longer matches the "Current state"
  excerpts above.
- `grep -rn "soulAvatarMap\|balaurAvatarMap" --include="*.go" .` shows a
  reference to either map **outside** `internal/store/owner_settings.go` and
  `internal/store/owner_settings_test.go`. (At plan time there are none. If a new
  caller appeared, the derivation must still preserve its expectations — stop and
  reassess.)
- The `SoulAvatars()` or `BalaurHeads()` slice no longer has 16 entries, or its
  keys/URLs differ from the excerpts (the source of truth drifted — your derived
  map would silently change behavior).
- `TestLegacySoulAvatarAliases` fails after your change — that means the
  `male`/`female` aliases were dropped. Do not edit the test to make it pass;
  fix the derivation so the aliases are present.
- `make lint` reports a U1000 unused symbol — likely the `avatarMap` helper is
  not actually referenced (you forgot to wire one of the derived vars), or a map
  literal was left behind alongside the derived var (name collision). Resolve by
  ensuring exactly one `var soulAvatarMap` and one `var balaurAvatarMap`, both
  using `avatarMap(...)`.
- A step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- **Adding/renaming an avatar is now a one-place edit**: append or edit an entry
  in `SoulAvatars()` or `BalaurHeads()` in `internal/store/owner_settings.go`.
  The lookup maps, validity checks, and URL resolvers all derive from those
  slices automatically. Do not reintroduce a hand-written map literal.
- **Legacy aliases live in the soul derivation only** — the
  `m["male"]`/`m["female"]` lines in the `soulAvatarMap` initializer. If a new
  alias is ever needed, add it the same way (alias-key → `m[<roster-key>]`), and
  add a line to `TestLegacySoulAvatarAliases`.
- **Reviewer should scrutinize**: that the derived `soulAvatarMap` contains
  exactly 18 keys (16 roster + 2 aliases) and `balaurAvatarMap` exactly 16, and
  that no map literal data was left behind to drift again. The existing tests
  already enforce the roster/alias contract — confirm they pass unchanged rather
  than relying on visual diff alone.
- Deferred (not in this plan): nothing. This is a self-contained dedupe with no
  follow-up. If `BalaurAvatarURLForKey` ever needs to also accept soul keys, that
  is a separate change.
