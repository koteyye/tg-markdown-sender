package bot

import (
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

type markdownInsertion struct {
	pos   int
	text  string
	order int
}

func restoreMarkdownEntities(text string, entities []telegram.MessageEntity) string {
	if len(entities) == 0 || text == "" {
		return text
	}

	insertions := make([]markdownInsertion, 0, len(entities)*2)
	for _, entity := range entities {
		prefix, suffix, ok := markdownMarkers(entity)
		if !ok {
			continue
		}

		start, end, ok := entityByteRange(text, entity)
		if !ok {
			continue
		}

		insertions = append(insertions,
			markdownInsertion{pos: start, text: prefix, order: 0},
			markdownInsertion{pos: end, text: suffix, order: 1},
		)
	}

	if len(insertions) == 0 {
		return text
	}

	sort.SliceStable(insertions, func(i, j int) bool {
		if insertions[i].pos != insertions[j].pos {
			return insertions[i].pos > insertions[j].pos
		}
		return insertions[i].order < insertions[j].order
	})

	var builder strings.Builder
	builder.Grow(len(text) + insertionTextLen(insertions))
	builder.WriteString(text)
	result := builder.String()

	for _, insertion := range insertions {
		result = result[:insertion.pos] + insertion.text + result[insertion.pos:]
	}

	return result
}

func markdownMarkers(entity telegram.MessageEntity) (prefix, suffix string, ok bool) {
	switch entity.Type {
	case "pre":
		language := strings.TrimSpace(entity.Language)
		if language == "" {
			return "```\n", "\n```", true
		}
		return "```" + language + "\n", "\n```", true
	case "code":
		return "`", "`", true
	case "custom_emoji":
		customEmojiID := strings.TrimSpace(entity.CustomEmojiID)
		if customEmojiID == "" {
			return "", "", false
		}
		return "![", "](tg://emoji?id=" + customEmojiID + ")", true
	default:
		return "", "", false
	}
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

func insertionTextLen(insertions []markdownInsertion) int {
	total := 0
	for _, insertion := range insertions {
		total += len(insertion.text)
	}
	return total
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
