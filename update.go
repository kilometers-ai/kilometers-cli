package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

var (
	// Version is set at build time via -ldflags
	Version = "dev"

	// Update check URL
	versionURL = "https://get.kilometers.ai/install/version.json"
)

// VersionInfo represents the version manifest structure
type VersionInfo struct {
	Latest    string            `json:"latest"`
	Stable    string            `json:"stable"`
	Updated   time.Time         `json:"updated"`
	Downloads map[string]string `json:"downloads"`
	Checksums map[string]string `json:"checksums"`
}

// CheckForUpdate checks if a new version is available
func CheckForUpdate() (bool, string, error) {
	if Version == "dev" {
		return false, "", fmt.Errorf("development version, skipping update check")
	}

	// Fetch version manifest
	resp, err := http.Get(versionURL)
	if err != nil {
		return false, "", fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	var info VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return false, "", fmt.Errorf("failed to parse version info: %w", err)
	}

	// Compare versions
	currentVersion := strings.TrimPrefix(Version, "v")
	latestVersion := strings.TrimPrefix(info.Latest, "v")

	if currentVersion != latestVersion {
		return true, info.Latest, nil
	}

	return false, "", nil
}

// GetPlatformKey returns the platform key for downloads
func GetPlatformKey() string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	return fmt.Sprintf("%s-%s", os, arch)
}

// SelfUpdate performs a self-update of the binary
func SelfUpdate() error {
	// Check for update
	hasUpdate, newVersion, err := CheckForUpdate()
	if err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}

	if !hasUpdate {
		fmt.Println("You are already running the latest version.")
		return nil
	}

	fmt.Printf("New version available: %s (current: %s)\n", newVersion, Version)
	fmt.Print("Do you want to update? [y/N]: ")

	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("Update cancelled.")
		return nil
	}

	// Fetch version manifest for download URL
	resp, err := http.Get(versionURL)
	if err != nil {
		return fmt.Errorf("failed to fetch version info: %w", err)
	}
	defer resp.Body.Close()

	var info VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return fmt.Errorf("failed to parse version info: %w", err)
	}

	// Get download URL for current platform
	platform := GetPlatformKey()
	downloadURL, ok := info.Downloads[platform]
	if !ok {
		return fmt.Errorf("no download available for platform: %s", platform)
	}

	// Download new binary
	fmt.Printf("Downloading %s...\n", downloadURL)
	resp, err = http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create temporary file
	tmpFile := execPath + ".new"
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Copy download to temporary file
	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to write update: %w", err)
	}

	// Make new binary executable
	if err := os.Chmod(tmpFile, 0755); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Backup current binary
	backupPath := execPath + ".backup"
	if err := os.Rename(execPath, backupPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move new binary into place
	if err := os.Rename(tmpFile, execPath); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, execPath)
		os.Remove(tmpFile)
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Remove backup
	os.Remove(backupPath)

	fmt.Printf("Successfully updated to version %s!\n", newVersion)
	fmt.Println("Please restart the command to use the new version.")

	return nil
}

// ShowVersion displays version information
func ShowVersion() {
	fmt.Printf("Kilometers CLI %s\n", Version)
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Go version: %s\n", runtime.Version())

	// Check for updates in background
	go func() {
		if hasUpdate, newVersion, err := CheckForUpdate(); err == nil && hasUpdate {
			fmt.Printf("\nUpdate available: %s (run 'km update' to install)\n", newVersion)
		}
	}()
}

// Add these command handlers to your main.go

func printHelp() {
	fmt.Printf(`Kilometers CLI %s

Usage:
  km <mcp-server-command> [args...]  Wrap an MCP server
  km version                         Show version information
  km update                          Update to latest version
  km help                            Show this help message

Examples:
  km npx @modelcontextprotocol/server-github
  km python -m slack_mcp_server
  km ./my-custom-mcp-server --config config.json

Environment Variables:
  Core Configuration:
    KILOMETERS_API_URL         API endpoint (default: http://localhost:5194)
    KILOMETERS_API_KEY         API authentication key
    KILOMETERS_CUSTOMER_ID     Customer identifier (default: "default")
    KM_DEBUG                   Enable debug logging
    KM_BATCH_SIZE              Events per batch (default: 10)

  Advanced Filtering:
    KM_ENABLE_RISK_DETECTION   Enable client-side risk analysis (true/false)
    KM_METHOD_WHITELIST        Comma-separated list of MCP methods to capture
                               Examples: "tools/call,resources/read"
                                        "tools/*,resources/list"
    KM_PAYLOAD_SIZE_LIMIT      Maximum payload size in bytes (0 = no limit)
    KM_HIGH_RISK_ONLY          Only capture high-risk events (true/false)
    KM_EXCLUDE_PING            Exclude ping messages (default: true)

Configuration Examples:
  # Basic monitoring
  km npx @modelcontextprotocol/server-github

  # Risk-focused monitoring (security mode)
  KM_ENABLE_RISK_DETECTION=true KM_HIGH_RISK_ONLY=true km mcp-server

  # Method-specific monitoring (only file operations)
  KM_METHOD_WHITELIST="resources/read,resources/write" km mcp-server

  # Bandwidth-conscious monitoring
  KM_PAYLOAD_SIZE_LIMIT=10240 KM_EXCLUDE_PING=true km mcp-server

  # Enterprise security mode
  KM_ENABLE_RISK_DETECTION=true \
  KM_METHOD_WHITELIST="resources/read,tools/call" \
  KM_PAYLOAD_SIZE_LIMIT=5120 \
  km mcp-server

Risk Levels:
  HIGH (75+)     - System file access, admin operations, large payloads
  MEDIUM (35+)   - Environment files, database queries, sensitive operations  
  LOW (10+)      - Standard tool usage, list operations, basic prompts

For more information, visit https://docs.kilometers.ai
`, Version)
}
