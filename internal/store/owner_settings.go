package store

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// AvatarEntry is one selectable avatar: key (stored in owner_settings /
// head records), human label, and served URL. The exported rosters are the
// single source of truth — web option builders iterate these.
type AvatarEntry struct {
	Key   string
	Label string
	URL   string
}

// avatarMap builds a key→URL lookup from a roster slice. The slices returned
// by SoulAvatars / BalaurHeads are the single source of truth; the package-level
// lookup maps below are derived from them at init so an avatar is declared once.
func avatarMap(entries []AvatarEntry) map[string]string {
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Key] = e.URL
	}
	return m
}

// GetOwnerSetting returns the value of a key from the owner_settings
// collection. Returns defaultVal if the key is not found or any error occurs.
func GetOwnerSetting(app core.App, key, defaultVal string) string {
	rec, err := app.FindFirstRecordByData("owner_settings", "key", key)
	if err != nil {
		return defaultVal
	}
	v := rec.GetString("value")
	if v == "" {
		return defaultVal
	}
	return v
}

// SetOwnerSetting upserts a key/value pair in owner_settings. The collection
// has a UNIQUE index on key, so two concurrent writers that both miss the
// initial lookup would otherwise collide on insert; on a failed save we retry
// once, by which point the row exists and the retry updates it.
func SetOwnerSetting(app core.App, key, value string) error {
	col, err := app.FindCollectionByNameOrId("owner_settings")
	if err != nil {
		return err
	}
	save := func() error {
		rec, err := app.FindFirstRecordByData("owner_settings", "key", key)
		if err != nil {
			rec = core.NewRecord(col)
			rec.Set("key", key)
		}
		rec.Set("value", value)
		return app.Save(rec)
	}
	if err := save(); err != nil {
		// A concurrent insert may have created the row between our lookup and
		// save (UNIQUE on key). Retry once: the row now exists, so we update it.
		if err := save(); err != nil {
			return fmt.Errorf("set owner setting %q: %w", key, err)
		}
	}
	return nil
}

// ── Soul avatar ────────────────────────────────────────────────────────

// soulAvatarMap is the key→URL lookup for soul avatars, derived from the
// SoulAvatars roster (the single source of truth). Legacy values "male" and
// "female" are kept as aliases for owner_settings written before the soul-NN
// keys existed.
var soulAvatarMap = func() map[string]string {
	m := avatarMap(SoulAvatars())
	m["male"] = m["soul-01"]   // legacy alias
	m["female"] = m["soul-02"] // legacy alias
	return m
}()

// ValidSoulAvatarKey reports whether key is a recognised soul avatar.
func ValidSoulAvatarKey(key string) bool {
	_, ok := soulAvatarMap[key]
	return ok
}

// SoulAvatars returns the roster of 16 soul avatars, the single source of truth.
func SoulAvatars() []AvatarEntry {
	return []AvatarEntry{
		// Basm world — human characters
		{"soul-01", "Him", "/static/avatars/soul-01.png"},
		{"soul-02", "Her", "/static/avatars/soul-02.png"},
		{"soul-03", "Elder", "/static/avatars/soul-03.png"},
		{"soul-04", "Youth", "/static/avatars/soul-04.png"},
		{"soul-05", "Maker", "/static/avatars/soul-05.png"},
		{"soul-06", "Cyclops", "/static/avatars/soul-06.png"},
		{"soul-07", "Gnome", "/static/avatars/soul-07.png"},
		{"soul-08", "Ogre", "/static/avatars/soul-08.png"},
		// Romanian mythological creatures
		{"soul-09", "Strigoi", "/static/avatars/soul-09.png"},
		{"soul-10", "Zmeu", "/static/avatars/soul-10.png"},
		{"soul-11", "Iele", "/static/avatars/soul-11.png"},
		{"soul-12", "Muma", "/static/avatars/soul-12.png"},
		{"soul-13", "Căpcăun", "/static/avatars/soul-13.png"},
		{"soul-14", "Solomonar", "/static/avatars/soul-14.png"},
		{"soul-15", "Vâlvă", "/static/avatars/soul-15.png"},
		{"soul-16", "Pricolici", "/static/avatars/soul-16.png"},
	}
}

// SoulAvatarURL resolves the owner's soul avatar preference to a static URL.
func SoulAvatarURL(app core.App) string {
	key := GetOwnerSetting(app, "soul_avatar", "soul-01")
	if url, ok := soulAvatarMap[key]; ok {
		return url
	}
	return "/static/avatars/soul-01.png"
}

// ── Balaur head avatar ─────────────────────────────────────────────────

// balaurAvatarMap is the key→URL lookup for Balaur heads, derived from the
// BalaurHeads roster (the single source of truth). No legacy aliases — the
// 16 balaur-NN keys are 1:1 with the roster.
var balaurAvatarMap = avatarMap(BalaurHeads())

// BalaurHeads returns the roster of 16 Balaur personalities, the single source of truth.
func BalaurHeads() []AvatarEntry {
	return []AvatarEntry{
		{"balaur-01", "Wise", "/static/avatars/balaur-01.png"},
		{"balaur-02", "Ancient", "/static/avatars/balaur-02.png"},
		{"balaur-03", "Guardian", "/static/avatars/balaur-03.png"},
		{"balaur-04", "Scholar", "/static/avatars/balaur-04.png"},
		{"balaur-05", "Wild", "/static/avatars/balaur-05.png"},
		{"balaur-06", "Storm", "/static/avatars/balaur-06.png"},
		{"balaur-07", "Night", "/static/avatars/balaur-07.png"},
		{"balaur-08", "Young", "/static/avatars/balaur-08.png"},
		{"balaur-09", "Ember", "/static/avatars/balaur-09.png"},
		{"balaur-10", "Frost", "/static/avatars/balaur-10.png"},
		{"balaur-11", "Healer", "/static/avatars/balaur-11.png"},
		{"balaur-12", "Trickster", "/static/avatars/balaur-12.png"},
		{"balaur-13", "Dreamer", "/static/avatars/balaur-13.png"},
		{"balaur-14", "Forest", "/static/avatars/balaur-14.png"},
		{"balaur-15", "Dawn", "/static/avatars/balaur-15.png"},
		{"balaur-16", "Sage", "/static/avatars/balaur-16.png"},
	}
}

// ValidBalaurAvatarKey reports whether key is a recognised Balaur head.
func ValidBalaurAvatarKey(key string) bool {
	_, ok := balaurAvatarMap[key]
	return ok
}

// BalaurAvatarURL resolves the owner's chosen Balaur head to a static URL.
func BalaurAvatarURL(app core.App) string {
	key := GetOwnerSetting(app, "balaur_avatar", "balaur-01")
	if url, ok := balaurAvatarMap[key]; ok {
		return url
	}
	return "/static/avatars/balaur-01.png"
}

// ── Owner identity ─────────────────────────────────────────────────────

// OwnerName returns the owner's display name for the chat "You" label.
// Falls back to "You" when not set.
func OwnerName(app core.App) string {
	return GetOwnerSetting(app, "display_name", "You")
}

// ── Balaur head avatar by key ──────────────────────────────────────

// BalaurAvatarURLForKey resolves a Balaur avatar key (balaur-01…balaur-16) to
// a static URL, falling back to the owner's default when the key is empty or
// unknown. Used to render a head's avatar (built-in or custom).
func BalaurAvatarURLForKey(app core.App, key string) string {
	if url, ok := balaurAvatarMap[key]; ok {
		return url
	}
	return BalaurAvatarURL(app)
}
