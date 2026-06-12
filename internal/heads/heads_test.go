package heads

import (
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	// Blank import registers the Balaur schema migrations, which
	// tests.NewTestApp runs during bootstrap.
	_ "github.com/alexradunet/balaur/migrations"
)

// newApp builds a throwaway PocketBase app with the Balaur schema applied.
func newApp(t *testing.T) core.App {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

func seedMemory(t *testing.T, app core.App, title string) {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("memories")
	if err != nil {
		t.Fatalf("memories collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("title", title)
	rec.Set("content", "private")
	rec.Set("status", "active")
	if err := app.Save(rec); err != nil {
		t.Fatalf("seeding memory: %v", err)
	}
}

func countAudit(t *testing.T, app core.App, action string, allowed bool) int {
	t.Helper()
	recs, err := app.FindRecordsByFilter(
		"audit_log",
		"action = {:action} && allowed = {:allowed}",
		"", 0, 0,
		dbx.Params{"action": action, "allowed": allowed},
	)
	if err != nil {
		t.Fatalf("querying audit_log: %v", err)
	}
	return len(recs)
}

// The headline invariant: a head with no grant on a collection cannot read
// it through the scoped path, and the denial is audited.
func TestScopedDeniesUngrantedAccess(t *testing.T) {
	app := newApp(t)
	seedMemory(t, app, "secret journal")

	// Tax head: may read conversations, nothing else.
	head, token, err := Spawn(app, "tax", "do my taxes", time.Hour, []Grant{
		{Target: "conversations", Read: true},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if token == "" {
		t.Fatal("expected a token")
	}

	scoped := AsHead(app, head)

	if _, err := scoped.Records("memories", "", "", 0, nil); err == nil {
		t.Fatal("tax head read memories without a grant")
	}
	if got := countAudit(t, app, "access.read", false); got != 1 {
		t.Fatalf("expected 1 denied audit row, got %d", got)
	}

	// The granted collection still works.
	if _, err := scoped.Records("conversations", "", "", 0, nil); err != nil {
		t.Fatalf("granted read failed: %v", err)
	}
	if got := countAudit(t, app, "access.read", true); got != 1 {
		t.Fatalf("expected 1 allowed audit row, got %d", got)
	}
}

func TestScopedWriteRequiresWriteGrant(t *testing.T) {
	app := newApp(t)

	head, _, err := Spawn(app, "reader", "read-only head", time.Hour, []Grant{
		{Target: "memories", Read: true}, // read, not write
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	col, _ := app.FindCollectionByNameOrId("memories")
	rec := core.NewRecord(col)
	rec.Set("title", "smuggled")

	if err := AsHead(app, head).Save(rec); err == nil {
		t.Fatal("read-only head wrote a memory")
	}
}

func TestMergeRevokesAccess(t *testing.T) {
	app := newApp(t)

	head, token, err := Spawn(app, "branch", "focused work", time.Hour, []Grant{
		{Target: "memories", Read: true},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	if err := Merge(app, head.Id); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	// Token no longer resolves: status is the kill switch.
	if _, err := Resolve(app, token); err == nil {
		t.Fatal("merged head token still resolves")
	}

	// And the scoped path fails closed even with a stale record handle.
	head.Set("status", "merged") // reflect persisted state on the in-memory copy
	if _, err := AsHead(app, head).Records("memories", "", "", 0, nil); err == nil {
		t.Fatal("merged head still reads")
	}
}

func TestResolveRejectsNonHeadTokens(t *testing.T) {
	app := newApp(t)

	// A token from the human owner's collection must not resolve as a head.
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("users collection: %v", err)
	}
	user := core.NewRecord(users)
	user.SetEmail("owner@balaur.local")
	user.SetRandomPassword()
	if err := app.Save(user); err != nil {
		t.Fatalf("saving user: %v", err)
	}
	token, err := user.NewStaticAuthToken(time.Hour)
	if err != nil {
		t.Fatalf("minting user token: %v", err)
	}

	if _, err := Resolve(app, token); err == nil {
		t.Fatal("user token resolved as a head")
	}
}

func TestExpiredHeadDeniesAccess(t *testing.T) {
	app := newApp(t)
	seedMemory(t, app, "private note")

	head, _, err := Spawn(app, "expiry-head", "will expire", time.Hour, []Grant{
		{Target: "memories", Read: true},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	// Back-date the expires field so the head is already expired.
	head.Set("expires", time.Now().Add(-time.Minute).UTC())
	if err := app.Save(head); err != nil {
		t.Fatalf("back-dating head: %v", err)
	}

	if _, err := AsHead(app, head).Records("memories", "", "", 0, nil); err == nil {
		t.Fatal("expired head should be denied")
	}
	if got := countAudit(t, app, "access.read", false); got < 1 {
		t.Fatalf("expected at least 1 denied audit row, got %d", got)
	}
}

func TestExpiredGrantDeniesAccess(t *testing.T) {
	app := newApp(t)
	seedMemory(t, app, "private note")

	head, _, err := Spawn(app, "grant-expiry", "grant will expire", time.Hour, []Grant{
		{Target: "memories", Read: true},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	// Back-date the grant's expires field.
	grants, err := app.FindRecordsByFilter("grants", "head = {:head}", "", 0, 0, dbx.Params{"head": head.Id})
	if err != nil || len(grants) == 0 {
		t.Fatalf("no grants found: %v", err)
	}
	grants[0].Set("expires", time.Now().Add(-time.Minute).UTC())
	if err := app.Save(grants[0]); err != nil {
		t.Fatalf("back-dating grant: %v", err)
	}

	if _, err := AsHead(app, head).Records("memories", "", "", 0, nil); err == nil {
		t.Fatal("expired grant should be denied")
	}
	if got := countAudit(t, app, "access.read", false); got < 1 {
		t.Fatalf("expected at least 1 denied audit row, got %d", got)
	}
}

func TestCrossHeadIsolation(t *testing.T) {
	app := newApp(t)

	headA, _, err := Spawn(app, "head-a", "reads memories only", time.Hour, []Grant{
		{Target: "memories", Read: true},
	})
	if err != nil {
		t.Fatalf("Spawn A: %v", err)
	}
	headB, _, err := Spawn(app, "head-b", "reads skills only", time.Hour, []Grant{
		{Target: "skills", Read: true},
	})
	if err != nil {
		t.Fatalf("Spawn B: %v", err)
	}

	// A cannot read skills.
	if _, err := AsHead(app, headA).Records("skills", "", "", 0, nil); err == nil {
		t.Fatal("head A read skills without a grant")
	}
	// B cannot read memories.
	if _, err := AsHead(app, headB).Records("memories", "", "", 0, nil); err == nil {
		t.Fatal("head B read memories without a grant")
	}
	// A can read memories.
	if _, err := AsHead(app, headA).Records("memories", "", "", 0, nil); err != nil {
		t.Fatalf("head A cannot read memories: %v", err)
	}
	// B can read skills.
	if _, err := AsHead(app, headB).Records("skills", "", "", 0, nil); err != nil {
		t.Fatalf("head B cannot read skills: %v", err)
	}
}

func TestRecordsCapResultSize(t *testing.T) {
	app := newApp(t)

	head, _, err := Spawn(app, "cap-head", "tests result cap", time.Hour, []Grant{
		{Target: "memories", Read: true},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	scoped := AsHead(app, head)

	// Seed 3 memory records.
	for _, title := range []string{"mem-a", "mem-b", "mem-c"} {
		seedMemory(t, app, title)
	}

	// limit=0 maps to cap, not zero results.
	rows, err := scoped.Records("memories", "", "", 0, nil)
	if err != nil {
		t.Fatalf("Records(limit=0): %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("expected 3 rows with limit=0 (uses cap), got %d", len(rows))
	}

	// limit=10_000 is silently capped to maxScopedRecords; 3 rows fit within cap.
	rows2, err := scoped.Records("memories", "", "", 10_000, nil)
	if err != nil {
		t.Fatalf("Records(limit=10000): %v", err)
	}
	if len(rows2) > maxScopedRecords {
		t.Errorf("result size %d exceeds maxScopedRecords=%d", len(rows2), maxScopedRecords)
	}
}

func TestRevokeClosesHead(t *testing.T) {
	app := newApp(t)

	head, token, err := Spawn(app, "revoke-me", "short-lived head", time.Hour, []Grant{
		{Target: "memories", Read: true},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	if err := Revoke(app, head.Id); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	// Token no longer resolves.
	if _, err := Resolve(app, token); err == nil {
		t.Fatal("revoked head token still resolves")
	}

	// Scoped access fails even with stale in-memory record.
	head.Set("status", "revoked")
	if _, err := AsHead(app, head).Records("memories", "", "", 0, nil); err == nil {
		t.Fatal("revoked head still reads")
	}

	// The revoke action is audited.
	if got := countAudit(t, app, "head.revoke", true); got < 1 {
		t.Fatalf("expected revoke audit row, got %d", got)
	}
}
