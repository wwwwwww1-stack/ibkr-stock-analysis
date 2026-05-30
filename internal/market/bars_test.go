package market

import (
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func TestCloseTimeFloorsToTimeframeBoundary(t *testing.T) {
	input := time.Date(2026, 5, 30, 14, 37, 12, 0, time.FixedZone("EDT", -4*60*60))

	tests := map[domain.Timeframe]string{
		domain.Timeframe1m:  "2026-05-30T18:37:00Z",
		domain.Timeframe5m:  "2026-05-30T18:35:00Z",
		domain.Timeframe15m: "2026-05-30T18:30:00Z",
		domain.Timeframe1h:  "2026-05-30T18:00:00Z",
	}

	for timeframe, want := range tests {
		t.Run(timeframe.String(), func(t *testing.T) {
			got := CloseTime(input, timeframe)
			if got.Format(time.RFC3339) != want {
				t.Fatalf("CloseTime = %s, want %s", got.Format(time.RFC3339), want)
			}
		})
	}
}

func TestClosedBarKeyUsesSymbolTimeframeAndUTC(t *testing.T) {
	closedAt := time.Date(2026, 5, 30, 14, 35, 0, 0, time.FixedZone("EDT", -4*60*60))

	key := ClosedBarKey("nvda", domain.Timeframe5m, closedAt)

	if key != "NVDA|5m|2026-05-30T18:35:00Z" {
		t.Fatalf("key = %q", key)
	}
}
