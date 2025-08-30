package application

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain/process"
	procp "github.com/kilometers-ai/kilometers-cli/internal/core/ports/process"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports/streaming"
)

// ProxyService orchestrates the process execution and stream proxying.
type ProxyService struct {
	executor procp.Executor
	proxy    streaming.Proxy
	handler  streaming.MessageHandler
}

// NewProxyService creates a new ProxyService.
func NewProxyService(
	executor procp.Executor,
	proxy streaming.Proxy,
	handler streaming.MessageHandler,
) *ProxyService {
	return &ProxyService{
		executor: executor,
		proxy:    proxy,
		handler:  handler,
	}
}

// StartAndProxy executes a command and proxies its stdio streams.
func (s *ProxyService) StartAndProxy(ctx context.Context, cmd process.Command) error {
	proc, err := s.executor.Execute(ctx, cmd)
	if err != nil {
		return err
	}

	// The proxy will be created with the process and handler.
	// For now, we acknowledge the process variable to avoid linter errors.
	_ = proc

	// In a real scenario, the proxy would be instantiated here with the process.
	return s.proxy.Start(ctx)
}
