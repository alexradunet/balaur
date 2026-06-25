package knowledge

import (
	"sort"
	"sync/atomic"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
)

// contextCacheKey is the app.Store() key under which the *contextCache singleton
// lives. Mirrors search.StoreKey's package-const style.
const contextCacheKey = "balaur.knowledge.contextCache"

// contextCacheComputes counts how many times loadContextCache ran the full
// nodes.ListByTypeStatus scan (a cache miss). It is the deterministic signal the
// warm-cache test asserts on: a warm read must NOT increment it. Test-only seam,
// kept tiny and unexported; service code never reads it.
var contextCacheComputes atomic.Int64

// contextCache holds the two sets that BuildContext injects on every turn: the
// upfront (importance>=4) active memories and the active skills, already
// filtered and ordered exactly as the uncached path produced them. It is a
// process-memory singleton (app.Store()), rebuilt lazily on the first turn after
// a boot or an invalidation; it never persists, so there is no migration.
//
// CONCURRENCY INVARIANT — load-bearing, do not break: the records held here are
// shared, read-only snapshots. A cached record is NEVER handed to app.Save. The
// per-turn knowledge.Touch (turn.go) does app.Save on the records BuildContext
// injected, so UpfrontMemories/ActiveSkills must return FRESH copies (copyForRead
// -> Record.Clone) per call; the cached slices below are mutated by no one. Were
// the cache to hand out its own pointers, one turn's Touch would mutate a record
// another goroutine is reading from the snapshot — a data race and silent
// corruption of the cached set.
type contextCache struct {
	upfront []*core.Record // computeUpfront(app, upfrontLimit) result, sorted
	skills  []*core.Record // computeActiveSkills(app) result, sorted by title
}

// computeUpfront is the uncached upfront-memory scan: the highest-importance
// (>=4) active memories, importance-desc, capped at limit. One source of truth
// for the selection semantics — UpfrontMemories reads it through the cache.
func computeUpfront(app core.App, limit int) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, string(Memory), StatusActive)
	if err != nil {
		return nil, err
	}
	hydrateAll(Memory, recs)
	var out []*core.Record
	for _, r := range recs {
		if r.GetInt("importance") >= 4 {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].GetInt("importance") > out[j].GetInt("importance")
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// computeActiveSkills is the uncached active-skill scan, ordered by title. One
// source of truth — ActiveSkills reads it through the cache.
func computeActiveSkills(app core.App) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, string(Skill), StatusActive)
	if err != nil {
		return nil, err
	}
	hydrateAll(Skill, recs)
	sort.SliceStable(recs, func(i, j int) bool {
		return recs[i].GetString("title") < recs[j].GetString("title")
	})
	return recs, nil
}

// loadContextCache returns the cached upfront/skill snapshot, computing it once
// per boot/invalidation. On a WARM cache it returns WITHOUT scanning: the GetOk
// fast-path short-circuits before any compute, so a warm turn never runs the
// nodes.ListByTypeStatus scan (the whole point of this cache). Only on a MISS
// does it compute both sets and store them via GetOrSet.
//
// A stored cache must always be a COMPLETE, correct snapshot: if either compute
// errors, loadContextCache returns the freshly computed (possibly partial) value
// WITHOUT storing it, so the next call retries rather than serving a snapshot
// that a reader could mistake for "no memories".
func loadContextCache(app core.App) (*contextCache, error) {
	if raw, ok := app.Store().GetOk(contextCacheKey); ok {
		if c, ok := raw.(*contextCache); ok && c != nil {
			return c, nil
		}
	}

	contextCacheComputes.Add(1) // miss: the scan runs (see the warm-cache test)
	upfront, err := computeUpfront(app, upfrontLimit)
	if err != nil {
		return &contextCache{}, err
	}
	skills, err := computeActiveSkills(app)
	if err != nil {
		return &contextCache{upfront: upfront}, err
	}

	fresh := &contextCache{upfront: upfront, skills: skills}
	// GetOrSet stores exactly once under the store write lock; a concurrent
	// miss that already populated the key wins, and we return its value.
	stored := app.Store().GetOrSet(contextCacheKey, func() any { return fresh })
	if c, ok := stored.(*contextCache); ok && c != nil {
		return c, nil
	}
	return fresh, nil
}

// invalidateContextCache drops the cached snapshot so the next turn recomputes
// it. Called AFTER any owner-initiated change to the ACTIVE memory/skill set or
// a cached field (status via Transition, title/body/importance via UpdateFields,
// or a hard delete via the delete hook). Deliberately NOT called from Touch:
// use_count/last_used are neither selected nor ordered on, so a Touch is a no-op
// for the cached set — invalidating on it would defeat the warm-cache win.
func invalidateContextCache(app core.App) {
	app.Store().Remove(contextCacheKey)
}

// copyForRead returns fresh Record clones of a cached slice so callers may pass
// them to Touch/app.Save without mutating the shared snapshot (see the
// CONCURRENCY INVARIANT on contextCache). Record.Clone copies collection +
// custom field data and the id; the props JSON decodes into a fresh map per
// node, so a clone's Touch never reaches back into the cached original.
func copyForRead(recs []*core.Record) []*core.Record {
	if len(recs) == 0 {
		return nil
	}
	out := make([]*core.Record, len(recs))
	for i, r := range recs {
		out[i] = r.Clone()
	}
	return out
}

// RegisterCacheInvalidation binds the delete-only seam that keeps the context
// cache correct on hard deletes. A memory/skill node dropped via nodes.Drop
// (from the tools/CLI layer) fires OnRecordAfterDeleteSuccess; the cache is
// invalidated only for memory/skill types. It is delete-ONLY, so it never fires
// on the per-turn Touch (an Update) — that Touch-safety is why the cache can be
// bound to a record hook at all. Call once at boot, next to registerSearchIndex.
func RegisterCacheInvalidation(app core.App) {
	app.OnRecordAfterDeleteSuccess("nodes").BindFunc(func(e *core.RecordEvent) error {
		switch Kind(e.Record.GetString("type")) {
		case Memory, Skill:
			invalidateContextCache(app)
		}
		return e.Next()
	})
}
