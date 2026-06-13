package bot

import (
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

func TestRestoreMarkdownEntities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		entities []telegram.MessageEntity
		want     string
	}{
		{
			name:     "pre without language",
			text:     "console.log(\"hello\");",
			entities: []telegram.MessageEntity{entityForSubstring(t, "console.log(\"hello\");", "console.log(\"hello\");", "pre", "")},
			want:     "```\nconsole.log(\"hello\");\n```",
		},
		{
			name:     "pre with language and unicode",
			text:     "console.log(\"Markdown рулит 🚀\");",
			entities: []telegram.MessageEntity{entityForSubstring(t, "console.log(\"Markdown рулит 🚀\");", "console.log(\"Markdown рулит 🚀\");", "pre", "javascript")},
			want:     "```javascript\nconsole.log(\"Markdown рулит 🚀\");\n```",
		},
		{
			name:     "inline code",
			text:     "Use fmt.Println here",
			entities: []telegram.MessageEntity{entityForSubstring(t, "Use fmt.Println here", "fmt.Println", "code", "")},
			want:     "Use `fmt.Println` here",
		},
		{
			name:     "russian text before block",
			text:     "Вставка кода:\n\nconsole.log(1);",
			entities: []telegram.MessageEntity{entityForSubstring(t, "Вставка кода:\n\nconsole.log(1);", "console.log(1);", "pre", "javascript")},
			want:     "Вставка кода:\n\n```javascript\nconsole.log(1);\n```",
		},
		{
			name:     "emoji before block",
			text:     "🚀\nconsole.log(1);",
			entities: []telegram.MessageEntity{entityForSubstring(t, "🚀\nconsole.log(1);", "console.log(1);", "pre", "")},
			want:     "🚀\n```\nconsole.log(1);\n```",
		},
		{
			name: "multiple entities",
			text: "Use fmt.Println\nconsole.log(1);",
			entities: []telegram.MessageEntity{
				entityForSubstring(t, "Use fmt.Println\nconsole.log(1);", "fmt.Println", "code", ""),
				entityForSubstring(t, "Use fmt.Println\nconsole.log(1);", "console.log(1);", "pre", "javascript"),
			},
			want: "Use `fmt.Println`\n```javascript\nconsole.log(1);\n```",
		},
		{
			name:     "custom emoji",
			text:     "😁 Premium",
			entities: []telegram.MessageEntity{customEmojiEntityForSubstring(t, "😁 Premium", "😁", "1234567890123456789")},
			want:     "![😁](tg://emoji?id=1234567890123456789) Premium",
		},
		{
			name: "custom emoji with other entities",
			text: "Старт 😁 and fmt.Println\nconsole.log(1);",
			entities: []telegram.MessageEntity{
				customEmojiEntityForSubstring(t, "Старт 😁 and fmt.Println\nconsole.log(1);", "😁", "1234567890123456789"),
				entityForSubstring(t, "Старт 😁 and fmt.Println\nconsole.log(1);", "fmt.Println", "code", ""),
				entityForSubstring(t, "Старт 😁 and fmt.Println\nconsole.log(1);", "console.log(1);", "pre", "javascript"),
			},
			want: "Старт ![😁](tg://emoji?id=1234567890123456789) and `fmt.Println`\n```javascript\nconsole.log(1);\n```",
		},
		{
			name:     "custom emoji without id does not change text",
			text:     "😁 Premium",
			entities: []telegram.MessageEntity{customEmojiEntityForSubstring(t, "😁 Premium", "😁", "")},
			want:     "😁 Premium",
		},
		{
			name:     "no entities does not change text",
			text:     "# Заголовок\n\n- пункт\n\n```go\nfmt.Println(\"ok\")\n```",
			entities: nil,
			want:     "# Заголовок\n\n- пункт\n\n```go\nfmt.Println(\"ok\")\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := restoreMarkdownEntities(tt.text, tt.entities)
			if got != tt.want {
				t.Fatalf("unexpected markdown:\nwant: %q\n got: %q", tt.want, got)
			}
		})
	}
}

func customEmojiEntityForSubstring(t *testing.T, text, substring, customEmojiID string) telegram.MessageEntity {
	t.Helper()

	entity := entityForSubstring(t, text, substring, "custom_emoji", "")
	entity.CustomEmojiID = customEmojiID
	return entity
}

func entityForSubstring(t *testing.T, text, substring, entityType, language string) telegram.MessageEntity {
	t.Helper()

	byteStart := strings.Index(text, substring)
	if byteStart < 0 {
		t.Fatalf("substring %q not found in %q", substring, text)
	}

	return telegram.MessageEntity{
		Type:     entityType,
		Offset:   utf16Len(text[:byteStart]),
		Length:   utf16Len(substring),
		Language: language,
	}
}

func utf16Len(text string) int {
	total := 0
	for _, r := range text {
		total += len(utf16.Encode([]rune{r}))
	}
	return total
}
