package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/cards"
)

// UICardMarker prefixes tool results that carry an inline card URL, so the web
// layer can render a live card embed instead of a plain tool row. Format:
// marker + type + "?" + url.Values-encoded params + "\n" + model-facing text.
//
// NOTE: The three marker kinds (ProposalMarker, ChoicesMarker, UICardMarker) each
// have different head formats: proposal uses "kind/id", choices uses a JSON object,
// uicard uses "type?query". Collapsing them into a shared helper would require
// abstraction over those differing formats without simplifying any call site, so
// they are kept separate — each pair is small and self-describing.
const UICardMarker = "\x00balaur-uicard:"

// MarkUICard builds a marked uicard result.
// Format: marker + type + "?" + url.Values-encoded params + "\n" + modelText.
func MarkUICard(typ string, params map[string]string, modelText string) string {
	vals := url.Values{}
	for k, v := range params {
		vals.Set(k, v)
	}
	query := vals.Encode()
	head := typ + "?" + query
	return UICardMarker + head + "\n" + modelText
}

// ParseUICard splits a marked uicard result. Returns typ, query (url-encoded
// params), rest (model-facing text), and ok. ok is false for ordinary text.
// Invariant: when ok is true, typ is always a registered card type, so
// consumers may embed it directly in a URL path without further validation.
func ParseUICard(s string) (typ, query, rest string, ok bool) {
	if !strings.HasPrefix(s, UICardMarker) {
		return "", "", s, false
	}
	s = strings.TrimPrefix(s, UICardMarker)
	head, rest, _ := strings.Cut(s, "\n")
	typ, query, _ = strings.Cut(head, "?")
	if typ == "" {
		return "", "", rest, false
	}
	if _, found := cards.Get(typ); !found {
		return "", "", rest, false
	}
	return typ, query, rest, true
}

// UITools returns the card_show and show_cards tools. (089 retired the board
// tools; 090 added show_cards for in-chat card clusters.)
func UITools(app core.App) []agent.Tool {
	return []agent.Tool{cardShowTool(app), showCardsTool(app)}
}

func cardShowTool(_ core.App) agent.Tool {
	// Build a rich description that embeds the real registry vocabulary,
	// so the model sees the actual types and their param docs.
	var b strings.Builder
	fmt.Fprint(&b, "Render a live UI card into the conversation. Choose a type from the registry; "+
		"the server renders it from the owner's real data. Available types:\n")
	for _, spec := range cards.All() {
		fmt.Fprintf(&b, "  %s (%s)", spec.Type, spec.Label)
		if len(spec.Params) > 0 {
			fmt.Fprint(&b, " — params: ")
			ps := make([]string, 0, len(spec.Params))
			for _, p := range spec.Params {
				entry := p.Name
				if p.Required {
					entry += " (required)"
				}
				if p.Doc != "" {
					entry += ": " + p.Doc
				}
				ps = append(ps, entry)
			}
			fmt.Fprint(&b, strings.Join(ps, "; "))
		}
		fmt.Fprint(&b, "\n")
	}
	desc := b.String()

	return agent.Tool{
		Spec: agent.ToolSpecOf("card_show",
			desc,
			obj(map[string]any{
				"type": str("Card type from the registry (e.g. today, quests, measure)."),
				"params": map[string]any{
					"type":                 "object",
					"description":          "Optional parameters for the card (string values).",
					"additionalProperties": map[string]any{"type": "string"},
				},
			}, "type")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Type   string            `json:"type"`
				Params map[string]string `json:"params"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return fmt.Sprintf("card_show: bad arguments: %s", err), nil
			}
			if args.Params == nil {
				args.Params = map[string]string{}
			}
			cleaned, err := cards.Validate(args.Type, args.Params)
			if err != nil {
				// Return plain-text error — the model self-corrects.
				return fmt.Sprintf("card_show: invalid card: %s", err), nil
			}
			spec, _ := cards.Get(args.Type)
			modelText := fmt.Sprintf("showing the owner the %s card", spec.Label)
			return MarkUICard(args.Type, cleaned, modelText), nil
		},
	}
}
