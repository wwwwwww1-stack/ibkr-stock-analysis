package account

import (
	"context"
	"sync"

	"ibkr-stock-analysis/internal/domain"
)

type MockProvider struct {
	mu       sync.Mutex
	snapshot domain.AccountSnapshot
	err      error
}

func NewMockProvider(snapshot domain.AccountSnapshot, err error) *MockProvider {
	return &MockProvider{snapshot: cloneSnapshot(snapshot), err: err}
}

func (p *MockProvider) SetSnapshot(snapshot domain.AccountSnapshot) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.snapshot = cloneSnapshot(snapshot)
	p.err = nil
}

func (p *MockProvider) SetError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.err = err
}

func (p *MockProvider) Snapshot(ctx context.Context) (domain.AccountSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return domain.AccountSnapshot{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return domain.AccountSnapshot{}, p.err
	}
	return cloneSnapshot(p.snapshot), nil
}

func cloneSnapshot(snapshot domain.AccountSnapshot) domain.AccountSnapshot {
	snapshot.Positions = append([]domain.AccountPosition(nil), snapshot.Positions...)
	snapshot.Notes = append([]string(nil), snapshot.Notes...)
	return snapshot
}
