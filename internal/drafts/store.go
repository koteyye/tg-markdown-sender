package drafts

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound         = errors.New("draft not found")
	ErrAlreadyPublished = errors.New("draft already published")
)

type Draft struct {
	ID          string
	Markdown    string
	Published   bool
	CreatedAt   time.Time
	PublishedAt *time.Time
}

type Store interface {
	Create(markdown string) (Draft, error)
	Get(id string) (Draft, bool)
	Delete(id string)
	MarkPublished(id string) (Draft, error)
}

type MemoryStore struct {
	mu     sync.RWMutex
	drafts map[string]Draft
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{drafts: make(map[string]Draft)}
}

func (s *MemoryStore) Create(markdown string) (Draft, error) {
	id, err := randomID()
	if err != nil {
		return Draft{}, err
	}

	draft := Draft{
		ID:        id,
		Markdown:  markdown,
		CreatedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.drafts[id] = draft

	return draft, nil
}

func (s *MemoryStore) Get(id string) (Draft, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	draft, ok := s.drafts[id]
	return draft, ok
}

func (s *MemoryStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.drafts, id)
}

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
