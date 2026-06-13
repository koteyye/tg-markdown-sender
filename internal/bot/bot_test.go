package bot

import (
	"strings"
	"testing"

	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

func TestIsAllowedUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		userID   int64
		ownerID  int64
		expected bool
	}{
		{"owner allowed", 42, 42, true},
		{"non-owner denied", 7, 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsAllowedUser(tt.userID, tt.ownerID); got != tt.expected {
				t.Fatalf("IsAllowedUser(%d, %d) = %v, want %v", tt.userID, tt.ownerID, got, tt.expected)
			}
		})
	}
}

func TestPublishErrorText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		err         error
		wantContain string
		wantAvoid   string
	}{
		{
			name:        "chat not found mentions channel config",
			err:         &telegram.APIError{Method: "sendRichMessage", HTTPStatus: 400, Code: 400, Description: "Bad Request: chat not found"},
			wantContain: "TELEGRAM_CHANNEL_ID",
			wantAvoid:   "Markdown",
		},
		{
			name:        "other error includes description",
			err:         &telegram.APIError{Method: "sendRichMessage", HTTPStatus: 400, Code: 400, Description: "some other problem"},
			wantContain: "some other problem",
			wantAvoid:   "TELEGRAM_CHANNEL_ID",
		},
		{
			name:        "nil api error falls back to network message",
			err:         nil,
			wantContain: "сетевой ошибки",
			wantAvoid:   "Markdown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := publishErrorText(tt.err)
			if !strings.Contains(got, tt.wantContain) {
				t.Fatalf("publish error must contain %q: %q", tt.wantContain, got)
			}
			if tt.wantAvoid != "" && strings.Contains(got, tt.wantAvoid) {
				t.Fatalf("publish error must avoid %q: %q", tt.wantAvoid, got)
			}
		})
	}
}

func TestLargestPhoto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		photos     []telegram.PhotoSize
		wantFileID string
		wantOK     bool
	}{
		{
			name: "prefers largest file size",
			photos: []telegram.PhotoSize{
				{FileID: "small", Width: 90, Height: 90, FileSize: 100},
				{FileID: "large", Width: 1280, Height: 720, FileSize: 1000},
				{FileID: "medium", Width: 640, Height: 480, FileSize: 500},
			},
			wantFileID: "large",
			wantOK:     true,
		},
		{
			name: "falls back to dimensions",
			photos: []telegram.PhotoSize{
				{FileID: "small", Width: 90, Height: 90},
				{FileID: "large", Width: 1280, Height: 720},
				{FileID: "medium", Width: 640, Height: 480},
			},
			wantFileID: "large",
			wantOK:     true,
		},
		{
			name:       "empty slice returns not ok",
			photos:     []telegram.PhotoSize{},
			wantFileID: "",
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			photo, ok := largestPhoto(tt.photos)
			if ok != tt.wantOK {
				t.Fatalf("largestPhoto ok = %v, want %v", ok, tt.wantOK)
			}
			if photo.FileID != tt.wantFileID {
				t.Fatalf("unexpected largest photo: %q, want %q", photo.FileID, tt.wantFileID)
			}
		})
	}
}
