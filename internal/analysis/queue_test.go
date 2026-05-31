package analysis

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

type queueClient struct {
	active    atomic.Int64
	maxActive atomic.Int64
	err       error
	output    domain.AgentOutput
}

func (c *queueClient) Analyze(ctx context.Context, input domain.AgentInput) (domain.AgentOutput, error) {
	current := c.active.Add(1)
	for {
		max := c.maxActive.Load()
		if current <= max || c.maxActive.CompareAndSwap(max, current) {
			break
		}
	}
	defer c.active.Add(-1)
	time.Sleep(10 * time.Millisecond)
	if c.err != nil {
		return domain.AgentOutput{}, c.err
	}
	return c.output, nil
}

func TestQueueLimitsConcurrency(t *testing.T) {
	client := &queueClient{output: validOutput(domain.DirectionNeutral)}
	queue := NewQueue(client, 3)
	inputs := make([]domain.AgentInput, 10)
	for i := range inputs {
		inputs[i] = domain.AgentInput{Symbol: string(rune('A' + i))}
	}

	results := queue.RunBatch(context.Background(), inputs, nil)

	if len(results) != 10 {
		t.Fatalf("results = %d, want 10", len(results))
	}
	if client.maxActive.Load() > 3 {
		t.Fatalf("max active = %d, want <= 3", client.maxActive.Load())
	}
}

func TestQueueMarksFailureAndKeepsPreviousResultStale(t *testing.T) {
	previous := domain.AnalysisResult{
		Symbol:    "NVDA",
		Timeframe: domain.Timeframe5m,
		Output:    validOutput(domain.DirectionNeutral),
		UpdatedAt: time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC),
	}
	client := &queueClient{err: errors.New("rate limited")}
	queue := NewQueue(client, 3)

	results := queue.RunBatch(context.Background(), []domain.AgentInput{{Symbol: "NVDA", Timeframe: domain.Timeframe5m}}, map[string]domain.AnalysisResult{"NVDA": previous})

	result := results[0]
	if result.Status != domain.JobStatusFailed {
		t.Fatalf("status = %q, want failed", result.Status)
	}
	if result.Result == nil || !result.Result.Stale {
		t.Fatalf("result = %#v, want stale previous result", result.Result)
	}
	if result.Error != "rate limited" {
		t.Fatalf("error = %q, want rate limited", result.Error)
	}
}

func TestQueueCompletesSuccessfulJob(t *testing.T) {
	client := &queueClient{output: validOutput(domain.DirectionLong)}
	queue := NewQueue(client, 3)

	results := queue.RunBatch(context.Background(), []domain.AgentInput{{Symbol: "NVDA", Timeframe: domain.Timeframe5m, CurrentPrice: 125}}, nil)

	result := results[0]
	if result.Status != domain.JobStatusComplete {
		t.Fatalf("status = %q, want complete", result.Status)
	}
	if result.Result == nil || result.Result.Output.Direction != domain.DirectionLong {
		t.Fatalf("result = %#v, want long output", result.Result)
	}
}

func TestQueueRejectsPositionManagementSharesAboveEnvelopeAndKeepsPreviousStale(t *testing.T) {
	maxShares := 50
	client := &queueClient{output: outputWithPositionManagement(domain.PositionManagementOutput{
		AccountAware:         true,
		SizingStatus:         domain.SizingStatusAvailable,
		AdvisoryAction:       domain.AdvisoryActionConsiderSetup,
		AdvisoryMaxShares:    ptrInt(51),
		ManualReviewRequired: true,
		ManagementNotes:      []string{"手动复核：仓位上限以内才考虑。"},
	})}
	previous := domain.AnalysisResult{Symbol: "NVDA", Output: validOutput(domain.DirectionLong), UpdatedAt: time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC)}
	queue := NewQueue(client, 1)

	results := queue.RunBatch(context.Background(), []domain.AgentInput{accountInput(maxShares, 5000)}, map[string]domain.AnalysisResult{"NVDA": previous})

	result := results[0]
	if result.Status != domain.JobStatusFailed {
		t.Fatalf("status = %q, want failed", result.Status)
	}
	if result.Result == nil || !result.Result.Stale {
		t.Fatalf("result = %#v, want stale previous result", result.Result)
	}
	if result.Error == "" || !contains(result.Error, "advisory_max_shares") {
		t.Fatalf("error = %q, want advisory_max_shares validation error", result.Error)
	}
}

func TestQueueRejectsPositionManagementNotionalAboveEnvelope(t *testing.T) {
	client := &queueClient{output: outputWithPositionManagement(domain.PositionManagementOutput{
		AccountAware:           true,
		SizingStatus:           domain.SizingStatusAvailable,
		AdvisoryAction:         domain.AdvisoryActionConsiderSetup,
		AdvisoryNotionalCapUSD: ptrFloat(5000.01),
		ManualReviewRequired:   true,
		ManagementNotes:        []string{"手动复核：不要超过配置上限。"},
	})}
	queue := NewQueue(client, 1)

	results := queue.RunBatch(context.Background(), []domain.AgentInput{accountInput(50, 5000)}, nil)

	if results[0].Status != domain.JobStatusFailed || !contains(results[0].Error, "advisory_notional_cap_usd") {
		t.Fatalf("result = %#v, want notional cap validation failure", results[0])
	}
}

func TestQueueRejectsPositionManagementWithoutManualReview(t *testing.T) {
	client := &queueClient{output: outputWithPositionManagement(domain.PositionManagementOutput{
		AccountAware:         true,
		SizingStatus:         domain.SizingStatusAvailable,
		AdvisoryAction:       domain.AdvisoryActionConsiderSetup,
		AdvisoryMaxShares:    ptrInt(10),
		ManualReviewRequired: false,
		ManagementNotes:      []string{"手动复核：观察触发。"},
	})}
	queue := NewQueue(client, 1)

	results := queue.RunBatch(context.Background(), []domain.AgentInput{accountInput(50, 5000)}, nil)

	if results[0].Status != domain.JobStatusFailed || !contains(results[0].Error, "manual_review_required") {
		t.Fatalf("result = %#v, want manual_review_required validation failure", results[0])
	}
}

func TestQueueRejectsPositionManagementExecutionLanguage(t *testing.T) {
	client := &queueClient{output: outputWithPositionManagement(domain.PositionManagementOutput{
		AccountAware:         true,
		SizingStatus:         domain.SizingStatusAvailable,
		AdvisoryAction:       domain.AdvisoryActionConsiderSetup,
		AdvisoryMaxShares:    ptrInt(10),
		ManualReviewRequired: true,
		ManagementNotes:      []string{"submit order after breakout"},
	})}
	queue := NewQueue(client, 1)

	results := queue.RunBatch(context.Background(), []domain.AgentInput{accountInput(50, 5000)}, nil)

	if results[0].Status != domain.JobStatusFailed || !contains(results[0].Error, "execution language") {
		t.Fatalf("result = %#v, want execution language validation failure", results[0])
	}
}

func TestQueueAllowsValidPositionManagementInsideEnvelope(t *testing.T) {
	client := &queueClient{output: outputWithPositionManagement(domain.PositionManagementOutput{
		AccountAware:           true,
		SizingStatus:           domain.SizingStatusAvailable,
		AdvisoryAction:         domain.AdvisoryActionConsiderSetup,
		AdvisoryMaxShares:      ptrInt(25),
		AdvisoryNotionalCapUSD: ptrFloat(2500),
		EstimatedRiskUSD:       ptrFloat(125),
		ManualReviewRequired:   true,
		ManagementNotes:        []string{"手动复核：只在触发和止损都清楚时考虑。"},
	})}
	queue := NewQueue(client, 1)

	results := queue.RunBatch(context.Background(), []domain.AgentInput{accountInput(50, 5000)}, nil)

	if results[0].Status != domain.JobStatusComplete {
		t.Fatalf("status = %q error = %q, want complete", results[0].Status, results[0].Error)
	}
}

func validOutput(direction domain.Direction) domain.AgentOutput {
	output := domain.AgentOutput{
		Direction:     direction,
		TakeProfit:    []float64{},
		Confidence:    0.55,
		Summary:       "Balanced.",
		PriceAction:   []string{"Inside range"},
		InvalidatedIf: "Range breaks",
		GeneratedAt:   time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC),
	}
	if direction == domain.DirectionLong || direction == domain.DirectionShort {
		stop := 124.2
		riskReward := 2.0
		output.EntryZone = &domain.EntryZone{Low: 125.1, High: 125.5}
		output.StopLoss = &stop
		output.TakeProfit = []float64{126.4}
		output.RiskReward = &riskReward
	}
	return output
}

func outputWithPositionManagement(position domain.PositionManagementOutput) domain.AgentOutput {
	output := validOutput(domain.DirectionLong)
	output.SetupQuality = domain.SetupQualityAPlus
	output.PositionManagement = &position
	return output
}

func accountInput(maxShares int, advisoryCap float64) domain.AgentInput {
	return domain.AgentInput{
		Symbol:       "NVDA",
		Timeframe:    domain.Timeframe5m,
		CurrentPrice: 100,
		AccountContext: &domain.AccountSnapshotContext{
			AvailableCashUSD:       10000,
			BuyingPowerUSD:         20000,
			SnapshotAt:             time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
			MaxStockTradeAmountUSD: 10000,
			SizingEnvelope: &domain.SizingEnvelope{
				Symbol:                 "NVDA",
				SizingStatus:           domain.SizingStatusAvailable,
				AdvisoryNotionalCapUSD: advisoryCap,
				AdvisoryMaxShares:      &maxShares,
			},
		},
	}
}

func ptrInt(value int) *int { return &value }

func ptrFloat(value float64) *float64 { return &value }

func contains(value string, fragment string) bool {
	return strings.Contains(value, fragment)
}
