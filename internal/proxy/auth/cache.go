package auth

import (
	"crypto/sha256"
	"sync"
	"time"
)

// AuthCache хранит успешные Identity по ключу login+password (хеш).
type AuthCache struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[[sha256.Size]byte]cacheEntry
}

type cacheEntry struct {
	id      Identity
	expires time.Time
}

// NewAuthCache создаёт кеш. ttl <= 0 — 5 минут.
func NewAuthCache(ttl time.Duration) *AuthCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &AuthCache{
		ttl:     ttl,
		entries: make(map[[sha256.Size]byte]cacheEntry),
	}
}

func (c *AuthCache) TTL() time.Duration {
	if c == nil {
		return 0
	}
	return c.ttl
}

func (c *AuthCache) Get(username, password string) (Identity, bool) {
	if c == nil {
		return Identity{}, false
	}
	key := cacheKey(username, password)
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expires) {
		if ok {
			delete(c.entries, key)
		}
		return Identity{}, false
	}
	return e.id, true
}

func (c *AuthCache) Set(username, password string, id Identity) {
	if c == nil {
		return
	}
	key := cacheKey(username, password)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{
		id:      id,
		expires: time.Now().Add(c.ttl),
	}
}

// Clear удаляет все записи (например после смены настроек прокси).
func (c *AuthCache) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.entries = make(map[[sha256.Size]byte]cacheEntry)
	c.mu.Unlock()
}

// Reconfigure сбрасывает записи и обновляет TTL.
func (c *AuthCache) Reconfigure(ttl time.Duration) {
	if c == nil {
		return
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	c.mu.Lock()
	c.ttl = ttl
	c.entries = make(map[[sha256.Size]byte]cacheEntry)
	c.mu.Unlock()
}

func cacheKey(username, password string) [sha256.Size]byte {
	const stackBufferSize = 256
	var stack [stackBufferSize]byte
	inputLen := len(username) + 1 + len(password)
	var input []byte
	if inputLen <= len(stack) {
		input = stack[:0:inputLen]
	} else {
		input = make([]byte, 0, inputLen)
	}
	input = append(input, username...)
	input = append(input, 0)
	input = append(input, password...)
	return sha256.Sum256(input)
}
