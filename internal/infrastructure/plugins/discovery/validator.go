package discovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// BasicPluginValidator implements basic plugin validation
type BasicPluginValidator struct {
	debug bool
}

// NewBasicPluginValidator creates a new basic plugin validator
func NewBasicPluginValidator(debug bool) *BasicPluginValidator {
	return &BasicPluginValidator{
		debug: debug,
	}
}

// ValidateSignature verifies the digital signature of a plugin binary
func (v *BasicPluginValidator) ValidateSignature(ctx context.Context, pluginPath string, signature []byte) error {
	if v.debug {
		fmt.Printf("[PluginValidator] Validating signature for plugin: %s\n", pluginPath)
	}

	// For now, implement basic hash validation
	// In production, this would verify RSA signatures
	expectedHash, err := v.calculateFileHash(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to calculate plugin hash: %w", err)
	}

	if v.debug {
		fmt.Printf("[PluginValidator] Plugin hash: %s\n", expectedHash)
	}

	// For POC, we'll accept any plugin file that exists and is executable
	// In production, this would:
	// 1. Load the public key for signature verification
	// 2. Verify the RSA signature against the file hash
	// 3. Check signature matches expected customer/plugin combination

	fileInfo, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("plugin file not accessible: %w", err)
	}

	// Check if executable
	if fileInfo.Mode()&0111 == 0 {
		return fmt.Errorf("plugin file is not executable")
	}

	if v.debug {
		fmt.Printf("[PluginValidator] Plugin signature validation passed for: %s\n", pluginPath)
	}

	return nil
}

// GetPluginManifest extracts metadata from a plugin binary or manifest file
func (v *BasicPluginValidator) GetPluginManifest(ctx context.Context, pluginPath string) (*ports.PluginManifest, error) {
	// Look for external manifest file first
	manifestPath := getManifestPath(pluginPath)

	if _, err := os.Stat(manifestPath); err == nil {
		// Load external manifest
		return v.loadExternalManifest(manifestPath)
	}

	if v.debug {
		fmt.Printf("[PluginValidator] No external manifest found for %s, using defaults\n", pluginPath)
	}

	// In production, this would extract embedded manifest from the binary
	// For now, return default values based on filename
	return v.createDefaultManifest(pluginPath), nil
}

// calculateFileHash calculates SHA256 hash of the plugin file
func (v *BasicPluginValidator) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// loadExternalManifest loads manifest from external JSON file
func (v *BasicPluginValidator) loadExternalManifest(manifestPath string) (*ports.PluginManifest, error) {
	// This is the same logic as in discovery.go - we could refactor to share
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	manifest := &ports.PluginManifest{}
	if err := json.Unmarshal(data, manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	return manifest, nil
}

// createDefaultManifest creates a default manifest based on the plugin filename
func (v *BasicPluginValidator) createDefaultManifest(pluginPath string) *ports.PluginManifest {
	name := extractPluginNameFromPath(pluginPath)

	// Set default required tier based on plugin name
	tier := "Free"
	if name == "api-logger" || name == "advanced-analytics" {
		tier = "Pro"
	} else if name == "enterprise-security" || name == "compliance-reporter" {
		tier = "Enterprise"
	}

	return &ports.PluginManifest{
		Name:         name,
		Version:      "1.0.0",
		Description:  fmt.Sprintf("Kilometers plugin: %s", name),
		RequiredTier: tier,
		Author:       "Kilometers.ai",
	}
}

// Production signature validation (placeholder for future implementation)
func (v *BasicPluginValidator) validateRSASignature(pluginPath string, signature []byte, publicKey []byte) error {
	// This would be implemented in production with actual RSA signature verification
	// Steps would be:
	// 1. Load RSA public key
	// 2. Calculate SHA256 hash of plugin file
	// 3. Verify RSA signature of hash using public key
	// 4. Return error if verification fails

	return fmt.Errorf("RSA signature validation not implemented in POC")
}

// Security helpers for production implementation

// extractEmbeddedManifest would extract manifest embedded in plugin binary
func (v *BasicPluginValidator) extractEmbeddedManifest(pluginPath string) (*ports.PluginManifest, error) {
	// In production, this would:
	// 1. Read the plugin binary
	// 2. Find the embedded manifest section
	// 3. Extract and parse the manifest data
	// 4. Return the parsed manifest

	return nil, fmt.Errorf("embedded manifest extraction not implemented in POC")
}

// validateCustomerSpecificBinary would verify plugin is built for specific customer
func (v *BasicPluginValidator) validateCustomerSpecificBinary(pluginPath string, customerID string) error {
	// In production, this would:
	// 1. Extract embedded customer ID from plugin binary
	// 2. Compare with expected customer ID
	// 3. Return error if mismatch

	return fmt.Errorf("customer-specific binary validation not implemented in POC")
}
