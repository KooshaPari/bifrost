package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// StartAll starts all services based on current mode
func (a *App) StartAll() {
	a.addLog("Starting all services...")

	if a.config.Mode == ModeClient {
		a.addLog("Client mode - connecting to remote server...")
		if a.checkRemoteHealth() {
			a.addLog(fmt.Sprintf("Connected to remote server at %s:%d", a.config.RemoteHost, a.config.RemotePort))
		} else {
			a.addLog("ERROR: Cannot reach remote server")
		}
		return
	}

	// Host mode: start local backend + SLM server
	a.StartBackend()

	// Wait for backend to be ready, then start SLM server
	go func() {
		if a.waitForService(fmt.Sprintf("http://localhost:%d/health", a.config.BackendPort), 120*time.Second) {
			a.addLog("Backend is ready, starting SLM server...")
			a.StartSLMServer()
		} else {
			a.addLog("Warning: Backend not responding, starting SLM server anyway...")
			a.StartSLMServer()
		}
	}()
}

// StopAll stops all services
func (a *App) StopAll() {
	a.addLog("Stopping all services...")
	a.StopSLMServer()
	a.StopBackend()
}

// StartBackend starts the appropriate backend for current platform
func (a *App) StartBackend() {
	switch a.config.Backend {
	case BackendMLX:
		a.StartMLX()
	case BackendVLLM:
		if runtime.GOOS == "windows" {
			a.StartVLLMWSL()
		} else {
			a.StartVLLMNative()
		}
	}
}

// StopBackend stops the current backend
func (a *App) StopBackend() {
	switch a.config.Backend {
	case BackendMLX:
		a.StopMLX()
	case BackendVLLM:
		if runtime.GOOS == "windows" {
			a.StopVLLMWSL()
		} else {
			a.StopVLLMNative()
		}
	}
}

// StartMLX starts MLX backend on macOS
func (a *App) StartMLX() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.vllmRunning {
		a.addLog("MLX already running")
		return
	}

	if runtime.GOOS != "darwin" {
		a.addLog("ERROR: MLX backend only available on macOS")
		return
	}

	a.addLog("Starting MLX backend...")

	cmd := exec.Command("python3", "-m", "mlx_lm.server",
		"--model", a.config.Model,
		"--port", fmt.Sprintf("%d", a.config.BackendPort),
		"--host", "0.0.0.0",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		a.addLog(fmt.Sprintf("ERROR: Failed to start MLX: %v", err))
		return
	}

	a.vllmCmd = cmd
	a.vllmRunning = true
	a.vllmStarted = time.Now()
	a.updateStatus()
	a.addLog(fmt.Sprintf("MLX started on port %d (PID: %d)", a.config.BackendPort, cmd.Process.Pid))

	go func() {
		cmd.Wait()
		a.mu.Lock()
		a.vllmRunning = false
		a.vllmCmd = nil
		a.mu.Unlock()
		a.updateStatus()
		a.addLog("MLX process exited")
	}()
}

// StopMLX stops MLX backend
func (a *App) StopMLX() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.vllmRunning || a.vllmCmd == nil {
		return
	}

	a.addLog("Stopping MLX...")

	if a.vllmCmd.Process != nil {
		a.vllmCmd.Process.Kill()
	}

	a.vllmRunning = false
	a.vllmCmd = nil
	a.updateStatus()
	a.addLog("MLX stopped")
}

// StartVLLMWSL starts vLLM in WSL on Windows
func (a *App) StartVLLMWSL() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.vllmRunning {
		a.addLog("vLLM already running")
		return
	}

	if runtime.GOOS != "windows" {
		a.addLog("ERROR: vLLM WSL mode only available on Windows")
		return
	}

	a.addLog("Starting vLLM in WSL...")

	vllmCmd := fmt.Sprintf(
		"source ~/vllm-env/bin/activate && vllm serve %s --port %d --host 0.0.0.0",
		a.config.Model,
		a.config.BackendPort,
	)

	cmd := exec.Command("wsl", "-e", "bash", "-c", vllmCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		a.addLog(fmt.Sprintf("ERROR: Failed to start vLLM: %v", err))
		return
	}

	a.vllmCmd = cmd
	a.vllmRunning = true
	a.vllmStarted = time.Now()
	a.updateStatus()
	a.addLog(fmt.Sprintf("vLLM started (PID: %d)", cmd.Process.Pid))

	go func() {
		cmd.Wait()
		a.mu.Lock()
		a.vllmRunning = false
		a.vllmCmd = nil
		a.mu.Unlock()
		a.updateStatus()
		a.addLog("vLLM process exited")
	}()
}

// StopVLLMWSL stops vLLM in WSL
func (a *App) StopVLLMWSL() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.vllmRunning || a.vllmCmd == nil {
		return
	}

	a.addLog("Stopping vLLM...")
	exec.Command("wsl", "-e", "pkill", "-f", "vllm").Run()

	if a.vllmCmd.Process != nil {
		a.vllmCmd.Process.Kill()
	}

	a.vllmRunning = false
	a.vllmCmd = nil
	a.updateStatus()
	a.addLog("vLLM stopped")
}

// StartVLLMNative starts vLLM natively on Linux
func (a *App) StartVLLMNative() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.vllmRunning {
		a.emitEvent("log", "vLLM already running")
		return
	}

	a.emitEvent("log", "Starting vLLM...")

	cmd := exec.Command("vllm", "serve", a.config.Model,
		"--port", fmt.Sprintf("%d", a.config.BackendPort),
		"--host", "0.0.0.0",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		a.emitEvent("error", fmt.Sprintf("Failed to start vLLM: %v", err))
		return
	}

	a.vllmCmd = cmd
	a.vllmRunning = true
	a.vllmStarted = time.Now()
	a.emitEvent("serviceUpdate", a.GetStatus())
	a.emitEvent("log", fmt.Sprintf("vLLM started (PID: %d)", cmd.Process.Pid))

	go func() {
		cmd.Wait()
		a.mu.Lock()
		a.vllmRunning = false
		a.vllmCmd = nil
		a.mu.Unlock()
		a.emitEvent("serviceUpdate", a.GetStatus())
		a.emitEvent("log", "vLLM process exited")
	}()
}

// StopVLLMNative stops vLLM native process
func (a *App) StopVLLMNative() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.vllmRunning || a.vllmCmd == nil {
		return
	}

	a.emitEvent("log", "Stopping vLLM...")
	if a.vllmCmd.Process != nil {
		a.vllmCmd.Process.Kill()
	}

	a.vllmRunning = false
	a.vllmCmd = nil
	a.emitEvent("serviceUpdate", a.GetStatus())
	a.emitEvent("log", "vLLM stopped")
}

// StartSLMServer starts the embedded SLM server
func (a *App) StartSLMServer() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.slmRunning {
		a.emitEvent("log", "SLM server already running")
		return
	}

	a.emitEvent("log", "Starting SLM server...")

	// Find the slm-server binary (bundled with the app)
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	slmServerPath := filepath.Join(exeDir, "slm-server")
	if runtime.GOOS == "windows" {
		slmServerPath += ".exe"
	}

	// Build command with arguments based on backend type
	backendArg := "vllm"
	if a.config.Backend == BackendMLX {
		backendArg = "mlx"
	}

	cmd := exec.Command(slmServerPath,
		"-port", fmt.Sprintf("%d", a.config.SLMPort),
		"-backend", backendArg,
		"-vllm-url", fmt.Sprintf("http://%s:%d", a.config.BackendHost, a.config.BackendPort),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		a.emitEvent("error", fmt.Sprintf("Failed to start SLM server: %v", err))
		return
	}

	a.slmServerCmd = cmd
	a.slmRunning = true
	a.slmStarted = time.Now()
	a.emitEvent("serviceUpdate", a.GetStatus())
	a.emitEvent("log", fmt.Sprintf("SLM server started on port %d", a.config.SLMPort))

	go func() {
		cmd.Wait()
		a.mu.Lock()
		a.slmRunning = false
		a.slmServerCmd = nil
		a.mu.Unlock()
		a.emitEvent("serviceUpdate", a.GetStatus())
		a.emitEvent("log", "SLM server process exited")
	}()
}

// Legacy methods for backward compatibility with tray menu
func (a *App) StartVLLM() {
	a.StartBackend()
}

func (a *App) StopVLLM() {
	a.StopBackend()
}

