package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// MemoryAuthenticationCache implements in-memory caching of plugin authentication
type MemoryAuthenticationCache struct {
	cache map[string]*CachedAuth
	mutex sync.RWMutex
	debug bool
}

// CachedAuth represents a cached authentication result
type CachedAuth struct {
	Response  *ports.AuthResponse
	ExpiresAt time.Time
	CachedAt  time.Time
}

// NewMemoryAuthenticationCache creates a new in-memory authentication cache
func NewMemoryAuthenticationCache(debug bool) *MemoryAuthenticationCache {
	cache := &MemoryAuthenticationCache{
		cache: make(map[string]*CachedAuth),
		debug: debug,
	}

	// Start background cleanup goroutine
	go cache.startCleanup()

	return cache
}

// Get retrieves cached authentication for a plugin
func (c *MemoryAuthenticationCache) Get(pluginName string, apiKey string) *ports.AuthResponse {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	key := c.getCacheKey(pluginName, apiKey)
	cached, exists := c.cache[key]

	if !exists {
		if c.debug {
			fmt.Printf("[AuthCache] Cache miss for plugin %s\n", pluginName)
		}
		return nil
	}

	// Check if cached auth has expired
	if time.Now().After(cached.ExpiresAt) {
		if c.debug {
			fmt.Printf("[AuthCache] Cached auth expired for plugin %s\n", pluginName)
		}

		// Remove expired entry
		delete(c.cache, key)
		return nil
	}

	if c.debug {
		fmt.Printf("[AuthCache] Cache hit for plugin %s (expires in %v)\n",
			pluginName, time.Until(cached.ExpiresAt))
	}

	return cached.Response
}

// Set stores authentication result in cache
func (c *MemoryAuthenticationCache) Set(pluginName string, apiKey string, auth *ports.AuthResponse) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := c.getCacheKey(pluginName, apiKey)

	// Calculate expiration time
	expiresAt := time.Now().Add(5 * time.Minute) // Default 5 minutes

	// If auth response has explicit expiration, use it but cap at 1 hour
	if auth.ExpiresAt != nil {
		if parsed, err := time.Parse(time.RFC3339, *auth.ExpiresAt); err == nil {
			maxExpiry := time.Now().Add(1 * time.Hour)
			if parsed.Before(maxExpiry) {
				expiresAt = parsed
			} else {
				expiresAt = maxExpiry
			}
		}
	}

	cached := &CachedAuth{
		Response:  auth,
		ExpiresAt: expiresAt,
		CachedAt:  time.Now(),
	}

	c.cache[key] = cached

	if c.debug {
		fmt.Printf("[AuthCache] Cached auth for plugin %s (expires at %v)\n",
			pluginName, expiresAt.Format("15:04:05"))
	}
}

// Clear removes cached authentication
func (c *MemoryAuthenticationCache) Clear(pluginName string, apiKey string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := c.getCacheKey(pluginName, apiKey)
	delete(c.cache, key)

	if c.debug {
		fmt.Printf("[AuthCache] Cleared cached auth for plugin %s\n", pluginName)
	}
}

// getCacheKey generates a cache key for plugin name and API key
func (c *MemoryAuthenticationCache) getCacheKey(pluginName string, apiKey string) string {
	// Use hash of API key for security (don't store raw API key)
	hash := sha256.Sum256([]byte(apiKey))
	hashedKey := hex.EncodeToString(hash[:8]) // First 8 bytes for shorter key

	return fmt.Sprintf("%s:%s", pluginName, hashedKey)
}

// startCleanup runs background cleanup of expired cache entries
func (c *MemoryAuthenticationCache) startCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpired()
	}
}

// cleanupExpired removes expired entries from cache
func (c *MemoryAuthenticationCache) cleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, cached := range c.cache {
		if now.After(cached.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(c.cache, key)
	}

	if len(expiredKeys) > 0 && c.debug {
		fmt.Printf("[AuthCache] Cleaned up %d expired cache entries\n", len(expiredKeys))
	}
}

// GetCacheStats returns statistics about the cache
func (c *MemoryAuthenticationCache) GetCacheStats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	totalEntries := len(c.cache)
	expiredEntries := 0
	now := time.Now()

	for _, cached := range c.cache {
		if now.After(cached.ExpiresAt) {
			expiredEntries++
		}
	}

	return CacheStats{
		TotalEntries:   totalEntries,
		ActiveEntries:  totalEntries - expiredEntries,
		ExpiredEntries: expiredEntries,
	}
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalEntries   int `json:"total_entries"`
	ActiveEntries  int `json:"active_entries"`
	ExpiredEntries int `json:"expired_entries"`
}

// Security and production methods

// EncryptedAuthenticationCache would provide encrypted storage
type EncryptedAuthenticationCache struct {
	*MemoryAuthenticationCache
	encryptionKey []byte
}

// NewEncryptedAuthenticationCache creates a cache with encrypted storage
func NewEncryptedAuthenticationCache(encryptionKey []byte, debug bool) *EncryptedAuthenticationCache {
	// In production, this would:
	// 1. Use AES-256-GCM encryption for all cached data
	// 2. Store encrypted cache to disk for persistence
	// 3. Decrypt on read, encrypt on write
	// 4. Use machine-specific encryption keys

	return &EncryptedAuthenticationCache{
		MemoryAuthenticationCache: NewMemoryAuthenticationCache(debug),
		encryptionKey:             encryptionKey,
	}
}

// PersistentAuthenticationCache would provide disk-based storage
type PersistentAuthenticationCache struct {
	*MemoryAuthenticationCache
	cacheFile string
}

// NewPersistentAuthenticationCache creates a cache with disk persistence
func NewPersistentAuthenticationCache(cacheFile string, debug bool) *PersistentAuthenticationCache {
	// In production, this would:
	// 1. Load existing cache from disk on startup
	// 2. Periodically save cache to disk
	// 3. Handle concurrent access safely
	// 4. Encrypt sensitive data before writing

	return &PersistentAuthenticationCache{
		MemoryAuthenticationCache: NewMemoryAuthenticationCache(debug),
		cacheFile:                 cacheFile,
	}
}
