package web

import (
	g "maragu.dev/gomponents"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/ui"
)

// emitToast appends an owner-facing toast pill into the body-level #toast-region
// over an already-open SSE stream (plan 174 S7). The .toast-in keyframe (basm.css)
// gives the pixel-snappy stamp; basmToast (basm.js) holds then auto-dismisses it.
// Tone is "info" (default), "success", or "warn".
func emitToast(sse *datastar.ServerSentEventGenerator, tone, msg string) {
	_ = sse.PatchElements(
		renderNodeHTML(ui.Toast(ui.ToastProps{Tone: tone}, g.Text(msg))),
		datastar.WithSelectorID("toast-region"), datastar.WithModeAppend(),
	)
}
