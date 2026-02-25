package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// AuthStatus represents the authentication status for a service
type AuthStatus struct {
	IsAuthenticated bool       `json:"isAuthenticated"`
	Email           string     `json:"email,omitempty"`
	Type            string     `json:"type"`
	Expired         *time.Time `json:"expired,omitempty"`
}

// IsExpired checks if the auth has expired
func (a *AuthStatus) IsExpired() bool {
	if a.Expired == nil {
		return false
	}
	return a.Expired.Before(time.Now())
}

// StatusText returns a human-readable status
func (a *AuthStatus) StatusText() string {
	if !a.IsAuthenticated {
		return "Not Connected"
	}
	if a.IsExpired() {
		return "Expired - Reconnect Required"
	}
	if a.Email != "" {
		return "Connected as " + a.Email
	}
	return "Connected"
}

// AuthManager manages authentication status for all services
type AuthManager struct {
	mu           sync.RWMutex
	authStatuses map[string]*AuthStatus
	authDir      string
	watcher      *fsnotify.Watcher
	onChange     func() // Callback when auth status changes
}

// NewAuthManager creates a new auth manager
func NewAuthManager() *AuthManager {
	authDir := getAuthDir()
	return &AuthManager{
		authStatuses: make(map[string]*AuthStatus),
		authDir:      authDir,
	}
}

// getAuthDir returns the auth directory path
func getAuthDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("USERPROFILE"), ".cli-proxy-api")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cli-proxy-api")
}

// SetOnChange sets the callback for auth status changes
func (am *AuthManager) SetOnChange(callback func()) {
	am.onChange = callback
}

// GetStatus returns the auth status for a service
func (am *AuthManager) GetStatus(serviceID string) *AuthStatus {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.authStatuses[serviceID]
}

// IsAuthenticated checks if a service is authenticated
func (am *AuthManager) IsAuthenticated(serviceID string) bool {
	status := am.GetStatus(serviceID)
	return status != nil && status.IsAuthenticated
}

// GetEmail returns the email for a service
func (am *AuthManager) GetEmail(serviceID string) string {
	status := am.GetStatus(serviceID)
	if status != nil {
		return status.Email
	}
	return ""
}

// IsExpired checks if a service's auth is expired
func (am *AuthManager) IsExpired(serviceID string) bool {
	status := am.GetStatus(serviceID)
	return status != nil && status.IsExpired()
}

// CheckAuthStatus scans the auth directory for auth files
func (am *AuthManager) CheckAuthStatus() {
	am.mu.Lock()
	defer am.mu.Unlock()

	foundServices := make(map[string]*AuthStatus)

	// Ensure directory exists
	if err := os.MkdirAll(am.authDir, 0755); err != nil {
		log.Printf("[AuthManager] Failed to create auth dir: %v", err)
		return
	}

	files, err := os.ReadDir(am.authDir)
	if err != nil {
		log.Printf("[AuthManager] Failed to read auth dir: %v", err)
		return
	}

	log.Printf("[AuthManager] Scanning %d files in auth directory", len(files))

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(am.authDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var authData struct {
			Type    string `json:"type"`
			Email   string `json:"email,omitempty"`
			Expired string `json:"expired,omitempty"`
		}

		if err := json.Unmarshal(data, &authData); err != nil {
			continue
		}

		if authData.Type == "" {
			continue
		}

		var expiredTime *time.Time
		if authData.Expired != "" {
			if t, err := time.Parse(time.RFC3339, authData.Expired); err == nil {
				expiredTime = &t
			}
		}

		status := &AuthStatus{
			IsAuthenticated: true,
			Email:           authData.Email,
			Type:            authData.Type,
			Expired:         expiredTime,
		}

		serviceKey := authData.Type
		foundServices[serviceKey] = status
		log.Printf("[AuthManager] Found auth for service '%s': %s", serviceKey, authData.Email)
	}

	am.authStatuses = foundServices
	log.Printf("[AuthManager] Auth status check complete. Found %d authenticated services", len(foundServices))
}

// StartMonitoring starts watching the auth directory for changes
func (am *AuthManager) StartMonitoring() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	am.watcher = watcher

	// Ensure directory exists
	if err := os.MkdirAll(am.authDir, 0755); err != nil {
		return err
	}

	// Add the auth directory to watch
	if err := watcher.Add(am.authDir); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
					log.Printf("[AuthManager] Auth directory changed: %s", event.Name)
					am.CheckAuthStatus()
					if am.onChange != nil {
						am.onChange()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("[AuthManager] Watcher error: %v", err)
			}
		}
	}()

	log.Printf("[AuthManager] Started monitoring auth directory: %s", am.authDir)
	return nil
}

// StopMonitoring stops watching the auth directory
func (am *AuthManager) StopMonitoring() {
	if am.watcher != nil {
		am.watcher.Close()
		am.watcher = nil
	}
}

// Disconnect removes auth for a service
func (am *AuthManager) Disconnect(serviceType string) error {
	files, err := os.ReadDir(am.authDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(am.authDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var authData struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &authData); err != nil {
			continue
		}

		if authData.Type == serviceType {
			if err := os.Remove(filePath); err != nil {
				return err
			}
			log.Printf("[AuthManager] Deleted auth file for %s: %s", serviceType, filePath)
			am.CheckAuthStatus()
			return nil
		}
	}

	return nil
}

// GetAllStatuses returns all auth statuses
func (am *AuthManager) GetAllStatuses() map[string]*AuthStatus {
	am.mu.RLock()
	defer am.mu.RUnlock()

	result := make(map[string]*AuthStatus)
	for k, v := range am.authStatuses {
		result[k] = v
	}
	return result
}

