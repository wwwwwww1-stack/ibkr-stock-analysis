package main

import (
	"context"

	"ibkr-stock-analysis/internal/agent"
	appsvc "ibkr-stock-analysis/internal/app"
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
	storePath, err := storage.DefaultPath()
	if err != nil {
		storePath = "settings.json"
	}
	store := storage.NewStore(storePath, 50)
	provider := market.NewIBKRProvider()
	agentClient := agent.NewProcessClient("agent-worker/dist/index.js")
	return &App{
		service: appsvc.NewService(store, provider, agentClient, nil),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.service = appsvc.NewService(a.serviceStore(), market.NewIBKRProvider(), agent.NewProcessClient("agent-worker/dist/index.js"), func(ctx context.Context, name string, payload any) {
		runtime.EventsEmit(ctx, name, payload)
	})
}

func (a *App) GetState() (domain.AppState, error) {
	return a.service.GetState(a.ctx)
}

func (a *App) SaveSettings(settings domain.Settings) (domain.AppState, error) {
	return a.service.SaveSettings(a.ctx, settings)
}

func (a *App) ConnectIBKR() (domain.AppState, error) {
	return a.service.ConnectIBKR(a.ctx)
}

func (a *App) DisconnectIBKR() (domain.AppState, error) {
	return a.service.DisconnectIBKR(a.ctx)
}

func (a *App) RunAnalysisNow() (domain.AppState, error) {
	return a.service.RunAnalysisNow(a.ctx)
}

func (a *App) serviceStore() *storage.Store {
	storePath, err := storage.DefaultPath()
	if err != nil {
		storePath = "settings.json"
	}
	return storage.NewStore(storePath, 50)
}
