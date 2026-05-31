package market

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ibkr-stock-analysis/internal/domain"

	"github.com/scmhub/ibapi"
)

type IBKRProvider struct {
	mu      sync.Mutex
	state   ProviderState
	wrapper *ibkrWrapper
	client  *ibapi.EClient
	nextID  atomic.Int64
}

func NewIBKRProvider() *IBKRProvider {
	wrapper := newIBKRWrapper()
	provider := &IBKRProvider{
		state: ProviderState{
			Status:    domain.ConnectionDisconnected,
			UpdatedAt: time.Now().UTC(),
		},
		wrapper: wrapper,
	}
	provider.client = ibapi.NewEClient(wrapper)
	provider.nextID.Store(1000)
	return provider
}

func (p *IBKRProvider) Connect(ctx context.Context, settings ConnectionSettings) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.setState(domain.ConnectionConnecting, "")
	if err := p.client.Connect(settings.Host, settings.Port, int64(settings.ClientID)); err != nil {
		p.setState(domain.ConnectionFailed, err.Error())
		return err
	}
	p.setState(domain.ConnectionConnected, "")
	return nil
}

func (p *IBKRProvider) Disconnect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.client != nil && p.client.IsConnected() {
		if err := p.client.Disconnect(); err != nil {
			p.setState(domain.ConnectionFailed, err.Error())
			return err
		}
	}
	p.setState(domain.ConnectionDisconnected, "")
	return nil
}

func (p *IBKRProvider) State() ProviderState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *IBKRProvider) Subscribe(ctx context.Context, symbol string, timeframe domain.Timeframe) (<-chan BarUpdate, Unsubscribe, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if !p.client.IsConnected() {
		return nil, nil, fmt.Errorf("ibkr is not connected")
	}
	reqID := p.nextID.Add(1)
	updates := make(chan BarUpdate, 16)
	p.wrapper.addRealtime(reqID, strings.ToUpper(strings.TrimSpace(symbol)), timeframe, updates)
	p.client.ReqRealTimeBars(reqID, usStockContract(symbol), 5, "TRADES", true, nil)

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			p.client.CancelRealTimeBars(reqID)
			p.wrapper.removeRealtime(reqID)
			close(updates)
		})
	}
	return updates, unsubscribe, nil
}

func (p *IBKRProvider) HistoricalBars(ctx context.Context, symbol string, timeframe domain.Timeframe, limit int) ([]domain.Bar, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !p.client.IsConnected() {
		return nil, fmt.Errorf("ibkr is not connected")
	}
	reqID := p.nextID.Add(1)
	result := p.wrapper.addHistorical(reqID)
	params := historicalRequestParameters(timeframe, limit)
	p.client.ReqHistoricalData(reqID, usStockContract(symbol), "", params.Duration, params.BarSize, params.WhatToShow, true, 2, false, nil)

	bars, err := p.waitHistoricalBars(ctx, reqID, result)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(bars) > limit {
		bars = bars[len(bars)-limit:]
	}
	return bars, nil
}

func (p *IBKRProvider) HistoricalBarsRange(ctx context.Context, symbol string, timeframe domain.Timeframe, start time.Time, end time.Time) ([]domain.Bar, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !p.client.IsConnected() {
		return nil, fmt.Errorf("ibkr is not connected")
	}
	reqID := p.nextID.Add(1)
	result := p.wrapper.addHistorical(reqID)
	params := historicalRangeRequestParameters(timeframe, start, end)
	p.client.ReqHistoricalData(reqID, usStockContract(symbol), params.EndDateTime, params.Duration, params.BarSize, params.WhatToShow, true, 2, false, nil)

	bars, err := p.waitHistoricalBars(ctx, reqID, result)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.Bar, 0, len(bars))
	for _, bar := range bars {
		if bar.Time.Before(start) || !bar.Time.Before(end) {
			continue
		}
		filtered = append(filtered, bar)
	}
	return filtered, nil
}

func (p *IBKRProvider) waitHistoricalBars(ctx context.Context, reqID int64, result historicalResult) ([]domain.Bar, error) {
	select {
	case bars := <-result.bars:
		return bars, nil
	case err := <-result.errs:
		return nil, err
	case <-ctx.Done():
		p.client.CancelHistoricalData(reqID)
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		p.client.CancelHistoricalData(reqID)
		return nil, fmt.Errorf("timed out waiting for IBKR historical bars")
	}
}

func (p *IBKRProvider) setState(status domain.ConnectionStatus, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = ProviderState{Status: status, Message: message, UpdatedAt: time.Now().UTC()}
}

type historicalParams struct {
	Duration   string
	BarSize    string
	WhatToShow string
}

type historicalRangeParams struct {
	EndDateTime string
	Duration    string
	BarSize     string
	WhatToShow  string
}

func historicalRequestParameters(timeframe domain.Timeframe, limit int) historicalParams {
	if limit <= 0 {
		limit = 100
	}
	duration := "2 D"
	switch timeframe {
	case domain.Timeframe15m:
		duration = "1 W"
	case domain.Timeframe1h:
		duration = "1 M"
	}
	if limit > 300 {
		duration = "1 M"
	}
	return historicalParams{
		Duration:   duration,
		BarSize:    ibkrBarSize(timeframe),
		WhatToShow: "TRADES",
	}
}

func historicalRangeRequestParameters(timeframe domain.Timeframe, start time.Time, end time.Time) historicalRangeParams {
	duration := "1 D"
	if end.After(start) {
		days := int(math.Ceil(end.Sub(start).Hours() / 24))
		if days < 1 {
			days = 1
		}
		duration = fmt.Sprintf("%d D", days)
	}
	return historicalRangeParams{
		EndDateTime: formatIBKREndDateTime(end),
		Duration:    duration,
		BarSize:     ibkrBarSize(timeframe),
		WhatToShow:  "TRADES",
	}
}

func formatIBKREndDateTime(t time.Time) string {
	return t.UTC().Format("20060102 15:04:05 UTC")
}

func ibkrBarSize(timeframe domain.Timeframe) string {
	switch timeframe {
	case domain.Timeframe1m:
		return "1 min"
	case domain.Timeframe5m:
		return "5 mins"
	case domain.Timeframe15m:
		return "15 mins"
	case domain.Timeframe1h:
		return "1 hour"
	default:
		return "5 mins"
	}
}

func usStockContract(symbol string) *ibapi.Contract {
	contract := ibapi.USStockAtSmart()
	contract.Symbol = strings.ToUpper(strings.TrimSpace(symbol))
	contract.Currency = "USD"
	contract.SecType = "STK"
	contract.Exchange = "SMART"
	return contract
}

type historicalResult struct {
	bars chan []domain.Bar
	errs chan error
}

type realtimeSubscription struct {
	symbol    string
	timeframe domain.Timeframe
	updates   chan BarUpdate
}

type ibkrWrapper struct {
	ibapi.Wrapper
	mu         sync.Mutex
	historical map[int64][]domain.Bar
	results    map[int64]historicalResult
	realtime   map[int64]realtimeSubscription
}

func newIBKRWrapper() *ibkrWrapper {
	return &ibkrWrapper{
		historical: make(map[int64][]domain.Bar),
		results:    make(map[int64]historicalResult),
		realtime:   make(map[int64]realtimeSubscription),
	}
}

func (w *ibkrWrapper) addHistorical(reqID int64) historicalResult {
	result := historicalResult{
		bars: make(chan []domain.Bar, 1),
		errs: make(chan error, 1),
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.results[reqID] = result
	w.historical[reqID] = []domain.Bar{}
	return result
}

func (w *ibkrWrapper) HistoricalData(reqID int64, bar *ibapi.Bar) {
	converted, err := convertIBKRBar(bar)
	w.mu.Lock()
	defer w.mu.Unlock()
	if err != nil {
		if result, ok := w.results[reqID]; ok {
			result.errs <- err
		}
		return
	}
	w.historical[reqID] = append(w.historical[reqID], converted)
}

func (w *ibkrWrapper) HistoricalDataEnd(reqID int64, startDateStr string, endDateStr string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	result, ok := w.results[reqID]
	if !ok {
		return
	}
	bars := append([]domain.Bar(nil), w.historical[reqID]...)
	delete(w.results, reqID)
	delete(w.historical, reqID)
	result.bars <- bars
}

func (w *ibkrWrapper) Error(reqID int64, errorTime int64, errCode int64, errString string, advancedOrderRejectJson string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if result, ok := w.results[reqID]; ok {
		result.errs <- fmt.Errorf("ibkr error %d: %s", errCode, errString)
	}
}

func (w *ibkrWrapper) addRealtime(reqID int64, symbol string, timeframe domain.Timeframe, updates chan BarUpdate) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.realtime[reqID] = realtimeSubscription{symbol: symbol, timeframe: timeframe, updates: updates}
}

func (w *ibkrWrapper) removeRealtime(reqID int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.realtime, reqID)
}

func (w *ibkrWrapper) RealtimeBar(reqID int64, unixTime int64, open float64, high float64, low float64, close float64, volume ibapi.Decimal, wap ibapi.Decimal, count int64) {
	w.mu.Lock()
	subscription, ok := w.realtime[reqID]
	w.mu.Unlock()
	if !ok {
		return
	}
	update := BarUpdate{
		Symbol:    subscription.symbol,
		Timeframe: subscription.timeframe,
		Closed:    true,
		Bar: domain.Bar{
			Time:   time.Unix(unixTime, 0).UTC(),
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: decimalToInt64(volume),
		},
	}
	select {
	case subscription.updates <- update:
	default:
	}
}

func convertIBKRBar(bar *ibapi.Bar) (domain.Bar, error) {
	if bar == nil {
		return domain.Bar{}, fmt.Errorf("nil IBKR bar")
	}
	t, err := parseIBKRBarDate(bar.Date)
	if err != nil {
		return domain.Bar{}, err
	}
	return domain.Bar{
		Time:   t,
		Open:   bar.Open,
		High:   bar.High,
		Low:    bar.Low,
		Close:  bar.Close,
		Volume: decimalToInt64(bar.Volume),
	}, nil
}

func parseIBKRBarDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty IBKR bar date")
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(seconds, 0).UTC(), nil
	}
	layouts := []string{"20060102 15:04:05", "20060102"}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported IBKR bar date %q", value)
}

func decimalToInt64(value ibapi.Decimal) int64 {
	text := value.String()
	if i, err := strconv.ParseInt(text, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(text, 64); err == nil {
		return int64(f)
	}
	return 0
}
