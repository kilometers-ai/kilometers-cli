package plugins

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstalledPluginInfo represents information about an installed plugin binary
type InstalledPluginInfo struct {
	Name string
	Path string
	Size int64
	Executable bool
}

// FindInstalledPlugin looks for an installed plugin binary in the specified directory
// Returns the plugin info if found and executable, otherwise returns an error
func FindInstalledPlugin(pluginName string, pluginsDir string) (*InstalledPluginInfo, error) {
	// Expand the plugins directory path
	expandedDir := ExpandPath(pluginsDir)
	
	// Use standard naming convention: km-plugin-{name}
	binaryName := fmt.Sprintf("km-plugin-%s", pluginName)
	pluginPath := filepath.Join(expandedDir, binaryName)
	
	// Check if plugin file exists
	info, err := os.Stat(pluginPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("plugin '%s' not found at %s", pluginName, pluginPath)
		}
		return nil, fmt.Errorf("failed to check plugin file: %w", err)
	}
	
	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("plugin '%s' exists but is not a regular file", pluginName)
	}
	
	// Check if it's executable
	isExecutable := info.Mode()&0111 != 0
	if !isExecutable {
		return nil, fmt.Errorf("plugin '%s' exists but is not executable", pluginName)
	}
	
	return &InstalledPluginInfo{
		Name: pluginName,
		Path: pluginPath,
		Size: info.Size(),
		Executable: isExecutable,
	}, nil
}

// GetPluginBinaryPath returns the expected path for a plugin binary
// This is a utility function that follows the standard naming convention
func GetPluginBinaryPath(pluginName string, pluginsDir string) string {
	expandedDir := ExpandPath(pluginsDir)
	binaryName := fmt.Sprintf("km-plugin-%s", pluginName)
	return filepath.Join(expandedDir, binaryName)
}