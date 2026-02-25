package main

import (
	"fmt"
	"image/color"
	"runtime"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Status colors
var (
	colorRunning = color.NRGBA{R: 34, G: 197, B: 94, A: 255}  // Green
	colorStopped = color.NRGBA{R: 239, G: 68, B: 68, A: 255}  // Red
	colorPending = color.NRGBA{R: 250, G: 204, B: 21, A: 255} // Yellow
)

// createMainUI creates the main application UI
func (a *App) createMainUI() fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Dashboard", theme.HomeIcon(), a.createDashboard()),
		container.NewTabItemWithIcon("Providers", theme.StorageIcon(), a.createProvidersView()),
		container.NewTabItemWithIcon("Logs", theme.DocumentIcon(), a.createLogsView()),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), a.createSettings()),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

// createStatusDot creates a colored status indicator dot
func createStatusDot(running binding.Bool) *fyne.Container {
	dot := canvas.NewCircle(colorStopped)
	dot.Resize(fyne.NewSize(12, 12))

	running.AddListener(binding.NewDataListener(func() {
		isRunning, _ := running.Get()
		if isRunning {
			dot.FillColor = colorRunning
		} else {
			dot.FillColor = colorStopped
		}
		dot.Refresh()
	}))

	return container.NewWithoutLayout(dot)
}

// createDashboard creates the dashboard view - mode aware
func (a *App) createDashboard() fyne.CanvasObject {
	// Header with mode and platform info
	modeLabel := widget.NewLabel(fmt.Sprintf("Mode: %s", a.config.Mode))
	modeLabel.TextStyle = fyne.TextStyle{Bold: true}

	platformLabel := widget.NewLabel(fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
	platformLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Mode-specific content
	if a.config.Mode == ModeClient {
		return a.createClientDashboard(modeLabel, platformLabel)
	}
	return a.createHostDashboard(modeLabel, platformLabel)
}

// createHostDashboard creates the dashboard for Host mode
func (a *App) createHostDashboard(modeLabel, platformLabel *widget.Label) fyne.CanvasObject {
	backendLabel := widget.NewLabel(fmt.Sprintf("Backend: %s", a.config.Backend))

	modelLabel := widget.NewLabel("")
	modelLabel.Wrapping = fyne.TextTruncate
	a.updateModelLabel(modelLabel)

	header := container.NewVBox(
		container.NewHBox(modeLabel, widget.NewSeparator(), backendLabel, layout.NewSpacer(), platformLabel),
		container.NewHBox(widget.NewLabel("Model:"), modelLabel),
	)

	// Backend service card
	backendCard := a.createServiceCard(
		a.getBackendDisplayName(),
		a.config.BackendPort,
		a.backendStatus,
		a.backendRunning,
		&a.vllmStarted,
		func() { go a.StartBackend() },
		func() { go a.StopBackend() },
		func() { go a.RestartBackend() },
	)

	// SLM Server card
	slmCard := a.createServiceCard(
		"SLM Server",
		a.config.SLMPort,
		a.slmStatus,
		a.slmRunningBind,
		&a.slmStarted,
		func() { go a.StartSLMServer() },
		func() { go a.StopSLMServer() },
		func() { go a.RestartSLMServer() },
	)

	// Action buttons
	startAllBtn := widget.NewButtonWithIcon("Start All", theme.MediaPlayIcon(), func() {
		go a.StartAll()
	})
	startAllBtn.Importance = widget.HighImportance

	stopAllBtn := widget.NewButtonWithIcon("Stop All", theme.MediaStopIcon(), func() {
		go a.StopAll()
	})

	restartBtn := widget.NewButtonWithIcon("Restart All", theme.ViewRefreshIcon(), func() {
		go a.RestartAll()
	})

	copyURLBtn := widget.NewButtonWithIcon("Copy URL", theme.ContentCopyIcon(), func() {
		a.CopyServerURL()
	})

	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		a.RefreshStatus()
	})

	actions := container.NewHBox(startAllBtn, stopAllBtn, restartBtn, layout.NewSpacer(), copyURLBtn, refreshBtn)

	// Cards grid
	cards := container.NewGridWithColumns(2, backendCard, slmCard)

	// Server URL display
	urlLabel := widget.NewLabel(fmt.Sprintf("Server URL: http://localhost:%d", a.config.SLMPort))
	urlLabel.TextStyle = fyne.TextStyle{Monospace: true}

	return container.NewVBox(
		widget.NewCard("", "", header),
		cards,
		widget.NewSeparator(),
		actions,
		widget.NewSeparator(),
		urlLabel,
	)
}

// createClientDashboard creates the dashboard for Client mode
func (a *App) createClientDashboard(modeLabel, platformLabel *widget.Label) fyne.CanvasObject {
	// Connection info
	var connInfo string
	if a.config.RemoteHost != "" {
		connInfo = fmt.Sprintf("%s:%d", a.config.RemoteHost, a.config.RemotePort)
	} else {
		connInfo = "Not configured"
	}
	connLabel := widget.NewLabel(fmt.Sprintf("Remote: %s", connInfo))

	header := container.NewVBox(
		container.NewHBox(modeLabel, widget.NewSeparator(), connLabel, layout.NewSpacer(), platformLabel),
	)

	// Connection status card
	statusDot := canvas.NewCircle(colorStopped)
	statusDot.Resize(fyne.NewSize(12, 12))

	statusLabel := widget.NewLabel("Disconnected")
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	testBtn := widget.NewButtonWithIcon("Test Connection", theme.ConfirmIcon(), func() {
		if a.checkRemoteHealth() {
			statusDot.FillColor = colorRunning
			statusLabel.SetText("Connected")
			a.showNotification("Connected", "Remote server is reachable")
		} else {
			statusDot.FillColor = colorStopped
			statusLabel.SetText("Disconnected")
			a.ShowErrorDialog("Failed", "Cannot reach remote server")
		}
		statusDot.Refresh()
	})

	copyURLBtn := widget.NewButtonWithIcon("Copy URL", theme.ContentCopyIcon(), func() {
		url := fmt.Sprintf("http://%s:%d", a.config.RemoteHost, a.config.RemotePort)
		a.mainWindow.Clipboard().SetContent(url)
		a.showNotification("Copied", "URL copied to clipboard")
	})

	connCard := widget.NewCard("Remote Connection", connInfo, container.NewVBox(
		container.NewHBox(container.NewWithoutLayout(statusDot), statusLabel),
		container.NewHBox(testBtn, copyURLBtn),
	))

	// Profile switcher if profiles exist
	var profileSection fyne.CanvasObject
	if len(a.config.RemoteProfiles) > 0 {
		profileSelect := widget.NewSelect([]string{}, func(name string) {
			for _, p := range a.config.RemoteProfiles {
				if p.Name == name {
					a.config.RemoteHost = p.Host
					a.config.RemotePort = p.Port
					connLabel.SetText(fmt.Sprintf("Remote: %s:%d", p.Host, p.Port))
					break
				}
			}
		})
		for _, p := range a.config.RemoteProfiles {
			profileSelect.Options = append(profileSelect.Options, p.Name)
		}
		if len(a.config.RemoteProfiles) > a.config.ActiveProfileID {
			profileSelect.SetSelected(a.config.RemoteProfiles[a.config.ActiveProfileID].Name)
		}
		profileSection = widget.NewCard("Quick Switch", "", profileSelect)
	} else {
		profileSection = widget.NewLabel("No saved profiles. Add profiles in Settings.")
	}

	settingsBtn := widget.NewButtonWithIcon("Configure Remote", theme.SettingsIcon(), func() {
		// Switch to settings tab
	})
	settingsBtn.Importance = widget.HighImportance

	return container.NewVBox(
		widget.NewCard("", "", header),
		connCard,
		profileSection,
		widget.NewSeparator(),
		container.NewHBox(settingsBtn),
	)
}

func (a *App) updateModelLabel(label *widget.Label) {
	model := a.config.Model
	if len(model) > 50 {
		model = "..." + model[len(model)-47:]
	}
	label.SetText(model)
}

// createServiceCard creates a card widget for a service with status dot, uptime, and controls
func (a *App) createServiceCard(name string, port int, statusBinding binding.String, runningBinding binding.Bool, startTime *time.Time, onStart, onStop, onRestart func()) *widget.Card {
	// Status dot
	statusDot := createStatusDot(runningBinding)

	// Status label
	statusLabel := widget.NewLabelWithData(statusBinding)
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Port label
	portLabel := widget.NewLabel(fmt.Sprintf("Port: %d", port))
	portLabel.TextStyle = fyne.TextStyle{Monospace: true}

	// Uptime label (updates periodically)
	uptimeLabel := widget.NewLabel("Uptime: --")
	go a.updateUptime(uptimeLabel, runningBinding, startTime)

	// Loading indicator for operations
	loadingBar := widget.NewProgressBarInfinite()
	loadingBar.Hide()

	// Control buttons with loading state
	startBtn := widget.NewButtonWithIcon("Start", theme.MediaPlayIcon(), func() {
		loadingBar.Show()
		go func() {
			onStart()
			loadingBar.Hide()
		}()
	})
	startBtn.Importance = widget.HighImportance

	stopBtn := widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() {
		loadingBar.Show()
		go func() {
			onStop()
			loadingBar.Hide()
		}()
	})
	stopBtn.Importance = widget.DangerImportance

	restartBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		loadingBar.Show()
		go func() {
			onRestart()
			loadingBar.Hide()
		}()
	})

	// Update button visibility based on running state
	runningBinding.AddListener(binding.NewDataListener(func() {
		running, _ := runningBinding.Get()
		if running {
			startBtn.Disable()
			stopBtn.Enable()
			restartBtn.Enable()
		} else {
			startBtn.Enable()
			stopBtn.Disable()
			restartBtn.Disable()
		}
	}))

	// Initial state
	startBtn.Enable()
	stopBtn.Disable()
	restartBtn.Disable()

	statusRow := container.NewHBox(statusDot, statusLabel)
	infoRow := container.NewHBox(portLabel, layout.NewSpacer(), uptimeLabel)
	buttonRow := container.NewHBox(startBtn, stopBtn, restartBtn)

	content := container.NewVBox(
		statusRow,
		infoRow,
		loadingBar,
		widget.NewSeparator(),
		buttonRow,
	)

	return widget.NewCard(name, "", content)
}

// updateUptime periodically updates the uptime label
func (a *App) updateUptime(label *widget.Label, runningBinding binding.Bool, startTime *time.Time) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		running, _ := runningBinding.Get()
		if running && !startTime.IsZero() {
			uptime := time.Since(*startTime)
			label.SetText(fmt.Sprintf("Uptime: %s", formatDuration(uptime)))
		} else {
			label.SetText("Uptime: --")
		}
	}
}

// formatDuration formats a duration as human-readable string
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// createLogsView creates the logs view with auto-scroll
func (a *App) createLogsView() fyne.CanvasObject {
	// Log list with monospace font
	list := widget.NewListWithData(
		a.logs,
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.TextStyle = fyne.TextStyle{Monospace: true}
			label.Wrapping = fyne.TextWrapBreak
			return label
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.Bind(item.(binding.String))
		},
	)

	// Auto-scroll to bottom when new logs added
	a.logs.AddListener(binding.NewDataListener(func() {
		length := a.logs.Length()
		if length > 0 {
			list.ScrollToBottom()
		}
	}))

	// Log count label
	countLabel := widget.NewLabel("0 entries")
	a.logs.AddListener(binding.NewDataListener(func() {
		length := a.logs.Length()
		countLabel.SetText(fmt.Sprintf("%d entries", length))
	}))

	// Toolbar buttons
	clearBtn := widget.NewButtonWithIcon("Clear", theme.DeleteIcon(), func() {
		a.logs.Set([]string{})
		a.addLog("Logs cleared")
	})

	openLogsBtn := widget.NewButtonWithIcon("Open Folder", theme.FolderOpenIcon(), func() {
		a.OpenLogsFolder()
	})

	copyLogsBtn := widget.NewButtonWithIcon("Copy All", theme.ContentCopyIcon(), func() {
		logs, _ := a.logs.Get()
		allLogs := ""
		for _, l := range logs {
			allLogs += l + "\n"
		}
		a.mainWindow.Clipboard().SetContent(allLogs)
		a.showNotification("Copied", "Logs copied to clipboard")
	})

	toolbar := container.NewHBox(countLabel, layout.NewSpacer(), clearBtn, copyLogsBtn, openLogsBtn)

	return container.NewBorder(nil, toolbar, nil, nil, list)
}

// createSettings creates the settings view with mode-aware UI
func (a *App) createSettings() fyne.CanvasObject {
	// Container for mode-specific settings (will be swapped)
	hostSettingsCard := container.NewVBox()
	clientSettingsCard := container.NewVBox()

	// Mode selection
	modeSelect := widget.NewSelect([]string{"host", "client"}, nil)

	// Backend selection (platform-aware)
	backendOptions := []string{"mlx", "vllm"}
	backendSelect := widget.NewSelect(backendOptions, nil)
	backendSelect.SetSelected(string(a.config.Backend))

	// === HOST MODE SETTINGS ===
	backendPortEntry := widget.NewEntry()
	backendPortEntry.SetText(strconv.Itoa(a.config.BackendPort))

	slmPortEntry := widget.NewEntry()
	slmPortEntry.SetText(strconv.Itoa(a.config.SLMPort))

	// Searchable Model Select with favorites at top
	modelSelect := a.createModelSelect(string(a.config.Backend))
	modelSearchEntry := widget.NewEntry()
	modelSearchEntry.SetPlaceHolder("Search HuggingFace models...")
	searchResults := container.NewVBox()
	searchLoading := widget.NewProgressBarInfinite()
	searchLoading.Hide()

	modelSearchEntry.OnChanged = func(query string) {
		if len(query) < 3 {
			searchResults.Objects = nil
			searchResults.Refresh()
			return
		}
		searchLoading.Show()
		go func() {
			results := SearchHuggingFaceModels(query, string(a.config.Backend))
			searchResults.Objects = nil
			for _, m := range results {
				model := m // capture
				btn := widget.NewButton(fmt.Sprintf("%s (%s)", model.Name, model.Author), func() {
					modelSelect.SetSelected(model.ID)
					a.config.Model = model.ID
					searchResults.Objects = nil
					searchResults.Refresh()
					modelSearchEntry.SetText("")
				})
				btn.Alignment = widget.ButtonAlignLeading
				searchResults.Add(btn)
			}
			searchLoading.Hide()
			searchResults.Refresh()
		}()
	}

	// Backend change updates model list
	backendSelect.OnChanged = func(backend string) {
		a.config.Backend = BackendType(backend)
		newSelect := a.createModelSelect(backend)
		modelSelect.Options = newSelect.Options
		if backend == "mlx" && !containsMLX(a.config.Model) {
			a.config.Model = "mlx-community/Qwen2.5-3B-Instruct-4bit"
		} else if backend == "vllm" && containsMLX(a.config.Model) {
			a.config.Model = "Qwen/Qwen2.5-3B-Instruct"
		}
		modelSelect.SetSelected(a.config.Model)
	}

	hostSettingsCard.Objects = []fyne.CanvasObject{
		widget.NewForm(
			widget.NewFormItem("Backend", backendSelect),
			widget.NewFormItem("Backend Port", backendPortEntry),
			widget.NewFormItem("SLM Port", slmPortEntry),
			widget.NewFormItem("Model", modelSelect),
		),
		widget.NewLabel("Search for more models:"),
		modelSearchEntry,
		searchLoading,
		searchResults,
	}

	// === CLIENT MODE SETTINGS ===
	profileSelect := widget.NewSelect([]string{}, nil)
	a.updateProfileSelect(profileSelect)

	remoteHostEntry := widget.NewEntry()
	remoteHostEntry.SetText(a.config.RemoteHost)
	remoteHostEntry.SetPlaceHolder("192.168.1.100 or hostname")

	remotePortEntry := widget.NewEntry()
	remotePortEntry.SetText(strconv.Itoa(a.config.RemotePort))

	profileNameEntry := widget.NewEntry()
	profileNameEntry.SetPlaceHolder("Profile name (e.g., Homebox)")

	testConnBtn := widget.NewButtonWithIcon("Test Connection", theme.ConfirmIcon(), func() {
		host := remoteHostEntry.Text
		port, _ := strconv.Atoi(remotePortEntry.Text)
		if host == "" {
			a.ShowErrorDialog("Error", "Enter a remote host")
			return
		}
		a.config.RemoteHost = host
		a.config.RemotePort = port
		if a.checkRemoteHealth() {
			a.ShowInfoDialog("Success", fmt.Sprintf("Connected to %s:%d", host, port))
		} else {
			a.ShowErrorDialog("Failed", fmt.Sprintf("Cannot reach %s:%d", host, port))
		}
	})

	saveProfileBtn := widget.NewButtonWithIcon("Save Profile", theme.DocumentSaveIcon(), func() {
		name := profileNameEntry.Text
		if name == "" {
			name = remoteHostEntry.Text
		}
		host := remoteHostEntry.Text
		port, _ := strconv.Atoi(remotePortEntry.Text)
		if host == "" {
			a.ShowErrorDialog("Error", "Enter a remote host")
			return
		}
		profile := RemoteProfile{Name: name, Host: host, Port: port}
		a.config.RemoteProfiles = append(a.config.RemoteProfiles, profile)
		a.updateProfileSelect(profileSelect)
		profileNameEntry.SetText("")
		a.showNotification("Saved", fmt.Sprintf("Profile '%s' saved", name))
	})

	profileSelect.OnChanged = func(selected string) {
		for i, p := range a.config.RemoteProfiles {
			if p.Name == selected {
				remoteHostEntry.SetText(p.Host)
				remotePortEntry.SetText(strconv.Itoa(p.Port))
				a.config.ActiveProfileID = i
				break
			}
		}
	}

	clientSettingsCard.Objects = []fyne.CanvasObject{
		widget.NewLabel("Saved Profiles:"),
		profileSelect,
		widget.NewSeparator(),
		widget.NewForm(
			widget.NewFormItem("Profile Name", profileNameEntry),
			widget.NewFormItem("Remote Host", remoteHostEntry),
			widget.NewFormItem("Remote Port", remotePortEntry),
		),
		container.NewHBox(testConnBtn, saveProfileBtn),
	}

	// === MODE SWITCHING ===
	hostCard := widget.NewCard("Host Settings", "Run local inference", hostSettingsCard)
	clientCard := widget.NewCard("Client Settings", "Connect to remote server", clientSettingsCard)

	// Initially show based on mode
	if a.config.Mode == ModeClient {
		hostCard.Hide()
	} else {
		clientCard.Hide()
	}

	modeSelect.OnChanged = func(mode string) {
		a.config.Mode = AppMode(mode)
		if mode == "host" {
			hostCard.Show()
			clientCard.Hide()
		} else {
			hostCard.Hide()
			clientCard.Show()
		}
	}
	modeSelect.SetSelected(string(a.config.Mode))

	// === COMMON SETTINGS ===
	launchAtLoginCheck := widget.NewCheck("Launch at login", func(checked bool) {
		a.SetLaunchAtLogin(checked)
	})
	launchAtLoginCheck.SetChecked(a.config.LaunchAtLogin)

	autoStartCheck := widget.NewCheck("Auto-start services on launch", func(checked bool) {
		a.config.AutoStartAll = checked
	})
	autoStartCheck.SetChecked(a.config.AutoStartAll)

	saveBtn := widget.NewButtonWithIcon("Save All Settings", theme.DocumentSaveIcon(), func() {
		a.config.BackendPort, _ = strconv.Atoi(backendPortEntry.Text)
		a.config.SLMPort, _ = strconv.Atoi(slmPortEntry.Text)
		a.config.RemoteHost = remoteHostEntry.Text
		a.config.RemotePort, _ = strconv.Atoi(remotePortEntry.Text)
		if err := a.SaveConfig(a.config); err != nil {
			a.ShowErrorDialog("Error", "Failed to save: "+err.Error())
		} else {
			a.showNotification("Saved", "Settings saved successfully")
		}
	})
	saveBtn.Importance = widget.HighImportance

	resetBtn := widget.NewButtonWithIcon("Reset to Defaults", theme.ContentClearIcon(), func() {
		a.ShowConfirmDialog("Reset Settings", "Reset all settings to defaults?", func(confirmed bool) {
			if confirmed {
				newApp := NewApp()
				a.config = newApp.config
				a.SaveConfig(a.config)
				a.showNotification("Reset", "Settings reset. Restart to apply.")
			}
		})
	})

	openConfigBtn := widget.NewButtonWithIcon("Config", theme.FolderOpenIcon(), func() { a.OpenConfigFolder() })
	openLogsBtn := widget.NewButtonWithIcon("Logs", theme.FolderOpenIcon(), func() { a.OpenLogsFolder() })
	updateBtn := widget.NewButtonWithIcon("Check Updates", theme.DownloadIcon(), func() { go a.CheckForUpdates() })

	commitShort := Commit
	if len(Commit) > 7 {
		commitShort = Commit[:7]
	}
	versionLabel := widget.NewLabel(fmt.Sprintf("Version: %s (%s) - %s/%s", Version, commitShort, runtime.GOOS, runtime.GOARCH))

	// === SERVICES SECTION ===
	servicesContainer := container.NewVBox()
	servicesLoading := widget.NewProgressBarInfinite()
	servicesLoading.Hide()

	refreshServicesBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		servicesLoading.Show()
		go func() {
			a.providerMgr.DiscoverServices(true)
			services := a.providerMgr.GetServices()
			servicesContainer.Objects = nil
			for _, svc := range services {
				servicesContainer.Add(a.createCompactServiceItem(svc))
			}
			if len(services) == 0 {
				servicesContainer.Add(widget.NewLabel("No services discovered"))
			}
			servicesContainer.Refresh()
			servicesLoading.Hide()
		}()
	})

	authFolderBtn := widget.NewButtonWithIcon("Auth Folder", theme.FolderOpenIcon(), func() {
		a.OpenAuthFolder()
	})

	// Initial services load
	services := a.providerMgr.GetServices()
	for _, svc := range services {
		servicesContainer.Add(a.createCompactServiceItem(svc))
	}
	if len(services) == 0 {
		servicesContainer.Add(widget.NewLabel("Click Refresh to discover services"))
	}

	servicesCard := widget.NewCard("Connected Services", "Manage provider authentication",
		container.NewVBox(
			container.NewBorder(nil, nil, nil, container.NewHBox(authFolderBtn, refreshServicesBtn), servicesLoading),
			servicesContainer,
		),
	)

	return container.NewVScroll(container.NewVBox(
		widget.NewCard("Mode", "", widget.NewForm(widget.NewFormItem("Mode", modeSelect))),
		hostCard,
		clientCard,
		servicesCard,
		widget.NewCard("Startup", "", container.NewVBox(launchAtLoginCheck, autoStartCheck)),
		widget.NewCard("Files & Updates", "", container.NewHBox(openConfigBtn, openLogsBtn, layout.NewSpacer(), updateBtn)),
		container.NewHBox(saveBtn, resetBtn),
		widget.NewSeparator(),
		versionLabel,
	))
}

// createCompactServiceItem creates a compact service item for settings view
func (a *App) createCompactServiceItem(svc ServiceDiscoveryInfo) fyne.CanvasObject {
	authStatus := a.authMgr.GetStatus(svc.ID)

	// Status dot
	statusDot := canvas.NewCircle(colorStopped)
	statusDot.Resize(fyne.NewSize(10, 10))
	if svc.Available {
		if authStatus != nil && authStatus.IsAuthenticated && !authStatus.IsExpired() {
			statusDot.FillColor = colorRunning
		} else if svc.IsConfigBased {
			statusDot.FillColor = colorRunning
		} else {
			statusDot.FillColor = colorPending
		}
	}
	statusContainer := container.NewWithoutLayout(statusDot)

	// Name
	nameLabel := widget.NewLabel(svc.DisplayName)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Status text
	statusText := "Not Connected"
	if svc.IsConfigBased {
		statusText = "Local"
	} else if authStatus != nil && authStatus.IsAuthenticated {
		if authStatus.IsExpired() {
			statusText = "Expired"
		} else if authStatus.Email != "" {
			statusText = authStatus.Email
		} else {
			statusText = "Connected"
		}
	}
	statusLabel := widget.NewLabel(statusText)
	statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Action button
	var actionBtn *widget.Button
	if svc.IsConfigBased {
		actionBtn = widget.NewButtonWithIcon("", theme.ComputerIcon(), nil)
		actionBtn.Disable()
	} else if authStatus != nil && authStatus.IsAuthenticated && !authStatus.IsExpired() {
		actionBtn = widget.NewButtonWithIcon("", theme.LogoutIcon(), func() {
			a.authMgr.Disconnect(svc.ID)
		})
	} else {
		actionBtn = widget.NewButtonWithIcon("", theme.LoginIcon(), func() {
			a.showProviderConnectDialog(svc)
		})
	}

	return container.NewHBox(statusContainer, nameLabel, layout.NewSpacer(), statusLabel, actionBtn)
}

// createModelSelect creates a select widget with favorite models
func (a *App) createModelSelect(backend string) *widget.Select {
	favorites := GetFavoriteModels(backend)
	options := make([]string, len(favorites))
	for i, m := range favorites {
		options[i] = m.ID
	}

	modelSelect := widget.NewSelect(options, func(selected string) {
		a.config.Model = selected
	})
	modelSelect.SetSelected(a.config.Model)
	return modelSelect
}

// updateProfileSelect updates the profile select options
func (a *App) updateProfileSelect(sel *widget.Select) {
	options := make([]string, len(a.config.RemoteProfiles))
	for i, p := range a.config.RemoteProfiles {
		options[i] = p.Name
	}
	sel.Options = options
	if len(options) > 0 && a.config.ActiveProfileID < len(options) {
		sel.SetSelected(options[a.config.ActiveProfileID])
	}
	sel.Refresh()
}

// containsMLX checks if a model string is for MLX
func containsMLX(model string) bool {
	return len(model) > 3 && (model[:3] == "mlx" || (len(model) > 12 && model[:12] == "mlx-community"))
}

// setupSystemTray configures the system tray menu with dynamic status
func (a *App) setupSystemTray(desk desktop.App) {
	// Status menu items (will be updated dynamically)
	backendItem := fyne.NewMenuItem(a.getBackendMenuLabel(), nil)
	backendItem.Disabled = true

	slmItem := fyne.NewMenuItem(a.getSLMMenuLabel(), nil)
	slmItem.Disabled = true

	// Update status items when bindings change
	a.backendRunning.AddListener(binding.NewDataListener(func() {
		backendItem.Label = a.getBackendMenuLabel()
	}))
	a.slmRunningBind.AddListener(binding.NewDataListener(func() {
		slmItem.Label = a.getSLMMenuLabel()
	}))

	menu := fyne.NewMenu("SLM Manager",
		fyne.NewMenuItem("Show Dashboard", func() {
			a.mainWindow.Show()
			a.mainWindow.RequestFocus()
		}),
		fyne.NewMenuItemSeparator(),
		backendItem,
		slmItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Start All", func() {
			go a.StartAll()
		}),
		fyne.NewMenuItem("Stop All", func() {
			go a.StopAll()
		}),
		fyne.NewMenuItem("Restart All", func() {
			go a.RestartAll()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Copy Server URL", func() {
			a.CopyServerURL()
		}),
		fyne.NewMenuItem("Open Config Folder", func() {
			a.OpenConfigFolder()
		}),
		fyne.NewMenuItem("Open Logs Folder", func() {
			a.OpenLogsFolder()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Check for Updates", func() {
			go a.CheckForUpdates()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			a.Shutdown()
			a.fyneApp.Quit()
		}),
	)

	desk.SetSystemTrayMenu(menu)
	desk.SetSystemTrayIcon(theme.ComputerIcon())
}

// getBackendMenuLabel returns the backend status label for tray menu
func (a *App) getBackendMenuLabel() string {
	running, _ := a.backendRunning.Get()
	status := "Stopped"
	if running {
		status = "Running"
	}
	return fmt.Sprintf("%s: %s", a.getBackendDisplayName(), status)
}

// getSLMMenuLabel returns the SLM server status label for tray menu
func (a *App) getSLMMenuLabel() string {
	running, _ := a.slmRunningBind.Get()
	status := "Stopped"
	if running {
		status = "Running"
	}
	return fmt.Sprintf("SLM Server: %s", status)
}


// createProvidersView creates the service providers management view
func (a *App) createProvidersView() fyne.CanvasObject {
	// Header
	title := widget.NewLabel("Service Providers")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	subtitle := widget.NewLabel("Manage LLM provider connections and authentication")
	subtitle.Alignment = fyne.TextAlignCenter

	// Loading indicator
	loadingBar := widget.NewProgressBarInfinite()
	loadingBar.Hide()

	// Service cards container (will be refreshed)
	serviceCards := container.NewVBox()

	// Function to refresh the service list
	refreshServices := func() {
		loadingBar.Show()
		go func() {
			a.providerMgr.DiscoverServices(true)
			services := a.providerMgr.GetServices()

			// Update UI on main thread
			serviceCards.Objects = nil
			for _, svc := range services {
				card := a.createServiceItemView(svc)
				serviceCards.Add(card)
			}

			if len(services) == 0 {
				placeholder := widget.NewLabel("No services discovered. Click Refresh to discover services.")
				placeholder.Alignment = fyne.TextAlignCenter
				serviceCards.Add(placeholder)
			}

			serviceCards.Refresh()
			loadingBar.Hide()
		}()
	}

	// Refresh button
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), refreshServices)

	// Open auth folder button
	authFolderBtn := widget.NewButtonWithIcon("Auth Folder", theme.FolderOpenIcon(), func() {
		a.OpenAuthFolder()
	})

	header := container.NewVBox(
		title,
		subtitle,
		container.NewBorder(nil, nil, nil, container.NewHBox(authFolderBtn, refreshBtn), loadingBar),
	)

	// Initial load
	services := a.providerMgr.GetServices()
	for _, svc := range services {
		card := a.createServiceItemView(svc)
		serviceCards.Add(card)
	}

	if len(services) == 0 {
		placeholder := widget.NewLabel("Loading services...")
		placeholder.Alignment = fyne.TextAlignCenter
		serviceCards.Add(placeholder)
		// Trigger initial discovery
		go refreshServices()
	}

	scroll := container.NewVScroll(serviceCards)

	return container.NewBorder(header, nil, nil, nil, scroll)
}

// createServiceItemView creates a service item view like vibeproxy's ServiceItemView
func (a *App) createServiceItemView(svc ServiceDiscoveryInfo) fyne.CanvasObject {
	// Get auth status for this service
	authStatus := a.authMgr.GetStatus(svc.ID)

	// Status indicator - green if available AND authenticated, yellow if available but not auth, red if unavailable
	statusDot := canvas.NewCircle(colorStopped)
	statusDot.Resize(fyne.NewSize(12, 12))
	if svc.Available {
		if authStatus != nil && authStatus.IsAuthenticated && !authStatus.IsExpired() {
			statusDot.FillColor = colorRunning // Green - connected
		} else if svc.IsConfigBased {
			statusDot.FillColor = colorRunning // Green - local service
		} else {
			statusDot.FillColor = colorPending // Yellow - available but not authenticated
		}
	}
	statusContainer := container.NewWithoutLayout(statusDot)

	// Service icon (using theme icons as placeholders)
	icon := a.getServiceIcon(svc.ID)

	// Provider name
	nameLabel := widget.NewLabel(svc.DisplayName)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Auth status text
	authStatusLabel := widget.NewLabel("")
	authStatusLabel.TextStyle = fyne.TextStyle{Italic: true}
	if svc.IsConfigBased {
		authStatusLabel.SetText("Local Service")
	} else if authStatus != nil && authStatus.IsAuthenticated {
		if authStatus.IsExpired() {
			authStatusLabel.SetText("⚠ Session Expired")
		} else if authStatus.Email != "" {
			authStatusLabel.SetText(fmt.Sprintf("✓ %s", authStatus.Email))
		} else {
			authStatusLabel.SetText("✓ Connected")
		}
	} else {
		authStatusLabel.SetText("Not Connected")
	}

	// Model count badge
	modelCountLabel := widget.NewLabel("")
	if svc.ModelCount > 0 {
		modelCountLabel.SetText(fmt.Sprintf("%d models", svc.ModelCount))
	}

	// Models section with loading
	modelsLabel := widget.NewLabel("")
	modelsLabel.Wrapping = fyne.TextWrapWord
	loadingModels := widget.NewProgressBarInfinite()
	loadingModels.Hide()

	modelsContainer := container.NewVBox(loadingModels, modelsLabel)
	modelsContainer.Hide()

	// Action buttons based on auth state
	var actionBtn *widget.Button
	if svc.IsConfigBased {
		actionBtn = widget.NewButtonWithIcon("Local", theme.ComputerIcon(), nil)
		actionBtn.Disable()
	} else if authStatus != nil && authStatus.IsAuthenticated && !authStatus.IsExpired() {
		// Show disconnect button
		actionBtn = widget.NewButtonWithIcon("Disconnect", theme.LogoutIcon(), func() {
			dialog.ShowConfirm("Disconnect", fmt.Sprintf("Disconnect from %s?", svc.DisplayName),
				func(confirmed bool) {
					if confirmed {
						if err := a.authMgr.Disconnect(svc.ID); err != nil {
							a.ShowErrorDialog("Error", fmt.Sprintf("Failed to disconnect: %v", err))
						} else {
							a.addLog(fmt.Sprintf("Disconnected from %s", svc.DisplayName))
							a.showNotification("Disconnected", fmt.Sprintf("Disconnected from %s", svc.DisplayName))
						}
					}
				}, a.mainWindow)
		})
		actionBtn.Importance = widget.DangerImportance
	} else if authStatus != nil && authStatus.IsExpired() {
		// Show reconnect button
		actionBtn = widget.NewButtonWithIcon("Reconnect", theme.ViewRefreshIcon(), func() {
			a.showProviderConnectDialog(svc)
		})
		actionBtn.Importance = widget.WarningImportance
	} else {
		// Show connect button
		actionBtn = widget.NewButtonWithIcon("Connect", theme.LoginIcon(), func() {
			a.showProviderConnectDialog(svc)
		})
		actionBtn.Importance = widget.HighImportance
	}

	// Expand to show models button
	var expanded bool
	expandBtn := widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), func() {
		expanded = !expanded
		if expanded {
			modelsContainer.Show()
			loadingModels.Show()
			// Load models asynchronously
			a.providerMgr.FetchModelsAsync(svc.ID, func(models []ProviderModel) {
				if len(models) > 0 {
					modelText := ""
					for i, m := range models {
						if i > 0 {
							modelText += "\n"
						}
						modelText += fmt.Sprintf("• %s", m.DisplayName)
						if m.Description != "" && len(m.Description) < 50 {
							modelText += fmt.Sprintf(" - %s", m.Description)
						}
					}
					modelsLabel.SetText(modelText)
				} else {
					modelsLabel.SetText("No models available")
				}
				loadingModels.Hide()
			})
		} else {
			modelsContainer.Hide()
		}
	})

	// Card layout
	headerRow := container.NewHBox(
		statusContainer,
		icon,
		nameLabel,
		layout.NewSpacer(),
		modelCountLabel,
	)

	authRow := container.NewHBox(
		authStatusLabel,
		layout.NewSpacer(),
		expandBtn,
		actionBtn,
	)

	content := container.NewVBox(
		headerRow,
		authRow,
		modelsContainer,
	)

	card := widget.NewCard("", "", content)
	return card
}

// getServiceIcon returns an icon for a service
func (a *App) getServiceIcon(serviceID string) fyne.CanvasObject {
	// Map service IDs to theme icons (in a real app, use custom icons)
	var icon fyne.Resource
	switch serviceID {
	case "anthropic", "claude":
		icon = theme.AccountIcon()
	case "openai", "gpt":
		icon = theme.ComputerIcon()
	case "google", "gemini":
		icon = theme.SearchIcon()
	case "openrouter":
		icon = theme.StorageIcon()
	case "ollama", "local-slm":
		icon = theme.HomeIcon()
	default:
		icon = theme.QuestionIcon()
	}
	return widget.NewIcon(icon)
}

// showProviderConnectDialog shows dialog to connect to a provider
func (a *App) showProviderConnectDialog(svc ServiceDiscoveryInfo) {
	// Get provider type info for auth details
	providerTypes := a.providerMgr.GetProviderTypes()
	var providerType *ProviderTypeInfo
	for _, pt := range providerTypes {
		if pt.Name == svc.ID {
			providerType = &pt
			break
		}
	}

	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder("Enter API Key")

	var docLink *widget.Hyperlink
	if providerType != nil && providerType.DocURL != "" {
		docLink = widget.NewHyperlink("Get API Key →", nil)
		docLink.OnTapped = func() {
			// Open URL in browser
			a.addLog(fmt.Sprintf("Opening docs: %s", providerType.DocURL))
		}
	}

	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Connect to %s", svc.DisplayName)),
		widget.NewSeparator(),
		widget.NewLabel("API Key:"),
		apiKeyEntry,
	)

	if docLink != nil {
		content.Add(docLink)
	}

	dialog.ShowCustomConfirm("Connect Provider", "Connect", "Cancel", content,
		func(confirmed bool) {
			if confirmed && apiKeyEntry.Text != "" {
				// Try to authenticate via API
				success, msg := a.apiClient.AuthenticateService(svc.ID)
				if success {
					a.addLog(fmt.Sprintf("Connected to %s", svc.DisplayName))
					a.showNotification("Connected", msg)
					// Refresh auth status
					a.authMgr.CheckAuthStatus()
				} else {
					a.ShowErrorDialog("Connection Failed", msg)
				}
			}
		}, a.mainWindow)
}

// OpenAuthFolder opens the auth folder in file manager
func (a *App) OpenAuthFolder() {
	authDir := getAuthDir()
	a.openFolder(authDir)
}

// createLoadingSkeleton creates a loading skeleton placeholder
func createLoadingSkeleton() fyne.CanvasObject {
	skeleton := canvas.NewRectangle(color.NRGBA{R: 200, G: 200, B: 200, A: 100})
	skeleton.SetMinSize(fyne.NewSize(200, 20))
	return skeleton
}