package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/wjffsx/miniclaw_go/internal/bus"
)

const (
	defaultAPIURL       = "https://api.telegram.org/bot%s/%s"
	maxMessageLength    = 4096
	defaultPollTimeout  = 30
	defaultPollInterval = 3 * time.Second
)

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      *Chat  `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	Title     string `json:"title,omitempty"`
}

type SendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type APIResponse struct {
	OK     bool        `json:"ok"`
	Result interface{} `json:"result,omitempty"`
	Error  *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Bot struct {
	token        string
	apiURL       string
	updateOffset int64
	httpClient   *http.Client
	messageBus   bus.MessageBus
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
	enabled      bool
	started      bool
	pollTimeout  int
	pollInterval time.Duration
}

type Config struct {
	Token       string
	PollTimeout int
}

func NewBot(cfg *Config, messageBus bus.MessageBus, ctx context.Context) *Bot {
	botCtx, cancel := context.WithCancel(ctx)

	pollTimeout := defaultPollTimeout
	if cfg.PollTimeout > 0 {
		pollTimeout = cfg.PollTimeout
	}

	return &Bot{
		token:        cfg.Token,
		apiURL:       fmt.Sprintf(defaultAPIURL, cfg.Token, "%s"),
		updateOffset: 0,
		httpClient: &http.Client{
			Timeout: time.Duration(pollTimeout+5) * time.Second,
		},
		messageBus: messageBus,
		ctx:        botCtx,
		cancel:     cancel,
		enabled:    cfg.Token != "",
	}
}

func (b *Bot) Start() error {
	if !b.enabled {
		log.Println("Telegram bot is disabled (no token configured)")
		return nil
	}

	b.mu.Lock()
	if b.started {
		b.mu.Unlock()
		return fmt.Errorf("bot already started")
	}
	b.started = true
	b.mu.Unlock()

	log.Println("Starting Telegram bot...")

	b.wg.Add(1)
	go b.pollUpdates()

	return nil
}

func (b *Bot) Stop() error {
	b.mu.Lock()
	if !b.started {
		b.mu.Unlock()
		return fmt.Errorf("bot not started")
	}
	b.started = false
	b.mu.Unlock()

	log.Println("Stopping Telegram bot...")
	b.cancel()
	b.wg.Wait()
	return nil
}

func (b *Bot) pollUpdates() {
	defer b.wg.Done()

	log.Println("Telegram polling task started")

	for {
		select {
		case <-b.ctx.Done():
			log.Println("Telegram polling task stopped")
			return
		default:
			if err := b.getUpdates(); err != nil {
				log.Printf("Error getting updates: %v", err)
				time.Sleep(defaultPollInterval)
			}
		}
	}
}

func (b *Bot) getUpdates() error {
	params := url.Values{}
	params.Add("offset", strconv.FormatInt(b.updateOffset, 10))
	params.Add("timeout", strconv.Itoa(defaultPollTimeout))

	apiURL := fmt.Sprintf(b.apiURL, b.token, "getUpdates?"+params.Encode())

	resp, err := b.httpClient.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to get updates: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.OK {
		if apiResp.Error != nil {
			return fmt.Errorf("API error: %s", apiResp.Error.Message)
		}
		return fmt.Errorf("API returned not OK")
	}

	updates, ok := apiResp.Result.([]interface{})
	if !ok {
		return fmt.Errorf("invalid result format")
	}

	for _, update := range updates {
		updateMap, ok := update.(map[string]interface{})
		if !ok {
			continue
		}

		updateID, ok := updateMap["update_id"].(float64)
		if !ok {
			continue
		}

		b.mu.Lock()
		if int64(updateID) >= b.updateOffset {
			b.updateOffset = int64(updateID) + 1
		}
		b.mu.Unlock()

		messageMap, ok := updateMap["message"].(map[string]interface{})
		if !ok {
			continue
		}

		text, ok := messageMap["text"].(string)
		if !ok || text == "" {
			continue
		}

		chatMap, ok := messageMap["chat"].(map[string]interface{})
		if !ok {
			continue
		}

		chatID, ok := chatMap["id"].(float64)
		if !ok {
			continue
		}

		chatIDStr := fmt.Sprintf("%.0f", chatID)
		log.Printf("Message from chat %s: %.40s...", chatIDStr, text)

		msg := &bus.Message{
			ID:      fmt.Sprintf("telegram-%d-%.0f", time.Now().UnixNano(), updateID),
			Channel: bus.ChannelTelegram,
			ChatID:  chatIDStr,
			Content: text,
		}

		if err := b.messageBus.Publish(b.ctx, bus.ChannelTelegram, msg); err != nil {
			log.Printf("Failed to publish message to bus: %v", err)
		}
	}

	return nil
}

func (b *Bot) SendMessage(chatID, text string) error {
	if !b.enabled {
		return fmt.Errorf("telegram bot is disabled")
	}

	textLen := len(text)
	offset := 0

	for offset < textLen {
		chunk := textLen - offset
		if chunk > maxMessageLength {
			chunk = maxMessageLength
		}

		segment := text[offset : offset+chunk]

		req := SendMessageRequest{
			ChatID:    chatID,
			Text:      segment,
			ParseMode: "Markdown",
		}

		if err := b.sendMessageRequest(req); err != nil {
			log.Printf("Markdown send failed, retrying plain: %v", err)
			req.ParseMode = ""
			if err := b.sendMessageRequest(req); err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}
		}

		offset += chunk
	}

	return nil
}

func (b *Bot) sendMessageRequest(req SendMessageRequest) error {
	apiURL := fmt.Sprintf(b.apiURL, b.token, "sendMessage")

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := b.httpClient.Post(apiURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.OK {
		if apiResp.Error != nil {
			return fmt.Errorf("API error: %s", apiResp.Error.Message)
		}
		return fmt.Errorf("API returned not OK")
	}

	return nil
}

func (b *Bot) SetToken(token string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.token = token
	b.enabled = token != ""
	log.Printf("Telegram bot token updated (len=%d)", len(token))
}

func (b *Bot) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.started
}

func (b *Bot) GetToken() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.token
}

func (b *Bot) GetAPIURL() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.apiURL
}

func (b *Bot) SetPollTimeout(timeout int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pollTimeout = timeout
}

func (b *Bot) SetPollInterval(interval int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pollInterval = time.Duration(interval) * time.Second
}

func (b *Bot) GetUpdates(offset int64) ([]Update, error) {
	params := url.Values{}
	params.Add("offset", strconv.FormatInt(offset, 10))
	params.Add("timeout", strconv.Itoa(defaultPollTimeout))

	apiURL := fmt.Sprintf(b.apiURL, b.token, "getUpdates?"+params.Encode())

	resp, err := b.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get updates: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.OK {
		if apiResp.Error != nil {
			return nil, fmt.Errorf("API error: %s", apiResp.Error.Message)
		}
		return nil, fmt.Errorf("API returned not OK")
	}

	updates, ok := apiResp.Result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid result format")
	}

	result := make([]Update, 0, len(updates))
	for _, update := range updates {
		updateMap, ok := update.(map[string]interface{})
		if !ok {
			continue
		}

		updateID, ok := updateMap["update_id"].(float64)
		if !ok {
			continue
		}

		result = append(result, Update{
			UpdateID: int64(updateID),
		})
	}

	return result, nil
}

func (b *Bot) SetWebhook(webhookURL string) error {
	if !b.enabled {
		return fmt.Errorf("telegram bot is disabled")
	}

	params := url.Values{}
	params.Add("url", webhookURL)

	apiURL := fmt.Sprintf(b.apiURL, b.token, "setWebhook?"+params.Encode())

	resp, err := b.httpClient.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.OK {
		if apiResp.Error != nil {
			return fmt.Errorf("API error: %s", apiResp.Error.Message)
		}
		return fmt.Errorf("API returned not OK")
	}

	return nil
}

func (b *Bot) DeleteWebhook() error {
	if !b.enabled {
		return fmt.Errorf("telegram bot is disabled")
	}

	apiURL := fmt.Sprintf(b.apiURL, b.token, "deleteWebhook")

	resp, err := b.httpClient.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.OK {
		if apiResp.Error != nil {
			return fmt.Errorf("API error: %s", apiResp.Error.Message)
		}
		return fmt.Errorf("API returned not OK")
	}

	return nil
}

func (b *Bot) GetMe() (*User, error) {
	if !b.enabled {
		return nil, fmt.Errorf("telegram bot is disabled")
	}

	apiURL := fmt.Sprintf(b.apiURL, b.token, "getMe")

	resp, err := b.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get me: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.OK {
		if apiResp.Error != nil {
			return nil, fmt.Errorf("API error: %s", apiResp.Error.Message)
		}
		return nil, fmt.Errorf("API returned not OK")
	}

	result, ok := apiResp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid result format")
	}

	user := &User{}
	if id, ok := result["id"].(float64); ok {
		user.ID = int64(id)
	}
	if firstName, ok := result["first_name"].(string); ok {
		user.FirstName = firstName
	}
	if lastName, ok := result["last_name"].(string); ok {
		user.LastName = lastName
	}
	if username, ok := result["username"].(string); ok {
		user.Username = username
	}

	return user, nil
}

func (b *Bot) handleUpdate(update *Update) {
	if update.Message == nil {
		return
	}

	if b.messageBus == nil {
		return
	}

	msg := &bus.Message{
		ID:      fmt.Sprintf("telegram-%d", update.UpdateID),
		Channel: bus.ChannelTelegram,
		ChatID:  fmt.Sprintf("%.0f", float64(update.Message.Chat.ID)),
		Content: update.Message.Text,
	}

	if err := b.messageBus.Publish(b.ctx, bus.ChannelTelegram, msg); err != nil {
		log.Printf("Failed to publish message to bus: %v", err)
	}
}

func (b *Bot) poll() {
	b.pollUpdates()
}
