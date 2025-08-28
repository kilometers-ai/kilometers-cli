package plugininfra

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	plugindomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/plugin"
)

// FileSystemInstaller handles plugin installation to the filesystem
type FileSystemInstaller struct {
	targetDir string
	debug     bool
}

// NewFileSystemInstaller creates a new filesystem plugin installer
func NewFileSystemInstaller(targetDir string, debug bool) *FileSystemInstaller {
	return &FileSystemInstaller{
		targetDir: expandPath(targetDir),
		debug:     debug,
	}
}

// Install installs a plugin binary to the plugins directory
func (i *FileSystemInstaller) Install(ctx context.Context, plugin plugindomain.Plugin, data []byte) error {
	// Ensure target directory exists
	if err := os.MkdirAll(i.targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Verify checksum if provided
	if plugin.Checksum != "" {
		if err := i.verifyChecksum(data, plugin.Checksum); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// Determine if data is tar.gz or raw binary
	if isGzip(data) {
		return i.installFromTarGz(plugin.Name, data)
	}

	// Install as raw binary
	binaryName := fmt.Sprintf("km-plugin-%s", plugin.Name)
	binaryPath := filepath.Join(i.targetDir, binaryName)

	if err := os.WriteFile(binaryPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write plugin binary: %w", err)
	}

	if i.debug {
		fmt.Printf("[DEBUG] Installed plugin %s to %s\n", plugin.Name, binaryPath)
	}

	return nil
}

// Uninstall removes a plugin from the system
func (i *FileSystemInstaller) Uninstall(ctx context.Context, pluginName string) error {
	// Try different naming conventions
	patterns := []string{
		fmt.Sprintf("km-plugin-%s", pluginName),
		fmt.Sprintf("km-plugin-%s-*", pluginName),
		pluginName,
	}

	found := false
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(i.targetDir, pattern))
		if err != nil {
			continue
		}

		for _, match := range matches {
			if err := os.Remove(match); err != nil {
				return fmt.Errorf("failed to remove plugin file %s: %w", match, err)
			}
			found = true
			if i.debug {
				fmt.Printf("[DEBUG] Removed plugin file: %s\n", match)
			}
		}
	}

	if !found {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	return nil
}

// GetInstalled returns list of installed plugins
func (i *FileSystemInstaller) GetInstalled(ctx context.Context) ([]plugindomain.PluginInstallStatus, error) {
	var installed []plugindomain.PluginInstallStatus

	// List all files in the plugins directory
	entries, err := os.ReadDir(i.targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return installed, nil // Empty list if directory doesn't exist
		}
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check if it's a plugin binary
		if !strings.HasPrefix(name, "km-plugin-") && !strings.Contains(name, "plugin") {
			continue
		}

		// Extract plugin name
		pluginName := strings.TrimPrefix(name, "km-plugin-")
		pluginName = strings.TrimSuffix(pluginName, filepath.Ext(pluginName))

		// Get file info
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Create install status
		status := plugindomain.PluginInstallStatus{
			Plugin: plugindomain.Plugin{
				Name: pluginName,
				// Version will be populated by the registry
			},
			IsInstalled: true,
			LocalPath:   filepath.Join(i.targetDir, name),
		}

		// Check if file is executable
		if info.Mode()&0111 == 0 {
			if i.debug {
				fmt.Printf("[DEBUG] Plugin %s is not executable\n", pluginName)
			}
		}

		installed = append(installed, status)
	}

	return installed, nil
}

// CheckForUpdates checks if any installed plugins have updates
func (i *FileSystemInstaller) CheckForUpdates(ctx context.Context, availablePlugins []plugindomain.Plugin) ([]plugindomain.PluginInstallStatus, error) {
	installed, err := i.GetInstalled(ctx)
	if err != nil {
		return nil, err
	}

	var updates []plugindomain.PluginInstallStatus

	// Create a map of available plugins for quick lookup
	availableMap := make(map[string]plugindomain.Plugin)
	for _, plugin := range availablePlugins {
		availableMap[plugin.Name] = plugin
	}

	// Check each installed plugin
	for _, inst := range installed {
		if available, exists := availableMap[inst.Plugin.Name]; exists {
			// Simple version comparison (would need semantic versioning in production)
			if inst.Plugin.Version != available.Version {
				inst.NeedsUpdate = true
				inst.CurrentVersion = inst.Plugin.Version
				inst.Plugin = available // Update to new plugin info
				updates = append(updates, inst)
			}
		}
	}

	return updates, nil
}

// installFromTarGz extracts and installs from a tar.gz archive
func (i *FileSystemInstaller) installFromTarGz(pluginName string, data []byte) error {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct target path
		targetPath := filepath.Join(i.targetDir, header.Name)

		// Security: prevent path traversal
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(i.targetDir)) {
			return fmt.Errorf("unsafe tar path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			file.Close()

			if i.debug {
				fmt.Printf("[DEBUG] Extracted: %s\n", targetPath)
			}
		}
	}

	return nil
}

// verifyChecksum verifies the downloaded plugin checksum
func (i *FileSystemInstaller) verifyChecksum(data []byte, expectedChecksum string) error {
	hash := sha256.Sum256(data)
	actualChecksum := hex.EncodeToString(hash[:])

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// isGzip checks if data is gzip compressed
func isGzip(data []byte) bool {
	return len(data) > 2 && data[0] == 0x1f && data[1] == 0x8b
}

// expandPath expands ~ in paths
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(homeDir, path[2:])
		}
	}
	return path
}
