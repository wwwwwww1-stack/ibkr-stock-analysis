package backtest

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"ibkr-stock-analysis/internal/analysis"
	"ibkr-stock-analysis/internal/domain"
)

const (
	defaultSimulatedShares      = 100
	partialExitShares           = 50
	cooldownBarsAfterLosingStop = 3

	skipReasonNotAPlus      = "not_a_plus"
	skipReasonCooldown      = "cooldown"
	skipReasonInvalidLevels = "invalid_levels"
	skipReasonUnmanageableR = "unmanageable_1r"
)

type Analyzer interface {
	Analyze(ctx context.Context, input domain.AgentInput) (domain.AgentOutput, error)
}

type ProgressReporter func(domain.BacktestProgress)

type Engine struct {
	analyzer Analyzer
	now      func() time.Time
	progress ProgressReporter
}

func NewEngine(analyzer Analyzer) *Engine {
	return &Engine{analyzer: analyzer}
}

func (e *Engine) SetProgressReporter(reporter ProgressReporter) {
	e.progress = reporter
}

func (e *Engine) Run(ctx context.Context, request domain.BacktestRequest, bars []domain.Bar) (domain.BacktestReport, error) {
	if e.analyzer == nil {
		return domain.BacktestReport{}, fmt.Errorf("analyzer is required")
	}
	normalized := normalizeRequest(request)
	if normalized.Symbol == "" {
		return domain.BacktestReport{}, fmt.Errorf("symbol is required")
	}
	if normalized.StartDate == "" {
		return domain.BacktestReport{}, fmt.Errorf("date is required")
	}
	if err := domain.ValidateBacktestCosts(normalized.SlippagePerShare, normalized.CommissionPerOrder); err != nil {
		return domain.BacktestReport{}, err
	}
	if len(bars) == 0 {
		return domain.BacktestReport{}, fmt.Errorf("No Data")
	}

	ordered := append([]domain.Bar(nil), bars...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Time.Before(ordered[j].Time)
	})

	report := domain.BacktestReport{
		Symbol:             normalized.Symbol,
		Date:               normalized.Date,
		StartDate:          normalized.StartDate,
		EndDate:            normalized.EndDate,
		Timeframe:          normalized.Timeframe,
		ShareQuantity:      normalized.ShareQuantity,
		SlippagePerShare:   normalized.SlippagePerShare,
		CommissionPerOrder: normalized.CommissionPerOrder,
		StartTime:          ordered[0].Time,
		EndTime:            barCloseTime(normalized.Timeframe, ordered[len(ordered)-1]),
		BarCount:           len(ordered),
		GeneratedAt:        e.timeNow(),
		Trades:             []domain.BacktestTrade{},
		SkippedSetups:      []domain.BacktestSkippedSetup{},
		SkipReasonCounts:   map[string]int{},
	}

	var open *position
	cooldownBars := 0
	for i, bar := range ordered {
		if err := ctx.Err(); err != nil {
			return domain.BacktestReport{}, err
		}
		e.reportProgress(domain.BacktestProgress{
			Symbol:        normalized.Symbol,
			Stage:         domain.BacktestProgressAnalyzing,
			ProcessedBars: i,
			TotalBars:     len(ordered),
			CurrentTime:   ptrTime(barCloseTime(normalized.Timeframe, bar)),
			Message:       fmt.Sprintf("Analyzing bar %d of %d", i+1, len(ordered)),
		})

		// A bar is either used to manage the active trade or to request a new
		// signal; never both. This avoids re-entering on the same OHLC bar that
		// just hit a stop or target.
		if open != nil {
			if trade, ok := managePosition(normalized, open, bar); ok {
				report.Trades = append(report.Trades, trade)
				if trade.ExitReason == domain.BacktestExitStopLoss && trade.NetPnL < 0 {
					cooldownBars = cooldownBarsAfterLosingStop
				}
				open = nil
			}
			e.reportProcessedBar(normalized.Symbol, normalized.Timeframe, i+1, len(ordered), bar)
			continue
		}

		input := agentInput(normalized.Symbol, normalized.Timeframe, ordered[:i+1])
		output, err := e.analyzer.Analyze(ctx, input)
		if err != nil {
			return domain.BacktestReport{}, err
		}
		signalTime := barCloseTime(normalized.Timeframe, bar)
		signal, skipped := setupFromOutput(signalTime, output)
		if skipped != nil {
			addSkippedSetup(&report, *skipped)
		}
		if signal == nil {
			if cooldownBars > 0 {
				cooldownBars--
			}
			e.reportProcessedBar(normalized.Symbol, normalized.Timeframe, i+1, len(ordered), bar)
			continue
		}
		if cooldownBars > 0 {
			addSkippedSetup(&report, skippedFromSetup(*signal, skipReasonCooldown))
			cooldownBars--
			e.reportProcessedBar(normalized.Symbol, normalized.Timeframe, i+1, len(ordered), bar)
			continue
		}
		next, enterSkipped := enterPosition(normalized, *signal, bar)
		if enterSkipped != nil {
			addSkippedSetup(&report, *enterSkipped)
			e.reportProcessedBar(normalized.Symbol, normalized.Timeframe, i+1, len(ordered), bar)
			continue
		}
		open = &next
		e.reportProcessedBar(normalized.Symbol, normalized.Timeframe, i+1, len(ordered), bar)
	}

	if open != nil {
		openPosition := markOpenPosition(normalized, *open, ordered[len(ordered)-1])
		report.OpenPosition = &openPosition
	}
	summarize(&report)
	e.reportProgress(domain.BacktestProgress{
		Symbol:        normalized.Symbol,
		Stage:         domain.BacktestProgressComplete,
		ProcessedBars: len(ordered),
		TotalBars:     len(ordered),
		CurrentTime:   ptrTime(report.EndTime),
		Message:       fmt.Sprintf("Backtest complete: %d of %d bars", len(ordered), len(ordered)),
	})
	return report, nil
}

func (e *Engine) reportProcessedBar(symbol string, timeframe domain.Timeframe, processed int, total int, bar domain.Bar) {
	e.reportProgress(domain.BacktestProgress{
		Symbol:        symbol,
		Stage:         domain.BacktestProgressAnalyzing,
		ProcessedBars: processed,
		TotalBars:     total,
		CurrentTime:   ptrTime(barCloseTime(timeframe, bar)),
		Message:       fmt.Sprintf("Processed bar %d of %d", processed, total),
	})
}

func (e *Engine) reportProgress(progress domain.BacktestProgress) {
	if e.progress != nil {
		e.progress(progress)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func addSkippedSetup(report *domain.BacktestReport, skipped domain.BacktestSkippedSetup) {
	report.SkippedSetups = append(report.SkippedSetups, skipped)
	if report.SkipReasonCounts == nil {
		report.SkipReasonCounts = map[string]int{}
	}
	report.SkipReasonCounts[skipped.Reason]++
}

func normalizeRequest(request domain.BacktestRequest) domain.BacktestRequest {
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	request.Date = strings.TrimSpace(request.Date)
	request.StartDate = strings.TrimSpace(request.StartDate)
	request.EndDate = strings.TrimSpace(request.EndDate)
	if request.StartDate == "" {
		request.StartDate = request.Date
	}
	if request.EndDate == "" {
		request.EndDate = request.StartDate
	}
	if request.Date == "" {
		request.Date = request.StartDate
	}
	if request.Timeframe == "" {
		request.Timeframe = domain.Timeframe5m
	}
	request.ShareQuantity = defaultSimulatedShares
	return request
}

func agentInput(symbol string, timeframe domain.Timeframe, bars []domain.Bar) domain.AgentInput {
	history := append([]domain.Bar(nil), bars...)
	derived := analysis.DeriveFeatures(history)
	return domain.AgentInput{
		Symbol:       symbol,
		Timeframe:    timeframe,
		CurrentPrice: history[len(history)-1].Close,
		Bars:         history,
		Derived:      derived,
	}
}

type setup struct {
	direction        domain.Direction
	setupQuality     domain.SetupQuality
	stopLoss         float64
	takeProfit       float64
	signalTime       time.Time
	summary          string
	confidence       float64
	invalidatedIf    string
	marketRegime     string
	tradeThesis      string
	counterargument  string
	noTradeReason    string
	rejectionReasons []string
}

func setupFromOutput(signalTime time.Time, output domain.AgentOutput) (*setup, *domain.BacktestSkippedSetup) {
	if output.Direction != domain.DirectionLong && output.Direction != domain.DirectionShort {
		return nil, nil
	}
	if output.SetupQuality != domain.SetupQualityAPlus {
		skipped := skippedFromOutput(signalTime, output, skipReasonNotAPlus)
		return nil, &skipped
	}
	if output.StopLoss == nil || len(output.TakeProfit) == 0 {
		skipped := skippedFromOutput(signalTime, output, skipReasonInvalidLevels)
		return nil, &skipped
	}
	return &setup{
		direction:        output.Direction,
		setupQuality:     output.SetupQuality,
		stopLoss:         *output.StopLoss,
		takeProfit:       output.TakeProfit[0],
		signalTime:       signalTime,
		summary:          output.Summary,
		confidence:       output.Confidence,
		invalidatedIf:    output.InvalidatedIf,
		marketRegime:     output.MarketRegime,
		tradeThesis:      output.TradeThesis,
		counterargument:  output.Counterargument,
		noTradeReason:    output.NoTradeReason,
		rejectionReasons: append([]string(nil), output.RejectionReasons...),
	}, nil
}

func skippedFromOutput(signalTime time.Time, output domain.AgentOutput, reason string) domain.BacktestSkippedSetup {
	return domain.BacktestSkippedSetup{
		Time:             signalTime,
		Direction:        output.Direction,
		SetupQuality:     output.SetupQuality,
		Reason:           reason,
		MarketRegime:     output.MarketRegime,
		TradeThesis:      output.TradeThesis,
		Counterargument:  output.Counterargument,
		NoTradeReason:    output.NoTradeReason,
		RejectionReasons: append([]string(nil), output.RejectionReasons...),
		Summary:          output.Summary,
		Confidence:       output.Confidence,
	}
}

func skippedFromSetup(signal setup, reason string) domain.BacktestSkippedSetup {
	return domain.BacktestSkippedSetup{
		Time:             signal.signalTime,
		Direction:        signal.direction,
		SetupQuality:     signal.setupQuality,
		Reason:           reason,
		MarketRegime:     signal.marketRegime,
		TradeThesis:      signal.tradeThesis,
		Counterargument:  signal.counterargument,
		NoTradeReason:    signal.noTradeReason,
		RejectionReasons: append([]string(nil), signal.rejectionReasons...),
		Summary:          signal.summary,
		Confidence:       signal.confidence,
	}
}

type position struct {
	setup
	entryTime        time.Time
	entryPrice       float64
	initialShares    int
	remainingShares  int
	partialShares    int
	currentStopLoss  float64
	oneRPrice        float64
	partialTaken     bool
	realizedGrossPnL float64
	exitCommission   float64
	exitLegs         []domain.BacktestExitLeg
}

func enterPosition(request domain.BacktestRequest, signal setup, bar domain.Bar) (position, *domain.BacktestSkippedSetup) {
	entryPrice := bar.Close + request.SlippagePerShare
	if signal.direction == domain.DirectionShort {
		entryPrice = bar.Close - request.SlippagePerShare
	}
	entryPrice = round2(entryPrice)
	if !validDirectionalLevels(signal, entryPrice) {
		skipped := skippedFromSetup(signal, skipReasonInvalidLevels)
		return position{}, &skipped
	}
	risk := math.Abs(entryPrice - signal.stopLoss)
	if risk <= 0 || !finitePositive(risk) {
		skipped := skippedFromSetup(signal, skipReasonInvalidLevels)
		return position{}, &skipped
	}
	oneRPrice := entryPrice + risk
	if signal.direction == domain.DirectionShort {
		oneRPrice = entryPrice - risk
	}
	if !targetSupportsOneR(signal, oneRPrice) {
		skipped := skippedFromSetup(signal, skipReasonUnmanageableR)
		return position{}, &skipped
	}
	return position{
		setup:           signal,
		entryTime:       barCloseTime(request.Timeframe, bar),
		entryPrice:      entryPrice,
		initialShares:   request.ShareQuantity,
		remainingShares: request.ShareQuantity,
		partialShares:   partialExitShares,
		currentStopLoss: signal.stopLoss,
		oneRPrice:       round2(oneRPrice),
		exitLegs:        []domain.BacktestExitLeg{},
	}, nil
}

func validDirectionalLevels(signal setup, entryPrice float64) bool {
	if !finitePositive(entryPrice) || !finitePositive(signal.stopLoss) || !finitePositive(signal.takeProfit) {
		return false
	}
	switch signal.direction {
	case domain.DirectionLong:
		return signal.stopLoss < entryPrice && signal.takeProfit > entryPrice
	case domain.DirectionShort:
		return signal.stopLoss > entryPrice && signal.takeProfit < entryPrice
	default:
		return false
	}
}

func targetSupportsOneR(signal setup, oneRPrice float64) bool {
	switch signal.direction {
	case domain.DirectionLong:
		return signal.takeProfit >= oneRPrice
	case domain.DirectionShort:
		return signal.takeProfit <= oneRPrice
	default:
		return false
	}
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func managePosition(request domain.BacktestRequest, open *position, bar domain.Bar) (domain.BacktestTrade, bool) {
	switch open.direction {
	case domain.DirectionLong:
		if bar.Low <= open.currentStopLoss {
			reason := domain.BacktestExitStopLoss
			if open.partialTaken && open.currentStopLoss == open.entryPrice {
				reason = domain.BacktestExitBreakeven
			}
			open.addExitLeg(exitLeg(request, *open, bar, open.currentStopLoss-request.SlippagePerShare, open.remainingShares, reason))
			return closeTrade(request, *open), true
		}
		if !open.partialTaken && bar.High >= open.oneRPrice {
			open.addExitLeg(exitLeg(request, *open, bar, open.oneRPrice-request.SlippagePerShare, open.partialShares, domain.BacktestExitPartial1R))
			open.partialTaken = true
			open.currentStopLoss = open.entryPrice
			return domain.BacktestTrade{}, false
		}
		if open.partialTaken && bar.High >= open.takeProfit {
			open.addExitLeg(exitLeg(request, *open, bar, open.takeProfit-request.SlippagePerShare, open.remainingShares, domain.BacktestExitTakeProfit))
			return closeTrade(request, *open), true
		}
	case domain.DirectionShort:
		if bar.High >= open.currentStopLoss {
			reason := domain.BacktestExitStopLoss
			if open.partialTaken && open.currentStopLoss == open.entryPrice {
				reason = domain.BacktestExitBreakeven
			}
			open.addExitLeg(exitLeg(request, *open, bar, open.currentStopLoss+request.SlippagePerShare, open.remainingShares, reason))
			return closeTrade(request, *open), true
		}
		if !open.partialTaken && bar.Low <= open.oneRPrice {
			open.addExitLeg(exitLeg(request, *open, bar, open.oneRPrice+request.SlippagePerShare, open.partialShares, domain.BacktestExitPartial1R))
			open.partialTaken = true
			open.currentStopLoss = open.entryPrice
			return domain.BacktestTrade{}, false
		}
		if open.partialTaken && bar.Low <= open.takeProfit {
			open.addExitLeg(exitLeg(request, *open, bar, open.takeProfit+request.SlippagePerShare, open.remainingShares, domain.BacktestExitTakeProfit))
			return closeTrade(request, *open), true
		}
	}
	return domain.BacktestTrade{}, false
}

func (p *position) addExitLeg(leg domain.BacktestExitLeg) {
	p.exitLegs = append(p.exitLegs, leg)
	p.remainingShares -= leg.Shares
	p.realizedGrossPnL += leg.GrossPnL
	p.exitCommission += leg.Commission
}

func markOpenPosition(request domain.BacktestRequest, open position, bar domain.Bar) domain.BacktestOpenPosition {
	shares := open.remainingShares
	markPrice := round2(bar.Close)
	gross := (markPrice - open.entryPrice) * float64(shares)
	if open.direction == domain.DirectionShort {
		gross = (open.entryPrice - markPrice) * float64(shares)
	}
	commission := request.CommissionPerOrder + open.exitCommission
	realizedNet := open.realizedGrossPnL - open.exitCommission
	net := gross - request.CommissionPerOrder
	notional := math.Abs(open.entryPrice * float64(shares))
	return domain.BacktestOpenPosition{
		Symbol:              request.Symbol,
		Direction:           open.direction,
		SetupQuality:        open.setupQuality,
		EntryTime:           open.entryTime,
		EntryPrice:          open.entryPrice,
		MarkTime:            barCloseTime(request.Timeframe, bar),
		MarkPrice:           markPrice,
		Shares:              shares,
		RemainingShares:     shares,
		InitialShares:       open.initialShares,
		PartialTaken:        open.partialTaken,
		InitialStopLoss:     open.stopLoss,
		StopLoss:            open.currentStopLoss,
		TakeProfit:          open.takeProfit,
		RealizedGrossPnL:    round2(open.realizedGrossPnL),
		RealizedCommission:  round2(open.exitCommission),
		RealizedNetPnL:      round2(realizedNet),
		UnrealizedGrossPnL:  round2(gross),
		Commission:          round2(commission),
		UnrealizedNetPnL:    round2(net),
		UnrealizedReturnPct: round2(percent(net, notional)),
		SignalTime:          open.signalTime,
		MarketRegime:        open.marketRegime,
		TradeThesis:         open.tradeThesis,
		Counterargument:     open.counterargument,
		NoTradeReason:       open.noTradeReason,
		RejectionReasons:    append([]string(nil), open.rejectionReasons...),
		Summary:             open.summary,
		Confidence:          open.confidence,
		InvalidatedIf:       open.invalidatedIf,
	}
}

func barCloseTime(timeframe domain.Timeframe, bar domain.Bar) time.Time {
	duration := timeframe.Duration()
	if duration <= 0 {
		return bar.Time.UTC()
	}
	return bar.Time.UTC().Add(duration)
}

func exitLeg(request domain.BacktestRequest, open position, bar domain.Bar, exitPrice float64, shares int, reason domain.BacktestExitReason) domain.BacktestExitLeg {
	adjustedExit := round2(exitPrice)
	gross := (adjustedExit - open.entryPrice) * float64(shares)
	if open.direction == domain.DirectionShort {
		gross = (open.entryPrice - adjustedExit) * float64(shares)
	}
	commission := request.CommissionPerOrder
	net := gross - commission
	notional := math.Abs(open.entryPrice * float64(shares))
	return domain.BacktestExitLeg{
		Time:       barCloseTime(request.Timeframe, bar),
		Price:      adjustedExit,
		Reason:     reason,
		Shares:     shares,
		GrossPnL:   round2(gross),
		Commission: round2(commission),
		NetPnL:     round2(net),
		ReturnPct:  round2(percent(net, notional)),
	}
}

func closeTrade(request domain.BacktestRequest, open position) domain.BacktestTrade {
	var gross float64
	commission := request.CommissionPerOrder
	for _, leg := range open.exitLegs {
		gross += leg.GrossPnL
		commission += leg.Commission
	}
	net := gross - commission
	notional := math.Abs(open.entryPrice * float64(open.initialShares))
	last := open.exitLegs[len(open.exitLegs)-1]
	return domain.BacktestTrade{
		Symbol:           request.Symbol,
		Direction:        open.direction,
		SetupQuality:     open.setupQuality,
		EntryTime:        open.entryTime,
		EntryPrice:       open.entryPrice,
		ExitTime:         last.Time,
		ExitPrice:        last.Price,
		ExitReason:       last.Reason,
		ExitLegs:         append([]domain.BacktestExitLeg(nil), open.exitLegs...),
		Shares:           open.initialShares,
		InitialStopLoss:  open.stopLoss,
		StopLoss:         open.currentStopLoss,
		TakeProfit:       open.takeProfit,
		GrossPnL:         round2(gross),
		Commission:       round2(commission),
		NetPnL:           round2(net),
		ReturnPct:        round2(percent(net, notional)),
		SignalTime:       open.signalTime,
		MarketRegime:     open.marketRegime,
		TradeThesis:      open.tradeThesis,
		Counterargument:  open.counterargument,
		NoTradeReason:    open.noTradeReason,
		RejectionReasons: append([]string(nil), open.rejectionReasons...),
		Summary:          open.summary,
		Confidence:       open.confidence,
		InvalidatedIf:    open.invalidatedIf,
	}
}

func summarize(report *domain.BacktestReport) {
	var totalNotional float64
	var equity float64
	var peak float64
	for _, trade := range report.Trades {
		report.TotalGrossPnL += trade.GrossPnL
		report.TotalCommission += trade.Commission
		report.TotalNetPnL += trade.NetPnL
		totalNotional += math.Abs(trade.EntryPrice * float64(trade.Shares))
		if trade.NetPnL > 0 {
			report.WinningTrades++
		} else if trade.NetPnL < 0 {
			report.LosingTrades++
		}
		equity += trade.NetPnL
		if equity > peak {
			peak = equity
		}
		if drawdown := peak - equity; drawdown > report.MaxDrawdown {
			report.MaxDrawdown = drawdown
		}
	}
	report.TradeCount = len(report.Trades)
	if report.TradeCount > 0 {
		report.WinRatePct = round2(float64(report.WinningTrades) / float64(report.TradeCount) * 100)
	}
	report.TotalGrossPnL = round2(report.TotalGrossPnL)
	report.TotalCommission = round2(report.TotalCommission)
	report.TotalNetPnL = round2(report.TotalNetPnL)
	report.TotalReturnPct = round2(percent(report.TotalNetPnL, totalNotional))
	report.MaxDrawdown = round2(report.MaxDrawdown)
}

func percent(value, base float64) float64 {
	if base == 0 {
		return 0
	}
	return value / base * 100
}

func (e *Engine) timeNow() time.Time {
	if e.now != nil {
		return e.now().UTC()
	}
	return time.Now().UTC()
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
