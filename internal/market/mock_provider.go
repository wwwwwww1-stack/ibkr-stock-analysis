package market

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

type MockProvider struct {
	mu          sync.RWMutex
	state       ProviderState
	historical  map[string][]domain.Bar
	subscribers map[string][]chan BarUpdate
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		state: ProviderState{
			Status:    domain.ConnectionDisconnected,
			UpdatedAt: time.Now().UTC(),
		},
		historical:  make(map[string][]domain.Bar),
		subscribers: make(map[string][]chan BarUpdate),
	}
}

func (p *MockProvider) Connect(ctx context.Context, settings ConnectionSettings) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = ProviderState{Status: domain.ConnectionConnected, UpdatedAt: time.Now().UTC()}
	return nil
}

func (p *MockProvider) Disconnect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = ProviderState{Status: domain.ConnectionDisconnected, UpdatedAt: time.Now().UTC()}
	return nil
}

func (p *MockProvider) State() ProviderState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

func (p *MockProvider) Subscribe(ctx context.Context, symbol string, timeframe domain.Timeframe) (<-chan BarUpdate, Unsubscribe, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	key := subscriptionKey(symbol, timeframe)
	updates := make(chan BarUpdate, 8)
	p.mu.Lock()
	p.subscribers[key] = append(p.subscribers[key], updates)
	p.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			subscribers := p.subscribers[key]
			for i, subscriber := range subscribers {
				if subscriber == updates {
					p.subscribers[key] = append(subscribers[:i], subscribers[i+1:]...)
					break
				}
			}
			close(updates)
		})
	}

	return updates, unsubscribe, nil
}

func (p *MockProvider) HistoricalBars(ctx context.Context, symbol string, timeframe domain.Timeframe, limit int) ([]domain.Bar, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := subscriptionKey(symbol, timeframe)
	p.mu.RLock()
	defer p.mu.RUnlock()
	bars, ok := p.historical[key]
	if !ok {
		return nil, fmt.Errorf("no data for %s %s", strings.ToUpper(symbol), timeframe)
	}
	if limit > 0 && len(bars) > limit {
		bars = bars[len(bars)-limit:]
	}
	out := make([]domain.Bar, len(bars))
	copy(out, bars)
	return out, nil
}

func (p *MockProvider) HistoricalBarsRange(ctx context.Context, symbol string, timeframe domain.Timeframe, start time.Time, end time.Time) ([]domain.Bar, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bars, err := p.HistoricalBars(ctx, symbol, timeframe, 0)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Bar, 0, len(bars))
	for _, bar := range bars {
		if bar.Time.Before(start) || !bar.Time.Before(end) {
			continue
		}
		out = append(out, bar)
	}
	return out, nil
}

func (p *MockProvider) SetHistoricalBars(symbol string, timeframe domain.Timeframe, bars []domain.Bar) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := subscriptionKey(symbol, timeframe)
	p.historical[key] = append([]domain.Bar(nil), bars...)
}

func (p *MockProvider) Emit(symbol string, timeframe domain.Timeframe, bar domain.Bar) {
	update := BarUpdate{
		Symbol:    strings.ToUpper(strings.TrimSpace(symbol)),
		Timeframe: timeframe,
		Bar:       bar,
		Closed:    true,
	}
	key := subscriptionKey(symbol, timeframe)

	p.mu.RLock()
	subscribers := append([]chan BarUpdate(nil), p.subscribers[key]...)
	p.mu.RUnlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber <- update:
		default:
		}
	}
}

func subscriptionKey(symbol string, timeframe domain.Timeframe) string {
	return strings.ToUpper(strings.TrimSpace(symbol)) + "|" + timeframe.String()
}
