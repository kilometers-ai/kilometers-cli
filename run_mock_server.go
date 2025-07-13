package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kilometers.ai/cli/test"
)

func main() {
	fmt.Println("🚀 Starting Mock MCP Server...")

	// Create and configure the mock server
	server := test.NewMockMCPServer()

	// Start the server in stdio mode (which is what the km tool expects)
	if err := server.StartStdioMode(); err != nil {
		log.Fatalf("Failed to start mock MCP server: %v", err)
	}

	fmt.Println("✅ Mock MCP Server running in stdio mode")
	fmt.Println("📡 Listening for JSON-RPC messages on stdin/stdout")
	fmt.Println("🛑 Press Ctrl+C to stop")

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan

	fmt.Println("\n🛑 Shutting down mock MCP server...")

	// Give the server a moment to clean up
	time.Sleep(100 * time.Millisecond)

	fmt.Println("✅ Mock MCP Server stopped")
}
