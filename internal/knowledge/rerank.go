package knowledge

import (
	"context"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
)

// rerankEnabled reports whether the embedding rerank second stage is on.
// OFF by default: deterministic lexical recall is the baseline; the
// embedding path is opt-in (BALAUR_EMBED_RERANK=1) and documented as a
// network path. See plans/073.
func rerankEnabled() bool { return os.Getenv("BALAUR_EMBED_RERANK") == "1" }

// cosine returns the cosine similarity of two equal-length vectors, in
// [-1,1]. Returns 0 for empty, length-mismatched, or zero-magnitude inputs
// so a degenerate vector can never reorder above a real match.
func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// RerankByEmbedding reorders an already-ranked candidate slice (the FTS/LIKE
// output of SearchActive) by cosine similarity between the query and each
// candidate's text. It is a SECOND ranking stage: candidates is assumed to be
// the small top-K from the lexical stage, NOT the whole memory store.
//
// It is opt-in and degrades cleanly to the input order unchanged when:
//   - the rerank flag is off (rerankEnabled()),
//   - client is nil,
//   - Embed returns an error or a vector count that doesn't match,
//   - candidates has 0 or 1 element.
//
// This preserves the deterministic-offline default: absent embeddings, recall
// behaves exactly as the lexical stage alone.
//
// NETWORK: Embed may hit the configured embed model. By design the caller
// passes the LOCAL embed client; memory content must not be sent to a remote
// provider by default (README/AGENTS safety rule).
func RerankByEmbedding(ctx context.Context, client llm.Client, query string, candidates []*core.Record) []*core.Record {
	if !rerankEnabled() || client == nil || len(candidates) < 2 {
		return candidates
	}

	// Build the embed input: query at index 0, one text per candidate after.
	// Candidate text mirrors writeMemoryLine: title + ": " + content.
	// These are the same fields the FTS index uses (title, content).
	texts := make([]string, 1+len(candidates))
	texts[0] = query
	for i, r := range candidates {
		title := r.GetString("title")
		content := strings.TrimSpace(r.GetString("content"))
		if content != "" && content != title {
			texts[i+1] = title + ": " + content
		} else {
			texts[i+1] = title
		}
	}

	vecs, err := client.Embed(ctx, texts)
	if err != nil || len(vecs) != len(candidates)+1 {
		// Clean degrade: any embed failure leaves FTS order intact.
		return candidates
	}

	queryVec := vecs[0]

	// Score each candidate by cosine similarity to the query vector.
	type scored struct {
		rec   *core.Record
		score float64
	}
	scored_ := make([]scored, len(candidates))
	for i, r := range candidates {
		scored_[i] = scored{rec: r, score: cosine(queryVec, vecs[i+1])}
	}

	// Stable sort descending: equal scores keep their original (lexical) order.
	sort.SliceStable(scored_, func(i, j int) bool {
		return scored_[i].score > scored_[j].score
	})

	out := make([]*core.Record, len(candidates))
	for i, s := range scored_ {
		out[i] = s.rec
	}
	return out
}
