package analysis

import (
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func TestDeriveFeaturesComputesRangeSwingsATRAndVolume(t *testing.T) {
	base := time.Date(2026, 5, 30, 14, 30, 0, 0, time.UTC)
	bars := []domain.Bar{
		{Time: base, Open: 100, High: 105, Low: 99, Close: 104, Volume: 100},
		{Time: base.Add(time.Minute), Open: 104, High: 106, Low: 102, Close: 103, Volume: 120},
		{Time: base.Add(2 * time.Minute), Open: 103, High: 104, Low: 98, Close: 99, Volume: 80},
		{Time: base.Add(3 * time.Minute), Open: 99, High: 101, Low: 97, Close: 100, Volume: 200},
		{Time: base.Add(4 * time.Minute), Open: 100, High: 103, Low: 99, Close: 102, Volume: 250},
	}

	derived := DeriveFeatures(bars)

	if derived.SessionHigh != 106 {
		t.Fatalf("session high = %v, want 106", derived.SessionHigh)
	}
	if derived.SessionLow != 97 {
		t.Fatalf("session low = %v, want 97", derived.SessionLow)
	}
	if len(derived.RecentSwingHighs) != 1 || derived.RecentSwingHighs[0] != 106 {
		t.Fatalf("swing highs = %#v, want [106]", derived.RecentSwingHighs)
	}
	if len(derived.RecentSwingLows) != 1 || derived.RecentSwingLows[0] != 97 {
		t.Fatalf("swing lows = %#v, want [97]", derived.RecentSwingLows)
	}
	if derived.ATR <= 0 {
		t.Fatalf("ATR = %v, want positive", derived.ATR)
	}
	if derived.VolumeContext != "above_average" {
		t.Fatalf("volume context = %q, want above_average", derived.VolumeContext)
	}
	if derived.LastCloseRelativeToRange != "upper_half" {
		t.Fatalf("range position = %q, want upper_half", derived.LastCloseRelativeToRange)
	}
}

func TestDeriveFeaturesHandlesEmptyBars(t *testing.T) {
	derived := DeriveFeatures(nil)

	if derived.SessionHigh != 0 || derived.SessionLow != 0 || derived.VolumeContext != "unknown" {
		t.Fatalf("derived = %#v, want zero range and unknown volume", derived)
	}
}
