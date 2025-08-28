package plugininfra

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	plugindomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/plugin"
)

// FileSystemRegistry manages plugin registration state in a JSON file
type FileSystemRegistry struct {
	configDir string
	filePath  string
}

// NewFileSystemRegistry creates a new filesystem-based plugin registry
func NewFileSystemRegistry(configDir string) *FileSystemRegistry {
	return &FileSystemRegistry{
		configDir: configDir,
		filePath:  filepath.Join(configDir, "plugin-registry.json"),
	}
}

// registryData represents the persisted registry format
type registryData struct {
	Version     string                                      `json:"version"`
	LastUpdated time.Time                                   `json:"last_updated"`
	Plugins     map[string]plugindomain.PluginInstallStatus `json:"plugins"`
}

// Load loads the plugin registry
func (r *FileSystemRegistry) Load(ctx context.Context) (map[string]plugindomain.PluginInstallStatus, error) {
	// Check if file exists
	if _, err := os.Stat(r.filePath); os.IsNotExist(err) {
		// Return empty registry if file doesn't exist
		return make(map[string]plugindomain.PluginInstallStatus), nil
	}

	// Read the file
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry file: %w", err)
	}

	// Parse the JSON
	var registry registryData
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry file: %w", err)
	}

	if registry.Plugins == nil {
		registry.Plugins = make(map[string]plugindomain.PluginInstallStatus)
	}

	return registry.Plugins, nil
}

// Save saves the plugin registry
func (r *FileSystemRegistry) Save(ctx context.Context, plugins map[string]plugindomain.PluginInstallStatus) error {
	// Ensure config directory exists
	if err := os.MkdirAll(r.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create registry data
	registry := registryData{
		Version:     "1.0",
		LastUpdated: time.Now(),
		Plugins:     plugins,
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write to file atomically
	tempFile := r.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	// Rename temp file to final location
	if err := os.Rename(tempFile, r.filePath); err != nil {
		// Try to clean up temp file
		os.Remove(tempFile)
		return fmt.Errorf("failed to save registry file: %w", err)
	}

	return nil
}

// AddPlugin adds a plugin to the registry
func (r *FileSystemRegistry) AddPlugin(ctx context.Context, plugin plugindomain.Plugin, path string) error {
	// Load current registry
	plugins, err := r.Load(ctx)
	if err != nil {
		return err
	}

	// Add or update the plugin
	plugins[plugin.Name] = plugindomain.PluginInstallStatus{
		Plugin:      plugin,
		IsInstalled: true,
		LocalPath:   path,
	}

	// Save the updated registry
	return r.Save(ctx, plugins)
}

// RemovePlugin removes a plugin from the registry
func (r *FileSystemRegistry) RemovePlugin(ctx context.Context, pluginName string) error {
	// Load current registry
	plugins, err := r.Load(ctx)
	if err != nil {
		return err
	}

	// Remove the plugin
	delete(plugins, pluginName)

	// Save the updated registry
	return r.Save(ctx, plugins)
}
