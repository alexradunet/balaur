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

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
