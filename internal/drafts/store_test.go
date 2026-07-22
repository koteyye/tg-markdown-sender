package drafts

import (
	"errors"
	"testing"

	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

func TestMemoryStoreCreateDraft(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	rm := telegram.InputRichMessage{Markdown: "# Title"}

	draft, err := store.Create(rm)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if draft.ID == "" {
		t.Fatal("draft ID must be set")
	}
	if draft.RichMessage.Markdown != "# Title" {
		t.Fatalf("unexpected markdown: %q", draft.RichMessage.Markdown)
	}

	got, ok := store.Get(draft.ID)
	if !ok {
		t.Fatal("draft must be stored")
	}
	if got.RichMessage.Markdown != draft.RichMessage.Markdown {
		t.Fatalf("stored markdown mismatch: %q", got.RichMessage.Markdown)
	}
}

func TestMemoryStoreMarkPublished(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(*MemoryStore) string
		wantErr error
	}{
		{
			name: "marks draft published",
			prepare: func(store *MemoryStore) string {
				draft, err := store.Create(telegram.InputRichMessage{Markdown: "hello"})
				if err != nil {
					t.Fatalf("Create returned error: %v", err)
				}
				return draft.ID
			},
			wantErr: nil,
		},
		{
			name: "returns ErrAlreadyPublished on second call",
			prepare: func(store *MemoryStore) string {
				draft, err := store.Create(telegram.InputRichMessage{Markdown: "hello"})
				if err != nil {
					t.Fatalf("Create returned error: %v", err)
				}
				if _, err := store.MarkPublished(draft.ID); err != nil {
					t.Fatalf("first MarkPublished returned error: %v", err)
				}
				return draft.ID
			},
			wantErr: ErrAlreadyPublished,
		},
		{
			name: "returns ErrNotFound for missing draft",
			prepare: func(*MemoryStore) string {
				return "missing-id"
			},
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := NewMemoryStore()
			id := tt.prepare(store)

			published, err := store.MarkPublished(id)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("MarkPublished error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && !published.Published {
				t.Fatal("draft must be marked as published")
			}
		})
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	draft, err := store.Create(telegram.InputRichMessage{Markdown: "to delete"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	store.Delete(draft.ID)
	if _, ok := store.Get(draft.ID); ok {
		t.Fatal("draft must be deleted")
	}
}

func TestMemoryStorePreservesMediaInDraft(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	rm := telegram.InputRichMessage{
		Markdown: "![](tg://photo?id=cover)",
		Media: []telegram.InputRichMessageMedia{
			{ID: "cover", Media: telegram.InputMedia{Type: "photo", Media: "file-id"}},
		},
	}

	draft, err := store.Create(rm)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	got, ok := store.Get(draft.ID)
	if !ok {
		t.Fatal("draft not found after create")
	}
	if len(got.RichMessage.Media) != 1 || got.RichMessage.Media[0].ID != "cover" {
		t.Fatalf("media not preserved in draft: %#v", got.RichMessage.Media)
	}
}
