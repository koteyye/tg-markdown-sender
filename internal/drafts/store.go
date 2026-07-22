// Package drafts предоставляет хранилище черновиков постов в памяти.
package drafts

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

var (
	// ErrNotFound возвращается, если черновик не найден в хранилище.
	ErrNotFound = errors.New("draft not found")
	// ErrAlreadyPublished возвращается, если черновик уже опубликован.
	ErrAlreadyPublished = errors.New("draft already published")
)

// Draft представляет черновик Rich Message поста.
type Draft struct {
	ID          string
	RichMessage telegram.InputRichMessage
	Published   bool
	CreatedAt   time.Time
	PublishedAt *time.Time
}

// Store описывает операции с черновиками постов.
type Store interface {
	Create(richMessage telegram.InputRichMessage) (Draft, error)
	Get(id string) (Draft, bool)
	Delete(id string)
	MarkPublished(id string) (Draft, error)
}

// MemoryStore хранит черновики в памяти с защитой мьютексом.
type MemoryStore struct {
	mu     sync.RWMutex
	drafts map[string]Draft
}

// NewMemoryStore создаёт новое in-memory хранилище черновиков.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{drafts: make(map[string]Draft)}
}

// Create создаёт черновик Rich Message поста.
func (s *MemoryStore) Create(richMessage telegram.InputRichMessage) (Draft, error) {
	id, err := randomID()
	if err != nil {
		return Draft{}, err
	}

	draft := Draft{
		ID:          id,
		RichMessage: richMessage,
		CreatedAt:   time.Now().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.drafts[id] = draft

	return draft, nil
}

// Get возвращает черновик по идентификатору.
func (s *MemoryStore) Get(id string) (Draft, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	draft, ok := s.drafts[id]
	return draft, ok
}

// Delete удаляет черновик из хранилища.
func (s *MemoryStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.drafts, id)
}

// MarkPublished помечает черновик как опубликованный.
func (s *MemoryStore) MarkPublished(id string) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	draft, ok := s.drafts[id]
	if !ok {
		return Draft{}, ErrNotFound
	}
	if draft.Published {
		return draft, ErrAlreadyPublished
	}

	now := time.Now().UTC()
	draft.Published = true
	draft.PublishedAt = &now
	s.drafts[id] = draft

	return draft, nil
}

func randomID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
