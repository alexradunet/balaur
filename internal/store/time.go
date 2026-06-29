package store

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// PBTime formats a time the way PocketBase stores DateTime fields, so
// filter comparisons match exactly. Moved here on its third caller
// (recap, tasks, web) per the one-source-of-truth rule.
func PBTime(t time.Time) string {
	return t.UTC().Format(types.DefaultDateLayout)
}

// OwnerLocation resolves the timezone that anchors owner-facing period
// math (recap days/weeks/months). An IANA name in owner_settings key
// "timezone" pins it across box moves; unset or invalid falls back to the
// box's local zone — exactly the pre-setting behavior. Resolved per call:
// no global state, and a dashboard edit takes effect on the next cron tick.
func OwnerLocation(app core.App) *time.Location {
	name := GetOwnerSetting(app, "timezone", "")
	if name == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return loc
}

// ParsePBTime parses a PocketBase DateTime string (the format PBTime emits).
func ParsePBTime(s string) (time.Time, error) {
	return time.Parse(types.DefaultDateLayout, s)
}
