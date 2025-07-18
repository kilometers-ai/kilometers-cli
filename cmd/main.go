package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"kilometers.ai/cli/internal/interfaces/cli"
	"kilometers.ai/cli/internal/interfaces/di"
)

func main() {
	// Initialize dependency injection container
	container, err := di.NewContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		container.Logger.Println("Received shutdown signal, shutting down gracefully...")
		cancel()

		// Perform graceful shutdown
		if err := container.Shutdown(ctx); err != nil {
			container.Logger.Printf("Error during shutdown: %v", err)
		}
		os.Exit(0)
	}()

	// Execute CLI with dependency injection
	cli.Execute(container.GetCLIContainer())
}
