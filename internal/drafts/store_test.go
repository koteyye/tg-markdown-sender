package drafts

import (
	"errors"
	"testing"

	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

func TestMemoryStoreCreateDraft(t *testing.T) {
	store := NewMemoryStore()

	draft, err := store.Create("# Title")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if draft.ID == "" {
		t.Fatal("draft ID must be set")
	}
	if draft.Markdown != "# Title" {
		t.Fatalf("unexpected markdown: %q", draft.Markdown)
	}

	got, ok := store.Get(draft.ID)
	if !ok {
		t.Fatal("draft must be stored")
	}
	if got.Markdown != draft.Markdown {
		t.Fatalf("stored markdown mismatch: %q", got.Markdown)
	}
}

func TestMemoryStoreMarkPublishedPreventsDuplicates(t *testing.T) {
	store := NewMemoryStore()
	draft, err := store.Create("hello")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	published, err := store.MarkPublished(draft.ID)
	if err != nil {
		t.Fatalf("first MarkPublished returned error: %v", err)
	}
	if !published.Published {
		t.Fatal("draft must be marked as published")
	}

	_, err = store.MarkPublished(draft.ID)
	if !errors.Is(err, ErrAlreadyPublished) {
		t.Fatalf("second MarkPublished must return ErrAlreadyPublished, got %v", err)
	}
}

func TestMemoryStoreCreatePhotoDraft(t *testing.T) {
	store := NewMemoryStore()
	entities := []telegram.MessageEntity{{Type: "bold", Offset: 0, Length: 5}}

	draft, err := store.CreatePhoto("photo-file-id", "hello", entities)
	if err != nil {
		t.Fatalf("CreatePhoto returned error: %v", err)
	}
	if draft.PhotoFileID != "photo-file-id" {
		t.Fatalf("unexpected photo file id: %q", draft.PhotoFileID)
	}
	if draft.Caption != "hello" {
		t.Fatalf("unexpected caption: %q", draft.Caption)
	}
	if len(draft.CaptionEntities) != 1 || draft.CaptionEntities[0].Type != "bold" {
		t.Fatalf("caption entities were not stored: %#v", draft.CaptionEntities)
	}
}
