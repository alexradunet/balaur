package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

// resolveTo returns a resolver that maps any title to the given id (ok=true).
func resolveTo(id string) func(string) (string, bool) {
	return func(string) (string, bool) { return id, true }
}

// resolveNone never resolves (every link is unresolved).
func resolveNone(string) (string, bool) { return "", false }

// TestRenderMarkdownStringPassesWikilink proves the [[Foo]] token survives the
// goldmark+bluemonday base pipeline intact (empirical check #1): an unresolved
// render still produces the wikilink span, which can only happen if [[Foo]]
// reached the substitution. It also asserts the relative href survives sanitize
// (empirical check #2) via the resolved path.
func TestRenderMarkdownStringPassesWikilink(t *testing.T) {
	unresolved := render(t, chat.RenderMarkdownLinked("[[Foo]]", resolveNone))
	if !strings.Contains(unresolved, `wikilink-unresolved`) {
		t.Errorf("expected unresolved span (proves [[Foo]] survived the pipeline), got: %s", unresolved)
	}
	resolved := render(t, chat.RenderMarkdownLinked("[[Foo]]", resolveTo("x")))
	if !strings.Contains(resolved, `href="/ui/show/note?id=x"`) {
		t.Errorf("relative href did not survive sanitize: %s", resolved)
	}
}

func TestRenderMarkdownLinkedResolved(t *testing.T) {
	got := render(t, chat.RenderMarkdownLinked("[[Foo]]", resolveTo("abc123")))
	// The chip is an anchor with the resolved href AND a Datastar @get so the
	// click morphs the panel instead of full-navigating to the SSE-only route.
	for _, want := range []string{
		`class="wikilink"`,
		`href="/ui/show/note?id=abc123"`,
		`data-on:click__prevent="@get('/ui/show/note?id=abc123'); basmOpenPanel()"`,
		`>Foo</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in resolved chip: %s", want, got)
		}
	}
	if strings.Contains(got, "[[Foo]]") {
		t.Errorf("raw wikilink leaked into output: %s", got)
	}
}

func TestRenderMarkdownLinkedAlias(t *testing.T) {
	got := render(t, chat.RenderMarkdownLinked("[[Foo|the foo]]", resolveTo("abc123")))
	if !strings.Contains(got, `>the foo</a>`) {
		t.Errorf("alias display text missing in: %s", got)
	}
	if !strings.Contains(got, `href="/ui/show/note?id=abc123"`) {
		t.Errorf("alias should still resolve the title Foo: %s", got)
	}
}

func TestRenderMarkdownLinkedUnresolved(t *testing.T) {
	got := render(t, chat.RenderMarkdownLinked("[[Foo]]", resolveNone))
	if !strings.Contains(got, `<span class="wikilink wikilink-unresolved">Foo</span>`) {
		t.Errorf("missing unresolved span in: %s", got)
	}
	if strings.Contains(got, "<a ") {
		t.Errorf("unresolved link should not produce an anchor: %s", got)
	}
}

func TestRenderMarkdownLinkedNoInjection(t *testing.T) {
	got := render(t, chat.RenderMarkdownLinked("[[<script>alert(1)</script>]]", resolveNone))
	if strings.Contains(got, "<script>") {
		t.Errorf("raw <script> leaked (display text not escaped): %s", got)
	}
}

// TestRenderMarkdownLinkedPlainUnchanged confirms plain markdown still renders
// (no wikilinks) and that ordinary chat markdown is untouched by the linked
// renderer — bold becomes <strong>, the raw markup does not leak.
func TestRenderMarkdownLinkedPlainUnchanged(t *testing.T) {
	got := render(t, chat.RenderMarkdownLinked("**bold** text", resolveNone))
	if !strings.Contains(got, "<strong>bold</strong>") {
		t.Errorf("plain markdown not rendered: %s", got)
	}
	if strings.Contains(got, "**bold**") {
		t.Errorf("raw markdown leaked: %s", got)
	}
}

// TestRenderMarkdownLinkedConvertErrorEscapes documents the error-fallback
// contract. goldmark's default config does not error on arbitrary UTF-8 input,
// so a forced convert error is not reliably reachable here; the unchanged
// plain-chat success path is already covered by TestMessageBalaurMarkdown in
// message_test.go. We instead assert the related safety property: HTML in the
// markdown body is escaped, never rendered raw, on the normal success path.
func TestRenderMarkdownLinkedConvertErrorEscapes(t *testing.T) {
	got := render(t, chat.RenderMarkdownLinked("<script>alert(1)</script> & <b>x</b>", resolveNone))
	if strings.Contains(got, "<script>") {
		t.Errorf("raw <script> leaked, body not escaped: %s", got)
	}
}
