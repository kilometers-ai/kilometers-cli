package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ConfigValidator validates configuration values
type ConfigValidator struct {
	apiKeyPattern *regexp.Regexp
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		// Common API key patterns: alphanumeric with dashes, 20+ chars
		apiKeyPattern: regexp.MustCompile(`^[a-zA-Z0-9\-_]{20,}$`),
	}
}

// ValidateAPIEndpoint validates an API endpoint URL
func (v *ConfigValidator) ValidateAPIEndpoint(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("API endpoint cannot be empty")
	}

	// Parse URL
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (must be http or https)", u.Scheme)
	}

	// Check host
	if u.Host == "" {
		return fmt.Errorf("URL must include host")
	}

	// Warn about non-HTTPS in production
	if u.Scheme == "http" && !strings.Contains(u.Host, "localhost") && !strings.Contains(u.Host, "127.0.0.1") {
		// This is just a warning, not an error
		fmt.Fprintf(os.Stderr, "Warning: Using non-HTTPS endpoint for non-localhost URL: %s\n", endpoint)
	}

	return nil
}

// ValidateAPIKey validates an API key format
func (v *ConfigValidator) ValidateAPIKey(key string) error {
	if key == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Check length
	if len(key) < 20 {
		return fmt.Errorf("API key too short (minimum 20 characters)")
	}

	if len(key) > 256 {
		return fmt.Errorf("API key too long (maximum 256 characters)")
	}

	// Check format - allow various formats as different services have different patterns
	// Just ensure it's not obviously invalid
	if strings.Contains(key, " ") {
		return fmt.Errorf("API key cannot contain spaces")
	}

	if strings.Contains(key, "\n") || strings.Contains(key, "\r") || strings.Contains(key, "\t") {
		return fmt.Errorf("API key cannot contain whitespace characters")
	}

	// Check for common placeholder values
	placeholders := []string{
		"your-api-key-here",
		"xxxx-xxxx-xxxx-xxxx",
		"<api-key>",
		"[api-key]",
		"${API_KEY}",
		"$API_KEY",
		"REPLACE_ME",
		"CHANGE_ME",
	}

	lowerKey := strings.ToLower(key)
	for _, placeholder := range placeholders {
		if strings.Contains(lowerKey, strings.ToLower(placeholder)) {
			return fmt.Errorf("API key appears to be a placeholder value")
		}
	}

	return nil
}

// ValidateBufferSize validates buffer size value
func (v *ConfigValidator) ValidateBufferSize(size interface{}) error {
	var sizeInt int

	switch val := size.(type) {
	case int:
		sizeInt = val
	case int64:
		sizeInt = int(val)
	case float64:
		sizeInt = int(val)
	case string:
		parsed, err := parseSize(val)
		if err != nil {
			return fmt.Errorf("invalid buffer size format: %w", err)
		}
		sizeInt = parsed
	default:
		return fmt.Errorf("invalid buffer size type: %T", size)
	}

	// Validate range
	minSize := 1024              // 1KB minimum
	maxSize := 100 * 1024 * 1024 // 100MB maximum

	if sizeInt < minSize {
		return fmt.Errorf("buffer size too small (minimum 1KB)")
	}

	if sizeInt > maxSize {
		return fmt.Errorf("buffer size too large (maximum 100MB)")
	}

	return nil
}

// ValidateLogLevel validates log level value
func (v *ConfigValidator) ValidateLogLevel(level string) error {
	validLevels := []string{"debug", "info", "warn", "error", "fatal", "panic", "trace"}

	normalizedLevel := strings.ToLower(strings.TrimSpace(level))

	for _, valid := range validLevels {
		if normalizedLevel == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid log level: %s (valid levels: %s)", level, strings.Join(validLevels, ", "))
}

// ValidatePluginsDir validates plugins directory path
func (v *ConfigValidator) ValidatePluginsDir(path string) error {
	if path == "" {
		// Empty is OK, will use default
		return nil
	}

	// Expand path
	expandedPath := expandPath(path)

	// Check if path exists
	info, err := os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, check if we can create it
			dir := filepath.Dir(expandedPath)
			if _, err := os.Stat(dir); err != nil {
				return fmt.Errorf("parent directory does not exist: %s", dir)
			}
			// Parent exists, so we can create this directory
			return nil
		}
		return fmt.Errorf("failed to check plugins directory: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("plugins path exists but is not a directory: %s", path)
	}

	// Check permissions
	testFile := filepath.Join(expandedPath, ".km-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("plugins directory is not writable: %w", err)
	}
	os.Remove(testFile)

	return nil
}

// ValidateTimeout validates timeout duration
func (v *ConfigValidator) ValidateTimeout(timeout interface{}) error {
	var duration time.Duration

	switch val := timeout.(type) {
	case time.Duration:
		duration = val
	case string:
		parsed, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("invalid timeout format: %w", err)
		}
		duration = parsed
	case int:
		duration = time.Duration(val) * time.Second
	case int64:
		duration = time.Duration(val) * time.Second
	case float64:
		duration = time.Duration(val * float64(time.Second))
	default:
		return fmt.Errorf("invalid timeout type: %T", timeout)
	}

	// Validate range
	minTimeout := 1 * time.Second
	maxTimeout := 5 * time.Minute

	if duration < minTimeout {
		return fmt.Errorf("timeout too short (minimum 1s)")
	}

	if duration > maxTimeout {
		return fmt.Errorf("timeout too long (maximum 5m)")
	}

	return nil
}

// ValidateAll validates all fields in a configuration map
func (v *ConfigValidator) ValidateAll(config map[string]interface{}) map[string]error {
	errors := make(map[string]error)

	// Validate each field if present
	if val, ok := config["api_endpoint"]; ok {
		if str, ok := val.(string); ok {
			if err := v.ValidateAPIEndpoint(str); err != nil {
				errors["api_endpoint"] = err
			}
		}
	}

	if val, ok := config["api_key"]; ok {
		if str, ok := val.(string); ok {
			if err := v.ValidateAPIKey(str); err != nil {
				errors["api_key"] = err
			}
		}
	}

	if val, ok := config["buffer_size"]; ok {
		if err := v.ValidateBufferSize(val); err != nil {
			errors["buffer_size"] = err
		}
	}

	if val, ok := config["log_level"]; ok {
		if str, ok := val.(string); ok {
			if err := v.ValidateLogLevel(str); err != nil {
				errors["log_level"] = err
			}
		}
	}

	if val, ok := config["plugins_dir"]; ok {
		if str, ok := val.(string); ok {
			if err := v.ValidatePluginsDir(str); err != nil {
				errors["plugins_dir"] = err
			}
		}
	}

	if val, ok := config["default_timeout"]; ok {
		if err := v.ValidateTimeout(val); err != nil {
			errors["default_timeout"] = err
		}
	}

	return errors
}

// parseSize parses size values (e.g., "1MB", "512KB", "1024")
func parseSize(val string) (int, error) {
	// Simple implementation for common size formats
	if val == "" {
		return 0, fmt.Errorf("empty size value")
	}
	
	// Try to parse as plain number first
	if size, err := strconv.Atoi(val); err == nil {
		return size, nil
	}
	
	// Add more sophisticated parsing if needed
	return 0, fmt.Errorf("unsupported size format: %s", val)
}

// expandPath expands ~ and environment variables in paths
func expandPath(path string) string {
	if path == "" {
		return path
	}
	
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			path = strings.Replace(path, "~", homeDir, 1)
		}
	}
	
	// Expand environment variables
	return os.ExpandEnv(path)
}
