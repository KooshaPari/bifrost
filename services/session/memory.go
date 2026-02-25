// Package session - hierarchical memory support
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// MemoryScope represents the scope of memory storage
type MemoryScope string

const (
	MemoryScopeGlobal  MemoryScope = "global"
	MemoryScopeSession MemoryScope = "session"
	MemoryScopeLocal   MemoryScope = "local"
)

// HierarchicalMemory provides NATS-backed hierarchical memory
type HierarchicalMemory struct {
	tracker  *Tracker
	globalKV jetstream.KeyValue
}

// HierarchicalMemoryConfig configures the hierarchical memory
type HierarchicalMemoryConfig struct {
	SessionTracker *Tracker
}

// NewHierarchicalMemory creates a hierarchical memory system
func NewHierarchicalMemory(cfg HierarchicalMemoryConfig) (*HierarchicalMemory, error) {
	hm := &HierarchicalMemory{
		tracker: cfg.SessionTracker,
	}

	// Create global KV bucket if tracker has NATS
	if cfg.SessionTracker != nil && cfg.SessionTracker.js != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		kv, err := cfg.SessionTracker.js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
			Bucket:      "bifrost-global-memory",
			Description: "Global hierarchical memory",
			TTL:         0, // Global memory doesn't expire
			Storage:     jetstream.FileStorage,
			Replicas:    1,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create global KV bucket: %w", err)
		}
		hm.globalKV = kv
	}

	return hm, nil
}

// Set stores a value in hierarchical memory
func (hm *HierarchicalMemory) Set(ctx context.Context, key string, value interface{}, scope MemoryScope, sessionID string) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	switch scope {
	case MemoryScopeGlobal:
		if hm.globalKV != nil {
			_, err := hm.globalKV.Put(ctx, key, data)
			return err
		}
	case MemoryScopeSession:
		if sessionID == "" {
			return fmt.Errorf("session ID required for session scope")
		}
		// Store in session-specific bucket
		sessionKey := fmt.Sprintf("session:%s:%s", sessionID, key)
		if hm.tracker.kv != nil {
			_, err := hm.tracker.kv.Put(ctx, sessionKey, data)
			return err
		}
	case MemoryScopeLocal:
		// Local scope is ephemeral - store in session's in-memory cache
		session, err := hm.tracker.GetOrCreate(ctx, sessionID)
		if err != nil {
			return err
		}
		// Use Actions map for local storage (temporary workaround)
		session.Actions["local:"+key] = 1
	}

	return nil
}

// Get retrieves a value from hierarchical memory
func (hm *HierarchicalMemory) Get(ctx context.Context, key string, scope MemoryScope, sessionID string) (interface{}, error) {
	switch scope {
	case MemoryScopeGlobal:
		if hm.globalKV != nil {
			entry, err := hm.globalKV.Get(ctx, key)
			if err != nil {
				return nil, err
			}
			var value interface{}
			if err := json.Unmarshal(entry.Value(), &value); err != nil {
				return nil, err
			}
			return value, nil
		}
	case MemoryScopeSession:
		if sessionID == "" {
			return nil, fmt.Errorf("session ID required for session scope")
		}
		sessionKey := fmt.Sprintf("session:%s:%s", sessionID, key)
		if hm.tracker.kv != nil {
			entry, err := hm.tracker.kv.Get(ctx, sessionKey)
			if err != nil {
				return nil, err
			}
			var value interface{}
			if err := json.Unmarshal(entry.Value(), &value); err != nil {
				return nil, err
			}
			return value, nil
		}
	}

	return nil, fmt.Errorf("key not found: %s", key)
}

// Delete removes a value from hierarchical memory
func (hm *HierarchicalMemory) Delete(ctx context.Context, key string, scope MemoryScope, sessionID string) error {
	switch scope {
	case MemoryScopeGlobal:
		if hm.globalKV != nil {
			return hm.globalKV.Delete(ctx, key)
		}
	case MemoryScopeSession:
		sessionKey := fmt.Sprintf("session:%s:%s", sessionID, key)
		if hm.tracker.kv != nil {
			return hm.tracker.kv.Delete(ctx, sessionKey)
		}
	}
	return nil
}

