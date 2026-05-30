package scheduler

import (
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
	"ibkr-stock-analysis/internal/market"
)

func TestSchedulerDoesNotEnqueueWhenDisconnected(t *testing.T) {
	s := New()
	closedAt := time.Date(2026, 5, 30, 18, 35, 0, 0, time.UTC)

	jobs := s.OnClosedBar(BatchRequest{
		Connected: false,
		Watchlist: []string{"NVDA", "AAPL"},
		Timeframe: domain.Timeframe5m,
		ClosedAt:  closedAt,
	})

	if len(jobs) != 0 {
		t.Fatalf("jobs = %#v, want none", jobs)
	}
}

func TestSchedulerEnqueuesOneJobPerSymbol(t *testing.T) {
	s := New()
	closedAt := time.Date(2026, 5, 30, 18, 35, 0, 0, time.UTC)

	jobs := s.OnClosedBar(BatchRequest{
		Connected: true,
		Watchlist: []string{"nvda", "AAPL", "NVDA"},
		Timeframe: domain.Timeframe5m,
		ClosedAt:  closedAt,
	})

	if len(jobs) != 2 {
		t.Fatalf("jobs = %#v, want two unique symbols", jobs)
	}
	if jobs[0].Symbol != "NVDA" || jobs[1].Symbol != "AAPL" {
		t.Fatalf("jobs = %#v, want NVDA then AAPL", jobs)
	}
	if jobs[0].Key != market.ClosedBarKey("NVDA", domain.Timeframe5m, closedAt) {
		t.Fatalf("key = %q, want closed bar key", jobs[0].Key)
	}
}

func TestSchedulerSuppressesDuplicateClosedBars(t *testing.T) {
	s := New()
	closedAt := time.Date(2026, 5, 30, 18, 35, 0, 0, time.UTC)
	request := BatchRequest{
		Connected: true,
		Watchlist: []string{"NVDA"},
		Timeframe: domain.Timeframe5m,
		ClosedAt:  closedAt,
	}

	first := s.OnClosedBar(request)
	second := s.OnClosedBar(request)

	if len(first) != 1 {
		t.Fatalf("first jobs = %#v, want one", first)
	}
	if len(second) != 0 {
		t.Fatalf("second jobs = %#v, want duplicate suppression", second)
	}
}

func TestSchedulerKeepsOnlyOnePendingBatchPerTimeframe(t *testing.T) {
	s := New()
	firstClose := time.Date(2026, 5, 30, 18, 35, 0, 0, time.UTC)
	secondClose := time.Date(2026, 5, 30, 18, 40, 0, 0, time.UTC)

	first := s.OnClosedBar(BatchRequest{
		Connected: true,
		Watchlist: []string{"NVDA"},
		Timeframe: domain.Timeframe5m,
		ClosedAt:  firstClose,
		Active:    true,
	})
	second := s.OnClosedBar(BatchRequest{
		Connected: true,
		Watchlist: []string{"AAPL"},
		Timeframe: domain.Timeframe5m,
		ClosedAt:  secondClose,
		Active:    true,
	})
	pending := s.Pending(domain.Timeframe5m)

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("first=%#v second=%#v, want both returned as latest batches", first, second)
	}
	if len(pending) != 1 || pending[0].Symbol != "AAPL" {
		t.Fatalf("pending = %#v, want only latest AAPL batch", pending)
	}
}
