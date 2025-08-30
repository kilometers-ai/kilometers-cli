package plugins

import (
	"testing"
)

func TestWirePluginManager(t *testing.T) {
	// Test that we can create a plugin manager using Wire DI
	opts := PluginManagerOptions{
		Debug:       true,
		ApiEndpoint: "http://localhost:5194",
		PluginDirs:  []string{"/tmp/test-plugins"},
		CLIVersion:  "test-1.0.0",
	}

	manager, err := InitializePluginManager(opts)
	if err != nil {
		t.Fatalf("Failed to initialize plugin manager with Wire: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil plugin manager")
	}

	// Verify the configuration was applied correctly
	if manager.config.Debug != opts.Debug {
		t.Errorf("Expected debug=%v, got %v", opts.Debug, manager.config.Debug)
	}

	if manager.config.ApiEndpoint != opts.ApiEndpoint {
		t.Errorf("Expected apiEndpoint=%s, got %s", opts.ApiEndpoint, manager.config.ApiEndpoint)
	}

	if manager.config.CLIVersion != opts.CLIVersion {
		t.Errorf("Expected cliVersion=%s, got %s", opts.CLIVersion, manager.config.CLIVersion)
	}

	// Verify that the validator is properly injected
	if manager.validator == nil {
		t.Error("Expected validator to be injected")
	}

	// Check that it's the WirePluginValidator type
	if _, ok := manager.validator.(*WirePluginValidator); !ok {
		t.Error("Expected validator to be WirePluginValidator")
	}

	// Verify other components are injected
	if manager.discovery == nil {
		t.Error("Expected discovery to be injected")
	}

	if manager.authenticator == nil {
		t.Error("Expected authenticator to be injected")
	}

	if manager.authCache == nil {
		t.Error("Expected authCache to be injected")
	}
}

func TestSimplePluginManager(t *testing.T) {
	// Test the simple manager initialization
	manager, err := InitializeSimplePluginManager()
	if err != nil {
		t.Fatalf("Failed to initialize simple plugin manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil plugin manager")
	}

	// Verify default configuration
	if !manager.config.Debug {
		t.Error("Expected debug to be true for simple manager")
	}

	if manager.config.ApiEndpoint != "http://localhost:5194" {
		t.Errorf("Expected default API endpoint, got %s", manager.config.ApiEndpoint)
	}

	if manager.config.CLIVersion != "test" {
		t.Errorf("Expected test CLI version, got %s", manager.config.CLIVersion)
	}
}