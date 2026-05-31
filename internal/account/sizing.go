package account

import (
	"math"
	"strings"

	"ibkr-stock-analysis/internal/domain"
)

type SizingInput struct {
	Symbol                 string
	Snapshot               *domain.AccountSnapshot
	MaxStockTradeAmountUSD *float64
	CurrentPrice           float64
	SetupQuality           domain.SetupQuality
	EntryZone              *domain.EntryZone
	StopLoss               *float64
}

func BuildSizingEnvelope(input SizingInput) domain.SizingEnvelope {
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	userCap := finiteNonNegativePtr(input.MaxStockTradeAmountUSD)
	envelope := domain.SizingEnvelope{
		Symbol:             symbol,
		SizingStatus:       domain.SizingStatusMissingAccountSnapshot,
		UserNotionalCapUSD: userCap,
	}
	if input.Snapshot == nil {
		return envelope
	}

	accountAvailable := finiteNonNegative(input.Snapshot.AvailableCashUSD)
	existingExposure := existingExposureUSD(symbol, input.Snapshot.Positions)
	newExposureCap := math.Min(accountAvailable, userCap)
	remainingSymbolCap := math.Max(0, userCap-existingExposure)
	advisoryCap := math.Min(newExposureCap, remainingSymbolCap)

	envelope.AccountNotionalAvailableUSD = accountAvailable
	envelope.NewExposureCapUSD = finiteNonNegative(newExposureCap)
	envelope.ExistingSymbolExposureUSD = existingExposure
	envelope.RemainingSymbolCapUSD = finiteNonNegative(remainingSymbolCap)
	envelope.AdvisoryNotionalCapUSD = finiteNonNegative(advisoryCap)

	referencePrice := referenceEntryPrice(input.EntryZone, input.CurrentPrice)
	if referencePrice != nil {
		envelope.ReferenceEntryPrice = referencePrice
		shares := int(math.Floor(envelope.AdvisoryNotionalCapUSD / *referencePrice))
		if shares < 0 {
			shares = 0
		}
		envelope.AdvisoryMaxShares = &shares
	}
	if referencePrice != nil && input.StopLoss != nil && finitePositive(*input.StopLoss) {
		risk := math.Abs(*referencePrice - *input.StopLoss)
		if finitePositive(risk) {
			envelope.RiskPerShare = &risk
			if envelope.AdvisoryMaxShares != nil {
				estimated := risk * float64(*envelope.AdvisoryMaxShares)
				estimated = finiteNonNegative(estimated)
				envelope.EstimatedRiskUSD = &estimated
			}
		}
	}

	envelope.SizingStatus = sizingStatus(input, envelope)
	if input.SetupQuality != domain.SetupQualityAPlus {
		envelope.AdvisoryNotionalCapUSD = 0
		if envelope.AdvisoryMaxShares != nil {
			zero := 0
			envelope.AdvisoryMaxShares = &zero
		}
		if envelope.EstimatedRiskUSD != nil {
			zero := 0.0
			envelope.EstimatedRiskUSD = &zero
		}
	}
	return envelope
}

func sizingStatus(input SizingInput, envelope domain.SizingEnvelope) domain.SizingStatus {
	if input.SetupQuality == domain.SetupQualityNone || input.SetupQuality == "" {
		return domain.SizingStatusNoTrade
	}
	if input.SetupQuality != domain.SetupQualityAPlus {
		return domain.SizingStatusNotAPlus
	}
	if envelope.AccountNotionalAvailableUSD <= 0 {
		return domain.SizingStatusBlockedByCash
	}
	if envelope.UserNotionalCapUSD <= 0 {
		return domain.SizingStatusBlockedByCap
	}
	if envelope.ExistingSymbolExposureUSD >= envelope.UserNotionalCapUSD && envelope.UserNotionalCapUSD > 0 {
		return domain.SizingStatusExistingPositionOverCap
	}
	if envelope.AdvisoryNotionalCapUSD <= 0 {
		return domain.SizingStatusBlockedByCap
	}
	if envelope.ReferenceEntryPrice == nil || input.StopLoss == nil || !finitePositive(*input.StopLoss) {
		return domain.SizingStatusMissingDirectionalLevels
	}
	return domain.SizingStatusAvailable
}

func existingExposureUSD(symbol string, positions []domain.AccountPosition) float64 {
	for _, position := range positions {
		if !strings.EqualFold(strings.TrimSpace(position.Symbol), symbol) {
			continue
		}
		return finiteNonNegative(math.Abs(position.MarketValueUSD))
	}
	return 0
}

func referenceEntryPrice(entryZone *domain.EntryZone, currentPrice float64) *float64 {
	if entryZone != nil && finitePositive(entryZone.Low) && finitePositive(entryZone.High) {
		low := math.Min(entryZone.Low, entryZone.High)
		high := math.Max(entryZone.Low, entryZone.High)
		midpoint := (low + high) / 2
		if finitePositive(midpoint) {
			return &midpoint
		}
	}
	if finitePositive(currentPrice) {
		price := currentPrice
		return &price
	}
	return nil
}

func finiteNonNegativePtr(value *float64) float64 {
	if value == nil {
		return 0
	}
	return finiteNonNegative(*value)
}

func finiteNonNegative(value float64) float64 {
	if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
