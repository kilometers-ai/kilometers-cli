package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Command represents a server command to be executed and monitored
type Command struct {
	executable string
	args       []string
	workingDir string
	env        map[string]string
}

// NewCommand creates a new Command value object
func NewCommand(executable string, args []string) (Command, error) {
	if executable == "" {
		return Command{}, fmt.Errorf("executable cannot be empty")
	}

	// Get current working directory as default
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = "."
	}

	return Command{
		executable: executable,
		args:       append([]string(nil), args...), // Copy slice
		workingDir: workingDir,
		env:        make(map[string]string),
	}, nil
}

// NewCommandWithOptions creates a command with additional options
func NewCommandWithOptions(executable string, args []string, workingDir string, env map[string]string) (Command, error) {
	if executable == "" {
		return Command{}, fmt.Errorf("executable cannot be empty")
	}

	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			workingDir = "."
		}
	}

	// Resolve working directory to absolute path
	if !filepath.IsAbs(workingDir) {
		absDir, err := filepath.Abs(workingDir)
		if err == nil {
			workingDir = absDir
		}
	}

	// Copy environment map
	envCopy := make(map[string]string)
	for k, v := range env {
		envCopy[k] = v
	}

	return Command{
		executable: executable,
		args:       append([]string(nil), args...), // Copy slice
		workingDir: workingDir,
		env:        envCopy,
	}, nil
}

// Executable returns the command executable
func (c Command) Executable() string {
	return c.executable
}

// Args returns a copy of the command arguments
func (c Command) Args() []string {
	return append([]string(nil), c.args...)
}

// WorkingDir returns the working directory for the command
func (c Command) WorkingDir() string {
	return c.workingDir
}

// Env returns a copy of the environment variables
func (c Command) Env() map[string]string {
	envCopy := make(map[string]string)
	for k, v := range c.env {
		envCopy[k] = v
	}
	return envCopy
}

// String returns a string representation of the command
func (c Command) String() string {
	if len(c.args) == 0 {
		return c.executable
	}
	return fmt.Sprintf("%s %s", c.executable, strings.Join(c.args, " "))
}

// FullCommandLine returns the complete command line including executable and args
func (c Command) FullCommandLine() []string {
	result := make([]string, 0, len(c.args)+1)
	result = append(result, c.executable)
	result = append(result, c.args...)
	return result
}

// WithEnv returns a new Command with additional environment variables
func (c Command) WithEnv(key, value string) Command {
	newEnv := make(map[string]string)
	for k, v := range c.env {
		newEnv[k] = v
	}
	newEnv[key] = value

	return Command{
		executable: c.executable,
		args:       append([]string(nil), c.args...),
		workingDir: c.workingDir,
		env:        newEnv,
	}
}

// WithWorkingDir returns a new Command with a different working directory
func (c Command) WithWorkingDir(workingDir string) Command {
	return Command{
		executable: c.executable,
		args:       append([]string(nil), c.args...),
		workingDir: workingDir,
		env:        c.Env(), // Use method to get copy
	}
}

// IsValid validates the command structure
func (c Command) IsValid() error {
	if c.executable == "" {
		return fmt.Errorf("executable cannot be empty")
	}

	// Check if working directory exists (if it's an absolute path)
	if filepath.IsAbs(c.workingDir) {
		if stat, err := os.Stat(c.workingDir); err != nil || !stat.IsDir() {
			return fmt.Errorf("working directory does not exist: %s", c.workingDir)
		}
	}

	return nil
}
