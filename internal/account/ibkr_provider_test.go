package account

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func TestIBKRProviderCollectsCashBuyingPowerAndUSDStockPositions(t *testing.T) {
	client := newFakeIBKRAccountClient()
	events := NewIBKREventHub()
	provider := NewIBKRProvider(IBKRProviderConfig{
		Client:        client,
		Events:        events,
		NextRequestID: sequenceIDs(7001),
		Timeout:       time.Second,
		Now:           fixedNow,
	})

	resultCh := make(chan snapshotResult, 1)
	go func() {
		snapshot, err := provider.Snapshot(context.Background())
		resultCh <- snapshotResult{snapshot: snapshot, err: err}
	}()
	reqID := client.waitAccountSummaryRequest(t)
	if reqID != 7001 {
		t.Fatalf("reqID = %d, want 7001", reqID)
	}
	if client.summaryTags != "TotalCashValue,BuyingPower" {
		t.Fatalf("summary tags = %q, want cash and buying power", client.summaryTags)
	}

	events.AccountSummary(reqID, "DU123", "TotalCashValue", "12000.50", "USD")
	events.AccountSummary(reqID, "DU123", "BuyingPower", "24001.00", "USD")
	events.AccountSummaryEnd(reqID)
	client.waitPositionsRequest(t)
	events.Position(IBKRPositionEvent{
		Account:      "DU123",
		Symbol:       "NVDA",
		SecurityType: "STK",
		Currency:     "USD",
		Quantity:     10,
		AverageCost:  100,
	})
	events.Position(IBKRPositionEvent{
		Account:      "DU123",
		Symbol:       "AAPL",
		SecurityType: "STK",
		Currency:     "USD",
		Quantity:     -5,
		AverageCost:  200,
	})
	events.PositionEnd()

	result := <-resultCh
	if result.err != nil {
		t.Fatalf("Snapshot returned error: %v", result.err)
	}
	if result.snapshot.AvailableCashUSD != 12000.50 || result.snapshot.BuyingPowerUSD != 24001 {
		t.Fatalf("snapshot cash/buying power = %.2f/%.2f", result.snapshot.AvailableCashUSD, result.snapshot.BuyingPowerUSD)
	}
	if !result.snapshot.SnapshotAt.Equal(fixedNow()) {
		t.Fatalf("snapshot time = %s, want fixed now", result.snapshot.SnapshotAt)
	}
	if len(result.snapshot.Positions) != 2 {
		t.Fatalf("positions = %#v, want two USD stocks", result.snapshot.Positions)
	}
	if got := result.snapshot.Positions[0]; got.Symbol != "NVDA" || got.Quantity != 10 || got.MarketValueUSD != 1000 {
		t.Fatalf("first position = %#v, want NVDA quantity 10 market value 1000", got)
	}
	if got := result.snapshot.Positions[1]; got.Symbol != "AAPL" || got.Quantity != -5 || got.MarketValueUSD != -1000 {
		t.Fatalf("second position = %#v, want AAPL short market value -1000", got)
	}
	if !client.canceledSummary || !client.canceledPositions {
		t.Fatalf("client cancellation summary=%v positions=%v, want both canceled", client.canceledSummary, client.canceledPositions)
	}
}

func TestIBKRProviderFiltersNonStockPositionsWithScopedNote(t *testing.T) {
	client := newFakeIBKRAccountClient()
	events := NewIBKREventHub()
	provider := NewIBKRProvider(IBKRProviderConfig{
		Client:        client,
		Events:        events,
		NextRequestID: sequenceIDs(8001),
		Timeout:       time.Second,
		Now:           fixedNow,
	})

	resultCh := make(chan snapshotResult, 1)
	go func() {
		snapshot, err := provider.Snapshot(context.Background())
		resultCh <- snapshotResult{snapshot: snapshot, err: err}
	}()
	reqID := client.waitAccountSummaryRequest(t)
	events.AccountSummary(reqID, "DU123", "TotalCashValue", "5000", "USD")
	events.AccountSummary(reqID, "DU123", "BuyingPower", "9000", "USD")
	events.AccountSummaryEnd(reqID)
	client.waitPositionsRequest(t)
	events.Position(IBKRPositionEvent{Symbol: "ES", SecurityType: "FUT", Currency: "USD", Quantity: 1, AverageCost: 5000})
	events.Position(IBKRPositionEvent{Symbol: "SAP", SecurityType: "STK", Currency: "EUR", Quantity: 1, AverageCost: 100})
	events.Position(IBKRPositionEvent{Symbol: "MSFT", SecurityType: "STK", Currency: "USD", Quantity: 3, AverageCost: 300})
	events.PositionEnd()

	result := <-resultCh
	if result.err != nil {
		t.Fatalf("Snapshot returned error: %v", result.err)
	}
	if len(result.snapshot.Positions) != 1 || result.snapshot.Positions[0].Symbol != "MSFT" {
		t.Fatalf("positions = %#v, want only MSFT", result.snapshot.Positions)
	}
	if len(result.snapshot.Notes) != 2 {
		t.Fatalf("notes = %#v, want scoped notes for two ignored positions", result.snapshot.Notes)
	}
}

func TestIBKRProviderTimeoutCancelsSubscriptions(t *testing.T) {
	client := newFakeIBKRAccountClient()
	events := NewIBKREventHub()
	provider := NewIBKRProvider(IBKRProviderConfig{
		Client:        client,
		Events:        events,
		NextRequestID: sequenceIDs(9001),
		Timeout:       5 * time.Millisecond,
		Now:           fixedNow,
	})

	_, err := provider.Snapshot(context.Background())
	if err == nil || !strings.Contains(err.Error(), "timed out waiting for IBKR account summary") {
		t.Fatalf("err = %v, want account summary timeout", err)
	}
	if !client.canceledSummary || !client.canceledPositions {
		t.Fatalf("client cancellation summary=%v positions=%v, want both canceled", client.canceledSummary, client.canceledPositions)
	}
}

func TestIBKRProviderContextCancellationCancelsSubscriptions(t *testing.T) {
	client := newFakeIBKRAccountClient()
	events := NewIBKREventHub()
	provider := NewIBKRProvider(IBKRProviderConfig{
		Client:        client,
		Events:        events,
		NextRequestID: sequenceIDs(9101),
		Timeout:       time.Second,
		Now:           fixedNow,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.Snapshot(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context canceled", err)
	}
	if !client.canceledSummary || !client.canceledPositions {
		t.Fatalf("client cancellation summary=%v positions=%v, want both canceled", client.canceledSummary, client.canceledPositions)
	}
}

func TestIBKRProviderReturnsScopedErrorForCallbackError(t *testing.T) {
	client := newFakeIBKRAccountClient()
	events := NewIBKREventHub()
	provider := NewIBKRProvider(IBKRProviderConfig{
		Client:        client,
		Events:        events,
		NextRequestID: sequenceIDs(9201),
		Timeout:       time.Second,
		Now:           fixedNow,
	})

	resultCh := make(chan snapshotResult, 1)
	go func() {
		snapshot, err := provider.Snapshot(context.Background())
		resultCh <- snapshotResult{snapshot: snapshot, err: err}
	}()
	reqID := client.waitAccountSummaryRequest(t)
	events.Error(reqID, errors.New("ibkr error 321: account data unavailable"))

	result := <-resultCh
	if result.err == nil || !strings.Contains(result.err.Error(), "account summary") {
		t.Fatalf("err = %v, want scoped account summary error", result.err)
	}
	if !client.canceledSummary || !client.canceledPositions {
		t.Fatalf("client cancellation summary=%v positions=%v, want both canceled", client.canceledSummary, client.canceledPositions)
	}
}

func TestIBKRProviderRejectsMissingCashOrBuyingPower(t *testing.T) {
	client := newFakeIBKRAccountClient()
	events := NewIBKREventHub()
	provider := NewIBKRProvider(IBKRProviderConfig{
		Client:        client,
		Events:        events,
		NextRequestID: sequenceIDs(9301),
		Timeout:       time.Second,
		Now:           fixedNow,
	})

	resultCh := make(chan snapshotResult, 1)
	go func() {
		snapshot, err := provider.Snapshot(context.Background())
		resultCh <- snapshotResult{snapshot: snapshot, err: err}
	}()
	reqID := client.waitAccountSummaryRequest(t)
	events.AccountSummary(reqID, "DU123", "TotalCashValue", "5000", "USD")
	events.AccountSummaryEnd(reqID)

	result := <-resultCh
	if result.err == nil || !strings.Contains(result.err.Error(), "buying power") {
		t.Fatalf("err = %v, want missing buying power error", result.err)
	}
}

type snapshotResult struct {
	snapshot domain.AccountSnapshot
	err      error
}

type fakeIBKRAccountClient struct {
	mu                 sync.Mutex
	connected          bool
	summaryRequested   chan int64
	positionsRequested chan struct{}
	summaryTags        string
	canceledSummary    bool
	canceledPositions  bool
}

func newFakeIBKRAccountClient() *fakeIBKRAccountClient {
	return &fakeIBKRAccountClient{
		connected:          true,
		summaryRequested:   make(chan int64, 1),
		positionsRequested: make(chan struct{}, 1),
	}
}

func (c *fakeIBKRAccountClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *fakeIBKRAccountClient) ReqAccountSummary(reqID int64, groupName string, tags string) {
	c.mu.Lock()
	c.summaryTags = tags
	c.mu.Unlock()
	c.summaryRequested <- reqID
}

func (c *fakeIBKRAccountClient) CancelAccountSummary(reqID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.canceledSummary = true
}

func (c *fakeIBKRAccountClient) ReqPositions() {
	c.positionsRequested <- struct{}{}
}

func (c *fakeIBKRAccountClient) CancelPositions() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.canceledPositions = true
}

func (c *fakeIBKRAccountClient) waitAccountSummaryRequest(t *testing.T) int64 {
	t.Helper()
	select {
	case reqID := <-c.summaryRequested:
		return reqID
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for account summary request")
		return 0
	}
}

func (c *fakeIBKRAccountClient) waitPositionsRequest(t *testing.T) {
	t.Helper()
	select {
	case <-c.positionsRequested:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for positions request")
	}
}

func sequenceIDs(first int64) func() int64 {
	next := first - 1
	return func() int64 {
		next++
		return next
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 31, 12, 45, 0, 0, time.UTC)
}
