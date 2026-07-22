# Domain docs

Balaur uses a single domain context. The root `CONTEXT.md` is the glossary; `docs/adr/` records accepted architectural decisions.

## Before exploring

Read `CONTEXT.md`, then read the ADRs relevant to the subsystem. Also read the subsystem documents required by `AGENTS.md`. If a document is absent, proceed without manufacturing one solely to satisfy the layout.

## Vocabulary

Use the glossary's exact terms in issue titles, specs, implementation plans, APIs, and tests. Avoid synonyms that collapse important distinctions such as entity versus placement, schedule versus deadline, or canonical data versus projection.

When a necessary term is missing or ambiguous, use `domain-modeling` through `grill-with-docs` to resolve it and update `CONTEXT.md`. Record a decision in `docs/adr/` when it establishes or changes an architectural boundary.

## Conflicts

Do not silently override an ADR. Surface the conflict, identify the ADR, and obtain an explicit architecture decision before implementing the contradiction.
