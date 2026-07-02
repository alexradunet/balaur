package cli

import (
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/alexradunet/balaur/internal/bridge"
)

// consentNotice is printed once to stderr at startup and included in
// --help, grounded in docs/superpowers/specs/2026-07-02-messenger-bridge-design.md
// §4 (spike §3.2 of the Phase-0 gateway spike).
const consentNotice = "EXPERIMENTAL: message text you send to the bot, and Balaur's replies, " +
	"transit Telegram's servers. Do not use this bridge for content that must never leave your infrastructure."

// bridgeCmd mounts the EXPERIMENTAL reference messenger bridge
// (docs/superpowers/specs/2026-07-02-messenger-bridge-design.md, plan 256
// spike). Unlike every other command in this package, telegramBridgeCmd's
// RunE is NOT wrapped by run() (cli.go) — that wrapper applies pending
// migrations and prints exactly one JSON envelope, both wrong for a
// long-running, deliberately DB-free process whose output is a structured
// log stream, not a single result. Its own RunE calls failJSON directly on
// error so the process still exits non-zero through cli.ExitCode()
// (PocketBase's Execute discards RunE errors — see the comment on
// exitCode above).
func bridgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bridge",
		Short: "EXPERIMENTAL: owner-run reference bridges relaying a messenger to this box's Balaur",
	}
	cmd.AddCommand(telegramBridgeCmd())
	return cmd
}

func telegramBridgeCmd() *cobra.Command {
	var balaurURL string
	var allowChat []int64
	var pollTimeout time.Duration

	cmd := &cobra.Command{
		Use:   "telegram",
		Short: "EXPERIMENTAL: relay an allowlisted Telegram chat to this box's Balaur",
		Long: consentNotice + "\n\n" +
			"Reads two required secrets from the environment (never from flags, so " +
			"they do not leak into `ps` output or shell history):\n" +
			"  BALAUR_TELEGRAM_BOT_TOKEN  the Telegram bot token\n" +
			"  BALAUR_MESSENGER_TOKEN     must match owner_settings.messenger_token\n\n" +
			"Design note: docs/superpowers/specs/2026-07-02-messenger-bridge-design.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			botToken := os.Getenv("BALAUR_TELEGRAM_BOT_TOKEN")
			if botToken == "" {
				// Never the value — only the env var name is reported.
				return failJSON(cmd, errors.New("BALAUR_TELEGRAM_BOT_TOKEN is required (set it in the environment, not as a flag)"))
			}
			messengerToken := os.Getenv("BALAUR_MESSENGER_TOKEN")
			if messengerToken == "" {
				return failJSON(cmd, errors.New("BALAUR_MESSENGER_TOKEN is required (set it in the environment, not as a flag)"))
			}
			if len(allowChat) == 0 {
				return failJSON(cmd, errors.New("--allow-chat is required; refusing to start without a sender allowlist"))
			}

			logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), nil))
			logger.Info(consentNotice)

			cfg := bridge.Config{
				BotToken:       botToken,
				MessengerToken: messengerToken,
				BalaurURL:      balaurURL,
				AllowedChatIDs: allowChat,
				PollTimeout:    pollTimeout,
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			if err := bridge.Run(ctx, cfg, logger); err != nil {
				return failJSON(cmd, err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&balaurURL, "balaur-url", "http://127.0.0.1:8090", "this box's Balaur base URL")
	cmd.Flags().Int64SliceVar(&allowChat, "allow-chat", nil, "allowlisted Telegram chat id (repeatable); required — the bridge refuses to start empty")
	cmd.Flags().DurationVar(&pollTimeout, "poll-timeout", 50*time.Second, "getUpdates long-poll timeout")
	return cmd
}
