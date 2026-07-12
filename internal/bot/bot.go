// Package bot реализует логику обработки сообщений и callback-ов Telegram-бота.
package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/koteyye/tg-markdown-sender/internal/config"
	"github.com/koteyye/tg-markdown-sender/internal/drafts"
	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

const (
	pollTimeoutSeconds  = 50
	storageCheckTimeout = 15 * time.Second
)

// TelegramClient описывает методы Telegram Bot API, необходимые боту.
type TelegramClient interface {
	GetUpdates(ctx context.Context, offset int64, timeout int) ([]telegram.Update, error)
	DownloadFile(ctx context.Context, fileID string) ([]byte, error)
	SendRichMessage(ctx context.Context, chatID any, markdown string, replyMarkup *telegram.ReplyMarkup) (*telegram.Message, error)
	SendPhoto(ctx context.Context, chatID any, photoFileID, caption string, captionEntities []telegram.MessageEntity, replyMarkup *telegram.ReplyMarkup) (*telegram.Message, error)
	SendMessage(ctx context.Context, chatID any, text string, replyMarkup *telegram.ReplyMarkup) (*telegram.Message, error)
	AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string, showAlert bool) error
}

// MediaStore сохраняет фото и возвращает публичный HTTPS URL.
type MediaStore interface {
	UploadPhoto(ctx context.Context, data []byte) (string, error)
}

// MediaStoreChecker checks whether an image store is reachable with its configured credentials.
type MediaStoreChecker interface {
	Check(ctx context.Context) error
}

// Bot обрабатывает входящие сообщения и callback-запросы от Telegram.
type Bot struct {
	cfg    config.Config
	tg     TelegramClient
	store  drafts.Store
	logger *slog.Logger
	media  MediaStore
}

// Option изменяет настройки Bot при создании.
type Option func(*Bot)

// WithMediaStore включает публикацию фотографий как Rich Markdown media blocks.
func WithMediaStore(store MediaStore) Option {
	return func(b *Bot) {
		b.media = store
	}
}

// New создаёт новый экземпляр Bot с указанными зависимостями.
func New(cfg config.Config, tg TelegramClient, store drafts.Store, logger *slog.Logger, opts ...Option) *Bot {
	if logger == nil {
		logger = slog.Default()
	}

	b := &Bot{
		cfg:    cfg,
		tg:     tg,
		store:  store,
		logger: logger,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Run запускает бесконечный цикл получения и обработки обновлений из Telegram.
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

// HandleUpdate обрабатывает одно обновление из Telegram.
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

	if len(msg.Photo) > 0 {
		if b.media != nil {
			return b.handleRichPhotoMessage(ctx, chatID, msg)
		}
		return b.handlePhotoMessage(ctx, chatID, msg)
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		_, err := b.tg.SendMessage(ctx, chatID, "Пришли мне текстовый пост в формате Markdown или фото с Markdown-подписью.", nil)
		return err
	}

	switch commandName(text) {
	case "/start":
		_, err := b.tg.SendMessage(ctx, chatID, "Пришли мне пост в формате Markdown или фото с Markdown-подписью. Я покажу предпросмотр и предложу опубликовать его в канал.", nil)
		return err
	case "/infoimage":
		_, err := b.tg.SendMessage(ctx, chatID, imageInfoText, nil)
		return err
	case "/checkstorage":
		return b.checkStorage(ctx, chatID)
	}

	if len(msg.Entities) > 0 {
		b.logger.Debug("incoming message entities", "entities", entityDebugAttrs(msg.Entities))
	}

	markdown := restoreMarkdownEntities(msg.Text, msg.Entities)
	draft, err := b.store.Create(markdown)
	if err != nil {
		return fmt.Errorf("create draft: %w", err)
	}

	_, err = b.tg.SendRichMessage(ctx, chatID, draft.Markdown, previewKeyboard(draft.ID))
	if err != nil {
		_, notifyErr := b.tg.SendMessage(ctx, chatID, markdownErrorText(err), nil)
		if notifyErr != nil {
			return fmt.Errorf("preview failed: %w; notify failed: %w", err, notifyErr)
		}
		return nil
	}

	return nil
}

const imageInfoText = "Изображения в Rich Markdown:\n\n1. Отправь фото без подписи. Я верну строку ![](https://...) - вставь её в нужное место Markdown-поста.\n\n2. Для одного фото и текста отправь фото с Markdown-подписью. Поставь {{image}} отдельной строкой там, где нужна картинка; без {{image}} она будет добавлена в конец поста.\n\nПодпись к фото ограничена 1024 символами. Для длинного поста сначала отправь фото без подписи.\n\nПроверить доступ к хранилищу: /checkstorage"

func (b *Bot) checkStorage(ctx context.Context, chatID int64) error {
	if b.media == nil {
		_, err := b.tg.SendMessage(ctx, chatID, "Хранилище изображений не настроено.", nil)
		return err
	}

	checker, ok := b.media.(MediaStoreChecker)
	if !ok {
		_, err := b.tg.SendMessage(ctx, chatID, "Проверка хранилища недоступна в текущей конфигурации.", nil)
		return err
	}

	checkCtx, cancel := context.WithTimeout(ctx, storageCheckTimeout)
	defer cancel()
	if err := checker.Check(checkCtx); err != nil {
		b.logger.Error("image storage check failed", "error", err)
		_, notifyErr := b.tg.SendMessage(ctx, chatID, "Хранилище недоступно. Проверь endpoint, ключи service account и права на bucket.", nil)
		return notifyErr
	}

	_, err := b.tg.SendMessage(ctx, chatID, "Хранилище доступно: подключение и права service account на bucket подтверждены.", nil)
	return err
}

func (b *Bot) handleRichPhotoMessage(ctx context.Context, chatID int64, msg *telegram.Message) error {
	photo, ok := largestPhoto(msg.Photo)
	if !ok {
		_, err := b.tg.SendMessage(ctx, chatID, "Не удалось прочитать фото. Попробуй отправить изображение ещё раз.", nil)
		return err
	}

	data, err := b.tg.DownloadFile(ctx, photo.FileID)
	if err != nil {
		return b.sendPhotoProcessingError(ctx, chatID, "скачать", err)
	}
	imageURL, err := b.media.UploadPhoto(ctx, data)
	if err != nil {
		return b.sendPhotoProcessingError(ctx, chatID, "загрузить", err)
	}
	if strings.TrimSpace(msg.Caption) == "" {
		_, err := b.tg.SendMessage(ctx, chatID, "Фото загружено в R2. Вставь эту строку в Markdown-пост:\n\n"+markdownImageBlock(imageURL), nil)
		return err
	}

	markdown := markdownWithImage(restoreMarkdownEntities(msg.Caption, msg.CaptionEntities), imageURL)
	draft, err := b.store.Create(markdown)
	if err != nil {
		return fmt.Errorf("create rich photo draft: %w", err)
	}

	_, err = b.tg.SendRichMessage(ctx, chatID, draft.Markdown, previewKeyboard(draft.ID))
	if err != nil {
		_, notifyErr := b.tg.SendMessage(ctx, chatID, markdownErrorText(err), nil)
		if notifyErr != nil {
			return fmt.Errorf("rich photo preview failed: %w; notify failed: %w", err, notifyErr)
		}
	}

	return nil
}

func (b *Bot) sendPhotoProcessingError(ctx context.Context, chatID int64, action string, cause error) error {
	b.logger.Error("rich photo processing failed", "action", action, "error", cause)
	_, err := b.tg.SendMessage(ctx, chatID, "Не удалось "+action+" фото для публикации. Попробуй ещё раз позже.", nil)
	return err
}

func (b *Bot) handlePhotoMessage(ctx context.Context, chatID int64, msg *telegram.Message) error {
	photo, ok := largestPhoto(msg.Photo)
	if !ok {
		_, err := b.tg.SendMessage(ctx, chatID, "Не удалось прочитать фото. Попробуй отправить изображение ещё раз.", nil)
		return err
	}

	if len(msg.CaptionEntities) > 0 {
		b.logger.Debug("incoming photo caption entities", "entities", entityDebugAttrs(msg.CaptionEntities))
	}

	draft, err := b.store.CreatePhoto(photo.FileID, msg.Caption, msg.CaptionEntities)
	if err != nil {
		return fmt.Errorf("create photo draft: %w", err)
	}

	_, err = b.tg.SendPhoto(ctx, chatID, draft.PhotoFileID, draft.Caption, draft.CaptionEntities, previewKeyboard(draft.ID))
	if err != nil {
		_, notifyErr := b.tg.SendMessage(ctx, chatID, "Не удалось отправить предпросмотр фото: "+telegramErrorDescription(err), nil)
		if notifyErr != nil {
			return fmt.Errorf("photo preview failed: %w; notify failed: %w", err, notifyErr)
		}
		return nil
	}

	return nil
}

func (b *Bot) handleCallback(ctx context.Context, callback *telegram.CallbackQuery) error {
	if !IsAllowedUser(callback.From.ID, b.cfg.OwnerID) {
		b.answerCallback(ctx, callback, "Доступ запрещён.", true)
		if callback.Message != nil {
			_, err := b.tg.SendMessage(ctx, callback.Message.Chat.ID, "Доступ запрещён.", nil)
			return err
		}
		return nil
	}

	action, draftID, ok := parseCallbackData(callback.Data)
	if !ok {
		b.answerCallback(ctx, callback, "Неизвестное действие.", true)
		return nil
	}

	switch action {
	case "publish":
		return b.publishDraft(ctx, callback, draftID)
	case "cancel":
		return b.cancelDraft(ctx, callback, draftID)
	default:
		b.answerCallback(ctx, callback, "Неизвестное действие.", true)
		return nil
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

	if err := b.publishDraftContent(ctx, draft); err != nil {
		b.answerCallback(ctx, callback, "Не удалось опубликовать.", true)
		if callback.Message != nil {
			_, notifyErr := b.tg.SendMessage(ctx, callback.Message.Chat.ID, publishErrorText(err), nil)
			if notifyErr != nil {
				return fmt.Errorf("publish failed: %w; notify failed: %w", err, notifyErr)
			}
		}
		return nil
	}

	_, err := b.store.MarkPublished(draftID)
	if errors.Is(err, drafts.ErrAlreadyPublished) {
		b.answerCallback(ctx, callback, "Пост уже опубликован.", true)
		return nil
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

func (b *Bot) publishDraftContent(ctx context.Context, draft drafts.Draft) error {
	if draft.PhotoFileID != "" {
		_, err := b.tg.SendPhoto(ctx, b.cfg.ChannelID, draft.PhotoFileID, draft.Caption, draft.CaptionEntities, nil)
		return err
	}

	_, err := b.tg.SendRichMessage(ctx, b.cfg.ChannelID, draft.Markdown, nil)
	return err
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

func (b *Bot) answerCallback(ctx context.Context, callback *telegram.CallbackQuery, text string, showAlert bool) {
	if err := b.tg.AnswerCallbackQuery(ctx, callback.ID, text, showAlert); err != nil {
		b.logger.Error("answer callback query failed", "error", err)
	}
}

// IsAllowedUser проверяет, что userID совпадает с ownerID.
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
	if apiErr, ok := errors.AsType[*telegram.APIError](err); ok && apiErr.Description != "" {
		return "Telegram не смог обработать Markdown: " + apiErr.Description + "\n\nПришли исправленный вариант новым сообщением."
	}
	return "Не удалось отправить сообщение в Telegram. Пришли исправленный вариант новым сообщением или повтори попытку позже."
}

func publishErrorText(err error) string {
	description := telegramErrorDescription(err)
	if description == "" {
		return "Не удалось опубликовать из-за сетевой ошибки Telegram. Повтори попытку позже."
	}
	if strings.Contains(strings.ToLower(description), "chat not found") {
		return "Не удалось опубликовать: Telegram не нашёл целевой канал.\n\nПроверь TELEGRAM_CHANNEL_ID, username канала и что бот добавлен администратором канала с правом публикации."
	}
	return "Не удалось опубликовать: " + description
}

func telegramErrorDescription(err error) string {
	if apiErr, ok := errors.AsType[*telegram.APIError](err); ok && apiErr.Description != "" {
		return apiErr.Description
	}
	return ""
}

func largestPhoto(photos []telegram.PhotoSize) (telegram.PhotoSize, bool) {
	if len(photos) == 0 {
		return telegram.PhotoSize{}, false
	}

	best := photos[0]
	bestScore := photoScore(best)
	for _, photo := range photos[1:] {
		score := photoScore(photo)
		if score > bestScore {
			best = photo
			bestScore = score
		}
	}

	return best, best.FileID != ""
}

func photoScore(photo telegram.PhotoSize) int {
	if photo.FileSize > 0 {
		return photo.FileSize
	}
	return photo.Width * photo.Height
}

func markdownWithImage(markdown, imageURL string) string {
	const imagePlaceholder = "{{image}}"

	imageBlock := markdownImageBlock(imageURL)
	if strings.Contains(markdown, imagePlaceholder) {
		return strings.ReplaceAll(markdown, imagePlaceholder, imageBlock)
	}
	if strings.TrimSpace(markdown) == "" {
		return imageBlock
	}
	return markdown + "\n\n" + imageBlock
}

func markdownImageBlock(imageURL string) string {
	return "![](" + imageURL + ")"
}

func commandName(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	command, _, _ := strings.Cut(fields[0], "@")
	return command
}
