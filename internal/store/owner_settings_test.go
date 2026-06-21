package store

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestAvatarRosters(t *testing.T) {
	souls := SoulAvatars()
	if len(souls) != 16 {
		t.Fatalf("SoulAvatars returned %d entries, want 16", len(souls))
	}
	for i, e := range souls {
		if e.Key == "" || e.Label == "" || e.URL == "" {
			t.Fatalf("SoulAvatars[%d] missing key/label/url: %#v", i, e)
		}
		if !strings.HasPrefix(e.URL, "/static/avatars/") {
			t.Fatalf("SoulAvatars[%d] URL not under /static/avatars/: %q", i, e.URL)
		}
	}

	heads := BalaurHeads()
	if len(heads) != 16 {
		t.Fatalf("BalaurHeads returned %d entries, want 16", len(heads))
	}
	for i, e := range heads {
		if e.Key == "" || e.Label == "" || e.URL == "" {
			t.Fatalf("BalaurHeads[%d] missing key/label/url: %#v", i, e)
		}
		if !strings.HasPrefix(e.URL, "/static/avatars/") {
			t.Fatalf("BalaurHeads[%d] URL not under /static/avatars/: %q", i, e.URL)
		}
	}
}

func TestValidAvatarKeysMatchRosters(t *testing.T) {
	souls := SoulAvatars()
	for _, e := range souls {
		if !ValidSoulAvatarKey(e.Key) {
			t.Fatalf("ValidSoulAvatarKey(%q) false, but in SoulAvatars roster", e.Key)
		}
	}
	if ValidSoulAvatarKey("nope") {
		t.Fatalf("ValidSoulAvatarKey(\"nope\") true, want false")
	}

	heads := BalaurHeads()
	for _, e := range heads {
		if !ValidBalaurAvatarKey(e.Key) {
			t.Fatalf("ValidBalaurAvatarKey(%q) false, but in BalaurHeads roster", e.Key)
		}
	}
	if ValidBalaurAvatarKey("nope") {
		t.Fatalf("ValidBalaurAvatarKey(\"nope\") true, want false")
	}
}

func TestLegacySoulAvatarAliases(t *testing.T) {
	app := storetest.NewApp(t)
	// Legacy aliases should still resolve
	if SoulAvatarURL(app) != "/static/avatars/soul-01.png" {
		t.Fatal("default soul avatar wrong")
	}
	if !ValidSoulAvatarKey("male") {
		t.Fatal("legacy male alias not valid")
	}
	if !ValidSoulAvatarKey("female") {
		t.Fatal("legacy female alias not valid")
	}
	if url := soulAvatarMap["male"]; url != "/static/avatars/soul-01.png" {
		t.Fatalf("male alias resolves to %q, want soul-01", url)
	}
	if url := soulAvatarMap["female"]; url != "/static/avatars/soul-02.png" {
		t.Fatalf("female alias resolves to %q, want soul-02", url)
	}
}

func TestSetOwnerSettingConcurrent(t *testing.T) {
	app := storetest.NewApp(t)
	const n = 24
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- SetOwnerSetting(app, "panel_active", fmt.Sprintf("/ui/show/quests?n=%d", i))
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent SetOwnerSetting: %v", err)
		}
	}
	// Exactly one row for the key — the UNIQUE index plus our converge logic.
	recs, err := app.FindRecordsByFilter("owner_settings", "key = 'panel_active'", "", 0, 0)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("want exactly one owner_settings row for the key, got %d", len(recs))
	}
}
