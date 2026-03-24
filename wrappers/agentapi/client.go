// Package agentapi provides a wrapper around the coder/agentapi library
// for TUI capture and CLI agent interaction. This wrapper extends the
// base functionality with Bifrost-specific features like metrics collection,
// NATS integration, and multi-tenant support.
package agentapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/agentapi/lib/httpapi"
	mf "github.com/coder/agentapi/lib/msgfmt"
	"github.com/coder/agentapi/lib/termexec"
)

// AgentType re-exports the agent type from msgfmt
type AgentType = mf.AgentType

// Agent type constants
const (
	AgentTypeClaude = mf.AgentTypeClaude
	AgentTypeGoose  = mf.AgentTypeGoose
	AgentTypeAider  = mf.AgentTypeAider
)

// Client wraps the agentapi server with Bifrost extensions
type Client struct {
	server   *httpapi.Server
	process  *termexec.Process
	config   Config
	logger   *slog.Logger
	mu       sync.RWMutex
	metrics  *Metrics
	handlers []EventHandler
}

// Config holds configuration for the agentapi client
type Config struct {
	AgentType      AgentType
	Command        string
	Args           []string
	Port           int
	ChatBasePath   string
	AllowedHosts   []string
	AllowedOrigins []string
	InitialPrompt  string
	WorkDir        string
	Env            []string
	// Bifrost extensions
	TenantID      string
	EnableMetrics bool
	MetricsPrefix string
}

// Metrics tracks agent interaction metrics
type Metrics struct {
	mu               sync.RWMutex
	MessagesSent     int64
	MessagesReceived int64
	TotalLatencyMs   int64
	ErrorCount       int64
	LastActivity     time.Time
}

// EventHandler is called when agent events occur
type EventHandler func(event Event)

// Event represents an agent event
type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

// EventType identifies the type of event
type EventType string

const (
	EventTypeMessage EventType = "message"
	EventTypeStatus  EventType = "status"
	EventTypeError   EventType = "error"
)

// NewClient creates a new agentapi client wrapper
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	logger := slog.Default().With("component", "agentapi", "tenant", cfg.TenantID)

	// Create the terminal process using the proper API
	process, err := termexec.StartProcess(ctx, termexec.StartProcessConfig{
		Program:        cfg.Command,
		Args:           cfg.Args,
		TerminalWidth:  80,
		TerminalHeight: 24,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create process: %w", err)
	}

	// Create the HTTP server
	serverCfg := httpapi.ServerConfig{
		AgentType:      cfg.AgentType,
		Process:        process,
		Port:           cfg.Port,
		ChatBasePath:   cfg.ChatBasePath,
		AllowedHosts:   cfg.AllowedHosts,
		AllowedOrigins: cfg.AllowedOrigins,
		InitialPrompt:  cfg.InitialPrompt,
	}

	server, err := httpapi.NewServer(ctx, serverCfg)
	if err != nil {
		process.Close(logger, 5*time.Second)
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	client := &Client{
		server:  server,
		process: process,
		config:  cfg,
		logger:  logger,
		metrics: &Metrics{LastActivity: time.Now()},
	}

	return client, nil
}

// Start starts the agent server
func (c *Client) Start(ctx context.Context) error {
	c.server.StartSnapshotLoop(ctx)
	return c.server.Start()
}

// Stop gracefully stops the agent
func (c *Client) Stop(ctx context.Context) error {
	if err := c.server.Stop(ctx); err != nil {
		return err
	}
	return c.process.Close()
}

// Handler returns the HTTP handler for embedding in other servers
func (c *Client) Handler() http.Handler {
	return c.server.Handler()
}

// GetMetrics returns current metrics
func (c *Client) GetMetrics() Metrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()
	return *c.metrics
}

// OnEvent registers an event handler
func (c *Client) OnEvent(handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = append(c.handlers, handler)
}
