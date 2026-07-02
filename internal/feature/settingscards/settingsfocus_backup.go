// settingsfocus_backup.go — the Backup settings section: write the Markdown
// mirror and the passphrase-encrypted backup archive without a terminal.
// UI face of the `balaur export` / `balaur export --encrypt` CLI verbs.
package settingscards

import (
	"fmt"

	g "maragu.dev/gomponents"
	data "maragu.dev/gomponents-datastar"
	h "maragu.dev/gomponents/html"
)

// BackupView is the view-model for the backup section. It carries only the
// RESULTS of the last action — never the passphrase (which is read from the
// form once, used, and discarded; it must not appear in any view-model field).
type BackupView struct {
	MirrorDone  bool   // last action wrote the Markdown mirror
	MirrorFiles int    // files written by that mirror run
	MirrorDest  string // where the mirror was written
	ArchivePath string // path of the encrypted archive just written; empty otherwise
	Error       string // owner-safe error line; empty when the last action succeeded
}

// BackupSection renders the backup controls (#backup-section): the Markdown
// mirror button and the encrypted-backup form. Re-render target after POST
// /ui/settings/export and /ui/settings/backup (outer patch #backup-section).
func BackupSection(v BackupView) g.Node {
	out := []g.Node{
		h.H2(h.Class("profile-card-title"), g.Text("Backup & export")),
		h.P(h.Class("profile-hint"), g.Text("Your record, portable: write a readable Markdown mirror of your active notes, memories, and journal — or wrap it in an encrypted archive for off-box backup. Nothing leaves this machine.")),
	}
	if v.Error != "" {
		out = append(out, h.P(h.Class("profile-hint"), g.Text("Error: "+v.Error)))
	}

	// Plain mirror.
	out = append(out,
		h.Form(
			data.On("submit", "@post('/ui/settings/export', {contentType:'form'})", data.ModifierPrevent),
			h.Div(h.Class("kcard-actions"),
				h.Button(h.Class("btn btn-primary btn-sm"), h.Type("submit"), g.Text("Write Markdown mirror")),
			),
		),
	)
	if v.MirrorDone {
		out = append(out, h.P(h.Class("profile-hint"),
			g.Text(fmt.Sprintf("Wrote %d files to %s.", v.MirrorFiles, v.MirrorDest))))
	}

	// Encrypted backup. The passphrase input is type=password + autocomplete=off
	// and is NEVER echoed back — BackupView has no passphrase field by design.
	out = append(out,
		h.Form(
			h.Class("profile-name-form"),
			data.On("submit", "@post('/ui/settings/backup', {contentType:'form'})", data.ModifierPrevent),
			h.Label(h.For("backup_passphrase"), g.Text("Passphrase")),
			h.Div(h.Class("profile-name-row"),
				h.Input(
					h.ID("backup_passphrase"),
					h.Name("passphrase"),
					h.Type("password"),
					h.Placeholder("choose a strong passphrase…"),
					g.Attr("autocomplete", "off"),
				),
				h.Button(h.Class("btn btn-primary"), h.Type("submit"), g.Text("Write encrypted backup")),
			),
		),
		h.P(h.Class("profile-hint"), g.Text("If you lose this passphrase, this backup is UNRECOVERABLE — there is no recovery, no escrow, no cloud.")),
	)
	if v.ArchivePath != "" {
		out = append(out, h.P(h.Class("profile-hint"), g.Text("Encrypted backup written to "+v.ArchivePath+". Restore with: balaur restore --archive <path> --out <dir>.")))
	}

	return h.Article(h.Class("profile-card"), h.ID("backup-section"), g.Group(out))
}
