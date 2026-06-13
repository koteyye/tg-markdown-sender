// Package telegram реализует HTTP-клиент для Telegram Bot API.
package telegram

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
	"strings"
	"time"
)

const defaultBaseURL = "https://api.telegram.org"

// Client выполняет запросы к Telegram Bot API.
type Client struct {
	token      string
	httpClient *http.Client
	logger     *slog.Logger
	baseURL    string
}

// Option изменяет настройки Client при создании.
type Option func(*Client)

// WithBaseURL задаёт альтернативный базовый URL для Telegram API.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// NewClient создаёт новый HTTP-клиент для Telegram Bot API.
func NewClient(token string, httpClient *http.Client, logger *slog.Logger, opts ...Option) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	c := &Client{
		token:      token,
		httpClient: httpClient,
		logger:     logger,
		baseURL:    defaultBaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// GetMe возвращает информацию о текущем боте.
func (c *Client) GetMe(ctx context.Context) (*BotInfo, error) {
	var botInfo BotInfo
	if err := c.do(ctx, http.MethodGet, "getMe", nil, nil, &botInfo); err != nil {
		return nil, err
	}
	return &botInfo, nil
}

// GetUpdates получает список входящих обновлений из Telegram.
func (c *Client) GetUpdates(ctx context.Context, offset int64, timeout int) ([]Update, error) {
	values := url.Values{}
	if offset > 0 {
		values.Set("offset", strconv.FormatInt(offset, 10))
	}
	values.Set("timeout", strconv.Itoa(timeout))
	values.Set("allowed_updates", `["message","callback_query"]`)

	var updates []Update
	err := c.do(ctx, http.MethodGet, "getUpdates", values, nil, &updates)
	return updates, err
}

// SendRichMessage отправляет форматированное Markdown-сообщение.
func (c *Client) SendRichMessage(ctx context.Context, chatID any, markdown string, replyMarkup *ReplyMarkup) (*Message, error) {
	var message Message
	body := SendRichMessageRequest{
		ChatID:      chatID,
		RichMessage: RichMessage{Markdown: markdown},
		ReplyMarkup: replyMarkup,
	}
	if err := c.do(ctx, http.MethodPost, "sendRichMessage", nil, body, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// SendMessage отправляет обычное текстовое сообщение.
func (c *Client) SendMessage(ctx context.Context, chatID any, text string, replyMarkup *ReplyMarkup) (*Message, error) {
	var message Message
	body := SendMessageRequest{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: replyMarkup,
	}
	if err := c.do(ctx, http.MethodPost, "sendMessage", nil, body, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// SendPhoto отправляет фото с подписью.
func (c *Client) SendPhoto(ctx context.Context, chatID any, photoFileID, caption string, captionEntities []MessageEntity, replyMarkup *ReplyMarkup) (*Message, error) {
	var message Message
	body := SendPhotoRequest{
		ChatID:          chatID,
		Photo:           photoFileID,
		Caption:         caption,
		CaptionEntities: captionEntities,
		ReplyMarkup:     replyMarkup,
	}
	if err := c.do(ctx, http.MethodPost, "sendPhoto", nil, body, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// AnswerCallbackQuery отвечает на callback-запрос inline-кнопки.
func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string, showAlert bool) error {
	body := AnswerCallbackQueryRequest{
		CallbackQueryID: callbackQueryID,
		Text:            text,
		ShowAlert:       showAlert,
	}
	return c.do(ctx, http.MethodPost, "answerCallbackQuery", nil, body, nil)
}

func (c *Client) do(ctx context.Context, httpMethod, apiMethod string, query url.Values, body any, result any) error {
	var encodedBody []byte
	var err error
	if body != nil {
		encodedBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal %s request: %w", apiMethod, err)
		}
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, httpMethod, c.methodURL(apiMethod, query), bytes.NewReader(encodedBody))
		if err != nil {
			return err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = c.networkError(apiMethod, err)
			if attempt < 3 {
				sleepBeforeRetry(ctx, attempt)
				continue
			}
			return lastErr
		}

		err = c.decodeResponse(resp, apiMethod, result)
		if err == nil {
			return nil
		}

		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 500 && attempt < 3 {
			lastErr = err
			_ = resp.Body.Close()
			sleepBeforeRetry(ctx, attempt)
			continue
		}
		return err
	}

	return lastErr
}

func (c *Client) networkError(apiMethod string, err error) error {
	description := strings.ReplaceAll(err.Error(), c.token, "<redacted>")
	return fmt.Errorf("%s network error after retries: %s", apiMethod, description)
}

func (c *Client) decodeResponse(resp *http.Response, apiMethod string, result any) error {
	defer func() {
		_ = resp.Body.Close()
	}()

	var envelope struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		ErrorCode   int             `json:"error_code,omitempty"`
		Description string          `json:"description,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		c.logger.Error("telegram api error", "method", apiMethod, "http_status", resp.StatusCode, "description", "invalid json response")
		return fmt.Errorf("%s invalid response: %w", apiMethod, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !envelope.OK {
		apiErr := &APIError{
			Method:      apiMethod,
			HTTPStatus:  resp.StatusCode,
			Code:        envelope.ErrorCode,
			Description: envelope.Description,
		}
		c.logger.Error("telegram api error", "method", apiMethod, "http_status", resp.StatusCode, "description", envelope.Description)
		return apiErr
	}

	if result == nil || len(envelope.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("%s decode result: %w", apiMethod, err)
	}

	return nil
}

func (c *Client) methodURL(apiMethod string, query url.Values) string {
	u := fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.token, apiMethod)
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return u
}

func sleepBeforeRetry(ctx context.Context, attempt int) {
	delay := time.Duration(attempt) * 250 * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

// APIError представляет ошибку, возвращённую Telegram Bot API.
type APIError struct {
	Method      string
	HTTPStatus  int
	Code        int
	Description string
}

func (e *APIError) Error() string {
	if e.Description == "" {
		return fmt.Sprintf("%s failed with HTTP %d", e.Method, e.HTTPStatus)
	}
	return fmt.Sprintf("%s failed with HTTP %d: %s", e.Method, e.HTTPStatus, e.Description)
}
