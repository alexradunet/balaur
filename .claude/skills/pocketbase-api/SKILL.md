---
name: pocketbase-api
description: >-
  Use to inspect, query, verify, or manipulate Balaur's data through its
  PocketBase API as the `claude` superuser. Reach for this WHENEVER you need to
  see or touch the real records — checking what's in a collection (conversations,
  messages, tasks, memories, skills, heads, llm_models/providers/settings,
  entries, summaries, audit_log, users, …), confirming a feature actually wrote
  the rows it should after you built it, debugging why the UI shows wrong/empty
  data, reading the audit log, counting or filtering records, or doing a
  one-off data fixup. Also covers when to use the stable `balaur` CLI instead of
  raw REST. Use it even when the user just says "check the database", "is it
  saved", "look at the conversations", "what's in tasks", "did that persist", or
  names any collection — don't hand-roll curl or guess the schema; this skill
  has the auth, the endpoints, and the live collection map.
---

# Balaur's PocketBase API

Balaur **is** a PocketBase app: every durable thing — conversations, the tasks
you commit to, memories, skills, model config, the audit log — is a row in an
inspectable SQLite-backed collection. You have a dedicated superuser
(`claude@balaur.local`) so you can look at and operate on that data directly.

Why this matters: the fastest way to *know* whether a change worked is to read
the records it produced, not to re-reason about the code. After you build a
feature, this is how you confirm it persisted the right rows. When the UI looks
wrong, this is how you see whether the bug is in the data or the rendering.

## Posture: inspect and verify first, mutate deliberately

Default to **reading**. Querying, counting, filtering, verifying — do that
freely; it can't hurt anything.

Your superuser token **bypasses every collection API rule** (that's what
superusers are for), so writes are unguarded — there's no rule layer to catch a
mistake. So:

- **Product logic goes through the app, not raw REST.** If the owner wants a
  task added or a memory approved, that's a `balaur` CLI command or an app
  action — they run the real pipeline (hooks, validation, audit). Raw `PATCH`
  skips all of it. Use REST writes for **inspection-driven fixups, test setup,
  and debugging**, not as a back door around the app's own invariants.
- **Keep mutations owner-initiated and auditable.** You're acting on the
  owner's behalf; don't invent data they didn't ask for.
- **Don't leak secrets.** `llm_providers.api_key`, and the `password`/`tokenKey`
  fields on auth collections, must never land in logs, commits, or anything that
  leaves the box. Project them out of your queries when you don't need them.

## Prerequisite: the server must be running

These calls hit `http://127.0.0.1:8090`. If nothing answers, start Balaur first
(see the **run-balaur** skill). Quick check:

```bash
curl -sf -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8090/ || echo down
```

## The `pb` helper — authenticated calls in one line

`scripts/pb.sh` authenticates as the `claude` superuser (reading the gitignored
creds at `.claude/balaur-claude-superuser.json`), caches the token, refreshes it
automatically on a 401, and forwards everything to `curl`. Use it for every
call instead of repeating the auth dance:

```bash
pb=.claude/skills/pocketbase-api/scripts/pb.sh

$pb GET /api/collections                                          # the live schema
$pb GET "/api/collections/tasks/records?perPage=5&sort=-created"  # 5 newest tasks
$pb GET "/api/collections/conversations/records?filter=(status='active')&fields=id,title,updated"
$pb POST /api/collections/tasks/records -d '{"title":"smoke test","status":"open"}'
$pb PATCH /api/collections/tasks/records/RECORD_ID -d '{"status":"done"}'
$pb DELETE /api/collections/tasks/records/RECORD_ID
```

Output is pretty-printed JSON. Non-2xx prints the body and exits non-zero.

## REST cheatsheet

| Goal | Call |
|---|---|
| List collections (live schema, source of truth) | `GET /api/collections` |
| List records | `GET /api/collections/<c>/records` |
| Newest first, 5 rows | `…/records?sort=-created&perPage=5` |
| Filter | `…/records?filter=(status='active' %26%26 kind='chat')` |
| Only some fields | `…/records?fields=id,title,created` |
| Just the count | `…/records?perPage=1&skipTotal=0` → read `totalItems` |
| Expand a relation | `…/records?expand=conversation` |
| One record | `GET /api/collections/<c>/records/<id>` |
| Create | `POST …/records -d '{…}'` |
| Update | `PATCH …/records/<id> -d '{…}'` |
| Delete | `DELETE …/records/<id>` |

Filter operators, multi-record batch writes, file upload/download, realtime SSE,
and settings/logs/backups endpoints are in **`references/rest-api.md`** — read it
when a query needs more than the cheatsheet (e.g. `~` contains, `||`, date
ranges, `expand` chains, `/api/batch`).

> URL-encode `&&` as `%26%26` and `||` as `%7C%7C` inside a `filter=` query
> string, or wrap the whole URL in quotes and let the shell pass it literally.

## Collection map (this app)

Domain collections (the product data you'll actually touch):

| Collection | Holds | Notable fields |
|---|---|---|
| `conversations` | Chat threads | title, kind, status |
| `messages` | Turns within a conversation | conversation→, role, content, tool_name, tool_payload, origin |
| `summaries` | Recap/period summaries | conversation→, period_type, period_start/end, content |
| `tasks` | Commitments | title, notes, status, due, recur, snoozed_until, done_at, source |
| `entries` | Life-log datapoints | kind, task→, value, value_num, unit, text, noted_at |
| `memories` | Memory lifecycle | title, content, status, category, importance, when_to_use, use_count |
| `skills` | Skill lifecycle | name, description, content, status, when_to_use, use_count |
| `heads` | Personas | name, purpose, balaur_avatar, capability_groups |
| `llm_providers` | Model providers | name, kind, base_url, **api_key (secret)**, enabled |
| `llm_models` | Configured models | provider→, label, chat_model, embed_model, enabled |
| `llm_settings` | Active-model pointer | key, active_model |
| `owner_settings` | Owner key/value settings | key, value |
| `extensions` | balaur-extensions ledger | name, path, sha256, status, tools, source |
| `audit_log` | The deeds ledger | actor, action, target, detail, allowed, created |
| `users` | Owner auth records | email, name, verified (auth collection) |

System collections (`_superusers`, `_authOrigins`, `_externalAuths`, `_mfas`,
`_otps`) are PocketBase internals — `_superusers` is where your own account
lives. Treat them as read-mostly.

This table can drift as schema evolves — `GET /api/collections` is always the
truth. Use it to confirm field names **and allowed values** before a write:
`select` fields reject anything outside their option list (e.g.
`tasks.status` ∈ `open|done|dropped`), so a `400` on create/update usually
means a bad enum value or a missing required field — read the collection's
`fields` to see what's valid.

## When to use the `balaur` CLI instead

Raw REST sees the rows. The **`balaur` CLI** runs the real domain semantics and
emits a stable `{"v":1,"kind":"…","data":…}` JSON envelope — it's the project's
documented interface for agents and test harnesses. Prefer it when you want the
*app's behavior*, not just the table:

- `balaur doctor` — preflight before driving a box (data dir, collections, model, gates).
- `balaur chat "<msg>"` — one real companion turn (context, agent loop, honesty check, persistence); reports tool calls + the words-vs-deeds verdict. Needs a model — use the fake-model harness for determinism (see **run-balaur**).
- `balaur task add/list/done/snooze/drop`, `balaur memory …`, `balaur skill …`, `balaur life …`, `balaur journal/day`, `balaur recap …` — domain operations with validation + audit.
- `balaur history`, `balaur audit [--action] [--actor]`, `balaur verify` — read the persisted turn, the deeds ledger, words-vs-deeds.
- `balaur self`, `balaur model`, `balaur ext …` — capability inventory, model choices, extension lifecycle.

Rule of thumb: **verifying or fixing up rows → REST (`pb`); exercising or
mutating product behavior → CLI.** Full command/flag table lives in the
README's "CLI for agents & test harnesses" section; every command also works on
a throwaway `--dir "$(mktemp -d)"`.

## References

- `references/rest-api.md` — the full PocketBase REST surface: filter grammar,
  pagination, expand, fields projection, batch, files, realtime, collections/
  settings/logs/backups admin. Read it when the cheatsheet isn't enough.
