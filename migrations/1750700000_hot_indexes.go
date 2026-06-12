package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(hotIndexesUp, hotIndexesDown)
}

func hotIndexesUp(app core.App) error {
	// Index messages by origin+created for BriefedToday and nudge polling.
	messages, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		return fmt.Errorf("messages: %w", err)
	}
	messages.AddIndex("idx_messages_origin_created", false, "origin, created", "")
	messages.AddIndex("idx_messages_conv_created", false, "conversation, created", "")
	if err := app.Save(messages); err != nil {
		return fmt.Errorf("saving messages indexes: %w", err)
	}

	// Index tasks for DueForNudge: status, nudged_at (equality), then due (range).
	tasks, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		return fmt.Errorf("tasks: %w", err)
	}
	tasks.AddIndex("idx_tasks_nudge", false, "status, nudged_at, due", "")
	if err := app.Save(tasks); err != nil {
		return fmt.Errorf("saving tasks indexes: %w", err)
	}

	// Index audit_log by actor for balaur audit --actor.
	auditLog, err := app.FindCollectionByNameOrId("audit_log")
	if err != nil {
		return fmt.Errorf("audit_log: %w", err)
	}
	auditLog.AddIndex("idx_audit_actor", false, "actor", "")
	if err := app.Save(auditLog); err != nil {
		return fmt.Errorf("saving audit_log indexes: %w", err)
	}

	return nil
}

func hotIndexesDown(app core.App) error {
	messages, err := app.FindCollectionByNameOrId("messages")
	if err == nil {
		messages.RemoveIndex("idx_messages_origin_created")
		messages.RemoveIndex("idx_messages_conv_created")
		app.Save(messages)
	}

	tasks, err := app.FindCollectionByNameOrId("tasks")
	if err == nil {
		tasks.RemoveIndex("idx_tasks_nudge")
		app.Save(tasks)
	}

	auditLog, err := app.FindCollectionByNameOrId("audit_log")
	if err == nil {
		auditLog.RemoveIndex("idx_audit_actor")
		app.Save(auditLog)
	}

	return nil
}
