package web

import (
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/knowledge"
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

	// Knowledge injection: tier-1 upfront + tier-2 recall + skills index.
	knowledgeBlock, usedMemories := knowledge.BuildContext(h.app, msg)
	loop := &agent.Loop{Client: client, Tools: h.agentTools()}
	history := []llm.Message{
		{Role: "system", Content: systemPrompt + knowledgeBlock},
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
			h.writeToolResult(w, ev.Text)
			fmt.Fprint(w, `</div></div><div class="msg msg-balaur"><div class="who">Balaur</div><div class="body">`)
			flush()
		case "error":
			fmt.Fprintf(w, `<span class="thinking">the thread snapped: %s</span>`, html.EscapeString(ev.Err.Error()))
			flush()
		}
	})

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

// llmClient builds the configured client. Provider choice is explicit:
// BALAUR_REMOTE_URL switches to an OpenAI-compatible endpoint; otherwise
// local GGUF paths in BALAUR_CHAT_MODEL/BALAUR_EMBED_MODEL run via kronk.
func (h *handlers) llmClient() (llm.Client, error) {
	if base := os.Getenv("BALAUR_REMOTE_URL"); base != "" {
		return &llm.OpenAIClient{
			BaseURL: base,
			APIKey:  os.Getenv("BALAUR_REMOTE_API_KEY"),
			Model:   os.Getenv("BALAUR_REMOTE_MODEL"),
		}, nil
	}
	if chat := os.Getenv("BALAUR_CHAT_MODEL"); chat != "" {
		return &llm.KronkClient{
			ChatModelFiles:  []string{chat},
			EmbedModelFiles: nonEmpty(os.Getenv("BALAUR_EMBED_MODEL")),
		}, nil
	}
	return nil, fmt.Errorf("no model configured: set BALAUR_CHAT_MODEL (local GGUF path) or BALAUR_REMOTE_URL")
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
