package bot

import (
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

func TestMarkdownFromCodeBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		entities []telegram.MessageEntity
		want     string
		wantOK   bool
	}{
		{
			name: "extracts md block and ignores surrounding text",
			text: "Комментарий вне поста\n\n**Знакомьтесь: Рефералодав**\n\n![](https://example.com/image.jpg)\n\nПосле поста",
			entities: []telegram.MessageEntity{entityForSubstring(
				t,
				"Комментарий вне поста\n\n**Знакомьтесь: Рефералодав**\n\n![](https://example.com/image.jpg)\n\nПосле поста",
				"**Знакомьтесь: Рефералодав**\n\n![](https://example.com/image.jpg)",
				"pre",
				"md",
			)},
			want:   "**Знакомьтесь: Рефералодав**\n\n![](https://example.com/image.jpg)",
			wantOK: true,
		},
		{
			name:     "supports unicode offsets",
			text:     "🚀\n**Текст с кодом `fmt.Println`**",
			entities: []telegram.MessageEntity{entityForSubstring(t, "🚀\n**Текст с кодом `fmt.Println`**", "**Текст с кодом `fmt.Println`**", "pre", "MD")},
			want:     "**Текст с кодом `fmt.Println`**",
			wantOK:   true,
		},
		{
			name:     "requires md language",
			text:     "# Heading",
			entities: []telegram.MessageEntity{entityForSubstring(t, "# Heading", "# Heading", "pre", "markdown")},
			wantOK:   false,
		},
		{
			name:     "rejects empty block",
			text:     " ",
			entities: []telegram.MessageEntity{entityForSubstring(t, " ", " ", "pre", "md")},
			wantOK:   false,
		},
		{
			name: "rejects multiple md blocks",
			text: "first\nsecond",
			entities: []telegram.MessageEntity{
				entityForSubstring(t, "first\nsecond", "first", "pre", "md"),
				entityForSubstring(t, "first\nsecond", "second", "pre", "md"),
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := markdownFromCodeBlock(tt.text, tt.entities)
			if ok != tt.wantOK {
				t.Fatalf("markdownFromCodeBlock() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("unexpected markdown:\nwant: %q\n got: %q", tt.want, got)
			}
		})
	}
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
