# Stage 5: Pre-merge deploy gate (manual rehearsal)

This is a **manual pre-merge gate, not a spawnable code ticket**. It has no
ticket frontmatter on purpose so the implement orchestration loop ignores it.
Shipping orphans any data left in the old IndexedDB vault, so this rehearsal is
mandatory before merging `001-directory-vault` to `main`. The owner accepted
this one-time manual migration in the grill (2026-07-24). The orchestrator owns
the merge; workers never push or merge.

Perform and report the result before merge:

- [ ] (a) Open the LIVE deployed site (https://alexradunet.github.io/balaur/) in a Chromium browser that has data present; use **Export whole space** to produce a version-2 `.orbit.json`.
- [ ] (b) Serve the feature branch locally (`python3 -m http.server 4173`), launch, and pick an **EMPTY** folder at the Vault gate.
- [ ] (c) **Import** the `.orbit.json` (restore replaces the picked folder's file tree; the existing confirm dialog warns the canonical vault is replaced).
- [ ] (d) Confirm the workspace renders complete: canvases, tasks, habits, journals.
- [ ] (e) Report the rehearsal result (pass/fail + any diagnostics) to the orchestrator before merge.

Dependencies: tickets 01–04 merged-ready on the `001-directory-vault` branch.
