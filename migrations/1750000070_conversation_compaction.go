package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(upConversationCompaction, downConversationCompaction)
}

// upConversationCompaction gives the master conversation a manual-compaction
// surface: `summary` holds the rolling, owner-triggered recap of earlier turns,
// and `compacted_through` marks the boundary past which those turns are folded
// (kept in the record, dropped from the live dock and model context). Messages
// are never deleted — compaction only summarises and advances the boundary, so
// the chronicle still reads the full day. Idempotent: skips fields already present.
func upConversationCompaction(app core.App) error {
	col, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		return fmt.Errorf("conversation_compaction: finding conversations: %w", err)
	}
	if col.Fields.GetByName("summary") == nil {
		col.Fields.Add(&core.TextField{Name: "summary", Max: 20000})
	}
	if col.Fields.GetByName("compacted_through") == nil {
		col.Fields.Add(&core.DateField{Name: "compacted_through"})
	}
	if err := app.Save(col); err != nil {
		return fmt.Errorf("conversation_compaction: saving conversations: %w", err)
	}
	return nil
}

// downConversationCompaction removes the two compaction fields.
func downConversationCompaction(app core.App) error {
	col, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		return fmt.Errorf("conversation_compaction down: finding conversations: %w", err)
	}
	for _, name := range []string{"summary", "compacted_through"} {
		if f := col.Fields.GetByName(name); f != nil {
			col.Fields.RemoveByName(name)
		}
	}
	if err := app.Save(col); err != nil {
		return fmt.Errorf("conversation_compaction down: saving conversations: %w", err)
	}
	return nil
}
