package scheduler

import (
	"sync"
	"time"

	"ibkr-stock-analysis/internal/domain"
	"ibkr-stock-analysis/internal/market"
)

type BatchRequest struct {
	Connected bool
	Watchlist []string
	Timeframe domain.Timeframe
	ClosedAt  time.Time
	Active    bool
}

type Job struct {
	Key       string           `json:"key"`
	Symbol    string           `json:"symbol"`
	Timeframe domain.Timeframe `json:"timeframe"`
	ClosedAt  time.Time        `json:"closed_at"`
	Status    domain.JobStatus `json:"status"`
}

type Scheduler struct {
	mu        sync.Mutex
	seen      map[string]struct{}
	pendingBy map[domain.Timeframe][]Job
}

func New() *Scheduler {
	return &Scheduler{
		seen:      make(map[string]struct{}),
		pendingBy: make(map[domain.Timeframe][]Job),
	}
}

func (s *Scheduler) OnClosedBar(request BatchRequest) []Job {
	if !request.Connected {
		return nil
	}

	symbols := domain.ParseWatchlist(joinSymbols(request.Watchlist))
	jobs := make([]Job, 0, len(symbols))

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, symbol := range symbols {
		key := market.ClosedBarKey(symbol, request.Timeframe, request.ClosedAt)
		if _, ok := s.seen[key]; ok {
			continue
		}
		s.seen[key] = struct{}{}
		jobs = append(jobs, Job{
			Key:       key,
			Symbol:    symbol,
			Timeframe: request.Timeframe,
			ClosedAt:  market.CloseTime(request.ClosedAt, request.Timeframe),
			Status:    domain.JobStatusQueued,
		})
	}

	if request.Active && len(jobs) > 0 {
		s.pendingBy[request.Timeframe] = append([]Job(nil), jobs...)
	}
	return jobs
}

func (s *Scheduler) Pending(timeframe domain.Timeframe) []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending := s.pendingBy[timeframe]
	out := make([]Job, len(pending))
	copy(out, pending)
	return out
}

func (s *Scheduler) ClearPending(timeframe domain.Timeframe) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pendingBy, timeframe)
}

func joinSymbols(symbols []string) string {
	out := ""
	for i, symbol := range symbols {
		if i > 0 {
			out += ","
		}
		out += symbol
	}
	return out
}
