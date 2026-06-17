# Plan 090: Ad-hoc card clusters — ArtifactMarker + show_cards tool + filtered bare "tasks" cluster card

> **Executor instructions**: Follow this plan step by step. Run every **Verify** and confirm the expected result before moving to the next step. On a STOP condition, stop and report — do not improvise. When done, update the 090 row in `plans/readme.md` (add it if absent, matching the existing column format) — unless a reviewer dispatched you and told you they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 3136bad..HEAD -- internal/tools/ui.go internal/cards/cards.go internal/ui/registry.go internal/web/chatstream.go internal/web/recap.go internal/web/cards.go internal/turn/tools.go internal/tasks/tasks.go internal/feature/taskcards internal/feature/storybook/stories_cards.go internal/feature/storybook/story.go`
> If any of these changed since `5857eee`, compare the live code to the "Current state" excerpts below; on mismatch NOT explained by the reconciliation note below, STOP and report.
>
> **RECONCILED FOR CURRENT MAIN (2026-06-17, anchor `5857eee`) — READ FIRST.** Plans 088 + 089 (and, in parallel, 086 + 087) have all LANDED on main. Confirmed deltas vs this plan's original 3136bad excerpts — reconcile against the live tree, do NOT STOP on these:
> - **`tools.UITools` is now `[]agent.Tool{cardShowTool(app)}`** (089 removed the board tools). So add `showCardsTool(app)` as the second element: `return []agent.Tool{cardShowTool(app), showCardsTool(app)}`. The "if 089 landed" note in Step 2 is now the ACTUAL case — there are no board tools to keep.
> - The core seams this plan extends are UNCHANGED since 3136bad and the excerpts still match: `chatstream.handleToolResult` (marker chain at `:177-190`), `recap.messageViews` (`mv.Role=="tool"` at `:262`), `internal/web/cards.go` `cardHTML`/`uicardBody`, `internal/cards/cards.go` (`quests` spec ~`:56`, `ValidateCards` ~`:242`, no `tasks` type yet), `internal/tasks/tasks.go` loaders, and `internal/feature/taskcards` (`taskViewOf` quests.go:88, `viewsOf`:110, `intParam`:157, `TaskCard` taskcard.go:25). Verify by grep, trust the excerpts.
> - `internal/feature/storybook/story.go`'s `stories` slice GREW (086/087 added settings/runtime stories; 088 added `sidebarStory`; 089 removed `topbarStory`). You only APPEND your new `Cluster` + `tasks` stories — don't rely on old line numbers; just add two registration lines + the builders.
> - This plan adds NO migration and does not touch the kronk/model code 086/087 changed — no overlap there.
>
> **Hard dependency on plan 088**: this plan reuses 088's artifact-append + persistence + reload-re-render pattern. 088 establishes the single-page chat shell and the deterministic `/ui/show` endpoint. Land 088 first. The marker/tool/cluster-card work here is **additive** to the existing card_show seam (`tools.MarkUICard`/`ParseUICard` + `chatStream.handleToolResult` + `recap.messageViews`), so most of it can be built against the pre-088 code; only the "should `/ui/show` accept a cluster?" decision (Step 7) touches 088's surface — and the recommendation is *no*, keeping 090 independent of 088's exact endpoint shape.

## Status
- **Priority**: P2
- **Effort**: M
- **Risk**: MED (a new in-band marker protocol; the live-stream and reload paths are SEPARATE and BOTH must handle it — the headline bug class)
- **Depends on**: plan **088** (artifact append + persistence + reload branch). Soft-adjacent to **090's siblings** 089/091 (independent surfaces).
- **Category**: feature (owner-requested)
- **Planned at**: commit `3136bad`, 2026-06-17

## Why this matters
088 lets a single domain artifact (one card) appear in the conversation — by sidebar click (deterministic) or by the agent's `card_show` tool (conversational). The owner also wants two things one card cannot express:

1. **A hand-picked cluster.** "Show me my quests and my weight" should land as ONE artifact holding N cards, not as N separate tool rows. The agent picks the cards; the server renders them into one container.
2. **The literal individual quests.** "Draw the cards for those quests" means rendering each matching task as its own `TaskCard` — a bare stack with no container chrome — not the summary `quests` card (which has a header + footer + a rolled-up list). Today there is no card type that renders *individual* tasks as standalone cards filtered by the agent's criteria.

This plan adds the smallest machinery for both: a new `ArtifactMarker`/`show_cards` tool that mirrors the existing `card_show`/`UICardMarker` seam exactly (so it inherits validation, persistence, and reload re-rendering for free), plus a new filtered `tasks` cluster card that draws bare `TaskCard`s. Both converge on the same `cards` registry, so the new `tasks` card is usable by `card_show`, `/ui/show` (088), AND `show_cards` clusters.

## The one bug you must not ship (read before coding)
**The live stream and the reload replay are TWO SEPARATE code paths that parse the SAME persisted marker.** A new marker added to only one of them shows the cluster live but makes it VANISH on refresh (or vice-versa). Concretely:
- LIVE: `internal/web/chatstream.go` → `handleToolResult` dispatches markers in order (uicard → choices → proposal → refresh → plain).
- RELOAD: `internal/web/recap.go` → `messageViews` re-parses the SAME markers out of persisted message content on page load.

**The new `ParseArtifact` branch MUST be added to BOTH `handleToolResult` AND `messageViews`, in the same commit.** A unit/integration check that loads a persisted artifact message and asserts the cluster re-renders is the regression net (Test plan).

## Current state (confirmed excerpts @ 3136bad)

**`internal/tools/ui.go:26-58`** — the exact marker pattern to mirror (define/MarkUICard/ParseUICard; note the registry-validation invariant on Parse):
```go
26	const UICardMarker = "\x00balaur-uicard:"
27
28	// MarkUICard builds a marked uicard result.
29	// Format: marker + type + "?" + url.Values-encoded params + "\n" + modelText.
30	func MarkUICard(typ string, params map[string]string, modelText string) string {
...
40	// ParseUICard splits a marked uicard result. Returns typ, query (url-encoded
41	// params), rest (model-facing text), and ok. ok is false for ordinary text.
42	// Invariant: when ok is true, typ is always a registered card type, so
43	// consumers may embed it directly in a URL path without further validation.
44	func ParseUICard(s string) (typ, query, rest string, ok bool) {
...
54		if _, found := cards.Get(typ); !found {
55			return "", "", rest, false
56		}
57		return typ, query, rest, true
58	}
```

**`internal/tools/ui.go:60-63`** — the tool-set constructor `show_cards` joins (it is UI core, always-on; called from both `turn.Tools` and `turn.ToolsForHead`):
```go
60	// UITools returns the card_show, board_compose, and board_add_card tools.
61	func UITools(app core.App) []agent.Tool {
62		return []agent.Tool{cardShowTool(app), boardComposeTool(app), boardAddCardTool(app)}
63	}
```
> NOTE: plan **089** retires `board_compose`/`board_add_card` (boards are gone). Run the drift check first to see the live `UITools` shape, then add `showCardsTool(app)` as the SECOND element:
> - if 089 has landed: `return []agent.Tool{cardShowTool(app), showCardsTool(app)}`
> - if 089 has NOT landed: `return []agent.Tool{cardShowTool(app), showCardsTool(app), boardComposeTool(app), boardAddCardTool(app)}` (089 will drop the board tools later)
>
> Either way, also update the `UITools` doc comment (`ui.go:60`) to name the live tools. The `show_cards` tool is the *replacement* surface for the hand-picked-N-cards capability `board_compose` used to serve, now rendered into chat instead of a board.

**`internal/tools/ui.go:65-124`** — `cardShowTool`: build the description from `cards.All()` (mirror this for `show_cards`), validate via `cards.Validate`, return `MarkUICard(...)`:
```go
65	func cardShowTool(_ core.App) agent.Tool {
66		// Build a rich description that embeds the real registry vocabulary,
67		// so the model sees the actual types and their param docs.
68		var b strings.Builder
69		fmt.Fprint(&b, "Render a live UI card into the conversation. Choose a type from the registry; "+
70			"the server renders it from the owner's real data. Available types:\n")
71		for _, spec := range cards.All() {
...
114			cleaned, err := cards.Validate(args.Type, args.Params)
...
121			return MarkUICard(args.Type, cleaned, modelText), nil
122		},
123		}
124	}
```

**`internal/tools/ui.go:126-220`** — `boardComposeTool`: the 1–8 count validation + `[]cards.Card` JSON arg shape + `cards.ValidateCards` to copy for the cluster arg (the relevant lines):
```go
133		"cards": map[string]any{
134			"type":        "array",
135			"description": "1–8 cards, each with a type from the registry and optional params.",
136			"minItems":    1,
137			"maxItems":    8,
138			"items": map[string]any{
139				"type": "object",
140				"properties": map[string]any{
141					"type":   map[string]any{"type": "string", "description": "Card type from the registry."},
142					"params": map[string]any{"type": "object", "description": "Optional params (string values).", "additionalProperties": map[string]any{"type": "string"}},
...
150		var args struct {
151			Name  string       `json:"name"`
152			Cards []cards.Card `json:"cards"`
153		}
...
176		cleaned, err := cards.ValidateCards(args.Cards)
177		if err != nil {
178			return fmt.Sprintf("board_compose: %s", err), nil
179		}
```

**`internal/tools/choices.go:14-51`** — the JSON-head marker variant (closest precedent for serializing a structured payload into a marker head; `MarkArtifact` will JSON-encode `{title, cards}` the same way):
```go
17	const ChoicesMarker = "\x00balaur-choices:"
...
33	func MarkChoices(prompt string, choices []Choice, modelText string) string {
34		head := choicesHead{Prompt: prompt, Choices: choices}
35		b, _ := json.Marshal(head)
36		return ChoicesMarker + string(b) + "\n" + modelText
37	}
...
40	func ParseChoices(s string) (prompt string, choices []Choice, modelText string, ok bool) {
41		if !strings.HasPrefix(s, ChoicesMarker) {
42			return "", nil, s, false
43		}
44		s = strings.TrimPrefix(s, ChoicesMarker)
45		headLine, rest, _ := strings.Cut(s, "\n")
46		var head choicesHead
47		if err := json.Unmarshal([]byte(headLine), &head); err != nil {
48			return "", nil, rest, false
49		}
50		return head.Prompt, head.Choices, rest, true
51	}
```

**`internal/cards/cards.go:219-268`** — `cards.Card` (the reuse type) + `cards.ValidateCards` (validate every card against the registry; clamps layout). DO NOT invent a parallel type:
```go
226	type Card struct {
227		Type   string            `json:"type"`
228		Params map[string]string `json:"params,omitempty"`
229		X      int               `json:"x,omitempty"` // 0-based col, 0..11
230		Y      int               `json:"y,omitempty"` // 0-based row unit
231		W      int               `json:"w,omitempty"` // col span 1..12; 0 = spec default
232		H      int               `json:"h,omitempty"` // row-unit span; 0 = spec default
233	}
...
242	func ValidateCards(cs []Card) ([]Card, error) {
243		out := make([]Card, 0, len(cs))
244		for i, c := range cs {
245			cleaned, err := Validate(c.Type, c.Params)
246			if err != nil {
247				return nil, fmt.Errorf("card[%d]: %w", i, err)
248			}
...
267		return out, nil
268	}
```

**`internal/cards/cards.go:45-66`** — the registry `init()` and the existing `quests` spec (the new `tasks` spec is appended here in definition order):
```go
45	func init() {
46		registry = []Spec{
47			{
48				Type:  "today",
...
55			{
56				Type:  "quests",
57				Label: "Quest log",
58				Icon:  "scroll",
59				W:     8,
60				H:     30,
61				Params: []ParamSpec{
62					{Name: "mode", Enum: []string{"summary", "manage"}, Doc: "summary (read-only) or manage (Done/Snooze/Drop inline)"},
63					{Name: "status", Enum: []string{"open", "done", "all"}, Doc: "filter by task status (default: open)"},
64					{Name: "limit", Doc: "maximum rows to show (default 10, max 50)"},
65				},
66			},
```
> Note `cards.Validate` (`cards.go:328-335`) clamps `limit` to [1,50] and `days` to [1,366]; enum params (`status`) reject unknown values. A new `terms` free-string param is truncated to `maxParamLen` (256). So the `tasks` spec's params get the same validation for free.

**`internal/web/chatstream.go:176-209`** — the LIVE dispatcher (`handleToolResult`) and `endTool` (the append-via-`endTool` shape the artifact branch reuses):
```go
176	func (s *chatStream) handleToolResult(ev agent.Event) {
177		if typ, query, rest, ok := tools.ParseUICard(ev.Text); ok {
178			s.endTool(rest, s.h.uicardBody(typ, query))
179			return
180		}
181		if prompt, choices, _, ok := tools.ParseChoices(ev.Text); ok {
...
197		s.endTool(clipText(ev.Text, 2000), "")
198	}
199
...
202	func (s *chatStream) endTool(content string, card template.HTML) {
203		s.morphNode(chat.ToolRow(chat.ToolRowProps{
204			Tool: s.toolName, Icon: toolIconFile(s.toolName), ID: s.toolID, BodyID: s.toolBody, Content: content,
205		}))
206		if card != "" {
207			s.appendNode(g.El("div", g.Attr("class", "k-inline"), g.Attr("id", s.toolID+"-card"), g.Raw(string(card))))
208		}
209	}
```

**`internal/web/recap.go:253-281`** — the RELOAD dispatcher (`messageViews`, the tool branch). The SAME marker order, re-parsed from persisted `mv.Content`:
```go
253		// Re-render marked tool results.
254		// Consumer order: uicard → choices → proposal → refresh → plain.
...
262		if mv.Role == "tool" {
263			if typ, query, rest, ok := tools.ParseUICard(mv.Content); ok {
264				mv.CardBody = h.uicardBody(typ, query)
265				mv.Content = rest
266			} else if _, _, modelText, ok := tools.ParseChoices(mv.Content); ok {
267				mv.Content = clipText(modelText, 2000)
268			} else if kind, id, rest, ok := tools.ParseProposal(mv.Content); ok {
269				mv.CardBody, mv.Content = h.proposalBody(kind, id), rest
270			} else if _, rest, ok := tools.ParseRefresh(mv.Content); ok {
271				mv.Content = clipText(rest, 2000)
272			}
273		}
```
> NOTE: `messageView` carries a SINGLE `CardBody template.HTML` (recap.go:169) and `renderMessages` wraps it once in a `k-inline` div (recap.go:200-202). The cluster is one node containing N cards, so it fits this single-slot shape exactly — `h.artifactBody(...)` returns ONE `template.HTML` (the cluster organism), assigned to `mv.CardBody`. No new field is needed.

**`internal/web/cards.go:120-150`** — `h.cardHTML(typ, params)` (the layering-safe render-by-type-string seam: validates via `cards.Validate` then `ui.LookupCard`; error-strips on failure) and `h.uicardBody`:
```go
120	func (h *handlers) cardHTML(typ string, params map[string]string) template.HTML {
121		if _, ok := cards.Get(typ); !ok {
122			return cardErrorStrip("no such card type: " + typ)
123		}
124		cleaned, err := cards.Validate(typ, params)
125		if err != nil {
126			return cardErrorStrip(err.Error())
127		}
128		var b strings.Builder
129		if err := h.cardInto(&b, typ, cleaned); err != nil {
130			h.app.Logger().Warn("board card render failed", "type", typ, "err", err)
131			return cardErrorStrip("could not render this card")
132		}
133		return template.HTML(b.String())
134	}
...
147	func (h *handlers) uicardBody(typ, query string) template.HTML {
148		vals, _ := url.ParseQuery(query)
149		return h.cardHTML(typ, queryToMap(vals))
150	}
```

**`internal/turn/tools.go:54-65`** — `ToolsForHead` always-on core includes `tools.UITools(app)`, so `show_cards` is automatically available to every head once it joins `UITools`:
```go
63	// Always-on core: interaction + UI composition.
64	ts := tools.ChoiceTools(app)
65	ts = append(ts, tools.UITools(app)...)
```

**`internal/tasks/tasks.go:159-199`** — the loaders to REUSE (`OpenTasks(app, terms)` + `Bucket(recs, now)`); do NOT add new task queries:
```go
159	func OpenTasks(app core.App, terms []string) ([]*core.Record, error) {
160		filter := "status = 'open'"
161		params := dbx.Params{}
162		for i, t := range terms {
...
171		return app.FindRecordsByFilter("tasks", filter, "due", 200, 0, params)
172	}
...
180	// Bucket splits records by due against now's local day.
181	func Bucket(recs []*core.Record, now time.Time) Buckets {
...
198		return b
199	}
```
> `tasks.Buckets` (tasks.go:176-178) has fields `Overdue, Today, Upcoming, Someday []*core.Record`. The today card already composes `bk.Overdue + bk.Today` (today.go:37). The `tasks` cluster card's `due`/`bucket` filter maps to these buckets.

**`internal/feature/taskcards/taskcard.go:13-37`** — `TaskView` + `TaskCard(v)` (the individual card the bare stack renders, root id `tcard-{id}`):
```go
13	type TaskView struct {
14		ID, Title, Status, DueLine, RecurLine, Notes string
15		Overdue                                      bool
16	}
...
25	func TaskCard(v TaskView) g.Node {
26		return Article(
27			Class("kcard tcard tcard-"+v.Status), ID("tcard-"+v.ID),
...
37	}
```

**`internal/feature/taskcards/quests.go:87-117`** — `taskViewOf(rec, now)` + `viewsOf(recs, now)` (already build `[]TaskView` from records — REUSE for the bare stack; they are package-private to `taskcards`, which is exactly where the new card renderer lives):
```go
88	func taskViewOf(rec *core.Record, now time.Time) TaskView {
...
108	}
...
110	func viewsOf(recs []*core.Record, now time.Time) []TaskView {
111		out := make([]TaskView, 0, len(recs))
112		for _, r := range recs {
113			out = append(out, taskViewOf(r, now))
114		}
115		return out
116	}
```

**`internal/feature/taskcards/register.go:14-32`** — where the new `tasks` renderer joins `Register`/`Unregister`:
```go
14	func Register(app core.App) {
15		ui.RegisterCard("today", func(_ ui.CardSize, _ map[string]string) (g.Node, error) {
16			return TodayCard(buildToday(app)), nil
17		})
18		registerQuests(app)
...
22	}
...
26	func Unregister() {
27		ui.UnregisterCard("today")
28		ui.UnregisterCard("quests")
...
32	}
```

**`internal/ui/registry.go:14-33`** — the `CardFunc`/`RegisterCard`/`LookupCard` contract the `tasks` renderer satisfies:
```go
16	type CardFunc func(size CardSize, params map[string]string) (g.Node, error)
...
24	func RegisterCard(typ string, fn CardFunc) { cardRegistry[typ] = fn }
...
30	func LookupCard(typ string) (CardFunc, bool) {
```

**`internal/feature/storybook/story.go:53-103`** — the ordered story registry (add `tasksclusterStory()`/`tasksbareStory()` here) and `internal/feature/storybook/stories_cards.go:98-125` (`taskcardStory` is the template to copy for the new card's story). `TestAllStoriesRender` (`story_test.go:35`) renders every registered story's `Page()`.

**Layering (internal/ui/chat free of feature imports)** — confirmed: `internal/ui/chat/dock.go` injects pre-rendered `g.Node` slots and does NOT import `internal/feature/*`. The cluster organism added to `internal/ui/chat` (or `internal/ui`) must likewise take a pre-rendered `[]g.Node` (or `[]template.HTML`) — the web layer renders each card via `h.cardHTML` and hands the organism the rendered children. The organism never imports `internal/cards` or `internal/feature`.

## Commands you will need
| Purpose | Command | Expected |
|---|---|---|
| Drift check | see header | excerpts match |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test all | `go test ./...` | all pass |
| Marker round-trip tests | `go test ./internal/tools/...` | ok (incl. new artifact tests) |
| Card registry + cluster card | `go test ./internal/cards/... ./internal/feature/taskcards/...` | ok |
| Storybook renders | `go test ./internal/feature/storybook/...` | `TestAllStoriesRender` passes |
| show_cards registered | `grep -n 'showCardsTool' internal/tools/ui.go` (or artifact.go) | present, and called from `UITools` |
| Both gateway paths | `grep -n 'ParseArtifact' internal/web/chatstream.go internal/web/recap.go` | present in BOTH |
| Whitespace | `git diff --check` | no output |
| gofmt | `gofmt -l internal/` | no output |

Sandbox note: a TLS-intercepting Hyperagent sandbox needs the GOPROXY shim (`docs/hyperagent-sandbox.md`). No network is needed for this plan's tests.

## Scope
**In scope**:
- `internal/tools/artifact.go` (NEW; or extend `internal/tools/ui.go`) — `ArtifactMarker`, `MarkArtifact(cards []cards.Card, title, modelText) string`, `ParseArtifact(s) (title string, cs []cards.Card, rest string, ok bool)` (JSON head; `ParseArtifact` runs `cards.ValidateCards` and returns `ok=false` on any invalid card), and `showCardsTool(app)` joined into `tools.UITools`.
- `internal/web/cards.go` — `h.artifactBody(title string, cs []cards.Card) template.HTML` (loops `h.cardHTML` per card into the cluster organism; error-strips per-card like `cardHTML` does).
- `internal/web/chatstream.go` — a `ParseArtifact` branch in `handleToolResult`, **before** the plain fallback.
- `internal/web/recap.go` — the SAME `ParseArtifact` branch in `messageViews`.
- `internal/ui/chat/cluster.go` (NEW; or `internal/ui/`) — the `Cluster` organism: a titled container wrapping pre-rendered card nodes. Storybook Story added in the same change.
- `internal/cards/cards.go` — a new `tasks` Spec (bare individual-task cluster card) with `ParamSpecs`.
- `internal/feature/taskcards/taskscluster.go` (NEW) — `ui.RegisterCard("tasks", fn)` rendering a BARE stack of `TaskCard`s (no container chrome); joined into `Register`/`Unregister`.
- `internal/feature/storybook/` — one Story for the `Cluster` organism and one for the bare `tasks` card; registered in `story.go`.

**Out of scope** (do NOT touch / build):
- Any change to `card_show`/`UICardMarker` (the single-card seam is untouched; clusters are a parallel marker).
- Making 088's deterministic `/ui/show` endpoint accept a cluster — **clusters are agent-only for v1** (Step 7 decision; record it, do not build it). Sidebar items map to single cards.
- Persisting clusters as a new record type — they persist EXACTLY like `card_show` does (the marker in the tool message's `content`, written by 088's persistence path). No schema, no migration.
- Re-introducing boards or any grid/layout (X/Y/W/H) for clusters — the cluster is a simple vertical stack; the layout fields on `cards.Card` are ignored by the cluster renderer (they are spec-defaulted/zero for agent-composed clusters).
- A `bucket=` cross-product with `status=done` (a done task has no "overdue/today" bucket) — define `bucket` as applying to OPEN tasks only (Step 5).

## Git workflow
- Branch `improve/090-adhoc-card-clusters` off `main` (or stacked on 088's branch if not yet merged).
- Conventional commits, e.g.:
  - `feat(tools): ArtifactMarker + show_cards tool — hand-picked card clusters in chat`
  - `feat(ui): chat Cluster organism + storybook story`
  - `feat(web): render + reload-replay card clusters (both gateway paths)`
  - `feat(cards): filtered bare "tasks" cluster card drawing individual TaskCards`
- Do NOT push or open a PR unless explicitly told.

## Steps

### Step 1: The artifact marker + serialization (`internal/tools/artifact.go`)
Mirror `choices.go` (JSON head) for the structured payload. New file `internal/tools/artifact.go`:
```go
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
// without further validation.
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
```
**Why JSON head, not the `?query` form of UICardMarker**: a uicard carries one type + a flat param map (url.Values fits); a cluster carries an *array* of `{type, params}` — JSON is the honest encoding and matches `choices.go`'s precedent. `ValidateCards` runs INSIDE `ParseArtifact` (not just in the tool) so the reload path is equally guarded — a hand-edited or stale persisted marker with a now-unknown type degrades to plain text instead of rendering an error/panic.
**Verify**: `go build ./internal/tools/...` → exit 0.

### Step 2: The `show_cards` tool (in `artifact.go`, joined to `UITools`)
Add `showCardsTool(app)` mirroring `cardShowTool` (registry-vocabulary description from `cards.All()`) + `boardComposeTool`'s 1–8 count check and `[]cards.Card` arg:
```go
// ShowCardsMax bounds a cluster (mirrors board_compose's old 1–8 cap).
const showCardsMax = 8

func showCardsTool(_ core.App) agent.Tool {
	var b strings.Builder
	fmt.Fprint(&b, "Render a cluster of live UI cards into the conversation as ONE artifact "+
		"(e.g. 'show my quests and my weight together'). Pick 1–8 cards; each is a "+
		"{type, params} from the registry; the server renders each from the owner's real "+
		"data. To draw the owner's individual quests as separate cards, use the \"tasks\" "+
		"card (a bare stack of task cards) with a status/due/terms filter. Available types:\n")
	for _, spec := range cards.All() {
		// …same loop as cardShowTool: spec.Type (spec.Label) — params: …
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
				return fmt.Sprintf("show_cards: %s", err), nil // model self-corrects
			}
			title := strings.TrimSpace(args.Title)
			modelText := fmt.Sprintf("showing the owner a cluster of %d cards", len(cleaned))
			return MarkArtifact(cleaned, title, modelText), nil
		},
	}
}
```
Then join it in `internal/tools/ui.go`:
```go
func UITools(app core.App) []agent.Tool {
	return []agent.Tool{cardShowTool(app), showCardsTool(app), boardComposeTool(app), boardAddCardTool(app)}
}
```
(If 089 has already removed the board tools, the slice is `{cardShowTool(app), showCardsTool(app)}`.) No change to `turn/tools.go` is needed — `ToolsForHead` and `Tools` both pull in `UITools` already (Current state). The `obj`/`str` helpers live in `internal/tools/os.go` (same package).
**Verify**: `go build ./...`; `grep -n 'showCardsTool' internal/tools/ui.go` → present in `UITools`; `go vet ./...`.

### Step 3: The Cluster organism (`internal/ui/chat/cluster.go`) + Story
A pre-rendered-children container, so it stays free of `internal/feature`/`internal/cards`:
```go
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ClusterProps configures the chat.Cluster organism — one conversation
// artifact holding N pre-rendered cards. Children are rendered by the caller
// (web layer via h.cardHTML), so internal/ui/chat imports no feature/cards.
type ClusterProps struct {
	Title string   // optional heading; omitted when ""
	Cards []g.Node // pre-rendered card nodes, in order
}

// Cluster renders a titled vertical stack of cards as one inline artifact.
// Root id "k-cluster" (scoped by the enclosing k-inline div the gateway adds).
func Cluster(p ClusterProps) g.Node {
	kids := []g.Node{h.Class("k-cluster")}
	if p.Title != "" {
		kids = append(kids, h.Header(h.Class("k-cluster-head"), h.H3(g.Text(p.Title))))
	}
	body := append([]g.Node{h.Class("k-cluster-body")}, p.Cards...)
	kids = append(kids, h.Div(body...))
	return h.Div(kids...)
}
```
Add `.k-cluster`/`.k-cluster-head`/`.k-cluster-body` rules at the END of `internal/web/assets/static/basm.css`, tokenized (layout tokens `--space-*`; square corners; single-dash classes; no raw hex). The stack is a simple `display:flex; flex-direction:column; gap:var(--space-…)`. Verify legibility in light AND dark (the cluster sits on the parchment chat surface → `--ink` tokens). No animation is added, so no reduced-motion block is required; if you add a reveal transition, gate it under the existing reduced-motion block.

Add a Story in `internal/feature/storybook/stories_cards.go` (copy `taskcardStory` shape) and register it in `story.go`'s `stories` slice. Use pre-rendered `taskcards.TaskCard(...)` nodes as the fixture children (a cluster of a couple of cards + a titled vs untitled variant). Follow the `ui-development` skill: this is a new organism, so the Story is required in the same change.
**Verify**: `go build ./...`; `go test ./internal/feature/storybook/...` → `TestAllStoriesRender` passes.

### Step 4: `h.artifactBody` + the LIVE gateway branch
In `internal/web/cards.go`, add the cluster render seam (loops the layering-safe `h.cardHTML` per card):
```go
// artifactBody server-renders a hand-picked cluster of cards for an inline chat
// artifact: each card is rendered via cardHTML (validated + error-stripped) and
// wrapped in the chat.Cluster organism. Mirrors uicardBody for single cards.
func (h *handlers) artifactBody(title string, cs []cards.Card) template.HTML {
	nodes := make([]g.Node, 0, len(cs))
	for _, c := range cs {
		nodes = append(nodes, g.Raw(string(h.cardHTML(c.Type, c.Params))))
	}
	var b strings.Builder
	_ = chat.Cluster(chat.ClusterProps{Title: title, Cards: nodes}).Render(&b)
	return template.HTML(b.String())
}
```
(Add the `g "maragu.dev/gomponents"`, `internal/ui/chat`, and `internal/cards` imports to cards.go if not already present — `cards` is already imported; `chat` and `g` are not — confirm and add.)

Then in `internal/web/chatstream.go` `handleToolResult`, add the branch **before** the plain fallback (`s.endTool(clipText(ev.Text, 2000), "")`):
```go
	if title, cs, rest, ok := tools.ParseArtifact(ev.Text); ok {
		s.endTool(rest, s.h.artifactBody(title, cs))
		return
	}
```
Placement: after the existing uicard/choices/proposal/refresh branches, before the final plain `s.endTool`. Order vs uicard does not matter (markers are distinct NUL prefixes), but keep it grouped with the other card-bearing branches for readability.
**Verify**: `go build ./...`; `grep -n 'ParseArtifact' internal/web/chatstream.go` → present.

### Step 5: The RELOAD gateway branch (THE bug you must not miss)
In `internal/web/recap.go` `messageViews`, add the SAME parse to the `if mv.Role == "tool"` chain (insert it as a new `else if`, grouped with the other card branches):
```go
		} else if title, cs, rest, ok := tools.ParseArtifact(mv.Content); ok {
			mv.CardBody, mv.Content = h.artifactBody(title, cs), rest
		}
```
The single `mv.CardBody` slot holds the whole cluster (one `template.HTML` from the organism); `renderMessages` (recap.go:200-202) wraps it in one `k-inline` div, identical to how a single uicard re-renders on reload. **This step + Step 4 are inseparable** — landing only one ships the "vanishes on refresh / never appears live" bug.
**Verify**: `go build ./...`; `grep -n 'ParseArtifact' internal/web/recap.go` → present. Both greps (Step 4 + this) must return a hit — confirm with `grep -rn 'ParseArtifact' internal/web/` showing it in BOTH files.

### Step 6: The filtered bare `tasks` cluster card
**6a — registry spec.** Append a `tasks` Spec to `registry` in `internal/cards/cards.go` `init()` (after `quests`, in definition order). Params (all optional, model-composable):
```go
{
	Type:  "tasks",
	Label: "Tasks",
	Icon:  "scroll",
	W:     4,
	H:     20,
	Params: []ParamSpec{
		{Name: "status", Enum: []string{"open", "done", "all"}, Doc: "filter by task status (default: open)"},
		{Name: "bucket", Enum: []string{"overdue", "today", "upcoming", "someday"}, Doc: "narrow OPEN tasks to one due bucket (ignored unless status is open)"},
		{Name: "terms", Doc: "space-separated search terms matched against title and notes (ANDed)"},
		{Name: "limit", Doc: "maximum cards to draw (default 12, max 50)"},
	},
},
```
> `status`/`bucket` are enums (rejected if unknown by `cards.Validate`). `terms` is a free string (truncated to 256). `limit` is clamped to [1,50] by `cards.Validate`. Do NOT add an `ids` param in v1 — the agent has no stable way to know record ids before drawing, and `terms` covers "draw the cards for THOSE quests"; record `ids` as deferred (Maintenance notes).

**6b — renderer.** New file `internal/feature/taskcards/taskscluster.go`. A BARE stack of individual `TaskCard`s — NO `ui.CardHead`, NO `Footer`, NO outer `kcard`/`ucard` container (contrast the `quests` card at quests.go:24-33 which wraps in an `Article.kcard.ucard` with head + footer). Reuse `tasks.OpenTasks` + `tasks.Bucket` + the existing package-private `viewsOf`/`taskViewOf`:
```go
package taskcards

import (
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

// renderTasks draws matching tasks as a BARE stack of individual TaskCards —
// no container/header chrome (contrast the quests summary card). It is the
// "draw the cards for those quests" surface, usable by card_show, /ui/show, and
// show_cards clusters.
func renderTasks(app core.App, params map[string]string) g.Node {
	now := time.Now()
	limit := intParam(params, "limit", 12) // cards.Validate already clamped to [1,50]
	status := params["status"]
	if status == "" {
		status = "open"
	}

	var recs []*core.Record
	switch status {
	case "done":
		recs, _ = app.FindRecordsByFilter("tasks", "status = 'done'", "-updated", limit, 0)
	case "all":
		recs, _ = app.FindRecordsByFilter("tasks", "status != 'dropped'", "-updated", limit, 0)
	default: // open
		var terms []string
		if t := strings.TrimSpace(params["terms"]); t != "" {
			terms = strings.Fields(t)
		}
		open, _ := tasks.OpenTasks(app, terms)
		open = filterBucket(open, params["bucket"], now) // no-op when bucket==""
		if len(open) > limit {
			open = open[:limit]
		}
		recs = open
	}

	rows := viewsOf(recs, now)
	if len(rows) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No tasks match."})
	}
	items := make([]g.Node, 0, len(rows))
	for _, r := range rows {
		items = append(items, TaskCard(r))
	}
	// Bare stack — no card container/head/footer.
	return Div(Class("tasks-stack"), g.Group(items))
}

// filterBucket narrows OPEN tasks to one due bucket via tasks.Bucket. Empty
// bucket returns recs unchanged. (Only meaningful for status=open.)
func filterBucket(recs []*core.Record, bucket string, now time.Time) []*core.Record {
	if bucket == "" {
		return recs
	}
	bk := tasks.Bucket(recs, now)
	switch bucket {
	case "overdue":
		return bk.Overdue
	case "today":
		return bk.Today
	case "upcoming":
		return bk.Upcoming
	case "someday":
		return bk.Someday
	}
	return recs
}

func registerTasks(app core.App) {
	ui.RegisterCard("tasks", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return renderTasks(app, params), nil
	})
}
```
Add `registerTasks(app)` to `Register` and `ui.UnregisterCard("tasks")` to `Unregister` in `register.go`. `intParam`/`viewsOf`/`taskViewOf`/`TaskCard` are already in the package (quests.go / taskcard.go). Add a `.tasks-stack` flex-column rule at the END of basm.css (gap via `--space-*`).
> Note `TaskCard` renders Done/Snooze/Drop forms that `@post` to `/ui/tasks/{id}/transition` (taskcard.go:60-83) — that endpoint already exists and works inline; the bare stack inherits working actions for free. A `done`/`all` task shows its status word instead of actions (taskcard.go:61-63), which is correct.

**6c — Story.** Add a `tasksbareStory()` in `stories_cards.go` (bare stack of 2–3 `taskcards.TaskCard` fixtures; document the `status`/`bucket`/`terms`/`limit` params and the "no container chrome" intent — Don't: "wrap it in a kcard header — that's the `quests` card's job"). Register in `story.go`.
**Verify**: `go test ./internal/cards/... ./internal/feature/taskcards/... ./internal/feature/storybook/...` → all pass; `go vet ./...`.

### Step 7: Decision — does `/ui/show` (088) accept clusters? (record, do NOT build)
**Recommendation: agent-only clusters for v1.** The deterministic sidebar (088) maps each domain item to a SINGLE card type (Quests→`quests`, Life→`lifelog`, etc.) — a sidebar click is "show me this domain's artifact," which is one card. Clusters are a *conversational* power ("show my quests AND my weight"), naturally expressed in language, so `show_cards` (agent) is the right and only door for v1. Making `/ui/show` accept a `cards=[…]` cluster spec would need a URL/JSON encoding for an array on a GET endpoint and a new sidebar UI to compose one — scope 088 does not have. **Do not extend `/ui/show`.** Record this so a future "let the owner save a cluster as a sidebar shortcut" knows it is net-new work. (If the owner later wants it, the cleanest path is a small `clusters` collection of saved `[]cards.Card` + a sidebar section — a separate plan.)
This step is documentation only; nothing to build. Note it in the plan's Maintenance notes and `internal/self/knowledge.md` (Step 8).

### Step 8: Self-knowledge + index
- Update `internal/self/knowledge.md` where it lists the agent UI tools / card composition: add `show_cards` (hand-picked cluster artifact) and the `tasks` bare-cluster card beside `card_show`. A stale self-description makes Balaur lie about itself (AGENTS.md).
- Update the 090 row in `plans/readme.md` (do not touch other rows; the advisor owns the index format).
**Verify**: `grep -n 'show_cards' internal/self/knowledge.md` → present.

## Test plan
- **`internal/tools`** (new `artifact_test.go`): (1) `MarkArtifact`→`ParseArtifact` round-trip preserves title + cards; (2) `ParseArtifact` on plain text / a `UICardMarker` / a `ChoicesMarker` string → `ok=false` (cross-marker non-collision, mirroring `TestParseUICardOnChoicesMarked`); (3) `ParseArtifact` on a marker whose head contains an UNKNOWN card type → `ok=false` (ValidateCards rejection, the reload-path guard); (4) `show_cards` Execute: happy path returns a string that `ParseArtifact` accepts; `>showCardsMax` cards → plain error string (not a marker); 0 cards → plain error.
- **`internal/cards`**: `tasks` spec is registered (`Get("tasks")` ok); `Validate("tasks", …)` clamps `limit`, rejects an unknown `status`/`bucket` enum, truncates a long `terms`.
- **`internal/feature/taskcards`** (new `taskscluster_test.go`, using the `internal/store` temp-app helper like the existing `quests_test.go`): seed open/overdue/done tasks; `renderTasks` with `status=open` returns one `tcard-{id}` per matching task and NO `ucard-quests`/`kcard-head` container; `bucket=overdue` narrows; `terms` filters; empty result renders the empty state. Render the node to a string and assert on the markup (mirror the existing taskcards tests' approach).
- **Web reload-replay** (the headline regression net — extend `internal/web/handlers_test.go`): persist a tool message whose `content` is a `MarkArtifact(...)` string (write a `messages` row via the same path `card_show` uses), then load history through `messageViews` and assert `mv.CardBody` is non-empty and contains the expected per-card root ids (e.g. `tcard-`/`ucard-`). This proves the RELOAD branch (Step 5) re-renders the cluster. If a live-stream harness exists for `card_show`, add the symmetric live assertion; otherwise the `grep` gate (both files contain `ParseArtifact`) + this reload test together cover both paths.
- **Storybook**: `go test ./internal/feature/storybook/...` → `TestAllStoriesRender` covers the new `Cluster` + `tasks` stories.

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0; `gofmt -l internal/` → no output; `git diff --check` → no output.
- [ ] `go test ./...` → all pass, including the new `artifact_test.go`, `taskscluster_test.go`, the `tasks` registry tests, and the web reload-replay test.
- [ ] `tools.ArtifactMarker`/`MarkArtifact`/`ParseArtifact` exist; `ParseArtifact` runs `cards.ValidateCards` and returns `ok=false` for any unknown card type (proven by test).
- [ ] `show_cards` is in `tools.UITools` (so both `turn.Tools` and `turn.ToolsForHead` expose it); its description is built from `cards.All()`; it caps the cluster at 1–8 cards.
- [ ] `grep -rn 'ParseArtifact' internal/web/` shows it in **BOTH** `chatstream.go` (live) **and** `recap.go` (reload).
- [ ] `h.artifactBody` renders each card via `h.cardHTML` into the `chat.Cluster` organism; a single bad card error-strips without blanking the cluster.
- [ ] `chat.Cluster` organism exists, imports no `internal/feature`/`internal/cards`, and has a storybook Story; `.k-cluster*` CSS appended to basm.css, legible in light + dark.
- [ ] `cards.Get("tasks")` returns a spec; `ui.LookupCard("tasks")` returns a renderer after `taskcards.Register`; the renderer draws a BARE stack of `TaskCard`s (no `kcard`/`ucard` container) and is in `Register`/`Unregister`; it has a storybook Story.
- [ ] The `tasks` card is reachable by `card_show`, `/ui/show` (088), AND `show_cards` (one registry, three doors).
- [ ] `/ui/show` (088) is NOT extended for clusters; the agent-only decision is recorded (knowledge.md + Maintenance notes).
- [ ] `internal/self/knowledge.md` updated (show_cards + tasks card); `plans/readme.md` 090 row updated.

## STOP conditions
- **Drift**: a cited file changed since `3136bad` and an excerpt no longer matches — STOP and report (especially `handleToolResult`'s branch order or `messageViews`' tool chain, and whether 089 already changed `UITools`).
- **Only one gateway path updated**: if `ParseArtifact` lands in `chatstream.go` but not `recap.go` (or vice-versa) — STOP; this is the headline bug. Both, same commit.
- **A parallel card type**: tempted to define a new struct instead of reusing `cards.Card`/`cards.ValidateCards` — STOP; reuse the registry type (it is the one source of truth and the validation invariant lives there).
- **`ParseArtifact` skips `ValidateCards`**: validating only in the tool and trusting the persisted marker on reload — STOP; the reload path MUST re-validate (a stale/hand-edited marker with a removed type must degrade to plain text, never render an error card or panic).
- **The `tasks` card grows container chrome**: adding `ui.CardHead`/a `kcard` wrapper to the bare stack — STOP; that duplicates the `quests` card. The whole point is bare individual `TaskCard`s.
- **Scope creep into `/ui/show` clusters** or a `clusters` collection — STOP; v1 is agent-only (Step 7).
- **A Verify fails twice** after a fix attempt — STOP and report the command + output.

## Maintenance notes
- **Two paths, one marker — forever.** Any future marker (or change to the artifact head shape) must be handled in BOTH `chatstream.handleToolResult` AND `recap.messageViews`. This is now a four-marker pattern (uicard, choices, proposal, refresh) plus artifact; the duplication is deliberate (live vs reload are genuinely different render contexts) but every addition pays the two-place tax. A reviewer should always grep both files.
- **The cluster carries the model's intent, not layout.** `cards.Card` has X/Y/W/H fields (from the boards era); the cluster renderer ignores them — it is a vertical stack. If a future plan wants grid clusters, that is a new organism + a deliberate decision, not an accident of the reused type.
- **`tasks` vs `quests`**: `quests` is the summary/manage roll-up (one card, container chrome, a list). `tasks` is the bare individual-card stack (the agent's "draw THESE as cards"). Keep them distinct; do not merge.
- **Deferred (record, don't build)**: an `ids` param on the `tasks` card (needs a stable id surface for the agent); `/ui/show` accepting clusters / saved sidebar cluster shortcuts (Step 7 — likely a small `clusters` collection + sidebar section in a later plan); a cluster reveal animation (add under the reduced-motion block if ever wanted).
- **If 089 lands first**: `UITools` will already have dropped `board_compose`/`board_add_card`; `show_cards` is then the sole multi-card composition tool. Confirm the `UITools` slice shape during the drift check.
