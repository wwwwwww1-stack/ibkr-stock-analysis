package market

import (
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func TestIBKRProviderSatisfiesMarketDataProvider(t *testing.T) {
	var _ MarketDataProvider = NewIBKRProvider()
}

func TestUSStockContractUsesSmartUSD(t *testing.T) {
	contract := usStockContract("nvda")

	if contract.Symbol != "NVDA" {
		t.Fatalf("symbol = %q, want NVDA", contract.Symbol)
	}
	if contract.SecType != "STK" {
		t.Fatalf("sec type = %q, want STK", contract.SecType)
	}
	if contract.Exchange != "SMART" {
		t.Fatalf("exchange = %q, want SMART", contract.Exchange)
	}
	if contract.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", contract.Currency)
	}
}

func TestHistoricalRequestParametersForTimeframes(t *testing.T) {
	tests := map[domain.Timeframe]struct {
		barSize  string
		duration string
	}{
		domain.Timeframe1m:  {barSize: "1 min", duration: "2 D"},
		domain.Timeframe5m:  {barSize: "5 mins", duration: "2 D"},
		domain.Timeframe15m: {barSize: "15 mins", duration: "1 W"},
		domain.Timeframe1h:  {barSize: "1 hour", duration: "1 M"},
	}

	for timeframe, want := range tests {
		t.Run(timeframe.String(), func(t *testing.T) {
			params := historicalRequestParameters(timeframe, 100)
			if params.BarSize != want.barSize {
				t.Fatalf("bar size = %q, want %q", params.BarSize, want.barSize)
			}
			if params.Duration != want.duration {
				t.Fatalf("duration = %q, want %q", params.Duration, want.duration)
			}
			if params.WhatToShow != "TRADES" {
				t.Fatalf("whatToShow = %q, want TRADES", params.WhatToShow)
			}
		})
	}
}

func TestParseIBKRBarDate(t *testing.T) {
	got, err := parseIBKRBarDate("1770000000")
	if err != nil {
		t.Fatalf("parse epoch returned error: %v", err)
	}
	if !got.Equal(time.Unix(1770000000, 0).UTC()) {
		t.Fatalf("epoch = %s", got)
	}

	got, err = parseIBKRBarDate("20260530 14:35:00")
	if err != nil {
		t.Fatalf("parse formatted returned error: %v", err)
	}
	if got.Location() != time.UTC {
		t.Fatalf("location = %s, want UTC", got.Location())
	}
}
