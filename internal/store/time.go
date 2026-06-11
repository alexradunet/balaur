package store

import (
	"time"

	"github.com/pocketbase/pocketbase/tools/types"
)

// PBTime formats a time the way PocketBase stores DateTime fields, so
// filter comparisons match exactly. Moved here on its third caller
// (recap, tasks, web) per the one-source-of-truth rule.
func PBTime(t time.Time) string {
	return t.UTC().Format(types.DefaultDateLayout)
}
