package rich

import (
	"encoding/json"
	"fmt"
)

// RichText — рекурсивное объединение: строка | массив RichText | типизированный узел.
// Полиморфизм реализован через пользовательские MarshalJSON/UnmarshalJSON.
//
//nolint:revive // имя сохраняет терминологию Telegram Bot API.
type RichText struct {
	// String содержит текст, когда узел является обычной строкой.
	String string
	// Items содержит дочерние узлы, когда узел является массивом RichText.
	Items []RichText
	// Node содержит типизированный узел форматирования.
	Node *RichTextNode
}

// IsZero сообщает, что RichText пуст и не должен сериализоваться в JSON.
func (t RichText) IsZero() bool {
	return t.String == "" && len(t.Items) == 0 && t.Node == nil
}

// RichTextNode описывает типизированный узел RichText (bold, italic, custom_emoji и т.д.).
//
//nolint:revive // имя сохраняет терминологию Telegram Bot API.
type RichTextNode struct {
	Type string `json:"type"`

	// text — рекурсивный дочерний RichText для узлов-обёрток.
	Text *RichText `json:"text,omitempty"`

	// Точечные поля типизированных узлов.
	CustomEmojiID   string `json:"custom_emoji_id,omitempty"`
	AlternativeText string `json:"alternative_text,omitempty"`
	Expression      string `json:"expression,omitempty"`
	Name            string `json:"name,omitempty"`
	AnchorName      string `json:"anchor_name,omitempty"`
	ReferenceName   string `json:"reference_name,omitempty"`
	URL             string `json:"url,omitempty"`
	EmailAddress    string `json:"email_address,omitempty"`
	PhoneNumber     string `json:"phone_number,omitempty"`
	BankCardNumber  string `json:"bank_card_number,omitempty"`
	Username        string `json:"username,omitempty"`
	Hashtag         string `json:"hashtag,omitempty"`
	Cashtag         string `json:"cashtag,omitempty"`
	BotCommand      string `json:"bot_command,omitempty"`
	UnixTime        int    `json:"unix_time,omitempty"`
	DateTimeFormat  string `json:"date_time_format,omitempty"`
	User            *User  `json:"user,omitempty"`
}

// User — минимальное представление пользователя Telegram для RichTextTextMention.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Дискриминаторы типов RichText.
const (
	TextPlain                  = "" // обычная строка не имеет дискриминатора
	TextBold                   = "bold"
	TextItalic                 = "italic"
	TextUnderline              = "underline"
	TextStrikethrough          = "strikethrough"
	TextSpoiler                = "spoiler"
	TextSubscript              = "subscript"
	TextSuperscript            = "superscript"
	TextMarked                 = "marked"
	TextCode                   = "code"
	TextCustomEmoji            = "custom_emoji"
	TextMathematicalExpression = "mathematical_expression"
	TextURL                    = "url"
	TextEmailAddress           = "email_address"
	TextPhoneNumber            = "phone_number"
	TextBankCardNumber         = "bank_card_number"
	TextMention                = "mention"
	TextTextMention            = "text_mention"
	TextHashtag                = "hashtag"
	TextCashtag                = "cashtag"
	TextBotCommand             = "bot_command"
	TextAnchor                 = "anchor"
	TextAnchorLink             = "anchor_link"
	TextReference              = "reference"
	TextReferenceLink          = "reference_link"
	TextDateTime               = "date_time"
)

// IsString сообщает, что узел является обычной строкой.
func (t RichText) IsString() bool { return t.Node == nil && t.Items == nil }

// MarshalJSON кодирует RichText как строку, массив или объект.
func (t RichText) MarshalJSON() ([]byte, error) {
	switch {
	case t.Node != nil:
		return json.Marshal(t.Node)
	case t.Items != nil:
		return json.Marshal(t.Items)
	default:
		return json.Marshal(t.String)
	}
}

// UnmarshalJSON декодирует RichText из строки, массива или объекта.
func (t *RichText) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	first := data[0]
	switch first {
	case '"':
		return json.Unmarshal(data, &t.String)
	case '[':
		t.Items = nil
		if err := json.Unmarshal(data, &t.Items); err != nil {
			return err
		}
		return nil
	case '{':
		node := &RichTextNode{}
		if err := json.Unmarshal(data, node); err != nil {
			return err
		}
		t.Node = node
		return nil
	default:
		return fmt.Errorf("richtext: unexpected token %q", string(first))
	}
}

// RichBlockCaption описывает подпись блока медиа.
//
//nolint:revive // имя сохраняет терминологию Telegram Bot API.
type RichBlockCaption struct {
	Text   RichText `json:"text"`
	Credit RichText `json:"credit,omitzero"`
}

// RichBlockTableCell описывает ячейку таблицы Rich Message.
//
//nolint:revive // имя сохраняет терминологию Telegram Bot API.
type RichBlockTableCell struct {
	Text     RichText `json:"text,omitzero"`
	IsHeader bool     `json:"is_header,omitempty"`
	Colspan  int      `json:"colspan,omitempty"`
	Rowspan  int      `json:"rowspan,omitempty"`
	Align    string   `json:"align"`
	Valign   string   `json:"valign"`
}

// BlockType содержит дискриминаторы типов RichBlock / InputRichBlock.
const (
	BlockParagraph              = "paragraph"
	BlockHeading                = "heading"
	BlockPreformatted           = "pre"
	BlockFooter                 = "footer"
	BlockDivider                = "divider"
	BlockMathematicalExpression = "mathematical_expression"
	BlockAnchor                 = "anchor"
	BlockList                   = "list"
	BlockBlockquote             = "blockquote"
	BlockPullquote              = "pullquote"
	BlockCollage                = "collage"
	BlockSlideshow              = "slideshow"
	BlockTable                  = "table"
	BlockDetails                = "details"
	BlockMap                    = "map"
	BlockAnimation              = "animation"
	BlockAudio                  = "audio"
	BlockPhoto                  = "photo"
	BlockVideo                  = "video"
	BlockVoiceNote              = "voice_note"
	BlockThinking               = "thinking"
)
