package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestCommandPalette(t *testing.T) {
	items := []ui.CommandItem{
		{Label: "Quests", Key: "quests", Icon: "scroll", URL: "/ui/show/quests"},
		{Label: "Settings", Key: "settings", URL: "/ui/show/settings?section=profile"},
	}
	got := render(t, ui.CommandPalette(items))

	cases := []struct {
		desc string
		want string
	}{
		{"root class", `class="cmd-palette"`},
		// data-show single-quotes are HTML-escaped to &#39; by gomponents
		{"root data-show", `data-show="$message.startsWith(&#39;/&#39;)"`},
		{"item class", `class="cmd-item"`},
		// per-item data-show contains a .startsWith( call (HTML-escaped quotes)
		{"quests data-show has startsWith", `&#39;quests&#39;.startsWith(`},
		{"settings data-show has startsWith", `&#39;settings&#39;.startsWith(`},
		// @get action in data-on:click__prevent (HTML-escaped)
		{"quests @get action", `@get(&#39;/ui/show/quests&#39;)`},
		// draft clear is in the same attribute
		{"quests clears draft", `$message = &#39;&#39;`},
		{"settings @get action", `@get(&#39;/ui/show/settings?section=profile&#39;)`},
		{"quests /key label", `/quests`},
		{"settings /key label", `/settings`},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if !strings.Contains(got, c.want) {
				t.Fatalf("missing %q in:\n%s", c.want, got)
			}
		})
	}
}
