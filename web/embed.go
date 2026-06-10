// Package web holds the embedded UI assets: templates and static files.
// Handlers live in internal/web; this package only exposes the file system
// so the single-binary build carries the whole interface.
package web

import "embed"

//go:embed static templates
var FS embed.FS
