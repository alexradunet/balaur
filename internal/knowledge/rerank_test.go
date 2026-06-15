package knowledge

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/alexradunet/balaur/internal/llmtest"
	"github.com/alexradunet/balaur/internal/storetest"
	"github.com/pocketbase/pocketbase/core"
)

// makeActiveMemory creates and approves a memory record for testing.
func makeActiveMemory(t *testing.T, app core.App, title, content string) *core.Record {
	t.Helper()
	rec, err := ProposeMemory(app, MemoryProposal{Title: title, Content: content, Importance: 2})
	if err != nil {
		t.Fatalf("ProposeMemory(%q): %v", title, err)
	}
	rec, err = Transition(app, Memory, rec.Id, StatusActive)
	if err != nil {
		t.Fatalf("Transition to active (%q): %v", title, err)
	}
	return rec
}

func TestRerankByEmbedding(t *testing.T) {
	app := storetest.NewApp(t)

	// Three candidates in a known lexical order: A, B, C.
	// We will script vectors so C is closest to the query, then B, then A.
	recA := makeActiveMemory(t, app, "Alpha memory", "unrelated topic")
	recB := makeActiveMemory(t, app, "Beta memory", "somewhat relevant")
	recC := makeActiveMemory(t, app, "Gamma memory", "directly on point")

	lexicalOrder := []*core.Record{recA, recB, recC}

	// EmbedFunc that assigns:
	//   query  → [1, 0]
	//   recA   → [0, 1]    cosine(query, A) = 0
	//   recB   → [0.5,0.5] cosine(query, B) ≈ 0.707
	//   recC   → [1, 0]    cosine(query, C) = 1
	// Expected reranked order: C, B, A.
	scriptedVecs := func(texts []string) ([][]float32, error) {
		vecs := make([][]float32, len(texts))
		vecs[0] = []float32{1, 0} // query
		vecs[1] = []float32{0, 1}
		vecs[2] = []float32{0.5, 0.5}
		vecs[3] = []float32{1, 0}
		return vecs, nil
	}

	t.Run("reorder happens when flag on", func(t *testing.T) {
		t.Setenv("BALAUR_EMBED_RERANK", "1")
		client := &llmtest.ScriptedClient{EmbedFunc: scriptedVecs}
		got := RerankByEmbedding(context.Background(), client, "query text", lexicalOrder)
		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
		if got[0].Id != recC.Id || got[1].Id != recB.Id || got[2].Id != recA.Id {
			t.Fatalf("order = [%s %s %s], want [C B A]",
				got[0].GetString("title"),
				got[1].GetString("title"),
				got[2].GetString("title"))
		}
	})

	t.Run("flag off returns input order unchanged", func(t *testing.T) {
		// No Setenv("BALAUR_EMBED_RERANK") → flag is off.
		client := &llmtest.ScriptedClient{EmbedFunc: scriptedVecs}
		got := RerankByEmbedding(context.Background(), client, "query text", lexicalOrder)
		for i, r := range lexicalOrder {
			if got[i].Id != r.Id {
				t.Fatalf("position %d: got %q, want %q", i, got[i].GetString("title"), r.GetString("title"))
			}
		}
	})

	t.Run("embed error returns input order unchanged", func(t *testing.T) {
		t.Setenv("BALAUR_EMBED_RERANK", "1")
		client := &llmtest.ScriptedClient{
			EmbedFunc: func(texts []string) ([][]float32, error) {
				return nil, errors.New("boom")
			},
		}
		got := RerankByEmbedding(context.Background(), client, "query text", lexicalOrder)
		for i, r := range lexicalOrder {
			if got[i].Id != r.Id {
				t.Fatalf("position %d: got %q, want %q", i, got[i].GetString("title"), r.GetString("title"))
			}
		}
	})

	t.Run("vector count mismatch returns input order unchanged", func(t *testing.T) {
		t.Setenv("BALAUR_EMBED_RERANK", "1")
		client := &llmtest.ScriptedClient{
			EmbedFunc: func(texts []string) ([][]float32, error) {
				// Return one fewer vector than len(candidates)+1.
				return [][]float32{{1, 0}, {0, 1}}, nil
			},
		}
		got := RerankByEmbedding(context.Background(), client, "query text", lexicalOrder)
		for i, r := range lexicalOrder {
			if got[i].Id != r.Id {
				t.Fatalf("position %d: got %q, want %q", i, got[i].GetString("title"), r.GetString("title"))
			}
		}
	})

	t.Run("nil client returns input order unchanged", func(t *testing.T) {
		t.Setenv("BALAUR_EMBED_RERANK", "1")
		got := RerankByEmbedding(context.Background(), nil, "query text", lexicalOrder)
		for i, r := range lexicalOrder {
			if got[i].Id != r.Id {
				t.Fatalf("position %d: got %q, want %q", i, got[i].GetString("title"), r.GetString("title"))
			}
		}
	})

	t.Run("fewer than 2 candidates returns unchanged", func(t *testing.T) {
		t.Setenv("BALAUR_EMBED_RERANK", "1")
		client := &llmtest.ScriptedClient{EmbedFunc: scriptedVecs}
		single := []*core.Record{recA}
		got := RerankByEmbedding(context.Background(), client, "query text", single)
		if len(got) != 1 || got[0].Id != recA.Id {
			t.Fatalf("single-element slice was changed unexpectedly")
		}
		got = RerankByEmbedding(context.Background(), client, "query text", nil)
		if got != nil {
			t.Fatalf("nil slice should return nil, got %v", got)
		}
	})
}

func TestCosine(t *testing.T) {
	const eps = 1e-6

	tests := []struct {
		name string
		a, b []float32
		want float64
	}{
		{
			name: "identical unit vectors",
			a:    []float32{1, 0},
			b:    []float32{1, 0},
			want: 1.0,
		},
		{
			name: "orthogonal vectors",
			a:    []float32{1, 0},
			b:    []float32{0, 1},
			want: 0.0,
		},
		{
			name: "length mismatch",
			a:    []float32{1, 0},
			b:    []float32{1, 0, 0},
			want: 0.0,
		},
		{
			name: "zero vector a",
			a:    []float32{0, 0},
			b:    []float32{1, 0},
			want: 0.0,
		},
		{
			name: "zero vector b",
			a:    []float32{1, 0},
			b:    []float32{0, 0},
			want: 0.0,
		},
		{
			name: "empty slices",
			a:    []float32{},
			b:    []float32{},
			want: 0.0,
		},
		{
			name: "45-degree angle",
			a:    []float32{1, 0},
			b:    []float32{1, 1},
			want: 1.0 / math.Sqrt2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cosine(tc.a, tc.b)
			if math.Abs(got-tc.want) > eps {
				t.Fatalf("cosine(%v, %v) = %f, want ~%f", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
