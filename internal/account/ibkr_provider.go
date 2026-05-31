package account

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

const (
	accountSummaryGroup = "All"
	accountSummaryTags  = "TotalCashValue,BuyingPower"
	defaultIBKRTimeout  = 15 * time.Second
)

type IBKRAccountClient interface {
	IsConnected() bool
	ReqAccountSummary(reqID int64, groupName string, tags string)
	CancelAccountSummary(reqID int64)
	ReqPositions()
	CancelPositions()
}

type IBKRProviderConfig struct {
	Client        IBKRAccountClient
	Events        *IBKREventHub
	NextRequestID func() int64
	Timeout       time.Duration
	Now           func() time.Time
}

type IBKRProvider struct {
	client        IBKRAccountClient
	events        *IBKREventHub
	nextRequestID func() int64
	timeout       time.Duration
	now           func() time.Time
}

func NewIBKRProvider(config IBKRProviderConfig) *IBKRProvider {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultIBKRTimeout
	}
	now := config.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	nextRequestID := config.NextRequestID
	if nextRequestID == nil {
		var mu sync.Mutex
		var next int64 = 10_000
		nextRequestID = func() int64 {
			mu.Lock()
			defer mu.Unlock()
			next++
			return next
		}
	}
	return &IBKRProvider{
		client:        config.Client,
		events:        config.Events,
		nextRequestID: nextRequestID,
		timeout:       timeout,
		now:           now,
	}
}

func (p *IBKRProvider) Snapshot(ctx context.Context) (domain.AccountSnapshot, error) {
	if p.client == nil || p.events == nil {
		return domain.AccountSnapshot{}, fmt.Errorf("account snapshot unavailable: IBKR account provider is not configured")
	}
	reqID := p.nextRequestID()
	summaryResult := p.events.AddAccountSummary(reqID)
	positionResult := p.events.AddPositions()
	defer p.events.RemoveAccountSummary(reqID)
	defer p.events.ClearPositions()
	defer p.client.CancelAccountSummary(reqID)
	defer p.client.CancelPositions()

	if err := ctx.Err(); err != nil {
		return domain.AccountSnapshot{}, err
	}
	if !p.client.IsConnected() {
		return domain.AccountSnapshot{}, fmt.Errorf("account snapshot unavailable: IBKR is not connected")
	}

	requestCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	p.client.ReqAccountSummary(reqID, accountSummaryGroup, accountSummaryTags)
	summary, err := waitAccountSummary(requestCtx, summaryResult)
	if err != nil {
		return domain.AccountSnapshot{}, err
	}
	if err := validateSummary(summary); err != nil {
		return domain.AccountSnapshot{}, err
	}

	p.client.ReqPositions()
	positions, notes, err := waitPositions(requestCtx, positionResult)
	if err != nil {
		return domain.AccountSnapshot{}, err
	}
	return domain.AccountSnapshot{
		AvailableCashUSD: summary.cash,
		BuyingPowerUSD:   summary.buyingPower,
		SnapshotAt:       p.now().UTC(),
		Positions:        positions,
		Notes:            notes,
	}, nil
}

type IBKRPositionEvent struct {
	Account      string
	Symbol       string
	SecurityType string
	Currency     string
	Quantity     float64
	AverageCost  float64
}

type IBKREventHub struct {
	mu          sync.Mutex
	summaries   map[int64]accountSummaryResult
	positionReq *positionResult
}

func NewIBKREventHub() *IBKREventHub {
	return &IBKREventHub{summaries: make(map[int64]accountSummaryResult)}
}

func (h *IBKREventHub) AddAccountSummary(reqID int64) accountSummaryResult {
	result := accountSummaryResult{
		events: make(chan accountSummaryEvent, 16),
		done:   make(chan struct{}, 1),
		errs:   make(chan error, 1),
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.summaries[reqID] = result
	return result
}

func (h *IBKREventHub) RemoveAccountSummary(reqID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.summaries, reqID)
}

func (h *IBKREventHub) AddPositions() positionResult {
	result := positionResult{
		events: make(chan IBKRPositionEvent, 16),
		done:   make(chan struct{}, 1),
		errs:   make(chan error, 1),
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.positionReq = &result
	return result
}

func (h *IBKREventHub) ClearPositions() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.positionReq = nil
}

func (h *IBKREventHub) AccountSummary(reqID int64, account string, tag string, value string, currency string) {
	h.mu.Lock()
	result, ok := h.summaries[reqID]
	h.mu.Unlock()
	if !ok {
		return
	}
	select {
	case result.events <- accountSummaryEvent{tag: tag, value: value, currency: currency}:
	default:
	}
}

func (h *IBKREventHub) AccountSummaryEnd(reqID int64) {
	h.mu.Lock()
	result, ok := h.summaries[reqID]
	h.mu.Unlock()
	if !ok {
		return
	}
	select {
	case result.done <- struct{}{}:
	default:
	}
}

func (h *IBKREventHub) Position(event IBKRPositionEvent) {
	h.mu.Lock()
	result := h.positionReq
	h.mu.Unlock()
	if result == nil {
		return
	}
	select {
	case result.events <- event:
	default:
	}
}

func (h *IBKREventHub) PositionEnd() {
	h.mu.Lock()
	result := h.positionReq
	h.mu.Unlock()
	if result == nil {
		return
	}
	select {
	case result.done <- struct{}{}:
	default:
	}
}

func (h *IBKREventHub) Error(reqID int64, err error) {
	if err == nil {
		return
	}
	h.mu.Lock()
	summary, hasSummary := h.summaries[reqID]
	position := h.positionReq
	h.mu.Unlock()
	if hasSummary {
		select {
		case summary.errs <- err:
		default:
		}
		return
	}
	if position != nil {
		select {
		case position.errs <- err:
		default:
		}
	}
}

type accountSummaryResult struct {
	events chan accountSummaryEvent
	done   chan struct{}
	errs   chan error
}

type accountSummaryEvent struct {
	tag      string
	value    string
	currency string
}

type positionResult struct {
	events chan IBKRPositionEvent
	done   chan struct{}
	errs   chan error
}

type accountSummary struct {
	cash           float64
	hasCash        bool
	buyingPower    float64
	hasBuyingPower bool
}

func waitAccountSummary(ctx context.Context, result accountSummaryResult) (accountSummary, error) {
	var summary accountSummary
	for {
		select {
		case event := <-result.events:
			applyAccountSummaryEvent(&summary, event)
		case <-result.done:
			drainAccountSummaryEvents(&summary, result.events)
			return summary, nil
		case err := <-result.errs:
			return accountSummary{}, fmt.Errorf("IBKR account summary error: %w", err)
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return accountSummary{}, fmt.Errorf("timed out waiting for IBKR account summary")
			}
			return accountSummary{}, ctx.Err()
		}
	}
}

func waitPositions(ctx context.Context, result positionResult) ([]domain.AccountPosition, []string, error) {
	var positions []domain.AccountPosition
	var notes []string
	for {
		select {
		case event := <-result.events:
			position, note, ok := convertPositionEvent(event)
			if note != "" {
				notes = append(notes, note)
			}
			if ok {
				positions = append(positions, position)
			}
		case <-result.done:
			drainedPositions, drainedNotes := drainPositionEvents(result.events)
			positions = append(positions, drainedPositions...)
			notes = append(notes, drainedNotes...)
			return positions, notes, nil
		case err := <-result.errs:
			return nil, nil, fmt.Errorf("IBKR position snapshot error: %w", err)
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, nil, fmt.Errorf("timed out waiting for IBKR positions")
			}
			return nil, nil, ctx.Err()
		}
	}
}

func drainAccountSummaryEvents(summary *accountSummary, events <-chan accountSummaryEvent) {
	for {
		select {
		case event := <-events:
			applyAccountSummaryEvent(summary, event)
		default:
			return
		}
	}
}

func drainPositionEvents(events <-chan IBKRPositionEvent) ([]domain.AccountPosition, []string) {
	var positions []domain.AccountPosition
	var notes []string
	for {
		select {
		case event := <-events:
			position, note, ok := convertPositionEvent(event)
			if note != "" {
				notes = append(notes, note)
			}
			if ok {
				positions = append(positions, position)
			}
		default:
			return positions, notes
		}
	}
}

func applyAccountSummaryEvent(summary *accountSummary, event accountSummaryEvent) {
	if !strings.EqualFold(strings.TrimSpace(event.currency), "USD") {
		return
	}
	value, ok := parseIBKRFloat(event.value)
	if !ok {
		return
	}
	switch strings.ToLower(strings.TrimSpace(event.tag)) {
	case "totalcashvalue":
		summary.cash = value
		summary.hasCash = true
	case "buyingpower":
		summary.buyingPower = value
		summary.hasBuyingPower = true
	}
}

func validateSummary(summary accountSummary) error {
	if !summary.hasCash {
		return fmt.Errorf("IBKR account summary missing USD available cash")
	}
	if !summary.hasBuyingPower {
		return fmt.Errorf("IBKR account summary missing USD buying power")
	}
	if !finite(summary.cash) {
		return fmt.Errorf("IBKR account summary available cash is invalid")
	}
	if !finite(summary.buyingPower) {
		return fmt.Errorf("IBKR account summary buying power is invalid")
	}
	return nil
}

func convertPositionEvent(event IBKRPositionEvent) (domain.AccountPosition, string, bool) {
	symbol := strings.ToUpper(strings.TrimSpace(event.Symbol))
	if symbol == "" {
		return domain.AccountPosition{}, "ignored account position with missing symbol", false
	}
	if !strings.EqualFold(strings.TrimSpace(event.SecurityType), "STK") || !strings.EqualFold(strings.TrimSpace(event.Currency), "USD") {
		return domain.AccountPosition{}, fmt.Sprintf("ignored non-USD stock position %s", symbol), false
	}
	averageCost := nonNegativeFinite(event.AverageCost)
	marketValue := event.Quantity * averageCost
	return domain.AccountPosition{
		Symbol:         symbol,
		Quantity:       finiteOrZero(event.Quantity),
		AverageCost:    averageCost,
		MarketPrice:    averageCost,
		MarketValueUSD: finiteOrZero(marketValue),
	}, "", true
}

func parseIBKRFloat(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || !finite(parsed) {
		return 0, false
	}
	return parsed, true
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func finiteOrZero(value float64) float64 {
	if !finite(value) {
		return 0
	}
	return value
}

func nonNegativeFinite(value float64) float64 {
	if value < 0 || !finite(value) {
		return 0
	}
	return value
}
