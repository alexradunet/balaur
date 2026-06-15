package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// SkeletonProps configures a Skeleton loading placeholder. Variant defaults to
// "line". Width/Height (line/block) and Size (avatar square) are optional CSS
// length overrides ("100%", "120px"); zero-values use the variant's CSS default.
type SkeletonProps struct {
	Variant string // "line" (default), "block", "avatar"
	Width   string
	Height  string
	Size    string
}

// Skeleton renders a carved parchment loading placeholder with a sliding sheen
// (.skeleton + .skeleton-<variant>, animated by @keyframes sk-sheen). Purely
// decorative: aria-hidden, no children. Only genuine per-instance dimension
// overrides are emitted as the --sk-w / --sk-h custom properties.
func Skeleton(p SkeletonProps) g.Node {
	variant := p.Variant
	if variant == "" {
		variant = "line"
	}
	attrs := []g.Node{
		h.Class("skeleton skeleton-" + variant),
		g.Attr("aria-hidden", "true"),
	}
	if style := skeletonStyle(variant, p); style != "" {
		attrs = append(attrs, h.Style(style))
	}
	return h.Span(attrs...)
}

// skeletonStyle builds the inline custom-property override, or "" when the
// caller relies on the variant defaults. For "avatar", Size drives both axes.
func skeletonStyle(variant string, p SkeletonProps) string {
	if variant == "avatar" {
		if p.Size != "" {
			return "--sk-w:" + p.Size + ";--sk-h:" + p.Size
		}
		return ""
	}
	var s string
	if p.Width != "" {
		s = "--sk-w:" + p.Width
	}
	if p.Height != "" {
		if s != "" {
			s += ";"
		}
		s += "--sk-h:" + p.Height
	}
	return s
}

// SkeletonLine is the common case: a single text-line placeholder. width is an
// optional CSS length ("" keeps the 100% default).
func SkeletonLine(width string) g.Node {
	return Skeleton(SkeletonProps{Variant: "line", Width: width})
}
