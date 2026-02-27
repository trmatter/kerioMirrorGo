package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"kerio-mirror-go/config"
	"kerio-mirror-go/utils"
)

const (
	maxRetries    = 3
	retryDelay    = 3 * time.Second
)

// Notifier sends notifications to a Telegram chat via Bot API.
type Notifier struct {
	cfg *config.Config
}

// New creates a Notifier using the provided config.
func New(cfg *config.Config) *Notifier {
	return &Notifier{cfg: cfg}
}

// Enabled returns true if bot token and chat ID are configured.
func (n *Notifier) Enabled() bool {
	return n.cfg.TelegramBotToken != "" && n.cfg.TelegramChatID != ""
}

// Send posts a message to the configured Telegram chat.
// HTML parse mode is used; use <b>, <i>, <code> tags as needed.
// Retries up to maxRetries times on transient network errors (EOF, connection reset).
func (n *Notifier) Send(text string) error {
	if !n.Enabled() {
		return nil
	}

	// Build payload once; re-create reader on each attempt.
	payload := map[string]interface{}{
		"chat_id":    n.cfg.TelegramChatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal error: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.cfg.TelegramBotToken)

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Create a fresh client per attempt to avoid reusing stale connections.
		client, err := utils.CreateHTTPClient(n.cfg.ProxyURL, 15*time.Second)
		if err != nil {
			client = &http.Client{Timeout: 15 * time.Second}
		}

		resp, err := client.Post(apiURL, "application/json", bytes.NewReader(body))
		if err != nil {
			lastErr = fmt.Errorf("telegram: send error (attempt %d/%d): %w", attempt, maxRetries, err)
			if isTransient(err) && attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return lastErr
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body) // drain body so connection can be reused

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("telegram: API returned status %d", resp.StatusCode)
		}
		return nil
	}
	return lastErr
}

// isTransient returns true for network errors that are worth retrying.
func isTransient(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	// Unwrap to catch wrapped io.EOF from net/http
	var urlErr interface{ Unwrap() error }
	if errors.As(err, &urlErr) {
		inner := urlErr.Unwrap()
		if inner != nil && (errors.Is(inner, io.EOF) || errors.Is(inner, io.ErrUnexpectedEOF)) {
			return true
		}
	}
	return false
}

// NotifyError sends a message if TelegramNotifyOnError is enabled.
func (n *Notifier) NotifyError(text string) error {
	if !n.cfg.TelegramNotifyOnError {
		return nil
	}
	return n.Send(text)
}

// NotifySuccess sends a message if TelegramNotifyOnSuccess is enabled.
func (n *Notifier) NotifySuccess(text string) error {
	if !n.cfg.TelegramNotifyOnSuccess {
		return nil
	}
	return n.Send(text)
}

// NotifyStart sends a message if TelegramNotifyOnStart is enabled.
func (n *Notifier) NotifyStart(text string) error {
	if !n.cfg.TelegramNotifyOnStart {
		return nil
	}
	return n.Send(text)
}
