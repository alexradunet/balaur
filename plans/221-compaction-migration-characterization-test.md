# Plan 221: Characterization test for `1750000070_conversation_compaction` — REJECTED (pure-schema migration, no row transform)

- **Status**: REJECTED — not worth doing.
- **Category**: tests
- **Planned at**: commit `ef9f2df`, 2026-06-30
- **Finding**: audit #12 / TEST-03 — "the one behavioral-looking migration without the up/down characterization test its siblings have."

## Verdict

No characterization test is warranted. The audit flagged
`1750000070_conversation_compaction` as the lone migration lacking the round-trip
test its siblings (`tasks_to_nodes`, `measures_to_nodes`, `journal_unify`) carry.
On inspection it is a **pure-schema** migration that transforms **zero rows**, so
the characterization-test pattern (seed rows → up → assert transform → down →
assert restore) does not apply — its behavior is fully covered by the schema
baseline snapshot (`migrations/schema_test.go`).

## Evidence (the whole migration)

`migrations/1750000070_conversation_compaction.go` — `up` only ADDS two fields to
the `conversations` collection; `down` removes them. No `FindRecords*`, no
per-row rewrite, no data movement:

```go
// up (lines 20-35): adds fields, idempotent, saves the collection
func upConversationCompaction(app core.App) error {
	col, _ := app.FindCollectionByNameOrId("conversations")
	if col.Fields.GetByName("summary") == nil {
		col.Fields.Add(&core.TextField{Name: "summary", Max: 20000})
	}
	if col.Fields.GetByName("compacted_through") == nil {
		col.Fields.Add(&core.DateField{Name: "compacted_through"})
	}
	return app.Save(col)
}
// down (lines 38-52): removes the two fields. Nothing else.
```

Its own doc comment confirms there is no row transform:
> "Messages are never deleted — compaction only summarises and advances the
> boundary" (`1750000070_conversation_compaction.go:18-19`).

The siblings that DO have round-trip tests genuinely rewrite rows — e.g.
`migrations/datamigrations_test.go:49 TestDataMigrationTasksRoundTrip`,
`:278 TestDataMigrationMeasuresRoundTrip`, `:454 TestDataMigrationJournalUnify` —
because `tasks`/`measures`/`journal` were *moved into the nodes spine* (data
migrations). Adding columns is not in that class.

## Why rejected rather than "write it anyway"

A characterization test for a field-add would assert only that two columns exist
after `up` and are gone after `down` — which `migrations/schema_test.go`
(`TestSchemaBaseline`) already covers by snapshotting the full schema. Writing a
second test for the same guarantee is duplicate coverage with no added safety,
against the repo's suckless/no-write-only-output rules.

## If this ever changes

If a future migration with this prefix is rewritten to actually transform
`conversations`/`messages` rows (e.g. backfilling `summary` from existing turns),
re-open this: add a round-trip test in `migrations/datamigrations_test.go`
modeled on `TestDataMigrationTasksRoundTrip` (seed pre-migration rows, run up,
assert the transform, run down, assert restore).
