// Package session - replication routines
package session

import (
	"context"
	"time"
)

// replicateSession replicates a session to peer instances
func (mgr *DistributedManager) replicateSession(ctx context.Context, session *DistributedState) {
	mgr.replicationMu.RLock()
	peers := make(map[string]string)
	for k, v := range mgr.peers {
		if k != session.HostID { // Don't replicate to self
			peers[k] = v
		}
	}
	mgr.replicationMu.RUnlock()

	// Replicate to each peer
	for hostID := range peers {
		go func(hostID string) {
			// Simulate replication delay
			time.Sleep(mgr.replicationLag)

			// TODO: Send replication message to peer via NATS
			// For now, just track that we attempted replication
			mgr.globalMu.Lock()
			session.ReplicaHosts = append(session.ReplicaHosts, hostID)
			mgr.globalMu.Unlock()

			mgr.stats.mu.Lock()
			mgr.stats.ReplicatedSessions++
			mgr.stats.mu.Unlock()
		}(hostID)
	}
}

// replicationSyncRoutine periodically syncs with peers
func (mgr *DistributedManager) replicationSyncRoutine() {
	ticker := time.NewTicker(mgr.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mgr.ctx.Done():
			return
		case <-ticker.C:
			mgr.performReplicationSync()
		}
	}
}

// performReplicationSync syncs session state with peers
func (mgr *DistributedManager) performReplicationSync() {
	mgr.globalMu.RLock()
	sessions := make([]*DistributedState, 0, len(mgr.sessions))
	for _, session := range mgr.sessions {
		sessions = append(sessions, session)
	}
	mgr.globalMu.RUnlock()

	// For each session, ensure it's replicated to configured replicas
	for _, session := range sessions {
		mgr.replicateSession(context.Background(), session)
	}

	mgr.stats.mu.Lock()
	mgr.stats.LastSyncTime = time.Now()
	mgr.stats.mu.Unlock()
}

// expiredSessionCleanupRoutine periodically removes expired sessions
func (mgr *DistributedManager) expiredSessionCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mgr.ctx.Done():
			return
		case <-ticker.C:
			mgr.cleanupExpiredSessions()
		}
	}
}

// cleanupExpiredSessions removes expired sessions
func (mgr *DistributedManager) cleanupExpiredSessions() {
	now := time.Now()
	var toDelete []string

	mgr.globalMu.RLock()
	for sessionID, session := range mgr.sessions {
		if now.After(session.ExpiresAt) {
			toDelete = append(toDelete, sessionID)
		}
	}
	mgr.globalMu.RUnlock()

	for _, sessionID := range toDelete {
		mgr.deleteSessionLocal(sessionID)

		// Delete from store
		if mgr.store != nil {
			mgr.store.Delete(context.Background(), sessionID)
		}
	}
}

