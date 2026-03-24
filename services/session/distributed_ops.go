// Package session - distributed session operations
package session

import (
	"context"
	"fmt"
	"time"
)

// GetSession retrieves a session
func (mgr *DistributedManager) GetSession(ctx context.Context, sessionID string) (*DistributedState, error) {
	mgr.globalMu.RLock()
	session, exists := mgr.sessions[sessionID]
	sessionMu, hasMu := mgr.sessionMutexes[sessionID]
	mgr.globalMu.RUnlock()

	if !exists {
		// Try to load from store
		if mgr.store != nil {
			return mgr.store.Load(ctx, sessionID)
		}
		return nil, fmt.Errorf("session not found")
	}

	if !hasMu {
		return nil, fmt.Errorf("session mutex not found")
	}

	sessionMu.RLock()
	defer sessionMu.RUnlock()

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		mgr.deleteSessionLocal(sessionID)
		return nil, fmt.Errorf("session expired")
	}

	// Update last activity
	session.LastActivity = time.Now()

	return session, nil
}

// UpdateSessionState updates session state atomically
func (mgr *DistributedManager) UpdateSessionState(ctx context.Context, sessionID string, updates map[string]interface{}) error {
	mgr.globalMu.RLock()
	session, exists := mgr.sessions[sessionID]
	sessionMu, hasMu := mgr.sessionMutexes[sessionID]
	mgr.globalMu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found")
	}

	if !hasMu {
		return fmt.Errorf("session mutex not found")
	}

	sessionMu.Lock()
	defer sessionMu.Unlock()

	// Apply updates
	for k, v := range updates {
		session.State[k] = v
	}

	session.Version++
	session.LastActivity = time.Now()

	// Persist changes
	if mgr.store != nil {
		if err := mgr.store.Save(ctx, session); err != nil {
			mgr.stats.mu.Lock()
			mgr.stats.SyncErrors++
			mgr.stats.mu.Unlock()
			return err
		}
	}

	// Replicate to peers
	go mgr.replicateSession(context.Background(), session)

	return nil
}

// MigrateSession migrates a session to another host (for failover)
func (mgr *DistributedManager) MigrateSession(ctx context.Context, sessionID string, newHostID string) error {
	mgr.globalMu.RLock()
	session, exists := mgr.sessions[sessionID]
	sessionMu, hasMu := mgr.sessionMutexes[sessionID]
	mgr.globalMu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found")
	}

	if !hasMu {
		return fmt.Errorf("session mutex not found")
	}

	sessionMu.Lock()
	oldHostID := session.HostID
	session.HostID = newHostID
	session.Version++
	sessionMu.Unlock()

	// Persist migration
	if mgr.store != nil {
		if err := mgr.store.Save(ctx, session); err != nil {
			// Revert
			sessionMu.Lock()
			session.HostID = oldHostID
			session.Version--
			sessionMu.Unlock()
			return err
		}
	}

	return nil
}

// DeleteSession deletes a session
func (mgr *DistributedManager) DeleteSession(ctx context.Context, sessionID string) error {
	mgr.deleteSessionLocal(sessionID)

	// Delete from store
	if mgr.store != nil {
		return mgr.store.Delete(ctx, sessionID)
	}

	return nil
}

// deleteSessionLocal removes session from local cache
func (mgr *DistributedManager) deleteSessionLocal(sessionID string) {
	mgr.globalMu.Lock()
	delete(mgr.sessions, sessionID)
	delete(mgr.sessionMutexes, sessionID)
	mgr.globalMu.Unlock()

	mgr.stats.mu.Lock()
	mgr.stats.ActiveSessions--
	mgr.stats.ExpiredSessions++
	mgr.stats.mu.Unlock()
}

// RegisterPeer registers another instance for replication
func (mgr *DistributedManager) RegisterPeer(hostID string, address string) {
	mgr.replicationMu.Lock()
	mgr.peers[hostID] = address
	mgr.replicationMu.Unlock()
}

// Stats returns manager statistics
func (mgr *DistributedManager) Stats() ManagerStats {
	mgr.stats.mu.RLock()
	defer mgr.stats.mu.RUnlock()
	return *mgr.stats
}

// Close closes the manager
func (mgr *DistributedManager) Close() error {
	mgr.globalMu.Lock()
	if mgr.closed {
		mgr.globalMu.Unlock()
		return nil
	}
	mgr.closed = true
	mgr.globalMu.Unlock()

	mgr.cancel()

	// Clean up all sessions
	mgr.globalMu.Lock()
	mgr.sessions = make(map[string]*DistributedState)
	mgr.sessionMutexes = make(map[string]*sync.RWMutex)
	mgr.globalMu.Unlock()

	return nil
}
