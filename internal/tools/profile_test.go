package tools

import (
	"context"
	"testing"

	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestProfileSetTool(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := profileSetTool(app).Execute(context.Background(), `{"display_name":"Alexandru"}`); err != nil {
		t.Fatalf("profile_set: %v", err)
	}
	if got := store.GetOwnerSetting(app, "display_name", ""); got != "Alexandru" {
		t.Errorf("display_name = %q, want Alexandru", got)
	}
	if countModelAudit(t, app, "profile.edit") != 1 {
		t.Error("profile.edit should audit actor=model")
	}
	// An invalid avatar key is rejected (the write never happens).
	if _, err := profileSetTool(app).Execute(context.Background(), `{"soul_avatar":"not-a-key"}`); err == nil {
		t.Error("an invalid soul avatar key should fail")
	}
	// An empty payload is rejected — at least one field is required.
	if _, err := profileSetTool(app).Execute(context.Background(), `{}`); err == nil {
		t.Error("an empty profile_set should fail")
	}
}
