package drafts

import (
	"errors"
	"testing"
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
