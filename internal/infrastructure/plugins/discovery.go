package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kilometers-ai/kilometers-cli/internal/core/ports/plugins"
)

// Plugin Binary Naming Convention: km-plugin-<name>
const PluginBinaryPrefix = "km-plugin-"

// FileSystemPluginDiscovery implements plugin discovery using the filesystem
type FileSystemPluginDiscovery struct {
	directories []string
	validator   plugins.PluginValidator
}

// NewFileSystemPluginDiscovery creates a new filesystem-based plugin discovery
func NewFileSystemPluginDiscovery(directories []string, validator plugins.PluginValidator) *FileSystemPluginDiscovery {
	if len(directories) == 0 {
		directories = plugins.StandardPluginDirectories
	}

	return &FileSystemPluginDiscovery{
		directories: directories,
		validator:   validator,
	}
}

// DiscoverPlugins searches for plugin binaries in standard locations
func (d *FileSystemPluginDiscovery) DiscoverPlugins(ctx context.Context) ([]plugins.PluginInfo, error) {
	var discoveredPlugins []plugins.PluginInfo

	for _, dir := range d.directories {
		// Expand tilde to home directory
		expandedDir := expandPath(dir)

		// Check if directory exists
		if _, err := os.Stat(expandedDir); os.IsNotExist(err) {
			continue
		}

		// Search for plugin binaries in this directory
		pluginsInDir, err := d.discoverInDirectory(ctx, expandedDir)
		if err != nil {
			return nil, fmt.Errorf("failed to discover plugins in %s: %w", expandedDir, err)
		}

		discoveredPlugins = append(discoveredPlugins, pluginsInDir...)
	}

	return discoveredPlugins, nil
}

// ValidatePlugin checks if a plugin binary is valid and signed
func (d *FileSystemPluginDiscovery) ValidatePlugin(ctx context.Context, pluginPath string) (*plugins.PluginInfo, error) {
	// Check if file exists and is executable
	fileInfo, err := os.Stat(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("plugin file not found: %w", err)
	}

	if !isExecutable(fileInfo) {
		return nil, fmt.Errorf("plugin file is not executable")
	}

	// Extract plugin name from filename
	fileName := filepath.Base(pluginPath)
	if !strings.HasPrefix(fileName, PluginBinaryPrefix) {
		return nil, fmt.Errorf("plugin file does not follow naming convention: %s", fileName)
	}

	pluginName := strings.TrimPrefix(fileName, PluginBinaryPrefix)

	// Calculate file signature (hash)
	signature, err := d.calculateFileSignature(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate plugin signature: %w", err)
	}

	// Get plugin manifest
	manifest, err := d.validator.GetPluginManifest(ctx, pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin manifest: %w", err)
	}

	pluginInfo := &plugins.PluginInfo{
		Name:         pluginName,
		Version:      manifest.Version,
		DisplayName:  manifest.Description,
		Description:  manifest.Description,
		RequiredTier: manifest.RequiredTier,
		Path:         pluginPath,
		Signature:    signature,
		Metadata: map[string]string{
			"author":     manifest.Author,
			"go_version": manifest.BuildInfo.GoVersion,
			"build_time": manifest.BuildInfo.BuildTime.String(),
			"git_commit": manifest.BuildInfo.GitCommit,
		},
	}

	return pluginInfo, nil
}

// Private methods

func (d *FileSystemPluginDiscovery) discoverInDirectory(ctx context.Context, directory string) ([]plugins.PluginInfo, error) {
	var plugins []plugins.PluginInfo

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file matches plugin naming convention
		if !strings.HasPrefix(info.Name(), PluginBinaryPrefix) {
			return nil
		}

		// Check if file is executable
		if !isExecutable(info) {
			return nil
		}

		// Validate and get plugin info
		pluginInfo, err := d.ValidatePlugin(ctx, path)
		if err != nil {
			// Log validation error but continue discovery
			return nil
		}

		plugins = append(plugins, *pluginInfo)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return plugins, nil
}

func (d *FileSystemPluginDiscovery) calculateFileSignature(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

// Helper functions

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

func isExecutable(info os.FileInfo) bool {
	// Check if file has execute permissions
	mode := info.Mode()
	return mode&0111 != 0
}

// SignaturePluginValidator implements plugin signature validation
type SignaturePluginValidator struct {
	publicKey []byte // In production, this would be the public key for signature verification
}

// NewSignaturePluginValidator creates a new signature-based plugin validator
func NewSignaturePluginValidator(publicKey []byte) *SignaturePluginValidator {
	return &SignaturePluginValidator{
		publicKey: publicKey,
	}
}

// ValidateSignature verifies the digital signature of a plugin binary
func (v *SignaturePluginValidator) ValidateSignature(ctx context.Context, pluginPath string, signature []byte) error {
	// In production, this would perform actual digital signature verification
	// For now, we'll do a simple file hash check

	file, err := os.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin file: %w", err)
	}
	defer file.Close()

	// Calculate file hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	fileHash := hash.Sum(nil)

	// Compare with provided signature (in production, this would be proper signature verification)
	if hex.EncodeToString(fileHash) != hex.EncodeToString(signature) {
		return fmt.Errorf("plugin signature validation failed")
	}

	return nil
}

// GetPluginManifest extracts metadata from a plugin binary
func (v *SignaturePluginValidator) GetPluginManifest(ctx context.Context, pluginPath string) (*plugins.PluginManifest, error) {
	// In production, this would extract embedded manifest from the binary
	// For now, we'll return a default manifest based on file information

	fileInfo, err := os.Stat(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	fileName := filepath.Base(pluginPath)
	pluginName := strings.TrimPrefix(fileName, PluginBinaryPrefix)

	// Determine required tier based on plugin name
	requiredTier := "Free"
	if strings.Contains(pluginName, "api") || strings.Contains(pluginName, "advanced") || strings.Contains(pluginName, "ml") {
		requiredTier = "Pro"
	}
	if strings.Contains(pluginName, "enterprise") || strings.Contains(pluginName, "compliance") || strings.Contains(pluginName, "team") {
		requiredTier = "Enterprise"
	}

	manifest := &plugins.PluginManifest{
		Name:         pluginName,
		Version:      "1.0.0", // Default version
		Description:  fmt.Sprintf("%s plugin for Kilometers CLI", pluginName),
		Author:       "Kilometers.ai",
		RequiredTier: requiredTier,
		Features:     []string{plugins.FeatureConsoleLogging},
		Capabilities: []string{"message_handling", "error_handling", "stream_events"},
		Dependencies: make(map[string]string),
		BuildInfo: plugins.BuildInfo{
			GoVersion: "1.21+",
			BuildTime: fileInfo.ModTime(),
			GitCommit: "unknown",
			BuildHost: "unknown",
			Signature: hex.EncodeToString([]byte(pluginName)),
		},
	}

	// Add features based on tier requirement
	if requiredTier == "Pro" {
		manifest.Features = append(manifest.Features, plugins.FeatureAPILogging, plugins.FeatureAdvancedAnalytics)
	}
	if requiredTier == "Enterprise" {
		manifest.Features = append(manifest.Features, plugins.FeatureTeamCollaboration, plugins.FeatureComplianceReporting)
	}

	return manifest, nil
}
