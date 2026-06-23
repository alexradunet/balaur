// Package turn is Balaur's channel-agnostic conversation layer: one owner
// turn runs through the same pipeline — context assembly, the agent loop,
// the runtime honesty check, persistence — no matter which surface carried
// it. Gateways (web, CLI, future messengers) adapt events to their medium
// and render the Result; they never re-implement the behavior. That is
// what makes one surface's output trustworthy evidence about another's.
package turn

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/heads"
	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/verify"
)

// RecentTurnWindow caps how many prior text turns enter the model context.
// Persistence is unbounded; context is not (the master-conversation
// footgun defusal — see internal/conversation).
const RecentTurnWindow = 20

// maxSteps reads the tool-round cap for one turn. The agent default (8)
// fits conversation; self-development sessions legitimately need more
// rounds (read, edit, test, build, verify), so BALAUR_MAX_STEPS raises it
// explicitly. 0 keeps the agent package's default.
func maxSteps() int {
	n, err := strconv.Atoi(os.Getenv("BALAUR_MAX_STEPS"))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// Result is what one turn produced, after persistence.
type Result struct {
	// Reply is the final visible assistant text.
	Reply string
	// Turn holds every message the loop appended during the run —
	// assistant and tool rounds, including any self-repair pass — in
	// order. These are the persisted records of the turn.
	Turn []llm.Message
	// CheckNote is the owner-facing honesty note. Non-empty only when the
	// reply claimed a capture no tool performed and self-repair also
	// failed; it has already been persisted with origin "check".
	CheckNote string
	// UsedMemories are the approved memory records injected into context.
	UsedMemories []*core.Record
}

// Run drives one owner turn to completion: persist the user message,
// assemble context, run the agent loop (events stream to emit, which may
// be nil), apply the verify honesty check with one self-repair pass, and
// persist everything the loop appended. The Result reflects what was
// persisted even when the loop or persistence erred mid-turn; both loop and
// persistence errors are joined and returned so callers can report them
// (gateways will usually have surfaced them already via the "error" event).
func Run(ctx context.Context, app core.App, client llm.Client, userText string, emit func(agent.Event)) (Result, error) {
	if emit == nil {
		emit = func(agent.Event) {}
	}
	var res Result

	// The master conversation: load the recent window BEFORE persisting
	// the new user turn, so the window holds prior turns only.
	master, err := conversation.Master(app)
	if err != nil {
		return res, err
	}
	head := heads.Active(app)
	recent, err := conversation.RecentTurns(app, master.Id, RecentTurnWindow)
	if err != nil {
		return res, err
	}
	if err := conversation.Append(app, master.Id, llm.Message{Role: "user", Content: userText}, ""); err != nil {
		return res, err
	}

	// Context = system prompt + present moment + today block + knowledge
	// block + recent turns + this turn. Persistence is not context: the
	// full record stays in SQLite. The moment line grounds relative dates;
	// the today block is what lets the companion speak like someone who
	// knows the owner's day, unprompted.
	now := time.Now().In(store.OwnerLocation(app))
	knowledgeBlock, usedMemories := knowledge.BuildContext(app, userText)
	res.UsedMemories = usedMemories
	todayBlock := tasks.TodayBlock(app, now)
	loop := &agent.Loop{Client: client, Tools: ToolsForHead(app, head.Groups), MaxSteps: maxSteps()}
	history := make([]llm.Message, 0, len(recent)+2)
	history = append(history, llm.Message{Role: "system", Content: systemPrompt + headFlavor(head.Name, head.Purpose) + nowLine(now) + todayBlock + knowledgeBlock})
	history = append(history, recent...)
	history = append(history, llm.Message{Role: "user", Content: userText})
	contextLen := len(history)

	final, runErr := loop.Run(ctx, history, emit)
	res.Turn = final[contextLen:]

	// Runtime honesty check (verify, don't trust): a reply must not claim
	// a capture that no tool performed. One self-repair pass gives the
	// model the chance to actually call the tool; if it still claims
	// without doing, the owner sees a plain note. The correction message
	// is scaffolding — model context only, never persisted.
	if runErr == nil && !verify.CaptureSucceeded(res.Turn) && verify.ClaimsCapture(verify.LastAssistantText(res.Turn)) {
		retryBase := append(final, llm.Message{Role: "user", Content: verify.Correction})
		if final2, retryErr := loop.Run(ctx, retryBase, emit); retryErr == nil {
			res.Turn = append(res.Turn, final2[len(retryBase):]...)
		}
		if !verify.CaptureSucceeded(res.Turn) && verify.ClaimsCapture(verify.LastAssistantText(res.Turn)) {
			res.CheckNote = verify.Note
		}
	}

	// Persist every turn the loop appended (assistant and tool rounds).
	// Tool turns carry the call id; map it back to the tool's name from
	// the preceding assistant turn so the record reads human.
	//
	// When the honesty check failed (CheckNote set), the assistant's visible
	// text claimed a capture no tool performed. Persist that text with
	// OriginUncommitted: the record keeps it (the owner still sees what was
	// said, corrected by the note below) but RecentTurns bars it from context,
	// so the model never reads its own unbacked "Task saved" back as a pattern
	// to imitate — the failure that snowballs a poisoned thread.
	var persistErr error
	toolNames := map[string]string{}
	for _, m := range res.Turn {
		name := ""
		if m.Role == "tool" {
			name = toolNames[m.ToolCallID]
		}
		for _, tc := range m.ToolCalls {
			toolNames[tc.ID] = tc.Name
		}
		origin := ""
		if res.CheckNote != "" && m.Role == "assistant" && m.Content != "" {
			origin = conversation.OriginUncommitted
		}
		if err := conversation.AppendOrigin(app, master.Id, m, name, origin); err != nil {
			persistErr = fmt.Errorf("persisting turn: %w", err)
			break // do not break the caller's stream mid-reply; the error travels in the return
		}
	}

	// Memories that informed this turn count as used.
	for _, m := range usedMemories {
		knowledge.Touch(app, knowledge.Memory, m)
	}

	if res.CheckNote != "" {
		if err := conversation.AppendOrigin(app, master.Id,
			llm.Message{Role: "assistant", Content: res.CheckNote}, "", conversation.OriginCheck); err != nil {
			persistErr = errors.Join(persistErr, fmt.Errorf("persisting check note: %w", err))
		}
	}

	res.Reply = verify.LastAssistantText(res.Turn)
	return res, errors.Join(runErr, persistErr)
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
	"remembered until it is.\n\n" +
	"Commitments: when the owner voices something to do, a deadline, or a " +
	"repeating practice, capture it with `task_add` — a concrete due time " +
	"computed from the present moment stated below when one is implied, " +
	"recurrence like daily, every:3d, weekly:mon,thu or monthly:15 for " +
	"repeating ones, and useful context folded into notes. " +
	"A commitment exists ONLY after a task_add result says 'Task saved' — " +
	"never tell the owner a reminder is set without that result in this turn; " +
	"when unsure, check task_list. For weekday rules the first due must land " +
	"on one of the named weekdays, computed from the present moment. " +
	"Check `task_list` before claiming what is or isn't on the book; mark " +
	"things done with `task_done` when the owner says they did them — never " +
	"call task_done unprompted. Snooze or drop on request. To reschedule or " +
	"edit an existing task — a new due time, a renamed title, corrected notes, " +
	"or a changed recurrence — use `task_update` with the task's id; never tell " +
	"the owner you changed a task without a task_update result this turn. When " +
	"they ask how a habit or ritual has been going, use `task_history` for its " +
	"completions and streak. Never invent tasks the owner didn't voice.\n\n" +
	"Life log: when the owner reports something they track — a measurement, " +
	"a practice, a milestone — keep it with `log_entry`, using a short " +
	"consistent kind. Check `entry_series` (without a kind) for the kinds " +
	"already in use before coining a new one (prefer singular names: weight, " +
	"not weights); the owner decides what is worth tracking. Log only what " +
	"they state, never invent values, and never moralize about the numbers.\n\n" +
	"Journal: when the owner offers a reflection for the record — or asks to " +
	"journal something — keep it with `journal_write`, their words VERBATIM, " +
	"never paraphrased or embellished. Offer gently when something reads like " +
	"a diary line; never push, never write it unasked. Their thoughts live on " +
	"the day pages (/day).\n\n" +
	"Yourself: when the owner asks what you can do, how you work, or about " +
	"your own code, consult the `self` tool first (sections: overview, " +
	"architecture, capabilities, source, devloop) — never guess about your " +
	"own capabilities. With OS access enabled and BALAUR_SOURCE configured " +
	"you can analyze and develop your own code: follow the devloop section " +
	"exactly, and never claim a fix is tested or built without those tool " +
	"results in this turn. You can grow new tools by writing a " +
	"balaur-extension and submitting it with `propose_extension`; it runs " +
	"only after the owner approves, so never claim an extension capability " +
	"before its tool exists in your registry."

// headFlavor frames the active head's purpose as an addendum to the base
// Balaur system prompt. The main head (empty purpose) adds nothing.
func headFlavor(name, purpose string) string {
	if purpose == "" {
		return ""
	}
	return "\n\nRight now you answer as your " + name + " head — " + purpose + "."
}

// nowLine grounds the model in the present moment. Relative dates in the
// owner's words ("tomorrow at 10") must resolve against the box's clock
// and timezone — never against the model's training prior.
func nowLine(now time.Time) string {
	zone, _ := now.Zone()
	return fmt.Sprintf("\n\nThe present moment: %s (%s, UTC%s). "+
		"Resolve every relative date and time the owner says — today, tonight, "+
		"tomorrow, in two hours, next friday — against this moment, in this timezone.",
		now.Format("Monday, January 2 2006, 15:04"), zone, now.Format("-07:00"))
}
