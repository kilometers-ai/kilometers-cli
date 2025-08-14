package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSystemPluginDiscovery implements plugin discovery by scanning directories
type FileSystemPluginDiscovery struct {
	directories []string
	debug       bool
}

// NewFileSystemPluginDiscovery creates a new filesystem-based plugin discovery
func NewFileSystemPluginDiscovery(directories []string, debug bool) *FileSystemPluginDiscovery {
	return &FileSystemPluginDiscovery{
		directories: directories,
		debug:       debug,
	}
}

// DiscoverPlugins searches for plugin binaries in configured directories
func (d *FileSystemPluginDiscovery) DiscoverPlugins(ctx context.Context) ([]PluginInfo, error) {
	var discoveredPlugins []PluginInfo

	for _, dir := range d.directories {
		// Expand user home directory
		expandedDir := expandPath(dir)

		if d.debug {
			fmt.Printf("[PluginDiscovery] Scanning directory: %s\n", expandedDir)
		}

		// Check if directory exists, create if needed
		if _, err := os.Stat(expandedDir); os.IsNotExist(err) {
			if d.debug {
				fmt.Printf("[PluginDiscovery] Creating plugins directory: %s\n", expandedDir)
			}
			if err := os.MkdirAll(expandedDir, 0755); err != nil {
				if d.debug {
					fmt.Printf("[PluginDiscovery] Failed to create directory %s: %v\n", expandedDir, err)
				}
				continue
			}
		}

		// Scan directory for plugin binaries
		plugins, err := d.scanDirectory(ctx, expandedDir)
		if err != nil {
			if d.debug {
				fmt.Printf("[PluginDiscovery] Error scanning directory %s: %v\n", expandedDir, err)
			}
			continue
		}

		discoveredPlugins = append(discoveredPlugins, plugins...)
	}

	if d.debug {
		fmt.Printf("[PluginDiscovery] Discovered %d plugins\n", len(discoveredPlugins))
	}

	return discoveredPlugins, nil
}

// ValidatePlugin checks if a plugin binary is valid and extracts metadata
func (d *FileSystemPluginDiscovery) ValidatePlugin(ctx context.Context, pluginPath string) (*PluginInfo, error) {
	// Check if file exists and is executable
	fileInfo, err := os.Stat(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("plugin file not found: %w", err)
	}

	// Check if file is executable
	if fileInfo.Mode()&0111 == 0 {
		return nil, fmt.Errorf("plugin file is not executable: %s", pluginPath)
	}

	// Look for manifest file
	manifestPath := getManifestPath(pluginPath)
	manifest, err := d.loadManifest(manifestPath)
	if err != nil {
		if d.debug {
			fmt.Printf("[PluginDiscovery] Warning: Could not load manifest for %s: %v\n", pluginPath, err)
		}

		// Use default values if manifest is missing
		manifest = &PluginManifest{
			Name:         extractPluginNameFromPath(pluginPath),
			Version:      "unknown",
			Description:  "Plugin without manifest",
			RequiredTier: "Free",
		}
	}

	return &PluginInfo{
		Name:         manifest.Name,
		Version:      manifest.Version,
		Path:         pluginPath,
		RequiredTier: manifest.RequiredTier,
		Signature:    nil, // Will be loaded by validator
	}, nil
}

// scanDirectory scans a single directory for plugin binaries
func (d *FileSystemPluginDiscovery) scanDirectory(ctx context.Context, dirPath string) ([]PluginInfo, error) {
	var foundPlugins []PluginInfo

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		// Skip directories and non-executable files
		if entry.IsDir() {
			continue
		}

		// Look for plugin binaries (km-plugin-* pattern)
		if !strings.HasPrefix(entry.Name(), "km-plugin-") {
			continue
		}

		pluginPath := filepath.Join(dirPath, entry.Name())

		// Validate and extract plugin info
		pluginInfo, err := d.ValidatePlugin(ctx, pluginPath)
		if err != nil {
			if d.debug {
				fmt.Printf("[PluginDiscovery] Invalid plugin %s: %v\n", pluginPath, err)
			}
			continue
		}

		foundPlugins = append(foundPlugins, *pluginInfo)

		if d.debug {
			fmt.Printf("[PluginDiscovery] Found plugin: %s v%s at %s\n",
				pluginInfo.Name, pluginInfo.Version, pluginPath)
		}
	}

	return foundPlugins, nil
}

// loadManifest loads plugin manifest from JSON file
func (d *FileSystemPluginDiscovery) loadManifest(manifestPath string) (*PluginManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	return &manifest, nil
}

// SimplePluginValidator implements basic plugin validation
type SimplePluginValidator struct {
	debug bool
}

// NewSimplePluginValidator creates a basic plugin validator
func NewSimplePluginValidator(debug bool) *SimplePluginValidator {
	return &SimplePluginValidator{debug: debug}
}

// ValidateSignature performs basic signature validation
func (v *SimplePluginValidator) ValidateSignature(ctx context.Context, pluginPath string, signature []byte) error {
	// For now, we'll perform basic validation
	// In production, this would verify cryptographic signatures

	if v.debug {
		fmt.Printf("[PluginValidator] Validating signature for %s\n", pluginPath)
	}

	// Check if plugin binary exists and is executable
	fileInfo, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("plugin binary not found: %w", err)
	}

	if fileInfo.Mode()&0111 == 0 {
		return fmt.Errorf("plugin binary is not executable")
	}

	// TODO: Implement actual signature verification
	// This would involve:
	// 1. Loading public key
	// 2. Verifying binary signature
	// 3. Checking certificate chain

	return nil
}

// GetPluginManifest extracts metadata from a plugin
func (v *SimplePluginValidator) GetPluginManifest(ctx context.Context, pluginPath string) (*PluginManifest, error) {
	manifestPath := getManifestPath(pluginPath)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		// Use defaults if manifest doesn't exist
		return &PluginManifest{
			Name:         extractPluginNameFromPath(pluginPath),
			Version:      "unknown",
			Description:  "Plugin without manifest",
			RequiredTier: "Free",
		}, nil
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// Helper functions

// expandPath expands ~ to user home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path // Return original if can't get home dir
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// getManifestPath returns the expected manifest path for a plugin binary
func getManifestPath(pluginPath string) string {
	dir := filepath.Dir(pluginPath)
	base := filepath.Base(pluginPath)

	// Remove km-plugin- prefix if present
	name := strings.TrimPrefix(base, "km-plugin-")

	return filepath.Join(dir, name+".manifest.json")
}

// extractPluginNameFromPath extracts plugin name from file path
func extractPluginNameFromPath(pluginPath string) string {
	base := filepath.Base(pluginPath)

	// Remove km-plugin- prefix if present
	name := strings.TrimPrefix(base, "km-plugin-")

	// Remove any file extension
	if idx := strings.LastIndex(name, "."); idx != -1 {
		name = name[:idx]
	}

	return name
}
