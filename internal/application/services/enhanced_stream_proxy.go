package services

import (
	"context"
	"fmt"
	"os"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins"
)

// EnhancedStreamProxy manages stdin/stdout streams with plugin integration
type EnhancedStreamProxy struct {
	process       ports.Process
	correlationID string
	config        domain.MonitorConfig
	messageLogger ports.MessageHandler
	pluginManager ports.PluginManager
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewEnhancedStreamProxy creates a new enhanced stream proxy
func NewEnhancedStreamProxy(
	process ports.Process,
	correlationID string,
	config domain.MonitorConfig,
	logger ports.MessageHandler,
	pluginManager ports.PluginManager,
) *EnhancedStreamProxy {
	return &EnhancedStreamProxy{
		process:       process,
		correlationID: correlationID,
		config:        config,
		messageLogger: logger,
		pluginManager: pluginManager,
	}
}

// Start begins the enhanced proxy operation
func (p *EnhancedStreamProxy) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)
	defer p.cancel()

	// Start stdin forwarding (from client to server)
	go p.forwardStdin()

	// Start stdout processing (from server to client)
	go p.processStdout()

	// Wait for context cancellation
	<-p.ctx.Done()
	return nil
}

// Stop terminates the proxy
func (p *EnhancedStreamProxy) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
}

// forwardStdin forwards stdin from client to server
func (p *EnhancedStreamProxy) forwardStdin() {
	buffer := make([]byte, p.config.BufferSize)
	
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			// Read from stdin
			n, err := os.Stdin.Read(buffer)
			if err != nil {
				return
			}

			if n > 0 {
				data := buffer[:n]
				
				// Process through plugins before forwarding
				processedData, err := p.processInboundMessage(data)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[Proxy] Inbound processing error: %v\n", err)
					continue
				}

				// Forward to server if not blocked
				if processedData != nil {
					stdin := p.process.Stdin()
					if stdin != nil {
						stdin.Write(processedData)
					}
				}
			}
		}
	}
}

// processStdout processes stdout from server with plugins
func (p *EnhancedStreamProxy) processStdout() {
	buffer := make([]byte, p.config.BufferSize)
	stdout := p.process.Stdout()
	
	if stdout == nil {
		return
	}

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			// Read from server stdout
			n, err := stdout.Read(buffer)
			if err != nil {
				return
			}

			if n > 0 {
				data := buffer[:n]
				
				// Process through plugins
				processedData, err := p.processOutboundMessage(data)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[Proxy] Outbound processing error: %v\n", err)
					continue
				}

				// Forward to client if not blocked
				if processedData != nil {
					os.Stdout.Write(processedData)
				}
			}
		}
	}
}

// processInboundMessage processes messages going from client to server
func (p *EnhancedStreamProxy) processInboundMessage(data []byte) ([]byte, error) {
	// Parse the JSON-RPC message
	message, err := domain.NewJSONRPCMessageFromRaw(
		data,
		domain.DirectionInbound,
		p.correlationID,
	)
	if err != nil {
		// If we can't parse as JSON-RPC, just forward raw data
		return data, nil
	}

	// Log the original message
	p.messageLogger.LogMessage(message)

	// Process through filter plugins
	filteredMessage, err := p.applyFilterPlugins(message)
	if err != nil {
		return nil, fmt.Errorf("message blocked by filter: %w", err)
	}

	// Process through security plugins
	if err := p.applySecurityPlugins(filteredMessage); err != nil {
		fmt.Fprintf(os.Stderr, "[Security] Warning: %v\n", err)
	}

	// Process through analytics plugins
	p.applyAnalyticsPlugins(filteredMessage)

	// Return processed message payload
	return filteredMessage.Payload(), nil
}

// processOutboundMessage processes messages going from server to client
func (p *EnhancedStreamProxy) processOutboundMessage(data []byte) ([]byte, error) {
	// Parse the JSON-RPC message
	message, err := domain.NewJSONRPCMessageFromRaw(
		data,
		domain.DirectionOutbound,
		p.correlationID,
	)
	if err != nil {
		// If we can't parse as JSON-RPC, just forward raw data
		return data, nil
	}

	// Log the original message
	p.messageLogger.LogMessage(message)

	// Process through filter plugins
	filteredMessage, err := p.applyFilterPlugins(message)
	if err != nil {
		return nil, fmt.Errorf("message blocked by filter: %w", err)
	}

	// Process through security plugins
	if err := p.applySecurityPlugins(filteredMessage); err != nil {
		fmt.Fprintf(os.Stderr, "[Security] Warning: %v\n", err)
	}

	// Process through analytics plugins
	p.applyAnalyticsPlugins(filteredMessage)

	// Return processed message payload
	return filteredMessage.Payload(), nil
}

// applyFilterPlugins applies all available filter plugins
func (p *EnhancedStreamProxy) applyFilterPlugins(message *domain.JSONRPCMessage) (*domain.JSONRPCMessage, error) {
	// Type assertion to access plugin manager methods
	pluginManagerImpl, ok := p.pluginManager.(*plugins.PluginManagerImpl)
	if !ok {
		return message, nil
	}

	filterPlugins := pluginManagerImpl.GetFilterPlugins(p.ctx)
	
	for _, filterPlugin := range filterPlugins {
		if filterPlugin.ShouldFilter(p.ctx, message) {
			filteredMessage, err := filterPlugin.FilterMessage(p.ctx, message)
			if err != nil {
				return nil, err
			}
			message = filteredMessage
		}
	}

	return message, nil
}

// applySecurityPlugins applies all available security plugins
func (p *EnhancedStreamProxy) applySecurityPlugins(message *domain.JSONRPCMessage) error {
	pluginManagerImpl, ok := p.pluginManager.(*plugins.PluginManagerImpl)
	if !ok {
		return nil
	}

	securityPlugins := pluginManagerImpl.GetSecurityPlugins(p.ctx)
	
	for _, securityPlugin := range securityPlugins {
		result, err := securityPlugin.CheckSecurity(p.ctx, message)
		if err != nil {
			return fmt.Errorf("security plugin error: %w", err)
		}

		if !result.IsSecure {
			// Log security issues but don't block (just warn)
			for _, issue := range result.Issues {
				fmt.Fprintf(os.Stderr, "[Security] %s: %s (Severity: %s)\n", 
					issue.Type, issue.Description, issue.Severity)
			}
		}
	}

	return nil
}

// applyAnalyticsPlugins applies all available analytics plugins
func (p *EnhancedStreamProxy) applyAnalyticsPlugins(message *domain.JSONRPCMessage) {
	pluginManagerImpl, ok := p.pluginManager.(*plugins.PluginManagerImpl)
	if !ok {
		return
	}

	analyticsPlugins := pluginManagerImpl.GetAnalyticsPlugins(p.ctx)
	
	for _, analyticsPlugin := range analyticsPlugins {
		if err := analyticsPlugin.AnalyzeMessage(p.ctx, message); err != nil {
			fmt.Fprintf(os.Stderr, "[Analytics] Plugin error: %v\n", err)
		}
	}
}
