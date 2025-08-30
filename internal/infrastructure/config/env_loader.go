package configinfra

import (
	"context"
	"os"
	"strconv"
	"time"

	configdomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/config"
	configports "github.com/kilometers-ai/kilometers-cli/internal/core/ports/config"
)

type EnvLoader struct{}

func NewEnvLoader() *EnvLoader { return &EnvLoader{} }

func (l *EnvLoader) Name() string { return "env" }

// Load implements Loader by returning the environment snapshot.
func (l *EnvLoader) Load(ctx context.Context) (configdomain.Snapshot, error) {
	return l.LoadEnv(), nil
}

// LoadEnv builds a snapshot from standard KM_* environment variables (priority 2).
func (l *EnvLoader) LoadEnv() configdomain.Snapshot {
	snap := make(configdomain.Snapshot)
	add := func(key, field string, convert func(string) interface{}) {
		if v := os.Getenv(key); v != "" {
			val := interface{}(v)
			if convert != nil {
				val = convert(v)
			}
			snap[field] = configdomain.Entry{Key: field, Value: val, Source: "env", SourcePath: key, Priority: 2}
		}
	}

	add("KM_API_KEY", "api_key", nil)
	add("KM_API_ENDPOINT", "api_endpoint", nil)
	add("KM_BUFFER_SIZE", "buffer_size", func(s string) interface{} { i, _ := strconv.Atoi(s); return i })
	add("KM_BATCH_SIZE", "batch_size", func(s string) interface{} { i, _ := strconv.Atoi(s); return i })
	add("KM_LOG_LEVEL", "log_level", nil)
	add("KM_DEBUG", "debug", func(s string) interface{} { b, _ := strconv.ParseBool(s); return b })
	add("KM_PLUGINS_DIR", "plugins_dir", nil)
	add("KM_AUTO_PROVISION", "auto_provision", func(s string) interface{} { b, _ := strconv.ParseBool(s); return b })
	add("KM_TIMEOUT", "default_timeout", func(s string) interface{} { d, _ := time.ParseDuration(s); return d })

	return snap
}

var _ configports.Loader = (*EnvLoader)(nil)
