package bot

import (
	"strings"
	"testing"

	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

func TestIsAllowedUser(t *testing.T) {
	if !IsAllowedUser(42, 42) {
		t.Fatal("owner must be allowed")
	}
	if IsAllowedUser(7, 42) {
		t.Fatal("non-owner must be denied")
	}
}

func TestPublishErrorTextChatNotFound(t *testing.T) {
	err := &telegram.APIError{
		Method:      "sendRichMessage",
		HTTPStatus:  400,
		Code:        400,
		Description: "Bad Request: chat not found",
	}

	got := publishErrorText(err)

	if strings.Contains(got, "Markdown") {
		t.Fatalf("publish error must not be reported as markdown error: %q", got)
	}
	if !strings.Contains(got, "TELEGRAM_CHANNEL_ID") {
		t.Fatalf("publish error must mention channel configuration: %q", got)
	}
}
