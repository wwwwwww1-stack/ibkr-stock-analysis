package analysis

import (
	"context"
	"errors"
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
