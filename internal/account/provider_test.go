package account

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func TestSnapshotProviderExposesOnlySnapshotRead(t *testing.T) {
	providerType := reflect.TypeOf((*SnapshotProvider)(nil)).Elem()
	if providerType.NumMethod() != 1 {
		t.Fatalf("SnapshotProvider methods = %d, want 1", providerType.NumMethod())
	}
	method := providerType.Method(0)
	if method.Name != "Snapshot" {
		t.Fatalf("SnapshotProvider method = %q, want Snapshot", method.Name)
	}
	forbidden := []string{"Order", "Trade", "Place", "Cancel", "Open", strings.Join([]string{"Trans", "mit"}, ""), "Stage", "Approve", "Mutate"}
	for _, token := range forbidden {
		if strings.Contains(method.Name, token) {
			t.Fatalf("SnapshotProvider exposes forbidden concept in method %q", method.Name)
		}
	}
}

func TestUnavailableProviderReturnsScopedUnavailableError(t *testing.T) {
	provider := UnavailableProvider{}

	snapshot, err := provider.Snapshot(context.Background())
	if err == nil {
		t.Fatal("Snapshot returned nil error, want unavailable")
	}
	if !strings.Contains(err.Error(), "account snapshot unavailable") {
		t.Fatalf("err = %v, want account snapshot unavailable", err)
	}
	if !snapshot.SnapshotAt.IsZero() || len(snapshot.Positions) != 0 {
		t.Fatalf("snapshot = %#v, want empty snapshot", snapshot)
	}
}

func TestMockProviderReturnsConfiguredSnapshot(t *testing.T) {
	want := domain.AccountSnapshot{
		AvailableCashUSD: 12000,
		BuyingPowerUSD:   24000,
		SnapshotAt:       time.Date(2026, 5, 31, 12, 30, 0, 0, time.UTC),
		Positions: []domain.AccountPosition{{
			Symbol:         "NVDA",
			Quantity:       10,
			AverageCost:    100,
			MarketPrice:    120,
			MarketValueUSD: 1200,
		}},
	}
	provider := NewMockProvider(want, nil)

	got, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if got.AvailableCashUSD != want.AvailableCashUSD || len(got.Positions) != 1 || got.Positions[0].Symbol != "NVDA" {
		t.Fatalf("snapshot = %#v, want %#v", got, want)
	}

	got.Positions[0].Symbol = "AAPL"
	again, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("second Snapshot returned error: %v", err)
	}
	if again.Positions[0].Symbol != "NVDA" {
		t.Fatalf("mock provider leaked mutable snapshot: %#v", again.Positions)
	}
}

func TestMockProviderReturnsConfiguredError(t *testing.T) {
	provider := NewMockProvider(domain.AccountSnapshot{}, errors.New("ibkr account unavailable"))

	_, err := provider.Snapshot(context.Background())
	if err == nil || err.Error() != "ibkr account unavailable" {
		t.Fatalf("err = %v, want configured error", err)
	}
}

func TestMockProviderHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider := NewMockProvider(domain.AccountSnapshot{}, nil)

	_, err := provider.Snapshot(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context canceled", err)
	}
}
