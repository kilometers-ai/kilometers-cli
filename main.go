// Kilometers CLI - CI/CD Pipeline Test
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	BuildTime = "unknown" // Overridden by ldflags
)

func handleCommands() bool {
	if len(os.Args) < 2 {
		return false
	}

	switch os.Args[1] {
	case "--version", "-v", "version":
		printVersion()
		return true
	case "--help", "-h", "help":
		printHelp()
		return true
	case "update":
		handleUpdate()
		return true
	case "init":
		handleInit()
		return true
	case "setup":
		handleSetup()
		return true
	case "validate":
		handleValidate()
		return true
	case "config":
		handleConfigCommand()
		return true
	}

	return false
}

// printVersion displays version information
func printVersion() {
	fmt.Printf("Kilometers CLI %s\n", Version)
	fmt.Printf("Build time: %s\n", BuildTime)
	fmt.Printf("Go version: %s\n", goVersion())
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// goVersion returns the Go version used to build the binary
func goVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.GoVersion
	}
	return "unknown"
}

func handleUpdate() {
	fmt.Printf("km cli tool @ %s\n", Version)
	fmt.Println("Update functionality coming soon!")
	fmt.Println("")
	fmt.Println("For now, download the latest version from:")
	fmt.Println("  https://github.com/kilometers-ai/kilometers/releases/latest")
}

func handleInit() {
	fmt.Println("🚀 Kilometers CLI Configuration Setup")
	fmt.Println("")
	fmt.Println("This will guide you through setting up your Kilometers CLI configuration.")
	fmt.Println("You can press Enter to accept default values shown in brackets.")
	fmt.Println("")

	scanner := bufio.NewScanner(os.Stdin)
	config := DefaultConfig()

	// API Key (required)
	fmt.Print("API Key (required): ")
	scanner.Scan()
	apiKey := strings.TrimSpace(scanner.Text())
	if apiKey == "" {
		fmt.Println("❌ API Key is required. Get yours from https://app.dev.kilometers.ai")
		os.Exit(1)
	}
	config.APIKey = apiKey

	// API URL (optional, default to production)
	fmt.Printf("API URL [%s]: ", "https://api.dev.kilometers.ai")
	scanner.Scan()
	apiURL := strings.TrimSpace(scanner.Text())
	if apiURL == "" {
		apiURL = "https://api.dev.kilometers.ai"
	}
	config.APIEndpoint = apiURL

	// Customer ID (optional, default to "default")
	fmt.Printf("Customer ID [%s]: ", "default")
	scanner.Scan()
	customerID := strings.TrimSpace(scanner.Text())
	if customerID == "" {
		customerID = "default"
	}
	// Note: Customer ID is not currently in the Config struct, it's an env var only
	// We'll save it as an environment variable suggestion

	// Debug mode (optional)
	fmt.Print("Enable debug mode? (y/N): ")
	scanner.Scan()
	debugResponse := strings.TrimSpace(strings.ToLower(scanner.Text()))
	config.Debug = debugResponse == "y" || debugResponse == "yes"

	// Batch size (optional)
	fmt.Printf("Batch size [%d]: ", config.BatchSize)
	scanner.Scan()
	batchSizeStr := strings.TrimSpace(scanner.Text())
	if batchSizeStr != "" {
		if batchSize, err := strconv.Atoi(batchSizeStr); err == nil && batchSize > 0 {
			config.BatchSize = batchSize
		}
	}

	// Save configuration
	if err := SaveConfig(config); err != nil {
		fmt.Printf("❌ Failed to save configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Println("✅ Configuration saved successfully!")
	fmt.Printf("   Config file: %s\n", getConfigPath())
	fmt.Println("")

	// Show environment variables for session
	fmt.Println("📝 For your current session, you can also set these environment variables:")
	fmt.Printf("   export KILOMETERS_API_KEY=\"%s\"\n", apiKey)
	fmt.Printf("   export KILOMETERS_API_URL=\"%s\"\n", apiURL)
	fmt.Printf("   export KILOMETERS_CUSTOMER_ID=\"%s\"\n", customerID)
	if config.Debug {
		fmt.Println("   export KM_DEBUG=true")
	}
	fmt.Println("")

	fmt.Println("🎉 Ready to use Kilometers CLI!")
	fmt.Println("")
	fmt.Println("Try it out:")
	fmt.Println("   km npx @modelcontextprotocol/server-github")
	fmt.Println("")
	fmt.Println("Dashboard: https://app.dev.kilometers.ai")
}

func handleSetup() {
	if len(os.Args) < 3 {
		fmt.Println("❌ Setup command requires an AI assistant type")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  km setup claude-desktop")
		fmt.Println("  km setup vscode")
		fmt.Println("  km setup chatgpt")
		fmt.Println("")
		os.Exit(1)
	}

	assistant := os.Args[2]
	switch assistant {
	case "claude-desktop":
		handleClaudeDesktopSetup()
	case "vscode":
		handleVSCodeSetup()
	case "chatgpt":
		handleChatGPTSetup()
	default:
		fmt.Printf("❌ Unknown AI assistant: %s\n", assistant)
		fmt.Println("")
		fmt.Println("Supported assistants:")
		fmt.Println("  claude-desktop")
		fmt.Println("  vscode")
		fmt.Println("  chatgpt")
		os.Exit(1)
	}
}

func handleValidate() {
	fmt.Println("🔍 Kilometers CLI Validation")
	fmt.Println("")

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		fmt.Println("Run 'km init' to set up your configuration")
		os.Exit(1)
	}

	// Test API connection
	logger := log.New(os.Stderr, "[km] ", log.LstdFlags)
	apiClient := NewAPIClient(config, logger)

	fmt.Print("Testing API connection... ")
	if err := apiClient.TestConnection(); err != nil {
		fmt.Printf("❌ Failed\n")
		fmt.Printf("Error: %v\n", err)
		fmt.Println("")
		fmt.Println("Please check:")
		fmt.Println("  1. Your API key is correct")
		fmt.Println("  2. Your internet connection")
		fmt.Println("  3. The API endpoint is accessible")
		os.Exit(1)
	}
	fmt.Println("✅ Connected")

	// Send test events to validate the pipeline
	fmt.Print("Sending test events... ")
	if err := sendTestEvents(apiClient); err != nil {
		fmt.Printf("❌ Failed\n")
		fmt.Printf("Error: %v\n", err)
		fmt.Println("")
		fmt.Println("The API connection works, but test events failed.")
		fmt.Println("This might be normal if the API doesn't support test events yet.")
	} else {
		fmt.Println("✅ Test events sent successfully")
	}

	fmt.Println("")
	fmt.Println("✅ Configuration is valid")
	fmt.Println("")
	fmt.Println("Your Kilometers CLI is ready to monitor MCP events!")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("1. Configure your AI assistant with 'km setup <assistant>'")
	fmt.Println("2. Start using your AI assistant with MCP servers")
	fmt.Println("3. Check your dashboard at https://app.dev.kilometers.ai for events")
}

func handleConfigCommand() {
	if len(os.Args) < 3 {
		fmt.Println("❌ Config command requires a subcommand")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  km config show")
		fmt.Println("")
		os.Exit(1)
	}

	subcommand := os.Args[2]
	switch subcommand {
	case "show":
		handleConfigShow()
	default:
		fmt.Printf("❌ Unknown config subcommand: %s\n", subcommand)
		fmt.Println("")
		fmt.Println("Available subcommands:")
		fmt.Println("  show")
		os.Exit(1)
	}
}

func handleConfigShow() {
	fmt.Println("📋 Kilometers CLI Configuration")
	fmt.Println("")

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		fmt.Println("Run 'km init' to set up your configuration")
		return
	}

	// Display configuration
	fmt.Printf("Config file: %s\n", getConfigPath())
	fmt.Printf("API Endpoint: %s\n", config.APIEndpoint)

	// Mask API key for security
	if config.APIKey != "" {
		maskedKey := config.APIKey[:8] + "..." + config.APIKey[len(config.APIKey)-4:]
		fmt.Printf("API Key: %s\n", maskedKey)
	} else {
		fmt.Println("API Key: (not set)")
	}

	fmt.Printf("Batch Size: %d\n", config.BatchSize)
	fmt.Printf("Debug Mode: %t\n", config.Debug)
	fmt.Printf("Risk Detection: %t\n", config.EnableRiskDetection)

	if len(config.MethodWhitelist) > 0 {
		fmt.Printf("Method Whitelist: %v\n", config.MethodWhitelist)
	}

	if config.PayloadSizeLimit > 0 {
		fmt.Printf("Payload Size Limit: %d bytes\n", config.PayloadSizeLimit)
	}

	fmt.Printf("High Risk Methods Only: %t\n", config.HighRiskMethodsOnly)
	fmt.Printf("Exclude Ping Messages: %t\n", config.ExcludePingMessages)
}

func handleClaudeDesktopSetup() {
	fmt.Println("🤖 Claude Desktop Setup")
	fmt.Println("")

	// Load configuration to get API key
	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		fmt.Println("Run 'km init' to set up your configuration first")
		os.Exit(1)
	}

	if config.APIKey == "" {
		fmt.Println("❌ API key not found in configuration")
		fmt.Println("Run 'km init' to set up your API key first")
		os.Exit(1)
	}

	// Detect Claude Desktop config file location
	configPath := getClaudeDesktopConfigPath()
	if configPath == "" {
		fmt.Println("❌ Could not find Claude Desktop config directory")
		fmt.Println("")
		fmt.Println("Please ensure Claude Desktop is installed and has been run at least once.")
		fmt.Println("")
		printClaudeDesktopManualInstructions(config.APIKey)
		os.Exit(1)
	}

	fmt.Printf("Found Claude Desktop config at: %s\n", configPath)
	fmt.Println("")

	// TODO: Implement config file backup and modification
	fmt.Println("⚠️  Automatic configuration is not yet implemented.")
	fmt.Println("Please manually configure Claude Desktop using the instructions below:")
	fmt.Println("")
	printClaudeDesktopManualInstructions(config.APIKey)
}

func handleVSCodeSetup() {
	fmt.Println("💻 VS Code Setup")
	fmt.Println("")

	// Load configuration to get API key
	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		fmt.Println("Run 'km init' to set up your configuration first")
		os.Exit(1)
	}

	if config.APIKey == "" {
		fmt.Println("❌ API key not found in configuration")
		fmt.Println("Run 'km init' to set up your API key first")
		os.Exit(1)
	}

	fmt.Println("📋 VS Code MCP Extension Configuration")
	fmt.Println("")
	fmt.Println("1. Install the MCP extension in VS Code")
	fmt.Println("2. Open VS Code Settings (Cmd/Ctrl + ,)")
	fmt.Println("3. Search for 'MCP' in settings")
	fmt.Println("4. Configure the MCP servers using this template:")
	fmt.Println("")
	fmt.Println("```json")
	fmt.Println(GetVSCodeTemplate())
	fmt.Println("```")
	fmt.Println("")
	fmt.Println("5. Replace the placeholder values with your actual tokens")
	fmt.Println("6. Save the settings")
	fmt.Println("7. Restart VS Code")
	fmt.Println("")
	fmt.Printf("Your Kilometers API key (%s) is already configured in the CLI.\n", config.APIKey)
	fmt.Println("The km wrapper will automatically use it to send events to your dashboard.")
	fmt.Println("")
	PrintMCPServerList()
}

func handleChatGPTSetup() {
	fmt.Println("🤖 ChatGPT Desktop Setup")
	fmt.Println("")

	// Load configuration to get API key
	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		fmt.Println("Run 'km init' to set up your configuration first")
		os.Exit(1)
	}

	if config.APIKey == "" {
		fmt.Println("❌ API key not found in configuration")
		fmt.Println("Run 'km init' to set up your API key first")
		os.Exit(1)
	}

	fmt.Println("⚠️  ChatGPT Desktop setup is not yet implemented.")
	fmt.Println("Please refer to the ChatGPT Desktop documentation for MCP setup instructions.")
	fmt.Println("")
	fmt.Printf("Your API key: %s\n", config.APIKey)
}

// getClaudeDesktopConfigPath returns the path to Claude Desktop's config file
func getClaudeDesktopConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	var configPath string
	switch runtime.GOOS {
	case "darwin": // macOS
		configPath = filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		configPath = filepath.Join(homeDir, "AppData", "Roaming", "Claude", "claude_desktop_config.json")
	case "linux":
		configPath = filepath.Join(homeDir, ".config", "claude", "claude_desktop_config.json")
	default:
		return ""
	}

	// Check if the directory exists (Claude has been run at least once)
	if _, err := os.Stat(filepath.Dir(configPath)); os.IsNotExist(err) {
		return ""
	}

	return configPath
}

// printClaudeDesktopManualInstructions prints manual setup instructions for Claude Desktop
func printClaudeDesktopManualInstructions(apiKey string) {
	fmt.Println("📋 Manual Claude Desktop Configuration")
	fmt.Println("")
	fmt.Println("1. Open Claude Desktop")
	fmt.Println("2. Go to Settings (gear icon)")
	fmt.Println("3. Click on 'Developer' tab")
	fmt.Println("4. Edit the MCP servers configuration")
	fmt.Println("5. Add this configuration:")
	fmt.Println("")
	fmt.Println("```json")
	fmt.Println(GetClaudeDesktopTemplateJSON())
	fmt.Println("```")
	fmt.Println("")
	fmt.Println("6. Replace the placeholder values with your actual tokens:")
	fmt.Println("   - 'your_github_token_here' with your GitHub Personal Access Token")
	fmt.Println("   - 'your_brave_api_key_here' with your Brave Search API key")
	fmt.Println("   - '/Users/your-username/Documents' with your actual documents path")
	fmt.Println("7. Remove any MCP servers you don't want to use")
	fmt.Println("8. Save the configuration")
	fmt.Println("9. Restart Claude Desktop")
	fmt.Println("")
	fmt.Printf("Your Kilometers API key (%s) is already configured in the CLI.\n", apiKey)
	fmt.Println("The km wrapper will automatically use it to send events to your dashboard.")
	fmt.Println("")
	PrintMCPServerList()
}

// sendTestEvents sends test MCP events to validate the pipeline
func sendTestEvents(apiClient *APIClient) error {
	if apiClient == nil {
		return fmt.Errorf("API client is nil")
	}

	// Create a test event
	testEvent := Event{
		ID:        fmt.Sprintf("test-%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Direction: "request",
		Method:    "tools/list",
		Payload:   []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`),
		Size:      50,
		RiskScore: 0,
	}

	// Send the test event
	events := []Event{testEvent}
	return apiClient.SendEventBatch(events)
}

type MCPMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Event struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Direction string    `json:"direction"` // "request" | "response"
	Method    string    `json:"method,omitempty"`
	Payload   []byte    `json:"payload"`
	Size      int       `json:"size"`
	RiskScore int       `json:"risk_score,omitempty"` // Client-side risk assessment
}

type ProcessWrapper struct {
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	events       chan Event
	wg           sync.WaitGroup
	logger       *log.Logger
	config       *Config
	apiClient    *APIClient
	eventBatch   []Event
	batchMutex   sync.Mutex
	riskDetector *RiskDetector

	// Filtering statistics
	totalEvents    int
	filteredEvents int
	statsMutex     sync.Mutex
}

func main() {
	if handleCommands() {
		return
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := ValidateConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := log.New(os.Stderr, "[km] ", log.LstdFlags)
	if config.Debug {
		logger.Printf("Debug mode enabled")
		logger.Printf("Configuration: API=%s, BatchSize=%d", config.APIEndpoint, config.BatchSize)
		logger.Printf("Filtering: RiskDetection=%v, MethodWhitelist=%v, PayloadLimit=%d",
			config.EnableRiskDetection, config.MethodWhitelist, config.PayloadSizeLimit)
	}

	logger.Printf("Starting Kilometers CLI wrapper for: %v", os.Args[1:])

	// Create API client
	apiClient := NewAPIClient(config, logger)

	// Test API connection
	if err := apiClient.TestConnection(); err != nil {
		logger.Printf("Warning: API connection test failed: %v", err)
		logger.Printf("Events will be logged locally only")
		apiClient = nil // Disable API client for this session
	}

	// Create and start the process wrapper
	wrapper, err := NewProcessWrapper(os.Args[1], os.Args[2:], config, apiClient, logger)
	if err != nil {
		logger.Fatalf("Failed to create process wrapper: %v", err)
	}

	// Start the wrapper
	if err := wrapper.Start(); err != nil {
		logger.Fatalf("Failed to start process wrapper: %v", err)
	}

	// Wait for completion
	if err := wrapper.Wait(); err != nil {
		logger.Printf("Process exited with error: %v", err)
		os.Exit(1)
	}

	// Print filtering statistics if any filtering was enabled
	wrapper.printFilteringStats()

	logger.Printf("Process completed successfully")
}

// NewProcessWrapper creates a new process wrapper
func NewProcessWrapper(command string, args []string, config *Config, apiClient *APIClient, logger *log.Logger) (*ProcessWrapper, error) {
	cmd := exec.Command(command, args...)

	// Create pipes for monitoring
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Initialize risk detector
	riskDetector := NewRiskDetector()

	return &ProcessWrapper{
		cmd:          cmd,
		stdin:        stdin,
		stdout:       stdout,
		stderr:       stderr,
		events:       make(chan Event, 100), // Buffered channel for events
		logger:       logger,
		config:       config,
		apiClient:    apiClient,
		eventBatch:   make([]Event, 0, config.BatchSize),
		riskDetector: riskDetector,
	}, nil
}

// Start begins the process and monitoring
func (pw *ProcessWrapper) Start() error {
	// Start the wrapped process
	if err := pw.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	pw.logger.Printf("Started process with PID: %d", pw.cmd.Process.Pid)

	// Start monitoring goroutines
	pw.wg.Add(4)
	go pw.monitorStdin()
	go pw.monitorStdout()
	go pw.forwardStderr()
	go pw.processEvents()

	return nil
}

// Wait waits for the process to complete
func (pw *ProcessWrapper) Wait() error {
	// Wait for the process to finish
	err := pw.cmd.Wait()

	// Give a brief moment for any final events to be processed
	time.Sleep(100 * time.Millisecond)

	// Close the events channel to signal event processor to stop
	close(pw.events)

	// Wait for all goroutines to finish
	pw.wg.Wait()

	return err
}

// monitorStdin reads from os.Stdin and forwards to the wrapped process while monitoring
func (pw *ProcessWrapper) monitorStdin() {
	defer pw.wg.Done()
	defer pw.stdin.Close()
	defer func() {
		if r := recover(); r != nil {
			// Gracefully handle panic from closed channel
			pw.logger.Printf("Monitor stdin exiting: %v", r)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Parse and potentially capture the request
		if msg := pw.parseMCPMessage(line); msg != nil {
			pw.incrementTotalEvents()

			// Apply filtering logic
			if pw.riskDetector.ShouldCaptureEvent(msg, line, pw.config) {
				// Calculate risk score for captured events
				riskScore := pw.riskDetector.AnalyzeEvent(msg, line)

				event := Event{
					ID:        pw.generateEventID(),
					Timestamp: time.Now(),
					Direction: "request",
					Method:    msg.Method,
					Payload:   line,
					Size:      len(line),
					RiskScore: riskScore,
				}

				// Log risk information in debug mode
				if pw.config.Debug {
					pw.logger.Printf("Captured request: method=%s, risk=%s(%d), size=%d",
						msg.Method, GetRiskLabel(riskScore), riskScore, len(line))
				}

				// Send event to processing channel (non-blocking)
				select {
				case pw.events <- event:
				default:
					pw.logger.Printf("Warning: event buffer full, dropping event")
				}
			} else {
				pw.incrementFilteredEvents()
				if pw.config.Debug {
					riskScore := pw.riskDetector.AnalyzeEvent(msg, line)
					pw.logger.Printf("Filtered request: method=%s, risk=%s(%d), size=%d",
						msg.Method, GetRiskLabel(riskScore), riskScore, len(line))
				}
			}
		}

		// Always forward the message (transparency requirement)
		if _, err := pw.stdin.Write(append(line, '\n')); err != nil {
			pw.logger.Printf("Error writing to stdin: %v", err)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		pw.logger.Printf("Error reading from stdin: %v", err)
	}
}

// monitorStdout reads from the wrapped process stdout and forwards to os.Stdout while monitoring
func (pw *ProcessWrapper) monitorStdout() {
	defer pw.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			// Gracefully handle panic from closed channel
			pw.logger.Printf("Monitor stdout exiting: %v", r)
		}
	}()

	scanner := bufio.NewScanner(pw.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Parse and potentially capture the response
		if msg := pw.parseMCPMessage(line); msg != nil {
			pw.incrementTotalEvents()

			// Apply filtering logic
			if pw.riskDetector.ShouldCaptureEvent(msg, line, pw.config) {
				// Calculate risk score for captured events
				riskScore := pw.riskDetector.AnalyzeEvent(msg, line)

				event := Event{
					ID:        pw.generateEventID(),
					Timestamp: time.Now(),
					Direction: "response",
					Method:    msg.Method,
					Payload:   line,
					Size:      len(line),
					RiskScore: riskScore,
				}

				// Log risk information in debug mode
				if pw.config.Debug {
					pw.logger.Printf("Captured response: method=%s, risk=%s(%d), size=%d",
						msg.Method, GetRiskLabel(riskScore), riskScore, len(line))
				}

				// Send event to processing channel (non-blocking)
				select {
				case pw.events <- event:
				default:
					pw.logger.Printf("Warning: event buffer full, dropping event")
				}
			} else {
				pw.incrementFilteredEvents()
				if pw.config.Debug {
					riskScore := pw.riskDetector.AnalyzeEvent(msg, line)
					pw.logger.Printf("Filtered response: method=%s, risk=%s(%d), size=%d",
						msg.Method, GetRiskLabel(riskScore), riskScore, len(line))
				}
			}
		}

		// Always forward the message (transparency requirement)
		if _, err := os.Stdout.Write(append(line, '\n')); err != nil {
			pw.logger.Printf("Error writing to stdout: %v", err)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		pw.logger.Printf("Error reading from stdout: %v", err)
	}
}

// forwardStderr simply forwards stderr from the wrapped process
func (pw *ProcessWrapper) forwardStderr() {
	defer pw.wg.Done()

	_, err := io.Copy(os.Stderr, pw.stderr)
	if err != nil {
		pw.logger.Printf("Error forwarding stderr: %v", err)
	}
}

// processEvents handles captured events and sends them to the API
func (pw *ProcessWrapper) processEvents() {
	defer pw.wg.Done()

	// Create a ticker for periodic batch flushing
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	eventCount := 0
	for {
		select {
		case event, ok := <-pw.events:
			if !ok {
				// Channel closed, flush remaining events and exit
				pw.flushBatch()
				pw.logger.Printf("Processed %d total events", eventCount)
				return
			}

			eventCount++

			// Log the event
			pw.logger.Printf("Event #%d: %s %s (%d bytes)",
				eventCount, event.Direction, event.Method, event.Size)

			// In debug mode, log the payload
			if pw.config.Debug {
				pw.logger.Printf("Payload: %s", string(event.Payload))
			}

			// Add to batch if API client is available
			if pw.apiClient != nil {
				pw.addToBatch(event)
			}

		case <-ticker.C:
			// Periodic flush of batched events
			if pw.apiClient != nil {
				pw.flushBatch()
			}
		}
	}
}

// addToBatch adds an event to the current batch
func (pw *ProcessWrapper) addToBatch(event Event) {
	pw.batchMutex.Lock()
	defer pw.batchMutex.Unlock()

	pw.eventBatch = append(pw.eventBatch, event)

	// Send batch if it's full
	if len(pw.eventBatch) >= pw.config.BatchSize {
		pw.sendBatch()
	}
}

// flushBatch sends any remaining events in the batch
func (pw *ProcessWrapper) flushBatch() {
	pw.batchMutex.Lock()
	defer pw.batchMutex.Unlock()

	if len(pw.eventBatch) > 0 {
		pw.sendBatch()
	}
}

// sendBatch sends the current batch to the API (must be called with mutex held)
func (pw *ProcessWrapper) sendBatch() {
	if len(pw.eventBatch) == 0 {
		return
	}

	// Create a copy of the batch
	batch := make([]Event, len(pw.eventBatch))
	copy(batch, pw.eventBatch)

	// Clear the current batch
	pw.eventBatch = pw.eventBatch[:0]

	// Send synchronously to ensure completion before process exit
	if err := pw.apiClient.SendEventBatch(batch); err != nil {
		pw.logger.Printf("Failed to send batch to API: %v", err)
		// TODO: Implement retry logic or local storage fallback
	}
}

// parseMCPMessage attempts to parse a JSON-RPC message
func (pw *ProcessWrapper) parseMCPMessage(data []byte) *MCPMessage {
	var msg MCPMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		// Not valid JSON or not an MCP message, that's ok
		return nil
	}

	// Validate it's a JSON-RPC 2.0 message
	if msg.JSONRPC != "2.0" {
		return nil
	}

	return &msg
}

// generateEventID creates a unique event identifier
func (pw *ProcessWrapper) generateEventID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), len(pw.eventBatch))
}

// printUsage prints the basic usage information
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <mcp-server-command> [args...]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nExample: %s npx @modelcontextprotocol/server-github\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nUse --help for more information\n")
}

// incrementTotalEvents increments the total events counter
func (pw *ProcessWrapper) incrementTotalEvents() {
	pw.statsMutex.Lock()
	defer pw.statsMutex.Unlock()
	pw.totalEvents++
}

// incrementFilteredEvents increments the filtered events counter
func (pw *ProcessWrapper) incrementFilteredEvents() {
	pw.statsMutex.Lock()
	defer pw.statsMutex.Unlock()
	pw.filteredEvents++
}

// printFilteringStats prints the filtering statistics
func (pw *ProcessWrapper) printFilteringStats() {
	pw.statsMutex.Lock()
	defer pw.statsMutex.Unlock()

	if pw.totalEvents > 0 {
		filteredRatio := float64(pw.filteredEvents) / float64(pw.totalEvents) * 100
		pw.logger.Printf("Filtering statistics:")
		pw.logger.Printf("Total events: %d", pw.totalEvents)
		pw.logger.Printf("Filtered events: %d (%.2f%%)", pw.filteredEvents, filteredRatio)
	} else {
		pw.logger.Printf("No events processed yet")
	}
}

// Date-based build test Sat Jun 28 03:52:01 EDT 2025
