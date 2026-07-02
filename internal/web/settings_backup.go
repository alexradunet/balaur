package web

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/export"
	"github.com/alexradunet/balaur/internal/feature/settingscards"
	"github.com/alexradunet/balaur/internal/store"
)

// settings_backup.go — the web face of `balaur export` (plans 194/195): the
// settings Backup section's two POSTs. Restore stays CLI-only. The passphrase
// is read from the form once, used, and discarded — never logged, never
// persisted, never echoed back (BackupView has no passphrase field).

// patchBackupSection re-renders #backup-section with the given result view.
func (h *handlers) patchBackupSection(e *core.RequestEvent, view settingscards.BackupView, toastTone, toastMsg string) error {
	var b strings.Builder
	if err := settingscards.BackupSection(view).Render(&b); err != nil {
		return e.InternalServerError("rendering backup section", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	patchOuterHTML(sse, "backup-section", b.String())
	if toastMsg != "" {
		emitToast(sse, toastTone, toastMsg)
	}
	return nil
}

// exportMirrorNow handles POST /ui/settings/export — writes the Markdown
// mirror to <data dir>/export (same default as the CLI verb) and re-renders
// the section with the destination + file count.
func (h *handlers) exportMirrorNow(e *core.RequestEvent) error {
	dest := filepath.Join(h.app.DataDir(), "export")
	files, err := export.ExportMirror(h.app, dest)
	if err != nil {
		h.app.Logger().Warn("export mirror from settings", "err", err)
		return h.patchBackupSection(e, settingscards.BackupView{Error: "could not write the mirror — see the server log"}, "", "")
	}
	// Audit strictly AFTER the successful write.
	store.Audit(h.app, "owner", "export.mirror", dest, true, map[string]any{"files": len(files)})
	view := settingscards.BackupView{MirrorDone: true, MirrorFiles: len(files), MirrorDest: dest}
	return h.patchBackupSection(e, view, "success", "Markdown mirror written.")
}

// backupEncryptNow handles POST /ui/settings/backup — renders the mirror into
// a wiped temp dir (same shape as the CLI's runEncrypt), encrypts it into
// <data dir>/backup/balaur-backup-<stamp>.enc, and shows the archive path.
// The passphrase comes from the form value only; it is not trimmed (only the
// empty string is rejected) and never logged.
func (h *handlers) backupEncryptNow(e *core.RequestEvent) error {
	passphrase := e.Request.FormValue("passphrase")
	if passphrase == "" {
		return e.BadRequestError("a passphrase is required", nil)
	}

	tmp, err := os.MkdirTemp("", "balaur-export-*")
	if err != nil {
		return e.InternalServerError("creating export temp dir", err)
	}
	defer os.RemoveAll(tmp)

	if _, err := export.ExportMirror(h.app, tmp); err != nil {
		h.app.Logger().Warn("export mirror for backup", "err", err)
		return h.patchBackupSection(e, settingscards.BackupView{Error: "could not render the mirror — see the server log"}, "", "")
	}

	backupDir := filepath.Join(h.app.DataDir(), "backup")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return e.InternalServerError("creating backup dir", err)
	}
	archive := filepath.Join(backupDir, "balaur-backup-"+time.Now().Format("20060102-150405")+".enc")
	if err := export.EncryptDir(tmp, archive, passphrase); err != nil {
		h.app.Logger().Warn("encrypting backup", "err", err)
		return h.patchBackupSection(e, settingscards.BackupView{Error: "could not encrypt the backup — see the server log"}, "", "")
	}
	// Audit strictly AFTER the successful write. No passphrase material anywhere.
	store.Audit(h.app, "owner", "export.encrypt", archive, true, nil)
	return h.patchBackupSection(e, settingscards.BackupView{ArchivePath: archive}, "success", "Encrypted backup written.")
}
