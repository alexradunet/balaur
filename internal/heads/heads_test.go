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
