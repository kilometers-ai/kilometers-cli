package provisioning

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// FileSystemPluginInstaller handles plugin installation to the file system
type FileSystemPluginInstaller struct {
	pluginDir string
}

// NewFileSystemPluginInstaller creates a new file system plugin installer
func NewFileSystemPluginInstaller(pluginDir string) (*FileSystemPluginInstaller, error) {
	// Expand home directory if needed
	if pluginDir[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		pluginDir = filepath.Join(home, pluginDir[2:])
	}

	// Create plugin directory if it doesn't exist
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugin directory: %w", err)
	}

	return &FileSystemPluginInstaller{
		pluginDir: pluginDir,
	}, nil
}

// InstallPlugin installs a downloaded plugin
func (i *FileSystemPluginInstaller) InstallPlugin(ctx context.Context, pluginData io.Reader, plugin domain.ProvisionedPlugin) error {
	// Create gzip reader
	gzReader, err := gzip.NewReader(pluginData)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Sanitize path
		target := filepath.Join(i.pluginDir, filepath.Clean(header.Name))
		if !filepath.HasPrefix(target, filepath.Clean(i.pluginDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Create the file
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			// Copy file contents
			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("failed to extract file: %w", err)
			}

			file.Close()

			// Make binary executable if it's the main plugin file
			if filepath.Ext(target) == "" && filepath.Base(target)[:10] == "km-plugin-" {
				if err := os.Chmod(target, 0755); err != nil {
					return fmt.Errorf("failed to make plugin executable: %w", err)
				}
			}
		}
	}

	return nil
}

// UninstallPlugin removes an installed plugin
func (i *FileSystemPluginInstaller) UninstallPlugin(ctx context.Context, pluginName string) error {
	// Find plugin files
	pattern := filepath.Join(i.pluginDir, fmt.Sprintf("km-plugin-%s-*", pluginName))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find plugin files: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	// Remove all plugin files
	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			return fmt.Errorf("failed to remove %s: %w", match, err)
		}
	}

	return nil
}

// GetInstalledPlugins returns all installed plugins
func (i *FileSystemPluginInstaller) GetInstalledPlugins(ctx context.Context) ([]domain.InstalledPlugin, error) {
	// Find all plugin binaries
	pattern := filepath.Join(i.pluginDir, "km-plugin-*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}

	var plugins []domain.InstalledPlugin

	for _, match := range matches {
		// Skip signature and manifest files
		if filepath.Ext(match) == ".sig" || filepath.Ext(match) == ".manifest" {
			continue
		}

		// Extract plugin info from filename
		// Format: km-plugin-NAME-HASH
		base := filepath.Base(match)
		if len(base) > 10 && base[:10] == "km-plugin-" {
			// Simple parsing - in production would read manifest
			info, err := os.Stat(match)
			if err != nil {
				continue
			}

			plugins = append(plugins, domain.InstalledPlugin{
				Name:        extractPluginName(base),
				Path:        match,
				InstalledAt: info.ModTime(),
				Enabled:     true,
			})
		}
	}

	return plugins, nil
}

// Helper function to extract plugin name from filename
func extractPluginName(filename string) string {
	// Remove "km-plugin-" prefix
	name := filename[10:]

	// Find last dash (before hash)
	lastDash := -1
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '-' {
			lastDash = i
			break
		}
	}

	if lastDash > 0 {
		return name[:lastDash]
	}

	return name
}
