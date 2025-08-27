package configinfra

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	configdomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/config"
	configports "github.com/kilometers-ai/kilometers-cli/internal/core/ports/config"
)

// FileLoader discovers configuration from filesystem (.env files and saved config)
// Mirrors current behavior and priorities: saved config priority 3; .env files priority 4.
type FileLoader struct{}

func NewFileLoader() *FileLoader { return &FileLoader{} }

func (l *FileLoader) Name() string { return "filesystem" }

func (l *FileLoader) Load(ctx context.Context) (configdomain.Snapshot, error) {
	snap := make(configdomain.Snapshot)

	homeDir, _ := os.UserHomeDir()
	workDir, _ := os.Getwd()
	searchPaths := []string{
		workDir,
		filepath.Join(homeDir, ".config", "kilometers"),
	}

	// Load .env files as priority 4
	for _, searchPath := range searchPaths {
		envPath := filepath.Join(searchPath, ".env")
		if _, err := os.Stat(envPath); err == nil {
			l.loadEnvFile(envPath, snap)
		}
	}

	// Load saved config as priority 3 (read JSON directly to avoid package cycles)
	configPath := filepath.Join(homeDir, ".config", "kilometers", "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		// very small JSON extraction without strong typing
		// fields: api_key, api_endpoint, buffer_size, batch_size, log_level, debug, plugins_dir, auto_provision, default_timeout
		// NOTE: best-effort parsing; mismatches are ignored
		var kv map[string]interface{}
		if err := json.Unmarshal(data, &kv); err == nil {
			toEntry := func(field string, v interface{}) {
				snap[field] = configdomain.Entry{Key: field, Value: v, Source: "file", SourcePath: configPath, Priority: 3}
			}
			if v, ok := kv["api_key"].(string); ok && v != "" {
				toEntry("api_key", v)
			}
			if v, ok := kv["api_endpoint"].(string); ok && v != "" {
				toEntry("api_endpoint", v)
			}
			if v, ok := toInt(kv["buffer_size"]); ok {
				toEntry("buffer_size", v)
			}
			if v, ok := toInt(kv["batch_size"]); ok {
				toEntry("batch_size", v)
			}
			if v, ok := kv["log_level"].(string); ok && v != "" {
				toEntry("log_level", v)
			}
			if v, ok := toBool(kv["debug"]); ok {
				toEntry("debug", v)
			}
			if v, ok := kv["plugins_dir"].(string); ok && v != "" {
				toEntry("plugins_dir", v)
			}
			if v, ok := toBool(kv["auto_provision"]); ok {
				toEntry("auto_provision", v)
			}
			if v, ok := toDuration(kv["default_timeout"]); ok {
				toEntry("default_timeout", v)
			}
		}
	}

	return snap, nil
}

func (l *FileLoader) loadEnvFile(path string, snap configdomain.Snapshot) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

		toEntry := func(field string, v interface{}) {
			snap[field] = configdomain.Entry{Key: field, Value: v, Source: "file", SourcePath: fmt.Sprintf("%s:%s", path, key), Priority: 4}
		}

		switch strings.ToUpper(key) {
		case "KM_API_KEY":
			toEntry("api_key", value)
		case "KM_API_ENDPOINT":
			toEntry("api_endpoint", value)
		case "KM_BUFFER_SIZE":
			if i, err := strconv.Atoi(value); err == nil {
				toEntry("buffer_size", i)
			}
		case "KM_BATCH_SIZE":
			if i, err := strconv.Atoi(value); err == nil {
				toEntry("batch_size", i)
			}
		case "KM_LOG_LEVEL":
			toEntry("log_level", value)
		case "KM_DEBUG":
			if b, err := strconv.ParseBool(value); err == nil {
				toEntry("debug", b)
			}
		case "KM_PLUGINS_DIR":
			toEntry("plugins_dir", value)
		case "KM_AUTO_PROVISION":
			if b, err := strconv.ParseBool(value); err == nil {
				toEntry("auto_provision", b)
			}
		}
	}
}

var _ configports.Loader = (*FileLoader)(nil)

func toInt(x interface{}) (int, bool) {
	switch t := x.(type) {
	case float64:
		return int(t), true
	case int:
		return t, true
	case string:
		if i, err := strconv.Atoi(t); err == nil {
			return i, true
		}
	}
	return 0, false
}

func toBool(x interface{}) (bool, bool) {
	switch t := x.(type) {
	case bool:
		return t, true
	case string:
		if b, err := strconv.ParseBool(t); err == nil {
			return b, true
		}
	}
	return false, false
}

func toDuration(x interface{}) (time.Duration, bool) {
	switch t := x.(type) {
	case string:
		if d, err := time.ParseDuration(t); err == nil {
			return d, true
		}
	}
	return 0, false
}
