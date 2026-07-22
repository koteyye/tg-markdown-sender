package rich

import "fmt"

// UnsupportedBlockError сообщает о входящем блоке, который бот не умеет преобразовывать.
type UnsupportedBlockError struct {
	Type string
}

func (e *UnsupportedBlockError) Error() string {
	return fmt.Sprintf("unsupported rich block type: %q", e.Type)
}

// MissingFileIDError сообщает о медиа-блоке без file_id.
type MissingFileIDError struct {
	Type string
}

func (e *MissingFileIDError) Error() string {
	return fmt.Sprintf("rich block %q has no file_id", e.Type)
}

// Convert преобразует входящий RichMessage в исходящий InputRichMessage.
// Преобразование рекурсивно по всем контейнерам; медиа используют file_id;
// RichText, включая custom emoji, сохраняется без потерь.
// Возвращает UnsupportedBlockError для неизвестного типа блока.
func Convert(in RichMessage) (InputRichMessage, error) {
	out := InputRichMessage{IsRTL: in.IsRTL}
	if len(in.Blocks) == 0 {
		return out, nil
	}
	blocks, err := convertBlocks(in.Blocks)
	if err != nil {
		return InputRichMessage{}, err
	}
	out.Blocks = blocks
	return out, nil
}

func convertBlocks(in []RichBlock) ([]InputRichBlock, error) {
	out := make([]InputRichBlock, 0, len(in))
	for _, block := range in {
		converted, err := convertBlock(block)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}

func convertBlock(in RichBlock) (InputRichBlock, error) {
	switch {
	case isTextBlock(in.Type):
		return convertTextBlock(in), nil
	case isContainerBlock(in.Type):
		return convertContainerBlock(in)
	case isMediaBlock(in.Type):
		out := InputRichBlock{Type: in.Type}
		if err := convertMediaBlock(&out, in); err != nil {
			return InputRichBlock{}, err
		}
		return out, nil
	case in.Type == BlockTable:
		return convertTableBlock(in)
	case in.Type == BlockMap:
		return convertMapBlock(in), nil
	case in.Type == BlockAnchor:
		return InputRichBlock{Type: in.Type, Name: in.Name}, nil
	case in.Type == BlockMathematicalExpression:
		return InputRichBlock{Type: in.Type, Expression: in.Expression}, nil
	default:
		return InputRichBlock{}, &UnsupportedBlockError{Type: in.Type}
	}
}

func isTextBlock(t string) bool {
	switch t {
	case BlockParagraph, BlockHeading, BlockFooter, BlockPullquote,
		BlockPreformatted, BlockThinking, BlockDivider:
		return true
	default:
		return false
	}
}

func convertTextBlock(in RichBlock) InputRichBlock {
	out := InputRichBlock{Type: in.Type, Text: in.Text, Credit: in.Credit, Size: in.Size}
	if in.Type == BlockPreformatted {
		out.Language = in.Language
	}
	return out
}

func isContainerBlock(t string) bool {
	switch t {
	case BlockList, BlockBlockquote, BlockCollage, BlockSlideshow, BlockDetails:
		return true
	default:
		return false
	}
}

func convertContainerBlock(in RichBlock) (InputRichBlock, error) {
	out := InputRichBlock{Type: in.Type}
	switch in.Type {
	case BlockList:
		items, err := convertListItems(in.Items)
		if err != nil {
			return InputRichBlock{}, err
		}
		out.Items = items
	case BlockBlockquote:
		blocks, err := convertBlocks(in.Blocks)
		if err != nil {
			return InputRichBlock{}, err
		}
		out.Blocks = blocks
		out.Credit = in.Credit
	case BlockCollage, BlockSlideshow:
		blocks, err := convertBlocks(in.Blocks)
		if err != nil {
			return InputRichBlock{}, err
		}
		out.Blocks = blocks
		assignCaption(&out, in)
	case BlockDetails:
		blocks, err := convertBlocks(in.Blocks)
		if err != nil {
			return InputRichBlock{}, err
		}
		out.Summary = in.Summary
		out.Blocks = blocks
		out.IsOpen = in.IsOpen
	}
	return out, nil
}

func convertTableBlock(in RichBlock) (InputRichBlock, error) {
	cells, err := convertTableCells(in.Cells)
	if err != nil {
		return InputRichBlock{}, err
	}
	out := InputRichBlock{
		Type: in.Type, Cells: cells, IsBordered: in.IsBordered, IsStriped: in.IsStriped,
	}
	assignTableCaption(&out, in)
	return out, nil
}

func convertMapBlock(in RichBlock) InputRichBlock {
	out := InputRichBlock{
		Type: in.Type, Location: in.Location, Zoom: in.Zoom, Width: in.Width, Height: in.Height,
	}
	assignCaption(&out, in)
	return out
}

// assignCaption копирует RichBlockCaption из входящего блока, если он задан.
func assignCaption(out *InputRichBlock, in RichBlock) {
	if caption, ok := in.CaptionValue(); ok {
		out.Caption = caption
	}
}

// assignTableCaption копирует подпись таблицы (RichText) из входящего блока.
func assignTableCaption(out *InputRichBlock, in RichBlock) {
	if caption, ok := in.TableCaptionValue(); ok {
		out.TableCaption = caption
	}
}

// convertMediaBlock заполняет соответствующее медиа-поле исходящего блока.
func convertMediaBlock(out *InputRichBlock, in RichBlock) error {
	switch in.Type {
	case BlockPhoto:
		best, ok := bestPhotoSize(in.Photo)
		if !ok {
			return &MissingFileIDError{Type: BlockPhoto}
		}
		out.Photo = NewPhotoMedia(best.FileID)
	case BlockVideo:
		fileID := mediaFileID(BlockVideo, in.Video != nil, func() string { return in.Video.FileID })
		if fileID == "" {
			return &MissingFileIDError{Type: BlockVideo}
		}
		out.Video = NewVideoMedia(fileID)
	case BlockAnimation:
		fileID := mediaFileID(BlockAnimation, in.Animation != nil, func() string { return in.Animation.FileID })
		if fileID == "" {
			return &MissingFileIDError{Type: BlockAnimation}
		}
		out.Animation = NewAnimationMedia(fileID)
	case BlockAudio:
		fileID := mediaFileID(BlockAudio, in.Audio != nil, func() string { return in.Audio.FileID })
		if fileID == "" {
			return &MissingFileIDError{Type: BlockAudio}
		}
		out.Audio = NewAudioMedia(fileID)
	case BlockVoiceNote:
		fileID := mediaFileID(BlockVoiceNote, in.VoiceNote != nil, func() string { return in.VoiceNote.FileID })
		if fileID == "" {
			return &MissingFileIDError{Type: BlockVoiceNote}
		}
		out.VoiceNote = NewVoiceNoteMedia(fileID)
	}
	assignCaption(out, in)
	return nil
}

// mediaFileID возвращает file_id медиа или пустую строку, если объект отсутствует.
func mediaFileID(_ string, present bool, fileID func() string) string {
	if !present {
		return ""
	}
	return fileID()
}

func convertListItems(in []RichBlockListItem) ([]InputRichBlockListItem, error) {
	out := make([]InputRichBlockListItem, 0, len(in))
	for _, item := range in {
		blocks, err := convertBlocks(item.Blocks)
		if err != nil {
			return nil, err
		}
		out = append(out, InputRichBlockListItem{
			Blocks:      blocks,
			HasCheckbox: item.HasCheckbox,
			IsChecked:   item.IsChecked,
			Value:       item.Value,
			Type:        item.Type,
		})
	}
	return out, nil
}

func convertTableCells(in [][]RichBlockTableCell) ([][]RichBlockTableCell, error) {
	out := make([][]RichBlockTableCell, 0, len(in))
	for _, row := range in {
		out = append(out, append([]RichBlockTableCell(nil), row...))
	}
	return out, nil
}
