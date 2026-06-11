package store

import "github.com/pocketbase/pocketbase/core"

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

// soulAvatarMap maps avatar keys to their static file paths.
// Legacy values "male" and "female" are kept as aliases so existing
// owner_settings rows continue to work after upgrading.
var soulAvatarMap = map[string]string{
	// Basm world — human characters
	"soul-01": "/static/avatars/soul-01.png", // Him
	"soul-02": "/static/avatars/soul-02.png", // Her
	"soul-03": "/static/avatars/soul-03.png", // Elder
	"soul-04": "/static/avatars/soul-04.png", // Youth
	"soul-05": "/static/avatars/soul-05.png", // Maker
	"soul-06": "/static/avatars/soul-06.png", // Cyclops
	"soul-07": "/static/avatars/soul-07.png", // Gnome
	"soul-08": "/static/avatars/soul-08.png", // Ogre
	// Romanian mythological creatures
	"soul-09": "/static/avatars/soul-09.png", // Strigoi
	"soul-10": "/static/avatars/soul-10.png", // Zmeu
	"soul-11": "/static/avatars/soul-11.png", // Iele
	"soul-12": "/static/avatars/soul-12.png", // Muma Pădurii
	"soul-13": "/static/avatars/soul-13.png", // Căpcăun
	"soul-14": "/static/avatars/soul-14.png", // Solomonar
	"soul-15": "/static/avatars/soul-15.png", // Vâlvă
	"soul-16": "/static/avatars/soul-16.png", // Pricolici
	// legacy aliases from the two-option picker
	"male":   "/static/avatars/soul-01.png",
	"female": "/static/avatars/soul-02.png",
}

// ValidSoulAvatarKey reports whether key is a recognised avatar choice.
func ValidSoulAvatarKey(key string) bool {
	_, ok := soulAvatarMap[key]
	return ok
}

// SoulAvatarURL resolves the current soul avatar preference to a static URL.
// Falls back to soul-01 (Him) when the stored key is unknown or absent.
func SoulAvatarURL(app core.App) string {
	key := GetOwnerSetting(app, "soul_avatar", "soul-01")
	if url, ok := soulAvatarMap[key]; ok {
		return url
	}
	return "/static/avatars/soul-01.png"
}
