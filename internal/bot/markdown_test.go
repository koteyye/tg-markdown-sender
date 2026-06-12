package bot

import (
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

func TestRestoreMarkdownEntitiesPreWithoutLanguage(t *testing.T) {
	text := "console.log(\"hello\");"
	entities := []telegram.MessageEntity{entityForSubstring(t, text, "console.log(\"hello\");", "pre", "")}

	got := restoreMarkdownEntities(text, entities)
	want := "```\nconsole.log(\"hello\");\n```"

	if got != want {
		t.Fatalf("unexpected markdown:\nwant: %q\n got: %q", want, got)
	}
}

func TestRestoreMarkdownEntitiesPreWithLanguage(t *testing.T) {
	text := "console.log(\"Markdown рулит 🚀\");"
	entities := []telegram.MessageEntity{entityForSubstring(t, text, text, "pre", "javascript")}

	got := restoreMarkdownEntities(text, entities)
	want := "```javascript\nconsole.log(\"Markdown рулит 🚀\");\n```"

	if got != want {
		t.Fatalf("unexpected markdown:\nwant: %q\n got: %q", want, got)
	}
}

func TestRestoreMarkdownEntitiesInlineCode(t *testing.T) {
	text := "Use fmt.Println here"
	entities := []telegram.MessageEntity{entityForSubstring(t, text, "fmt.Println", "code", "")}

	got := restoreMarkdownEntities(text, entities)
	want := "Use `fmt.Println` here"

	if got != want {
		t.Fatalf("unexpected markdown:\nwant: %q\n got: %q", want, got)
	}
}

func TestRestoreMarkdownEntitiesRussianTextBeforeBlock(t *testing.T) {
	text := "Вставка кода:\n\nconsole.log(1);"
	entities := []telegram.MessageEntity{entityForSubstring(t, text, "console.log(1);", "pre", "javascript")}

	got := restoreMarkdownEntities(text, entities)
	want := "Вставка кода:\n\n```javascript\nconsole.log(1);\n```"

	if got != want {
		t.Fatalf("unexpected markdown:\nwant: %q\n got: %q", want, got)
	}
}

func TestRestoreMarkdownEntitiesEmojiBeforeBlock(t *testing.T) {
	text := "🚀\nconsole.log(1);"
	entities := []telegram.MessageEntity{entityForSubstring(t, text, "console.log(1);", "pre", "")}

	got := restoreMarkdownEntities(text, entities)
	want := "🚀\n```\nconsole.log(1);\n```"

	if got != want {
		t.Fatalf("unexpected markdown:\nwant: %q\n got: %q", want, got)
	}
}

func TestRestoreMarkdownEntitiesMultipleEntities(t *testing.T) {
	text := "Use fmt.Println\nconsole.log(1);"
	entities := []telegram.MessageEntity{
		entityForSubstring(t, text, "fmt.Println", "code", ""),
		entityForSubstring(t, text, "console.log(1);", "pre", "javascript"),
	}

	got := restoreMarkdownEntities(text, entities)
	want := "Use `fmt.Println`\n```javascript\nconsole.log(1);\n```"

	if got != want {
		t.Fatalf("unexpected markdown:\nwant: %q\n got: %q", want, got)
	}
}

func TestRestoreMarkdownEntitiesCustomEmoji(t *testing.T) {
	text := "😁 Premium"
	entities := []telegram.MessageEntity{
		customEmojiEntityForSubstring(t, text, "😁", "1234567890123456789"),
	}

	got := restoreMarkdownEntities(text, entities)
	want := "![😁](tg://emoji?id=1234567890123456789) Premium"

	if got != want {
		t.Fatalf("unexpected markdown:\nwant: %q\n got: %q", want, got)
	}
}

func TestRestoreMarkdownEntitiesCustomEmojiWithOtherEntities(t *testing.T) {
	text := "Старт 😁 and fmt.Println\nconsole.log(1);"
	entities := []telegram.MessageEntity{
		customEmojiEntityForSubstring(t, text, "😁", "1234567890123456789"),
		entityForSubstring(t, text, "fmt.Println", "code", ""),
		entityForSubstring(t, text, "console.log(1);", "pre", "javascript"),
	}

	got := restoreMarkdownEntities(text, entities)
	want := "Старт ![😁](tg://emoji?id=1234567890123456789) and `fmt.Println`\n```javascript\nconsole.log(1);\n```"

	if got != want {
		t.Fatalf("unexpected markdown:\nwant: %q\n got: %q", want, got)
	}
}

func TestRestoreMarkdownEntitiesCustomEmojiWithoutIDDoesNotChangeText(t *testing.T) {
	text := "😁 Premium"
	entities := []telegram.MessageEntity{
		customEmojiEntityForSubstring(t, text, "😁", ""),
	}

	got := restoreMarkdownEntities(text, entities)

	if got != text {
		t.Fatalf("custom emoji without id must not change text:\nwant: %q\n got: %q", text, got)
	}
}

func TestRestoreMarkdownEntitiesNoEntitiesDoesNotChangeText(t *testing.T) {
	text := "# Заголовок\n\n- пункт\n\n```go\nfmt.Println(\"ok\")\n```"

	got := restoreMarkdownEntities(text, nil)

	if got != text {
		t.Fatalf("text without entities must not change:\nwant: %q\n got: %q", text, got)
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
