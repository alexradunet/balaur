package ui

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// PagerProps configures a Pagination. Total/Page are 1-based; HrefFor maps a page
// number to its URL.
type PagerProps struct {
	Total   int
	Page    int
	HrefFor func(int) string
}

// Pagination renders prev / a window of numbered slabs / next, with ellipses
// when the window is clipped. The active page is a raised gold chip; prev/next
// are disabled (non-link spans) at the bounds. Pure render — navigation is via
// the slab links.
func Pagination(p PagerProps) g.Node {
	if p.Total < 1 {
		p.Total = 1
	}
	if p.Page < 1 {
		p.Page = 1
	}
	start := max(1, min(p.Page-1, p.Total-2))
	end := min(p.Total, start+2)

	kids := []g.Node{h.Class("pager"), h.Aria("label", "Pagination")}
	kids = append(kids, pagerSlab(p, "‹", p.Page-1, p.Page <= 1, false))
	if start > 1 {
		kids = append(kids, pagerGap())
	}
	for n := start; n <= end; n++ {
		kids = append(kids, pagerSlab(p, strconv.Itoa(n), n, false, n == p.Page))
	}
	if end < p.Total {
		kids = append(kids, pagerGap())
	}
	kids = append(kids, pagerSlab(p, "›", p.Page+1, p.Page >= p.Total, false))
	return h.Nav(kids...)
}

// pagerSlab renders one slab: a link, or a non-link span when disabled.
func pagerSlab(p PagerProps, label string, page int, disabled, active bool) g.Node {
	cls := "pager-slab"
	if active {
		cls += " pager-active"
	}
	if disabled {
		return h.Span(h.Class(cls+" pager-disabled"), g.Attr("aria-disabled", "true"), g.Text(label))
	}
	attrs := []g.Node{h.Class(cls), h.Href(p.HrefFor(page))}
	if active {
		attrs = append(attrs, h.Aria("current", "page"))
	}
	attrs = append(attrs, g.Text(label))
	return h.A(attrs...)
}

func pagerGap() g.Node {
	return h.Span(h.Class("pager-gap"), g.Attr("aria-hidden", "true"), g.Text("…"))
}
