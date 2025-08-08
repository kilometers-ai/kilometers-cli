package domain

// MonitorConfig contains configuration for monitoring behavior
type MonitorConfig struct {
	BufferSize int
}

// DefaultMonitorConfig returns sensible defaults for monitoring
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		BufferSize: 1024 * 1024, // 1MB buffer
	}
}
