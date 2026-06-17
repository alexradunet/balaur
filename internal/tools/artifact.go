package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/cards"
)

// ArtifactMarker prefixes a tool result that carries a hand-picked cluster of
// cards — ONE conversation artifact holding N live cards. Mirrors UICardMarker:
// a NUL-prefixed marker (inert to the model) + a JSON head + "\n" + model text.
//
// Format: ArtifactMarker + JSON(artifactHead) + "\n" + modelText.
const ArtifactMarker = "\x00balaur-artifact:"

// artifactHead is the self-describing JSON head line embedded in a marked result.
type artifactHead struct {
	Title string       `json:"title,omitempty"`
	Cards []cards.Card `json:"cards"`
}

// MarkArtifact builds a marked cluster result.
// Format: marker + JSON(artifactHead) + "\n" + modelText.
func MarkArtifact(cs []cards.Card, title, modelText string) string {
	b, _ := json.Marshal(artifactHead{Title: title, Cards: cs})
	return ArtifactMarker + string(b) + "\n" + modelText
}

// ParseArtifact splits a marked cluster result. ok is false for ordinary text.
// Invariant: when ok is true, every returned card is a registered, validated
// card type (ValidateCards), so the gateway may render each by type-string
// without further validation. ValidateCards runs here (not only in the tool)
// so a stale/hand-edited persisted marker with an unknown card type degrades
// to plain text instead of an error or panic.
func ParseArtifact(s string) (title string, cs []cards.Card, rest string, ok bool) {
	if !strings.HasPrefix(s, ArtifactMarker) {
		return "", nil, s, false
	}
	s = strings.TrimPrefix(s, ArtifactMarker)
	headLine, rest, _ := strings.Cut(s, "\n")
	var head artifactHead
	if err := json.Unmarshal([]byte(headLine), &head); err != nil {
		return "", nil, rest, false
	}
	cleaned, err := cards.ValidateCards(head.Cards)
	if err != nil || len(cleaned) == 0 {
		return "", nil, rest, false
	}
	return head.Title, cleaned, rest, true
}

// showCardsMax bounds a cluster (mirrors board_compose's old 1–8 cap).
const showCardsMax = 8

func showCardsTool(_ core.App) agent.Tool {
	// Build a rich description that embeds the real registry vocabulary,
	// so the model sees the actual types and their param docs.
	var b strings.Builder
	fmt.Fprint(&b, "Render a cluster of live UI cards into the conversation as ONE artifact "+
		"(e.g. 'show my quests and my weight together'). Pick 1–8 cards; each is a "+
		"{type, params} from the registry; the server renders each from the owner's real "+
		"data. To draw the owner's individual quests as separate cards, use the \"tasks\" "+
		"card (a bare stack of task cards) with a status/bucket/terms filter. Available types:\n")
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
		Spec: agent.ToolSpecOf("show_cards", desc,
			obj(map[string]any{
				"title": str("Optional heading shown above the cluster, e.g. 'Your week'."),
				"cards": map[string]any{
					"type":        "array",
					"description": fmt.Sprintf("1–%d cards, each a {type, params} from the registry.", showCardsMax),
					"minItems":    1,
					"maxItems":    showCardsMax,
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type":   map[string]any{"type": "string", "description": "Card type from the registry."},
							"params": map[string]any{"type": "object", "description": "Optional params (string values).", "additionalProperties": map[string]any{"type": "string"}},
						},
						"required": []string{"type"},
					},
				},
			}, "cards")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Title string       `json:"title"`
				Cards []cards.Card `json:"cards"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return fmt.Sprintf("show_cards: bad arguments: %s", err), nil
			}
			if len(args.Cards) == 0 {
				return "show_cards: at least 1 card is required", nil
			}
			if len(args.Cards) > showCardsMax {
				return fmt.Sprintf("show_cards: at most %d cards allowed, got %d", showCardsMax, len(args.Cards)), nil
			}
			cleaned, err := cards.ValidateCards(args.Cards)
			if err != nil {
				return fmt.Sprintf("show_cards: %s", err), nil
			}
			title := strings.TrimSpace(args.Title)
			modelText := fmt.Sprintf("showing the owner a cluster of %d cards", len(cleaned))
			return MarkArtifact(cleaned, title, modelText), nil
		},
	}
}
