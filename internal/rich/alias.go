package rich

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// aliasCharsetPattern описывает допустимый набор символов медиа-алиаса (1–64 символа).
var aliasCharsetPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// mediaRefPattern находит ссылки tg://photo|video|audio?id=<alias> в markdown.
var mediaRefPattern = regexp.MustCompile(`tg://(photo|video|audio)\?id=([A-Za-z0-9_-]+)`)

// ErrUnknownMediaAlias сообщает, что алиас медиа не найден в реестре владельца.
var ErrUnknownMediaAlias = errors.New("unknown media alias")

// MediaRef описывает ссылку на медиа-алиас, найденную в markdown.
type MediaRef struct {
	Scheme string // photo, video, audio
	Alias  string
}

// AliasRegistry хранит медиа-алиасы владельца в памяти: alias → InputRichMessageMedia.
type AliasRegistry struct {
	mu      sync.Mutex
	byOwner map[int64]map[string]InputRichMessageMedia
}

// NewAliasRegistry создаёт пустой реестр медиа-алиасов.
func NewAliasRegistry() *AliasRegistry {
	return &AliasRegistry{byOwner: make(map[int64]map[string]InputRichMessageMedia)}
}

// Register сохраняет медиа под новым сгенерированным алиасом для владельца и возвращает алиас.
func (r *AliasRegistry) Register(ownerID int64, media InputRichMessageMedia) (string, error) {
	alias, err := newMediaAlias()
	if err != nil {
		return "", err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byOwner[ownerID] == nil {
		r.byOwner[ownerID] = make(map[string]InputRichMessageMedia)
	}
	r.byOwner[ownerID][alias] = media
	return alias, nil
}

// Lookup возвращает медиа по алиасу для владельца.
func (r *AliasRegistry) Lookup(ownerID int64, alias string) (InputRichMessageMedia, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	media, ok := r.byOwner[ownerID][alias]
	return media, ok
}

// ResolveReferences находит все tg://media?id= ссылки в markdown и возвращает
// соответствующие InputRichMessageMedia из реестра владельца. Алиасы сохраняют
// порядок первого появления. Возвращает ошибку, если алиас не найден.
func (r *AliasRegistry) ResolveReferences(ownerID int64, markdown string) ([]InputRichMessageMedia, []MediaRef, error) {
	refs := ExtractMediaRefs(markdown)
	if len(refs) == 0 {
		return nil, refs, nil
	}

	seen := make(map[string]struct{}, len(refs))
	media := make([]InputRichMessageMedia, 0, len(refs))
	for _, ref := range refs {
		if _, ok := seen[ref.Alias]; ok {
			continue
		}
		seen[ref.Alias] = struct{}{}
		entry, ok := r.Lookup(ownerID, ref.Alias)
		if !ok {
			return nil, refs, fmt.Errorf("%w: %s", ErrUnknownMediaAlias, ref.Alias)
		}
		media = append(media, entry)
	}
	return media, refs, nil
}

// ExtractMediaRefs возвращает ссылки tg://photo|video|audio?id=<alias> в порядке появления.
func ExtractMediaRefs(markdown string) []MediaRef {
	matches := mediaRefPattern.FindAllStringSubmatch(markdown, -1)
	refs := make([]MediaRef, 0, len(matches))
	for _, m := range matches {
		refs = append(refs, MediaRef{Scheme: m[1], Alias: m[2]})
	}
	return refs
}

// ValidateAlias проверяет, что алиас соответствует ограничениям Telegram (1–64, charset).
func ValidateAlias(alias string) error {
	if !aliasCharsetPattern.MatchString(alias) {
		return fmt.Errorf("invalid media alias %q: must be 1-64 characters of A-Z, a-z, 0-9, _ or -", alias)
	}
	return nil
}

// newMediaAlias генерирует безопасный алиас вида photo_<8 hex>.
func newMediaAlias() (string, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return "photo_" + hex.EncodeToString(buf[:]), nil
}

// MarkdownImageRef возвращает ![](tg://photo?id=<alias>) для вставки в markdown.
func MarkdownImageRef(alias string) string {
	return "![](tg://photo?id=" + alias + ")"
}

// HasImagePlaceholder сообщает, что markdown содержит {{image}} плейсхолдер.
func HasImagePlaceholder(markdown string) bool {
	return strings.Contains(markdown, "{{image}}")
}
