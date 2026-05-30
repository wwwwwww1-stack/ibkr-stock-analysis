package market

import (
	"fmt"
	"strings"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func CloseTime(t time.Time, timeframe domain.Timeframe) time.Time {
	duration := timeframe.Duration()
	if duration <= 0 {
		return t.UTC()
	}
	return t.UTC().Truncate(duration)
}

func ClosedBarKey(symbol string, timeframe domain.Timeframe, closedAt time.Time) string {
	return fmt.Sprintf("%s|%s|%s", strings.ToUpper(strings.TrimSpace(symbol)), timeframe, CloseTime(closedAt, timeframe).Format(time.RFC3339))
}
