package plugins

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

func TestLocalKmpkgDiscovery_Basic(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	
	// Create discovery service
	discovery := NewLocalKmpkgDiscovery([]string{tempDir}, true)
	
	ctx := context.Background()

	// Test with empty directory
	packages, err := discovery.DiscoverKmpkgPackages(ctx)
	if err != nil {
		t.Fatalf("Expected no error for empty directory, got: %v", err)
	}
	
	if len(packages) != 0 {
		t.Fatalf("Expected 0 packages in empty directory, got: %d", len(packages))
	}
}

func TestLocalKmpkgDiscovery_WithValidPackage(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	
	// Create a test .kmpkg file
	kmpkgPath := filepath.Join(tempDir, "test-plugin.kmpkg")
	err := createTestKmpkgFile(kmpkgPath, "test-plugin", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create test .kmpkg file: %v", err)
	}
	
	// Create discovery service
	discovery := NewLocalKmpkgDiscovery([]string{tempDir}, true)
	
	ctx := context.Background()

	// Test discovery
	packages, err := discovery.DiscoverKmpkgPackages(ctx)
	if err != nil {
		t.Fatalf("Discovery failed: %v", err)
	}
	
	if len(packages) != 1 {
		t.Fatalf("Expected 1 package, got: %d", len(packages))
	}
	
	pkg := packages[0]
	if pkg.Metadata.Name != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got: %s", pkg.Metadata.Name)
	}
	
	if pkg.Metadata.Version != "1.0.0" {
		t.Errorf("Expected plugin version '1.0.0', got: %s", pkg.Metadata.Version)
	}
	
	if pkg.FileName != "test-plugin.kmpkg" {
		t.Errorf("Expected filename 'test-plugin.kmpkg', got: %s", pkg.FileName)
	}
}

func TestLocalKmpkgDiscovery_FindSpecificPackage(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	
	// Create multiple test .kmpkg files
	err := createTestKmpkgFile(filepath.Join(tempDir, "plugin-a.kmpkg"), "plugin-a", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create test .kmpkg file: %v", err)
	}
	
	err = createTestKmpkgFile(filepath.Join(tempDir, "plugin-b.kmpkg"), "plugin-b", "2.0.0")
	if err != nil {
		t.Fatalf("Failed to create test .kmpkg file: %v", err)
	}
	
	// Create discovery service
	discovery := NewLocalKmpkgDiscovery([]string{tempDir}, true)
	
	ctx := context.Background()

	// Test finding specific package
	pkg, err := discovery.FindKmpkgPackage(ctx, "plugin-b")
	if err != nil {
		t.Fatalf("Failed to find plugin-b: %v", err)
	}
	
	if pkg.Metadata.Name != "plugin-b" {
		t.Errorf("Expected plugin name 'plugin-b', got: %s", pkg.Metadata.Name)
	}
	
	if pkg.Metadata.Version != "2.0.0" {
		t.Errorf("Expected plugin version '2.0.0', got: %s", pkg.Metadata.Version)
	}
	
	// Test finding non-existent package
	_, err = discovery.FindKmpkgPackage(ctx, "nonexistent-plugin")
	if err == nil {
		t.Fatal("Expected error when finding non-existent plugin")
	}
}

func TestFileParser_ValidateKmpkgArchive(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	
	// Create a valid .kmpkg file
	kmpkgPath := filepath.Join(tempDir, "valid-plugin.kmpkg")
	err := createTestKmpkgFile(kmpkgPath, "valid-plugin", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create test .kmpkg file: %v", err)
	}
	
	// Test validation
	parser := NewFileParser(true)
	err = parser.ValidateKmpkgArchive(kmpkgPath)
	if err != nil {
		t.Fatalf("Validation failed for valid package: %v", err)
	}
}

// createTestKmpkgFile creates a minimal valid .kmpkg file for testing
func createTestKmpkgFile(filePath, pluginName, version string) error {
	// Create ZIP file
	zipFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	
	// Create metadata.json
	metadata := kmsdk.KmpkgMetadata{
		PluginInfo: kmsdk.PluginInfo{
			Name:         pluginName,
			Version:      version,
			Description:  "Test plugin for unit testing",
			RequiredTier: "Free",
			Author:       "Test Suite",
		},
		BinaryName: pluginName + "-binary",
		Checksum:   "test-checksum",
		CreatedAt:  time.Now(),
	}
	
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	
	// Add metadata.json to ZIP
	metadataWriter, err := zipWriter.Create("metadata.json")
	if err != nil {
		return err
	}
	
	_, err = metadataWriter.Write(metadataBytes)
	if err != nil {
		return err
	}
	
	// Add dummy binary to ZIP
	binaryWriter, err := zipWriter.Create(metadata.BinaryName)
	if err != nil {
		return err
	}
	
	_, err = binaryWriter.Write([]byte("dummy binary content"))
	if err != nil {
		return err
	}
	
	return nil
}