package chat_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)

// render renders a gomponents node to its HTML string, failing the test on
// error. Shared by the organism tests in this package.
func render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}
