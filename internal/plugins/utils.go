package plugins

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands ~ to user home directory
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, path[2:])
		}
	}
	return path
}