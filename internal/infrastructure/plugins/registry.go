package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// FilePluginRegistryStore manages the plugin registry on disk
type FilePluginRegistryStore struct {
	registryPath string
	mu           sync.RWMutex
}

// NewFilePluginRegistryStore creates a new file-based registry store
func NewFilePluginRegistryStore(configDir string) (*FilePluginRegistryStore, error) {
	// Expand home directory if needed
	if configDir[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(home, configDir[2:])
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	registryPath := filepath.Join(configDir, "plugin-registry.json")

	return &FilePluginRegistryStore{
		registryPath: registryPath,
	}, nil
}

// SaveRegistry saves the plugin registry to disk
func (s *FilePluginRegistryStore) SaveRegistry(registry *domain.PluginRegistry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Marshal registry to JSON
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write to temporary file first
	tempFile := s.registryPath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	// Rename to final location (atomic on most systems)
	if err := os.Rename(tempFile, s.registryPath); err != nil {
		os.Remove(tempFile) // Clean up
		return fmt.Errorf("failed to save registry: %w", err)
	}

	return nil
}

// LoadRegistry loads the plugin registry from disk
func (s *FilePluginRegistryStore) LoadRegistry() (*domain.PluginRegistry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Read registry file
	data, err := os.ReadFile(s.registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty registry if file doesn't exist
			return &domain.PluginRegistry{
				Plugins: make(map[string]domain.InstalledPlugin),
			}, nil
		}
		return nil, fmt.Errorf("failed to read registry file: %w", err)
	}

	// Unmarshal registry
	var registry domain.PluginRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal registry: %w", err)
	}

	// Ensure plugins map is initialized
	if registry.Plugins == nil {
		registry.Plugins = make(map[string]domain.InstalledPlugin)
	}

	return &registry, nil
}

// UpdatePlugin updates a single plugin in the registry
func (s *FilePluginRegistryStore) UpdatePlugin(plugin domain.InstalledPlugin) error {
	// Load current registry
	registry, err := s.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Update plugin
	registry.Plugins[plugin.Name] = plugin

	// Save registry
	return s.SaveRegistry(registry)
}
