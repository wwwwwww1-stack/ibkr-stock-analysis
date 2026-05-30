package market

import (
	"context"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

type ConnectionSettings struct {
	Host     string
	Port     int
	ClientID int
}

type ProviderState struct {
	Status    domain.ConnectionStatus `json:"status"`
	Message   string                  `json:"message,omitempty"`
	UpdatedAt time.Time               `json:"updated_at"`
}

type BarUpdate struct {
	Symbol    string           `json:"symbol"`
	Timeframe domain.Timeframe `json:"timeframe"`
	Bar       domain.Bar       `json:"bar"`
	Closed    bool             `json:"closed"`
}

type Unsubscribe func()

type MarketDataProvider interface {
	Connect(ctx context.Context, settings ConnectionSettings) error
	Disconnect(ctx context.Context) error
	State() ProviderState
	Subscribe(ctx context.Context, symbol string, timeframe domain.Timeframe) (<-chan BarUpdate, Unsubscribe, error)
	HistoricalBars(ctx context.Context, symbol string, timeframe domain.Timeframe, limit int) ([]domain.Bar, error)
}
