package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"ibkr-stock-analysis/internal/agent"
	"ibkr-stock-analysis/internal/analysis"
	"ibkr-stock-analysis/internal/backtest"
	"ibkr-stock-analysis/internal/capture"
	"ibkr-stock-analysis/internal/domain"
	"ibkr-stock-analysis/internal/market"
	"ibkr-stock-analysis/internal/scheduler"
	"ibkr-stock-analysis/internal/storage"
)

type EventEmitter func(ctx context.Context, name string, payload any)

type Service struct {
	mu          sync.Mutex
	store       *storage.Store
	provider    market.MarketDataProvider
	agentClient agent.Client
	capturer    capture.ChartCapturer
	scheduler   *scheduler.Scheduler
	emit        EventEmitter
	state       domain.AppState
	history     map[string]domain.AnalysisResult
	autoCancel  context.CancelFunc
	autoWG      sync.WaitGroup
	autoEnabled bool
	autoLead    time.Duration
}

func NewService(store *storage.Store, provider market.MarketDataProvider, agentClient agent.Client, emit EventEmitter) *Service {
	snapshot, _ := store.Load()
	settings := snapshot.Settings.Normalize()
	service := &Service{
		store:       store,
		provider:    provider,
		agentClient: agentClient,
		scheduler:   scheduler.New(),
		emit:        emit,
		state: domain.AppState{
			Settings:         settings,
			ConnectionStatus: provider.State().Status,
			Symbols:          symbolStates(settings.Watchlist),
		},
		history:  make(map[string]domain.AnalysisResult),
		autoLead: 10 * time.Second,
	}
	for _, result := range snapshot.History {
		service.history[result.Symbol] = result
	}
	service.applyHistory()
	return service
}

func (s *Service) SetScheduledAnalysisEnabled(enabled bool) {
	s.mu.Lock()
	s.autoEnabled = enabled
	s.state.ScheduledAnalysisEnabled = enabled
	s.mu.Unlock()
	if !enabled {
		s.stopScheduledAnalysis()
	}
}

func (s *Service) UpdateScheduledAnalysisEnabled(ctx context.Context, enabled bool) (domain.AppState, error) {
	s.mu.Lock()
	s.autoEnabled = enabled
	s.state.ScheduledAnalysisEnabled = enabled
	shouldStart := enabled && s.state.ConnectionStatus == domain.ConnectionConnected
	state := s.state
	s.emitLocked(ctx, "settings:update", s.state)
	s.mu.Unlock()

	if enabled {
		if shouldStart {
			s.startScheduledAnalysis(ctx)
		}
		return state, nil
	}

	s.stopScheduledAnalysis()
	return s.GetState(ctx)
}

func (s *Service) Close() {
	s.stopScheduledAnalysis()
}

func (s *Service) SetChartCapturer(capturer capture.ChartCapturer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.capturer = capturer
}

func (s *Service) ListChartWindows(ctx context.Context) ([]domain.ChartWindow, error) {
	s.mu.Lock()
	capturer := s.capturer
	s.mu.Unlock()
	lister, ok := capturer.(capture.ChartWindowLister)
	if !ok || lister == nil {
		return nil, fmt.Errorf("chart window listing is unavailable")
	}
	windows, err := lister.ListWindows(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ChartWindow, 0, len(windows))
	for _, window := range windows {
		out = append(out, domain.ChartWindow{
			ID:      window.ID,
			AppName: window.AppName,
			Title:   window.Title,
		})
	}
	return out, nil
}

func (s *Service) GetState(ctx context.Context) (domain.AppState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state, nil
}

func (s *Service) SaveSettings(ctx context.Context, settings domain.Settings) (domain.AppState, error) {
	s.mu.Lock()
	s.state.Settings = settings.Normalize()
	s.state.Symbols = symbolStates(s.state.Settings.Watchlist)
	s.applyHistory()
	if err := s.persistLocked(); err != nil {
		s.mu.Unlock()
		return domain.AppState{}, err
	}
	s.emitLocked(ctx, "settings:update", s.state)
	state := s.state
	restartScheduled := s.autoEnabled && s.state.ConnectionStatus == domain.ConnectionConnected
	s.mu.Unlock()
	if restartScheduled {
		s.startScheduledAnalysis(ctx)
	}
	return state, nil
}

func (s *Service) ConnectIBKR(ctx context.Context) (domain.AppState, error) {
	s.mu.Lock()
	settings := s.state.Settings
	s.mu.Unlock()

	var err error
	connectedPort := settings.IBKRPort
	for _, port := range ibkrConnectionPortCandidates(settings) {
		err = s.provider.Connect(ctx, market.ConnectionSettings{
			Host:     settings.IBKRHost,
			Port:     port,
			ClientID: settings.IBKRClientID,
		})
		if err == nil {
			connectedPort = port
			break
		}
	}

	s.mu.Lock()
	s.state.ConnectionStatus = s.provider.State().Status
	if err != nil {
		s.state.LastError = err.Error()
		s.emitLocked(ctx, "connection:update", s.state)
		state := s.state
		s.mu.Unlock()
		return state, err
	}
	if s.state.Settings.IBKRPort != connectedPort {
		s.state.Settings.IBKRPort = connectedPort
		if persistErr := s.persistLocked(); persistErr != nil {
			s.state.LastError = persistErr.Error()
			s.emitLocked(ctx, "connection:update", s.state)
			state := s.state
			s.mu.Unlock()
			return state, persistErr
		}
	}
	s.state.LastError = ""
	s.emitLocked(ctx, "connection:update", s.state)
	state := s.state
	startScheduled := s.autoEnabled
	s.mu.Unlock()
	if startScheduled {
		s.startScheduledAnalysis(ctx)
	}
	return state, nil
}

func ibkrConnectionPortCandidates(settings domain.Settings) []int {
	ports := []int{settings.IBKRPort}
	if !isLocalIBKRHost(settings.IBKRHost) {
		return dedupePositivePorts(ports)
	}
	ports = append(ports, 7497, 7496, 4002, 4001)
	return dedupePositivePorts(ports)
}

func isLocalIBKRHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "", "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	default:
		return false
	}
}

func dedupePositivePorts(ports []int) []int {
	seen := make(map[int]struct{}, len(ports))
	candidates := make([]int, 0, len(ports))
	for _, port := range ports {
		if port <= 0 {
			continue
		}
		if _, ok := seen[port]; ok {
			continue
		}
		seen[port] = struct{}{}
		candidates = append(candidates, port)
	}
	return candidates
}

func (s *Service) DisconnectIBKR(ctx context.Context) (domain.AppState, error) {
	s.stopScheduledAnalysis()
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

func (s *Service) RunAnalysisNow(ctx context.Context, symbol string) (domain.AppState, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return domain.AppState{}, fmt.Errorf("symbol is required")
	}

	s.mu.Lock()
	settings := s.state.Settings
	if s.state.ConnectionStatus != domain.ConnectionConnected {
		s.mu.Unlock()
		return domain.AppState{}, fmt.Errorf("IBKR is not connected")
	}
	if !containsSymbol(settings.Watchlist, symbol) {
		s.mu.Unlock()
		return domain.AppState{}, fmt.Errorf("%s is not in the watchlist", symbol)
	}
	s.markSymbol(symbol, domain.JobStatusQueued, "")
	s.emitLocked(ctx, "analysis:update", s.state)
	s.mu.Unlock()

	input, err := s.buildAgentInput(ctx, symbol, settings, nil)
	if err != nil {
		return s.setSymbolErrorAndState(ctx, symbol, domain.JobStatusNoData, err.Error())
	}
	s.setSymbolPrice(ctx, symbol, input.Bars[len(input.Bars)-1])
	previous := s.historySnapshot()

	queue := analysis.NewQueue(s.agentClient, 1)
	results := queue.RunBatch(ctx, []domain.AgentInput{input}, previous)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyJobResultsLocked(results)
	if err := s.persistJobResultsLocked(results); err != nil {
		return domain.AppState{}, err
	}
	s.emitLocked(ctx, "analysis:update", s.state)
	if len(results) > 0 && results[0].Error != "" {
		return s.state, errors.New(results[0].Error)
	}
	return s.state, nil
}

func (s *Service) RunScreenshotAnalysis(ctx context.Context, symbol string) (domain.AppState, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return domain.AppState{}, fmt.Errorf("symbol is required")
	}

	s.mu.Lock()
	settings := s.state.Settings
	capturer := s.capturer
	if s.state.ConnectionStatus != domain.ConnectionConnected {
		s.mu.Unlock()
		return domain.AppState{}, fmt.Errorf("IBKR is not connected")
	}
	if !containsSymbol(settings.Watchlist, symbol) {
		s.mu.Unlock()
		return domain.AppState{}, fmt.Errorf("%s is not in the watchlist", symbol)
	}
	s.markSymbol(symbol, domain.JobStatusQueued, "")
	s.emitLocked(ctx, "analysis:update", s.state)
	s.mu.Unlock()

	if capturer == nil {
		return s.setSymbolErrorAndState(ctx, symbol, domain.JobStatusFailed, "chart screenshot capture is unavailable")
	}
	chartCapture, err := capturer.CaptureChart(ctx, captureTarget(settings.ChartWindow))
	if err != nil {
		return s.setSymbolErrorAndState(ctx, symbol, domain.JobStatusFailed, err.Error())
	}
	if strings.TrimSpace(chartCapture.Path) != "" {
		defer os.Remove(chartCapture.Path)
	}

	capturedAt := chartCapture.CapturedAt
	chartImage := &domain.ChartImageInput{
		Provided:   true,
		Source:     chartCapture.Source,
		MimeType:   chartCapture.MimeType,
		CapturedAt: &capturedAt,
		Path:       chartCapture.Path,
	}
	input, err := s.buildAgentInput(ctx, symbol, settings, chartImage)
	if err != nil {
		return s.setSymbolErrorAndState(ctx, symbol, domain.JobStatusNoData, err.Error())
	}
	s.setSymbolPrice(ctx, symbol, input.Bars[len(input.Bars)-1])
	previous := s.historySnapshot()

	queue := analysis.NewQueue(s.agentClient, 1)
	results := queue.RunBatch(ctx, []domain.AgentInput{input}, previous)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyJobResultsLocked(results)
	if err := s.persistJobResultsLocked(results); err != nil {
		return domain.AppState{}, err
	}
	s.emitLocked(ctx, "analysis:update", s.state)
	if len(results) > 0 && results[0].Error != "" {
		return s.state, errors.New(results[0].Error)
	}
	return s.state, nil
}

func (s *Service) buildAgentInput(ctx context.Context, symbol string, settings domain.Settings, chartImage *domain.ChartImageInput) (domain.AgentInput, error) {
	bars, err := s.provider.HistoricalBars(ctx, symbol, settings.SelectedTimeframe, 120)
	if err != nil {
		return domain.AgentInput{}, err
	}
	if len(bars) == 0 {
		return domain.AgentInput{}, fmt.Errorf("No Data")
	}
	derived := analysis.DeriveFeatures(bars)
	input := domain.AgentInput{
		Symbol:                symbol,
		Timeframe:             settings.SelectedTimeframe,
		CurrentPrice:          bars[len(bars)-1].Close,
		Bars:                  bars,
		Derived:               derived,
		MultiTimeframeContext: s.buildMultiTimeframeContext(ctx, symbol, settings.SelectedTimeframe, bars),
		ChartImage:            chartImage,
	}
	if err := ctx.Err(); err != nil {
		return domain.AgentInput{}, err
	}
	return input, nil
}

func (s *Service) buildMultiTimeframeContext(ctx context.Context, symbol string, primaryTimeframe domain.Timeframe, primaryBars []domain.Bar) []domain.TimeframeContext {
	timeframes := []domain.Timeframe{domain.Timeframe15m, domain.Timeframe1h}
	contexts := make([]domain.TimeframeContext, 0, len(timeframes))
	for _, timeframe := range timeframes {
		bars := primaryBars
		if timeframe != primaryTimeframe {
			var err error
			bars, err = s.provider.HistoricalBars(ctx, symbol, timeframe, 120)
			if err != nil {
				contexts = append(contexts, domain.TimeframeContext{
					Timeframe: timeframe,
					Available: false,
					Error:     err.Error(),
				})
				continue
			}
		}
		if len(bars) == 0 {
			contexts = append(contexts, domain.TimeframeContext{
				Timeframe: timeframe,
				Available: false,
				Error:     "No Data",
			})
			continue
		}
		derived := analysis.DeriveFeatures(bars)
		currentPrice := bars[len(bars)-1].Close
		contexts = append(contexts, domain.TimeframeContext{
			Timeframe:    timeframe,
			Available:    true,
			CurrentPrice: &currentPrice,
			Bars:         bars,
			Derived:      &derived,
		})
	}
	return contexts
}

func (s *Service) runScheduledScreenshotAnalysis(ctx context.Context, boundary time.Time) (domain.AppState, error) {
	s.mu.Lock()
	settings := s.state.Settings
	connected := s.state.ConnectionStatus == domain.ConnectionConnected
	active := s.autoEnabled
	s.mu.Unlock()

	if !active || ctx.Err() != nil {
		return s.GetState(ctx)
	}

	jobs := s.scheduler.OnClosedBar(scheduler.BatchRequest{
		Connected: connected,
		Watchlist: settings.Watchlist,
		Timeframe: settings.SelectedTimeframe,
		ClosedAt:  boundary,
		Active:    active,
	})
	if len(jobs) == 0 {
		return s.GetState(ctx)
	}

	var state domain.AppState
	var firstErr error
	for _, job := range jobs {
		if ctx.Err() != nil || !s.scheduledAnalysisEnabled() {
			s.scheduler.ClearPending(settings.SelectedTimeframe)
			return s.GetState(ctx)
		}
		nextState, err := s.RunScreenshotAnalysis(ctx, job.Symbol)
		state = nextState
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	s.scheduler.ClearPending(settings.SelectedTimeframe)
	return state, firstErr
}

func (s *Service) scheduledAnalysisEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.autoEnabled
}

func (s *Service) GetAnalysisHistory(ctx context.Context, query domain.AnalysisHistoryQuery) (domain.AnalysisHistoryPage, error) {
	if err := ctx.Err(); err != nil {
		return domain.AnalysisHistoryPage{}, err
	}
	return s.store.QueryAnalysisHistory(query)
}

func (s *Service) ExportAnalysisHistoryCSV(ctx context.Context, query domain.AnalysisHistoryQuery, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return s.store.WriteAnalysisHistoryCSV(file, query)
}

func (s *Service) RunBacktest(ctx context.Context, request domain.BacktestRequest) (domain.BacktestReport, error) {
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	request.Date = strings.TrimSpace(request.Date)
	request.StartDate = strings.TrimSpace(request.StartDate)
	request.EndDate = strings.TrimSpace(request.EndDate)
	request.StartTime = strings.TrimSpace(request.StartTime)
	request.EndTime = strings.TrimSpace(request.EndTime)
	if request.StartDate == "" {
		request.StartDate = request.Date
	}
	if request.EndDate == "" {
		request.EndDate = request.StartDate
	}
	if request.Date == "" {
		request.Date = request.StartDate
	}
	request.Timeframe = domain.Timeframe5m
	if request.ShareQuantity <= 0 {
		request.ShareQuantity = 100
	}
	if request.Symbol == "" {
		return domain.BacktestReport{}, fmt.Errorf("symbol is required")
	}
	if request.StartDate == "" {
		return domain.BacktestReport{}, fmt.Errorf("date is required")
	}
	if err := domain.ValidateBacktestCosts(request.SlippagePerShare, request.CommissionPerOrder); err != nil {
		return domain.BacktestReport{}, err
	}

	s.mu.Lock()
	connected := s.state.ConnectionStatus == domain.ConnectionConnected
	watchlist := append([]string(nil), s.state.Settings.Watchlist...)
	s.mu.Unlock()
	if !connected {
		return domain.BacktestReport{}, fmt.Errorf("IBKR is not connected")
	}
	if !containsSymbol(watchlist, request.Symbol) {
		return domain.BacktestReport{}, fmt.Errorf("%s is not in the watchlist", request.Symbol)
	}

	window, err := backtestRangeWindow(request.StartDate, request.EndDate, request.StartTime, request.EndTime)
	if err != nil {
		return domain.BacktestReport{}, err
	}
	request.StartDate = window.startDate
	request.EndDate = window.endDate
	request.Date = request.StartDate
	request.StartTime = window.startClock
	request.EndTime = window.endClock

	s.emitBacktestProgress(ctx, domain.BacktestProgress{
		Symbol:  strings.ToUpper(strings.TrimSpace(request.Symbol)),
		Stage:   domain.BacktestProgressFetching,
		Message: "Fetching historical bars from IBKR",
	})
	bars, err := s.provider.HistoricalBarsRange(ctx, request.Symbol, domain.Timeframe5m, window.startUTC, window.endUTC)
	if err != nil {
		return domain.BacktestReport{}, err
	}
	bars = filterBacktestBarsByDailyWindow(bars, window)
	if len(bars) == 0 {
		return domain.BacktestReport{}, fmt.Errorf("No Data for %s %s %s-%s US/Eastern from IBKR", request.Symbol, backtestDateLabel(request.StartDate, request.EndDate), request.StartTime, request.EndTime)
	}
	s.emitBacktestProgress(ctx, domain.BacktestProgress{
		Symbol:        strings.ToUpper(strings.TrimSpace(request.Symbol)),
		Stage:         domain.BacktestProgressAnalyzing,
		ProcessedBars: 0,
		TotalBars:     len(bars),
		Message:       fmt.Sprintf("Preparing to analyze %d bars", len(bars)),
	})
	engine := backtest.NewEngine(s.agentClient)
	engine.SetProgressReporter(func(progress domain.BacktestProgress) {
		s.emitBacktestProgress(ctx, progress)
	})
	return engine.Run(ctx, request, bars)
}

func (s *Service) startScheduledAnalysis(ctx context.Context) {
	s.stopScheduledAnalysis()

	loopCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	if !s.autoEnabled {
		s.mu.Unlock()
		cancel()
		return
	}
	s.autoCancel = cancel
	s.autoWG.Add(1)
	s.mu.Unlock()

	go func() {
		defer s.autoWG.Done()
		s.scheduledAnalysisLoop(loopCtx)
	}()
}

func (s *Service) stopScheduledAnalysis() {
	s.mu.Lock()
	cancel := s.autoCancel
	s.autoCancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		s.autoWG.Wait()
	}
}

func (s *Service) scheduledAnalysisLoop(ctx context.Context) {
	for {
		s.mu.Lock()
		timeframe := s.state.Settings.SelectedTimeframe
		lead := s.autoLead
		s.mu.Unlock()

		runAt, boundary := scheduler.NextBoundaryRun(time.Now().UTC(), timeframe, lead)
		delay := time.Until(runAt)
		if delay < 0 {
			delay = 0
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
		}

		_, _ = s.runScheduledScreenshotAnalysis(ctx, boundary)
	}
}

func (s *Service) setSymbolErrorAndState(ctx context.Context, symbol string, status domain.JobStatus, message string) (domain.AppState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markSymbol(symbol, status, message)
	s.emitLocked(ctx, "analysis:error", s.state)
	return s.state, errors.New(message)
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

func (s *Service) markSymbol(symbol string, status domain.JobStatus, message string) {
	for i := range s.state.Symbols {
		if s.state.Symbols[i].Symbol == symbol {
			s.state.Symbols[i].JobStatus = status
			s.state.Symbols[i].Error = message
		}
	}
}

func (s *Service) applyJobResultsLocked(results []analysis.JobResult) {
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
}

func (s *Service) historySnapshot() map[string]domain.AnalysisResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := make(map[string]domain.AnalysisResult, len(s.history))
	for symbol, result := range s.history {
		previous[symbol] = result
	}
	return previous
}

func (s *Service) persistLocked() error {
	return s.store.SaveSettings(s.state.Settings)
}

func (s *Service) persistJobResultsLocked(results []analysis.JobResult) error {
	for _, job := range results {
		if job.Status != domain.JobStatusComplete || job.Result == nil || job.Result.Stale {
			continue
		}
		if _, err := s.store.AppendAnalysisResult(*job.Result); err != nil {
			return err
		}
	}
	return nil
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

func (s *Service) emitBacktestProgress(ctx context.Context, progress domain.BacktestProgress) {
	if s.emit != nil {
		s.emit(ctx, "backtest:progress", progress)
	}
}

func containsSymbol(symbols []string, target string) bool {
	for _, symbol := range symbols {
		if strings.EqualFold(strings.TrimSpace(symbol), target) {
			return true
		}
	}
	return false
}

func regularSessionWindow(date string) (time.Time, time.Time, error) {
	return backtestWindow(date, "09:30", "16:00")
}

type backtestRange struct {
	startDate      string
	endDate        string
	startClock     string
	endClock       string
	startUTC       time.Time
	endUTC         time.Time
	location       *time.Location
	startDay       time.Time
	endDay         time.Time
	startClockTime time.Time
	endClockTime   time.Time
}

func backtestRangeWindow(startDate string, endDate string, startClock string, endClock string) (backtestRange, error) {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return backtestRange{}, err
	}
	startDay, err := parseBacktestDate(startDate, "start_date", location)
	if err != nil {
		return backtestRange{}, err
	}
	endDay, err := parseBacktestDate(endDate, "end_date", location)
	if err != nil {
		return backtestRange{}, err
	}
	if endDay.Before(startDay) {
		return backtestRange{}, fmt.Errorf("end_date must be on or after start_date")
	}
	startTime, err := parseBacktestClock(startClock, "09:30", "start_time")
	if err != nil {
		return backtestRange{}, err
	}
	endTime, err := parseBacktestClock(endClock, "16:00", "end_time")
	if err != nil {
		return backtestRange{}, err
	}
	start := dateWithClock(startDay, startTime, location)
	end := dateWithClock(startDay, endTime, location)
	sessionStart := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 9, 30, 0, 0, location)
	sessionEnd := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 16, 0, 0, 0, location)
	if !start.Before(end) {
		return backtestRange{}, fmt.Errorf("start_time must be before end_time")
	}
	if start.Before(sessionStart) || end.After(sessionEnd) {
		return backtestRange{}, fmt.Errorf("backtest time window must be within 09:30-16:00 US/Eastern")
	}
	return backtestRange{
		startDate:      startDay.Format("2006-01-02"),
		endDate:        endDay.Format("2006-01-02"),
		startClock:     startTime.Format("15:04"),
		endClock:       endTime.Format("15:04"),
		startUTC:       dateWithClock(startDay, startTime, location).UTC(),
		endUTC:         dateWithClock(endDay, endTime, location).UTC(),
		location:       location,
		startDay:       startDay,
		endDay:         endDay,
		startClockTime: startTime,
		endClockTime:   endTime,
	}, nil
}

func parseBacktestDate(value string, field string, location *time.Location) (time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), location)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be YYYY-MM-DD", field)
	}
	return day, nil
}

func dateWithClock(day time.Time, clock time.Time, location *time.Location) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, location)
}

func filterBacktestBarsByDailyWindow(bars []domain.Bar, window backtestRange) []domain.Bar {
	filtered := make([]domain.Bar, 0, len(bars))
	for _, bar := range bars {
		local := bar.Time.In(window.location)
		day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, window.location)
		if day.Before(window.startDay) || day.After(window.endDay) {
			continue
		}
		start := dateWithClock(day, window.startClockTime, window.location).UTC()
		end := dateWithClock(day, window.endClockTime, window.location).UTC()
		if bar.Time.Before(start) || !bar.Time.Before(end) {
			continue
		}
		filtered = append(filtered, bar)
	}
	return filtered
}

func backtestDateLabel(startDate string, endDate string) string {
	if startDate == endDate {
		return startDate
	}
	return startDate + "-" + endDate
}

func backtestWindow(date string, startClock string, endClock string) (time.Time, time.Time, error) {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	day, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(date), location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("date must be YYYY-MM-DD")
	}
	startTime, err := parseBacktestClock(startClock, "09:30", "start_time")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endTime, err := parseBacktestClock(endClock, "16:00", "end_time")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start := time.Date(day.Year(), day.Month(), day.Day(), startTime.Hour(), startTime.Minute(), 0, 0, location)
	end := time.Date(day.Year(), day.Month(), day.Day(), endTime.Hour(), endTime.Minute(), 0, 0, location)
	sessionStart := time.Date(day.Year(), day.Month(), day.Day(), 9, 30, 0, 0, location)
	sessionEnd := time.Date(day.Year(), day.Month(), day.Day(), 16, 0, 0, 0, location)
	if !start.Before(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("start_time must be before end_time")
	}
	if start.Before(sessionStart) || end.After(sessionEnd) {
		return time.Time{}, time.Time{}, fmt.Errorf("backtest time window must be within 09:30-16:00 US/Eastern")
	}
	return start.UTC(), end.UTC(), nil
}

func parseBacktestClock(value string, fallback string, field string) (time.Time, error) {
	clock := strings.TrimSpace(value)
	if clock == "" {
		clock = fallback
	}
	parsed, err := time.Parse("15:04", clock)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be HH:MM", field)
	}
	return parsed, nil
}

func captureTarget(window *domain.ChartWindow) capture.WindowTarget {
	if window == nil {
		return capture.WindowTarget{}
	}
	return capture.WindowTarget{
		ID:      window.ID,
		AppName: window.AppName,
		Title:   window.Title,
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
