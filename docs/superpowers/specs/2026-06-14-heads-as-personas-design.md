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

- One conversation, full trust. Switching a head changes the voice/expertise
  (a system-prompt flavor), the avatar shown, and — optionally — which tools
  the head is offered.
- Give each head an optional **tool selection** (capability groups). Empty =
  all tools, so the main head and any unconfigured head behave exactly as
  today. This shapes a head's focus without rebuilding the security layer.
- Ship a small built-in roster — one **main** head plus three specialists —
  and let the owner create custom heads when they need them.
- Switch the active head from a picker in the chat dock, beside the model
  switcher. The choice persists as the owner's "current head".
- Remove the dormant autonomy/security machinery entirely, so the codebase
  gets simpler, not more layered.

## Non-goals (explicitly out of scope)

- **No autonomous sub-agents.** Heads never run on their own; there is no
  spawn-by-AI, no delegated background task.
- **No per-head data scoping/grants/audit (the security boundary).** A head
  can be offered a *subset of tools* (see "Per-head tool groups"), but that is
  capability shaping, not enforcement: the tools that *are* offered still run
  with the owner's full trust and full data access. An un-offered tool is
  simply absent from the model's tool list — there is no fail-closed check, no
  audit. The `grants` collection, `heads.Scoped`, and the audited rule boundary
  are deleted and **not replaced**.
- **No branch/separate conversations per head.** There is one `master`
  thread. Switching heads does not fork history. (`turn.RunFor`,
  `conversation.ForHead`, and the dock conversation-swap are deleted.)
- **No merge-back.** The unshipped `docs/head-tools-design.md` / plan 019
  described *grant-derived scoped tools* — a security model. That approach
  stays rejected; per-head tool groups here are a non-security filter, not
  grants, so the doc is removed.

## The model

A **head** is four fields:

- `name` — e.g. "Scholar"
- `purpose` — a short system-prompt flavor, e.g. "explains, researches, weighs
  trade-offs; precise and cites its reasoning"
- `balaur_avatar` — one of `balaur-01`…`balaur-16`
- `tools` — an optional list of capability-group keys (see below). Empty = all.

There is exactly one conversation (the existing `master`). The **active head**
is an owner setting. On each turn, `turn.Run` reads the active head and:

1. appends its `purpose` to the system prompt (the main head adds nothing),
2. filters the offered tool set to the head's groups (empty = all tools), and
3. renders the assistant's messages with the active head's `balaur_avatar`.

Memories, skills, history, and the honesty check are unchanged and shared. The
Balaur metaphor is exact: **one body (memory, history, data), many heads
(voices, each with the reach you give it).** Switching mid-conversation is fine
and intentional — it is the dragon turning a different head toward you over the
same shared memory.

### Built-in roster (in Go, always present, not editable)

The `tools` column lists each built-in's default capability groups (see
"Per-head tool groups"); empty means the full toolset.

| key       | role | purpose flavor | tools |
|-----------|------|----------------|-------|
| `balaur`  | **main** | general companion; uses the owner's default `balaur_avatar`; adds no extra flavor (the baseline system prompt) | *(all)* |
| `scholar` | specialist | explains, researches, weighs trade-offs; precise and cites its reasoning | `memory` |
| `planner` | specialist | turns goals into concrete tasks and steps; outcome-oriented | `tasks`, `memory` |
| `coach`   | specialist | accountability, reflection, journaling; warm and direct | `journal`, `life`, `memory` |

Built-ins are defined in code so they are always available and cannot be
deleted or corrupted. Their avatars and tool groups are fixed defaults
(assigned in the plan). Re-skinning or editing a built-in is **out of scope for
v1** — only custom heads are editable; built-ins are read-only.

### Custom heads (owner-created)

Stored as rows in the `heads` base collection. Created/deleted from the Heads
manage card (`name` + `purpose` + pick an avatar + tick capability groups). No
scoping, no auth, no expiry — just the four fields plus `created`.

### Per-head tool groups

The owner shapes a head's reach by selecting **capability groups**, not
individual tools. The groups map one-to-one onto the tool constructors that
`turn.Tools` already assembles (`internal/turn/tools.go`), so the mapping is
nearly free and self-maintaining — adding a tool to, say, `KnowledgeTools`
automatically reaches every head with the `memory` group.

| group        | constructor               | tools (today) |
|--------------|---------------------------|---------------|
| `memory`     | `tools.KnowledgeTools`    | remember, recall, skill, propose_skill |
| `tasks`      | `tools.TaskTools`         | task_add, task_list, task_done, task_snooze, task_drop |
| `life`       | `tools.LifeTools`         | log_entry, entry_series, entry_drop |
| `journal`    | `tools.JournalTools`      | journal_write |
| `os`         | `tools.OSAccess`          | read, write, edit, bash — only present when `BALAUR_OS_ACCESS=1` |
| `extensions` | `ext.ProposeTool` + `ext.Tools` | propose_extension + approved balaur-extensions |

**Always-on core** (every head, not selectable): `tools.ChoiceTools`
(offer_choices), `tools.UITools` (card/board composition), and `self.Tool`.
These are interaction and self-description tools, not domain mutations, so
every head keeps them. `self.Tool` is built from the head's *own* filtered tool
names, so a narrowed head reports only the capabilities it actually has and
never claims a tool it wasn't given.

**Empty selection = all groups** — identical to today's behavior. This is the
default for the main head and any custom head the owner leaves unconfigured.

**This is a filter, not a sandbox.** A group the owner omits is simply not
offered to the model that turn; the tools that *are* offered still execute with
full owner trust. Only the two interactive chat gateways run a turn
(`internal/web/chat.go`, `internal/cli/chat.go`), so adopting the active head's
filter never narrows an agent-initiated message.

## Data model & migration

**Approach #1 (approved): clean teardown + a tiny base collection.** Because
there is zero head/grant/branch data, this is risk-free.

One **new** migration (historical migrations are not rewritten):

- **Drop** the `grants` collection.
- **Drop and recreate** `heads` as a *base* collection with fields:
  `name` (text, required), `purpose` (text), `balaur_avatar` (text),
  `tools` (JSON — an array of capability-group keys; empty/absent = all), and
  the default `created`/`updated`. The old `heads` was an `AuthCollection`
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
  its `purpose` to the system prompt, filter the tool set to the head's groups,
  and expose its avatar for rendering. This is the single integration point.
  (`turn.RunFor` is deleted; head turns now flow through `Run` and therefore
  get the honesty check they previously skipped — a net improvement.)
- **Tool filtering** lives in `internal/turn/tools.go` as a new
  `ToolsForHead(app, groups)` beside the existing `Tools(app)`: empty `groups`
  returns `Tools(app)` verbatim (zero behavior change); otherwise it assembles
  the always-on core plus the selected group constructors, and builds
  `self.Tool` from the resulting names. `Run` calls `ToolsForHead` instead of
  `Tools`.
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
  avatar + ~6 capability-group checkboxes). The group checkboxes are the only
  new control versus the avatar picker the card already had. The card stays in
  the typed card registry (`internal/cards/cards.go`).
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
  collection with the four fields (`name`, `purpose`, `balaur_avatar`,
  `tools`), and the branch `conversations` fields/indexes are gone.
- Active-head resolution: built-in key resolves; custom id resolves; deleted
  custom id falls back to `balaur`; default is `balaur`.
- `turn.Run` system-prompt injection: a specialist's `purpose` appears in the
  prompt; the main head adds nothing.
- `ToolsForHead` filtering: empty groups returns the full `Tools(app)` set; a
  head with `["memory"]` is offered `recall` but not `task_add`; the always-on
  core (offer_choices, UI, self) is present regardless; `self.Tool` reports
  only the head's filtered tool names.
- Dock: switcher renders the active head and the roster; selecting a head sets
  `active_head` and re-renders the dock avatar/label without swapping the
  conversation.
- Custom head CRUD: create and delete a custom head with chosen tool groups;
  built-ins cannot be deleted.

## Open questions

None blocking. Built-in avatar assignments and exact `purpose` wording are
chosen in the implementation plan; they do not affect the architecture.
