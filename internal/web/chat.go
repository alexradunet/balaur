package web

import (
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/tools"
)

// chat handles one user turn. v1 keeps it deliberately simple: the turn is
// answered over a streamed chunked response that HTMX appends to the chat
// (hx-swap beforeend); conversation persistence wires in with the
// conversations UI. The fragment shape mirrors templates/home.html.
func (h *handlers) chat(e *core.RequestEvent) error {
	msg := strings.TrimSpace(e.Request.FormValue("message"))
	if msg == "" {
		return e.BadRequestError("empty message", nil)
	}

	client, err := h.llmClient()
	if err != nil {
		return h.renderError(e, err)
	}
	if local, ok := client.(*llm.KronkClient); ok && !local.ChatLoaded() {
		return h.renderError(e, fmt.Errorf("model is not loaded yet"))
	}

	w := e.Response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, _ := w.(http.Flusher)
	flush := func() {
		if flusher != nil {
			flusher.Flush()
		}
	}

	// Echo the user's message, then open the assistant fragment.
	fmt.Fprintf(w, `<div class="msg msg-user"><div class="who">You</div><div class="body">%s</div></div>`, html.EscapeString(msg))
	fmt.Fprint(w, `<div class="msg msg-balaur"><div class="who">Balaur</div><div class="body">`)
	flush()

	loop := &agent.Loop{Client: client, Tools: h.agentTools()}
	history := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: msg},
	}

	_, runErr := loop.Run(e.Request.Context(), history, func(ev agent.Event) {
		switch ev.Kind {
		case "text":
			fmt.Fprint(w, html.EscapeString(ev.Text))
			flush()
		case "tool_start":
			fmt.Fprintf(w, `</div></div><div class="msg msg-tool"><div class="who">tool · %s</div><div class="body">`, html.EscapeString(ev.Tool))
			flush()
		case "tool_result":
			fmt.Fprintf(w, `%s</div></div><div class="msg msg-balaur"><div class="who">Balaur</div><div class="body">`, html.EscapeString(clipText(ev.Text, 2000)))
			flush()
		case "error":
			fmt.Fprintf(w, `<span class="thinking">the thread snapped: %s</span>`, html.EscapeString(ev.Err.Error()))
			flush()
		}
	})

	fmt.Fprint(w, `</div></div>`)
	flush()
	_ = runErr // already surfaced in-stream; the fragment stays well-formed
	return nil
}

const systemPrompt = "You are Balaur, a wise personal companion. " +
	"Speak plainly and warmly, without flattery or hype. " +
	"Use tools when they genuinely help; otherwise just answer."

func clipText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// llmClient builds the configured client. Provider choice is explicit:
// BALAUR_REMOTE_URL switches to an OpenAI-compatible endpoint; otherwise
// local GGUF paths in BALAUR_CHAT_MODEL/BALAUR_EMBED_MODEL run via kronk.
func (h *handlers) llmClient() (llm.Client, error) {
	if h.models != nil {
		path, err := h.models.Store.ActiveChatModelPath()
		if err != nil {
			return nil, fmt.Errorf("loading active model: %w", err)
		}
		if path != "" {
			return h.localKronkClient(path), nil
		}
	}
	if base := os.Getenv("BALAUR_REMOTE_URL"); base != "" {
		return &llm.OpenAIClient{
			BaseURL: base,
			APIKey:  os.Getenv("BALAUR_REMOTE_API_KEY"),
			Model:   os.Getenv("BALAUR_REMOTE_MODEL"),
		}, nil
	}
	if chat := os.Getenv("BALAUR_CHAT_MODEL"); chat != "" {
		return h.localKronkClient(chat), nil
	}
	return nil, fmt.Errorf("no model configured: set BALAUR_CHAT_MODEL (local GGUF path) or BALAUR_REMOTE_URL")
}

func (h *handlers) localKronkClient(chatPath string) *llm.KronkClient {
	h.localMu.Lock()
	defer h.localMu.Unlock()
	return h.localKronkClientLocked(chatPath)
}

func (h *handlers) localKronkClientLocked(chatPath string) *llm.KronkClient {
	if h.localClient != nil && len(h.localClient.ChatModelFiles) == 1 && h.localClient.ChatModelFiles[0] == chatPath {
		return h.localClient
	}
	h.localErr = ""
	h.localLoad = false
	h.localClient = &llm.KronkClient{
		ChatModelFiles:  []string{chatPath},
		EmbedModelFiles: nonEmpty(os.Getenv("BALAUR_EMBED_MODEL")),
	}
	return h.localClient
}

func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

// agentTools returns the enabled tool set. OS access is opt-in (AGENTS.md).
func (h *handlers) agentTools() []agent.Tool {
	if os.Getenv("BALAUR_OS_ACCESS") == "1" {
		return tools.OSAccess(h.app)
	}
	return nil
}

func (h *handlers) renderError(e *core.RequestEvent, err error) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(e.Response,
		`<div class="msg msg-balaur"><div class="who">Balaur</div><div class="body"><span class="thinking">%s</span></div></div>`,
		html.EscapeString(err.Error()))
	return nil
}
