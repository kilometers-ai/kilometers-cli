package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/http"
)

// ManifestCache handles caching of plugin manifests
type ManifestCache struct {
	cacheDir string
	ttl      time.Duration
	mutex    sync.RWMutex
	debug    bool
}

// CachedManifest represents a cached manifest with metadata
type CachedManifest struct {
	Manifest  *http.PluginManifestResponse `json:"manifest"`
	CachedAt  time.Time                    `json:"cached_at"`
	ExpiresAt time.Time                    `json:"expires_at"`
	APIKey    string                       `json:"api_key_hash"` // Hashed for security
}

// NewManifestCache creates a new manifest cache
func NewManifestCache(cacheDir string, ttl time.Duration, debug bool) (*ManifestCache, error) {
	// Expand path and create directory
	expandedDir := ExpandPath(cacheDir)
	if err := os.MkdirAll(expandedDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &ManifestCache{
		cacheDir: expandedDir,
		ttl:      ttl,
		debug:    debug,
	}, nil
}

// Get retrieves a cached manifest if valid
func (c *ManifestCache) Get(apiKeyHash string) (*http.PluginManifestResponse, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cacheFile := c.getCacheFilePath(apiKeyHash)

	// Check if cache file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		if c.debug {
			fmt.Printf("[ManifestCache] Cache miss: file not found for key %s\n", apiKeyHash[:8])
		}
		return nil, false
	}

	// Read cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if c.debug {
			fmt.Printf("[ManifestCache] Cache read error: %v\n", err)
		}
		return nil, false
	}

	// Parse cached manifest
	var cached CachedManifest
	if err := json.Unmarshal(data, &cached); err != nil {
		if c.debug {
			fmt.Printf("[ManifestCache] Cache parse error: %v\n", err)
		}
		return nil, false
	}

	// Check if cache is expired
	if time.Now().After(cached.ExpiresAt) {
		if c.debug {
			fmt.Printf("[ManifestCache] Cache expired for key %s\n", apiKeyHash[:8])
		}
		// Remove expired cache file
		os.Remove(cacheFile)
		return nil, false
	}

	// Verify API key hash matches
	if cached.APIKey != apiKeyHash {
		if c.debug {
			fmt.Printf("[ManifestCache] API key mismatch for cached manifest\n")
		}
		return nil, false
	}

	if c.debug {
		fmt.Printf("[ManifestCache] Cache hit for key %s (expires in %v)\n",
			apiKeyHash[:8], cached.ExpiresAt.Sub(time.Now()))
	}

	return cached.Manifest, true
}

// Set stores a manifest in the cache
func (c *ManifestCache) Set(apiKeyHash string, manifest *http.PluginManifestResponse) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	cached := CachedManifest{
		Manifest:  manifest,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
		APIKey:    apiKeyHash,
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cached manifest: %w", err)
	}

	// Write to cache file
	cacheFile := c.getCacheFilePath(apiKeyHash)
	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	if c.debug {
		fmt.Printf("[ManifestCache] Cached manifest for key %s (expires at %s)\n",
			apiKeyHash[:8], cached.ExpiresAt.Format(time.RFC3339))
	}

	return nil
}

// Clear removes cached manifest for a specific API key
func (c *ManifestCache) Clear(apiKeyHash string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cacheFile := c.getCacheFilePath(apiKeyHash)
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	if c.debug {
		fmt.Printf("[ManifestCache] Cleared cache for key %s\n", apiKeyHash[:8])
	}

	return nil
}

// ClearAll removes all cached manifests
func (c *ManifestCache) ClearAll() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Read cache directory
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	var removed int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only remove manifest cache files
		if filepath.Ext(entry.Name()) == ".json" &&
			entry.Name() != "plugin_registry.json" {
			fullPath := filepath.Join(c.cacheDir, entry.Name())
			if err := os.Remove(fullPath); err != nil {
				if c.debug {
					fmt.Printf("[ManifestCache] Failed to remove %s: %v\n", entry.Name(), err)
				}
			} else {
				removed++
			}
		}
	}

	if c.debug {
		fmt.Printf("[ManifestCache] Cleared %d cache files\n", removed)
	}

	return nil
}

// CleanExpired removes expired cache entries
func (c *ManifestCache) CleanExpired() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	now := time.Now()
	var removed int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check manifest cache files
		if filepath.Ext(entry.Name()) == ".json" &&
			entry.Name() != "plugin_registry.json" {

			fullPath := filepath.Join(c.cacheDir, entry.Name())

			// Read and check expiration
			data, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}

			var cached CachedManifest
			if err := json.Unmarshal(data, &cached); err != nil {
				continue
			}

			// Remove if expired
			if now.After(cached.ExpiresAt) {
				if err := os.Remove(fullPath); err == nil {
					removed++
					if c.debug {
						fmt.Printf("[ManifestCache] Removed expired cache: %s\n", entry.Name())
					}
				}
			}
		}
	}

	if c.debug && removed > 0 {
		fmt.Printf("[ManifestCache] Cleaned %d expired cache entries\n", removed)
	}

	return nil
}

// GetCacheInfo returns information about the cache
func (c *ManifestCache) GetCacheInfo() (CacheInfo, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return CacheInfo{}, fmt.Errorf("failed to read cache directory: %w", err)
	}

	var info CacheInfo
	info.TTL = c.ttl
	info.CacheDir = c.cacheDir

	now := time.Now()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) == ".json" &&
			entry.Name() != "plugin_registry.json" {

			fullPath := filepath.Join(c.cacheDir, entry.Name())

			// Read cache file
			data, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}

			var cached CachedManifest
			if err := json.Unmarshal(data, &cached); err != nil {
				continue
			}

			info.TotalEntries++

			if now.After(cached.ExpiresAt) {
				info.ExpiredEntries++
			} else {
				info.ValidEntries++
			}
		}
	}

	return info, nil
}

// getCacheFilePath returns the cache file path for an API key hash
func (c *ManifestCache) getCacheFilePath(apiKeyHash string) string {
	// Use up to 16 chars of hash for filename to avoid collisions
	hashPart := apiKeyHash
	if len(apiKeyHash) > 16 {
		hashPart = apiKeyHash[:16]
	}
	filename := fmt.Sprintf("manifest_%s.json", hashPart)
	return filepath.Join(c.cacheDir, filename)
}

// CacheInfo contains information about the cache state
type CacheInfo struct {
	TTL            time.Duration `json:"ttl"`
	CacheDir       string        `json:"cache_dir"`
	TotalEntries   int           `json:"total_entries"`
	ValidEntries   int           `json:"valid_entries"`
	ExpiredEntries int           `json:"expired_entries"`
}

// CachedPluginApiClient wraps PluginApiClient with caching
type CachedPluginApiClient struct {
	*http.PluginApiClient
	cache      *ManifestCache
	apiKeyHash string
	debug      bool
}

// NewCachedPluginApiClient creates a plugin API client with manifest caching
func NewCachedPluginApiClient(cacheDir string, cacheTTL time.Duration, debug bool, cliVersion string) (*CachedPluginApiClient, error) {
	// Create base API client
	apiClient, err := http.NewPluginApiClient(debug, cliVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Create cache
	cache, err := NewManifestCache(cacheDir, cacheTTL, debug)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	// Create API key hash for cache key
	// In production, this should use a proper hash function
	apiKeyHash := fmt.Sprintf("%x", []byte("default"))

	return &CachedPluginApiClient{
		PluginApiClient: apiClient,
		cache:           cache,
		apiKeyHash:      apiKeyHash,
		debug:           debug,
	}, nil
}

// GetPluginManifest retrieves manifest with caching (uses POST endpoint)
func (c *CachedPluginApiClient) GetPluginManifest(ctx context.Context) (*http.PluginManifestResponse, error) {
	// For backwards compatibility, call PostPluginManifest with empty installed plugins
	return c.PostPluginManifest(ctx, []http.InstalledPluginInfo{})
}

// PostPluginManifest retrieves manifest with caching using the POST endpoint
func (c *CachedPluginApiClient) PostPluginManifest(ctx context.Context, installedPlugins []http.InstalledPluginInfo) (*http.PluginManifestResponse, error) {
	// Try cache first
	if manifest, found := c.cache.Get(c.apiKeyHash); found {
		return manifest, nil
	}

	// Cache miss - fetch from API
	if c.debug {
		fmt.Println("[CachedPluginApiClient] Cache miss, fetching from API")
	}

	manifest, err := c.PluginApiClient.PostPluginManifest(ctx, installedPlugins)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := c.cache.Set(c.apiKeyHash, manifest); err != nil {
		// Log error but don't fail the request
		if c.debug {
			fmt.Printf("[CachedPluginApiClient] Failed to cache manifest: %v\n", err)
		}
	}

	return manifest, nil
}

// ClearCache clears the manifest cache
func (c *CachedPluginApiClient) ClearCache() error {
	return c.cache.Clear(c.apiKeyHash)
}

// GetCacheInfo returns cache information
func (c *CachedPluginApiClient) GetCacheInfo() (CacheInfo, error) {
	return c.cache.GetCacheInfo()
}
