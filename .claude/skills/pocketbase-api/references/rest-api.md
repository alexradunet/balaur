# PocketBase REST reference (Balaur)

Depth reference for the `pocketbase-api` skill. All examples assume the `pb`
helper (`scripts/pb.sh`), which adds auth + base URL. Raw form is
`curl -H "Authorization: <superuser-token>" http://127.0.0.1:8090<path>`.

Contents:
1. Auth
2. Listing & search (pagination, sort, filter, fields, expand, skipTotal)
3. Filter grammar (operators, functions, dates)
4. View / create / update / delete a record
5. Batch writes
6. Files (upload, download)
7. Realtime (SSE)
8. Admin: collections, settings, logs, backups
9. Gotchas

---

## 1. Auth

Superuser login (what `pb` does for you):

```
POST /api/collections/_superusers/auth-with-password
body: {"identity":"claude@balaur.local","password":"…"}
→ {"token":"…","record":{…}}
```

Send the token as `Authorization: <token>` (a raw `Bearer ` prefix is also
accepted). Superuser tokens **bypass all collection API rules** — every record
in every collection is readable and writable. Owner (non-super) users live in
the `users` auth collection and would auth at
`/api/collections/users/auth-with-password`, but you normally act as superuser.

---

## 2. Listing & search

```
GET /api/collections/<collection>/records
```

Query params (combine freely):

| Param | Meaning | Example |
|---|---|---|
| `page` | 1-based page | `page=2` |
| `perPage` | page size (default 30, max 500) | `perPage=100` |
| `sort` | comma list; `-` = desc | `sort=-created,title` |
| `filter` | predicate (see §3) | `filter=(status='done')` |
| `fields` | projection; supports `expand.*` and `:excerpt(n)` | `fields=id,title,created` |
| `expand` | pull in relations (see below) | `expand=conversation` |
| `skipTotal` | `1` skips the count query (faster) | `skipTotal=1` |

Response shape:

```json
{"page":1,"perPage":30,"totalItems":42,"totalPages":2,"items":[ {…}, … ]}
```

**Count only:** `?perPage=1&skipTotal=0` then read `totalItems`.

**Expand** follows relation fields. `messages.conversation` is a relation, so
`GET /api/collections/messages/records?expand=conversation` adds an `expand`
object to each item: `{"id":…,"conversation":"REL_ID","expand":{"conversation":{…}}}`.
Nested/multi: `expand=conversation,task` or `expand=task.parent`. Max depth 6.
Expanded relations obey the *related* collection's view rule — as superuser,
moot.

---

## 3. Filter grammar

Operators:

| Op | Meaning |
|---|---|
| `=` `!=` | equal / not |
| `>` `>=` `<` `<=` | comparison |
| `~` `!~` | contains / not-contains (wildcards implied; `%` for explicit) |
| `?=` `?!=` `?~` … | "any of" variants for multi-value/relation fields |

Combine with `&&`, `||`, and parentheses:

```
filter=(status='active' && kind='chat')
filter=(due >= '2026-06-01' && due <= '2026-06-30')
filter=(title ~ 'plant' || notes ~ 'plant')
```

Values: single-quote strings; numbers bare; `true`/`false`/`null` bare.
Dates are UTC strings `'YYYY-MM-DD HH:MM:SS.sssZ'` (a date-only `'2026-06-01'`
compares fine). Macros are available: `@now`, `@todayStart`, `@todayEnd`,
`@monthStart`, etc. Reference fields by relation path: `filter=(task.status='done')`.

**URL-encoding:** inside a query string, encode `&&`→`%26%26`, `||`→`%7C%7C`,
`#`→`%23`. Easiest: wrap the whole URL in double quotes when calling `pb` and
keep `&&`/`||` literal only if there are no other `&` params; otherwise encode.

---

## 4. View / create / update / delete

```
GET    /api/collections/<c>/records/<id>          # one record (supports ?expand=…&fields=…)
POST   /api/collections/<c>/records   -d '{…}'    # create
PATCH  /api/collections/<c>/records/<id> -d '{…}' # partial update
DELETE /api/collections/<c>/records/<id>          # delete → 204, no body
```

- Send only the fields you set; PATCH is partial.
- Relation fields take the related record `id` (or array of ids for multi).
- Read-only/system fields (`id`, `created`, `updated`, `tokenKey`) are managed by
  PocketBase; don't set them. `id` may be supplied on create (15-char) but let
  PB generate it unless you have a reason.
- Auth-collection create/update: set `password` (and PB derives `tokenKey`).
- A `400` usually means validation: a `select` field given a value outside its
  option list, a missing `required` field, or a bad relation id. The response
  body's `data` map names the offending field — read it. Confirm options with
  `GET /api/collections/<c>` → `fields`.

---

## 5. Batch writes

Multiple create/update/delete in one transactional request (must be enabled in
settings; it is by default for superusers):

```
POST /api/batch
{"requests":[
  {"method":"POST","url":"/api/collections/tasks/records","body":{"title":"a","status":"todo"}},
  {"method":"PATCH","url":"/api/collections/tasks/records/ID","body":{"status":"done"}},
  {"method":"DELETE","url":"/api/collections/tasks/records/ID2"}
]}
```

All-or-nothing: any failed sub-request rolls back the whole batch.

---

## 6. Files

Upload is multipart (not JSON), so call `curl` directly with the token:

```
curl -H "Authorization: $TOKEN" -X PATCH \
  http://127.0.0.1:8090/api/collections/heads/records/ID \
  -F balaur_avatar=@/path/to/avatar.png
```

Download/serve a stored file:

```
GET /api/files/<collection>/<recordId>/<filename>?thumb=100x100   # thumb optional
```

For a protected file, append a short-lived token from
`POST /api/files/token` → `?token=…`.

---

## 7. Realtime (SSE)

```
GET /api/realtime           # SSE stream; first event delivers a clientId
POST /api/realtime          # {"clientId":"…","subscriptions":["messages","tasks/ID"]}
```

Subscribe to a whole collection or a single record id. Useful to watch
`messages` land during a `balaur chat` turn. Note Balaur's *own* UI uses
Datastar SSE separately — this is the PocketBase realtime API, for inspection.

---

## 8. Admin endpoints

| Purpose | Endpoint |
|---|---|
| List/inspect collections (schema) | `GET /api/collections` |
| One collection | `GET /api/collections/<idOrName>` |
| Create/alter/delete collection | `POST`/`PATCH`/`DELETE /api/collections[/<id>]` |
| App settings | `GET /api/settings`, `PATCH /api/settings` |
| Logs | `GET /api/logs`, `GET /api/logs/<id>`, `GET /api/logs/stats` |
| Backups | `GET/POST /api/backups`, `…/backups/<key>` (download/restore/delete) |

**Schema changes belong in migrations**, not ad-hoc `PATCH /api/collections`.
Balaur defines collections via files in `migrations/`; altering schema over REST
diverges the running DB from source. Read schema freely; change it in code.

---

## 9. Gotchas

- **Superuser bypasses rules** — there's no safety net on writes. Re-read §Posture
  in SKILL.md before mutating.
- **Secrets in responses** — `llm_providers.api_key` and auth `password`/`tokenKey`
  come back in record reads. Project them out (`fields=…`) and never paste them
  anywhere persistent.
- **`updated`/`created`** are PB-managed timestamps; sort/filter on them, don't write them.
- **Relations are ids**, not embedded objects — use `expand` to hydrate.
- **DELETE returns 204** with an empty body (the `pb` helper prints nothing and
  exits 0).
- **Date strings are UTC**; Balaur renders owner-timezone in the app, but the API
  stores/compares UTC.
