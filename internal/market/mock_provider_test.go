package market

import (
	"context"
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func TestMockProviderConnectionStateTransitions(t *testing.T) {
	provider := NewMockProvider()

	if provider.State().Status != domain.ConnectionDisconnected {
		t.Fatalf("initial state = %q, want disconnected", provider.State().Status)
	}
	if err := provider.Connect(context.Background(), ConnectionSettings{Host: "127.0.0.1", Port: 7497, ClientID: 1001}); err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
	if provider.State().Status != domain.ConnectionConnected {
		t.Fatalf("connected state = %q, want connected", provider.State().Status)
	}
	if err := provider.Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect returned error: %v", err)
	}
	if provider.State().Status != domain.ConnectionDisconnected {
		t.Fatalf("disconnected state = %q, want disconnected", provider.State().Status)
	}
}

func TestMockProviderHistoricalBars(t *testing.T) {
	provider := NewMockProvider()
	now := time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC)
	bar := domain.Bar{Time: now, Open: 10, High: 11, Low: 9, Close: 10.5, Volume: 100}
	provider.SetHistoricalBars("nvda", domain.Timeframe5m, []domain.Bar{bar})

	bars, err := provider.HistoricalBars(context.Background(), "NVDA", domain.Timeframe5m, 10)
	if err != nil {
		t.Fatalf("HistoricalBars returned error: %v", err)
	}
	if len(bars) != 1 || !bars[0].Time.Equal(now) {
		t.Fatalf("bars = %#v, want seeded bar", bars)
	}
}

func TestMockProviderHistoricalBarsRangeFiltersByTime(t *testing.T) {
	provider := NewMockProvider()
	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	provider.SetHistoricalBars("nvda", domain.Timeframe5m, []domain.Bar{
		{Time: start.Add(-5 * time.Minute), Close: 99},
		{Time: start, Close: 100},
		{Time: start.Add(5 * time.Minute), Close: 101},
		{Time: start.Add(390 * time.Minute), Close: 102},
	})

	bars, err := provider.HistoricalBarsRange(context.Background(), "NVDA", domain.Timeframe5m, start, start.Add(390*time.Minute))
	if err != nil {
		t.Fatalf("HistoricalBarsRange returned error: %v", err)
	}

	if len(bars) != 2 || bars[0].Close != 100 || bars[1].Close != 101 {
		t.Fatalf("bars = %#v, want only bars inside [start,end)", bars)
	}
}

func TestMockProviderReturnsNoDataForMissingSymbol(t *testing.T) {
	provider := NewMockProvider()

	_, err := provider.HistoricalBars(context.Background(), "MISSING", domain.Timeframe5m, 10)
	if err == nil {
		t.Fatal("expected missing data error")
	}
}

func TestMockProviderSubscribeEmitsBars(t *testing.T) {
	provider := NewMockProvider()
	updates, unsubscribe, err := provider.Subscribe(context.Background(), "AAPL", domain.Timeframe1m)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer unsubscribe()

	bar := domain.Bar{Time: time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC), Close: 200}
	provider.Emit("aapl", domain.Timeframe1m, bar)

	select {
	case got := <-updates:
		if got.Symbol != "AAPL" || got.Bar.Close != 200 {
			t.Fatalf("update = %#v, want AAPL close 200", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for update")
	}
}
