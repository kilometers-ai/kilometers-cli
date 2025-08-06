package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// SecureFileTokenCache implements a file-based token cache with encryption
type SecureFileTokenCache struct {
	cacheFile  string
	encryptKey []byte
	mu         sync.RWMutex
}

// NewSecureFileTokenCache creates a new secure file-based token cache
func NewSecureFileTokenCache(cacheDir string) (*SecureFileTokenCache, error) {
	// Expand home directory if needed
	if cacheDir[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cacheDir = filepath.Join(home, cacheDir[2:])
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cacheFile := filepath.Join(cacheDir, ".auth_cache")

	// Generate encryption key from machine-specific data
	encryptKey := generateEncryptionKey()

	return &SecureFileTokenCache{
		cacheFile:  cacheFile,
		encryptKey: encryptKey,
	}, nil
}

// GetToken retrieves a cached token for the given scope
func (c *SecureFileTokenCache) GetToken(scope string) (*domain.AuthToken, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Read cache file
	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cache exists
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Decrypt data
	decrypted, err := c.decrypt(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt cache: %w", err)
	}

	// Unmarshal cache
	var cache domain.AuthCache
	if err := json.Unmarshal(decrypted, &cache); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	// Get token for scope
	token := cache.GetToken(scope)
	if token == nil {
		return nil, nil
	}

	// Return the token even if expired - the caller will check expiration
	return token, nil
}

// SetToken stores a token for the given scope
func (c *SecureFileTokenCache) SetToken(scope string, token *domain.AuthToken) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Read existing cache
	var cache domain.AuthCache

	data, err := os.ReadFile(c.cacheFile)
	if err == nil {
		decrypted, err := c.decrypt(data)
		if err == nil {
			json.Unmarshal(decrypted, &cache)
		}
	}

	// Update cache
	cache.SetToken(scope, token)

	// Clean up expired tokens
	cache.ClearExpiredTokens()

	// Marshal cache
	cacheData, err := json.Marshal(&cache)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Encrypt data
	encrypted, err := c.encrypt(cacheData)
	if err != nil {
		return fmt.Errorf("failed to encrypt cache: %w", err)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(c.cacheFile, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// RemoveToken removes a token for the given scope
func (c *SecureFileTokenCache) RemoveToken(scope string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Read existing cache
	var cache domain.AuthCache

	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to remove
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	decrypted, err := c.decrypt(data)
	if err != nil {
		return fmt.Errorf("failed to decrypt cache: %w", err)
	}

	if err := json.Unmarshal(decrypted, &cache); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	// Remove token
	if cache.Tokens != nil {
		delete(cache.Tokens, scope)
	}

	// Marshal cache
	cacheData, err := json.Marshal(&cache)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Encrypt data
	encrypted, err := c.encrypt(cacheData)
	if err != nil {
		return fmt.Errorf("failed to encrypt cache: %w", err)
	}

	// Write to file
	if err := os.WriteFile(c.cacheFile, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Clear removes all cached tokens
func (c *SecureFileTokenCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove cache file
	if err := os.Remove(c.cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	return nil
}

// Private encryption methods

func (c *SecureFileTokenCache) encrypt(data []byte) ([]byte, error) {
	// Create cipher
	block, err := aes.NewCipher(c.encryptKey)
	if err != nil {
		return nil, err
	}

	// GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	// Encode to base64 for storage
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

func (c *SecureFileTokenCache) decrypt(data []byte) ([]byte, error) {
	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	// Create cipher
	block, err := aes.NewCipher(c.encryptKey)
	if err != nil {
		return nil, err
	}

	// GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// generateEncryptionKey generates a machine-specific encryption key
func generateEncryptionKey() []byte {
	// In production, this would use more sophisticated machine fingerprinting
	// For now, use a combination of hostname and user
	hostname, _ := os.Hostname()
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME") // Windows
	}

	// Create key material
	keyMaterial := fmt.Sprintf("kilometers-cli:%s:%s", hostname, user)

	// Generate 32-byte key using SHA256
	hash := sha256.Sum256([]byte(keyMaterial))
	return hash[:]
}

// MemoryTokenCache implements an in-memory token cache (for testing)
type MemoryTokenCache struct {
	tokens map[string]*domain.AuthToken
	mu     sync.RWMutex
}

// NewMemoryTokenCache creates a new in-memory token cache
func NewMemoryTokenCache() *MemoryTokenCache {
	return &MemoryTokenCache{
		tokens: make(map[string]*domain.AuthToken),
	}
}

// GetToken retrieves a cached token for the given scope
func (c *MemoryTokenCache) GetToken(scope string) (*domain.AuthToken, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	token, exists := c.tokens[scope]
	if !exists {
		return nil, nil
	}

	// Return the token even if expired - the caller will check expiration
	return token, nil
}

// SetToken stores a token for the given scope
func (c *MemoryTokenCache) SetToken(scope string, token *domain.AuthToken) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokens[scope] = token
	return nil
}

// RemoveToken removes a token for the given scope
func (c *MemoryTokenCache) RemoveToken(scope string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.tokens, scope)
	return nil
}

// Clear removes all cached tokens
func (c *MemoryTokenCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokens = make(map[string]*domain.AuthToken)
	return nil
}
