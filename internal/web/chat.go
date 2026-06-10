package web

import (
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/tools"
)

// recentTurnWindow caps how many prior text turns enter the model context.
// Persistence is unbounded; context is not (the master-conversation
// footgun defusal — see internal/conversation).
const recentTurnWindow = 20

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
	clientRendered := e.Request.FormValue("client_rendered") == "1"

	w := e.Response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, _ := w.(http.Flusher)
	flush := func() {
		if flusher != nil {
			flusher.Flush()
		}
	}

	// When the browser optimistically rendered the user row, this response
	// replaces only the pending Balaur row. Without JS, keep the old echo path.
	if !clientRendered {
		fmt.Fprintf(w, `<div class="msg msg-user"><div class="who">You</div><div class="body">%s</div></div>`, html.EscapeString(msg))
	}
	fmt.Fprint(w, `<div class="msg msg-balaur"><div class="who">Balaur</div><div class="body">`)
	flush()

	// The master conversation: load the recent window BEFORE persisting the
	// new user turn, so the window holds prior turns only.
	master, err := conversation.Master(h.app)
	if err != nil {
		return h.renderError(e, err)
	}
	recent, err := conversation.RecentTurns(h.app, master.Id, recentTurnWindow)
	if err != nil {
		return h.renderError(e, err)
	}
	if err := conversation.Append(h.app, master.Id, llm.Message{Role: "user", Content: msg}, ""); err != nil {
		return h.renderError(e, err)
	}

	// Context = system prompt + knowledge block + recent turns + this turn.
	// Persistence is not context: the full record stays in SQLite.
	knowledgeBlock, usedMemories := knowledge.BuildContext(h.app, msg)
	loop := &agent.Loop{Client: client, Tools: h.agentTools()}
	history := make([]llm.Message, 0, len(recent)+2)
	history = append(history, llm.Message{Role: "system", Content: systemPrompt + knowledgeBlock})
	history = append(history, recent...)
	history = append(history, llm.Message{Role: "user", Content: msg})
	contextLen := len(history)

	final, runErr := loop.Run(e.Request.Context(), history, func(ev agent.Event) {
		switch ev.Kind {
		case "text":
			fmt.Fprint(w, html.EscapeString(ev.Text))
			flush()
		case "tool_start":
			fmt.Fprintf(w, `</div></div><div class="msg msg-tool"><div class="who">tool · %s</div><div class="body">`, html.EscapeString(ev.Tool))
			flush()
		case "tool_result":
			h.writeToolResult(w, ev.Text)
			fmt.Fprint(w, `</div></div><div class="msg msg-balaur"><div class="who">Balaur</div><div class="body">`)
			flush()
		case "error":
			fmt.Fprintf(w, `<span class="thinking">the thread snapped: %s</span>`, html.EscapeString(ev.Err.Error()))
			flush()
		}
	})

	// Persist every turn the loop appended (assistant and tool rounds).
	// Tool turns carry the call id; map it back to the tool's name from the
	// preceding assistant turn so the record reads human.
	toolNames := map[string]string{}
	for _, m := range final[contextLen:] {
		name := ""
		if m.Role == "tool" {
			name = toolNames[m.ToolCallID]
		}
		for _, tc := range m.ToolCalls {
			toolNames[tc.ID] = tc.Name
		}
		if err := conversation.Append(h.app, master.Id, m, name); err != nil {
			break // persistence failure must not break the stream mid-reply
		}
	}

	// Memories that informed this turn count as used.
	for _, m := range usedMemories {
		knowledge.Touch(h.app, knowledge.Memory, m)
	}

	fmt.Fprint(w, `</div></div>`)
	flush()
	_ = runErr // already surfaced in-stream; the fragment stays well-formed
	return nil
}

const systemPrompt = "You are Balaur, a wise personal companion. " +
	"Speak plainly and warmly, without flattery or hype. " +
	"Use tools when they genuinely help; otherwise just answer.\n\n" +
	"Memory discipline: when the owner shares something durable — a fact about " +
	"their life, a standing preference, a person, a project, a constraint — " +
	"propose remembering it with the `remember` tool. Propose sparingly: one " +
	"clear memory beats five vague ones. Never re-propose something already in " +
	"your memory context or something the owner declined. When you notice a " +
	"repeatable procedure worth keeping, propose it with `propose_skill`. " +
	"Proposals require the owner's approval; never claim something is " +
	"remembered until it is."

// writeToolResult renders a tool result row. Proposal-marked results render
// as approval cards instead of raw text (the Hyperagent card pattern).
func (h *handlers) writeToolResult(w http.ResponseWriter, text string) {
	kind, id, rest, ok := tools.ParseProposal(text)
	if !ok {
		fmt.Fprint(w, html.EscapeString(clipText(text, 2000)))
		return
	}
	fmt.Fprint(w, html.EscapeString(rest))
	// Close the tool row and inject the live card fetched by HTMX, so the
	// card in chat is the same template the /memory and /skills pages use.
	fmt.Fprintf(w,
		`</div></div><div class="k-inline" hx-get="/ui/knowledge/%s/%s/card" hx-trigger="load" hx-swap="innerHTML"></div><div class="msg msg-tool" hidden><div class="body">`,
		html.EscapeString(kind), html.EscapeString(id))
}

func clipText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// llmClient builds the configured client. Active downloaded models selected
// in the UI take precedence; otherwise env configuration is explicit.
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
	return llm.FromEnv()
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

// agentTools returns the enabled tool set: knowledge tools always (they only
// propose — the consent boundary holds), OS access opt-in (AGENTS.md).
func (h *handlers) agentTools() []agent.Tool {
	ts := tools.KnowledgeTools(h.app)
	if os.Getenv("BALAUR_OS_ACCESS") == "1" {
		ts = append(ts, tools.OSAccess(h.app)...)
	}
	return ts
}

func (h *handlers) renderError(e *core.RequestEvent, err error) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(e.Response,
		`<div class="msg msg-balaur"><div class="who">Balaur</div><div class="body"><span class="thinking">%s</span></div></div>`,
		html.EscapeString(err.Error()))
	return nil
}
