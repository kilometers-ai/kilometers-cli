package plugins

import (
	"testing"
)

func TestNewPluginJWTAuthenticator(t *testing.T) {
	// Test with no API key
	auth := NewPluginJWTAuthenticator("", "", false)
	if auth == nil {
		t.Fatal("Expected authenticator to be created")
	}
	if auth.GetAPIKey() != "" {
		t.Errorf("Expected empty API key, got %s", auth.GetAPIKey())
	}
	if auth.GetAPIEndpoint() != "https://api.kilometers.ai" {
		t.Errorf("Expected default endpoint, got %s", auth.GetAPIEndpoint())
	}
}

func TestNewPluginJWTAuthenticatorWithCustomEndpoint(t *testing.T) {
	// Test with custom endpoint
	customEndpoint := "http://localhost:5194"
	auth := NewPluginJWTAuthenticator("test-key", customEndpoint, true)
	if auth == nil {
		t.Fatal("Expected authenticator to be created")
	}
	if auth.GetAPIKey() != "test-key" {
		t.Errorf("Expected test-key, got %s", auth.GetAPIKey())
	}
	if auth.GetAPIEndpoint() != customEndpoint {
		t.Errorf("Expected %s, got %s", customEndpoint, auth.GetAPIEndpoint())
	}
}

func TestSetDebug(t *testing.T) {
	auth := NewPluginJWTAuthenticator("", "", false)
	auth.SetDebug(true)
	// No way to check debug state directly, but this tests the method exists
}