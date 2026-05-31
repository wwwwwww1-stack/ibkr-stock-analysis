package account

import (
	"context"
	"fmt"

	"ibkr-stock-analysis/internal/domain"
)

type SnapshotProvider interface {
	Snapshot(ctx context.Context) (domain.AccountSnapshot, error)
}

type UnavailableProvider struct{}

func (UnavailableProvider) Snapshot(ctx context.Context) (domain.AccountSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return domain.AccountSnapshot{}, err
	}
	return domain.AccountSnapshot{}, fmt.Errorf("account snapshot unavailable")
}
