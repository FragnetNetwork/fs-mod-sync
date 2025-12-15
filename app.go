package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/FragnetNetwork/fs-mod-sync/internal/config"
	"github.com/FragnetNetwork/fs-mod-sync/internal/downloader"
	"github.com/FragnetNetwork/fs-mod-sync/internal/models"
	"github.com/FragnetNetwork/fs-mod-sync/internal/parser"
	"github.com/FragnetNetwork/fs-mod-sync/internal/scanner"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var Version = "dev"

type App struct {
	ctx      context.Context
	cancelMu sync.Mutex
	cancelFn context.CancelFunc
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		return &config.Config{}
	}
	return cfg
}

func (a *App) SaveConfig(cfg *config.Config) error {
	return config.Save(cfg)
}

func (a *App) ValidateURL(url string) models.ValidationResult {
	resp, err := http.Get(url)
	if err != nil {
		return models.ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("Failed to connect: %s", err.Error()),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("Server returned status %d", resp.StatusCode),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("Failed to read response: %s", err.Error()),
		}
	}

	html := string(body)

	if !parser.IsValidModsPage(html) {
		return models.ValidationResult{
			Valid: false,
			Error: "Public Mod Download is not enabled. Please enable it in the Farming Simulator control panel.",
		}
	}

	mods, gameVersion, err := parser.ParseHTML(html, url)
	if err != nil {
		return models.ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("Failed to parse page: %s", err.Error()),
		}
	}

	downloadableMods := 0
	for _, mod := range mods {
		if !mod.IsDLC && mod.URL != "" {
			downloadableMods++
		}
	}

	return models.ValidationResult{
		Valid:       true,
		GameVersion: gameVersion,
		ModCount:    downloadableMods,
	}
}

func (a *App) GetDefaultModsDir(gameVersion string) string {
	return config.GetDefaultModsDir(gameVersion)
}

func (a *App) BrowseDirectory() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Mods Directory",
	})
	if err != nil {
		return ""
	}
	return dir
}

func (a *App) GetSyncStatus(serverURL, modsDir string) (*models.SyncResult, error) {
	mods, gameVersion, err := parser.FetchAndParse(serverURL)
	if err != nil {
		return nil, err
	}

	localMods, err := scanner.ScanLocalMods(modsDir)
	if err != nil {
		return nil, err
	}

	var totalSize int64
	var modsToSync int

	for i := range mods {
		if mods[i].IsDLC || mods[i].URL == "" {
			continue
		}

		localDesc, exists := localMods[mods[i].Filename]
		if !exists {
			mods[i].NeedsUpdate = true
			mods[i].LocalVersion = ""
			totalSize += mods[i].SizeBytes
			modsToSync++
		} else if localDesc.Version != mods[i].Version && localDesc.Version != "unknown" {
			mods[i].NeedsUpdate = true
			mods[i].LocalVersion = localDesc.Version
			totalSize += mods[i].SizeBytes
			modsToSync++
		} else {
			mods[i].LocalVersion = localDesc.Version
		}
	}

	status := models.SyncStatus{
		TotalMods:      len(mods),
		ModsToSync:     modsToSync,
		TotalSize:      parser.FormatSize(totalSize),
		TotalSizeBytes: totalSize,
		GameVersion:    gameVersion,
	}

	return &models.SyncResult{
		Status: status,
		Mods:   mods,
	}, nil
}

func (a *App) StartSync(serverURL, modsDir string) error {
	mods, _, err := parser.FetchAndParse(serverURL)
	if err != nil {
		return err
	}

	localMods, err := scanner.ScanLocalMods(modsDir)
	if err != nil {
		return err
	}

	var modsToDownload []models.Mod
	for _, mod := range mods {
		if mod.IsDLC || mod.URL == "" {
			continue
		}

		localDesc, exists := localMods[mod.Filename]
		if !exists || (localDesc.Version != mod.Version && localDesc.Version != "unknown") {
			modsToDownload = append(modsToDownload, mod)
		}
	}

	if len(modsToDownload) == 0 {
		runtime.EventsEmit(a.ctx, "sync:complete", nil)
		return nil
	}

	a.cancelMu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelFn = cancel
	a.cancelMu.Unlock()

	go func() {
		defer func() {
			a.cancelMu.Lock()
			a.cancelFn = nil
			a.cancelMu.Unlock()
		}()

		for i, mod := range modsToDownload {
			select {
			case <-ctx.Done():
				runtime.EventsEmit(a.ctx, "sync:cancelled", nil)
				return
			default:
			}

			destPath := filepath.Join(modsDir, mod.Filename)

			err := downloader.Download(ctx, mod.URL, destPath, func(downloaded, total int64, speed string) {
				progress := float64(0)
				if total > 0 {
					progress = float64(downloaded) / float64(total)
				}

				event := models.ProgressEvent{
					Filename:    mod.Filename,
					Progress:    progress,
					Downloaded:  i + 1,
					Total:       len(modsToDownload),
					CurrentSize: downloaded,
					TotalSize:   total,
					Speed:       speed,
				}
				runtime.EventsEmit(a.ctx, "download:progress", event)
			})

			if err != nil {
				if ctx.Err() != nil {
					runtime.EventsEmit(a.ctx, "sync:cancelled", nil)
					return
				}
				runtime.EventsEmit(a.ctx, "download:error", map[string]string{
					"filename": mod.Filename,
					"error":    err.Error(),
				})
				continue
			}

			runtime.EventsEmit(a.ctx, "download:complete", mod.Filename)
		}

		runtime.EventsEmit(a.ctx, "sync:complete", nil)
	}()

	return nil
}

func (a *App) CancelSync() {
	a.cancelMu.Lock()
	defer a.cancelMu.Unlock()

	if a.cancelFn != nil {
		a.cancelFn()
	}
}

func (a *App) CheckModsDirectory(path string) bool {
	return strings.TrimSpace(path) != ""
}

func (a *App) Quit() {
	runtime.Quit(a.ctx)
}

func (a *App) GetVersion() string {
	return Version
}

func (a *App) GetGitHubRepo() string {
	return "https://github.com/FragnetNetwork/fs-mod-sync"
}
