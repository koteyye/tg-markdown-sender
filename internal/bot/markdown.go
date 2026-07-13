package bot

import (
	"strings"
	"unicode/utf16"

	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

const markdownCodeBlockLanguage = "md"

// markdownFromCodeBlock returns the original Markdown protected from Telegram's client-side parser.
func markdownFromCodeBlock(text string, entities []telegram.MessageEntity) (string, bool) {
	var markdown string
	found := false

	for _, entity := range entities {
		isMarkdownBlock := entity.Type == "pre" && strings.EqualFold(
			strings.TrimSpace(entity.Language),
			markdownCodeBlockLanguage,
		)
		if !isMarkdownBlock {
			continue
		}

		start, end, ok := entityByteRange(text, entity)
		if !ok || found {
			return "", false
		}

		markdown = text[start:end]
		found = true
	}

	if !found || strings.TrimSpace(markdown) == "" {
		return "", false
	}
	return markdown, true
}

func entityByteRange(text string, entity telegram.MessageEntity) (start, end int, ok bool) {
	if entity.Offset < 0 || entity.Length <= 0 {
		return 0, 0, false
	}

	start, ok = byteIndexForUTF16Offset(text, entity.Offset)
	if !ok {
		return 0, 0, false
	}

	end, ok = byteIndexForUTF16Offset(text, entity.Offset+entity.Length)
	if !ok || end < start {
		return 0, 0, false
	}

	return start, end, true
}

func byteIndexForUTF16Offset(text string, target int) (int, bool) {
	if target < 0 {
		return 0, false
	}

	offset := 0
	for byteIndex, r := range text {
		if offset == target {
			return byteIndex, true
		}
		if offset > target {
			return 0, false
		}
		offset += len(utf16.Encode([]rune{r}))
	}

	if offset == target {
		return len(text), true
	}
	return 0, false
}

func entityDebugAttrs(entities []telegram.MessageEntity) []map[string]string {
	attrs := make([]map[string]string, 0, len(entities))
	for _, entity := range entities {
		attrs = append(attrs, map[string]string{
			"type":            entity.Type,
			"custom_emoji_id": entity.CustomEmojiID,
		})
	}
	return attrs
}
