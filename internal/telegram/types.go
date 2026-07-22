package telegram

import "github.com/koteyye/tg-markdown-sender/internal/rich"

//nolint:revive // группа алиасов позволяет остальному коду зависеть только от пакета telegram.
type (
	InputRichMessage      = rich.InputRichMessage
	InputRichMessageMedia = rich.InputRichMessageMedia
	InputMedia            = rich.InputMedia
)

// Update представляет одно входящее обновление из Telegram.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// Message представляет сообщение, полученное от Telegram.
type Message struct {
	MessageID       int64             `json:"message_id"`
	From            *User             `json:"from,omitempty"`
	Chat            Chat              `json:"chat"`
	Text            string            `json:"text,omitempty"`
	Entities        []MessageEntity   `json:"entities,omitempty"`
	Caption         string            `json:"caption,omitempty"`
	CaptionEntities []MessageEntity   `json:"caption_entities,omitempty"`
	Photo           []PhotoSize       `json:"photo,omitempty"`
	RichMessage     *rich.RichMessage `json:"rich_message,omitempty"`
}

// MessageEntity описывает entity форматирования внутри сообщения.
type MessageEntity struct {
	Type          string `json:"type"`
	Offset        int    `json:"offset"`
	Length        int    `json:"length"`
	Language      string `json:"language,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
}

// PhotoSize описывает один размер фотографии.
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id,omitempty"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size,omitempty"`
}

// User представляет информацию о пользователе Telegram.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// BotInfo содержит информацию о текущем боте.
type BotInfo struct {
	ID                      int64  `json:"id"`
	IsBot                   bool   `json:"is_bot"`
	FirstName               string `json:"first_name,omitempty"`
	Username                string `json:"username,omitempty"`
	CanJoinGroups           bool   `json:"can_join_groups,omitempty"`
	CanReadAllGroupMessages bool   `json:"can_read_all_group_messages,omitempty"`
	SupportsInlineQueries   bool   `json:"supports_inline_queries,omitempty"`
}

// Chat представляет чат, в котором было отправлено сообщение.
type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type,omitempty"`
	Username string `json:"username,omitempty"`
}

// CallbackQuery представляет callback-запрос от inline-кнопки.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

// ReplyMarkup описывает inline-клавиатуру.
type ReplyMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton описывает одну inline-кнопку.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// SendRichMessageRequest — тело запроса sendRichMessage.
type SendRichMessageRequest struct {
	ChatID      any                   `json:"chat_id"`
	RichMessage rich.InputRichMessage `json:"rich_message"`
	ReplyMarkup *ReplyMarkup          `json:"reply_markup,omitempty"`
}

// SendMessageRequest — тело запроса sendMessage.
type SendMessageRequest struct {
	ChatID      any          `json:"chat_id"`
	Text        string       `json:"text"`
	ReplyMarkup *ReplyMarkup `json:"reply_markup,omitempty"`
}

// AnswerCallbackQueryRequest — тело запроса answerCallbackQuery.
type AnswerCallbackQueryRequest struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}
