package taskcards

// Internal test (package taskcards, not taskcards_test) so it can call the
// unexported buildToday directly. today_test.go already owns package
// taskcards_test and stays untouched; seedTask is shared from
// taskscluster_test.go (same package).

import (
	"strings"
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestBuildTodayRendersOwnerZoneDueLine guards the regression this plan
// fixes: buildToday must resolve now in the owner's configured timezone, not
// the host zone, so the rendered due line matches the owner's wall clock.
func TestBuildTodayRendersOwnerZoneDueLine(t *testing.T) {
	if _, off := time.Now().Zone(); off == 14*3600 {
		t.Skip("host zone is UTC+14; owner-zone and host-zone results coincide")
	}
	app := storetest.NewApp(t)
	if err := store.SetOwnerSetting(app, "timezone", "Pacific/Kiritimati"); err != nil {
		t.Fatalf("set timezone: %v", err)
	}

	// Far-past due — always Overdue in every zone, so always in buildToday's rows.
	seedTask(t, app, "Renew passport", "open", time.Date(2020, 1, 2, 12, 0, 0, 0, time.UTC))

	v := buildToday(app)
	if len(v.Rows) != 1 {
		t.Fatalf("Rows = %d, want 1", len(v.Rows))
	}
	// 12:00 UTC is 02:00 the next day in UTC+14; a host-zone regression
	// renders the host wall time instead.
	if !strings.Contains(v.Rows[0].DueLine, "Jan 3 at 02:00") {
		t.Errorf("DueLine = %q, want it to contain %q", v.Rows[0].DueLine, "Jan 3 at 02:00")
	}
}
