package plugins

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// FileParser provides reusable file parsing functionality for plugin packages
type FileParser struct {
	debug bool
}

// NewFileParser creates a new file parser
func NewFileParser(debug bool) *FileParser {
	return &FileParser{
		debug: debug,
	}
}

// ParseKmpkgFile parses a .kmpkg file and extracts its metadata
func (p *FileParser) ParseKmpkgFile(filePath string) (*kmsdk.KmpkgPackage, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if p.debug {
		fmt.Printf("[FileParser] Parsing .kmpkg file: %s (%d bytes)\n", filePath, fileInfo.Size())
	}

	// Parse as tar.gz format
	metadata, err := p.extractMetadataFromKmpkg(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	// Create the package structure
	pkg := &kmsdk.KmpkgPackage{
		FilePath:   filePath,
		FileName:   fileInfo.Name(),
		FileSize:   fileInfo.Size(),
		ModifiedAt: fileInfo.ModTime(),
		Metadata:   *metadata,
	}

	if p.debug {
		fmt.Printf("[FileParser] Successfully parsed package: %s v%s\n", pkg.Metadata.Name, pkg.Metadata.Version)
	}

	return pkg, nil
}

// extractMetadataFromKmpkg extracts metadata from a tar.gz format .kmpkg file
func (p *FileParser) extractMetadataFromKmpkg(filePath string) (*kmsdk.KmpkgMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue // Skip non-regular files
		}

		// Look for metadata.json or manifest.json
		if header.Name == "metadata.json" || header.Name == "manifest.json" || 
		   strings.HasSuffix(header.Name, "/metadata.json") || strings.HasSuffix(header.Name, "/manifest.json") {
			
			if p.debug {
				fmt.Printf("[FileParser] Found metadata file: %s\n", header.Name)
			}

			// Read the metadata content
			content, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read metadata file: %w", err)
			}

			// Parse JSON
			var metadata kmsdk.KmpkgMetadata
			if err := json.Unmarshal(content, &metadata); err != nil {
				return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
			}

			// Validate required fields
			if err := p.validateMetadata(&metadata); err != nil {
				return nil, fmt.Errorf("invalid metadata: %w", err)
			}

			return &metadata, nil
		}
	}

	return nil, fmt.Errorf("metadata.json or manifest.json not found in tar.gz archive")
}

// ExtractBinaryFromTarGz finds and extracts the plugin binary from a tar.gz archive
func (p *FileParser) ExtractBinaryFromTarGz(filePath string, binaryName string, targetPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
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
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue // Skip non-regular files
		}

		// Look for the binary file
		if header.Name == binaryName || strings.HasSuffix(header.Name, "/"+binaryName) {
			if p.debug {
				fmt.Printf("[FileParser] Found binary file: %s\n", header.Name)
			}

			// Create the target file
			targetFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create target file: %w", err)
			}
			defer targetFile.Close()

			// Copy the binary content
			written, err := io.Copy(targetFile, tarReader)
			if err != nil {
				return fmt.Errorf("failed to extract binary: %w", err)
			}

			// Make the binary executable
			if err := os.Chmod(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to make binary executable: %w", err)
			}

			if p.debug {
				fmt.Printf("[FileParser] Extracted binary %s (%d bytes) to %s\n", binaryName, written, targetPath)
			}

			return nil
		}
	}

	return fmt.Errorf("binary %s not found in .kmpkg archive", binaryName)
}

// ListTarGzContents lists all files in a tar.gz archive (for debugging)
func (p *FileParser) ListTarGzContents(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var contents []string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}

		contents = append(contents, fmt.Sprintf("%s (%d bytes)", header.Name, header.Size))
	}

	return contents, nil
}

// validateMetadata validates that required metadata fields are present
func (p *FileParser) validateMetadata(metadata *kmsdk.KmpkgMetadata) error {
	if metadata.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if metadata.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	if metadata.BinaryName == "" {
		return fmt.Errorf("binary name is required")
	}
	if metadata.RequiredTier == "" {
		metadata.RequiredTier = "Free" // Default to Free tier
	}
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now() // Set default creation time
	}

	return nil
}

// ValidateKmpkgArchive performs basic validation of a .kmpkg tar.gz archive structure
func (p *FileParser) ValidateKmpkgArchive(filePath string) error {
	// Try to extract metadata to validate structure
	metadata, err := p.extractMetadataFromKmpkg(filePath)
	if err != nil {
		return fmt.Errorf("failed to validate archive structure: %w", err)
	}

	// Check if binary exists in archive
	if metadata.BinaryName != "" {
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()

		tarReader := tar.NewReader(gzReader)
		hasBinary := false

		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read tar entry: %w", err)
			}

			if header.Typeflag == tar.TypeReg && 
			   (header.Name == metadata.BinaryName || strings.HasSuffix(header.Name, "/"+metadata.BinaryName)) {
				hasBinary = true
				break
			}
		}

		if !hasBinary {
			return fmt.Errorf("binary %s not found in .kmpkg archive", metadata.BinaryName)
		}
	}

	if p.debug {
		fmt.Printf("[FileParser] Archive validation passed: %s\n", filePath)
		if contents, err := p.ListTarGzContents(filePath); err == nil {
			fmt.Printf("[FileParser] Archive contents: %v\n", contents)
		}
	}

	return nil
}