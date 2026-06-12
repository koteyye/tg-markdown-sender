package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/koteyye/tg-markdown-sender/internal/config"
	"github.com/koteyye/tg-markdown-sender/internal/drafts"
	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

const pollTimeoutSeconds = 50

type TelegramClient interface {
	GetUpdates(ctx context.Context, offset int64, timeout int) ([]telegram.Update, error)
	SendRichMessage(ctx context.Context, chatID any, markdown string, replyMarkup *telegram.ReplyMarkup) (*telegram.Message, error)
	SendMessage(ctx context.Context, chatID any, text string, replyMarkup *telegram.ReplyMarkup) (*telegram.Message, error)
	AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string, showAlert bool) error
}

type Bot struct {
	cfg    config.Config
	tg     TelegramClient
	store  drafts.Store
	logger *slog.Logger
}

func New(cfg config.Config, tg TelegramClient, store drafts.Store, logger *slog.Logger) *Bot {
	if logger == nil {
		logger = slog.Default()
	}

	return &Bot{
		cfg:    cfg,
		tg:     tg,
		store:  store,
		logger: logger,
	}
}

func (b *Bot) Run(ctx context.Context) error {
	var offset int64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		updates, err := b.tg.GetUpdates(ctx, offset, pollTimeoutSeconds)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			b.logger.Error("getUpdates failed", "error", err)
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			if err := b.HandleUpdate(ctx, update); err != nil {
				b.logger.Error("handle update failed", "update_id", update.UpdateID, "error", err)
			}
		}
	}
}

func (b *Bot) HandleUpdate(ctx context.Context, update telegram.Update) error {
	switch {
	case update.Message != nil:
		return b.handleMessage(ctx, update.Message)
	case update.CallbackQuery != nil:
		return b.handleCallback(ctx, update.CallbackQuery)
	default:
		return nil
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *telegram.Message) error {
	if msg.From == nil {
		return nil
	}
	chatID := msg.Chat.ID
	if !IsAllowedUser(msg.From.ID, b.cfg.OwnerID) {
		_, err := b.tg.SendMessage(ctx, chatID, "Доступ запрещён.", nil)
		return err
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		_, err := b.tg.SendMessage(ctx, chatID, "Пришли мне текстовый пост в формате Markdown.", nil)
		return err
	}

	if text == "/start" {
		_, err := b.tg.SendMessage(ctx, chatID, "Пришли мне пост в формате Markdown. Я покажу предпросмотр и предложу опубликовать его в канал.", nil)
		return err
	}

	draft, err := b.store.Create(msg.Text)
	if err != nil {
		return fmt.Errorf("create draft: %w", err)
	}

	_, err = b.tg.SendRichMessage(ctx, chatID, draft.Markdown, previewKeyboard(draft.ID))
	if err != nil {
		_, notifyErr := b.tg.SendMessage(ctx, chatID, markdownErrorText(err), nil)
		if notifyErr != nil {
			return fmt.Errorf("preview failed: %w; notify failed: %v", err, notifyErr)
		}
		return nil
	}

	return nil
}

func (b *Bot) handleCallback(ctx context.Context, callback *telegram.CallbackQuery) error {
	if !IsAllowedUser(callback.From.ID, b.cfg.OwnerID) {
		_ = b.tg.AnswerCallbackQuery(ctx, callback.ID, "Доступ запрещён.", true)
		if callback.Message != nil {
			_, err := b.tg.SendMessage(ctx, callback.Message.Chat.ID, "Доступ запрещён.", nil)
			return err
		}
		return nil
	}

	action, draftID, ok := parseCallbackData(callback.Data)
	if !ok {
		return b.tg.AnswerCallbackQuery(ctx, callback.ID, "Неизвестное действие.", true)
	}

	switch action {
	case "publish":
		return b.publishDraft(ctx, callback, draftID)
	case "cancel":
		return b.cancelDraft(ctx, callback, draftID)
	default:
		return b.tg.AnswerCallbackQuery(ctx, callback.ID, "Неизвестное действие.", true)
	}
}

func (b *Bot) publishDraft(ctx context.Context, callback *telegram.CallbackQuery, draftID string) error {
	draft, ok := b.store.Get(draftID)
	if !ok {
		return b.tg.AnswerCallbackQuery(ctx, callback.ID, "Черновик не найден.", true)
	}
	if draft.Published {
		return b.tg.AnswerCallbackQuery(ctx, callback.ID, "Пост уже опубликован.", true)
	}

	if _, err := b.tg.SendRichMessage(ctx, b.cfg.ChannelID, draft.Markdown, nil); err != nil {
		_ = b.tg.AnswerCallbackQuery(ctx, callback.ID, "Не удалось опубликовать.", true)
		if callback.Message != nil {
			_, notifyErr := b.tg.SendMessage(ctx, callback.Message.Chat.ID, markdownErrorText(err), nil)
			if notifyErr != nil {
				return fmt.Errorf("publish failed: %w; notify failed: %v", err, notifyErr)
			}
		}
		return nil
	}

	_, err := b.store.MarkPublished(draftID)
	if errors.Is(err, drafts.ErrAlreadyPublished) {
		return b.tg.AnswerCallbackQuery(ctx, callback.ID, "Пост уже опубликован.", true)
	}
	if err != nil {
		return err
	}

	if err := b.tg.AnswerCallbackQuery(ctx, callback.ID, "✅ Пост опубликован в канал.", false); err != nil {
		return err
	}
	if callback.Message != nil {
		_, err = b.tg.SendMessage(ctx, callback.Message.Chat.ID, "✅ Пост опубликован в канал.", nil)
		return err
	}
	return nil
}

func (b *Bot) cancelDraft(ctx context.Context, callback *telegram.CallbackQuery, draftID string) error {
	b.store.Delete(draftID)
	if err := b.tg.AnswerCallbackQuery(ctx, callback.ID, "Черновик удалён.", false); err != nil {
		return err
	}
	if callback.Message != nil {
		_, err := b.tg.SendMessage(ctx, callback.Message.Chat.ID, "Черновик удалён.", nil)
		return err
	}
	return nil
}

func IsAllowedUser(userID, ownerID int64) bool {
	return userID == ownerID
}

func previewKeyboard(draftID string) *telegram.ReplyMarkup {
	return &telegram.ReplyMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{{Text: "🚀 Опубликовать", CallbackData: "publish:" + draftID}},
			{{Text: "🗑 Отменить", CallbackData: "cancel:" + draftID}},
		},
	}
}

func parseCallbackData(data string) (action, draftID string, ok bool) {
	action, draftID, found := strings.Cut(data, ":")
	if !found || action == "" || draftID == "" {
		return "", "", false
	}
	return action, draftID, true
}

func markdownErrorText(err error) string {
	var apiErr *telegram.APIError
	if errors.As(err, &apiErr) && apiErr.Description != "" {
		return "Telegram не смог обработать Markdown: " + apiErr.Description + "\n\nПришли исправленный вариант новым сообщением."
	}
	return "Не удалось отправить сообщение в Telegram. Пришли исправленный вариант новым сообщением или повтори попытку позже."
}
