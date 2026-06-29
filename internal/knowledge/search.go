package knowledge

import (
	"sort"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/search"
)

// search.go is the active-knowledge search surface: searchActiveNodes is the
// shared FTS5-or-substring-fallback skeleton; SearchActive is the memory-scoped
// recall path; SearchAllActive is the cross-type surface. Split out of
// knowledge.go (plan 204).

// searchActiveNodes is the shared FTS5-or-fallback skeleton behind SearchActive
// and SearchAllActive. It runs the FTS query, keeps only records that pass the
// `keep` predicate (the consent status=active filter lives there), ranks the
// survivors by the FTS id order, caps to limit, and returns them. When no index
// is present or the FTS path yields nothing, it delegates to `fallback`, which
// owns its own source query, match predicate, ordering, and limit cap.
//
//   - query    returns the FTS-ranked node ids for `terms`.
//   - keep     decides whether an FTS-hydrated record is returned, and may mutate
//     it (e.g. hydrate memory aliases). Return (record, true) to keep.
//   - fallback yields the already-matched, already-ordered, already-capped
//     substring-scan result when the FTS path does not produce hits.
func searchActiveNodes(
	app core.App,
	terms []string,
	limit int,
	query func(ix *search.Index) ([]string, error),
	keep func(r *core.Record) (*core.Record, bool),
	fallback func() ([]*core.Record, error),
) ([]*core.Record, error) {
	// --- FTS5 fast path ---
	if raw, ok := app.Store().GetOk(search.StoreKey); ok {
		if ix, ok := raw.(*search.Index); ok && ix != nil {
			ids, err := query(ix)
			if err == nil && len(ids) > 0 {
				recs, err := app.FindRecordsByIds("nodes", ids)
				if err == nil {
					var active []*core.Record
					for _, r := range recs {
						if kept, ok := keep(r); ok {
							active = append(active, kept)
						}
					}
					if len(active) > 0 {
						order := make(map[string]int, len(ids))
						for i, id := range ids {
							order[id] = i
						}
						sort.Slice(active, func(i, j int) bool {
							return order[active[i].Id] < order[active[j].Id]
						})
						if limit > 0 && len(active) > limit {
							active = active[:limit]
						}
						return active, nil
					}
				}
			}
		}
	}

	// --- fallback (per-caller: source, match predicate, ordering, limit cap) ---
	return fallback()
}

// SearchActive finds active memories matching any of the given terms. When a
// FTS5 sidecar index is available it is bm25-ranked; otherwise it falls back to
// a deterministic substring scan over the active memory nodes.
func SearchActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	return searchActiveNodes(app, terms, limit,
		func(ix *search.Index) ([]string, error) {
			return ix.QueryKind(terms, string(Memory), limit)
		},
		func(r *core.Record) (*core.Record, bool) {
			if r.GetString("type") == string(Memory) && r.GetString("status") == StatusActive {
				return hydrate(Memory, r), true
			}
			return nil, false
		},
		func() ([]*core.Record, error) {
			recs, err := nodes.ListByTypeStatus(app, string(Memory), StatusActive)
			if err != nil {
				return nil, err
			}
			hydrateAll(Memory, recs)
			var matched []*core.Record
			for _, r := range recs {
				for _, t := range terms {
					t = strings.ToLower(strings.TrimSpace(t))
					if t == "" {
						continue
					}
					if matchesQuery(Memory, r, t) {
						matched = append(matched, r)
						break
					}
				}
			}
			sort.SliceStable(matched, func(i, j int) bool {
				return matched[i].GetInt("importance") > matched[j].GetInt("importance")
			})
			if limit > 0 && len(matched) > limit {
				matched = matched[:limit]
			}
			return matched, nil
		},
	)
}

// SearchAllActive is the cross-type search surface: it returns active nodes of
// ANY type matching the terms, ranked by bm25 when the FTS5 sidecar is
// available. Unlike SearchActive (which stays memory-scoped and hydrates memory
// aliases for context/recall callers), this returns RAW node records — the
// caller renders each hit by its node `type`. A node that is not active is never
// returned (the consent filter). When the index is unavailable it falls back to
// a deterministic substring scan over active nodes' title/body.
func SearchAllActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	return searchActiveNodes(app, terms, limit,
		func(ix *search.Index) ([]string, error) {
			return ix.Query(terms, limit)
		},
		func(r *core.Record) (*core.Record, bool) {
			if r.GetString("status") == StatusActive {
				return r, true
			}
			return nil, false
		},
		func() ([]*core.Record, error) {
			recs, err := app.FindRecordsByFilter(
				"nodes", "status = 'active'", "-updated,-created", 0, 0, nil)
			if err != nil {
				return nil, err
			}
			var matched []*core.Record
			for _, r := range recs {
				for _, t := range terms {
					t = strings.ToLower(strings.TrimSpace(t))
					if t == "" {
						continue
					}
					if strings.Contains(strings.ToLower(r.GetString("title")), t) ||
						strings.Contains(strings.ToLower(r.GetString("body")), t) {
						matched = append(matched, r)
						break
					}
				}
			}
			if limit > 0 && len(matched) > limit {
				matched = matched[:limit]
			}
			return matched, nil
		},
	)
}
