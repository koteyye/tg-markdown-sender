package rich

import (
	"errors"
	"fmt"
	"strings"
)

// Лимиты Rich Message по спецификации Telegram Bot API.
const (
	MaxRichTextChars    = 32768
	MaxBlocks           = 500
	MaxNestingDepth     = 16
	MaxMediaAttachments = 50
	MaxTableColumns     = 20
)

// errSingleSource возвращается, если не задан ровно один источник содержимого.
var errSingleSource = errors.New("rich message must contain exactly one of markdown, html or blocks")

// Validate проверяет InputRichMessage перед отправкой в Telegram:
// ровно один источник содержимого, корректные медиа-алиасы и лимиты.
func Validate(m InputRichMessage) error {
	if err := validateSingleContentSource(m); err != nil {
		return err
	}
	if err := validateMedia(m.Media); err != nil {
		return err
	}
	return validateBlocks(m.Blocks)
}

// validateSingleContentSource гарантирует ровно один источник: markdown, html или blocks.
func validateSingleContentSource(m InputRichMessage) error {
	sources := 0
	if m.Markdown != "" {
		sources++
	}
	if m.HTML != "" {
		sources++
	}
	if len(m.Blocks) > 0 {
		sources++
	}
	if sources != 1 {
		return errSingleSource
	}
	return nil
}

func validateMedia(media []InputRichMessageMedia) error {
	if len(media) > MaxMediaAttachments {
		return fmt.Errorf("rich message exceeds limit of %d media attachments", MaxMediaAttachments)
	}
	seen := make(map[string]struct{}, len(media))
	for _, item := range media {
		if err := ValidateAlias(item.ID); err != nil {
			return err
		}
		if _, ok := seen[item.ID]; ok {
			return fmt.Errorf("duplicate media alias %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		if !isAllowedInputMediaType(item.Media.Type) {
			return fmt.Errorf("unsupported input media type %q", item.Media.Type)
		}
		if strings.TrimSpace(item.Media.Media) == "" {
			return fmt.Errorf("media alias %q has empty media reference", item.ID)
		}
	}
	return nil
}

func isAllowedInputMediaType(t string) bool {
	switch t {
	case MediaTypePhoto, MediaTypeVideo, MediaTypeAnimation, MediaTypeAudio, MediaTypeVoiceNote:
		return true
	default:
		return false
	}
}

func validateBlocks(blocks []InputRichBlock) error {
	count, err := countBlocks(blocks, 1)
	if err != nil {
		return err
	}
	if count > MaxBlocks {
		return fmt.Errorf("rich message exceeds limit of %d blocks", MaxBlocks)
	}
	if err := validateMediaBlockCount(blocks, 0); err != nil {
		return err
	}
	return validateTableColumns(blocks)
}

func countBlocks(blocks []InputRichBlock, depth int) (int, error) {
	if depth > MaxNestingDepth {
		return 0, fmt.Errorf("rich message exceeds nesting depth of %d", MaxNestingDepth)
	}
	count := 0
	for _, block := range blocks {
		count++
		switch block.Type {
		case BlockList:
			for _, item := range block.Items {
				n, err := countBlocks(item.Blocks, depth+1)
				if err != nil {
					return 0, err
				}
				count += n
			}
		case BlockBlockquote, BlockCollage, BlockSlideshow, BlockDetails:
			n, err := countBlocks(block.Blocks, depth+1)
			if err != nil {
				return 0, err
			}
			count += n
		}
	}
	return count, nil
}

func validateMediaBlockCount(blocks []InputRichBlock, current int) (err error) {
	for _, block := range blocks {
		if isMediaBlock(block.Type) {
			current++
			if current > MaxMediaAttachments {
				return fmt.Errorf("rich message exceeds limit of %d media attachments", MaxMediaAttachments)
			}
		}
		switch block.Type {
		case BlockList:
			for _, item := range block.Items {
				if err = validateMediaBlockCount(item.Blocks, current); err != nil {
					return err
				}
			}
		case BlockBlockquote, BlockCollage, BlockSlideshow, BlockDetails:
			if err = validateMediaBlockCount(block.Blocks, current); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTableColumns(blocks []InputRichBlock) error {
	for _, block := range blocks {
		if block.Type == BlockTable {
			for _, row := range block.Cells {
				if len(row) > MaxTableColumns {
					return fmt.Errorf("table exceeds limit of %d columns", MaxTableColumns)
				}
			}
		}
		switch block.Type {
		case BlockList:
			for _, item := range block.Items {
				if err := validateTableColumns(item.Blocks); err != nil {
					return err
				}
			}
		case BlockBlockquote, BlockCollage, BlockSlideshow, BlockDetails:
			if err := validateTableColumns(block.Blocks); err != nil {
				return err
			}
		}
	}
	return nil
}

func isMediaBlock(t string) bool {
	switch t {
	case BlockPhoto, BlockVideo, BlockAnimation, BlockAudio, BlockVoiceNote:
		return true
	default:
		return false
	}
}
