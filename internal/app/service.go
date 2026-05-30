package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ibkr-stock-analysis/internal/agent"
	"ibkr-stock-analysis/internal/analysis"
	"ibkr-stock-analysis/internal/domain"
	"ibkr-stock-analysis/internal/market"
	"ibkr-stock-analysis/internal/storage"
)

type EventEmitter func(ctx context.Context, name string, payload any)

type Service struct {
	mu          sync.Mutex
	store       *storage.Store
	provider    market.MarketDataProvider
	agentClient agent.Client
	emit        EventEmitter
	state       domain.AppState
	history     map[string]domain.AnalysisResult
}

func NewService(store *storage.Store, provider market.MarketDataProvider, agentClient agent.Client, emit EventEmitter) *Service {
	snapshot, _ := store.Load()
	settings := snapshot.Settings.Normalize()
	service := &Service{
		store:       store,
		provider:    provider,
		agentClient: agentClient,
		emit:        emit,
		state: domain.AppState{
			Settings:         settings,
			ConnectionStatus: provider.State().Status,
			Symbols:          symbolStates(settings.Watchlist),
		},
		history: make(map[string]domain.AnalysisResult),
	}
	for _, result := range snapshot.History {
		service.history[result.Symbol] = result
	}
	service.applyHistory()
	return service
}

func (s *Service) GetState(ctx context.Context) (domain.AppState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state, nil
}

func (s *Service) SaveSettings(ctx context.Context, settings domain.Settings) (domain.AppState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Settings = settings.Normalize()
	s.state.Symbols = symbolStates(s.state.Settings.Watchlist)
	s.applyHistory()
	if err := s.persistLocked(); err != nil {
		return domain.AppState{}, err
	}
	s.emitLocked(ctx, "settings:update", s.state)
	return s.state, nil
}

func (s *Service) ConnectIBKR(ctx context.Context) (domain.AppState, error) {
	s.mu.Lock()
	settings := s.state.Settings
	s.mu.Unlock()

	err := s.provider.Connect(ctx, market.ConnectionSettings{
		Host:     settings.IBKRHost,
		Port:     settings.IBKRPort,
		ClientID: settings.IBKRClientID,
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ConnectionStatus = s.provider.State().Status
	if err != nil {
		s.state.LastError = err.Error()
		s.emitLocked(ctx, "connection:update", s.state)
		return s.state, err
	}
	s.state.LastError = ""
	s.emitLocked(ctx, "connection:update", s.state)
	return s.state, nil
}

func (s *Service) DisconnectIBKR(ctx context.Context) (domain.AppState, error) {
	err := s.provider.Disconnect(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ConnectionStatus = s.provider.State().Status
	if err != nil {
		s.state.LastError = err.Error()
		return s.state, err
	}
	s.emitLocked(ctx, "connection:update", s.state)
	return s.state, nil
}

func (s *Service) RunAnalysisNow(ctx context.Context) (domain.AppState, error) {
	s.mu.Lock()
	settings := s.state.Settings
	if s.state.ConnectionStatus != domain.ConnectionConnected {
		s.mu.Unlock()
		return domain.AppState{}, fmt.Errorf("IBKR is not connected")
	}
	s.markAll(domain.JobStatusQueued, "")
	s.emitLocked(ctx, "analysis:update", s.state)
	s.mu.Unlock()

	inputs := make([]domain.AgentInput, 0, len(settings.Watchlist))
	for _, symbol := range settings.Watchlist {
		bars, err := s.provider.HistoricalBars(ctx, symbol, settings.SelectedTimeframe, 120)
		if err != nil {
			s.setSymbolError(ctx, symbol, domain.JobStatusNoData, err.Error())
			continue
		}
		if len(bars) == 0 {
			s.setSymbolError(ctx, symbol, domain.JobStatusNoData, "No Data")
			continue
		}
		derived := analysis.DeriveFeatures(bars)
		inputs = append(inputs, domain.AgentInput{
			Symbol:       symbol,
			Timeframe:    settings.SelectedTimeframe,
			CurrentPrice: bars[len(bars)-1].Close,
			Bars:         bars,
			Derived:      derived,
		})
		s.setSymbolPrice(ctx, symbol, bars[len(bars)-1])
	}

	queue := analysis.NewQueue(s.agentClient, 3)
	results := queue.RunBatch(ctx, inputs, s.history)

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, job := range results {
		for i := range s.state.Symbols {
			if s.state.Symbols[i].Symbol != job.Symbol {
				continue
			}
			s.state.Symbols[i].JobStatus = job.Status
			s.state.Symbols[i].Error = job.Error
			if job.Result != nil {
				s.state.Symbols[i].Result = job.Result
				now := job.Result.UpdatedAt
				s.state.Symbols[i].LastAnalysisTime = &now
				if !job.Result.Stale {
					s.history[job.Symbol] = *job.Result
				}
			}
		}
	}
	if err := s.persistLocked(); err != nil {
		return domain.AppState{}, err
	}
	s.emitLocked(ctx, "analysis:update", s.state)
	return s.state, nil
}

func (s *Service) setSymbolError(ctx context.Context, symbol string, status domain.JobStatus, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Symbols {
		if s.state.Symbols[i].Symbol == symbol {
			s.state.Symbols[i].JobStatus = status
			s.state.Symbols[i].Error = message
		}
	}
	s.emitLocked(ctx, "analysis:error", s.state)
}

func (s *Service) setSymbolPrice(ctx context.Context, symbol string, bar domain.Bar) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Symbols {
		if s.state.Symbols[i].Symbol == symbol {
			closeTime := bar.Time
			price := bar.Close
			s.state.Symbols[i].LastClosedBarTime = &closeTime
			s.state.Symbols[i].CurrentPrice = &price
			s.state.Symbols[i].MarketDataStatus = "ready"
			s.state.Symbols[i].JobStatus = domain.JobStatusQueued
		}
	}
	s.emitLocked(ctx, "market:update", s.state)
}

func (s *Service) markAll(status domain.JobStatus, message string) {
	for i := range s.state.Symbols {
		s.state.Symbols[i].JobStatus = status
		s.state.Symbols[i].Error = message
	}
}

func (s *Service) persistLocked() error {
	history := make([]domain.AnalysisResult, 0, len(s.history))
	for _, result := range s.history {
		history = append(history, result)
	}
	return s.store.Save(storage.Snapshot{Settings: s.state.Settings, History: history})
}

func (s *Service) applyHistory() {
	for i := range s.state.Symbols {
		if result, ok := s.history[s.state.Symbols[i].Symbol]; ok {
			resultCopy := result
			s.state.Symbols[i].Result = &resultCopy
			updatedAt := result.UpdatedAt
			if updatedAt.IsZero() {
				updatedAt = time.Now().UTC()
			}
			s.state.Symbols[i].LastAnalysisTime = &updatedAt
		}
	}
}

func (s *Service) emitLocked(ctx context.Context, name string, payload any) {
	if s.emit != nil {
		s.emit(ctx, name, payload)
	}
}

func symbolStates(symbols []string) []domain.SymbolState {
	states := make([]domain.SymbolState, 0, len(symbols))
	for _, symbol := range symbols {
		states = append(states, domain.SymbolState{
			Symbol:           symbol,
			MarketDataStatus: "idle",
			JobStatus:        domain.JobStatusIdle,
		})
	}
	return states
}
