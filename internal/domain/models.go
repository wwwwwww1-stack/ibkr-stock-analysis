package domain

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Timeframe string

const (
	Timeframe1m  Timeframe = "1m"
	Timeframe5m  Timeframe = "5m"
	Timeframe15m Timeframe = "15m"
	Timeframe1h  Timeframe = "1h"
)

func ParseTimeframe(value string) (Timeframe, error) {
	tf := Timeframe(strings.TrimSpace(value))
	switch tf {
	case Timeframe1m, Timeframe5m, Timeframe15m, Timeframe1h:
		return tf, nil
	default:
		return "", fmt.Errorf("unsupported timeframe %q", value)
	}
}

func (t Timeframe) Duration() time.Duration {
	switch t {
	case Timeframe1m:
		return time.Minute
	case Timeframe5m:
		return 5 * time.Minute
	case Timeframe15m:
		return 15 * time.Minute
	case Timeframe1h:
		return time.Hour
	default:
		return 0
	}
}

func (t Timeframe) String() string {
	return string(t)
}

var symbolPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.\-]{0,15}$`)

func ParseWatchlist(input string) []string {
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\t' || r == ';'
	})
	seen := make(map[string]struct{}, len(parts))
	symbols := make([]string, 0, len(parts))
	for _, part := range parts {
		symbol := strings.ToUpper(strings.TrimSpace(part))
		if symbol == "" {
			continue
		}
		if !symbolPattern.MatchString(symbol) {
			continue
		}
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		symbols = append(symbols, symbol)
	}
	return symbols
}

type Direction string

const (
	DirectionLong    Direction = "long"
	DirectionShort   Direction = "short"
	DirectionNeutral Direction = "neutral"
)

type JobStatus string

const (
	JobStatusIdle      JobStatus = "idle"
	JobStatusQueued    JobStatus = "queued"
	JobStatusAnalyzing JobStatus = "analyzing"
	JobStatusComplete  JobStatus = "complete"
	JobStatusFailed    JobStatus = "failed"
	JobStatusNoData    JobStatus = "no_data"
)

type ConnectionStatus string

const (
	ConnectionDisconnected ConnectionStatus = "disconnected"
	ConnectionConnecting   ConnectionStatus = "connecting"
	ConnectionConnected    ConnectionStatus = "connected"
	ConnectionFailed       ConnectionStatus = "failed"
)

type Bar struct {
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume int64     `json:"volume"`
}

type DerivedFeatures struct {
	SessionHigh              float64   `json:"session_high"`
	SessionLow               float64   `json:"session_low"`
	RecentSwingHighs         []float64 `json:"recent_swing_highs"`
	RecentSwingLows          []float64 `json:"recent_swing_lows"`
	ATR                      float64   `json:"atr"`
	VolumeContext            string    `json:"volume_context"`
	LastCloseRelativeToRange string    `json:"last_close_relative_to_range"`
}

type AgentInput struct {
	Symbol       string          `json:"symbol"`
	Timeframe    Timeframe       `json:"timeframe"`
	CurrentPrice float64         `json:"current_price"`
	Bars         []Bar           `json:"bars"`
	Derived      DerivedFeatures `json:"derived"`
}

type EntryZone struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

type AgentOutput struct {
	Direction     Direction  `json:"direction"`
	EntryZone     *EntryZone `json:"entry_zone,omitempty"`
	StopLoss      *float64   `json:"stop_loss,omitempty"`
	TakeProfit    []float64  `json:"take_profit"`
	RiskReward    *float64   `json:"risk_reward,omitempty"`
	Confidence    float64    `json:"confidence"`
	Summary       string     `json:"summary"`
	PriceAction   []string   `json:"price_action"`
	InvalidatedIf string     `json:"invalidated_if"`
	GeneratedAt   time.Time  `json:"generated_at"`
}

func (o AgentOutput) Validate() error {
	var problems []string
	switch o.Direction {
	case DirectionLong, DirectionShort, DirectionNeutral:
	default:
		problems = append(problems, "direction must be long, short, or neutral")
	}
	if o.Confidence < 0 || o.Confidence > 1 {
		problems = append(problems, "confidence must be between 0 and 1")
	}
	if o.GeneratedAt.IsZero() {
		problems = append(problems, "generated_at must be present")
	}
	if strings.TrimSpace(o.Summary) == "" {
		problems = append(problems, "summary must be present")
	}
	if strings.TrimSpace(o.InvalidatedIf) == "" {
		problems = append(problems, "invalidated_if must be present")
	}

	if o.Direction == DirectionLong || o.Direction == DirectionShort {
		if o.EntryZone == nil {
			problems = append(problems, "entry_zone must be present for directional analysis")
		} else if o.EntryZone.Low > o.EntryZone.High {
			problems = append(problems, "entry_zone.low must be less than or equal to entry_zone.high")
		}
		if o.StopLoss == nil {
			problems = append(problems, "stop_loss must be present for directional analysis")
		}
		if len(o.TakeProfit) == 0 {
			problems = append(problems, "take_profit must be present for directional analysis")
		}
		if o.RiskReward == nil || *o.RiskReward <= 0 {
			problems = append(problems, "risk_reward must be positive for directional analysis")
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

type AnalysisResult struct {
	Symbol       string      `json:"symbol"`
	Timeframe    Timeframe   `json:"timeframe"`
	CurrentPrice float64     `json:"current_price"`
	Output       AgentOutput `json:"output"`
	Stale        bool        `json:"stale"`
	Error        string      `json:"error,omitempty"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type SymbolState struct {
	Symbol            string          `json:"symbol"`
	MarketDataStatus  string          `json:"market_data_status"`
	LastClosedBarTime *time.Time      `json:"last_closed_bar_time,omitempty"`
	LastAnalysisTime  *time.Time      `json:"last_analysis_time,omitempty"`
	JobStatus         JobStatus       `json:"job_status"`
	CurrentPrice      *float64        `json:"current_price,omitempty"`
	Result            *AnalysisResult `json:"result,omitempty"`
	Error             string          `json:"error,omitempty"`
}

type Settings struct {
	IBKRHost          string    `json:"ibkr_host"`
	IBKRPort          int       `json:"ibkr_port"`
	IBKRClientID      int       `json:"ibkr_client_id"`
	Watchlist         []string  `json:"watchlist"`
	SelectedTimeframe Timeframe `json:"selected_timeframe"`
}

func DefaultSettings() Settings {
	return Settings{
		IBKRHost:          "127.0.0.1",
		IBKRPort:          7497,
		IBKRClientID:      1001,
		Watchlist:         []string{},
		SelectedTimeframe: Timeframe5m,
	}
}

func (s Settings) Normalize() Settings {
	if strings.TrimSpace(s.IBKRHost) == "" {
		s.IBKRHost = "127.0.0.1"
	}
	if s.IBKRPort == 0 {
		s.IBKRPort = 7497
	}
	if s.IBKRClientID == 0 {
		s.IBKRClientID = 1001
	}
	if _, err := ParseTimeframe(string(s.SelectedTimeframe)); err != nil {
		s.SelectedTimeframe = Timeframe5m
	}
	input := strings.Join(s.Watchlist, ",")
	s.Watchlist = ParseWatchlist(input)
	return s
}

type AppState struct {
	Settings         Settings         `json:"settings"`
	ConnectionStatus ConnectionStatus `json:"connection_status"`
	Symbols          []SymbolState    `json:"symbols"`
	LastError        string           `json:"last_error,omitempty"`
}
