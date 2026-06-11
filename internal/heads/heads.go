// Package heads manages Balaur's sub-agent identities. A head is an auth
// record in the `heads` collection; what it may touch is written as rows in
// `grants`. Every access decision goes through Allow, and every decision —
// allowed or denied — lands in `audit_log`.
//
// THE RULE BOUNDARY (AGENTS.md): Go-side record access bypasses PocketBase
// API rules by design. Tool code acting for a head must therefore never call
// app.Find*/app.Save directly; it calls Scoped(app, headID).Records(...) /
// .Save(...), which checks grants and audits first.
package heads

import (
	"errors"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
)

// ErrDenied is returned when a head lacks a grant for the requested access.
var ErrDenied = errors.New("heads: access denied")

// SpawnOption is a functional option for Spawn.
type SpawnOption func(*core.Record)

// WithAvatar assigns a Balaur head personality (e.g. "balaur-05" Wild) to the
// spawned head. When a sub-head chat UI renders the conversation, it will use
// this avatar instead of the owner's default Balaur head.
// See store.BalaurAvatarMap for valid keys (balaur-01…balaur-16).
func WithAvatar(key string) SpawnOption {
	return func(r *core.Record) {
		if key != "" {
			r.Set("balaur_avatar", key)
		}
	}
}

// Spawn creates a head with the given grants and returns the head record
// plus a short-lived static auth token for it. The token is how the head's
// identity travels through the system; it is never persisted.
func Spawn(app core.App, name, purpose string, ttl time.Duration, grants []Grant, opts ...SpawnOption) (*core.Record, string, error) {
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		return nil, "", fmt.Errorf("finding heads collection: %w", err)
	}

	head := core.NewRecord(col)
	head.Set("name", name)
	head.Set("purpose", purpose)
	head.Set("status", "active")
	head.Set("expires", time.Now().Add(ttl).UTC())
	// Auth records require an email/password pair internally; heads cannot
	// log in (PasswordAuth disabled), but the fields must be set.
	head.SetEmail(fmt.Sprintf("head-%d@balaur.local", time.Now().UnixNano()))
	head.SetRandomPassword()

	// Apply options before save so all fields land in one write.
	for _, opt := range opts {
		opt(head)
	}

	if err := app.Save(head); err != nil {
		return nil, "", fmt.Errorf("saving head: %w", err)
	}

	for _, g := range grants {
		if err := writeGrant(app, head.Id, g); err != nil {
			return nil, "", err
		}
	}

	token, err := head.NewStaticAuthToken(ttl)
	if err != nil {
		return nil, "", fmt.Errorf("minting head token: %w", err)
	}

	store.Audit(app, head.Id, "runtime", "head.spawn", name, true, map[string]any{"ttl": ttl.String()})
	return head, token, nil
}

// Grant describes one permission row: a head may read and/or write one
// target collection until expiry.
type Grant struct {
	Target string // conversations | messages | memories | skills
	Read   bool
	Write  bool
}

func writeGrant(app core.App, headID string, g Grant) error {
	col, err := app.FindCollectionByNameOrId("grants")
	if err != nil {
		return fmt.Errorf("finding grants collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("head", headID)
	rec.Set("target", g.Target)
	rec.Set("read", g.Read)
	rec.Set("write", g.Write)
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("saving grant %s: %w", g.Target, err)
	}
	return nil
}

// Resolve verifies a head token and returns the head record. Expired or
// revoked heads fail closed.
func Resolve(app core.App, token string) (*core.Record, error) {
	head, err := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
	if err != nil {
		return nil, fmt.Errorf("verifying head token: %w", err)
	}
	if head.Collection().Name != "heads" {
		return nil, ErrDenied
	}
	if head.GetString("status") != "active" {
		return nil, ErrDenied
	}
	return head, nil
}

// Merge closes a head: marks it merged and deletes its grants. Its tokens
// keep verifying until expiry, so status — checked in Resolve and the
// Scoped path — is the kill switch.
func Merge(app core.App, headID string) error {
	return closeHead(app, headID, "merged", "head.merge")
}

// Revoke closes a head without merging (abort path).
func Revoke(app core.App, headID string) error {
	return closeHead(app, headID, "revoked", "head.revoke")
}

func closeHead(app core.App, headID, status, action string) error {
	head, err := app.FindRecordById("heads", headID)
	if err != nil {
		return fmt.Errorf("finding head: %w", err)
	}
	head.Set("status", status)
	if err := app.Save(head); err != nil {
		return fmt.Errorf("updating head status: %w", err)
	}
	grants, err := app.FindRecordsByFilter("grants", "head = {:head}", "", 0, 0, dbx.Params{"head": headID})
	if err != nil {
		return fmt.Errorf("listing grants: %w", err)
	}
	for _, g := range grants {
		if err := app.Delete(g); err != nil {
			return fmt.Errorf("deleting grant: %w", err)
		}
	}
	store.Audit(app, headID, "runtime", action, head.GetString("name"), true, nil)
	return nil
}
