package telegram

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendRichMessageSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/botsecret/sendRichMessage") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var req SendRichMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.RichMessage.Markdown != "# Title" {
			t.Fatalf("unexpected markdown: %q", req.RichMessage.Markdown)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":10,"chat":{"id":42,"type":"private"}}}`))
	}))
	defer server.Close()

	client := NewClient("secret", server.Client(), testLogger(), WithBaseURL(server.URL))

	msg, err := client.SendRichMessage(context.Background(), int64(42), "# Title", nil)
	if err != nil {
		t.Fatalf("SendRichMessage returned error: %v", err)
	}
	if msg.MessageID != 10 {
		t.Fatalf("unexpected message id: %d", msg.MessageID)
	}
}

func TestGetMeSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/botsecret/getMe") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"id":100,"is_bot":true,"username":"publisher_bot"}}`))
	}))
	defer server.Close()

	client := NewClient("secret", server.Client(), testLogger(), WithBaseURL(server.URL))

	botInfo, err := client.GetMe(context.Background())
	if err != nil {
		t.Fatalf("GetMe returned error: %v", err)
	}
	if botInfo.ID != 100 {
		t.Fatalf("unexpected bot id: %d", botInfo.ID)
	}
	if botInfo.Username != "publisher_bot" {
		t.Fatalf("unexpected username: %q", botInfo.Username)
	}
}

func TestSendRichMessageAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: can't parse markdown"}`))
	}))
	defer server.Close()

	client := NewClient("secret", server.Client(), testLogger(), WithBaseURL(server.URL))

	_, err := client.SendRichMessage(context.Background(), "@channel", "*broken", nil)
	if err == nil {
		t.Fatal("expected API error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("unexpected HTTP status: %d", apiErr.HTTPStatus)
	}
	if !strings.Contains(apiErr.Description, "markdown") {
		t.Fatalf("unexpected description: %q", apiErr.Description)
	}
}

func TestSendPhotoSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/botsecret/sendPhoto") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var req SendPhotoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Photo != "photo-file-id" {
			t.Fatalf("unexpected photo: %q", req.Photo)
		}
		if req.Caption != "hello" {
			t.Fatalf("unexpected caption: %q", req.Caption)
		}
		if len(req.CaptionEntities) != 1 || req.CaptionEntities[0].Type != "bold" {
			t.Fatalf("unexpected caption entities: %#v", req.CaptionEntities)
		}
		if req.ReplyMarkup == nil {
			t.Fatal("reply markup must be sent")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":11,"chat":{"id":42,"type":"private"},"caption":"hello"}}`))
	}))
	defer server.Close()

	client := NewClient("secret", server.Client(), testLogger(), WithBaseURL(server.URL))

	msg, err := client.SendPhoto(
		context.Background(),
		int64(42),
		"photo-file-id",
		"hello",
		[]MessageEntity{{Type: "bold", Offset: 0, Length: 5}},
		&ReplyMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "ok", CallbackData: "ok"}}}},
	)
	if err != nil {
		t.Fatalf("SendPhoto returned error: %v", err)
	}
	if msg.MessageID != 11 {
		t.Fatalf("unexpected message id: %d", msg.MessageID)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
