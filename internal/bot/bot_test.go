package bot

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/koteyye/tg-markdown-sender/internal/config"
	"github.com/koteyye/tg-markdown-sender/internal/drafts"
	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

const (
	wantFileID      = "large"
	testPhotoFileID = "photo-file-id"
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

func TestHandleMessageUsesMarkdownCodeBlock(t *testing.T) {
	t.Parallel()

	client := &recordingTelegramClient{}
	b := New(config.Config{OwnerID: 42}, client, drafts.NewMemoryStore(), nil)
	markdown := "/start\n\n**Знакомьтесь: Рефералодав**\n\n![](https://example.com/image.jpg)"

	err := b.handleMessage(context.Background(), &telegram.Message{
		From:     &telegram.User{ID: 42},
		Chat:     telegram.Chat{ID: 100},
		Text:     "Текст вне блока\n" + markdown + "\nТекст вне блока",
		Entities: []telegram.MessageEntity{entityForSubstring(t, "Текст вне блока\n"+markdown+"\nТекст вне блока", markdown, "pre", "md")},
	})
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}
	if len(client.richMessages) != 1 {
		t.Fatalf("expected one rich preview, got %d", len(client.richMessages))
	}
	if client.richMessages[0].markdown != markdown {
		t.Fatalf("unexpected preview markdown:\nwant: %q\n got: %q", markdown, client.richMessages[0].markdown)
	}
	if len(client.messages) != 0 {
		t.Fatalf("expected no plain responses, got %#v", client.messages)
	}
}

func TestHandleMessageRejectsMarkdownOutsideCodeBlock(t *testing.T) {
	t.Parallel()

	client := &recordingTelegramClient{}
	b := New(config.Config{OwnerID: 42}, client, drafts.NewMemoryStore(), nil)

	err := b.handleMessage(context.Background(), &telegram.Message{
		From: &telegram.User{ID: 42},
		Chat: telegram.Chat{ID: 100},
		Text: "**Знакомьтесь: Рефералодав**",
	})
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}
	if len(client.richMessages) != 0 {
		t.Fatalf("expected no rich previews, got %d", len(client.richMessages))
	}
	if len(client.messages) != 1 || !strings.Contains(client.messages[0], "```md") {
		t.Fatalf("expected markdown input instructions, got %#v", client.messages)
	}
}

func TestScheduleKeyboard(t *testing.T) {
	t.Parallel()

	keyboard := scheduleKeyboard("draft-id")
	if len(keyboard.InlineKeyboard) != 2 {
		t.Fatalf("expected two keyboard rows, got %d", len(keyboard.InlineKeyboard))
	}

	buttons := make([]telegram.InlineKeyboardButton, 0, 5)
	for _, row := range keyboard.InlineKeyboard {
		buttons = append(buttons, row...)
	}
	if len(buttons) != 5 {
		t.Fatalf("expected five schedule slots, got %d", len(buttons))
	}

	for index, slot := range scheduleSlots {
		button := buttons[index]
		if button.Text != slot.label {
			t.Fatalf("button %d text = %q, want %q", index, button.Text, slot.label)
		}
		wantCallback := "schedule-at:" + strconv.Itoa(slot.hour) + ":draft-id"
		if button.CallbackData != wantCallback {
			t.Fatalf("button %d callback = %q, want %q", index, button.CallbackData, wantCallback)
		}
	}
}

func TestPreviewKeyboardIncludesScheduleAction(t *testing.T) {
	t.Parallel()

	keyboard := previewKeyboard("draft-id")
	buttons := make([]telegram.InlineKeyboardButton, 0, 3)
	for _, row := range keyboard.InlineKeyboard {
		buttons = append(buttons, row...)
	}

	for _, button := range buttons {
		if button.Text == "Отправить потом" && button.CallbackData == "schedule:draft-id" {
			return
		}
	}
	t.Fatalf("schedule button not found: %#v", buttons)
}

func TestParseScheduleData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     string
		wantHour int
		wantID   string
		wantOK   bool
	}{
		{
			name:     "valid slot",
			data:     "09:draft-id",
			wantHour: 9,
			wantID:   "draft-id",
			wantOK:   true,
		},
		{
			name:   "unsupported slot",
			data:   "10:draft-id",
			wantOK: false,
		},
		{
			name:   "missing draft ID",
			data:   "09:",
			wantOK: false,
		},
		{
			name:   "invalid hour",
			data:   "morning:draft-id",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hour, draftID, ok := parseScheduleData(tt.data)
			if ok != tt.wantOK {
				t.Fatalf("parseScheduleData() ok = %v, want %v", ok, tt.wantOK)
			}
			if hour != tt.wantHour || draftID != tt.wantID {
				t.Fatalf("parseScheduleData() = (%d, %q), want (%d, %q)", hour, draftID, tt.wantHour, tt.wantID)
			}
		})
	}
}

func TestNextScheduleTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		now  time.Time
		hour int
		want time.Time
	}{
		{
			name: "uses selected time today when it has not passed",
			now:  time.Date(2026, time.July, 14, 8, 59, 59, 0, moscowLocation),
			hour: 9,
			want: time.Date(2026, time.July, 14, 9, 0, 0, 0, moscowLocation),
		},
		{
			name: "moves passed time to tomorrow",
			now:  time.Date(2026, time.July, 14, 9, 0, 0, 0, moscowLocation),
			hour: 9,
			want: time.Date(2026, time.July, 15, 9, 0, 0, 0, moscowLocation),
		},
		{
			name: "converts current time to moscow time",
			now:  time.Date(2026, time.July, 14, 6, 30, 0, 0, time.UTC),
			hour: 12,
			want: time.Date(2026, time.July, 14, 12, 0, 0, 0, moscowLocation),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := nextScheduleTime(tt.now, tt.hour); !got.Equal(tt.want) {
				t.Fatalf("nextScheduleTime() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestHandleScheduleRequestShowsTimeSlots(t *testing.T) {
	t.Parallel()

	client := &recordingTelegramClient{}
	store := drafts.NewMemoryStore()
	b := New(config.Config{OwnerID: 42}, client, store, nil)
	draft, err := store.Create("# Post")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	err = b.handleCallback(context.Background(), &telegram.CallbackQuery{
		ID:      "callback-id",
		From:    telegram.User{ID: 42},
		Message: &telegram.Message{Chat: telegram.Chat{ID: 100}},
		Data:    "schedule:" + draft.ID,
	})
	if err != nil {
		t.Fatalf("handleCallback returned error: %v", err)
	}
	if len(client.messages) != 1 || client.messages[0] != "Выберите время отправки по МСК:" {
		t.Fatalf("unexpected schedule prompt: %#v", client.messages)
	}
	if len(client.messageReplyMarkups) != 1 || client.messageReplyMarkups[0] == nil {
		t.Fatalf("expected schedule keyboard, got %#v", client.messageReplyMarkups)
	}
}

func TestScheduleDraft(t *testing.T) {
	t.Parallel()

	client := &recordingTelegramClient{}
	store := drafts.NewMemoryStore()
	b := New(config.Config{OwnerID: 42}, client, store, nil)
	b.now = func() time.Time {
		return time.Date(2026, time.July, 14, 10, 30, 0, 0, moscowLocation)
	}
	draft, err := store.Create("# Post")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	err = b.handleCallback(context.Background(), &telegram.CallbackQuery{
		ID:      "callback-id",
		From:    telegram.User{ID: 42},
		Message: &telegram.Message{Chat: telegram.Chat{ID: 100}},
		Data:    "schedule-at:09:" + draft.ID,
	})
	if err != nil {
		t.Fatalf("handleCallback returned error: %v", err)
	}

	b.scheduledMu.Lock()
	scheduled, ok := b.scheduledPost[draft.ID]
	b.scheduledMu.Unlock()
	if !ok {
		t.Fatal("draft was not scheduled")
	}
	wantTime := time.Date(2026, time.July, 15, 9, 0, 0, 0, moscowLocation)
	if !scheduled.at.Equal(wantTime) {
		t.Fatalf("scheduled time = %s, want %s", scheduled.at, wantTime)
	}
	if scheduled.chatID != 100 {
		t.Fatalf("scheduled chat ID = %d, want 100", scheduled.chatID)
	}
	if len(client.messages) != 1 || !strings.Contains(client.messages[0], "15.07.2026 09:00 МСК") {
		t.Fatalf("unexpected schedule confirmation: %#v", client.messages)
	}
}

func TestPublishDueScheduledDrafts(t *testing.T) {
	t.Parallel()

	client := &recordingTelegramClient{}
	store := drafts.NewMemoryStore()
	b := New(config.Config{OwnerID: 42, ChannelID: "@channel"}, client, store, nil)
	now := time.Date(2026, time.July, 14, 9, 0, 1, 0, moscowLocation)
	b.now = func() time.Time { return now }
	draft, err := store.Create("# Post")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	b.setScheduledDraft(scheduledDraft{
		draftID: draft.ID,
		chatID:  100,
		at:      now.Add(-time.Second),
	})

	b.publishDueDrafts(context.Background())

	if len(client.richMessages) != 1 || client.richMessages[0].markdown != "# Post" {
		t.Fatalf("unexpected published posts: %#v", client.richMessages)
	}
	published, ok := store.Get(draft.ID)
	if !ok || !published.Published {
		t.Fatalf("scheduled draft was not marked published: %#v", published)
	}
	b.scheduledMu.Lock()
	_, stillScheduled := b.scheduledPost[draft.ID]
	b.scheduledMu.Unlock()
	if stillScheduled {
		t.Fatal("published draft must be removed from the schedule")
	}
	if len(client.messages) != 1 || client.messages[0] != "Пост опубликован по расписанию." {
		t.Fatalf("unexpected publish notification: %#v", client.messages)
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

	caption := "# Title\n\n{{image}}\n\nText after image"
	err := b.handleMessage(context.Background(), &telegram.Message{
		From:            &telegram.User{ID: 42},
		Chat:            telegram.Chat{ID: 100},
		Caption:         caption,
		CaptionEntities: []telegram.MessageEntity{entityForSubstring(t, caption, caption, "pre", "md")},
		Photo:           []telegram.PhotoSize{{FileID: testPhotoFileID, Width: 1280, Height: 720, FileSize: 1000}},
	})
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}
	if client.downloadedFileID != testPhotoFileID {
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
		Photo: []telegram.PhotoSize{{FileID: testPhotoFileID, Width: 1280, Height: 720, FileSize: 1000}},
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

func TestHandleRichPhotoMessageRejectsCaptionOutsideCodeBlock(t *testing.T) {
	t.Parallel()

	client := &recordingTelegramClient{downloadedFile: []byte("photo-data")}
	media := &recordingMediaStore{publicURL: "https://media.example.com/images/image.jpg"}
	b := New(
		config.Config{OwnerID: 42},
		client,
		drafts.NewMemoryStore(),
		nil,
		WithMediaStore(media),
	)

	err := b.handleMessage(context.Background(), &telegram.Message{
		From:    &telegram.User{ID: 42},
		Chat:    telegram.Chat{ID: 100},
		Caption: "**Caption**",
		Photo:   []telegram.PhotoSize{{FileID: testPhotoFileID, Width: 1280, Height: 720, FileSize: 1000}},
	})
	if err != nil {
		t.Fatalf("handleMessage returned error: %v", err)
	}
	if client.downloadedFileID != "" || len(media.uploadedPhoto) != 0 {
		t.Fatal("photo must not be processed before validating its Markdown caption")
	}
	if len(client.richMessages) != 0 {
		t.Fatalf("expected no rich previews, got %d", len(client.richMessages))
	}
	if len(client.messages) != 1 || !strings.Contains(client.messages[0], "```md") {
		t.Fatalf("expected markdown input instructions, got %#v", client.messages)
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
	downloadedFile      []byte
	downloadedFileID    string
	richMessages        []richMessageCall
	messages            []string
	messageReplyMarkups []*telegram.ReplyMarkup
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

func (c *recordingTelegramClient) SendMessage(_ context.Context, _ any, text string, replyMarkup *telegram.ReplyMarkup) (*telegram.Message, error) {
	c.messages = append(c.messages, text)
	c.messageReplyMarkups = append(c.messageReplyMarkups, replyMarkup)
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
