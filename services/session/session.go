// Package session provides session tracking for agent interactions.
// Ported from CLIProxyAPI internal/tools/session.go with bifrost-extensions integration.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Context tracks cumulative state for a single agent session
type Context struct {
	SessionID      string    `json:"session_id"`
	StartTime      time.Time `json:"start_time"`
	LastUpdated    time.Time `json:"last_updated"`
	IterationCount int       `json:"iteration_count"`

	// Model routing history
	ModelDecisions []ModelDecision `json:"model_decisions"`

	// Tool usage history
	ToolsUsed        []ToolUsage    `json:"tools_used"`
	ToolsRecommended map[string]int `json:"tools_recommended"` // tool name -> recommend count

	// Accumulated context
	Domains map[string]int `json:"domains"` // domain -> frequency
	Actions map[string]int `json:"actions"` // action -> frequency

	// Performance tracking
	TotalLatencyMs int64 `json:"total_latency_ms"`
	SuccessCount   int   `json:"success_count"`
	FailureCount   int   `json:"failure_count"`
}

// ModelDecision records a model routing decision
type ModelDecision struct {
	Iteration     int       `json:"iteration"`
	Timestamp     time.Time `json:"timestamp"`
	SelectedModel string    `json:"selected_model"`
	Domain        string    `json:"domain"`
	Action        string    `json:"action"`
	Confidence    float64   `json:"confidence"`
	LatencyMs     int64     `json:"latency_ms"`
}

// ToolUsage records a tool invocation
type ToolUsage struct {
	Iteration  int                    `json:"iteration"`
	Timestamp  time.Time              `json:"timestamp"`
	ToolName   string                 `json:"tool_name"`
	Arguments  map[string]interface{} `json:"arguments,omitempty"`
	Success    bool                   `json:"success"`
	LatencyMs  int64                  `json:"latency_ms"`
	ResultSize int                    `json:"result_size"`
}

// Tracker manages session state using NATS JetStream KV
type Tracker struct {
	nc         *nats.Conn
	js         jetstream.JetStream
	kv         jetstream.KeyValue
	bucketName string
	ttl        time.Duration

	// In-memory cache for hot sessions
	cache   map[string]*Context
	cacheMu sync.RWMutex
}

// TrackerConfig configures the session tracker
type TrackerConfig struct {
	NatsURL    string
	BucketName string
	TTL        time.Duration
}

// NewTracker creates a session tracker backed by NATS JetStream KV
func NewTracker(cfg TrackerConfig) (*Tracker, error) {
	if cfg.NatsURL == "" {
		cfg.NatsURL = "nats://localhost:4222"
	}
	if cfg.BucketName == "" {
		cfg.BucketName = "bifrost-sessions"
	}
	if cfg.TTL == 0 {
		cfg.TTL = 24 * time.Hour
	}

	// Connect to NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Get JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create or get KV bucket
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      cfg.BucketName,
		Description: "Bifrost session tracking",
		TTL:         cfg.TTL,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create KV bucket: %w", err)
	}

	return &Tracker{
		nc:         nc,
		js:         js,
		kv:         kv,
		bucketName: cfg.BucketName,
		ttl:        cfg.TTL,
		cache:      make(map[string]*Context),
	}, nil
}

// NewInMemoryTracker creates a session tracker without NATS (for testing)
func NewInMemoryTracker() *Tracker {
	return &Tracker{
		cache: make(map[string]*Context),
		ttl:   24 * time.Hour,
	}
}

// Close closes the NATS connection
func (t *Tracker) Close() error {
	if t.nc != nil {
		t.nc.Close()
	}
	return nil
}

// JetStream returns the JetStream context for hierarchical memory
func (t *Tracker) JetStream() jetstream.JetStream {
	return t.js
}

// KV returns the KeyValue store for direct access
func (t *Tracker) KV() jetstream.KeyValue {
	return t.kv
}
