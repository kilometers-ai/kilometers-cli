package test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

// MockMCPServer implements a JSON-RPC 2.0 compliant MCP server for testing
type MockMCPServer struct {
	listener        net.Listener
	connections     map[string]*MCPConnection
	mu              sync.RWMutex
	responses       map[string]interface{}
	errorInjections map[string]error
	latencyConfig   map[string]time.Duration
	messageHandlers map[string]func(interface{}) interface{}
	isRunning       bool
	serverAddress   string
	connectionCount int
	messageCount    int64
	ctx             context.Context
	cancel          context.CancelFunc
}

// MCPConnection represents a connection to the MCP server
type MCPConnection struct {
	conn       net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
	id         string
	isActive   bool
	lastPing   time.Time
	messageLog []MCPMessage
}

// MCPMessage represents a JSON-RPC 2.0 message
type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewMockMCPServer creates a new mock MCP server
func NewMockMCPServer() *MockMCPServer {
	ctx, cancel := context.WithCancel(context.Background())

	server := &MockMCPServer{
		connections:     make(map[string]*MCPConnection),
		responses:       make(map[string]interface{}),
		errorInjections: make(map[string]error),
		latencyConfig:   make(map[string]time.Duration),
		messageHandlers: make(map[string]func(interface{}) interface{}),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Set up default handlers
	server.SetupDefaultHandlers()

	return server
}

// Start starts the mock MCP server on the specified port
func (s *MockMCPServer) Start(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("server is already running")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	s.listener = listener
	s.serverAddress = listener.Addr().String()
	s.isRunning = true

	go s.acceptConnections()

	return nil
}

// Stop stops the mock MCP server
func (s *MockMCPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("server is not running")
	}

	s.cancel()
	s.isRunning = false

	// Close all connections
	for _, conn := range s.connections {
		conn.conn.Close()
	}

	if s.listener != nil {
		s.listener.Close()
	}

	return nil
}

// acceptConnections handles incoming connections
func (s *MockMCPServer) acceptConnections() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if s.isRunning {
					fmt.Printf("Error accepting connection: %v\n", err)
				}
				continue
			}

			s.handleConnection(conn)
		}
	}
}

// handleConnection handles a new connection
func (s *MockMCPServer) handleConnection(conn net.Conn) {
	s.mu.Lock()
	s.connectionCount++
	connID := fmt.Sprintf("conn-%d", s.connectionCount)

	mcpConn := &MCPConnection{
		conn:       conn,
		reader:     bufio.NewReader(conn),
		writer:     bufio.NewWriter(conn),
		id:         connID,
		isActive:   true,
		lastPing:   time.Now(),
		messageLog: make([]MCPMessage, 0),
	}

	s.connections[connID] = mcpConn
	s.mu.Unlock()

	go s.handleMessages(mcpConn)
}

// handleMessages handles messages from a connection
func (s *MockMCPServer) handleMessages(conn *MCPConnection) {
	defer func() {
		s.mu.Lock()
		delete(s.connections, conn.id)
		s.mu.Unlock()
		conn.conn.Close()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			line, err := conn.reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Error reading from connection %s: %v\n", conn.id, err)
				}
				return
			}

			if len(line) == 0 {
				continue
			}

			var message MCPMessage
			if err := json.Unmarshal([]byte(line), &message); err != nil {
				fmt.Printf("Error parsing message from connection %s: %v\n", conn.id, err)
				continue
			}

			conn.messageLog = append(conn.messageLog, message)
			s.messageCount++

			response := s.processMessage(message)
			if response != nil {
				s.sendMessage(conn, *response)
			}
		}
	}
}

// processMessage processes an incoming message and returns a response
func (s *MockMCPServer) processMessage(message MCPMessage) *MCPMessage {
	// Check for error injection
	if err, exists := s.errorInjections[message.Method]; exists {
		return &MCPMessage{
			JSONRPC: "2.0",
			ID:      message.ID,
			Error: &MCPError{
				Code:    -32603,
				Message: err.Error(),
			},
		}
	}

	// Add latency if configured
	if latency, exists := s.latencyConfig[message.Method]; exists {
		time.Sleep(latency)
	}

	// Check for custom handler
	if handler, exists := s.messageHandlers[message.Method]; exists {
		result := handler(message.Params)
		return &MCPMessage{
			JSONRPC: "2.0",
			ID:      message.ID,
			Result:  result,
		}
	}

	// Check for configured response
	if response, exists := s.responses[message.Method]; exists {
		return &MCPMessage{
			JSONRPC: "2.0",
			ID:      message.ID,
			Result:  response,
		}
	}

	// Default response
	return &MCPMessage{
		JSONRPC: "2.0",
		ID:      message.ID,
		Result:  map[string]interface{}{"status": "ok"},
	}
}

// sendMessage sends a message to a connection
func (s *MockMCPServer) sendMessage(conn *MCPConnection, message MCPMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = conn.writer.WriteString(string(data) + "\n")
	if err != nil {
		return err
	}

	return conn.writer.Flush()
}

// SetupDefaultHandlers sets up default MCP method handlers
func (s *MockMCPServer) SetupDefaultHandlers() {
	// Initialize method
	s.messageHandlers["initialize"] = func(params interface{}) interface{} {
		return map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"logging":   map[string]interface{}{},
				"prompts":   map[string]interface{}{},
				"resources": map[string]interface{}{},
				"tools":     map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "mock-mcp-server",
				"version": "0.1.0",
			},
		}
	}

	// Ping method
	s.messageHandlers["ping"] = func(params interface{}) interface{} {
		return map[string]interface{}{"result": "pong"}
	}

	// List tools method
	s.messageHandlers["tools/list"] = func(params interface{}) interface{} {
		return map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{
					"name":        "file_read",
					"description": "Read a file",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path": map[string]interface{}{"type": "string"},
						},
					},
				},
			},
		}
	}

	// Call tool method
	s.messageHandlers["tools/call"] = func(params interface{}) interface{} {
		return map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Mock tool response",
				},
			},
		}
	}
}

// Configuration methods for testing

// SetResponse sets a predefined response for a method
func (s *MockMCPServer) SetResponse(method string, response interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[method] = response
}

// SetError injects an error for a method
func (s *MockMCPServer) SetError(method string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errorInjections[method] = err
}

// SetLatency sets artificial latency for a method
func (s *MockMCPServer) SetLatency(method string, latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latencyConfig[method] = latency
}

// SetHandler sets a custom handler for a method
func (s *MockMCPServer) SetHandler(method string, handler func(interface{}) interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageHandlers[method] = handler
}

// GetStats returns server statistics
func (s *MockMCPServer) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	connectionDetails := make(map[string]interface{})
	for id, conn := range s.connections {
		connectionDetails[id] = map[string]interface{}{
			"active":        conn.isActive,
			"last_ping":     conn.lastPing,
			"message_count": len(conn.messageLog),
		}
	}

	return map[string]interface{}{
		"is_running":       s.isRunning,
		"server_address":   s.serverAddress,
		"connection_count": len(s.connections),
		"total_messages":   s.messageCount,
		"connections":      connectionDetails,
	}
}

// GetConnectionMessages returns all messages from a connection
func (s *MockMCPServer) GetConnectionMessages(connID string) []MCPMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if conn, exists := s.connections[connID]; exists {
		return conn.messageLog
	}
	return nil
}

// SendNotification sends a notification to all connections
func (s *MockMCPServer) SendNotification(method string, params interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	notification := MCPMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	for _, conn := range s.connections {
		if err := s.sendMessage(conn, notification); err != nil {
			fmt.Printf("Error sending notification to connection %s: %v\n", conn.id, err)
		}
	}

	return nil
}

// StartStdioMode starts the server in stdio mode for process monitoring tests
func (s *MockMCPServer) StartStdioMode() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("server is already running")
	}

	s.isRunning = true

	// Create a connection for stdio
	mcpConn := &MCPConnection{
		reader:     bufio.NewReader(os.Stdin),
		writer:     bufio.NewWriter(os.Stdout),
		id:         "stdio",
		isActive:   true,
		lastPing:   time.Now(),
		messageLog: make([]MCPMessage, 0),
	}

	s.connections["stdio"] = mcpConn

	go s.handleStdioMessages(mcpConn)

	return nil
}

// handleStdioMessages handles messages in stdio mode
func (s *MockMCPServer) handleStdioMessages(conn *MCPConnection) {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			line, err := conn.reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
				}
				return
			}

			if len(line) == 0 {
				continue
			}

			var message MCPMessage
			if err := json.Unmarshal([]byte(line), &message); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing message: %v\n", err)
				continue
			}

			conn.messageLog = append(conn.messageLog, message)
			s.messageCount++

			response := s.processMessage(message)
			if response != nil {
				data, err := json.Marshal(response)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
					continue
				}

				fmt.Println(string(data))
			}
		}
	}
}
