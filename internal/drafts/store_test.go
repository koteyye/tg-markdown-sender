package drafts

import (
	"errors"
	"testing"

	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

func TestMemoryStoreCreateDraft(t *testing.T) {
	t.Parallel()

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
				draft, err := store.Create("hello")
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
				draft, err := store.Create("hello")
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

func TestMemoryStoreCreatePhotoDraft(t *testing.T) {
	t.Parallel()

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

func TestMemoryStoreDelete(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	draft, err := store.Create("to delete")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	store.Delete(draft.ID)
	if _, ok := store.Get(draft.ID); ok {
		t.Fatal("draft must be deleted")
	}
}
