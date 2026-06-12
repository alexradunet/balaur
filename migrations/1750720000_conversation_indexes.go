package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(conversationIndexesUp, conversationIndexesDown)
}

func conversationIndexesUp(app core.App) error {
	// Step 1: Dedupe before adding unique indexes. A live box may already
	// hold duplicates that would cause index creation to fail.

	// Dedupe open branch conversations per head: keep the oldest (lowest
	// created), set status='merged' on the rest.
	branchDups, err := app.FindRecordsByFilter("conversations",
		"kind = 'branch' && status = 'open'", "created", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("loading branch conversations: %w", err)
	}
	seenBranchHead := map[string]bool{}
	for _, rec := range branchDups {
		headID := rec.GetString("head")
		if seenBranchHead[headID] {
			// This is a duplicate — keep oldest (first seen), merge the rest.
			rec.Set("status", "merged")
			if err := app.Save(rec); err != nil {
				return fmt.Errorf("merging duplicate branch conversation %s: %w", rec.Id, err)
			}
		} else {
			seenBranchHead[headID] = true
		}
	}

	// Dedupe open master conversations: keep the oldest, merge the rest.
	// (There should be at most one, but be defensive.)
	masterDups, err := app.FindRecordsByFilter("conversations",
		"kind = 'master' && status = 'open'", "created", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("loading master conversations: %w", err)
	}
	keptMaster := false
	for _, rec := range masterDups {
		if keptMaster {
			rec.Set("status", "merged")
			if err := app.Save(rec); err != nil {
				return fmt.Errorf("merging duplicate master conversation %s: %w", rec.Id, err)
			}
		} else {
			keptMaster = true
		}
	}

	// Step 2: Add indexes on the conversations collection.
	conversations, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		return fmt.Errorf("conversations: %w", err)
	}
	// Unique partial index: at most one open branch conversation per head.
	conversations.AddIndex("idx_conversations_open_branch_head", true, "head", "kind = 'branch' AND status = 'open'")
	// Unique partial index: at most one open master conversation.
	conversations.AddIndex("idx_conversations_open_master", true, "kind", "kind = 'master' AND status = 'open'")
	// Non-unique covering index for non-open lookups.
	conversations.AddIndex("idx_conversations_head", false, "head", "")
	if err := app.Save(conversations); err != nil {
		return fmt.Errorf("saving conversations indexes: %w", err)
	}

	// Step 3: Add done_at index for life/day.go range query.
	tasks, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		return fmt.Errorf("tasks: %w", err)
	}
	tasks.AddIndex("idx_tasks_done_at", false, "status, done_at", "")
	if err := app.Save(tasks); err != nil {
		return fmt.Errorf("saving tasks done_at index: %w", err)
	}

	return nil
}

func conversationIndexesDown(app core.App) error {
	// Note: the deduplication (merging duplicate conversations) performed in
	// Up is intentionally NOT reversed here — restoring duplicate open records
	// would be unsafe and is not needed for rollback correctness.

	conversations, err := app.FindCollectionByNameOrId("conversations")
	if err == nil {
		conversations.RemoveIndex("idx_conversations_open_branch_head")
		conversations.RemoveIndex("idx_conversations_open_master")
		conversations.RemoveIndex("idx_conversations_head")
		app.Save(conversations)
	}

	tasks, err := app.FindCollectionByNameOrId("tasks")
	if err == nil {
		tasks.RemoveIndex("idx_tasks_done_at")
		app.Save(tasks)
	}

	return nil
}
