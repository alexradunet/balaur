package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestToolRow(t *testing.T) {
	got := render(t, chat.ToolRow(chat.ToolRowProps{
		Tool: "task_add", Icon: "scroll", Content: "added task: water the tomatoes · every 2 days 18:00",
	}))
	for _, want := range []string{
		`<div class="msg msg-tool">`,
		`<div class="who"><img class="tool-icon" src="/static/icons/scroll.png" alt="">tool · task_add</div>`,
		`<div class="body">added task: water the tomatoes · every 2 days 18:00</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("tool row missing %q in: %s", want, got)
		}
	}
}
