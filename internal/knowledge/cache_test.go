package knowledge

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

// activeMemory proposes a memory at the given importance and activates it,
// returning its node id. Helper for the cache tests.
func activeMemory(t *testing.T, app core.App, title string, importance int) string {
	t.Helper()
	rec, err := ProposeMemory(app, MemoryProposal{
		Title:      title,
		Content:    title + " body",
		Importance: importance,
		Source:     "chat",
	})
	if err != nil {
		t.Fatalf("ProposeMemory(%q): %v", title, err)
	}
	if _, err := Transition(app, Memory, rec.Id, StatusActive); err != nil {
		t.Fatalf("activate memory %q: %v", title, err)
	}
	return rec.Id
}

// activeSkill proposes a skill and activates it, returning its node id.
func activeSkill(t *testing.T, app core.App, name string) string {
	t.Helper()
	rec, err := ProposeSkill(app, SkillProposal{
		Name:        name,
		Description: name + " desc",
		Content:     name + " procedure",
		WhenToUse:   "when " + name,
	})
	if err != nil {
		t.Fatalf("ProposeSkill(%q): %v", name, err)
	}
	if _, err := Transition(app, Skill, rec.Id, StatusActive); err != nil {
		t.Fatalf("activate skill %q: %v", name, err)
	}
	return rec.Id
}

func ids(recs []*core.Record) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.Id
	}
	return out
}

func contains(recs []*core.Record, id string) bool {
	for _, r := range recs {
		if r.Id == id {
			return true
		}
	}
	return false
}

func mustUpfront(t *testing.T, app core.App) []*core.Record {
	t.Helper()
	recs, err := UpfrontMemories(app, upfrontLimit)
	if err != nil {
		t.Fatalf("UpfrontMemories: %v", err)
	}
	return recs
}

func mustSkills(t *testing.T, app core.App) []*core.Record {
	t.Helper()
	recs, err := ActiveSkills(app)
	if err != nil {
		t.Fatalf("ActiveSkills: %v", err)
	}
	return recs
}

// TestCacheParity asserts the cached path selects/orders exactly as the uncached
// logic did: upfront keeps only importance>=4, importance-desc; skills ordered by
// title asc. A second (warm) read returns the same ids in the same order.
func TestCacheParity(t *testing.T) {
	app := storetest.NewApp(t)

	hi := activeMemory(t, app, "high", 5)
	mid := activeMemory(t, app, "mid", 4)
	low := activeMemory(t, app, "low", 2) // below threshold, excluded

	activeSkill(t, app, "Zebra")
	activeSkill(t, app, "Alpha")

	up := mustUpfront(t, app)
	if contains(up, low) {
		t.Fatalf("low-importance memory leaked into upfront: %v", ids(up))
	}
	if len(up) != 2 || up[0].Id != hi || up[1].Id != mid {
		t.Fatalf("upfront order = %v, want [%s %s] (importance desc)", ids(up), hi, mid)
	}

	sk := mustSkills(t, app)
	if len(sk) != 2 || sk[0].GetString("name") != "Alpha" || sk[1].GetString("name") != "Zebra" {
		t.Fatalf("skills order = %v, want [Alpha Zebra]", []string{sk[0].GetString("name"), sk[1].GetString("name")})
	}

	// Warm read: same ids, same order.
	up2 := mustUpfront(t, app)
	if len(up2) != len(up) {
		t.Fatalf("warm upfront len = %d, want %d", len(up2), len(up))
	}
	for i := range up {
		if up[i].Id != up2[i].Id {
			t.Fatalf("warm upfront[%d] = %s, want %s", i, up2[i].Id, up[i].Id)
		}
	}
}

// TestCacheAddInvalidates proves Transition-to-active drops the cache: a memory
// approved after the cache was warmed shows up on the next read.
func TestCacheAddInvalidates(t *testing.T) {
	app := storetest.NewApp(t)
	activeMemory(t, app, "first", 5)
	_ = mustUpfront(t, app) // warm

	added := activeMemory(t, app, "added", 5)
	if !contains(mustUpfront(t, app), added) {
		t.Fatal("newly approved memory missing — Transition-to-active did not invalidate")
	}
}

// TestCacheDropInvalidates is the NON-NEGOTIABLE consent guarantee: a hard-deleted
// memory must not keep being injected. Uses the delete hook (option a).
func TestCacheDropInvalidates(t *testing.T) {
	app := storetest.NewApp(t)
	RegisterCacheInvalidation(app) // storetest does not run main.go

	id := activeMemory(t, app, "secret", 5)
	if !contains(mustUpfront(t, app), id) {
		t.Fatal("memory missing before drop")
	}
	if err := nodes.Drop(app, id); err != nil {
		t.Fatalf("Drop: %v", err)
	}
	if contains(mustUpfront(t, app), id) {
		t.Fatal("DROPPED memory still injected — stale cache leaks a deleted memory")
	}
}

// TestCacheArchiveInvalidates: archiving moves the node out of active.
func TestCacheArchiveInvalidates(t *testing.T) {
	app := storetest.NewApp(t)
	id := activeMemory(t, app, "archive-me", 5)
	_ = mustUpfront(t, app) // warm

	if _, err := Transition(app, Memory, id, StatusArchived); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if contains(mustUpfront(t, app), id) {
		t.Fatal("archived memory still injected — archive did not invalidate")
	}
}

// TestCacheEditInvalidates: dropping importance below 4 removes it from upfront.
func TestCacheEditInvalidates(t *testing.T) {
	app := storetest.NewApp(t)
	id := activeMemory(t, app, "demote-me", 5)
	_ = mustUpfront(t, app) // warm

	if _, err := UpdateFields(app, Memory, id, map[string]string{"importance": "2"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}
	if contains(mustUpfront(t, app), id) {
		t.Fatal("demoted memory still injected — importance edit did not invalidate")
	}
}

// TestCacheSkillInvalidates: archiving a skill removes it from ActiveSkills.
func TestCacheSkillInvalidates(t *testing.T) {
	app := storetest.NewApp(t)
	id := activeSkill(t, app, "TempSkill")
	if !contains(mustSkills(t, app), id) {
		t.Fatal("skill missing before archive")
	}
	if _, err := Transition(app, Skill, id, StatusArchived); err != nil {
		t.Fatalf("archive skill: %v", err)
	}
	if contains(mustSkills(t, app), id) {
		t.Fatal("archived skill still indexed — skill archive did not invalidate")
	}
}

// TestCacheTouchDoesNotInvalidate is the warm-cache regression guard (round 1):
// the per-turn Touch must NOT remove the cache key, or the warm path would never
// exist.
func TestCacheTouchDoesNotInvalidate(t *testing.T) {
	app := storetest.NewApp(t)
	activeMemory(t, app, "warm", 5)

	up := mustUpfront(t, app) // warm
	if len(up) != 1 {
		t.Fatalf("expected 1 upfront, got %d", len(up))
	}
	if _, ok := app.Store().GetOk(contextCacheKey); !ok {
		t.Fatal("cache key absent after warming — UpfrontMemories did not populate it")
	}

	Touch(app, Memory, up[0])

	if _, ok := app.Store().GetOk(contextCacheKey); !ok {
		t.Fatal("Touch removed the cache key — warm-cache win lost (Touch must not invalidate)")
	}
}

// TestCacheReturnedRecordIsThrowaway is the DATA-RACE regression guard (round 2):
// Touch(app, Memory, rec) on a record returned by UpfrontMemories must NOT mutate
// the cached snapshot. Reverting copyForRead to `return recs` makes this FAIL.
func TestCacheReturnedRecordIsThrowaway(t *testing.T) {
	app := storetest.NewApp(t)
	activeMemory(t, app, "throwaway", 5)

	up := mustUpfront(t, app) // warm
	if len(up) != 1 {
		t.Fatalf("expected 1 upfront, got %d", len(up))
	}
	before := nodes.PropInt(up[0], "use_count")

	// Touch the RETURNED record (turn.go does exactly this after a turn).
	Touch(app, Memory, up[0])

	// Re-read the cached snapshot. If the returned record had been the cached
	// pointer, Touch's app.Save would have bumped the cached use_count.
	again := mustUpfront(t, app)
	if len(again) != 1 {
		t.Fatalf("expected 1 upfront on re-read, got %d", len(again))
	}
	if after := nodes.PropInt(again[0], "use_count"); after != before {
		t.Fatalf("cached snapshot use_count changed %d -> %d — Touch mutated the cache (copyForRead broken)", before, after)
	}
}

// TestCacheWarmReadDoesNotRescan is the CORE perf guard (round 4): a warm read
// must NOT re-run the nodes.ListByTypeStatus scan. We assert on the deterministic
// compute counter (incremented only on a cache MISS): the first read computes
// once, subsequent warm reads compute zero more times, and only an invalidation
// causes the next read to recompute. Reverting loadContextCache to compute
// unconditionally (the round-4 bug) makes the warm-read assertion FAIL.
//
// We ALSO mutate a memory hook-free via app.Save and confirm the warm read does
// not reflect it (a second, behavioral witness that no fresh scan ran).
func TestCacheWarmReadDoesNotRescan(t *testing.T) {
	app := storetest.NewApp(t)
	id := activeMemory(t, app, "stale-probe", 5)

	start := contextCacheComputes.Load()
	if !contains(mustUpfront(t, app), id) { // miss: warms the cache, computes once
		t.Fatal("memory missing before warm")
	}
	if got := contextCacheComputes.Load() - start; got != 1 {
		t.Fatalf("first (cold) read computed %d times, want 1", got)
	}

	// Raw, hook-free demotion: load by id and app.Save directly. This bypasses
	// every invalidation seam, so a warm read MUST still serve the stale snapshot.
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	props := nodes.Props(rec)
	props["importance"] = 1
	rec.Set("props", props)
	if err := app.Save(rec); err != nil {
		t.Fatalf("raw save: %v", err)
	}

	warm := contextCacheComputes.Load()
	if !contains(mustUpfront(t, app), id) {
		t.Fatal("warm read reflected a hook-free change — it re-scanned instead of serving the snapshot")
	}
	_ = mustUpfront(t, app) // another warm read
	if got := contextCacheComputes.Load() - warm; got != 0 {
		t.Fatalf("warm reads computed %d times, want 0 — the scan re-ran on a warm cache", got)
	}

	// Now invalidate explicitly and confirm the next read recomputes (and now
	// reflects the demotion: importance 1 < 4, so it drops out of upfront).
	invalidateContextCache(app)
	if contains(mustUpfront(t, app), id) {
		t.Fatal("after invalidation the demoted memory is still injected — recompute did not run")
	}
	if got := contextCacheComputes.Load() - warm; got != 1 {
		t.Fatalf("post-invalidation reads computed %d times, want exactly 1", got)
	}
}
