package ui

import (
	"encoding/json"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// CompactDialogProps configures the manual-compaction modal. In form mode the
// owner reviews/edits the proposed Draft, then Accepts (commit), Refreshes
// (regenerate), or Declines (discard). In message mode (Message set, no Draft)
// it shows a single note + Close — used when there is nothing to fold or the
// model is unavailable.
type CompactDialogProps struct {
	Draft      string // proposed summary, editable (form mode)
	Message    string // info line (message mode); takes precedence over Draft
	AcceptURL  string // @post target that commits the edited summary
	RefreshURL string // @post target that regenerates the draft
	Signal     string // signal bound to the textarea; defaults to "compactDraft"
	Open       bool   // render the <dialog> in place (storybook); production uses showModal()
}

// CompactDialog renders the compaction modal as a native <dialog id="compact-modal">
// — the gateway appends it to the dock and opens it with showModal(). Decline /
// Close call the dialog's own close(); Accept / Refresh @post to the gateway.
// The textarea two-way binds the draft signal (seeded from Draft) so Accept can
// post the owner's edits.
func CompactDialog(p CompactDialogProps) g.Node {
	sig := p.Signal
	if sig == "" {
		sig = "compactDraft"
	}

	kids := []g.Node{h.ID("compact-modal"), h.Class("compact-modal")}
	if p.Open {
		kids = append(kids, h.Open())
	}

	if p.Message != "" {
		kids = append(kids,
			h.Div(h.Class("dlg-kicker"), g.Text("Compact today")),
			h.P(h.Class("compact-modal-msg"), g.Text(p.Message)),
			h.Div(h.Class("dlg-actions"),
				Button(ButtonProps{Variant: "ghost", Size: "sm"}, h.Type("button"),
					g.Attr("data-on:click", "el.closest('dialog').close()"), g.Text("Close")),
			),
		)
		return h.Dialog(kids...)
	}

	// json.Marshal turns the draft into a valid JS string literal (handles quotes
	// and newlines) so it can seed the signal inline.
	raw, _ := json.Marshal(p.Draft)
	kids = append(kids,
		g.Attr("data-signals:"+sig, string(raw)),
		h.Div(h.Class("dlg-kicker"), g.Text("Compact today")),
		h.H2(h.Class("dlg-title"), g.Text("Fold today into a summary")),
		h.P(h.Class("compact-modal-hint"),
			g.Text("Review and edit the summary. Accepting clears today's thread for a clean slate; the summary stays in context.")),
		h.Textarea(h.Class("compact-modal-text"), h.Rows("8"),
			g.Attr("autocomplete", "off"), g.Attr("data-bind:"+sig)),
		h.Div(h.Class("dlg-actions"),
			Button(ButtonProps{Variant: "ghost", Size: "sm"}, h.Type("button"),
				g.Attr("data-on:click", "el.closest('dialog').close()"), g.Text("Decline")),
			Button(ButtonProps{Variant: "wood", Size: "sm"}, h.Type("button"),
				g.Attr("data-on:click", "@post('"+p.RefreshURL+"')"), g.Text("Refresh")),
			Button(ButtonProps{Size: "sm"}, h.Type("button"),
				g.Attr("data-on:click", "@post('"+p.AcceptURL+"')"), g.Text("Accept")),
		),
	)
	return h.Dialog(kids...)
}
