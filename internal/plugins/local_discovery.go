package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// LocalKmpkgDiscovery implements discovery of local .kmpkg packages
type LocalKmpkgDiscovery struct {
	pluginDirs []string
	parser     *FileParser
	debug      bool
}

// NewLocalKmpkgDiscovery creates a new local .kmpkg discovery service
func NewLocalKmpkgDiscovery(pluginDirs []string, debug bool) *LocalKmpkgDiscovery {
	return &LocalKmpkgDiscovery{
		pluginDirs: pluginDirs,
		parser:     NewFileParser(debug),
		debug:      debug,
	}
}

// DiscoverKmpkgPackages discovers all .kmpkg files in the configured plugin directories
func (d *LocalKmpkgDiscovery) DiscoverKmpkgPackages(ctx context.Context) ([]kmsdk.KmpkgPackage, error) {
	var packages []kmsdk.KmpkgPackage
	var errors []error

	// Recursively scan all configured plugin directories for .kmpkg files
	// Supports nested directory structures like ~/.km/plugins/category/subcategory/plugin.kmpkg
	for _, dir := range d.pluginDirs {
		expandedDir := ExpandPath(dir)

		if d.debug {
			fmt.Printf("[LocalKmpkgDiscovery] Recursively scanning directory tree: %s\n", expandedDir)
		}

		// Check if directory exists
		if _, err := os.Stat(expandedDir); os.IsNotExist(err) {
			if d.debug {
				fmt.Printf("[LocalKmpkgDiscovery] Directory does not exist: %s\n", expandedDir)
			}
			continue
		}

		// Find all .kmpkg files in directory
		foundPackages, err := d.scanDirectoryForKmpkgs(ctx, expandedDir)
		if err != nil {
			errors = append(errors, fmt.Errorf("error scanning %s: %w", expandedDir, err))
			continue
		}

		packages = append(packages, foundPackages...)
	}

	if d.debug {
		fmt.Printf("[LocalKmpkgDiscovery] Found %d .kmpkg packages total\n", len(packages))
	}

	// Return results even if there were some errors
	if len(errors) > 0 && len(packages) == 0 {
		return nil, fmt.Errorf("failed to discover packages: %v", errors)
	}

	return packages, nil
}

// FindKmpkgPackage finds a specific .kmpkg package by name
func (d *LocalKmpkgDiscovery) FindKmpkgPackage(ctx context.Context, packageName string) (*kmsdk.KmpkgPackage, error) {
	packages, err := d.DiscoverKmpkgPackages(ctx)
	if err != nil {
		return nil, err
	}

	for _, pkg := range packages {
		if pkg.Metadata.Name == packageName {
			return &pkg, nil
		}
	}

	return nil, fmt.Errorf("package %s not found in local directories", packageName)
}

// scanDirectoryForKmpkgs recursively scans a directory tree for .kmpkg files
func (d *LocalKmpkgDiscovery) scanDirectoryForKmpkgs(ctx context.Context, dir string) ([]kmsdk.KmpkgPackage, error) {
	var packages []kmsdk.KmpkgPackage

	// Use filepath.WalkDir for recursive directory traversal
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		// Handle walk errors
		if err != nil {
			if d.debug {
				fmt.Printf("[LocalKmpkgDiscovery] Error accessing path %s: %v\n", path, err)
			}
			return nil // Continue walking despite errors
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip directories and non-.kmpkg files
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".kmpkg") {
			return nil
		}

		if d.debug {
			fmt.Printf("[LocalKmpkgDiscovery] Processing .kmpkg file: %s\n", path)
		}

		// Parse the .kmpkg file using the reusable parser
		pkg, err := d.parser.ParseKmpkgFile(path)
		if err != nil {
			if d.debug {
				fmt.Printf("[LocalKmpkgDiscovery] Failed to parse %s: %v\n", path, err)
			}
			return nil // Skip invalid packages but don't fail entirely
		}

		packages = append(packages, *pkg)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
	}

	return packages, nil
}

// ValidateKmpkgPackage validates that a .kmpkg package is well-formed
func (d *LocalKmpkgDiscovery) ValidateKmpkgPackage(packagePath string) error {
	return d.parser.ValidateKmpkgArchive(packagePath)
}

// GetPackageDirectories returns the configured plugin directories
func (d *LocalKmpkgDiscovery) GetPackageDirectories() []string {
	var expandedDirs []string
	for _, dir := range d.pluginDirs {
		expandedDirs = append(expandedDirs, ExpandPath(dir))
	}
	return expandedDirs
}
