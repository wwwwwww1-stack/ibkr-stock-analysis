package analysis

import (
	"math"

	"ibkr-stock-analysis/internal/domain"
)

func DeriveFeatures(bars []domain.Bar) domain.DerivedFeatures {
	if len(bars) == 0 {
		return domain.DerivedFeatures{
			RecentSwingHighs:         []float64{},
			RecentSwingLows:          []float64{},
			VolumeContext:            "unknown",
			LastCloseRelativeToRange: "unknown",
		}
	}

	sessionHigh := bars[0].High
	sessionLow := bars[0].Low
	var totalVolume int64
	var trueRangeSum float64
	var previousClose float64

	for i, bar := range bars {
		sessionHigh = math.Max(sessionHigh, bar.High)
		sessionLow = math.Min(sessionLow, bar.Low)
		totalVolume += bar.Volume
		trueRange := bar.High - bar.Low
		if i > 0 {
			trueRange = math.Max(trueRange, math.Abs(bar.High-previousClose))
			trueRange = math.Max(trueRange, math.Abs(bar.Low-previousClose))
		}
		trueRangeSum += trueRange
		previousClose = bar.Close
	}

	return domain.DerivedFeatures{
		SessionHigh:              sessionHigh,
		SessionLow:               sessionLow,
		RecentSwingHighs:         recentSwingHighs(bars, 3),
		RecentSwingLows:          recentSwingLows(bars, 3),
		ATR:                      round2(trueRangeSum / float64(len(bars))),
		VolumeContext:            volumeContext(bars, totalVolume),
		LastCloseRelativeToRange: closeRelativeToRange(bars[len(bars)-1].Close, sessionHigh, sessionLow),
	}
}

func recentSwingHighs(bars []domain.Bar, limit int) []float64 {
	swings := make([]float64, 0, limit)
	for i := 1; i < len(bars)-1; i++ {
		if bars[i].High > bars[i-1].High && bars[i].High > bars[i+1].High {
			swings = append(swings, bars[i].High)
		}
	}
	return tail(swings, limit)
}

func recentSwingLows(bars []domain.Bar, limit int) []float64 {
	swings := make([]float64, 0, limit)
	for i := 1; i < len(bars)-1; i++ {
		if bars[i].Low < bars[i-1].Low && bars[i].Low < bars[i+1].Low {
			swings = append(swings, bars[i].Low)
		}
	}
	return tail(swings, limit)
}

func tail(values []float64, limit int) []float64 {
	if len(values) > limit {
		values = values[len(values)-limit:]
	}
	if values == nil {
		return []float64{}
	}
	return values
}

func volumeContext(bars []domain.Bar, total int64) string {
	if len(bars) < 2 {
		return "unknown"
	}
	average := float64(total-bars[len(bars)-1].Volume) / float64(len(bars)-1)
	last := float64(bars[len(bars)-1].Volume)
	switch {
	case last >= average*1.2:
		return "above_average"
	case last <= average*0.8:
		return "below_average"
	default:
		return "average"
	}
}

func closeRelativeToRange(close, high, low float64) string {
	if high <= low {
		return "unknown"
	}
	position := (close - low) / (high - low)
	switch {
	case position >= 0.66:
		return "near_high"
	case position >= 0.5:
		return "upper_half"
	case position <= 0.34:
		return "near_low"
	default:
		return "lower_half"
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
