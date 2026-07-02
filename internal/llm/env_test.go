package llm

import (
	"context"
	"errors"
	"testing"
)

func TestCollect(t *testing.T) {
	streamErr := errors.New("model exploded")

	cases := []struct {
		name     string
		chunks   []Chunk
		cancel   bool // cancel ctx before draining
		wantText string
		wantErr  error // matched with errors.Is; nil means no error
	}{
		{name: "complete reply", chunks: []Chunk{{Content: "hello "}, {Content: "world"}, {Done: true}}, wantText: "hello world"},
		{name: "terminal err chunk", chunks: []Chunk{{Content: "par"}, {Err: streamErr}}, wantText: "par", wantErr: streamErr},
		{name: "cancelled and cut mid-stream", chunks: []Chunk{{Content: "trunc"}}, cancel: true, wantText: "trunc", wantErr: context.Canceled},
		{name: "done outruns the cancel", chunks: []Chunk{{Content: "full"}, {Done: true}}, cancel: true, wantText: "full"},
		{name: "close without done, live ctx", chunks: []Chunk{{Content: "odd"}}, wantText: "odd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			ch := make(chan Chunk, len(tc.chunks))
			for _, c := range tc.chunks {
				ch <- c
			}
			close(ch)

			text, err := Collect(ctx, ch)
			if text != tc.wantText {
				t.Errorf("text = %q, want %q", text, tc.wantText)
			}
			if tc.wantErr == nil && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want errors.Is %v", err, tc.wantErr)
			}
		})
	}
}
