package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// APIEndpointDiscoverer discovers API endpoints from various sources
type APIEndpointDiscoverer struct {
	httpClient    *http.Client
	wellKnownURLs []string
}

// NewAPIEndpointDiscoverer creates a new API endpoint discoverer
func NewAPIEndpointDiscoverer() *APIEndpointDiscoverer {
	return &APIEndpointDiscoverer{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		wellKnownURLs: []string{
			"https://api.kilometers.ai",
			"https://api.kilometers.io",
			"http://localhost:5194", // Default local development
			"http://localhost:8080",
			"http://localhost:3000",
		},
	}
}

// DiscoverEndpoints searches for API endpoints from various sources
func (d *APIEndpointDiscoverer) DiscoverEndpoints(ctx context.Context) ([]string, error) {
	endpoints := make(map[string]bool) // Use map to deduplicate

	// 1. Check Docker Compose files
	if dockerEndpoints := d.discoverFromDockerCompose(); len(dockerEndpoints) > 0 {
		for _, ep := range dockerEndpoints {
			endpoints[ep] = true
		}
	}

	// 2. Check environment files
	if envEndpoints := d.discoverFromEnvFiles(); len(envEndpoints) > 0 {
		for _, ep := range envEndpoints {
			endpoints[ep] = true
		}
	}

	// 3. Check running processes (docker ps, etc.)
	if processEndpoints := d.discoverFromRunningProcesses(); len(processEndpoints) > 0 {
		for _, ep := range processEndpoints {
			endpoints[ep] = true
		}
	}

	// 4. Check well-known URLs
	for _, url := range d.wellKnownURLs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if d.isEndpointReachable(ctx, url) {
				endpoints[url] = true
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(endpoints))
	for ep := range endpoints {
		result = append(result, ep)
	}

	return result, nil
}

// ValidateEndpoint checks if an endpoint is valid and reachable
func (d *APIEndpointDiscoverer) ValidateEndpoint(ctx context.Context, endpoint string) error {
	// Validate URL format
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	// Try to reach the endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/health", nil)
	if err != nil {
		// Try without /health
		req, err = http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("endpoint not reachable: %w", err)
	}
	defer resp.Body.Close()

	// Accept any 2xx or 3xx status code
	if resp.StatusCode >= 400 {
		return fmt.Errorf("endpoint returned error status: %d", resp.StatusCode)
	}

	return nil
}

// discoverFromDockerCompose searches for API endpoints in docker-compose files
func (d *APIEndpointDiscoverer) discoverFromDockerCompose() []string {
	var endpoints []string

	composeFiles := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
		"docker-compose.dev.yml",
		"docker-compose.local.yml",
	}

	for _, filename := range composeFiles {
		if data, err := os.ReadFile(filename); err == nil {
			// Parse YAML
			var compose map[string]interface{}
			if err := yaml.Unmarshal(data, &compose); err == nil {
				// Look for services
				if services, ok := compose["services"].(map[string]interface{}); ok {
					for serviceName, serviceData := range services {
						if service, ok := serviceData.(map[string]interface{}); ok {
							// Check for kilometers-api or api service
							if strings.Contains(serviceName, "api") || strings.Contains(serviceName, "kilometers") {
								// Look for ports
								if ports, ok := service["ports"].([]interface{}); ok {
									for _, port := range ports {
										if portStr, ok := port.(string); ok {
											if ep := d.parsePortMapping(portStr); ep != "" {
												endpoints = append(endpoints, ep)
											}
										}
									}
								}

								// Look for environment variables
								if env, ok := service["environment"].([]interface{}); ok {
									for _, e := range env {
										if envStr, ok := e.(string); ok {
											if strings.Contains(envStr, "API_ENDPOINT") || strings.Contains(envStr, "API_URL") {
												parts := strings.SplitN(envStr, "=", 2)
												if len(parts) == 2 {
													endpoints = append(endpoints, parts[1])
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return endpoints
}

// discoverFromEnvFiles searches for API endpoints in .env files
func (d *APIEndpointDiscoverer) discoverFromEnvFiles() []string {
	var endpoints []string

	envFiles := []string{
		".env",
		".env.local",
		".env.development",
		".env.dev",
	}

	// Regular expression to match API URLs
	urlRegex := regexp.MustCompile(`(?i)(API_ENDPOINT|API_URL|KILOMETERS_API_ENDPOINT|BACKEND_URL)\s*=\s*["']?([^"'\s]+)["']?`)

	for _, filename := range envFiles {
		if data, err := os.ReadFile(filename); err == nil {
			matches := urlRegex.FindAllStringSubmatch(string(data), -1)
			for _, match := range matches {
				if len(match) > 2 {
					endpoints = append(endpoints, match[2])
				}
			}
		}
	}

	// Also check in parent directories (up to 2 levels)
	for i := 1; i <= 2; i++ {
		parentPath := strings.Repeat("../", i)
		for _, filename := range envFiles {
			path := filepath.Join(parentPath, filename)
			if data, err := os.ReadFile(path); err == nil {
				matches := urlRegex.FindAllStringSubmatch(string(data), -1)
				for _, match := range matches {
					if len(match) > 2 {
						endpoints = append(endpoints, match[2])
					}
				}
			}
		}
	}

	return endpoints
}

// discoverFromRunningProcesses checks for API endpoints from running processes
func (d *APIEndpointDiscoverer) discoverFromRunningProcesses() []string {
	var endpoints []string

	// Check if docker is available
	if dockerEndpoints := d.checkDockerContainers(); len(dockerEndpoints) > 0 {
		endpoints = append(endpoints, dockerEndpoints...)
	}

	// Check for common development servers
	if devEndpoints := d.checkDevelopmentServers(); len(devEndpoints) > 0 {
		endpoints = append(endpoints, devEndpoints...)
	}

	return endpoints
}

// checkDockerContainers looks for running docker containers
func (d *APIEndpointDiscoverer) checkDockerContainers() []string {
	var endpoints []string

	// Execute docker ps and look for kilometers-api containers
	// This is a simplified check - in production you'd use Docker API
	cmd := "docker ps --format '{{.Names}}\t{{.Ports}}' 2>/dev/null | grep -i 'kilometers\\|api'"
	if output, err := executeCommand(cmd); err == nil {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			// Parse port mappings
			parts := strings.Split(line, "\t")
			if len(parts) >= 2 {
				ports := parts[1]
				// Extract port mappings like "0.0.0.0:5194->5194/tcp"
				portRegex := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+):(\d+)->`)
				matches := portRegex.FindAllStringSubmatch(ports, -1)
				for _, match := range matches {
					if len(match) >= 3 {
						host := match[1]
						port := match[2]
						if host == "0.0.0.0" {
							host = "localhost"
						}
						endpoints = append(endpoints, fmt.Sprintf("http://%s:%s", host, port))
					}
				}
			}
		}
	}

	return endpoints
}

// checkDevelopmentServers checks for common development servers
func (d *APIEndpointDiscoverer) checkDevelopmentServers() []string {
	var endpoints []string

	// Common development ports
	devPorts := []int{3000, 3001, 4000, 5000, 5194, 8000, 8080, 8081}

	for _, port := range devPorts {
		endpoint := fmt.Sprintf("http://localhost:%d", port)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if d.isEndpointReachable(ctx, endpoint) {
			// Try to verify it's a kilometers API
			if d.isKilometersAPI(ctx, endpoint) {
				endpoints = append(endpoints, endpoint)
			}
		}
	}

	return endpoints
}

// isEndpointReachable checks if an endpoint is reachable
func (d *APIEndpointDiscoverer) isEndpointReachable(ctx context.Context, endpoint string) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return false
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < 500
}

// isKilometersAPI tries to verify if an endpoint is a Kilometers API
func (d *APIEndpointDiscoverer) isKilometersAPI(ctx context.Context, endpoint string) bool {
	// Try common API paths
	paths := []string{"/health", "/api/health", "/api/version", "/version"}

	for _, path := range paths {
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint+path, nil)
		if err != nil {
			continue
		}

		resp, err := d.httpClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			// Try to parse response
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				// Look for indicators that this is a Kilometers API
				if _, ok := result["kilometers"]; ok {
					return true
				}
				if service, ok := result["service"].(string); ok && strings.Contains(strings.ToLower(service), "kilometers") {
					return true
				}
			}
			// Even if we can't parse, a 200 response is promising
			return true
		}
	}

	return false
}

// parsePortMapping parses Docker port mapping to endpoint
func (d *APIEndpointDiscoverer) parsePortMapping(portStr string) string {
	// Handle formats like "5194:5194" or "0.0.0.0:5194:5194"
	parts := strings.Split(portStr, ":")
	if len(parts) >= 2 {
		port := parts[len(parts)-2] // Get the host port
		return fmt.Sprintf("http://localhost:%s", port)
	}
	return ""
}

// executeCommand executes a shell command and returns output
func executeCommand(cmd string) (string, error) {
	// This is a simplified version - in production use exec.Command
	return "", fmt.Errorf("not implemented")
}
