// Package rich реализует модель данных Telegram Bot API Rich Messages:
// декодирование входящих сообщений, преобразование в исходящий InputRichMessage,
// управление медиа-алиасами и валидацию перед отправкой.
package rich

import "encoding/json"

// MessageKind перечисляет взаимоисключающие источники содержимого InputRichMessage.
type MessageKind int

const (
	// KindEmpty — содержимое не задано (невалидный InputRichMessage).
	KindEmpty MessageKind = iota
	// KindMarkdown — содержимое в поле Markdown.
	KindMarkdown
	// KindHTML — содержимое в поле HTML.
	KindHTML
	// KindBlocks — содержимое в поле Blocks.
	KindBlocks
)

// RichMessage представляет входящее сообщение, созданное во встроенном редакторе Telegram.
//
//nolint:revive // имя сохраняет терминологию Telegram Bot API.
type RichMessage struct {
	Blocks []RichBlock `json:"blocks"`
	IsRTL  bool        `json:"is_rtl,omitempty"`
}

// InputRichMessage представляет исходящее Rich Message тело для sendRichMessage.
// Ровно один источник содержимого должен быть задан: Markdown, HTML или Blocks.
type InputRichMessage struct {
	Markdown            string                  `json:"markdown,omitempty"`
	HTML                string                  `json:"html,omitempty"`
	Blocks              []InputRichBlock        `json:"blocks,omitempty"`
	Media               []InputRichMessageMedia `json:"media,omitempty"`
	IsRTL               bool                    `json:"is_rtl,omitempty"`
	SkipEntityDetection bool                    `json:"skip_entity_detection,omitempty"`
}

// Kind возвращает заданный источник содержимого InputRichMessage.
func (m InputRichMessage) Kind() MessageKind {
	switch {
	case m.Markdown != "":
		return KindMarkdown
	case m.HTML != "":
		return KindHTML
	case len(m.Blocks) > 0:
		return KindBlocks
	default:
		return KindEmpty
	}
}

// InputRichMessageMedia описывает медиа-элемент, на который ссылается markdown/html через tg://.
type InputRichMessageMedia struct {
	ID    string     `json:"id"`
	Media InputMedia `json:"media"`
}

// IsZero сообщает, что InputMedia пуста и не должна сериализоваться в JSON.
func (m InputMedia) IsZero() bool {
	return m.Type == "" && m.Media == ""
}

// InputMedia описывает медиа, прикрепляемое к Rich Message.
// Конкретный тип определяется полем Type.
type InputMedia struct {
	Type                  string          `json:"type"`
	Media                 string          `json:"media"`
	Caption               string          `json:"caption,omitempty"`
	ParseMode             string          `json:"parse_mode,omitempty"`
	CaptionEntities       json.RawMessage `json:"caption_entities,omitempty"`
	ShowCaptionAboveMedia bool            `json:"show_caption_above_media,omitempty"`
	HasSpoiler            bool            `json:"has_spoiler,omitempty"`
	Thumbnail             string          `json:"thumbnail,omitempty"`
	Cover                 string          `json:"cover,omitempty"`
	StartTimestamp        int             `json:"start_timestamp,omitempty"`
	Width                 int             `json:"width,omitempty"`
	Height                int             `json:"height,omitempty"`
	Duration              int             `json:"duration,omitempty"`
	SupportsStreaming     bool            `json:"supports_streaming,omitempty"`
	Performer             string          `json:"performer,omitempty"`
	Title                 string          `json:"title,omitempty"`
}

// Допустимые значения InputMedia.Type внутри Rich Message.
const (
	MediaTypePhoto     = "photo"
	MediaTypeVideo     = "video"
	MediaTypeAnimation = "animation"
	MediaTypeAudio     = "audio"
	MediaTypeVoiceNote = "voice_note"
)

// NewPhotoMedia создаёт InputMediaPhoto из file_id.
func NewPhotoMedia(fileID string) InputMedia {
	return InputMedia{Type: MediaTypePhoto, Media: fileID}
}

// NewVideoMedia создаёт InputMediaVideo из file_id.
func NewVideoMedia(fileID string) InputMedia {
	return InputMedia{Type: MediaTypeVideo, Media: fileID}
}

// NewAnimationMedia создаёт InputMediaAnimation из file_id.
func NewAnimationMedia(fileID string) InputMedia {
	return InputMedia{Type: MediaTypeAnimation, Media: fileID}
}

// NewAudioMedia создаёт InputMediaAudio из file_id.
func NewAudioMedia(fileID string) InputMedia {
	return InputMedia{Type: MediaTypeAudio, Media: fileID}
}

// NewVoiceNoteMedia создаёт InputMediaVoiceNote из file_id.
func NewVoiceNoteMedia(fileID string) InputMedia {
	return InputMedia{Type: MediaTypeVoiceNote, Media: fileID}
}
