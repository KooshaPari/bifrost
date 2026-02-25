// Package session - distributed session management
// Ported from CLIProxyAPI internal/agentapi/agentapi-distributed-sessions.go
package session

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DistributedState represents the state of a distributed session
type DistributedState struct {
	ID           string
	AgentType    string
	CreatedAt    time.Time
	LastActivity time.Time
	ExpiresAt    time.Time
	State        map[string]interface{}
	HostID       string   // Which service instance hosts this session
	ReplicaHosts []string // Backup hosts for failover
	Version      int64    // For optimistic locking
	Checksum     string   // For integrity checking
}

// DistributedManager manages sessions across multiple instances
type DistributedManager struct {
	sessions       map[string]*DistributedState
	sessionMutexes map[string]*sync.RWMutex
	globalMu       sync.RWMutex

	// Replication
	peers         map[string]string // Host ID -> Address
	replicationMu sync.RWMutex

	// Persistence
	store Store

	// Configuration
	sessionTTL     time.Duration
	replicationLag time.Duration
	syncInterval   time.Duration

	// Stats
	stats  *ManagerStats
	ctx    context.Context
	cancel context.CancelFunc
	closed bool
}

// Store interface for persisting sessions
type Store interface {
	Save(ctx context.Context, session *DistributedState) error
	Load(ctx context.Context, sessionID string) (*DistributedState, error)
	Delete(ctx context.Context, sessionID string) error
	ListExpired(ctx context.Context, before time.Time) ([]string, error)
}

// ManagerStats tracks session manager statistics
type ManagerStats struct {
	TotalSessions      int32
	ActiveSessions     int32
	ExpiredSessions    int32
	ReplicatedSessions int32
	SyncErrors         int64
	LastSyncTime       time.Time
	mu                 sync.RWMutex
}

// DistributedManagerConfig holds configuration
type DistributedManagerConfig struct {
	SessionTTL     time.Duration
	ReplicationLag time.Duration
	SyncInterval   time.Duration
	Store          Store
}

// NewDistributedManager creates a new distributed session manager
func NewDistributedManager(config DistributedManagerConfig) *DistributedManager {
	if config.SessionTTL == 0 {
		config.SessionTTL = 24 * time.Hour
	}
	if config.SyncInterval == 0 {
		config.SyncInterval = 1 * time.Minute
	}
	if config.ReplicationLag == 0 {
		config.ReplicationLag = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	mgr := &DistributedManager{
		sessions:       make(map[string]*DistributedState),
		sessionMutexes: make(map[string]*sync.RWMutex),
		peers:          make(map[string]string),
		sessionTTL:     config.SessionTTL,
		replicationLag: config.ReplicationLag,
		syncInterval:   config.SyncInterval,
		store:          config.Store,
		stats:          &ManagerStats{},
		ctx:            ctx,
		cancel:         cancel,
	}

	// Start background routines
	go mgr.expiredSessionCleanupRoutine()
	go mgr.replicationSyncRoutine()

	return mgr
}

// CreateSession creates a new distributed session
func (mgr *DistributedManager) CreateSession(ctx context.Context, agentType string, hostID string) (*DistributedState, error) {
	mgr.globalMu.Lock()
	if mgr.closed {
		mgr.globalMu.Unlock()
		return nil, fmt.Errorf("session manager is closed")
	}

	sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())
	session := &DistributedState{
		ID:           sessionID,
		AgentType:    agentType,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    time.Now().Add(mgr.sessionTTL),
		State:        make(map[string]interface{}),
		HostID:       hostID,
		ReplicaHosts: make([]string, 0),
		Version:      1,
	}

	mgr.sessions[sessionID] = session
	mgr.sessionMutexes[sessionID] = &sync.RWMutex{}
	mgr.globalMu.Unlock()

	// Persist to store
	if mgr.store != nil {
		if err := mgr.store.Save(ctx, session); err != nil {
			mgr.deleteSessionLocal(sessionID)
			return nil, err
		}
	}

	// Replicate to peers
	go mgr.replicateSession(context.Background(), session)

	mgr.stats.mu.Lock()
	mgr.stats.TotalSessions++
	mgr.stats.ActiveSessions++
	mgr.stats.mu.Unlock()

	return session, nil
}

