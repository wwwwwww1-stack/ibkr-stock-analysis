package account

import (
	"math"
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func TestBuildSizingEnvelopeCashLimited(t *testing.T) {
	envelope := BuildSizingEnvelope(SizingInput{
		Symbol:                 "NVDA",
		Snapshot:               snapshotWithCash(5000, nil),
		MaxStockTradeAmountUSD: ptr(10000),
		CurrentPrice:           100,
		SetupQuality:           domain.SetupQualityAPlus,
		EntryZone:              &domain.EntryZone{Low: 98, High: 102},
		StopLoss:               ptr(95),
	})

	if envelope.SizingStatus != domain.SizingStatusAvailable {
		t.Fatalf("status = %q, want available", envelope.SizingStatus)
	}
	if envelope.AdvisoryNotionalCapUSD != 5000 {
		t.Fatalf("advisory cap = %.2f, want 5000", envelope.AdvisoryNotionalCapUSD)
	}
	if envelope.AdvisoryMaxShares == nil || *envelope.AdvisoryMaxShares != 50 {
		t.Fatalf("advisory max shares = %#v, want 50", envelope.AdvisoryMaxShares)
	}
	if envelope.EstimatedRiskUSD == nil || *envelope.EstimatedRiskUSD != 250 {
		t.Fatalf("estimated risk = %#v, want 250", envelope.EstimatedRiskUSD)
	}
	assertFiniteNonNegativeEnvelope(t, envelope)
}

func TestBuildSizingEnvelopeCapLimited(t *testing.T) {
	envelope := BuildSizingEnvelope(SizingInput{
		Symbol:                 "NVDA",
		Snapshot:               snapshotWithCash(20000, nil),
		MaxStockTradeAmountUSD: ptr(7500),
		CurrentPrice:           100,
		SetupQuality:           domain.SetupQualityAPlus,
		EntryZone:              &domain.EntryZone{Low: 100, High: 100},
		StopLoss:               ptr(90),
	})

	if envelope.AdvisoryNotionalCapUSD != 7500 {
		t.Fatalf("advisory cap = %.2f, want 7500", envelope.AdvisoryNotionalCapUSD)
	}
	if envelope.AdvisoryMaxShares == nil || *envelope.AdvisoryMaxShares != 75 {
		t.Fatalf("advisory max shares = %#v, want 75", envelope.AdvisoryMaxShares)
	}
	assertFiniteNonNegativeEnvelope(t, envelope)
}

func TestBuildSizingEnvelopeExistingPositionOverCap(t *testing.T) {
	position := domain.AccountPosition{
		Symbol:         "NVDA",
		Quantity:       60,
		AverageCost:    90,
		MarketPrice:    100,
		MarketValueUSD: 6000,
	}
	envelope := BuildSizingEnvelope(SizingInput{
		Symbol:                 "NVDA",
		Snapshot:               snapshotWithCash(20000, []domain.AccountPosition{position}),
		MaxStockTradeAmountUSD: ptr(5000),
		CurrentPrice:           100,
		SetupQuality:           domain.SetupQualityAPlus,
		EntryZone:              &domain.EntryZone{Low: 99, High: 101},
		StopLoss:               ptr(95),
	})

	if envelope.SizingStatus != domain.SizingStatusExistingPositionOverCap {
		t.Fatalf("status = %q, want existing_position_over_cap", envelope.SizingStatus)
	}
	if envelope.RemainingSymbolCapUSD != 0 || envelope.AdvisoryNotionalCapUSD != 0 {
		t.Fatalf("remaining = %.2f advisory = %.2f, want zero", envelope.RemainingSymbolCapUSD, envelope.AdvisoryNotionalCapUSD)
	}
	if envelope.AdvisoryMaxShares == nil || *envelope.AdvisoryMaxShares != 0 {
		t.Fatalf("advisory max shares = %#v, want 0", envelope.AdvisoryMaxShares)
	}
	assertFiniteNonNegativeEnvelope(t, envelope)
}

func TestBuildSizingEnvelopeUnavailableSharesWhenReferencePriceMissing(t *testing.T) {
	envelope := BuildSizingEnvelope(SizingInput{
		Symbol:                 "NVDA",
		Snapshot:               snapshotWithCash(10000, nil),
		MaxStockTradeAmountUSD: ptr(10000),
		CurrentPrice:           0,
		SetupQuality:           domain.SetupQualityAPlus,
		StopLoss:               ptr(95),
	})

	if envelope.SizingStatus != domain.SizingStatusMissingDirectionalLevels {
		t.Fatalf("status = %q, want missing_directional_levels", envelope.SizingStatus)
	}
	if envelope.ReferenceEntryPrice != nil {
		t.Fatalf("reference entry price = %#v, want nil", envelope.ReferenceEntryPrice)
	}
	if envelope.AdvisoryMaxShares != nil {
		t.Fatalf("advisory max shares = %#v, want nil", envelope.AdvisoryMaxShares)
	}
	assertFiniteNonNegativeEnvelope(t, envelope)
}

func TestBuildSizingEnvelopeMissingStopKeepsSharesAndOmitsRisk(t *testing.T) {
	envelope := BuildSizingEnvelope(SizingInput{
		Symbol:                 "NVDA",
		Snapshot:               snapshotWithCash(10000, nil),
		MaxStockTradeAmountUSD: ptr(10000),
		CurrentPrice:           100,
		SetupQuality:           domain.SetupQualityAPlus,
		EntryZone:              &domain.EntryZone{Low: 100, High: 100},
	})

	if envelope.SizingStatus != domain.SizingStatusMissingDirectionalLevels {
		t.Fatalf("status = %q, want missing_directional_levels", envelope.SizingStatus)
	}
	if envelope.AdvisoryMaxShares == nil || *envelope.AdvisoryMaxShares != 100 {
		t.Fatalf("advisory max shares = %#v, want 100", envelope.AdvisoryMaxShares)
	}
	if envelope.RiskPerShare != nil || envelope.EstimatedRiskUSD != nil {
		t.Fatalf("risk per share = %#v estimated = %#v, want nil risk fields", envelope.RiskPerShare, envelope.EstimatedRiskUSD)
	}
	assertFiniteNonNegativeEnvelope(t, envelope)
}

func TestBuildSizingEnvelopeNonAPlusSetupIsNoTrade(t *testing.T) {
	envelope := BuildSizingEnvelope(SizingInput{
		Symbol:                 "NVDA",
		Snapshot:               snapshotWithCash(10000, nil),
		MaxStockTradeAmountUSD: ptr(10000),
		CurrentPrice:           100,
		SetupQuality:           domain.SetupQualityB,
		EntryZone:              &domain.EntryZone{Low: 99, High: 101},
		StopLoss:               ptr(95),
	})

	if envelope.SizingStatus != domain.SizingStatusNotAPlus {
		t.Fatalf("status = %q, want not_a_plus", envelope.SizingStatus)
	}
	if envelope.AdvisoryNotionalCapUSD != 0 {
		t.Fatalf("advisory cap = %.2f, want zero for non-A+ setup", envelope.AdvisoryNotionalCapUSD)
	}
	assertFiniteNonNegativeEnvelope(t, envelope)
}

func TestBuildSizingEnvelopeClampsNegativeInputsToZero(t *testing.T) {
	position := domain.AccountPosition{Symbol: "NVDA", MarketValueUSD: -250}
	envelope := BuildSizingEnvelope(SizingInput{
		Symbol:                 "NVDA",
		Snapshot:               snapshotWithCash(-100, []domain.AccountPosition{position}),
		MaxStockTradeAmountUSD: ptr(-10),
		CurrentPrice:           -1,
		SetupQuality:           domain.SetupQualityAPlus,
		EntryZone:              &domain.EntryZone{Low: -5, High: -1},
		StopLoss:               ptr(math.Inf(1)),
	})

	if envelope.SizingStatus != domain.SizingStatusBlockedByCash {
		t.Fatalf("status = %q, want blocked_by_cash", envelope.SizingStatus)
	}
	if envelope.AdvisoryMaxShares != nil {
		t.Fatalf("advisory max shares = %#v, want nil without valid reference price", envelope.AdvisoryMaxShares)
	}
	assertFiniteNonNegativeEnvelope(t, envelope)
}

func TestBuildSizingEnvelopeMissingSnapshot(t *testing.T) {
	envelope := BuildSizingEnvelope(SizingInput{
		Symbol:                 "NVDA",
		MaxStockTradeAmountUSD: ptr(10000),
		CurrentPrice:           100,
		SetupQuality:           domain.SetupQualityAPlus,
		EntryZone:              &domain.EntryZone{Low: 100, High: 100},
		StopLoss:               ptr(95),
	})

	if envelope.SizingStatus != domain.SizingStatusMissingAccountSnapshot {
		t.Fatalf("status = %q, want missing_account_snapshot", envelope.SizingStatus)
	}
	if envelope.AdvisoryNotionalCapUSD != 0 {
		t.Fatalf("advisory cap = %.2f, want zero", envelope.AdvisoryNotionalCapUSD)
	}
	assertFiniteNonNegativeEnvelope(t, envelope)
}

func snapshotWithCash(cash float64, positions []domain.AccountPosition) *domain.AccountSnapshot {
	return &domain.AccountSnapshot{
		AvailableCashUSD: cash,
		BuyingPowerUSD:   cash,
		SnapshotAt:       time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
		Positions:        positions,
	}
}

func ptr(value float64) *float64 {
	return &value
}

func assertFiniteNonNegativeEnvelope(t *testing.T, envelope domain.SizingEnvelope) {
	t.Helper()
	values := map[string]float64{
		"account_notional_available_usd": envelope.AccountNotionalAvailableUSD,
		"user_notional_cap_usd":          envelope.UserNotionalCapUSD,
		"new_exposure_cap_usd":           envelope.NewExposureCapUSD,
		"existing_symbol_exposure_usd":   envelope.ExistingSymbolExposureUSD,
		"remaining_symbol_cap_usd":       envelope.RemainingSymbolCapUSD,
		"advisory_notional_cap_usd":      envelope.AdvisoryNotionalCapUSD,
	}
	for name, value := range values {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			t.Fatalf("%s = %v, want finite non-negative", name, value)
		}
	}
	for name, value := range map[string]*float64{
		"reference_entry_price": envelope.ReferenceEntryPrice,
		"risk_per_share":        envelope.RiskPerShare,
		"estimated_risk_usd":    envelope.EstimatedRiskUSD,
	} {
		if value != nil && (*value < 0 || math.IsNaN(*value) || math.IsInf(*value, 0)) {
			t.Fatalf("%s = %v, want nil or finite non-negative", name, *value)
		}
	}
	if envelope.AdvisoryMaxShares != nil && *envelope.AdvisoryMaxShares < 0 {
		t.Fatalf("advisory_max_shares = %d, want non-negative", *envelope.AdvisoryMaxShares)
	}
}
