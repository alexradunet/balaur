# Heads as switchable personas — design

> **Status: approved design, not yet implemented.** Brainstormed and approved
> 2026-06-14. Replaces the dormant multi-head sub-agent machinery with a KISS
> persona model. The next step is an implementation plan (writing-plans).

## Why

Balaur ("multi-headed dragon") shipped a sub-agent feature: each *head* is an
auth record in the `heads` collection, its permissions are `grants` rows,
access is mediated by `heads.Scoped` and written to `audit_log`, and a head
runs a *branch* conversation through `turn.RunFor`. The intent was scoped,
auditable autonomous sub-agents.

In practice the machinery is **dormant**. As of 2026-06-14 the live DB holds
**0 heads, 0 grants, and 1 conversation (`kind='master'`)**. Across the whole
repo, `heads.Spawn` is called *only from tests* — there is no UI button, CLI
command, or LLM tool that creates a head at runtime. The feature is complete
scaffolding that nothing drives.

We want to **keep the lore and lose the complexity**. The multi-headed dragon
is core to Balaur's identity, and the owner may want specialized voices
(a researcher, a planner, a coach). But the *autonomy + security* layer
(grants, scoped data path, audit actors, spawn/merge/revoke, branch
conversations) is expensive and unused. The lore — a name, a purpose, one of
the 16 `balaur-NN` avatars — is nearly free.

So: a head becomes a **persona you switch to**, not a sandboxed worker.

## Goals

- One conversation, full tools, full trust. Switching a head changes only the
  voice/expertise (a system-prompt flavor) and the avatar shown.
- Ship a small built-in roster — one **main** head plus three specialists —
  and let the owner create custom heads when they need them.
- Switch the active head from a picker in the chat dock, beside the model
  switcher. The choice persists as the owner's "current head".
- Remove the dormant autonomy/security machinery entirely, so the codebase
  gets simpler, not more layered.

## Non-goals (explicitly out of scope)

- **No autonomous sub-agents.** Heads never run on their own; there is no
  spawn-by-AI, no delegated background task.
- **No per-head scoping/grants/audit.** Every head shares the owner's full
  tool set and trust. The `grants` collection and `heads.Scoped` are deleted.
- **No branch/separate conversations per head.** There is one `master`
  thread. Switching heads does not fork history. (`turn.RunFor`,
  `conversation.ForHead`, and the dock conversation-swap are deleted.)
- **No merge-back, no scoped head tools.** The unshipped `docs/head-tools-design.md`
  / plan 019 design becomes irrelevant and is removed.

## The model

A **head** is three fields:

- `name` — e.g. "Scholar"
- `purpose` — a short system-prompt flavor, e.g. "explains, researches, weighs
  trade-offs; precise and cites its reasoning"
- `balaur_avatar` — one of `balaur-01`…`balaur-16`

There is exactly one conversation (the existing `master`). The **active head**
is an owner setting. On each turn, `turn.Run` reads the active head and:

1. appends its `purpose` to the system prompt (the main head adds nothing), and
2. renders the assistant's messages with the active head's `balaur_avatar`.

Tools, memories, skills, history, and the honesty check are unchanged and
shared. The Balaur metaphor is exact: **one body (memory, history, tools),
many heads (voices).** Switching mid-conversation is fine and intentional —
it is the dragon turning a different head toward you over the same shared
memory.

### Built-in roster (in Go, always present, not editable)

| key       | role | purpose flavor |
|-----------|------|----------------|
| `balaur`  | **main** | general companion; uses the owner's default `balaur_avatar`; adds no extra flavor (the baseline system prompt) |
| `scholar` | specialist | explains, researches, weighs trade-offs; precise and cites its reasoning |
| `planner` | specialist | turns goals into concrete tasks and steps; outcome-oriented |
| `coach`   | specialist | accountability, reflection, journaling; warm and direct |

Built-ins are defined in code so they are always available and cannot be
deleted or corrupted. Their avatars are fixed defaults (assigned in the plan).
Re-skinning or editing a built-in is **out of scope for v1** — only custom
heads are editable; built-ins are read-only.

### Custom heads (owner-created)

Stored as rows in the `heads` base collection. Created/deleted from the Heads
manage card (`name` + `purpose` + pick an avatar). No scoping, no auth, no
expiry — just the three fields plus `created`.

## Data model & migration

**Approach #1 (approved): clean teardown + a tiny base collection.** Because
there is zero head/grant/branch data, this is risk-free.

One **new** migration (historical migrations are not rewritten):

- **Drop** the `grants` collection.
- **Drop and recreate** `heads` as a *base* collection with fields:
  `name` (text, required), `purpose` (text), `balaur_avatar` (text), and the
  default `created`/`updated`. The old `heads` was an `AuthCollection`
  (email/password/tokens/status/expires) — all of that is gone.
- **Drop** the branch-only `conversations` machinery added by
  `1750720000_conversation_indexes.go`: indexes
  `idx_conversations_open_branch_head`, `idx_conversations_open_master`,
  `idx_conversations_head`, and the now-unused `conversations` fields
  `kind`, `head`, `parent`. (Keep `conversations` itself, the single master
  row, `messages`, and `summaries`.)
- Owner-rules on the recreated `heads` collection follow the existing
  owner-only pattern (`@request.auth.collectionName = users`).

**Kept:** `conversations` (master), `messages`, `audit_log` (it records
non-head actions too — only the head-specific `head:`-actor writes go away
because the functions that made them are deleted), `owner_settings`,
`memories`, `skills`.

**Active-head setting.** Store `active_head` in `owner_settings`. Its value is
either a **built-in key** (`balaur` | `scholar` | `planner` | `coach`) or a
**custom head record id**. Resolution: if the value matches a built-in key,
use the built-in; else look up the record; if neither (e.g. the active custom
head was deleted), **fall back to `balaur`**. Default is `balaur`.

## Runtime wiring

- **`turn.Run`** gains active-head awareness: resolve the active head, append
  its `purpose` to the system prompt, and expose its avatar for rendering.
  This is the single integration point. (`turn.RunFor` is deleted; head turns
  now flow through `Run` and therefore get the honesty check they previously
  skipped — a net improvement.)
- **Avatar resolution** generalizes today's behavior: the main `balaur` head
  uses the owner's `balaur_avatar` setting (unchanged from today); specialists
  and custom heads use their own `balaur_avatar`.
- **Chat gateway** (`internal/web/chat.go`) is unchanged in shape — it still
  calls `turn.Run` for the one conversation. The head-specific gateway
  (`headChat`) is deleted.

## UI

- **Dock head switcher.** A small picker in the chat dock, beside the existing
  model switcher (`home.html` `model_switcher`). It lists the built-ins +
  custom heads, shows the active head's name/avatar, and on select POSTs to a
  new lightweight endpoint that sets `active_head` and re-renders the dock
  region (avatar + label). No conversation swap — same thread.
- **Heads manage card.** The existing heads card is repurposed from
  "roster of active head *records* with avatar picker + open-chat" to a
  persona manager: list built-ins (read-only) and customs (editable/deletable),
  show which is active, "make active", and "+ New head" (name + purpose +
  avatar). The card stays in the typed card registry (`internal/cards/cards.go`).
- **Removed UI:** the "Open chat →" branch button, the per-head branch dock,
  and the dock back-button/branch label in `home.html`/`dock_convo`.

## Removal & edit inventory (file-by-file)

**Delete whole file:**

- `internal/heads/heads.go`, `internal/heads/scoped.go`,
  `internal/heads/heads_test.go` (the entire `internal/heads` package)
- `internal/web/headsmgmt.go` (`headChat`, `setHeadAvatar`,
  `messageViewsForHead`, `headView`, `headViewFrom`)
- `web/templates/heads-focus.html` (the `head_card` branch partial)
- `docs/head-tools-design.md` (unshipped scoped-tools design, now irrelevant)

**Edit out the head/branch parts (keep the file):**

- `internal/conversation/conversation.go` — remove `ForHead` and branch logic;
  keep `Master`, `Append`, `AppendOrigin`, `RecentTurns`, history helpers.
- `internal/turn/turn.go` — remove `RunFor`; add active-head purpose injection
  into `Run`'s system prompt.
- `internal/web/dock.go` + `dock_test.go` — remove the branch-swap path from
  `dockConversation` (and its tests); add the head-switcher action/tests.
- `internal/web/cards.go` — rework `renderCardHeads` / `cardHeadsView` /
  `cardHeadsManageView` into the persona manager (built-ins + customs +
  make-active + create/delete).
- `internal/web/web.go` — remove routes `POST /ui/heads/{id}/chat` and
  `POST /ui/heads/{id}/avatar`; add `set active head` + custom-head CRUD routes.
- `internal/web/models.go` — `homeData`: drop `ConvPostURL`/`ConvHeadName`/
  `ConvBack` (chat always posts to `/ui/chat`); add active-head name/avatar for
  the dock.
- `internal/store/owner_settings.go` — remove the per-record
  `HeadBalaurAvatarURL`/`SetHeadBalaurAvatar`; add active-head resolution +
  avatar helpers. Keep `SoulAvatars`, `OwnerName`, `BalaurAvatarURL`, and the
  `BalaurHeads` 16-avatar roster (now the avatar choices for heads).
- `web/templates/home.html` — add the head switcher; simplify `dock_convo`
  (no branch back-button); `chat_draft` always posts to `/ui/chat`.
- `web/templates/cards.html` — repurpose `ucard_heads` / `ucard_heads_manage`.
- `web/static/basm.css` — keep `.head-*` classes used by the manage card;
  drop branch-chat-only rules (`.chatbar-back-link`, `.head-name-label`) if
  unused after the edit.
- `internal/cli/doctor.go` — remove `grants` from `coreCollections` (keep
  `heads`).

**Docs/identity to update for consistency:**

- `AGENTS.md` — keep the general "Go-side access bypasses API rules, be
  careful" caution; remove the head-grants-specific "rule boundary is sacred"
  passage (its subject — `Scoped`/`grants` — is gone).
- `README.md`, `DESIGN.md`, `internal/self/knowledge.md` — describe heads as
  **switchable personas**, not autonomous scoped sub-agents; drop the
  branch-conversation and merge-back/scoped-tools roadmap items.

**Keep unchanged:** `internal/cards/cards.go` Spec registry entry for the
heads card (label/params may get a minor tidy), `internal/cli/audit.go`
(the audit `head` field stays harmlessly), `plans/readme.md` history.

**Historical migrations are not edited or deleted.** `1749600000_init.go`
(auth `heads` + `grants`), `1750600000_head_avatar.go`, and
`1750720000_conversation_indexes.go` stay as immutable history. The single
**new** migration runs after them and supersedes them: on a fresh DB the
sequence creates the auth model and then tears it down to the persona model;
on the existing DB the new migration is the only one that runs. Within the new
migration the teardown order matters — drop `grants` and the
`conversations.head`/`kind`/`parent` fields (removing FKs to `heads`) before
dropping and recreating `heads` as a base collection.

## Edge cases

- **Active custom head deleted** → resolution falls back to `balaur`; the dock
  reflects the main head on next render.
- **Switching mid-conversation** → allowed; only subsequent turns use the new
  voice/avatar. History is shared and unchanged.
- **Empty `purpose`** (the main head, or a sparse custom head) → no flavor line
  is appended; baseline system prompt only.
- **Streaming guard** → the dock head switcher is disabled while a turn streams
  (mirrors the existing `$streaming` gate on the model switcher).

## Testing

**Delete:** `internal/heads/heads_test.go`; the branch/head cases in
`dock_test.go`; any `RunFor`/`ForHead`/`headChat` tests in the web and turn
suites; `TestHeadsFocus`/`TestHeadChat`/`seedHeadRec`-style helpers that target
the old model.

**Add:**

- Migration test: after migrating, `grants` is gone, `heads` is a base
  collection with the three fields, and the branch `conversations`
  fields/indexes are gone.
- Active-head resolution: built-in key resolves; custom id resolves; deleted
  custom id falls back to `balaur`; default is `balaur`.
- `turn.Run` system-prompt injection: a specialist's `purpose` appears in the
  prompt; the main head adds nothing.
- Dock: switcher renders the active head and the roster; selecting a head sets
  `active_head` and re-renders the dock avatar/label without swapping the
  conversation.
- Custom head CRUD: create and delete a custom head; built-ins cannot be
  deleted.

## Open questions

None blocking. Built-in avatar assignments and exact `purpose` wording are
chosen in the implementation plan; they do not affect the architecture.
