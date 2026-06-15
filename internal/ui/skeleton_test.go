package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestSkeleton(t *testing.T) {
	if got := render(t, ui.Skeleton(ui.SkeletonProps{})); got != `<span class="skeleton skeleton-line" aria-hidden="true"></span>` {
		t.Errorf("line default: %s", got)
	}
	if got := render(t, ui.Skeleton(ui.SkeletonProps{Variant: "block"})); got != `<span class="skeleton skeleton-block" aria-hidden="true"></span>` {
		t.Errorf("block: %s", got)
	}
	if got := render(t, ui.Skeleton(ui.SkeletonProps{Variant: "avatar", Size: "54px"})); got != `<span class="skeleton skeleton-avatar" aria-hidden="true" style="--sk-w:54px;--sk-h:54px"></span>` {
		t.Errorf("avatar+size: %s", got)
	}
	if got := render(t, ui.SkeletonLine("60%")); got != `<span class="skeleton skeleton-line" aria-hidden="true" style="--sk-w:60%"></span>` {
		t.Errorf("SkeletonLine width: %s", got)
	}
}
