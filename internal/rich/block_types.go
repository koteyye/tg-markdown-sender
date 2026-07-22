package rich

import (
	"encoding/json"
	"errors"
)

// RichBlock описывает входящий блок Rich Message.
// Поле caption хранится как RawMessage, т.к. тип зависит от блока:
// RichBlockCaption для медиа/коллажа/слайд-шоу/карты, RichText для таблицы.
//
//nolint:revive // имя сохраняет терминологию Telegram Bot API.
type RichBlock struct {
	Type       string                 `json:"type"`
	Text       RichText               `json:"text,omitzero"`
	Size       int                    `json:"size,omitempty"`
	Language   string                 `json:"language,omitempty"`
	Expression string                 `json:"expression,omitempty"`
	Name       string                 `json:"name,omitempty"`
	Items      []RichBlockListItem    `json:"items,omitempty"`
	Blocks     []RichBlock            `json:"blocks,omitempty"`
	Credit     RichText               `json:"credit,omitzero"`
	Cells      [][]RichBlockTableCell `json:"cells,omitempty"`
	IsBordered bool                   `json:"is_bordered,omitempty"`
	IsStriped  bool                   `json:"is_striped,omitempty"`
	IsOpen     bool                   `json:"is_open,omitempty"`
	Summary    RichText               `json:"summary,omitzero"`
	Location   *Location              `json:"location,omitempty"`
	Zoom       int                    `json:"zoom,omitempty"`
	Width      int                    `json:"width,omitempty"`
	Height     int                    `json:"height,omitempty"`
	Photo      []PhotoSize            `json:"photo,omitempty"`
	Video      *Video                 `json:"video,omitempty"`
	Animation  *Animation             `json:"animation,omitempty"`
	Audio      *Audio                 `json:"audio,omitempty"`
	VoiceNote  *Voice                 `json:"voice_note,omitempty"`
	HasSpoiler bool                   `json:"has_spoiler,omitempty"`

	// rawCaption — декодированный JSON поля caption; интерпретируется по Type.
	rawCaption json.RawMessage
}

// CaptionValue возвращает подпись типа RichBlockCaption (для медиа, коллажа, слайд-шоу, карты).
func (b RichBlock) CaptionValue() (*RichBlockCaption, bool) {
	if len(b.rawCaption) == 0 {
		return nil, false
	}
	var c RichBlockCaption
	if err := json.Unmarshal(b.rawCaption, &c); err != nil {
		return nil, false
	}
	return &c, true
}

// TableCaptionValue возвращает подпись таблицы как RichText.
func (b RichBlock) TableCaptionValue() (RichText, bool) {
	if len(b.rawCaption) == 0 {
		return RichText{}, false
	}
	var c RichText
	if err := json.Unmarshal(b.rawCaption, &c); err != nil {
		return RichText{}, false
	}
	return c, true
}

// UnmarshalJSON декодирует RichBlock и сохраняет поле caption без интерпретации типа.
func (b *RichBlock) UnmarshalJSON(data []byte) error {
	type raw struct {
		Type       string                 `json:"type"`
		Text       RichText               `json:"text,omitzero"`
		Size       int                    `json:"size,omitempty"`
		Language   string                 `json:"language,omitempty"`
		Expression string                 `json:"expression,omitempty"`
		Name       string                 `json:"name,omitempty"`
		Items      []RichBlockListItem    `json:"items,omitempty"`
		Blocks     []RichBlock            `json:"blocks,omitempty"`
		Credit     RichText               `json:"credit,omitzero"`
		Caption    json.RawMessage        `json:"caption,omitempty"`
		Cells      [][]RichBlockTableCell `json:"cells,omitempty"`
		IsBordered bool                   `json:"is_bordered,omitempty"`
		IsStriped  bool                   `json:"is_striped,omitempty"`
		IsOpen     bool                   `json:"is_open,omitempty"`
		Summary    RichText               `json:"summary,omitzero"`
		Location   *Location              `json:"location,omitempty"`
		Zoom       int                    `json:"zoom,omitempty"`
		Width      int                    `json:"width,omitempty"`
		Height     int                    `json:"height,omitempty"`
		Photo      []PhotoSize            `json:"photo,omitempty"`
		Video      *Video                 `json:"video,omitempty"`
		Animation  *Animation             `json:"animation,omitempty"`
		Audio      *Audio                 `json:"audio,omitempty"`
		VoiceNote  *Voice                 `json:"voice_note,omitempty"`
		HasSpoiler bool                   `json:"has_spoiler,omitempty"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	b.Type = r.Type
	b.Text = r.Text
	b.Size = r.Size
	b.Language = r.Language
	b.Expression = r.Expression
	b.Name = r.Name
	b.Items = r.Items
	b.Blocks = r.Blocks
	b.Credit = r.Credit
	b.Cells = r.Cells
	b.IsBordered = r.IsBordered
	b.IsStriped = r.IsStriped
	b.IsOpen = r.IsOpen
	b.Summary = r.Summary
	b.Location = r.Location
	b.Zoom = r.Zoom
	b.Width = r.Width
	b.Height = r.Height
	b.Photo = r.Photo
	b.Video = r.Video
	b.Animation = r.Animation
	b.Audio = r.Audio
	b.VoiceNote = r.VoiceNote
	b.HasSpoiler = r.HasSpoiler
	b.rawCaption = r.Caption
	return nil
}

// RichBlockListItem описывает элемент входящего списка.
//
//nolint:revive // имя сохраняет терминологию Telegram Bot API.
type RichBlockListItem struct {
	Label       string      `json:"label,omitempty"`
	Blocks      []RichBlock `json:"blocks"`
	HasCheckbox bool        `json:"has_checkbox,omitempty"`
	IsChecked   bool        `json:"is_checked,omitempty"`
	Value       int         `json:"value,omitempty"`
	Type        string      `json:"type,omitempty"`
}

// InputRichBlock описывает исходящий блок Rich Message.
type InputRichBlock struct {
	Type       string
	Text       RichText
	Size       int
	Language   string
	Expression string
	Name       string
	Items      []InputRichBlockListItem
	Blocks     []InputRichBlock
	Credit     RichText
	Caption    *RichBlockCaption
	Cells      [][]RichBlockTableCell
	IsBordered bool
	IsStriped  bool
	IsOpen     bool
	Summary    RichText
	Location   *Location
	Zoom       int
	Width      int
	Height     int
	// TableCaption — подпись таблицы (RichText), только для BlockTable.
	TableCaption RichText
	// Медиа-блоки хранят InputMedia под тем же именем поля, что и тип контента.
	Photo     InputMedia
	Video     InputMedia
	Animation InputMedia
	Audio     InputMedia
	VoiceNote InputMedia
}

// InputRichBlockListItem описывает элемент исходящего списка.
type InputRichBlockListItem struct {
	Blocks      []InputRichBlock `json:"blocks"`
	HasCheckbox bool             `json:"has_checkbox,omitempty"`
	IsChecked   bool             `json:"is_checked,omitempty"`
	Value       int              `json:"value,omitempty"`
	Type        string           `json:"type,omitempty"`
}

// blockJSON описывает поля InputRichBlock при (де)сериализации в JSON.
// Используется и для marshal, и для unmarshal, чтобы избежать дублирования.
type blockJSON struct {
	Type         string                   `json:"type"`
	Text         RichText                 `json:"text,omitzero"`
	Size         int                      `json:"size,omitempty"`
	Language     string                   `json:"language,omitempty"`
	Expression   string                   `json:"expression,omitempty"`
	Name         string                   `json:"name,omitempty"`
	Items        []InputRichBlockListItem `json:"items,omitempty"`
	Blocks       []InputRichBlock         `json:"blocks,omitempty"`
	Credit       RichText                 `json:"credit,omitzero"`
	Caption      *RichBlockCaption        `json:"caption,omitempty"`
	Cells        [][]RichBlockTableCell   `json:"cells,omitempty"`
	IsBordered   bool                     `json:"is_bordered,omitempty"`
	IsStriped    bool                     `json:"is_striped,omitempty"`
	IsOpen       bool                     `json:"is_open,omitempty"`
	Summary      RichText                 `json:"summary,omitzero"`
	Location     *Location                `json:"location,omitempty"`
	Zoom         int                      `json:"zoom,omitempty"`
	Width        int                      `json:"width,omitempty"`
	Height       int                      `json:"height,omitempty"`
	TableCaption RichText                 `json:"table_caption,omitzero"`
	Photo        InputMedia               `json:"photo,omitzero"`
	Video        InputMedia               `json:"video,omitzero"`
	Animation    InputMedia               `json:"animation,omitzero"`
	Audio        InputMedia               `json:"audio,omitzero"`
	VoiceNote    InputMedia               `json:"voice_note,omitzero"`
}

// MarshalJSON сериализует InputRichBlock, включая только поля, релевантные типу блока.
func (b InputRichBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(blockJSON(b))
}

// UnmarshalJSON декодирует InputRichBlock из JSON.
func (b *InputRichBlock) UnmarshalJSON(data []byte) error {
	var v blockJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if v.Type == "" {
		return errors.New("rich block: missing type")
	}
	*b = InputRichBlock(v)
	return nil
}

// mediaJSON описывает поля InputMedia при сериализации в JSON.
type mediaJSON struct {
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

// MarshalJSON сериализует InputMedia, опуская пустые поля.
func (m InputMedia) MarshalJSON() ([]byte, error) {
	return json.Marshal(mediaJSON(m))
}
