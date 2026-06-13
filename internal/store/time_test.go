package store

import (
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/storetest"
)

// TestOwnerLocationDefaultsToLocal verifies that when no "timezone" key is
// set in owner_settings, OwnerLocation returns time.Local — the pre-setting
// behavior, byte-identical to time.Now() callers before this plan.
func TestOwnerLocationDefaultsToLocal(t *testing.T) {
	app := storetest.NewApp(t)
	got := OwnerLocation(app)
	if got != time.Local {
		t.Errorf("OwnerLocation with no key: got %q, want time.Local (%q)", got, time.Local)
	}
}

// TestOwnerLocationReadsSetting verifies that when the "timezone" key is set
// to a valid IANA name, OwnerLocation returns a matching *time.Location.
func TestOwnerLocationReadsSetting(t *testing.T) {
	app := storetest.NewApp(t)
	if err := SetOwnerSetting(app, "timezone", "Europe/Bucharest"); err != nil {
		t.Fatalf("SetOwnerSetting: %v", err)
	}
	loc := OwnerLocation(app)
	if loc.String() != "Europe/Bucharest" {
		t.Errorf("OwnerLocation returned %q, want %q", loc.String(), "Europe/Bucharest")
	}
}

// TestOwnerLocationInvalidFallsBack verifies that an unrecognised IANA name
// falls back to time.Local rather than erroring.
func TestOwnerLocationInvalidFallsBack(t *testing.T) {
	app := storetest.NewApp(t)
	if err := SetOwnerSetting(app, "timezone", "Not/AZone"); err != nil {
		t.Fatalf("SetOwnerSetting: %v", err)
	}
	got := OwnerLocation(app)
	if got != time.Local {
		t.Errorf("OwnerLocation with invalid key: got %q, want time.Local (%q)", got, time.Local)
	}
}

// TestOwnerLocationPeriodStability verifies that loading a named timezone
// produces a location with deterministic, time-stable offset for a fixed
// wall-clock time. This is the period-stability property: given a fixed
// wall-clock moment (e.g. 2024-03-15 10:00 Europe/Bucharest), the UTC
// instant is always the same regardless of the test process's local zone.
//
// We cannot import internal/recap here (cycle: recap→store), so we assert
// directly on the loaded *time.Location without invoking period helpers.
func TestOwnerLocationPeriodStability(t *testing.T) {
	app := storetest.NewApp(t)
	if err := SetOwnerSetting(app, "timezone", "Europe/Bucharest"); err != nil {
		t.Fatalf("SetOwnerSetting: %v", err)
	}
	loc := OwnerLocation(app)

	// 2024-03-15 is in EET (UTC+2) for Europe/Bucharest (DST starts late March).
	wall := time.Date(2024, 3, 15, 10, 0, 0, 0, loc)
	wantUTC := time.Date(2024, 3, 15, 8, 0, 0, 0, time.UTC)
	if !wall.UTC().Equal(wantUTC) {
		t.Errorf("period stability: %v in Europe/Bucharest → UTC %v, want %v",
			wall, wall.UTC(), wantUTC)
	}
}
