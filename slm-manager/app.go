package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name      string `json:"name"`
	Running   bool   `json:"running"`
	Port      int    `json:"port"`
	Error     string `json:"error,omitempty"`
	StartedAt string `json:"startedAt,omitempty"`
}

// AppMode defines whether the app runs as host or client
type AppMode string

const (
	ModeHost   AppMode = "host"
	ModeClient AppMode = "client"
)

// BackendType defines the inference backend
type BackendType string

const (
	BackendMLX  BackendType = "mlx"
	BackendVLLM BackendType = "vllm"
)

// RemoteProfile represents a saved remote server configuration
type RemoteProfile struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Config holds app configuration
type Config struct {
	Mode            AppMode         `json:"mode"`
	Backend         BackendType     `json:"backend"`
	BackendHost     string          `json:"backendHost"`
	BackendPort     int             `json:"backendPort"`
	SLMPort         int             `json:"slmPort"`
	Model           string          `json:"model"`
	RemoteHost      string          `json:"remoteHost"`
	RemotePort      int             `json:"remotePort"`
	RemoteProfiles  []RemoteProfile `json:"remoteProfiles"`
	ActiveProfileID int             `json:"activeProfileId"`
	AutoStartAll    bool            `json:"autoStartAll"`
	LaunchAtLogin   bool            `json:"launchAtLogin"`
	RepoOwner       string          `json:"repoOwner"`
	RepoName        string          `json:"repoName"`
}

// App struct
type App struct {
	fyneApp    fyne.App
	mainWindow fyne.Window

	// Service management
	slmServerCmd *exec.Cmd
	vllmCmd      *exec.Cmd
	mu           sync.Mutex

	// Status
	slmRunning  bool
	vllmRunning bool
	slmStarted  time.Time
	vllmStarted time.Time

	// Configuration
	config Config

	// API Client
	apiClient *APIClient

	// Provider management
	providerMgr *ProviderManager

	// Auth management
	authMgr *AuthManager

	// UI bindings
	logs           binding.StringList
	backendStatus  binding.String
	slmStatus      binding.String
	backendRunning binding.Bool
	slmRunningBind binding.Bool
	isLoading      binding.Bool
}

// NewApp creates a new App instance
func NewApp() *App {
	backend := BackendVLLM
	model := "Qwen/Qwen2.5-3B-Instruct"
	if runtime.GOOS == "darwin" {
		backend = BackendMLX
		model = "mlx-community/Qwen2.5-3B-Instruct-4bit"
	}

	gatewayURL := "http://localhost:8081"
	apiClient := NewAPIClient(gatewayURL)
	authMgr := NewAuthManager()
	providerMgr := NewProviderManager(gatewayURL)

	return &App{
		config: Config{
			Mode:           ModeHost,
			Backend:        backend,
			BackendHost:    "localhost",
			BackendPort:    8000,
			SLMPort:        8081,
			Model:          model,
			RemoteHost:     "",
			RemotePort:     8081,
			RemoteProfiles: []RemoteProfile{},
			AutoStartAll:   true,
			RepoOwner:      "kooshapari",
			RepoName:       "bifrost-extensions",
		},
		apiClient:      apiClient,
		providerMgr:    providerMgr,
		authMgr:        authMgr,
		logs:           binding.NewStringList(),
		backendStatus:  binding.NewString(),
		slmStatus:      binding.NewString(),
		backendRunning: binding.NewBool(),
		slmRunningBind: binding.NewBool(),
		isLoading:      binding.NewBool(),
	}
}

// Run starts the Fyne application
func (a *App) Run() {
	a.fyneApp = app.NewWithID("io.automaze.slm-manager")
	a.fyneApp.SetIcon(theme.ComputerIcon())

	// Load config
	if err := a.LoadConfig(); err != nil {
		log.Printf("Warning: Could not load config: %v", err)
	}

	// Initialize status
	a.backendStatus.Set("Stopped")
	a.slmStatus.Set("Stopped")

	// Initialize auth manager
	a.authMgr.CheckAuthStatus()
	if err := a.authMgr.StartMonitoring(); err != nil {
		log.Printf("Warning: Could not start auth monitoring: %v", err)
	}

	// Start periodic service discovery
	a.providerMgr.StartPeriodicRefresh()

	// Initial service discovery
	go a.providerMgr.DiscoverServices(true)

	// Create main window
	a.mainWindow = a.fyneApp.NewWindow("SLM Manager")
	a.mainWindow.SetContent(a.createMainUI())
	a.mainWindow.Resize(fyne.NewSize(700, 500))

	// Setup system tray
	if desk, ok := a.fyneApp.(desktop.App); ok {
		a.setupSystemTray(desk)
	}

	// Hide window on close (keep in tray)
	a.mainWindow.SetCloseIntercept(func() {
		a.mainWindow.Hide()
	})

	// Auto-start if configured
	if a.config.AutoStartAll {
		go func() {
			time.Sleep(2 * time.Second)
			a.StartAll()
		}()
	}

	// Start periodic health check
	go a.startHealthCheckPolling()

	a.addLog(fmt.Sprintf("SLM Manager v%s started", Version))
	a.mainWindow.ShowAndRun()
}

// startHealthCheckPolling periodically checks service health
func (a *App) startHealthCheckPolling() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		health := a.HealthCheck()

		// Update status if external changes detected
		a.mu.Lock()
		if health["backend"] != a.vllmRunning && a.vllmCmd == nil {
			a.vllmRunning = health["backend"]
		}
		if health["slm"] != a.slmRunning && a.slmServerCmd == nil {
			a.slmRunning = health["slm"]
		}
		a.mu.Unlock()

		a.updateStatus()
	}
}

// AppStatus represents overall app status
type AppStatus struct {
	Version   string          `json:"version"`
	Commit    string          `json:"commit"`
	BuildDate string          `json:"buildDate"`
	Services  []ServiceStatus `json:"services"`
}

// GetStatus returns the current app status
func (a *App) GetStatus() AppStatus {
	a.mu.Lock()
	defer a.mu.Unlock()

	var services []ServiceStatus

	if a.config.Mode == ModeHost {
		backendName := a.getBackendDisplayName()
		services = []ServiceStatus{
			{Name: backendName, Running: a.vllmRunning, Port: a.config.BackendPort, StartedAt: formatTime(a.vllmStarted)},
			{Name: "SLM Server", Running: a.slmRunning, Port: a.config.SLMPort, StartedAt: formatTime(a.slmStarted)},
		}
	} else {
		remoteStatus := a.checkRemoteHealth()
		services = []ServiceStatus{
			{Name: fmt.Sprintf("Remote: %s:%d", a.config.RemoteHost, a.config.RemotePort), Running: remoteStatus, Port: a.config.RemotePort},
		}
	}

	return AppStatus{Version: Version, Commit: Commit, BuildDate: BuildDate, Services: services}
}

// getBackendDisplayName returns human-readable name for current backend
func (a *App) getBackendDisplayName() string {
	switch a.config.Backend {
	case BackendMLX:
		return "MLX Backend"
	case BackendVLLM:
		if runtime.GOOS == "windows" {
			return "vLLM (WSL)"
		}
		return "vLLM"
	default:
		return "Backend"
	}
}

// GetPlatformInfo returns platform-specific information
func (a *App) GetPlatformInfo() map[string]interface{} {
	return map[string]interface{}{
		"os":               runtime.GOOS,
		"arch":             runtime.GOARCH,
		"defaultBackend":   a.getDefaultBackend(),
		"supportsMLX":      runtime.GOOS == "darwin" && runtime.GOARCH == "arm64",
		"supportsVLLM":     runtime.GOOS == "windows" || runtime.GOOS == "linux",
		"supportsVLLMWSL":  runtime.GOOS == "windows",
		"mode":             a.config.Mode,
		"backend":          a.config.Backend,
	}
}

func (a *App) getDefaultBackend() BackendType {
	if runtime.GOOS == "darwin" {
		return BackendMLX
	}
	return BackendVLLM
}

// checkRemoteHealth checks if remote server is reachable
func (a *App) checkRemoteHealth() bool {
	if a.config.RemoteHost == "" {
		return false
	}
	url := fmt.Sprintf("http://%s:%d/health", a.config.RemoteHost, a.config.RemotePort)
	return a.waitForService(url, 2*time.Second)
}

// GetConfig returns the current configuration
func (a *App) GetConfig() Config {
	return a.config
}

// SaveConfig saves the configuration using native dialog for confirmation
func (a *App) SaveConfig(config Config) error {
	a.config = config

	// Persist to config file
	configPath := a.getConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	// Native notification
	a.showNotification("Settings Saved", "Configuration has been saved successfully.")
	return nil
}

// LoadConfig loads configuration from file
func (a *App) LoadConfig() error {
	configPath := a.getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Use defaults
		}
		return err
	}
	return json.Unmarshal(data, &a.config)
}

func (a *App) getConfigPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "SLMManager", "config.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "slm-manager", "config.json")
}

// GetVersionInfo returns version information
func (a *App) GetVersionInfo() map[string]string {
	return map[string]string{
		"version":   Version,
		"commit":    Commit,
		"buildDate": BuildDate,
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
	}
}

// --- Native Dialog Methods ---

// ShowConfirmDialog shows a native confirmation dialog
func (a *App) ShowConfirmDialog(title, message string, callback func(bool)) {
	dialog.ShowConfirm(title, message, callback, a.mainWindow)
}

// ShowErrorDialog shows a native error dialog
func (a *App) ShowErrorDialog(title, message string) {
	dialog.ShowError(fmt.Errorf("%s", message), a.mainWindow)
}

// ShowInfoDialog shows a native info dialog
func (a *App) ShowInfoDialog(title, message string) {
	dialog.ShowInformation(title, message, a.mainWindow)
}

// OpenLogsFolder opens the logs folder in native file explorer
func (a *App) OpenLogsFolder() {
	logsPath := a.getLogsPath()
	os.MkdirAll(logsPath, 0755)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", logsPath)
	case "darwin":
		cmd = exec.Command("open", logsPath)
	default:
		cmd = exec.Command("xdg-open", logsPath)
	}
	cmd.Start()
}

func (a *App) getLogsPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "SLMManager", "logs")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "slm-manager", "logs")
}

// showNotification shows a native system notification
func (a *App) showNotification(title, message string) {
	switch runtime.GOOS {
	case "windows":
		// Use PowerShell for Windows toast notification
		script := `
		[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
		[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
		$template = @"
		<toast>
			<visual>
				<binding template="ToastText02">
					<text id="1">` + title + `</text>
					<text id="2">` + message + `</text>
				</binding>
			</visual>
		</toast>
"@
		$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
		$xml.LoadXml($template)
		$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
		[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("SLM Manager").Show($toast)
		`
		exec.Command("powershell", "-Command", script).Start()

	case "darwin":
		// Use osascript for macOS notification
		script := `display notification "` + message + `" with title "` + title + `"`
		exec.Command("osascript", "-e", script).Start()

	default:
		// Use notify-send for Linux
		exec.Command("notify-send", title, message).Start()
	}
}

// addLog adds a log message
func (a *App) addLog(message string) {
	log.Println(message)
	a.logs.Append(fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message))
}

// updateStatus updates the UI status bindings
func (a *App) updateStatus() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.vllmRunning {
		a.backendStatus.Set("Running")
	} else {
		a.backendStatus.Set("Stopped")
	}
	a.backendRunning.Set(a.vllmRunning)

	if a.slmRunning {
		a.slmStatus.Set("Running")
	} else {
		a.slmStatus.Set("Stopped")
	}
	a.slmRunningBind.Set(a.slmRunning)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// RefreshStatus manually refreshes the status of all services
func (a *App) RefreshStatus() {
	a.addLog("Refreshing service status...")
	health := a.HealthCheck()

	a.mu.Lock()
	if health["backend"] && !a.vllmRunning {
		a.vllmRunning = true
		a.addLog("Backend detected as running")
	} else if !health["backend"] && a.vllmRunning && a.vllmCmd == nil {
		a.vllmRunning = false
		a.addLog("Backend detected as stopped")
	}

	if health["slm"] && !a.slmRunning {
		a.slmRunning = true
		a.addLog("SLM Server detected as running")
	} else if !health["slm"] && a.slmRunning && a.slmServerCmd == nil {
		a.slmRunning = false
		a.addLog("SLM Server detected as stopped")
	}
	a.mu.Unlock()

	a.updateStatus()
	a.addLog("Status refresh complete")
}

// CopyServerURL copies the server URL to clipboard
func (a *App) CopyServerURL() {
	serverURL := a.GetServerURL()
	a.mainWindow.Clipboard().SetContent(serverURL)
	a.showNotification("Copied", "Server URL copied to clipboard")
}

// GetServerURL returns the current server URL
func (a *App) GetServerURL() string {
	return "http://localhost:" + fmt.Sprintf("%d", a.config.SLMPort)
}

// GetLaunchAtLogin returns current launch at login setting
func (a *App) GetLaunchAtLogin() bool {
	return a.config.LaunchAtLogin
}

// SetLaunchAtLogin sets the launch at login preference
func (a *App) SetLaunchAtLogin(enabled bool) error {
	a.config.LaunchAtLogin = enabled

	var err error
	switch runtime.GOOS {
	case "windows":
		err = a.setWindowsStartup(enabled)
	case "darwin":
		err = a.setMacOSStartup(enabled)
	default:
		err = a.setLinuxStartup(enabled)
	}

	if err != nil {
		a.addLog("ERROR: Failed to set startup: " + err.Error())
		return err
	}

	// Save config
	return a.SaveConfig(a.config)
}

// setWindowsStartup adds/removes registry entry for Windows startup
func (a *App) setWindowsStartup(enabled bool) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	keyPath := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	valueName := "SLMManager"

	if enabled {
		// Add registry entry
		cmd := exec.Command("reg", "add", keyPath, "/v", valueName, "/t", "REG_SZ", "/d", exePath, "/f")
		return cmd.Run()
	} else {
		// Remove registry entry
		cmd := exec.Command("reg", "delete", keyPath, "/v", valueName, "/f")
		cmd.Run() // Ignore error if key doesn't exist
		return nil
	}
}

// setMacOSStartup adds/removes launchd plist for macOS startup
func (a *App) setMacOSStartup(enabled bool) error {
	home, _ := os.UserHomeDir()
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "io.automaze.slm-manager.plist")

	if enabled {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}

		plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>io.automaze.slm-manager</string>
    <key>ProgramArguments</key>
    <array>
        <string>` + exePath + `</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>
`
		os.MkdirAll(filepath.Dir(plistPath), 0755)
		return os.WriteFile(plistPath, []byte(plist), 0644)
	} else {
		os.Remove(plistPath)
		return nil
	}
}

// setLinuxStartup adds/removes .desktop file for Linux autostart
func (a *App) setLinuxStartup(enabled bool) error {
	home, _ := os.UserHomeDir()
	autostartPath := filepath.Join(home, ".config", "autostart", "slm-manager.desktop")

	if enabled {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}

		desktop := `[Desktop Entry]
Type=Application
Name=SLM Manager
Exec=` + exePath + `
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
`
		os.MkdirAll(filepath.Dir(autostartPath), 0755)
		return os.WriteFile(autostartPath, []byte(desktop), 0644)
	} else {
		os.Remove(autostartPath)
		return nil
	}
}

// OpenConfigFolder opens the config folder in native file explorer
func (a *App) OpenConfigFolder() {
	configPath := filepath.Dir(a.getConfigPath())
	os.MkdirAll(configPath, 0755)
	a.openFolder(configPath)
}

// openFolder opens a folder in the native file explorer
func (a *App) openFolder(path string) {
	os.MkdirAll(path, 0755)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	cmd.Start()
}

// OpenURL opens a URL in the default browser
func (a *App) OpenURL(urlStr string) {
	parsedURL, _ := url.Parse(urlStr)
	a.fyneApp.OpenURL(parsedURL)
}

// RestartServices restarts all running services
func (a *App) RestartServices() {
	a.ShowConfirmDialog("Restart Services", "This will restart all running services. Continue?", func(confirmed bool) {
		if confirmed {
			a.StopAll()
			time.Sleep(time.Second)
			a.StartAll()
			a.showNotification("Services Restarted", "All services have been restarted")
		}
	})
}

// Shutdown gracefully stops all services
func (a *App) Shutdown() {
	a.addLog("Shutting down...")
	a.StopAll()
}
