package heads

import (
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Scoped is the ONLY sanctioned data path for code acting on behalf of a
// head. It checks grants before touching records and audits every decision.
type Scoped struct {
	app  core.App
	head *core.Record
}

// AsHead wraps app access in the head's permission scope.
func AsHead(app core.App, head *core.Record) *Scoped {
	return &Scoped{app: app, head: head}
}

// allow checks status, expiry, and the grant table for one access. It
// audits the decision either way and returns ErrDenied on failure.
func (s *Scoped) allow(target, mode string) error {
	allowed := s.check(target, mode)
	audit(s.app, s.head.Id, "head:"+s.head.GetString("name"), "access."+mode, target, allowed, nil)
	if !allowed {
		return fmt.Errorf("%w: head %q may not %s %s", ErrDenied, s.head.GetString("name"), mode, target)
	}
	return nil
}

func (s *Scoped) check(target, mode string) bool {
	if s.head.GetString("status") != "active" {
		return false
	}
	if exp := s.head.GetDateTime("expires"); !exp.IsZero() && exp.Time().Before(time.Now()) {
		return false
	}
	grants, err := s.app.FindRecordsByFilter(
		"grants",
		"head = {:head} && target = {:target}",
		"", 0, 0,
		dbx.Params{"head": s.head.Id, "target": target},
	)
	if err != nil {
		return false // fail closed
	}
	for _, g := range grants {
		if exp := g.GetDateTime("expires"); !exp.IsZero() && exp.Time().Before(time.Now()) {
			continue
		}
		if g.GetBool(mode) {
			return true
		}
	}
	return false
}

// Records lists records from a collection the head holds a read grant for.
func (s *Scoped) Records(collection, filter, sort string, limit int, params dbx.Params) ([]*core.Record, error) {
	if err := s.allow(collection, "read"); err != nil {
		return nil, err
	}
	return s.app.FindRecordsByFilter(collection, filter, sort, limit, 0, params)
}

// Save writes a record if the head holds a write grant for its collection.
func (s *Scoped) Save(record *core.Record) error {
	if err := s.allow(record.Collection().Name, "write"); err != nil {
		return err
	}
	return s.app.Save(record)
}
