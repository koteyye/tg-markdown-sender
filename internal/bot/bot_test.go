package bot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/koteyye/tg-markdown-sender/internal/config"
	"github.com/koteyye/tg-markdown-sender/internal/drafts"
	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

const (
	wantFileID = "large"
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
			name:       "prefers largest file size",
			photos:     []telegram.PhotoSize{{FileID: "small", Width: 90, Height: 90, FileSize: 100}, {FileID: wantFileID, Width: 1280, Height: 720, FileSize: 1000}, {FileID: "medium", Width: 640, Height: 480, FileSize: 500}},
			wantFileID: wantFileID,
			wantOK:     true,
		},
		{
			name:       "falls back to dimensions",
			photos:     []telegram.PhotoSize{{FileID: "small", Width: 90, Height: 90}, {FileID: wantFileID, Width: 1280, Height: 720}, {FileID: "medium", Width: 640, Height: 480}},
			wantFileID: wantFileID,
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

func TestHandleRichPhotoMessage(t *testing.T) {
	t.Parallel()

	client := &recordingTelegramClient{downloadedFile: []byte("photo-data")}
	media := &recordingMediaStore{publicURL: "https://media.example.com/images/image.jpg"}
	b := New(
		config.Config{OwnerID: 42, ChannelID: "@channel"},
		client,
		drafts.NewMemoryStore(),
		nil,
		WithMediaStore(media),
	)

	err := b.handleMessage(context.Background(), &telegram.Message{
		From:    &telegram.User{ID: 42},
		Chat:    telegram.Chat{ID: 100},
		Caption: "# Title\n\n{{image}}\n\nText after image",
		Photo:   []telegram.PhotoSize{{FileID: "photo-file-id", Width: 1280, Height: 720, FileSize: 1000}},
	})
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}
	if client.downloadedFileID != "photo-file-id" {
		t.Fatalf("unexpected downloaded file id: %q", client.downloadedFileID)
	}
	if string(media.uploadedPhoto) != "photo-data" {
		t.Fatalf("unexpected uploaded photo: %q", media.uploadedPhoto)
	}
	if len(client.richMessages) != 1 {
		t.Fatalf("expected one rich preview, got %d", len(client.richMessages))
	}
	wantMarkdown := "# Title\n\n![](https://media.example.com/images/image.jpg)\n\nText after image"
	if client.richMessages[0].markdown != wantMarkdown {
		t.Fatalf("unexpected preview markdown:\nwant: %q\n got: %q", wantMarkdown, client.richMessages[0].markdown)
	}
	if client.richMessages[0].replyMarkup == nil {
		t.Fatal("preview must include publish controls")
	}
}

func TestMarkdownWithImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name:     "replaces placeholder",
			markdown: "Before\n\n{{image}}\n\nAfter",
			want:     "Before\n\n![](https://media.example.com/image.jpg)\n\nAfter",
		},
		{
			name:     "appends when placeholder is absent",
			markdown: "Post",
			want:     "Post\n\n![](https://media.example.com/image.jpg)",
		},
		{
			name:     "creates image-only post",
			markdown: " \n\t",
			want:     "![](https://media.example.com/image.jpg)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := markdownWithImage(tt.markdown, "https://media.example.com/image.jpg")
			if got != tt.want {
				t.Fatalf("markdownWithImage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleRichPhotoMessageWithoutCaptionReturnsMarkdownImageBlock(t *testing.T) {
	t.Parallel()

	client := &recordingTelegramClient{downloadedFile: []byte("photo-data")}
	media := &recordingMediaStore{publicURL: "https://media.example.com/images/image.jpg"}
	b := New(
		config.Config{OwnerID: 42, ChannelID: "@channel"},
		client,
		drafts.NewMemoryStore(),
		nil,
		WithMediaStore(media),
	)

	err := b.handleMessage(context.Background(), &telegram.Message{
		From:  &telegram.User{ID: 42},
		Chat:  telegram.Chat{ID: 100},
		Photo: []telegram.PhotoSize{{FileID: "photo-file-id", Width: 1280, Height: 720, FileSize: 1000}},
	})
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}
	if len(client.richMessages) != 0 {
		t.Fatalf("expected no rich preview, got %d", len(client.richMessages))
	}
	if len(client.messages) != 1 {
		t.Fatalf("expected one upload response, got %d", len(client.messages))
	}
	if !strings.Contains(client.messages[0], "![](https://media.example.com/images/image.jpg)") {
		t.Fatalf("upload response does not contain Markdown image block: %q", client.messages[0])
	}
}

func TestHandleInfoImageCommand(t *testing.T) {
	t.Parallel()

	client := &recordingTelegramClient{}
	b := New(config.Config{OwnerID: 42}, client, drafts.NewMemoryStore(), nil)

	err := b.handleMessage(context.Background(), &telegram.Message{
		From: &telegram.User{ID: 42},
		Chat: telegram.Chat{ID: 100},
		Text: "/infoimage@publisher_bot",
	})
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}
	if len(client.messages) != 1 {
		t.Fatalf("expected one response, got %d", len(client.messages))
	}
	if !strings.Contains(client.messages[0], "{{image}}") || !strings.Contains(client.messages[0], "/checkstorage") {
		t.Fatalf("unexpected image info response: %q", client.messages[0])
	}
	if len(client.richMessages) != 0 {
		t.Fatalf("expected no Rich Markdown preview, got %d", len(client.richMessages))
	}
}

func TestHandleCheckStorageCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		media       MediaStore
		wantMessage string
		wantChecks  int
	}{
		{
			name:        "reports unconfigured storage",
			wantMessage: "не настроено",
		},
		{
			name:        "reports accessible storage",
			media:       &recordingMediaStore{},
			wantMessage: "доступно",
			wantChecks:  1,
		},
		{
			name:        "reports inaccessible storage",
			media:       &recordingMediaStore{checkErr: errors.New("access denied")},
			wantMessage: "недоступно",
			wantChecks:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &recordingTelegramClient{}
			options := make([]Option, 0, 1)
			if tt.media != nil {
				options = append(options, WithMediaStore(tt.media))
			}
			b := New(config.Config{OwnerID: 42}, client, drafts.NewMemoryStore(), nil, options...)

			err := b.handleMessage(context.Background(), &telegram.Message{
				From: &telegram.User{ID: 42},
				Chat: telegram.Chat{ID: 100},
				Text: "/checkstorage",
			})
			if err != nil {
				t.Fatalf("handleMessage returned error: %v", err)
			}
			if len(client.messages) != 1 || !strings.Contains(client.messages[0], tt.wantMessage) {
				t.Fatalf("unexpected storage response: %#v", client.messages)
			}
			if media, ok := tt.media.(*recordingMediaStore); ok && media.checkCalls != tt.wantChecks {
				t.Fatalf("Check was called %d times, want %d", media.checkCalls, tt.wantChecks)
			}
		})
	}
}

type richMessageCall struct {
	markdown    string
	replyMarkup *telegram.ReplyMarkup
}

type recordingTelegramClient struct {
	downloadedFile   []byte
	downloadedFileID string
	richMessages     []richMessageCall
	messages         []string
}

func (c *recordingTelegramClient) GetUpdates(context.Context, int64, int) ([]telegram.Update, error) {
	return nil, nil
}

func (c *recordingTelegramClient) DownloadFile(_ context.Context, fileID string) ([]byte, error) {
	c.downloadedFileID = fileID
	return c.downloadedFile, nil
}

func (c *recordingTelegramClient) SendRichMessage(_ context.Context, _ any, markdown string, replyMarkup *telegram.ReplyMarkup) (*telegram.Message, error) {
	c.richMessages = append(c.richMessages, richMessageCall{markdown: markdown, replyMarkup: replyMarkup})
	return &telegram.Message{}, nil
}

func (c *recordingTelegramClient) SendPhoto(context.Context, any, string, string, []telegram.MessageEntity, *telegram.ReplyMarkup) (*telegram.Message, error) {
	return &telegram.Message{}, nil
}

func (c *recordingTelegramClient) SendMessage(_ context.Context, _ any, text string, _ *telegram.ReplyMarkup) (*telegram.Message, error) {
	c.messages = append(c.messages, text)
	return &telegram.Message{}, nil
}

func (c *recordingTelegramClient) AnswerCallbackQuery(context.Context, string, string, bool) error {
	return nil
}

type recordingMediaStore struct {
	publicURL     string
	uploadedPhoto []byte
	checkErr      error
	checkCalls    int
}

func (s *recordingMediaStore) UploadPhoto(_ context.Context, data []byte) (string, error) {
	s.uploadedPhoto = append([]byte(nil), data...)
	return s.publicURL, nil
}

func (s *recordingMediaStore) Check(context.Context) error {
	s.checkCalls++
	return s.checkErr
}
