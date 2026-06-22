---
name: go-standards
description: Use when writing, reviewing, or refactoring Go in Balaur (anything under internal/ or main.go) — to apply the repo's Go idioms, tooling, and conventions. Covers error handling (%w, errors.Is/As/Join), context threading, the modern stdlib (slices/maps/cmp, min/max, for range int), structured logging via app.Logger() (slog), the gomponents html alias + g.Text-vs-g.Raw escaping rule, PocketBase patterns (records-as-domain-model, app.Save bypasses API rules by design, GetOrSet for check-then-act, audit-after-save), owner-timezone cron math, the suckless/dead-code rules, the testing idioms (table-driven, t.TempDir/Cleanup/Context, fake llm.Client, no time.Sleep), and the gofmt/vet/staticcheck/govulncheck/modernize tool surface.
---

# Balaur Go standards

This is the Go-idioms checklist for Balaur. AGENTS.md is the law; this skill is
the working detail for writing and reviewing Go. Read AGENTS.md first.

**Announce at start:** "Using the go-standards skill."

## Before you finish any Go change, run the gates

- `gofmt -l .` (empty), `go vet ./...`, `go test ./...`, `CGO_ENABLED=0 go build ./...`
- `make lint` (gofmt + vet + staticcheck + test) and, for dependency work,
  `make vulncheck` (govulncheck). Keep staticcheck CLEAN — dead code and
  deprecated APIs are build failures, not review nits.
- `git diff --check`.

## Errors

- Wrap with `fmt.Errorf("doing x: %w", err)` — `%w`, not `%v`, unless you are
  deliberately flattening at a boundary or formatting a recovered `any`.
- Error strings: lowercase, no trailing punctuation.
- Use `errors.Is`/`errors.As` to inspect; `errors.Join` to accumulate.
- Don't log AND return the same error (double-handling) — return it; let the
  top of the turn/handler log once via `app.Logger()`.

## Modern stdlib (Go 1.26)

- `slices.Contains`/`SortStableFunc`/`Reverse`/`Backward`, `maps.*`, `cmp.Compare`
  instead of hand-rolled membership/sort/reverse loops.
- `min`/`max`/`clear` builtins; `for range int` for counting loops; `strings.Cut`.
- `any`, never `interface{}`.
- Periodically run `modernize ./internal/...` and apply `-fix` (it is
  behavior-preserving); skip `migrations/` (frozen).

## Logging

- `app.Logger().Info/Warn/Error(msg, "key", val, …)` — structured slog. No
  `log.Printf`/`fmt.Print*` in service code; `log.Fatal` only in `main`.

## Context & concurrency

- `context.Context` is the first parameter, threaded to all IO/LLM/subprocess
  calls; never stored in a struct. Honor cancellation on every channel send
  (route sends through a ctx-guarded helper — see internal/kronk/client.go).
- Stop tickers/timers (`time.NewTimer` + `defer Stop`, not `time.After` in a
  select). No goroutine without a stop path. CI runs `-race`.
- Check-then-act on `app.Store()` or records is a race — use `GetOrSet` /
  retry-on-conflict, never `GetOk`+`Set` or read-modify-write.

## PocketBase & data

- Domain packages own their own PocketBase reads/writes — records ARE the domain
  model (not "missing a repository layer"). `internal/store` is for cross-cutting
  concerns only (audit, owner settings, llm config, time).
- `app.Save`/`app.Find*` bypass collection API rules BY DESIGN — code that writes
  on the owner's behalf is trusted. Keep mutations owner-initiated and audited.
- Audit strictly AFTER a successful write. Redact secrets (API keys) from audit
  entries and logs.
- Wall-clock / per-day cron math uses `time.Now().In(store.OwnerLocation(app))`,
  not bare `time.Now()`.

## gomponents UI

- Alias the html package as `h "maragu.dev/gomponents/html"`; gomponents core as
  `g`; datastar as `data`. Do not dot-import.
- User/model text → escaping `g.Text`. `g.Raw` only for already-trusted,
  already-rendered component HTML.
- Never hand-roll markup — compose from `internal/ui` / `internal/feature/*cards`
  and keep the storybook in sync (see the `ui-development` skill).

## Tests

- Standard `testing`, table-driven, no assertion frameworks. `t.TempDir`,
  `t.Cleanup`, `t.Context`. Fake the `llm.Client` (internal/llmtest); store tests
  use internal/store / internal/storetest temp-dir apps.
- No `time.Sleep` for synchronization. Assertions must check something real
  (staticcheck SA4006 catches assign-but-never-used).

## Suckless

- Delete dead code rather than commenting it out; one source of truth per
  concern; copy 30 lines before importing 3000. Every new dependency must justify
  itself.
