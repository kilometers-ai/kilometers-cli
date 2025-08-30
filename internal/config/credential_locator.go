package config

import (
	"context"
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
	"runtime"
	"strings"
	"time"
)

// CredentialLocator safely locates and stores credentials
type CredentialLocator struct {
	searchPaths []string
	cipher      cipher.AEAD
}

// NewCredentialLocator creates a new credential locator
func NewCredentialLocator() (*CredentialLocator, error) {
	homeDir, _ := os.UserHomeDir()

	// Generate machine-specific encryption key
	key := generateMachineKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &CredentialLocator{
		searchPaths: []string{
			filepath.Join(homeDir, ".km", "credentials"),
			filepath.Join(homeDir, ".kilometers", "credentials"),
			filepath.Join(homeDir, ".config", "km", "credentials"),
			filepath.Join(homeDir, ".config", "kilometers", "credentials"),
			filepath.Join(homeDir, ".kmrc"),
			filepath.Join(homeDir, ".kilometersrc"),
		},
		cipher: gcm,
	}, nil
}

// LocateAPIKey searches for API key in secure locations
func (l *CredentialLocator) LocateAPIKey(ctx context.Context) (string, string, error) {
	// 1. Check credential files
	for _, path := range l.searchPaths {
		if key, err := l.loadKeyFromFile(path); err == nil && key != "" {
			return key, fmt.Sprintf("file:%s", path), nil
		}
	}

	// 2. Check OS keychain/credential manager
	if key, source, err := l.checkOSCredentialStore(); err == nil && key != "" {
		return key, source, nil
	}

	// 3. Check encrypted cache
	if key, err := l.loadFromEncryptedCache(); err == nil && key != "" {
		return key, "encrypted_cache", nil
	}

	// 4. Check git config (some users store API keys there)
	if key, err := l.checkGitConfig(); err == nil && key != "" {
		return key, "git_config", nil
	}

	// 5. Check SSH agent (for SSH-based auth)
	if key, err := l.checkSSHAgent(); err == nil && key != "" {
		return key, "ssh_agent", nil
	}

	return "", "", fmt.Errorf("no API key found")
}

// StoreAPIKey securely stores an API key
func (l *CredentialLocator) StoreAPIKey(ctx context.Context, key string) error {
	// Store in encrypted cache
	if err := l.saveToEncryptedCache(key); err != nil {
		return fmt.Errorf("failed to save to encrypted cache: %w", err)
	}

	// Also try to store in OS credential store
	if err := l.saveToOSCredentialStore(key); err != nil {
		// Log warning but don't fail
		fmt.Fprintf(os.Stderr, "Warning: failed to save to OS credential store: %v\n", err)
	}

	return nil
}

// loadKeyFromFile loads API key from a file
func (l *CredentialLocator) loadKeyFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(data))

	// Check if it's a JSON file
	if strings.HasPrefix(content, "{") {
		var creds map[string]string
		if err := json.Unmarshal(data, &creds); err == nil {
			if key, ok := creds["api_key"]; ok {
				return key, nil
			}
			if key, ok := creds["apiKey"]; ok {
				return key, nil
			}
		}
	}

	// Check if it's a key=value file
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "api_key=") || strings.HasPrefix(line, "API_KEY=") {
			return strings.TrimPrefix(strings.TrimPrefix(line, "api_key="), "API_KEY="), nil
		}
	}

	// If file contains just the key
	if len(content) > 20 && !strings.Contains(content, " ") && !strings.Contains(content, "\n") {
		return content, nil
	}

	return "", fmt.Errorf("no valid API key found in file")
}

// checkOSCredentialStore checks the OS-specific credential store
func (l *CredentialLocator) checkOSCredentialStore() (string, string, error) {
	switch runtime.GOOS {
	case "darwin":
		// Check macOS Keychain
		return l.checkMacOSKeychain()
	case "windows":
		// Check Windows Credential Manager
		return l.checkWindowsCredentialManager()
	case "linux":
		// Check Linux Secret Service
		return l.checkLinuxSecretService()
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// checkMacOSKeychain checks macOS Keychain for credentials
func (l *CredentialLocator) checkMacOSKeychain() (string, string, error) {
	// This is a simplified check - in production use keychain libraries
	// For now, we'll skip this and return not found
	return "", "", fmt.Errorf("macOS keychain not implemented")
}

// checkWindowsCredentialManager checks Windows Credential Manager
func (l *CredentialLocator) checkWindowsCredentialManager() (string, string, error) {
	// This is a simplified check - in production use Windows APIs
	// For now, we'll skip this and return not found
	return "", "", fmt.Errorf("Windows credential manager not implemented")
}

// checkLinuxSecretService checks Linux Secret Service
func (l *CredentialLocator) checkLinuxSecretService() (string, string, error) {
	// Check common Linux credential stores
	homeDir, _ := os.UserHomeDir()

	// Check GNOME Keyring
	gnomePath := filepath.Join(homeDir, ".local", "share", "keyrings", "kilometers.keyring")
	if _, err := os.Stat(gnomePath); err == nil {
		// This would require proper keyring parsing
		return "", "", fmt.Errorf("GNOME keyring parsing not implemented")
	}

	// For now, we'll skip this and return not found
	return "", "", fmt.Errorf("Linux secret service not implemented")
}

// saveToOSCredentialStore saves to the OS-specific credential store
func (l *CredentialLocator) saveToOSCredentialStore(key string) error {
	// This is a placeholder - implement OS-specific storage
	return nil
}

// loadFromEncryptedCache loads API key from encrypted cache
func (l *CredentialLocator) loadFromEncryptedCache() (string, error) {
	cachePath, err := l.getEncryptedCachePath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return "", err
	}

	// Decrypt the data
	decrypted, err := l.decrypt(data)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt cache: %w", err)
	}

	var cache struct {
		APIKey    string    `json:"api_key"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	if err := json.Unmarshal(decrypted, &cache); err != nil {
		return "", fmt.Errorf("failed to parse cache: %w", err)
	}

	// Check if cache is not too old (30 days)
	if time.Since(cache.UpdatedAt) > 30*24*time.Hour {
		return "", fmt.Errorf("cached credentials expired")
	}

	return cache.APIKey, nil
}

// saveToEncryptedCache saves API key to encrypted cache
func (l *CredentialLocator) saveToEncryptedCache(key string) error {
	cachePath, err := l.getEncryptedCachePath()
	if err != nil {
		return err
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(cachePath), 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := struct {
		APIKey    string    `json:"api_key"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		APIKey:    key,
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Encrypt the data
	encrypted, err := l.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt cache: %w", err)
	}

	// Write with restricted permissions
	if err := os.WriteFile(cachePath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	return nil
}

// getEncryptedCachePath returns the path to the encrypted cache
func (l *CredentialLocator) getEncryptedCachePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".km", ".credentials.enc"), nil
}

// checkGitConfig checks git config for API keys
func (l *CredentialLocator) checkGitConfig() (string, error) {
	// Check git config for kilometers.apikey
	// This is a simplified check
	return "", fmt.Errorf("git config not implemented")
}

// checkSSHAgent checks SSH agent for SSH-based auth
func (l *CredentialLocator) checkSSHAgent() (string, error) {
	// Check if SSH agent has kilometers-specific keys
	// This is a simplified check
	return "", fmt.Errorf("SSH agent not implemented")
}

// encrypt encrypts data using AES-GCM
func (l *CredentialLocator) encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, l.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := l.cipher.Seal(nonce, nonce, plaintext, nil)
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

// decrypt decrypts data using AES-GCM
func (l *CredentialLocator) decrypt(ciphertext []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(string(ciphertext))
	if err != nil {
		return nil, err
	}

	nonceSize := l.cipher.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return l.cipher.Open(nil, nonce, ciphertext, nil)
}

// generateMachineKey generates a machine-specific encryption key
func generateMachineKey() []byte {
	// Combine various machine-specific identifiers
	hostname, _ := os.Hostname()
	homeDir, _ := os.UserHomeDir()

	h := sha256.New()
	h.Write([]byte(hostname))
	h.Write([]byte(homeDir))
	h.Write([]byte(runtime.GOOS))
	h.Write([]byte(runtime.GOARCH))

	// Add more entropy if available
	if machineID, err := os.ReadFile("/etc/machine-id"); err == nil {
		h.Write(machineID)
	}

	return h.Sum(nil)
}
