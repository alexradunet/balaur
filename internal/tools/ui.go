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
	"github.com/alexradunet/balaur/internal/store"
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

// UITools returns the card_show, board_compose, and board_add_card tools.
func UITools(app core.App) []agent.Tool {
	return []agent.Tool{cardShowTool(app), boardComposeTool(app), boardAddCardTool(app)}
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

func boardComposeTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("board_compose",
			"Create a new board of cards for the owner (e.g. 'set up a board for the trip'). "+
				"The board appears at /boards. Each card is a {type, params} object — type must be "+
				"from the card registry.",
			obj(map[string]any{
				"name": str("Board name (max 80 chars), e.g. 'Trip planning'."),
				"cards": map[string]any{
					"type":        "array",
					"description": "1–8 cards, each with a type from the registry and optional params.",
					"minItems":    1,
					"maxItems":    8,
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type":   map[string]any{"type": "string", "description": "Card type from the registry."},
							"params": map[string]any{"type": "object", "description": "Optional params (string values).", "additionalProperties": map[string]any{"type": "string"}},
						},
						"required": []string{"type"},
					},
				},
			}, "name", "cards")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Name  string       `json:"name"`
				Cards []cards.Card `json:"cards"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return fmt.Sprintf("board_compose: bad arguments: %s", err), nil
			}

			// Validate name length.
			name := strings.TrimSpace(args.Name)
			if name == "" {
				return "board_compose: name is required", nil
			}
			if len(name) > 80 {
				return "board_compose: name must be 80 characters or fewer", nil
			}

			// Validate card count.
			if len(args.Cards) == 0 {
				return "board_compose: at least 1 card is required", nil
			}
			if len(args.Cards) > 8 {
				return fmt.Sprintf("board_compose: at most 8 cards allowed, got %d", len(args.Cards)), nil
			}

			// Validate and clean cards.
			cleaned, err := cards.ValidateCards(args.Cards)
			if err != nil {
				return fmt.Sprintf("board_compose: %s", err), nil
			}

			// Find next sort value.
			existing, _ := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)
			maxSort := -1
			for _, r := range existing {
				s := int(r.GetFloat("sort"))
				if s > maxSort {
					maxSort = s
				}
			}
			nextSort := maxSort + 1

			col, err := app.FindCollectionByNameOrId("boards")
			if err != nil {
				return fmt.Sprintf("board_compose: boards collection not found: %s", err), nil
			}

			raw, err := json.Marshal(cleaned)
			if err != nil {
				return fmt.Sprintf("board_compose: serialising cards: %s", err), nil
			}

			rec := core.NewRecord(col)
			rec.Set("name", name)
			rec.Set("sort", nextSort)
			rec.Set("cards", string(raw))
			if err := app.Save(rec); err != nil {
				return fmt.Sprintf("board_compose: saving board: %s", err), nil
			}

			// Audit the creation.
			store.Audit(app, "", "agent", "board_compose", rec.Id, true,
				map[string]any{"name": name, "cards": len(cleaned)})

			return fmt.Sprintf("board raised: %s (%d cards) — /boards/%s", name, len(cleaned), rec.Id), nil
		},
	}
}

func boardAddCardTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("board_add_card",
			"Add one typed card to an existing board (e.g. 'add my weight to the trip board'). "+
				"Use board_compose to create a new board instead.",
			obj(map[string]any{
				"board": str("Board name or id. Resolved by exact id, then case-insensitive exact name, then case-insensitive substring match."),
				"type":  str("Card type from the registry (e.g. today, quests, measure)."),
				"params": map[string]any{
					"type":                 "object",
					"description":          "Optional parameters for the card (string values).",
					"additionalProperties": map[string]any{"type": "string"},
				},
			}, "board", "type")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Board  string            `json:"board"`
				Type   string            `json:"type"`
				Params map[string]string `json:"params"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return fmt.Sprintf("board_add_card: bad arguments: %s", err), nil
			}
			if args.Params == nil {
				args.Params = map[string]string{}
			}

			// Validate the card type and params first.
			cleaned, err := cards.Validate(args.Type, args.Params)
			if err != nil {
				return fmt.Sprintf("board_add_card: invalid card: %s", err), nil
			}

			// Resolve the board: load all boards and match by id, then name, then substring.
			all, _ := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)

			// Helper: build listing for error messages.
			boardNames := func() string {
				names := make([]string, 0, len(all))
				for _, r := range all {
					names = append(names, r.GetString("name"))
				}
				return strings.Join(names, ", ")
			}

			// Try exact id match first, then case-insensitive exact name, then substring.
			var matched []*core.Record
			lq := strings.ToLower(args.Board)
			for _, r := range all {
				if r.Id == args.Board {
					matched = []*core.Record{r}
					break
				}
			}

			// Exact name match (case-insensitive).
			if len(matched) == 0 {
				for _, r := range all {
					if strings.ToLower(r.GetString("name")) == lq {
						matched = append(matched, r)
					}
				}
			}

			// Substring match (case-insensitive), only if no exact name match.
			if len(matched) == 0 && lq != "" {
				for _, r := range all {
					if strings.Contains(strings.ToLower(r.GetString("name")), lq) {
						matched = append(matched, r)
					}
				}
			}

			if len(matched) == 0 {
				return fmt.Sprintf("board_add_card: no board matches %q — boards: %s", args.Board, boardNames()), nil
			}
			if len(matched) > 1 {
				names := make([]string, 0, len(matched))
				for _, r := range matched {
					names = append(names, r.GetString("name"))
				}
				return fmt.Sprintf("board_add_card: %q matches multiple boards: %s — be more specific", args.Board, strings.Join(names, ", ")), nil
			}
			board := matched[0]

			// Decode existing cards, append the new one, re-encode.
			raw := board.GetString("cards")
			var existing []cards.Card
			if raw != "" {
				if err := json.Unmarshal([]byte(raw), &existing); err != nil {
					return fmt.Sprintf("board_add_card: decoding board cards: %s", err), nil
				}
			}

			newCard := cards.Card{Type: args.Type}
			if len(cleaned) > 0 {
				newCard.Params = cleaned
			}
			existing = append(existing, newCard)

			encoded, err := json.Marshal(existing)
			if err != nil {
				return fmt.Sprintf("board_add_card: encoding cards: %s", err), nil
			}

			board.Set("cards", string(encoded))
			if err := app.Save(board); err != nil {
				return fmt.Sprintf("board_add_card: saving board: %s", err), nil
			}

			// Audit the mutation.
			spec, _ := cards.Get(args.Type)
			store.Audit(app, "", "agent", "board_add_card", board.Id, true,
				map[string]any{"card_type": args.Type, "board": board.GetString("name")})

			return fmt.Sprintf("added %s to %s — /boards/%s", spec.Label, board.GetString("name"), board.Id), nil
		},
	}
}
