// Package bridge is the EXPERIMENTAL reference messenger bridge (plan 256
// spike): an owner-run process that long-polls the Telegram Bot API and
// relays allowlisted messages to a local Balaur's POST /api/messenger/turn.
// It deliberately reads no PocketBase records — all config arrives via
// Config (flags/env in internal/cli/bridge.go). See
// docs/superpowers/specs/2026-07-02-messenger-bridge-design.md.
package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Config configures a bridge run. Zero-valued optional fields get sane
// defaults inside Run.
type Config struct {
	BotToken        string        // Telegram bot token (secret — never logged, URLs containing it never logged)
	MessengerToken  string        // Balaur owner_settings.messenger_token value (secret — never logged)
	BalaurURL       string        // e.g. http://127.0.0.1:8090
	TelegramBaseURL string        // default https://api.telegram.org; tests point it at an httptest server
	AllowedChatIDs  []int64       // fail-closed sender allowlist; empty ⇒ Run refuses to start
	PollTimeout     time.Duration // getUpdates long-poll timeout (default 50s)
	RetryBase       time.Duration // backoff base (default 1s; tests use ~1ms)
	HTTP            *http.Client  // default: &http.Client{Timeout: PollTimeout + 10s}
}

const (
	defaultTelegramBaseURL = "https://api.telegram.org"
	defaultPollTimeout     = 50 * time.Second
	defaultRetryBase       = time.Second
	backoffCapFactor       = 60 // backoff caps at 60x the retry base
	maxBusyRetries         = 5  // bounded retries for a Balaur 429 busy response
)

// Run polls until ctx is cancelled; cancellation is the graceful shutdown
// path and returns nil.
func Run(ctx context.Context, cfg Config, log *slog.Logger) error {
	if cfg.BotToken == "" {
		return errors.New("bridge config: BotToken is required")
	}
	if cfg.MessengerToken == "" {
		return errors.New("bridge config: MessengerToken is required")
	}
	if cfg.BalaurURL == "" {
		return errors.New("bridge config: BalaurURL is required")
	}
	if len(cfg.AllowedChatIDs) == 0 {
		return errors.New("bridge: --allow-chat is required; refusing to start without a sender allowlist")
	}
	if cfg.TelegramBaseURL == "" {
		cfg.TelegramBaseURL = defaultTelegramBaseURL
	}
	if cfg.PollTimeout <= 0 {
		cfg.PollTimeout = defaultPollTimeout
	}
	if cfg.RetryBase <= 0 {
		cfg.RetryBase = defaultRetryBase
	}
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: cfg.PollTimeout + 10*time.Second}
	}

	allowed := make(map[int64]bool, len(cfg.AllowedChatIDs))
	for _, id := range cfg.AllowedChatIDs {
		allowed[id] = true
	}

	r := &runner{cfg: cfg, log: log, allowed: allowed}
	return r.loop(ctx)
}

// runner carries the resolved config and allowlist through one poll loop.
type runner struct {
	cfg     Config
	log     *slog.Logger
	allowed map[int64]bool
}

// tgUpdatesResponse is the Telegram getUpdates response shape.
type tgUpdatesResponse struct {
	OK     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}

type tgUpdate struct {
	UpdateID int64      `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

type tgMessage struct {
	Chat tgChat `json:"chat"`
	Text string `json:"text"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

// loop long-polls getUpdates and dispatches each update until ctx is done.
func (r *runner) loop(ctx context.Context) error {
	offset := int64(0)
	backoff := r.cfg.RetryBase
	for {
		if ctx.Err() != nil {
			return nil
		}
		updates, err := r.getUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			r.log.Warn("bridge: getUpdates failed", "error", err)
			if waitCtx(ctx, backoff) != nil {
				return nil
			}
			backoff = nextBackoff(backoff, r.cfg.RetryBase)
			continue
		}
		backoff = r.cfg.RetryBase
		for _, u := range updates {
			r.handleUpdate(ctx, u)
			offset = u.UpdateID + 1
		}
	}
}

// getUpdates calls Telegram's getUpdates and returns the batch of updates.
// Errors are stripped of the request URL (it embeds the bot token) before
// being returned, so callers can safely log them.
func (r *runner) getUpdates(ctx context.Context, offset int64) ([]tgUpdate, error) {
	q := url.Values{}
	q.Set("timeout", strconv.Itoa(int(r.cfg.PollTimeout.Seconds())))
	q.Set("offset", strconv.FormatInt(offset, 10))
	endpoint := fmt.Sprintf("%s/bot%s/getUpdates?%s", r.cfg.TelegramBaseURL, r.cfg.BotToken, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("building getUpdates request: %w", stripURL(err))
	}
	resp, err := r.cfg.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getUpdates request: %w", stripURL(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getUpdates: unexpected status %d", resp.StatusCode)
	}
	var body tgUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding getUpdates response: %w", err)
	}
	return body.Result, nil
}

// handleUpdate applies the sender allowlist, then relays an allowlisted
// message through Balaur's messenger endpoint and delivers the reply.
func (r *runner) handleUpdate(ctx context.Context, u tgUpdate) {
	if u.Message == nil || u.Message.Text == "" {
		return
	}
	chatID := u.Message.Chat.ID
	if !r.allowed[chatID] {
		r.log.Info("bridge: rejected sender", "chat_id", chatID)
		return
	}

	backoff := r.cfg.RetryBase
	for attempt := 0; attempt < maxBusyRetries; attempt++ {
		outcome, err := r.postTurn(ctx, u.Message.Text)
		if err != nil {
			r.log.Warn("bridge: turn request failed", "error", err)
			r.deliver(ctx, chatID, "Something went wrong — check Balaur's logs.")
			return
		}
		if !outcome.busy {
			r.deliver(ctx, chatID, outcome.reply)
			return
		}
		if waitCtx(ctx, backoff) != nil {
			return
		}
		backoff = nextBackoff(backoff, r.cfg.RetryBase)
	}
	r.deliver(ctx, chatID, "Balaur is busy — try again in a moment.")
}

// turnOutcome is the result of one POST to Balaur's messenger endpoint.
type turnOutcome struct {
	reply string
	busy  bool
}

// postTurn POSTs one message to Balaur's messenger endpoint. It never logs
// or returns the message text or the messenger token.
func (r *runner) postTurn(ctx context.Context, text string) (turnOutcome, error) {
	payload, err := json.Marshal(struct {
		Message string `json:"message"`
	}{Message: text})
	if err != nil {
		return turnOutcome{}, fmt.Errorf("encoding turn payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.cfg.BalaurURL+"/api/messenger/turn", bytes.NewReader(payload))
	if err != nil {
		return turnOutcome{}, fmt.Errorf("building turn request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.cfg.MessengerToken)

	resp, err := r.cfg.HTTP.Do(req)
	if err != nil {
		return turnOutcome{}, fmt.Errorf("turn request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return turnOutcome{busy: true}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return turnOutcome{}, fmt.Errorf("turn: unexpected status %d", resp.StatusCode)
	}
	var body struct {
		Reply string `json:"reply"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return turnOutcome{}, fmt.Errorf("decoding turn response: %w", err)
	}
	return turnOutcome{reply: body.Reply}, nil
}

// deliver sends text to chatID via Telegram's sendMessage, retrying once on
// failure before logging a warning and giving up.
func (r *runner) deliver(ctx context.Context, chatID int64, text string) {
	if err := r.sendMessage(ctx, chatID, text); err != nil {
		r.log.Warn("bridge: sendMessage failed, retrying once", "error", err)
		if err := r.sendMessage(ctx, chatID, text); err != nil {
			r.log.Warn("bridge: sendMessage retry failed, giving up", "error", err)
		}
	}
}

// sendMessage calls Telegram's sendMessage. Errors are stripped of the
// request URL (it embeds the bot token) before being returned.
func (r *runner) sendMessage(ctx context.Context, chatID int64, text string) error {
	payload, err := json.Marshal(struct {
		ChatID int64  `json:"chat_id"`
		Text   string `json:"text"`
	}{ChatID: chatID, Text: text})
	if err != nil {
		return fmt.Errorf("encoding sendMessage payload: %w", err)
	}
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", r.cfg.TelegramBaseURL, r.cfg.BotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building sendMessage request: %w", stripURL(err))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.cfg.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("sendMessage request: %w", stripURL(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sendMessage: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// stripURL unwraps a *url.Error to its underlying cause, dropping the
// request URL — Telegram request URLs embed the bot token, so the URL
// itself must never reach a log line or a wrapped error message.
func stripURL(err error) error {
	var uerr *url.Error
	if errors.As(err, &uerr) {
		return uerr.Err
	}
	return err
}

// nextBackoff doubles cur, capped at backoffCapFactor x base.
func nextBackoff(cur, base time.Duration) time.Duration {
	capped := base * backoffCapFactor
	next := cur * 2
	if next > capped {
		return capped
	}
	if next < base {
		return base
	}
	return next
}

// waitCtx sleeps for d or returns ctx.Err() if ctx is cancelled first. It
// never uses a bare time.Sleep so shutdown stays prompt.
func waitCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
