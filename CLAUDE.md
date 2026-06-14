# CLAUDE.md

Balaur's canonical agent instructions live in **`AGENTS.md`** (the cross-tool
standard, shared with Codex and other agents). It is the single source of
truth — read it first. This file only imports it and adds Claude Code–specific
notes so the two never drift.

@AGENTS.md

## Claude Code notes

- **Build/test surface** (see `Makefile`): `make run` (single run), `make dev`
  (hot reload via air), `make build` (CGO-free), `make test` → `go test ./...`,
  `make vet`, `make fmt` (gofmt). Builds must pass with `CGO_ENABLED=0`.
- **Formatting** is enforced as gofmt (`make fmt` fails on unformatted files).
  A `PostToolUse` hook runs `gofmt -w` on every Go file I edit, so edits stay
  clean automatically.
- **Design reference**: `DESIGN.md` and `Balaur.html` capture the visual/system
  design; web UI is server-rendered gomponents + Datastar (no Node build step).
