// Package verify is the runtime's distrust of unverified claims: small
// models lie politely ("Setting it now") without calling any tool. The
// agent loop knows which tools actually ran, so the runtime audits words
// against deeds — deterministically, no extra model judgment. Detection is
// pattern-based and precision-tuned: it cannot catch every phrasing, but
// the caller's repair pass turns the common failure into a real tool call,
// and an honest note covers the rest. Verify, don't trust.
package verify

import (
	"regexp"
	"strings"

	"github.com/alexradunet/balaur/internal/llm"
)

// captureTools are the mutating verbs whose success makes a capture claim
// legitimate.
var captureTools = map[string]bool{
	"task_add": true, "task_done": true, "task_snooze": true, "task_drop": true,
	"log_entry": true, "entry_drop": true, "journal_write": true,
}

// Correction is the synthetic turn fed back to the model for one
// self-repair pass. It is scaffolding, not conversation — callers must not
// persist it.
const Correction = "[runtime check] Your reply claims a reminder, task, or log entry " +
	"was saved, but no capture tool succeeded this turn. Either call the right tool " +
	"NOW to actually save it, or plainly tell the owner nothing was saved. Do not " +
	"repeat the claim without a tool result."

// Note is the owner-facing line when self-repair also failed.
const Note = "Runtime check: the reply above claims something was saved, but no " +
	"capture tool ran this turn. Nothing is on the book from it — ask again, and " +
	"trust the task card, not the words."

// CaptureSucceeded reports whether any capture tool returned a non-error
// result within the turn. Tool messages carry only call ids; names come
// from the preceding assistant turns' tool calls.
func CaptureSucceeded(turn []llm.Message) bool {
	names := map[string]string{}
	for _, m := range turn {
		for _, tc := range m.ToolCalls {
			names[tc.ID] = tc.Name
		}
		if m.Role == "tool" && captureTools[names[m.ToolCallID]] &&
			!strings.HasPrefix(m.Content, "error:") {
			return true
		}
	}
	return false
}

// LastAssistantText returns the turn's final visible reply.
func LastAssistantText(turn []llm.Message) string {
	for i := len(turn) - 1; i >= 0; i-- {
		if turn[i].Role == "assistant" && strings.TrimSpace(turn[i].Content) != "" {
			return turn[i].Content
		}
	}
	return ""
}

// claimPatterns assert a capture happened. Precision over recall: every
// pattern is anchored on affirmative phrasing observed in real transcripts;
// offers and questions are excluded sentence-wise below.
var claimPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bsetting (it|that|this|the (reminder|task|nudge)) now\b`),
	regexp.MustCompile(`(?i)\bi('ll| will) remind you\b`),
	regexp.MustCompile(`(?i)\b(i have|i've|i just|just) (set|saved|created|added|scheduled|logged)\b`),
	regexp.MustCompile(`(?i)\b(reminder|task|nudge|it)( is| was|'s)( now| already)? (set|saved|created|added|scheduled|in place)\b`),
	regexp.MustCompile(`(?i)\balready set\b`),
	regexp.MustCompile(`(?i)\b(reminder|nudge) (due|at|set for|scheduled for) \d{1,2}[:.]\d{2}\b`),
}

// offerGuard marks a sentence as an offer or request, not a claim.
var offerGuard = regexp.MustCompile(`(?i)\b(want me to|shall i|should i|can i|could i|would you like|do you want)\b`)

// ClaimsCapture reports whether the text asserts that a capture happened.
func ClaimsCapture(text string) bool {
	for _, sentence := range splitSentences(text) {
		if strings.Contains(sentence, "?") || offerGuard.MatchString(sentence) {
			continue
		}
		for _, p := range claimPatterns {
			if p.MatchString(sentence) {
				return true
			}
		}
	}
	return false
}

func splitSentences(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '\n'
	})
}
