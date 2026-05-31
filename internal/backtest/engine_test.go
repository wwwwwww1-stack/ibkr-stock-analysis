package backtest

import (
	"context"
	"strings"
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func TestEngineRunsOnePositionAtATimeAndAppliesCosts(t *testing.T) {
	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	bars := []domain.Bar{
		{Time: start, Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1000},
		{Time: start.Add(5 * time.Minute), Open: 100.5, High: 101.2, Low: 100.2, Close: 101, Volume: 1100},
		{Time: start.Add(10 * time.Minute), Open: 101, High: 105.5, Low: 100.8, Close: 105, Volume: 1200},
		{Time: start.Add(15 * time.Minute), Open: 105, High: 105.2, Low: 102.5, Close: 103, Volume: 1200},
		{Time: start.Add(20 * time.Minute), Open: 103, High: 103.2, Low: 99.5, Close: 100, Volume: 1200},
		{Time: start.Add(25 * time.Minute), Open: 100, High: 100.5, Low: 98.5, Close: 99, Volume: 1200},
		{Time: start.Add(30 * time.Minute), Open: 99, High: 99.2, Low: 98.5, Close: 98.7, Volume: 1200},
	}
	stop := 98.0
	target := 105.0
	shortStop := 101.0
	shortTarget := 98.97
	agent := &scriptedAnalyzer{
		outputs: []domain.AgentOutput{
			directionalOutput(domain.DirectionLong, domain.SetupQualityAPlus, stop, target, start, "long setup"),
			directionalOutput(domain.DirectionShort, domain.SetupQualityAPlus, shortStop, shortTarget, start.Add(20*time.Minute), "short setup"),
		},
	}

	report, err := NewEngine(agent).Run(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		Date:               "2026-05-29",
		ShareQuantity:      100,
		SlippagePerShare:   0.01,
		CommissionPerOrder: 1,
	}, bars)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(agent.inputs) != 2 {
		t.Fatalf("agent calls = %d, want 2; inputs = %#v", len(agent.inputs), agent.inputs)
	}
	if got := len(agent.inputs[0].Bars); got != 1 {
		t.Fatalf("first agent input bars = %d, want no future bars beyond first close", got)
	}
	if got := agent.inputs[1].Bars[len(agent.inputs[1].Bars)-1].Time; !got.Equal(bars[4].Time) {
		t.Fatalf("second agent input last bar = %s, want next bar after exit %s", got, bars[4].Time)
	}
	if report.TradeCount != 2 || len(report.Trades) != 2 {
		t.Fatalf("trades = %#v, want two closed trades", report.Trades)
	}
	trade := report.Trades[0]
	if trade.Direction != domain.DirectionLong {
		t.Fatalf("direction = %q, want long", trade.Direction)
	}
	if trade.ExitReason != domain.BacktestExitTakeProfit {
		t.Fatalf("exit reason = %q, want take_profit", trade.ExitReason)
	}
	if trade.EntryPrice != 100.01 {
		t.Fatalf("entry price = %.2f, want current close plus slippage 100.01", trade.EntryPrice)
	}
	if !trade.EntryTime.Equal(bars[0].Time.Add(5 * time.Minute)) {
		t.Fatalf("entry time = %s, want first bar close time %s", trade.EntryTime, bars[0].Time.Add(5*time.Minute))
	}
	if !trade.ExitTime.Equal(bars[3].Time.Add(5 * time.Minute)) {
		t.Fatalf("exit time = %s, want exit bar close time %s", trade.ExitTime, bars[3].Time.Add(5*time.Minute))
	}
	if trade.ExitPrice != 104.99 {
		t.Fatalf("exit price = %.2f, want 104.99", trade.ExitPrice)
	}
	if trade.NetPnL != 346 {
		t.Fatalf("net pnl = %.2f, want 346.00", trade.NetPnL)
	}
	shortTrade := report.Trades[1]
	if shortTrade.Direction != domain.DirectionShort {
		t.Fatalf("second direction = %q, want short", shortTrade.Direction)
	}
	if shortTrade.EntryPrice != 99.99 {
		t.Fatalf("short entry price = %.2f, want current close minus slippage 99.99", shortTrade.EntryPrice)
	}
	if !shortTrade.EntryTime.Equal(bars[4].Time.Add(5 * time.Minute)) {
		t.Fatalf("short entry time = %s, want short signal bar close time %s", shortTrade.EntryTime, bars[4].Time.Add(5*time.Minute))
	}
	if !shortTrade.ExitTime.Equal(bars[6].Time.Add(5 * time.Minute)) {
		t.Fatalf("short exit time = %s, want short exit bar close time %s", shortTrade.ExitTime, bars[6].Time.Add(5*time.Minute))
	}
	if shortTrade.ExitPrice != 98.98 {
		t.Fatalf("short exit price = %.2f, want target plus slippage 98.98", shortTrade.ExitPrice)
	}
	if shortTrade.NetPnL != 97.5 {
		t.Fatalf("short net pnl = %.2f, want 97.50", shortTrade.NetPnL)
	}
	if report.TotalNetPnL != 443.5 || report.WinRatePct != 100 {
		t.Fatalf("summary = %#v, want net pnl 443.50 and win rate 100", report)
	}
	if !report.EndTime.Equal(bars[len(bars)-1].Time.Add(5 * time.Minute)) {
		t.Fatalf("report end time = %s, want last bar close time %s", report.EndTime, bars[len(bars)-1].Time.Add(5*time.Minute))
	}
}

func TestEngineSkipsDirectionallyInvalidSetups(t *testing.T) {
	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	bars := []domain.Bar{
		{Time: start, Open: 316, High: 317, Low: 315, Close: 316.15, Volume: 1000},
		{Time: start.Add(5 * time.Minute), Open: 316.1, High: 317.2, Low: 212.7, Close: 315, Volume: 1100},
		{Time: start.Add(10 * time.Minute), Open: 315, High: 318, Low: 314, Close: 317, Volume: 1200},
	}
	longStop := 212.75
	longTarget := 218.15
	shortStop := 100.0
	shortTarget := 340.0
	agent := &scriptedAnalyzer{
		outputs: []domain.AgentOutput{
			directionalOutput(domain.DirectionLong, domain.SetupQualityAPlus, longStop, longTarget, start, "long levels are below entry"),
			directionalOutput(domain.DirectionShort, domain.SetupQualityAPlus, shortStop, shortTarget, start.Add(5*time.Minute), "short levels are above entry"),
		},
	}

	report, err := NewEngine(agent).Run(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		Date:               "2026-05-29",
		ShareQuantity:      100,
		SlippagePerShare:   0.01,
		CommissionPerOrder: 0.35,
	}, bars)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(report.Trades) != 0 {
		t.Fatalf("trades = %#v, want invalid directional setups skipped", report.Trades)
	}
	if report.TotalNetPnL != 0 || report.TotalGrossPnL != 0 {
		t.Fatalf("summary = %#v, want zero PnL for skipped invalid setups", report)
	}
}

func TestEngineSkipsNonAPlusDirectionalSetups(t *testing.T) {
	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	bars := []domain.Bar{
		{Time: start, Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1000},
		{Time: start.Add(5 * time.Minute), Open: 100, High: 101, Low: 99.5, Close: 100.5, Volume: 1000},
	}
	stop := 98.0
	target := 104.0
	agent := &scriptedAnalyzer{outputs: []domain.AgentOutput{
		directionalOutput(domain.DirectionLong, domain.SetupQualityA, stop, target, start, "good but not A+"),
		directionalOutput(domain.DirectionLong, domain.SetupQualityB, stop, target, start.Add(5*time.Minute), "still not A+"),
	}}

	report, err := NewEngine(agent).Run(context.Background(), domain.BacktestRequest{
		Symbol:        "NVDA",
		Date:          "2026-05-29",
		ShareQuantity: 100,
	}, bars)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(report.Trades) != 0 {
		t.Fatalf("trades = %#v, want non-A+ directional setups skipped", report.Trades)
	}
	if len(report.SkippedSetups) != 2 {
		t.Fatalf("skipped setups = %#v, want two non-A+ skips", report.SkippedSetups)
	}
	if report.SkipReasonCounts["not_a_plus"] != 2 {
		t.Fatalf("skip counts = %#v, want not_a_plus count 2", report.SkipReasonCounts)
	}
}

func TestEngineTakesOneRPartialThenExitsRemainingAtTarget(t *testing.T) {
	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	bars := []domain.Bar{
		{Time: start, Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1000},
		{Time: start.Add(5 * time.Minute), Open: 100, High: 104.5, Low: 99.8, Close: 102, Volume: 1000},
		{Time: start.Add(10 * time.Minute), Open: 102, High: 104.5, Low: 101.5, Close: 104, Volume: 1000},
	}
	stop := 98.0
	target := 104.0
	agent := &scriptedAnalyzer{outputs: []domain.AgentOutput{
		directionalOutput(domain.DirectionLong, domain.SetupQualityAPlus, stop, target, start, "A+ long"),
	}}

	report, err := NewEngine(agent).Run(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		Date:               "2026-05-29",
		ShareQuantity:      100,
		CommissionPerOrder: 1,
	}, bars)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(report.Trades) != 1 {
		t.Fatalf("trades = %#v, want one grouped trade", report.Trades)
	}
	trade := report.Trades[0]
	if len(trade.ExitLegs) != 2 {
		t.Fatalf("exit legs = %#v, want partial and final legs", trade.ExitLegs)
	}
	if trade.ExitLegs[0].Reason != domain.BacktestExitPartial1R || trade.ExitLegs[0].Shares != 50 || trade.ExitLegs[0].Price != 102 {
		t.Fatalf("partial leg = %#v, want 50 shares at 102", trade.ExitLegs[0])
	}
	if trade.ExitLegs[1].Reason != domain.BacktestExitTakeProfit || trade.ExitLegs[1].Shares != 50 || trade.ExitLegs[1].Price != 104 {
		t.Fatalf("final leg = %#v, want 50 shares at target 104", trade.ExitLegs[1])
	}
	if !trade.ExitLegs[0].Time.Equal(bars[1].Time.Add(5*time.Minute)) || !trade.ExitLegs[1].Time.Equal(bars[2].Time.Add(5*time.Minute)) {
		t.Fatalf("exit leg times = %#v, want one exit action per bar", trade.ExitLegs)
	}
	if trade.GrossPnL != 300 || trade.Commission != 3 || trade.NetPnL != 297 {
		t.Fatalf("trade pnl = gross %.2f commission %.2f net %.2f, want 300/3/297", trade.GrossPnL, trade.Commission, trade.NetPnL)
	}
	if report.TotalNetPnL != 297 || report.TradeCount != 1 {
		t.Fatalf("summary = %#v, want one closed trade with 297 net", report)
	}
}

func TestEngineMovesRemainingSharesStopToBreakevenAfterPartial(t *testing.T) {
	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	bars := []domain.Bar{
		{Time: start, Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1000},
		{Time: start.Add(5 * time.Minute), Open: 100, High: 102.2, Low: 99.8, Close: 102, Volume: 1000},
		{Time: start.Add(10 * time.Minute), Open: 102, High: 102.4, Low: 99.5, Close: 100, Volume: 1000},
	}
	stop := 98.0
	target := 104.0
	agent := &scriptedAnalyzer{outputs: []domain.AgentOutput{
		directionalOutput(domain.DirectionLong, domain.SetupQualityAPlus, stop, target, start, "A+ long"),
	}}

	report, err := NewEngine(agent).Run(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		Date:               "2026-05-29",
		ShareQuantity:      100,
		CommissionPerOrder: 1,
	}, bars)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	trade := report.Trades[0]
	if len(trade.ExitLegs) != 2 {
		t.Fatalf("exit legs = %#v, want partial and breakeven legs", trade.ExitLegs)
	}
	if trade.ExitLegs[1].Reason != domain.BacktestExitBreakeven || trade.ExitLegs[1].Price != 100 {
		t.Fatalf("final leg = %#v, want breakeven stop at entry 100", trade.ExitLegs[1])
	}
	if trade.NetPnL != 97 {
		t.Fatalf("net pnl = %.2f, want 97 after three simulated orders", trade.NetPnL)
	}
}

func TestEngineCoolsDownThreeBarsAfterLosingStop(t *testing.T) {
	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	bars := []domain.Bar{
		{Time: start, Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1000},
		{Time: start.Add(5 * time.Minute), Open: 100, High: 100.2, Low: 98.8, Close: 99, Volume: 1000},
		{Time: start.Add(10 * time.Minute), Open: 99, High: 100.5, Low: 98.5, Close: 100, Volume: 1000},
		{Time: start.Add(15 * time.Minute), Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1000},
		{Time: start.Add(20 * time.Minute), Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1000},
	}
	stop := 99.0
	target := 104.0
	agent := &scriptedAnalyzer{outputs: []domain.AgentOutput{
		directionalOutput(domain.DirectionLong, domain.SetupQualityAPlus, stop, target, start, "A+ long stopped"),
		directionalOutput(domain.DirectionLong, domain.SetupQualityAPlus, stop, target, start.Add(10*time.Minute), "cooldown A+ 1"),
		directionalOutput(domain.DirectionLong, domain.SetupQualityAPlus, stop, target, start.Add(15*time.Minute), "cooldown A+ 2"),
		directionalOutput(domain.DirectionLong, domain.SetupQualityAPlus, stop, target, start.Add(20*time.Minute), "cooldown A+ 3"),
	}}

	report, err := NewEngine(agent).Run(context.Background(), domain.BacktestRequest{
		Symbol:        "NVDA",
		Date:          "2026-05-29",
		ShareQuantity: 100,
	}, bars)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(report.Trades) != 1 || report.Trades[0].ExitReason != domain.BacktestExitStopLoss {
		t.Fatalf("trades = %#v, want one losing stop trade", report.Trades)
	}
	if report.SkipReasonCounts["cooldown"] != 3 {
		t.Fatalf("skip counts = %#v, want three cooldown skips", report.SkipReasonCounts)
	}
}

func TestEngineReportsOpenPositionAfterPartialWithRealizedAndUnrealizedPnL(t *testing.T) {
	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	bars := []domain.Bar{
		{Time: start, Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1000},
		{Time: start.Add(5 * time.Minute), Open: 100, High: 102.2, Low: 99.8, Close: 102, Volume: 1000},
		{Time: start.Add(10 * time.Minute), Open: 102, High: 103.5, Low: 101.5, Close: 103, Volume: 1000},
	}
	stop := 98.0
	target := 104.0
	agent := &scriptedAnalyzer{outputs: []domain.AgentOutput{
		directionalOutput(domain.DirectionLong, domain.SetupQualityAPlus, stop, target, start, "A+ long still open"),
	}}

	report, err := NewEngine(agent).Run(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		Date:               "2026-05-29",
		ShareQuantity:      100,
		CommissionPerOrder: 1,
	}, bars)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(report.Trades) != 0 {
		t.Fatalf("trades = %#v, want no closed trade while remaining shares are open", report.Trades)
	}
	if report.OpenPosition == nil {
		t.Fatal("open position = nil, want remaining 50 shares open")
	}
	open := report.OpenPosition
	if open.Shares != 50 || open.RemainingShares != 50 || !open.PartialTaken {
		t.Fatalf("open position shares = %#v, want partial taken with 50 remaining", open)
	}
	if open.StopLoss != 100 {
		t.Fatalf("open stop = %.2f, want breakeven stop at entry 100", open.StopLoss)
	}
	if open.RealizedGrossPnL != 100 || open.UnrealizedGrossPnL != 150 {
		t.Fatalf("open pnl = realized %.2f unrealized %.2f, want 100 and 150", open.RealizedGrossPnL, open.UnrealizedGrossPnL)
	}
}

func TestEngineRejectsExcessiveSlippage(t *testing.T) {
	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	bars := []domain.Bar{
		{Time: start, Open: 316, High: 317, Low: 315, Close: 316.15, Volume: 1000},
	}

	_, err := NewEngine(&scriptedAnalyzer{}).Run(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		Date:               "2026-05-29",
		ShareQuantity:      100,
		SlippagePerShare:   100,
		CommissionPerOrder: 0.35,
	}, bars)
	if err == nil || !strings.Contains(err.Error(), "slippage_per_share must be at most $1.00 per share") {
		t.Fatalf("err = %v, want excessive slippage validation error", err)
	}
}

func TestEngineHandlesShortStopLossConservativelyAndKeepsOpenPositionAtRangeEnd(t *testing.T) {
	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	bars := []domain.Bar{
		{Time: start, Open: 100, High: 101, Low: 99, Close: 100, Volume: 1000},
		{Time: start.Add(5 * time.Minute), Open: 99.5, High: 100.5, Low: 98.5, Close: 99, Volume: 1000},
		{Time: start.Add(10 * time.Minute), Open: 99, High: 102.2, Low: 94.8, Close: 101, Volume: 1000},
		{Time: start.Add(15 * time.Minute), Open: 101, High: 101.5, Low: 100.5, Close: 101, Volume: 1000},
	}
	stop := 102.0
	target := 95.0
	output := directionalOutput(domain.DirectionShort, domain.SetupQualityAPlus, stop, target, start, "short setup")
	agent := &scriptedAnalyzer{
		outputs: []domain.AgentOutput{
			output,
			{
				Direction:        domain.DirectionNeutral,
				SetupQuality:     domain.SetupQualityNone,
				TakeProfit:       []float64{},
				Confidence:       0.5,
				MarketRegime:     "震荡",
				Counterargument:  "区间未破。",
				NoTradeReason:    "没有 A+ 触发。",
				RejectionReasons: []string{"震荡"},
				Summary:          "flat",
				PriceAction:      []string{"balanced"},
				InvalidatedIf:    "range breaks",
				GeneratedAt:      start.Add(10 * time.Minute),
			},
		},
	}

	report, err := NewEngine(agent).Run(context.Background(), domain.BacktestRequest{
		Symbol:        "NVDA",
		Date:          "2026-05-29",
		ShareQuantity: 100,
	}, bars)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(report.Trades) != 1 {
		t.Fatalf("trades = %#v, want one short trade", report.Trades)
	}
	trade := report.Trades[0]
	if trade.ExitReason != domain.BacktestExitStopLoss {
		t.Fatalf("exit reason = %q, want stop_loss for same-bar ambiguous stop/target", trade.ExitReason)
	}
	if trade.NetPnL != -200 {
		t.Fatalf("net pnl = %.2f, want -200.00", trade.NetPnL)
	}
	if report.MaxDrawdown != 200 {
		t.Fatalf("max drawdown = %.2f, want 200.00", report.MaxDrawdown)
	}

	openAgent := &scriptedAnalyzer{outputs: []domain.AgentOutput{output}}
	openReport, err := NewEngine(openAgent).Run(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		StartDate:          "2026-05-29",
		EndDate:            "2026-05-29",
		ShareQuantity:      100,
		CommissionPerOrder: 1,
	}, bars[:2])
	if err != nil {
		t.Fatalf("open-position Run returned error: %v", err)
	}
	if len(openReport.Trades) != 0 {
		t.Fatalf("open-position trades = %#v, want no forced range-end exit", openReport.Trades)
	}
	if openReport.OpenPosition == nil {
		t.Fatalf("open position = nil, want range-end open short")
	}
	if openReport.OpenPosition.Direction != domain.DirectionShort {
		t.Fatalf("open position direction = %q, want short", openReport.OpenPosition.Direction)
	}
	if openReport.OpenPosition.MarkPrice != 99 {
		t.Fatalf("mark price = %.2f, want last close 99.00", openReport.OpenPosition.MarkPrice)
	}
	if openReport.OpenPosition.UnrealizedGrossPnL != 100 || openReport.OpenPosition.UnrealizedNetPnL != 99 {
		t.Fatalf("open pnl = gross %.2f net %.2f, want 100.00 and 99.00", openReport.OpenPosition.UnrealizedGrossPnL, openReport.OpenPosition.UnrealizedNetPnL)
	}
}

func TestEngineCarriesPositionAcrossDaysAndKeepsStopFillAtStopLevel(t *testing.T) {
	firstDay := time.Date(2026, 5, 28, 19, 50, 0, 0, time.UTC)
	secondDay := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	bars := []domain.Bar{
		{Time: firstDay, Open: 100, High: 101, Low: 99.5, Close: 100, Volume: 1000},
		{Time: firstDay.Add(5 * time.Minute), Open: 100, High: 100.5, Low: 99.5, Close: 100.2, Volume: 1000},
		{Time: secondDay, Open: 95, High: 96, Low: 94, Close: 95, Volume: 1500},
	}
	stop := 98.0
	target := 105.0
	agent := &scriptedAnalyzer{outputs: []domain.AgentOutput{
		directionalOutput(domain.DirectionLong, domain.SetupQualityAPlus, stop, target, firstDay, "long setup"),
	}}

	report, err := NewEngine(agent).Run(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		StartDate:          "2026-05-28",
		EndDate:            "2026-05-29",
		ShareQuantity:      100,
		SlippagePerShare:   0.01,
		CommissionPerOrder: 1,
	}, bars)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if report.OpenPosition != nil {
		t.Fatalf("open position = %#v, want closed stop trade", report.OpenPosition)
	}
	if len(report.Trades) != 1 {
		t.Fatalf("trades = %#v, want one cross-day stop trade", report.Trades)
	}
	trade := report.Trades[0]
	if trade.ExitReason != domain.BacktestExitStopLoss {
		t.Fatalf("exit reason = %q, want stop_loss", trade.ExitReason)
	}
	if trade.ExitPrice != 97.99 {
		t.Fatalf("exit price = %.2f, want stop level minus slippage 97.99 rather than gap open", trade.ExitPrice)
	}
	if !trade.ExitTime.Equal(secondDay.Add(5 * time.Minute)) {
		t.Fatalf("exit time = %s, want second day bar close %s", trade.ExitTime, secondDay.Add(5*time.Minute))
	}
}

type scriptedAnalyzer struct {
	outputs []domain.AgentOutput
	inputs  []domain.AgentInput
}

func (a *scriptedAnalyzer) Analyze(ctx context.Context, input domain.AgentInput) (domain.AgentOutput, error) {
	if err := ctx.Err(); err != nil {
		return domain.AgentOutput{}, err
	}
	a.inputs = append(a.inputs, input)
	if len(a.outputs) == 0 {
		return domain.AgentOutput{}, nil
	}
	output := a.outputs[0]
	a.outputs = a.outputs[1:]
	return output, nil
}

func directionalOutput(direction domain.Direction, quality domain.SetupQuality, stop float64, target float64, generatedAt time.Time, summary string) domain.AgentOutput {
	riskReward := 2.0
	entry := domain.EntryZone{Low: 99, High: 101}
	if direction == domain.DirectionShort {
		entry = domain.EntryZone{Low: 99, High: 101}
	}
	rejections := []string{}
	noTradeReason := ""
	if quality != domain.SetupQualityAPlus {
		rejections = []string{"不是 A+ 级别"}
		noTradeReason = "结构还不够清晰，不能当作 A+。"
	}
	return domain.AgentOutput{
		Direction:        direction,
		SetupQuality:     quality,
		EntryZone:        &entry,
		StopLoss:         &stop,
		TakeProfit:       []float64{target},
		RiskReward:       &riskReward,
		Confidence:       0.7,
		MarketRegime:     "趋势回踩",
		TradeThesis:      "价格在关键位置完成回踩并重新出现主动买盘。",
		Counterargument:  "如果回到触发区下方，说明突破失败。",
		NoTradeReason:    noTradeReason,
		RejectionReasons: rejections,
		Summary:          summary,
		PriceAction:      []string{"关键位置有反应"},
		InvalidatedIf:    "触发区被反向收回",
		GeneratedAt:      generatedAt,
	}
}
