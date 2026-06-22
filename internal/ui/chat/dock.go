package chat

import (
	"fmt"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// DockVariant selects the dock's width/density: "rail" (the right-rail
// sidebar), "overlay" (.dock-full full-screen over content), "home" (the
// full-screen companion chat home — the widest column). The variant maps to a
// CSS width token, collapsing the three hand-synced #dock width blocks.
type DockVariant string

const (
	DockRail    DockVariant = "rail"
	DockOverlay DockVariant = "overlay"
	DockHome    DockVariant = "home"
)

// DockProps configures the chat.Dock organism. Feature-owned slots (Convo,
// Composer, Switchers) are pre-rendered g.Node values injected by the caller
// so internal/ui/chat does not import internal/feature/*.
type DockProps struct {
	Variant   DockVariant
	HasRecap  bool
	NowMillis int64  // nudge-poll cursor seed
	Convo     g.Node // #chat section content (history/greeting) — caller-rendered
	Composer  g.Node // ui.Composer node — caller-rendered (feature-injected)
	Switchers g.Node // chatbar/switcher node — caller-rendered (may be nil)
}

// Dock renders the companion chat dock chrome — the children of <aside id="dock">
// (shell.go owns the <aside> wrapper). It emits, in order:
//  1. .dock-grip — the resize handle
//  2. .dock-head — the full-screen toggle button
//  3. #recap.recap-zone — the telescope sentinel (only when HasRecap)
//  4. #dock-convo — the flex wrapper around #chat
//  5. #nudge-poll — the agent-initiated-message poller
//  6. the Composer slot (ui.Composer, caller-rendered)
//  7. <dialog id="model-modal"> — opened by basm.js after a model panel swap
//
// The Switchers slot is appended after the Composer when non-nil (pre-rendered
// chatbar/head-switcher gomponents node — may be nil; wiring into homePage is deferred).
//
// attrs are passed to a wrapper div that carries the variant class
// (dock-v-rail / dock-v-overlay / dock-v-home). The variant class is the
// single source of truth for the chat column width; see basm.css § chat dock
// variants.
func Dock(p DockProps, attrs ...g.Node) g.Node {
	variantClass := fmt.Sprintf("dock-v-%s", p.Variant)

	wrapAttrs := []g.Node{h.Class(variantClass)}
	wrapAttrs = append(wrapAttrs, attrs...)
	wrapAttrs = append(wrapAttrs,
		dockGrip(),
		dockHead(),
		recapZone(p.HasRecap),
		dockConvo(p.Convo),
		nudgePoll(p.NowMillis),
		p.Composer,
		h.Dialog(h.ID("model-modal"), g.Attr("aria-labelledby", "model-modal-title")),
		p.Switchers,
	)

	return h.Div(wrapAttrs...)
}

// dockGrip renders the resize handle bar.
func dockGrip() g.Node {
	return h.Div(h.Class("dock-grip"), h.Title("drag to resize"), g.Attr("aria-hidden", "true"))
}

// dockHead renders the full-screen toggle button header.
func dockHead() g.Node {
	return h.Header(h.Class("dock-head"),
		h.Button(h.Class("dock-btn dock-full-btn"), h.Type("button"),
			g.Attr("onclick", "basmToggleDockFull()"),
			h.Aria("label", "Toggle full-screen chat"),
		),
	)
}

// recapZone renders the telescope sentinel div when HasRecap is true. The
// data-on:intersect__once attribute triggers a lazy load of recap bands when
// the element scrolls into view — #recap is patched in-place by recap.go.
func recapZone(hasRecap bool) g.Node {
	if !hasRecap {
		return nil
	}
	return h.Div(h.ID("recap"), h.Class("recap-zone"),
		g.Attr("data-on:intersect__once", "@get('/ui/recap/bands')"),
		h.P(h.Class("recap-hint"), g.Text("◇ further back…")),
	)
}

// dockConvo renders the #dock-convo flex wrapper that contains #chat. The
// convo node (the history/greeting, pre-rendered by the caller) is placed
// inside the <section class="chat" id="chat">.
func dockConvo(convo g.Node) g.Node {
	return h.Div(h.ID("dock-convo"),
		h.Section(h.Class("chat"), h.ID("chat"), g.Attr("aria-live", "polite"),
			convo,
		),
	)
}

// nudgePoll renders the signal seed + 30-second interval poller. The three
// data-signals seeds are read by:
//   - nudgeSince: tasks.go advances the cursor after each batch
//   - dockMaster: prevents multi-tab duplicate polling (only true on one tab)
//   - streaming: chatstream.go sets/clears it; head-switcher reads it
func nudgePoll(nowMillis int64) g.Node {
	return h.Div(h.ID("nudge-poll"),
		g.Attr("data-signals:nudgeSince", fmt.Sprintf("%d", nowMillis)),
		g.Attr("data-signals:dockMaster", "true"),
		g.Attr("data-signals:streaming", "false"),
		g.Attr("data-on:interval__duration.30s", "$dockMaster && @get('/ui/chat/nudges?since='+$nudgeSince)"),
	)
}
