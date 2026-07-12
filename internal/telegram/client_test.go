package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type handlerCheck struct {
	decodeInto  func(*json.Decoder) error
	respondWith []byte
}

func setupTestServer(t *testing.T, method string, check handlerCheck) (*httptest.Server, *Client) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/bot"+method) {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if check.decodeInto != nil {
			if err := check.decodeInto(json.NewDecoder(r.Body)); err != nil {
				t.Fatalf("decode request: %v", err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(check.respondWith)
	}))
	t.Cleanup(server.Close)

	return server, NewClient("secret", server.Client(), testLogger(), WithBaseURL(server.URL))
}

func TestSendRichMessageSuccess(t *testing.T) {
	t.Parallel()

	_, client := setupTestServer(t, "secret/sendRichMessage", handlerCheck{
		decodeInto: func(dec *json.Decoder) error {
			var req SendRichMessageRequest
			if err := dec.Decode(&req); err != nil {
				return err
			}
			if req.RichMessage.Markdown != "# Title" {
				return errors.New("unexpected markdown: " + req.RichMessage.Markdown)
			}
			return nil
		},
		respondWith: []byte(`{"ok":true,"result":{"message_id":10,"chat":{"id":42,"type":"private"}}}`),
	})

	msg, err := client.SendRichMessage(context.Background(), int64(42), "# Title", nil)
	if err != nil {
		t.Fatalf("SendRichMessage returned error: %v", err)
	}
	if msg.MessageID != 10 {
		t.Fatalf("unexpected message id: %d", msg.MessageID)
	}
}

func TestGetMeSuccess(t *testing.T) {
	t.Parallel()

	_, client := setupTestServer(t, "secret/getMe", handlerCheck{
		respondWith: []byte(`{"ok":true,"result":{"id":100,"is_bot":true,"username":"publisher_bot"}}`),
	})

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

func TestGetUpdatesSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/botsecret/getUpdates") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("offset") != "7" {
			t.Fatalf("unexpected offset: %q", r.URL.Query().Get("offset"))
		}
		if r.URL.Query().Get("timeout") != "50" {
			t.Fatalf("unexpected timeout: %q", r.URL.Query().Get("timeout"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":7,"message":{"message_id":1,"chat":{"id":42}}}]}`))
	}))
	defer server.Close()

	client := NewClient("secret", server.Client(), testLogger(), WithBaseURL(server.URL))

	updates, err := client.GetUpdates(context.Background(), 7, 50)
	if err != nil {
		t.Fatalf("GetUpdates returned error: %v", err)
	}
	if len(updates) != 1 || updates[0].UpdateID != 7 {
		t.Fatalf("unexpected updates: %+v", updates)
	}
}

func TestSendMessageSuccess(t *testing.T) {
	t.Parallel()

	_, client := setupTestServer(t, "secret/sendMessage", handlerCheck{
		decodeInto: func(dec *json.Decoder) error {
			var req SendMessageRequest
			if err := dec.Decode(&req); err != nil {
				return err
			}
			if req.Text != "hello" {
				return errors.New("unexpected text: " + req.Text)
			}
			return nil
		},
		respondWith: []byte(`{"ok":true,"result":{"message_id":12,"chat":{"id":42,"type":"private"}}}`),
	})

	msg, err := client.SendMessage(context.Background(), int64(42), "hello", nil)
	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
	if msg.MessageID != 12 {
		t.Fatalf("unexpected message id: %d", msg.MessageID)
	}
}

func TestAnswerCallbackQuerySuccess(t *testing.T) {
	t.Parallel()

	_, client := setupTestServer(t, "secret/answerCallbackQuery", handlerCheck{
		decodeInto: func(dec *json.Decoder) error {
			var req AnswerCallbackQueryRequest
			if err := dec.Decode(&req); err != nil {
				return err
			}
			if req.CallbackQueryID != "callback-id" {
				return errors.New("unexpected callback query id: " + req.CallbackQueryID)
			}
			return nil
		},
		respondWith: []byte(`{"ok":true}`),
	})

	if err := client.AnswerCallbackQuery(context.Background(), "callback-id", "ok", false); err != nil {
		t.Fatalf("AnswerCallbackQuery returned error: %v", err)
	}
}

func TestSendRichMessageAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("unexpected HTTP status: %d", apiErr.HTTPStatus)
	}
	if !strings.Contains(apiErr.Description, "markdown") {
		t.Fatalf("unexpected description: %q", apiErr.Description)
	}
}

func TestDoRetriesOnServerError(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":500,"description":"internal error"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":13,"chat":{"id":42,"type":"private"}}}`))
	}))
	defer server.Close()

	client := NewClient("secret", server.Client(), testLogger(), WithBaseURL(server.URL))

	msg, err := client.SendMessage(context.Background(), int64(42), "hello", nil)
	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
	if msg.MessageID != 13 {
		t.Fatalf("unexpected message id: %d", msg.MessageID)
	}
	if requests.Load() != 3 {
		t.Fatalf("expected 3 requests, got %d", requests.Load())
	}
}

func TestSendPhotoSuccess(t *testing.T) {
	t.Parallel()

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

func TestDownloadFileSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/botsecret/getFile":
			if r.URL.Query().Get("file_id") != "photo-file-id" {
				t.Fatalf("unexpected file id: %q", r.URL.Query().Get("file_id"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"result":{"file_id":"photo-file-id","file_path":"photos/image.jpg"}}`))
		case "/file/botsecret/photos/image.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("image-bytes"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient("secret", server.Client(), testLogger(), WithBaseURL(server.URL))
	data, err := client.DownloadFile(context.Background(), "photo-file-id")
	if err != nil {
		t.Fatalf("DownloadFile returned error: %v", err)
	}
	if string(data) != "image-bytes" {
		t.Fatalf("unexpected downloaded data: %q", data)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
