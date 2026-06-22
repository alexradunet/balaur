package chat_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/uitest"
)

// render renders a gomponents node to its HTML string, failing the test on
// error. Shared by the organism tests in this package.
func render(t *testing.T, n g.Node) string {
	return uitest.Render(t, n)
}
