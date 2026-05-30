package analysis

import (
	"context"
	"strings"
	"sync"
	"time"

	"ibkr-stock-analysis/internal/agent"
	"ibkr-stock-analysis/internal/domain"
)

type JobResult struct {
	Symbol string                 `json:"symbol"`
	Status domain.JobStatus       `json:"status"`
	Result *domain.AnalysisResult `json:"result,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

type Queue struct {
	client      agent.Client
	concurrency int
}

func NewQueue(client agent.Client, concurrency int) *Queue {
	if concurrency <= 0 {
		concurrency = 3
	}
	return &Queue{client: client, concurrency: concurrency}
}

func (q *Queue) RunBatch(ctx context.Context, inputs []domain.AgentInput, previous map[string]domain.AnalysisResult) []JobResult {
	results := make([]JobResult, len(inputs))
	sem := make(chan struct{}, q.concurrency)
	var wg sync.WaitGroup

	for i, input := range inputs {
		i, input := i, input
		results[i] = JobResult{Symbol: normalize(input.Symbol), Status: domain.JobStatusQueued}
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i].Status = domain.JobStatusAnalyzing
			output, err := q.client.Analyze(ctx, input)
			if err != nil {
				results[i].Status = domain.JobStatusFailed
				results[i].Error = err.Error()
				if previousResult, ok := previous[normalize(input.Symbol)]; ok {
					previousResult.Stale = true
					results[i].Result = &previousResult
				}
				return
			}
			result := domain.AnalysisResult{
				Symbol:       normalize(input.Symbol),
				Timeframe:    input.Timeframe,
				CurrentPrice: input.CurrentPrice,
				Output:       output,
				Stale:        false,
				UpdatedAt:    time.Now().UTC(),
			}
			results[i].Status = domain.JobStatusComplete
			results[i].Result = &result
		}()
	}

	wg.Wait()
	return results
}

func normalize(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}
