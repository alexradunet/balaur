package store

import "github.com/pocketbase/pocketbase/core"

// AvatarEntry is one selectable avatar: key (stored in owner_settings /
// head records), human label, and served URL. The exported rosters are the
// single source of truth — web option builders iterate these.
type AvatarEntry struct {
	Key   string
	Label string
	URL   string
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

// SetOwnerSetting upserts a key/value pair in owner_settings.
func SetOwnerSetting(app core.App, key, value string) error {
	col, err := app.FindCollectionByNameOrId("owner_settings")
	if err != nil {
		return err
	}
	rec, err := app.FindFirstRecordByData("owner_settings", "key", key)
	if err != nil {
		rec = core.NewRecord(col)
		rec.Set("key", key)
	}
	rec.Set("value", value)
	return app.Save(rec)
}

// ── Soul avatar ────────────────────────────────────────────────────────

// soulAvatarMap maps avatar keys to their static file paths.
// Legacy values "male" and "female" are kept as aliases.
var soulAvatarMap = map[string]string{
	"soul-01": "/static/avatars/soul-01.png", // Him
	"soul-02": "/static/avatars/soul-02.png", // Her
	"soul-03": "/static/avatars/soul-03.png", // Elder
	"soul-04": "/static/avatars/soul-04.png", // Youth
	"soul-05": "/static/avatars/soul-05.png", // Maker
	"soul-06": "/static/avatars/soul-06.png", // Cyclops
	"soul-07": "/static/avatars/soul-07.png", // Gnome
	"soul-08": "/static/avatars/soul-08.png", // Ogre
	"soul-09": "/static/avatars/soul-09.png", // Strigoi
	"soul-10": "/static/avatars/soul-10.png", // Zmeu
	"soul-11": "/static/avatars/soul-11.png", // Iele
	"soul-12": "/static/avatars/soul-12.png", // Muma Pădurii
	"soul-13": "/static/avatars/soul-13.png", // Căpcăun
	"soul-14": "/static/avatars/soul-14.png", // Solomonar
	"soul-15": "/static/avatars/soul-15.png", // Vâlvă
	"soul-16": "/static/avatars/soul-16.png", // Pricolici
	"male":    "/static/avatars/soul-01.png", // legacy alias
	"female":  "/static/avatars/soul-02.png", // legacy alias
}

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

var balaurAvatarMap = map[string]string{
	"balaur-01": "/static/avatars/balaur-01.png", // Wise (default)
	"balaur-02": "/static/avatars/balaur-02.png", // Ancient
	"balaur-03": "/static/avatars/balaur-03.png", // Guardian
	"balaur-04": "/static/avatars/balaur-04.png", // Scholar
	"balaur-05": "/static/avatars/balaur-05.png", // Wild
	"balaur-06": "/static/avatars/balaur-06.png", // Storm
	"balaur-07": "/static/avatars/balaur-07.png", // Night
	"balaur-08": "/static/avatars/balaur-08.png", // Young
	"balaur-09": "/static/avatars/balaur-09.png", // Ember
	"balaur-10": "/static/avatars/balaur-10.png", // Frost
	"balaur-11": "/static/avatars/balaur-11.png", // Healer
	"balaur-12": "/static/avatars/balaur-12.png", // Trickster
	"balaur-13": "/static/avatars/balaur-13.png", // Dreamer
	"balaur-14": "/static/avatars/balaur-14.png", // Forest
	"balaur-15": "/static/avatars/balaur-15.png", // Dawn
	"balaur-16": "/static/avatars/balaur-16.png", // Sage
}

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

// ── Per-head Balaur avatar ─────────────────────────────────────────

// HeadBalaurAvatarURL resolves the Balaur avatar URL for a specific head
// record. If the head has its own balaur_avatar set, that takes precedence.
// Otherwise falls back to the owner's balaur_avatar preference, then to
// balaur-01 (Wise). Pass an empty headID to get the owner's default.
func HeadBalaurAvatarURL(app core.App, headID string) string {
	if headID != "" {
		head, err := app.FindRecordById("heads", headID)
		if err == nil {
			key := head.GetString("balaur_avatar")
			if url, ok := balaurAvatarMap[key]; ok {
				return url
			}
		}
	}
	return BalaurAvatarURL(app) // owner's default or balaur-01
}

// SetHeadBalaurAvatar persists a Balaur head personality on a head record.
func SetHeadBalaurAvatar(app core.App, headID, key string) error {
	head, err := app.FindRecordById("heads", headID)
	if err != nil {
		return err
	}
	head.Set("balaur_avatar", key)
	return app.Save(head)
}
