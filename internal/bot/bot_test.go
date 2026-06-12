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

func TestLargestPhotoPrefersLargestFileSize(t *testing.T) {
	photo, ok := largestPhoto([]telegram.PhotoSize{
		{FileID: "small", Width: 90, Height: 90, FileSize: 100},
		{FileID: "large", Width: 1280, Height: 720, FileSize: 1000},
		{FileID: "medium", Width: 640, Height: 480, FileSize: 500},
	})
	if !ok {
		t.Fatal("expected photo")
	}
	if photo.FileID != "large" {
		t.Fatalf("unexpected largest photo: %q", photo.FileID)
	}
}

func TestLargestPhotoFallsBackToDimensions(t *testing.T) {
	photo, ok := largestPhoto([]telegram.PhotoSize{
		{FileID: "small", Width: 90, Height: 90},
		{FileID: "large", Width: 1280, Height: 720},
		{FileID: "medium", Width: 640, Height: 480},
	})
	if !ok {
		t.Fatal("expected photo")
	}
	if photo.FileID != "large" {
		t.Fatalf("unexpected largest photo: %q", photo.FileID)
	}
}
