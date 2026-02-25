package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/creativeprojects/go-selfupdate"
)

// UpdateInfo contains update information
type UpdateInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ReleaseNotes    string `json:"releaseNotes"`
	ReleaseURL      string `json:"releaseUrl"`
}

// CheckForUpdates checks for available updates
func (a *App) CheckForUpdates() {
	a.addLog("Checking for updates...")

	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		a.addLog(fmt.Sprintf("ERROR: Failed to create GitHub source: %v", err))
		return
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		a.addLog(fmt.Sprintf("ERROR: Failed to create updater: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(
		fmt.Sprintf("%s/%s", a.config.RepoOwner, a.config.RepoName),
	))
	if err != nil {
		a.addLog(fmt.Sprintf("ERROR: Failed to detect latest version: %v", err))
		return
	}

	if found && latest.GreaterThan(Version) {
		a.addLog(fmt.Sprintf("Update available: %s → %s", Version, latest.Version()))
		a.showNotification("Update Available", fmt.Sprintf("Version %s is available", latest.Version()))
	} else {
		a.addLog("Already on latest version")
		a.showNotification("Up to Date", "You are running the latest version")
	}
}

// ApplyUpdate downloads and applies an update
func (a *App) ApplyUpdate() error {
	a.addLog("Downloading update...")

	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return fmt.Errorf("failed to create GitHub source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(
		fmt.Sprintf("%s/%s", a.config.RepoOwner, a.config.RepoName),
	))
	if err != nil {
		return fmt.Errorf("failed to detect latest version: %w", err)
	}

	if !found {
		return fmt.Errorf("no releases found")
	}

	if !latest.GreaterThan(Version) {
		return fmt.Errorf("already on latest version")
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	a.addLog(fmt.Sprintf("Updating to %s...", latest.Version()))

	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	a.addLog("Update complete! Restarting...")

	// Restart the application
	go func() {
		time.Sleep(1 * time.Second)
		a.restartApp()
	}()

	return nil
}

// restartApp restarts the application after an update
func (a *App) restartApp() {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get executable: %v", err)
		return
	}

	// Stop all services first
	a.Shutdown()

	// Restart
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()

	// Quit current instance
	a.fyneApp.Quit()
	os.Exit(0)
}
