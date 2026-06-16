package web

import (
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llmtest"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
)

// seedScriptedModel injects the shared scripted fake llm.Client (internal/llmtest)
// and registers it as the active local model, so the model resolves and reports
// ready without a daemon. Pass one llmtest.Reply per expected ChatStream call.
// Returns the model record id.
func seedScriptedModel(tb testing.TB, app core.App, replies ...llmtest.Reply) string {
	tb.Helper()
	turn.SetTestClient(app, llmtest.New(replies...))
	return activateLocalModel(tb, app)
}

// seedFailingModel injects a scripted client whose ChatStream/Embed always error,
// for exercising the handlers' fallback paths.
func seedFailingModel(tb testing.TB, app core.App) string {
	tb.Helper()
	turn.SetTestClient(app, &llmtest.ScriptedClient{Err: errors.New("fake model unavailable")})
	return activateLocalModel(tb, app)
}

func activateLocalModel(tb testing.TB, app core.App) string {
	tb.Helper()
	id, err := store.SaveLocalModel(app, "fake-model", "")
	if err != nil {
		tb.Fatalf("SaveLocalModel: %v", err)
	}
	if err := store.SetActiveLLMModel(app, id, "test"); err != nil {
		tb.Fatalf("SetActiveLLMModel: %v", err)
	}
	return id
}
