package main

import (
	"context"
	"path/filepath"
	"strings"

	"ibkr-stock-analysis/internal/account"
	"ibkr-stock-analysis/internal/agent"
	appsvc "ibkr-stock-analysis/internal/app"
	"ibkr-stock-analysis/internal/capture"
	"ibkr-stock-analysis/internal/domain"
	"ibkr-stock-analysis/internal/market"
	"ibkr-stock-analysis/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx     context.Context
	service *appsvc.Service
}

// NewApp creates a new App application struct
func NewApp() *App {
	service := newRuntimeService(nil)
	service.SetChartCapturer(capture.NewInteractiveCapturer())
	service.SetScheduledAnalysisEnabled(true)
	return &App{service: service}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.service = newRuntimeService(func(ctx context.Context, name string, payload any) {
		runtime.EventsEmit(ctx, name, payload)
	})
	a.service.SetChartCapturer(capture.NewInteractiveCapturer())
	a.service.SetScheduledAnalysisEnabled(true)
}

func (a *App) shutdown(ctx context.Context) {
	if a.service != nil {
		a.service.Close()
	}
}

func (a *App) GetState() (domain.AppState, error) {
	return a.service.GetState(a.ctx)
}

func (a *App) SaveSettings(settings domain.Settings) (domain.AppState, error) {
	return a.service.SaveSettings(a.ctx, settings)
}

func (a *App) RefreshAccountSnapshot() (domain.AppState, error) {
	return a.service.RefreshAccountSnapshot(a.ctx)
}

func (a *App) ConnectIBKR() (domain.AppState, error) {
	return a.service.ConnectIBKR(a.ctx)
}

func (a *App) DisconnectIBKR() (domain.AppState, error) {
	return a.service.DisconnectIBKR(a.ctx)
}

func (a *App) SetScheduledAnalysisEnabled(enabled bool) (domain.AppState, error) {
	return a.service.UpdateScheduledAnalysisEnabled(a.ctx, enabled)
}

func (a *App) RunAnalysisNow(symbol string) (domain.AppState, error) {
	return a.service.RunAnalysisNow(a.ctx, symbol)
}

func (a *App) RunScreenshotAnalysis(symbol string) (domain.AppState, error) {
	return a.service.RunScreenshotAnalysis(a.ctx, symbol)
}

func (a *App) GetAnalysisHistory(query domain.AnalysisHistoryQuery) (domain.AnalysisHistoryPage, error) {
	return a.service.GetAnalysisHistory(a.ctx, query)
}

func (a *App) ExportAnalysisHistoryCSV(query domain.AnalysisHistoryQuery) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Signals History",
		DefaultFilename: "signals-history.csv",
		Filters: []runtime.FileFilter{{
			DisplayName: "CSV Files (*.csv)",
			Pattern:     "*.csv",
		}},
		CanCreateDirectories: true,
	})
	if err != nil {
		return "", err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if filepath.Ext(path) == "" {
		path += ".csv"
	}
	if err := a.service.ExportAnalysisHistoryCSV(a.ctx, query, path); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) RunBacktest(request domain.BacktestRequest) (domain.BacktestReport, error) {
	return a.service.RunBacktest(a.ctx, request)
}

func (a *App) ListChartWindows() ([]domain.ChartWindow, error) {
	return a.service.ListChartWindows(a.ctx)
}

func (a *App) serviceStore() *storage.Store {
	return defaultStore()
}

func newRuntimeService(emit appsvc.EventEmitter) *appsvc.Service {
	store := defaultStore()
	provider := market.NewIBKRProvider()
	accountProvider := account.NewIBKRProvider(account.IBKRProviderConfig{
		Client:        provider.AccountSnapshotClient(),
		Events:        provider.AccountSnapshotEvents(),
		NextRequestID: provider.NextAccountRequestID,
	})
	return appsvc.NewService(store, provider, agent.NewCodexClient(), emit, accountProvider)
}

func defaultStore() *storage.Store {
	storePath, err := storage.DefaultPath()
	if err != nil {
		storePath = "app.db"
	}
	legacyPath, err := storage.LegacyJSONPath()
	if err != nil {
		legacyPath = "settings.json"
	}
	return storage.NewStoreWithLegacyPath(storePath, legacyPath, 50)
}
