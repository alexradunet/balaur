package ui_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestPagination(t *testing.T) {
	got := render(t, ui.Pagination(ui.PagerProps{
		Total: 5, Page: 3,
		HrefFor: func(n int) string { return "/p/" + strconv.Itoa(n) },
	}))
	for _, want := range []string{
		`<nav class="pager" aria-label="Pagination">`,
		`<a class="pager-slab pager-active" href="/p/3" aria-current="page">3</a>`,
		`<a class="pager-slab" href="/p/2">2</a>`,
		`<a class="pager-slab" href="/p/4">4</a>`,
		`<span class="pager-gap" aria-hidden="true">…</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("pager missing %q in: %s", want, got)
		}
	}
}

func TestPaginationBounds(t *testing.T) {
	got := render(t, ui.Pagination(ui.PagerProps{Total: 3, Page: 1, HrefFor: func(n int) string { return "#" }}))
	if !strings.Contains(got, `<span class="pager-slab pager-disabled" aria-disabled="true">‹</span>`) {
		t.Errorf("prev should be disabled at page 1: %s", got)
	}
	if !strings.Contains(got, `>›</a>`) {
		t.Errorf("next should be a link at page 1: %s", got)
	}
}
