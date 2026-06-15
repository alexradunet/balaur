// Package assets embeds Balaur's static web assets — the Hearthwood basm.css,
// self-hosted fonts, pixel icons, avatars, crest, and logo — so the single
// binary serves them. Served at /static/... via apis.Static in internal/web.
package assets

import "embed"

//go:embed static
var FS embed.FS
