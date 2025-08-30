package cli

import (
	"fmt"
	"strings"

	"github.com/kilometers-ai/kilometers-cli/internal/config"
)

// ParsedFlags represents the parsed command-line flags
type ParsedFlags struct {
	// Essential flags only
	BufferSize string

	// Server command
	ServerCommand []string
}

// ParseMonitorFlags manually parses monitor command flags to handle --server -- syntax
func ParseMonitorFlags(args []string) (*ParsedFlags, error) {
	flags := &ParsedFlags{
		// Set defaults
		BufferSize: "1MB",
	}

	// Find the --server and -- positions
	serverIndex := -1
	separatorIndex := -1

	for i, arg := range args {
		if arg == "--server" {
			serverIndex = i
		} else if arg == "--" && serverIndex != -1 {
			separatorIndex = i
			break
		}
	}

	if serverIndex == -1 {
		return nil, fmt.Errorf("--server flag is required")
	}

	if separatorIndex == -1 {
		return nil, fmt.Errorf("-- separator is required after --server flag")
	}

	// Extract server command
	if separatorIndex+1 >= len(args) {
		return nil, fmt.Errorf("server command is required after -- separator")
	}
	flags.ServerCommand = args[separatorIndex+1:]

	// Parse flags before --server
	flagArgs := args[:serverIndex]

	for i := 0; i < len(flagArgs); i++ {
		arg := flagArgs[i]

		if !strings.HasPrefix(arg, "-") {
			return nil, fmt.Errorf("unexpected argument: %s", arg)
		}

		// Handle different flag formats
		if strings.HasPrefix(arg, "--") {
			// Long flag format --flag=value or --flag value
			flagName := strings.TrimPrefix(arg, "--")
			var flagValue string

			if strings.Contains(flagName, "=") {
				parts := strings.SplitN(flagName, "=", 2)
				flagName = parts[0]
				flagValue = parts[1]
			} else {
				// Check if next argument is the value
				if i+1 < len(flagArgs) && !strings.HasPrefix(flagArgs[i+1], "-") {
					i++
					flagValue = flagArgs[i]
				} else {
					// Boolean flag
					flagValue = "true"
				}
			}

			if err := setFlag(flags, flagName, flagValue); err != nil {
				return nil, fmt.Errorf("invalid flag --%s: %w", flagName, err)
			}
		} else {
			// Short flag format -f value
			return nil, fmt.Errorf("short flags not supported yet: %s", arg)
		}
	}

	return flags, nil
}

// setFlag sets a specific flag value
func setFlag(flags *ParsedFlags, name, value string) error {
	switch name {
	case "buffer-size":
		flags.BufferSize = value
	case "help", "h":
		return fmt.Errorf("help requested")
	default:
		return fmt.Errorf("unknown flag: %s", name)
	}

	return nil
}

// ToMonitorConfig converts parsed flags to domain config
func (f *ParsedFlags) ToMonitorConfig() (config.MonitorConfig, error) {
	config := config.DefaultMonitorConfig()

	// Parse buffer size
	bufferSize, err := parseBufferSize(f.BufferSize)
	if err != nil {
		return config, fmt.Errorf("invalid buffer size: %w", err)
	}
	config.BufferSize = bufferSize

	return config, nil
}
