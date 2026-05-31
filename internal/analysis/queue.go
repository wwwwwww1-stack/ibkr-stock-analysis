package analysis

import (
	"context"
	"fmt"
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
				failJobResult(&results[i], input.Symbol, err, previous)
				return
			}
			if err := validatePositionManagement(input, output); err != nil {
				failJobResult(&results[i], input.Symbol, err, previous)
				return
			}
			result := domain.AnalysisResult{
				Symbol:           normalize(input.Symbol),
				Timeframe:        input.Timeframe,
				CurrentPrice:     input.CurrentPrice,
				Output:           output,
				ContextSummaries: domain.SummarizeTimeframeContexts(input.MultiTimeframeContext),
				Stale:            false,
				UpdatedAt:        time.Now().UTC(),
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

func failJobResult(result *JobResult, symbol string, err error, previous map[string]domain.AnalysisResult) {
	result.Status = domain.JobStatusFailed
	result.Error = err.Error()
	if previousResult, ok := previous[normalize(symbol)]; ok {
		previousResult.Stale = true
		result.Result = &previousResult
	}
}

func validatePositionManagement(input domain.AgentInput, output domain.AgentOutput) error {
	position := output.PositionManagement
	if position == nil {
		return nil
	}
	if input.AccountContext == nil || input.AccountContext.SizingEnvelope == nil {
		return fmt.Errorf("position_management requires backend account sizing envelope")
	}
	if !position.ManualReviewRequired {
		return fmt.Errorf("position_management.manual_review_required must be true")
	}
	if !allowedSizingStatus(position.SizingStatus) {
		return fmt.Errorf("position_management.sizing_status is invalid")
	}
	if !allowedAdvisoryAction(position.AdvisoryAction) {
		return fmt.Errorf("position_management.advisory_action is invalid")
	}
	envelope := input.AccountContext.SizingEnvelope
	if position.AdvisoryMaxShares != nil {
		if envelope.AdvisoryMaxShares == nil {
			return fmt.Errorf("position_management.advisory_max_shares is unavailable in backend envelope")
		}
		if *position.AdvisoryMaxShares > *envelope.AdvisoryMaxShares {
			return fmt.Errorf("position_management.advisory_max_shares exceeds backend envelope")
		}
	}
	if position.AdvisoryNotionalCapUSD != nil && *position.AdvisoryNotionalCapUSD > envelope.AdvisoryNotionalCapUSD {
		return fmt.Errorf("position_management.advisory_notional_cap_usd exceeds backend envelope")
	}
	for _, note := range position.ManagementNotes {
		if containsExecutionLanguage(note) {
			return fmt.Errorf("position_management.management_notes contain execution language")
		}
	}
	return nil
}

func allowedSizingStatus(status domain.SizingStatus) bool {
	switch status {
	case domain.SizingStatusAvailable,
		domain.SizingStatusBlockedByCash,
		domain.SizingStatusBlockedByCap,
		domain.SizingStatusExistingPositionOverCap,
		domain.SizingStatusMissingAccountSnapshot,
		domain.SizingStatusMissingDirectionalLevels,
		domain.SizingStatusNotAPlus,
		domain.SizingStatusNoTrade:
		return true
	default:
		return false
	}
}

func allowedAdvisoryAction(action domain.AdvisoryAction) bool {
	switch action {
	case domain.AdvisoryActionNoTrade,
		domain.AdvisoryActionWatch,
		domain.AdvisoryActionConsiderSetup,
		domain.AdvisoryActionManageExisting,
		domain.AdvisoryActionReviewRisk:
		return true
	default:
		return false
	}
}

func containsExecutionLanguage(value string) bool {
	lower := strings.ToLower(value)
	for _, token := range []string{
		"submit order",
		"transmit",
		"place order",
		"cancel order",
		"buy now",
		"sell now",
		"auto manage",
		"order id",
		"order payload",
	} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}
