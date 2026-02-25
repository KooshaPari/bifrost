// Package session - session operations
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetOrCreate retrieves or creates a session context
func (t *Tracker) GetOrCreate(ctx context.Context, sessionID string) (*Context, error) {
	// Check cache first
	t.cacheMu.RLock()
	if session, ok := t.cache[sessionID]; ok {
		t.cacheMu.RUnlock()
		return session, nil
	}
	t.cacheMu.RUnlock()

	// Try to load from NATS KV
	if t.kv != nil {
		entry, err := t.kv.Get(ctx, sessionID)
		if err == nil {
			var session Context
			if err := json.Unmarshal(entry.Value(), &session); err == nil {
				t.cacheMu.Lock()
				t.cache[sessionID] = &session
				t.cacheMu.Unlock()
				return &session, nil
			}
		}
	}

	// Create new session
	session := &Context{
		SessionID:        sessionID,
		StartTime:        time.Now(),
		LastUpdated:      time.Now(),
		IterationCount:   0,
		ModelDecisions:   make([]ModelDecision, 0),
		ToolsUsed:        make([]ToolUsage, 0),
		ToolsRecommended: make(map[string]int),
		Domains:          make(map[string]int),
		Actions:          make(map[string]int),
	}

	t.cacheMu.Lock()
	t.cache[sessionID] = session
	t.cacheMu.Unlock()

	// Persist to NATS KV
	if err := t.persist(ctx, session); err != nil {
		// Log but don't fail - we have the in-memory version
		fmt.Printf("Warning: failed to persist session %s: %v\n", sessionID, err)
	}

	return session, nil
}

// persist saves session to NATS KV
func (t *Tracker) persist(ctx context.Context, session *Context) error {
	if t.kv == nil {
		return nil // In-memory mode
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	_, err = t.kv.Put(ctx, session.SessionID, data)
	return err
}

// RecordModelDecision records a model routing decision for the session
func (t *Tracker) RecordModelDecision(ctx context.Context, sessionID string, decision ModelDecision) error {
	session, err := t.GetOrCreate(ctx, sessionID)
	if err != nil {
		return err
	}

	t.cacheMu.Lock()
	session.IterationCount++
	decision.Iteration = session.IterationCount
	decision.Timestamp = time.Now()
	session.ModelDecisions = append(session.ModelDecisions, decision)
	session.LastUpdated = time.Now()
	session.TotalLatencyMs += decision.LatencyMs

	// Track domain/action frequency
	if decision.Domain != "" {
		session.Domains[decision.Domain]++
	}
	if decision.Action != "" {
		session.Actions[decision.Action]++
	}
	t.cacheMu.Unlock()

	return t.persist(ctx, session)
}

// RecordToolUsage records a tool invocation for the session
func (t *Tracker) RecordToolUsage(ctx context.Context, sessionID string, usage ToolUsage) error {
	session, err := t.GetOrCreate(ctx, sessionID)
	if err != nil {
		return err
	}

	t.cacheMu.Lock()
	usage.Iteration = session.IterationCount
	usage.Timestamp = time.Now()
	session.ToolsUsed = append(session.ToolsUsed, usage)
	session.LastUpdated = time.Now()
	session.TotalLatencyMs += usage.LatencyMs

	if usage.Success {
		session.SuccessCount++
	} else {
		session.FailureCount++
	}
	t.cacheMu.Unlock()

	return t.persist(ctx, session)
}

// RecordToolRecommendations records which tools were recommended
func (t *Tracker) RecordToolRecommendations(ctx context.Context, sessionID string, toolNames []string) error {
	session, err := t.GetOrCreate(ctx, sessionID)
	if err != nil {
		return err
	}

	t.cacheMu.Lock()
	for _, name := range toolNames {
		session.ToolsRecommended[name]++
	}
	session.LastUpdated = time.Now()
	t.cacheMu.Unlock()

	return t.persist(ctx, session)
}

// GetRecentTools returns tools used in recent iterations (for context-aware pruning)
func (t *Tracker) GetRecentTools(ctx context.Context, sessionID string, limit int) ([]string, error) {
	session, err := t.GetOrCreate(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	t.cacheMu.RLock()
	defer t.cacheMu.RUnlock()

	// Get unique tool names from recent usage (most recent first)
	seen := make(map[string]bool)
	var tools []string

	for i := len(session.ToolsUsed) - 1; i >= 0 && len(tools) < limit; i-- {
		name := session.ToolsUsed[i].ToolName
		if !seen[name] {
			tools = append(tools, name)
			seen[name] = true
		}
	}

	return tools, nil
}

// GetDominantDomain returns the most frequent domain in the session
func (t *Tracker) GetDominantDomain(ctx context.Context, sessionID string) (string, error) {
	session, err := t.GetOrCreate(ctx, sessionID)
	if err != nil {
		return "", err
	}

	t.cacheMu.RLock()
	defer t.cacheMu.RUnlock()

	var maxDomain string
	var maxCount int

	for domain, count := range session.Domains {
		if count > maxCount {
			maxDomain = domain
			maxCount = count
		}
	}

	return maxDomain, nil
}

