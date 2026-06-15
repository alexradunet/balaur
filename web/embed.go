// Package web holds the embedded HTML templates — the legacy html/template
// surface being migrated to gomponents. Static assets moved to
// internal/web/assets. This package and its templates are removed once the
// migration is complete.
package web

import "embed"

//go:embed templates
var FS embed.FS
