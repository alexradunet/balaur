package chat

import (
	"bytes"
	"html"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	g "maragu.dev/gomponents"
)

// md converts assistant Markdown to HTML. Default goldmark config (no
// html.WithUnsafe) escapes raw HTML blocks, so model-emitted <script> never
// renders; the bluemonday pass below then strips dangerous link schemes
// (javascript:, data:) and any stray markup goldmark passed through. Built once
// and reused — both values are concurrency-safe after construction.
var (
	md     = goldmark.New()
	mdSane = bluemonday.UGCPolicy()
)

// linkChipRe matches the literal [[Title]] / [[Title|alias]] text that survives
// goldmark+bluemonday. Kept in lockstep with internal/nodes wikilinkRe.
var linkChipRe = regexp.MustCompile(`\[\[([^\[\]|]+?)(?:\|([^\[\]]*))?\]\]`)

// renderMarkdownString runs the goldmark+bluemonday pipeline and returns the
// sanitized HTML. ok=false signals a goldmark convert error so callers reproduce
// the original escaped-plain-text fallback (g.Text) instead of trusting raw HTML.
func renderMarkdownString(s string) (string, bool) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(s), &buf); err != nil {
		return "", false
	}
	return mdSane.Sanitize(buf.String()), true
}

// renderMarkdown turns assistant Markdown into a trusted, sanitized HTML node.
// On any error it falls back to escaped plain text — a render failure must never
// blank or unescape the bubble.
//
// ponytail: re-renders the whole accumulated buffer on every streamed token.
// Fine for a local single-owner app with short replies; revisit only if a
// measurement shows it on the hot path.
func renderMarkdown(s string) g.Node {
	out, ok := renderMarkdownString(s)
	if !ok {
		return g.Text(s) // unchanged from the original error fallback
	}
	return g.Raw(out)
}

// RenderMarkdownLinked renders node Markdown like renderMarkdown, then turns each
// [[wikilink]] into a clickable chip. resolve maps a link title to the target
// node's id; ok=false means unresolved (after plan 161's save hook, unresolved
// links should be rare — a stub is created on save — but render must still
// degrade gracefully to a non-link span). The display text is the alias when
// present, else the title. Every chip links to the 160 generic node viewer
// /ui/show/note?id=<id> — that route renders a node of any type by id (it derives
// the type from the record), so no type param is needed; never /ui/show/node
// (no such card) or /ui/notes/{id}.
//
// Used by BOTH the chat bubble path and the note card (plan 161): the note card
// (internal/feature/knowledgecards) imports this leaf UI package to render its
// body's [[links]] as chips. A future cleanup may promote this renderer from
// internal/ui/chat to a shared internal/ui atom so cards need not import the chat
// package — DEFERRED, not in 161.
func RenderMarkdownLinked(s string, resolve func(title string) (id string, ok bool)) g.Node {
	base, ok := renderMarkdownString(s)
	if !ok {
		return g.Text(s) // a convert error has no [[links]] to substitute
	}
	out := linkChipRe.ReplaceAllStringFunc(base, func(m string) string {
		sub := linkChipRe.FindStringSubmatch(m)
		title := strings.TrimSpace(sub[1])
		display := title
		if len(sub) > 2 && strings.TrimSpace(sub[2]) != "" {
			display = strings.TrimSpace(sub[2])
		}
		display = html.EscapeString(display)
		if title == "" {
			return m
		}
		id, ok := resolve(title)
		if !ok || id == "" {
			return `<span class="wikilink wikilink-unresolved">` + display + `</span>`
		}
		href := "/ui/show/note?id=" + html.EscapeString(id)
		// Datastar @get so the click morphs #panel-inner instead of doing a full
		// browser navigation — /ui/show is SSE-only, so a plain href would render
		// raw "event: datastar-patch-elements" text. basmOpenPanel() reveals the
		// panel when the chip is clicked from a chat bubble (no-op when already in
		// the panel). Injected after the bluemonday pass, so the attribute survives.
		action := "@get('" + href + "'); basmOpenPanel()"
		return `<a class="wikilink" href="` + href + `" data-on:click__prevent="` + action + `">` + display + `</a>`
	})
	return g.Raw(out)
}
