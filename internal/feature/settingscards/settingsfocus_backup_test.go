package settingscards_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/settingscards"
)

// TestBackupSectionContract guards the ids, both @post targets, the password
// input, autocomplete=off, and the unrecoverable-passphrase warning so the SSE
// patch (outer #backup-section) and the passphrase discipline both hold.
func TestBackupSectionContract(t *testing.T) {
	got := renderNode(t, settingscards.BackupSection(settingscards.BackupView{}))
	for _, want := range []string{
		`id="backup-section"`,
		`class="profile-card"`,
		// gomponents escapes ' → &#39; in attribute values
		`data-on:submit__prevent="@post(&#39;/ui/settings/export&#39;, {contentType:&#39;form&#39;})"`,
		`data-on:submit__prevent="@post(&#39;/ui/settings/backup&#39;, {contentType:&#39;form&#39;})"`,
		`name="passphrase"`,
		`type="password"`,
		`autocomplete="off"`,
		`UNRECOVERABLE`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("BackupSection missing %q in:\n%s", want, got)
		}
	}
}

// TestBackupSectionResults asserts the result lines rendered for each outcome
// of the last action, and that an owner-safe error line renders.
func TestBackupSectionResults(t *testing.T) {
	got := renderNode(t, settingscards.BackupSection(settingscards.BackupView{
		MirrorDone:  true,
		MirrorFiles: 3,
		MirrorDest:  "/data/export",
		ArchivePath: "/data/backup/balaur-backup-x.enc",
	}))
	for _, want := range []string{
		"Wrote 3 files to /data/export.",
		"balaur restore",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("BackupSection results missing %q in:\n%s", want, got)
		}
	}

	errGot := renderNode(t, settingscards.BackupSection(settingscards.BackupView{Error: "boom"}))
	if !strings.Contains(errGot, "Error: boom") {
		t.Errorf("BackupSection error missing %q in:\n%s", "Error: boom", errGot)
	}
}
