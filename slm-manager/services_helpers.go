package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

// StopSLMServer stops the SLM server
func (a *App) StopSLMServer() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.slmRunning || a.slmServerCmd == nil {
		return
	}

	a.addLog("Stopping SLM server...")

	if a.slmServerCmd.Process != nil {
		if runtime.GOOS == "windows" {
			exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", a.slmServerCmd.Process.Pid)).Run()
		} else {
			a.slmServerCmd.Process.Kill()
		}
	}

	a.slmRunning = false
	a.slmServerCmd = nil
	a.updateStatus()
	a.addLog("SLM server stopped")
}

// waitForService waits for a service to become available
func (a *App) waitForService(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(2 * time.Second)
	}
	return false
}



// HealthCheck checks if services are responding
func (a *App) HealthCheck() map[string]bool {
	results := make(map[string]bool)
	client := &http.Client{Timeout: 2 * time.Second}

	// Check backend (vLLM or MLX)
	backendURL := fmt.Sprintf("http://localhost:%d/health", a.config.BackendPort)
	if resp, err := client.Get(backendURL); err == nil {
		resp.Body.Close()
		results["backend"] = resp.StatusCode == http.StatusOK
	} else {
		results["backend"] = false
	}

	// Check SLM Server
	slmURL := fmt.Sprintf("http://localhost:%d/health", a.config.SLMPort)
	if resp, err := client.Get(slmURL); err == nil {
		resp.Body.Close()
		results["slm"] = resp.StatusCode == http.StatusOK
	} else {
		results["slm"] = false
	}

	return results
}

// GetLogs returns recent logs (placeholder for now)
func (a *App) GetLogs() []string {
	// TODO: Implement log buffer
	return []string{}
}

// RestartSLMServer restarts the SLM server
func (a *App) RestartSLMServer() {
	a.StopSLMServer()
	time.Sleep(500 * time.Millisecond)
	a.StartSLMServer()
}

// RestartVLLM restarts vLLM
func (a *App) RestartVLLM() {
	a.StopVLLM()
	time.Sleep(500 * time.Millisecond)
	a.StartVLLM()
}

// RestartAll restarts all services
func (a *App) RestartAll() {
	a.StopAll()
	time.Sleep(1 * time.Second)
	a.StartAll()
}

