package llm

import (
	"context"
	"fmt"
)

// Collect drains a ChatStream into the full text reply. For background
// work (summaries) where streaming buys nothing.
//
// A stream that closes without a terminal Done or Err chunk while ctx is
// dead was cut mid-generation (the provider bridges guard sends with
// ctx.Done and can drop the terminal chunk on cancellation); that is an
// error, never a short success. A Done chunk marks the reply complete
// even if ctx expired immediately afterwards.
func Collect(ctx context.Context, ch <-chan Chunk) (string, error) {
	var text string
	var done bool
	for chunk := range ch {
		if chunk.Err != nil {
			return text, chunk.Err
		}
		if chunk.Done {
			done = true
		}
		text += chunk.Content
	}
	if !done {
		if err := ctx.Err(); err != nil {
			return text, fmt.Errorf("model stream interrupted: %w", err)
		}
	}
	return text, nil
}
