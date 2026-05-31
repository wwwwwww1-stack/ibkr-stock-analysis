package domain

import (
	"errors"
	"fmt"
	"math"
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

type ChartImageInput struct {
	Provided   bool       `json:"provided"`
	Source     string     `json:"source,omitempty"`
	MimeType   string     `json:"mime_type,omitempty"`
	CapturedAt *time.Time `json:"captured_at,omitempty"`
	Path       string     `json:"-"`
}

type TimeframeContext struct {
	Timeframe    Timeframe        `json:"timeframe"`
	Available    bool             `json:"available"`
	Error        string           `json:"error,omitempty"`
	CurrentPrice *float64         `json:"current_price,omitempty"`
	Bars         []Bar            `json:"bars,omitempty"`
	Derived      *DerivedFeatures `json:"derived,omitempty"`
}

type TimeframeContextSummary struct {
	Timeframe         Timeframe        `json:"timeframe"`
	Available         bool             `json:"available"`
	Error             string           `json:"error,omitempty"`
	CurrentPrice      *float64         `json:"current_price,omitempty"`
	BarCount          int              `json:"bar_count"`
	LastClosedBarTime *time.Time       `json:"last_closed_bar_time,omitempty"`
	Derived           *DerivedFeatures `json:"derived,omitempty"`
}

func SummarizeTimeframeContexts(contexts []TimeframeContext) []TimeframeContextSummary {
	if len(contexts) == 0 {
		return []TimeframeContextSummary{}
	}
	summaries := make([]TimeframeContextSummary, 0, len(contexts))
	for _, context := range contexts {
		summary := TimeframeContextSummary{
			Timeframe:    context.Timeframe,
			Available:    context.Available,
			Error:        context.Error,
			CurrentPrice: copyFloat64Ptr(context.CurrentPrice),
			BarCount:     len(context.Bars),
			Derived:      copyDerivedFeaturesPtr(context.Derived),
		}
		if len(context.Bars) > 0 {
			lastClosed := context.Bars[len(context.Bars)-1].Time
			summary.LastClosedBarTime = &lastClosed
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func copyFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func copyDerivedFeaturesPtr(value *DerivedFeatures) *DerivedFeatures {
	if value == nil {
		return nil
	}
	copied := *value
	copied.RecentSwingHighs = append([]float64(nil), value.RecentSwingHighs...)
	copied.RecentSwingLows = append([]float64(nil), value.RecentSwingLows...)
	return &copied
}

type AgentInput struct {
	Symbol                string                  `json:"symbol"`
	Timeframe             Timeframe               `json:"timeframe"`
	CurrentPrice          float64                 `json:"current_price"`
	Bars                  []Bar                   `json:"bars"`
	Derived               DerivedFeatures         `json:"derived"`
	MultiTimeframeContext []TimeframeContext      `json:"multi_timeframe_context,omitempty"`
	ChartImage            *ChartImageInput        `json:"chart_image,omitempty"`
	AccountContext        *AccountSnapshotContext `json:"account_context,omitempty"`
}

type EntryZone struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

type SetupQuality string

const (
	SetupQualityAPlus SetupQuality = "a_plus"
	SetupQualityA     SetupQuality = "a"
	SetupQualityB     SetupQuality = "b"
	SetupQualityC     SetupQuality = "c"
	SetupQualityNone  SetupQuality = "none"
)

type AccountSnapshotStatus string

const (
	AccountSnapshotUnavailable AccountSnapshotStatus = "unavailable"
	AccountSnapshotLoading     AccountSnapshotStatus = "loading"
	AccountSnapshotReady       AccountSnapshotStatus = "ready"
	AccountSnapshotStale       AccountSnapshotStatus = "stale"
	AccountSnapshotFailed      AccountSnapshotStatus = "failed"
)

type SizingStatus string

const (
	SizingStatusAvailable                SizingStatus = "available"
	SizingStatusBlockedByCash            SizingStatus = "blocked_by_cash"
	SizingStatusBlockedByCap             SizingStatus = "blocked_by_cap"
	SizingStatusExistingPositionOverCap  SizingStatus = "existing_position_over_cap"
	SizingStatusMissingAccountSnapshot   SizingStatus = "missing_account_snapshot"
	SizingStatusMissingDirectionalLevels SizingStatus = "missing_directional_levels"
	SizingStatusNotAPlus                 SizingStatus = "not_a_plus"
	SizingStatusNoTrade                  SizingStatus = "no_trade"
)

type AdvisoryAction string

const (
	AdvisoryActionNoTrade        AdvisoryAction = "no_trade"
	AdvisoryActionWatch          AdvisoryAction = "watch"
	AdvisoryActionConsiderSetup  AdvisoryAction = "consider_setup"
	AdvisoryActionManageExisting AdvisoryAction = "manage_existing"
	AdvisoryActionReviewRisk     AdvisoryAction = "review_risk"
)

type AccountSnapshotState struct {
	Status    AccountSnapshotStatus `json:"status"`
	Snapshot  *AccountSnapshot      `json:"snapshot,omitempty"`
	Error     string                `json:"error,omitempty"`
	UpdatedAt *time.Time            `json:"updated_at,omitempty"`
}

type AccountSnapshot struct {
	AvailableCashUSD float64           `json:"available_cash_usd"`
	BuyingPowerUSD   float64           `json:"buying_power_usd"`
	SnapshotAt       time.Time         `json:"snapshot_at"`
	Positions        []AccountPosition `json:"positions"`
	Notes            []string          `json:"notes,omitempty"`
}

type AccountPosition struct {
	Symbol           string  `json:"symbol"`
	Quantity         float64 `json:"quantity"`
	AverageCost      float64 `json:"average_cost"`
	MarketPrice      float64 `json:"market_price"`
	MarketValueUSD   float64 `json:"market_value_usd"`
	UnrealizedPnLUSD float64 `json:"unrealized_pnl_usd"`
}

type SizingEnvelope struct {
	Symbol                      string       `json:"symbol"`
	SizingStatus                SizingStatus `json:"sizing_status"`
	AccountNotionalAvailableUSD float64      `json:"account_notional_available_usd"`
	UserNotionalCapUSD          float64      `json:"user_notional_cap_usd"`
	NewExposureCapUSD           float64      `json:"new_exposure_cap_usd"`
	ExistingSymbolExposureUSD   float64      `json:"existing_symbol_exposure_usd"`
	RemainingSymbolCapUSD       float64      `json:"remaining_symbol_cap_usd"`
	AdvisoryNotionalCapUSD      float64      `json:"advisory_notional_cap_usd"`
	ReferenceEntryPrice         *float64     `json:"reference_entry_price,omitempty"`
	AdvisoryMaxShares           *int         `json:"advisory_max_shares,omitempty"`
	RiskPerShare                *float64     `json:"risk_per_share,omitempty"`
	EstimatedRiskUSD            *float64     `json:"estimated_risk_usd,omitempty"`
}

type AccountSnapshotContext struct {
	AvailableCashUSD       float64                  `json:"available_cash_usd"`
	BuyingPowerUSD         float64                  `json:"buying_power_usd"`
	SnapshotAt             time.Time                `json:"snapshot_at"`
	Positions              []AccountPositionContext `json:"positions"`
	MaxStockTradeAmountUSD float64                  `json:"max_stock_trade_amount_usd"`
	SizingEnvelope         *SizingEnvelope          `json:"sizing_envelope,omitempty"`
}

type AccountPositionContext struct {
	Symbol           string  `json:"symbol"`
	Quantity         float64 `json:"quantity"`
	AverageCost      float64 `json:"average_cost"`
	MarketPrice      float64 `json:"market_price"`
	MarketValueUSD   float64 `json:"market_value_usd"`
	UnrealizedPnLUSD float64 `json:"unrealized_pnl_usd"`
}

type AgentOutput struct {
	Direction          Direction                 `json:"direction"`
	SetupQuality       SetupQuality              `json:"setup_quality"`
	EntryZone          *EntryZone                `json:"entry_zone,omitempty"`
	StopLoss           *float64                  `json:"stop_loss,omitempty"`
	TakeProfit         []float64                 `json:"take_profit"`
	RiskReward         *float64                  `json:"risk_reward,omitempty"`
	Confidence         float64                   `json:"confidence"`
	MarketRegime       string                    `json:"market_regime"`
	TradeThesis        string                    `json:"trade_thesis"`
	Counterargument    string                    `json:"counterargument"`
	NoTradeReason      string                    `json:"no_trade_reason"`
	RejectionReasons   []string                  `json:"rejection_reasons"`
	Summary            string                    `json:"summary"`
	PriceAction        []string                  `json:"price_action"`
	InvalidatedIf      string                    `json:"invalidated_if"`
	GeneratedAt        time.Time                 `json:"generated_at"`
	PositionManagement *PositionManagementOutput `json:"position_management,omitempty"`
}

type PositionManagementOutput struct {
	AccountAware           bool           `json:"account_aware"`
	SizingStatus           SizingStatus   `json:"sizing_status"`
	AdvisoryAction         AdvisoryAction `json:"advisory_action"`
	AdvisoryMaxShares      *int           `json:"advisory_max_shares,omitempty"`
	AdvisoryNotionalCapUSD *float64       `json:"advisory_notional_cap_usd,omitempty"`
	EstimatedRiskUSD       *float64       `json:"estimated_risk_usd,omitempty"`
	ExistingExposureUSD    *float64       `json:"existing_exposure_usd,omitempty"`
	ManagementNotes        []string       `json:"management_notes"`
	ManualReviewRequired   bool           `json:"manual_review_required"`
}

func (o AgentOutput) Validate() error {
	var problems []string
	switch o.Direction {
	case DirectionLong, DirectionShort, DirectionNeutral:
	default:
		problems = append(problems, "direction must be long, short, or neutral")
	}
	switch o.SetupQuality {
	case SetupQualityAPlus, SetupQualityA, SetupQualityB, SetupQualityC, SetupQualityNone:
	default:
		problems = append(problems, "setup_quality must be a_plus, a, b, c, or none")
	}
	if o.Direction == DirectionNeutral && o.SetupQuality == SetupQualityAPlus {
		problems = append(problems, "setup_quality a_plus requires a long or short direction")
	}
	if o.Confidence < 0 || o.Confidence > 1 {
		problems = append(problems, "confidence must be between 0 and 1")
	}
	if o.GeneratedAt.IsZero() {
		problems = append(problems, "generated_at must be present")
	}
	if strings.TrimSpace(o.MarketRegime) == "" {
		problems = append(problems, "market_regime must be present")
	}
	if strings.TrimSpace(o.Counterargument) == "" {
		problems = append(problems, "counterargument must be present")
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
		if strings.TrimSpace(o.TradeThesis) == "" {
			problems = append(problems, "trade_thesis must be present for directional analysis")
		}
	}
	if o.Direction == DirectionNeutral && strings.TrimSpace(o.NoTradeReason) == "" {
		problems = append(problems, "no_trade_reason must be present for neutral analysis")
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

type AnalysisResult struct {
	Symbol           string                    `json:"symbol"`
	Timeframe        Timeframe                 `json:"timeframe"`
	CurrentPrice     float64                   `json:"current_price"`
	Output           AgentOutput               `json:"output"`
	ContextSummaries []TimeframeContextSummary `json:"context_summaries,omitempty"`
	Stale            bool                      `json:"stale"`
	Error            string                    `json:"error,omitempty"`
	UpdatedAt        time.Time                 `json:"updated_at"`
}

type AnalysisHistoryQuery struct {
	Symbol    string    `json:"symbol,omitempty"`
	Timeframe Timeframe `json:"timeframe,omitempty"`
	Direction Direction `json:"direction,omitempty"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}

type AnalysisHistoryRecord struct {
	ID     int64          `json:"id"`
	Result AnalysisResult `json:"result"`
}

type AnalysisHistoryPage struct {
	Records []AnalysisHistoryRecord `json:"records"`
	Total   int                     `json:"total"`
	Limit   int                     `json:"limit"`
	Offset  int                     `json:"offset"`
}

type BacktestExitReason string

const (
	BacktestExitTakeProfit BacktestExitReason = "take_profit"
	BacktestExitStopLoss   BacktestExitReason = "stop_loss"
	BacktestExitEndOfDay   BacktestExitReason = "end_of_day"
	BacktestExitPartial1R  BacktestExitReason = "partial_1r"
	BacktestExitBreakeven  BacktestExitReason = "breakeven"
)

type BacktestRequest struct {
	Symbol             string    `json:"symbol"`
	Date               string    `json:"date,omitempty"`
	StartDate          string    `json:"start_date,omitempty"`
	EndDate            string    `json:"end_date,omitempty"`
	StartTime          string    `json:"start_time,omitempty"`
	EndTime            string    `json:"end_time,omitempty"`
	Timeframe          Timeframe `json:"timeframe,omitempty"`
	ShareQuantity      int       `json:"share_quantity"`
	SlippagePerShare   float64   `json:"slippage_per_share"`
	CommissionPerOrder float64   `json:"commission_per_order"`
}

type BacktestProgressStage string

const (
	BacktestProgressFetching  BacktestProgressStage = "fetching"
	BacktestProgressAnalyzing BacktestProgressStage = "analyzing"
	BacktestProgressComplete  BacktestProgressStage = "complete"
)

type BacktestProgress struct {
	Symbol        string                `json:"symbol"`
	Stage         BacktestProgressStage `json:"stage"`
	ProcessedBars int                   `json:"processed_bars"`
	TotalBars     int                   `json:"total_bars"`
	CurrentTime   *time.Time            `json:"current_time,omitempty"`
	Message       string                `json:"message"`
}

const MaxBacktestSlippagePerShare = 1.00

func ValidateBacktestCosts(slippagePerShare float64, commissionPerOrder float64) error {
	if slippagePerShare < 0 || math.IsNaN(slippagePerShare) || math.IsInf(slippagePerShare, 0) {
		return fmt.Errorf("slippage_per_share must be non-negative")
	}
	if slippagePerShare > MaxBacktestSlippagePerShare {
		return fmt.Errorf("slippage_per_share must be at most $%.2f per share", MaxBacktestSlippagePerShare)
	}
	if commissionPerOrder < 0 || math.IsNaN(commissionPerOrder) || math.IsInf(commissionPerOrder, 0) {
		return fmt.Errorf("commission_per_order must be non-negative")
	}
	return nil
}

type BacktestTrade struct {
	Symbol           string             `json:"symbol"`
	Direction        Direction          `json:"direction"`
	SetupQuality     SetupQuality       `json:"setup_quality"`
	EntryTime        time.Time          `json:"entry_time"`
	EntryPrice       float64            `json:"entry_price"`
	ExitTime         time.Time          `json:"exit_time"`
	ExitPrice        float64            `json:"exit_price"`
	ExitReason       BacktestExitReason `json:"exit_reason"`
	ExitLegs         []BacktestExitLeg  `json:"exit_legs"`
	Shares           int                `json:"shares"`
	InitialStopLoss  float64            `json:"initial_stop_loss"`
	StopLoss         float64            `json:"stop_loss"`
	TakeProfit       float64            `json:"take_profit"`
	GrossPnL         float64            `json:"gross_pnl"`
	Commission       float64            `json:"commission"`
	NetPnL           float64            `json:"net_pnl"`
	ReturnPct        float64            `json:"return_pct"`
	SignalTime       time.Time          `json:"signal_time"`
	MarketRegime     string             `json:"market_regime"`
	TradeThesis      string             `json:"trade_thesis"`
	Counterargument  string             `json:"counterargument"`
	NoTradeReason    string             `json:"no_trade_reason"`
	RejectionReasons []string           `json:"rejection_reasons"`
	Summary          string             `json:"summary"`
	Confidence       float64            `json:"confidence"`
	InvalidatedIf    string             `json:"invalidated_if"`
}

type BacktestExitLeg struct {
	Time       time.Time          `json:"time"`
	Price      float64            `json:"price"`
	Reason     BacktestExitReason `json:"reason"`
	Shares     int                `json:"shares"`
	GrossPnL   float64            `json:"gross_pnl"`
	Commission float64            `json:"commission"`
	NetPnL     float64            `json:"net_pnl"`
	ReturnPct  float64            `json:"return_pct"`
}

type BacktestSkippedSetup struct {
	Time             time.Time    `json:"time"`
	Direction        Direction    `json:"direction"`
	SetupQuality     SetupQuality `json:"setup_quality"`
	Reason           string       `json:"reason"`
	MarketRegime     string       `json:"market_regime"`
	TradeThesis      string       `json:"trade_thesis"`
	Counterargument  string       `json:"counterargument"`
	NoTradeReason    string       `json:"no_trade_reason"`
	RejectionReasons []string     `json:"rejection_reasons"`
	Summary          string       `json:"summary"`
	Confidence       float64      `json:"confidence"`
}

type BacktestOpenPosition struct {
	Symbol              string       `json:"symbol"`
	Direction           Direction    `json:"direction"`
	SetupQuality        SetupQuality `json:"setup_quality"`
	EntryTime           time.Time    `json:"entry_time"`
	EntryPrice          float64      `json:"entry_price"`
	MarkTime            time.Time    `json:"mark_time"`
	MarkPrice           float64      `json:"mark_price"`
	Shares              int          `json:"shares"`
	RemainingShares     int          `json:"remaining_shares"`
	InitialShares       int          `json:"initial_shares"`
	PartialTaken        bool         `json:"partial_taken"`
	InitialStopLoss     float64      `json:"initial_stop_loss"`
	StopLoss            float64      `json:"stop_loss"`
	TakeProfit          float64      `json:"take_profit"`
	RealizedGrossPnL    float64      `json:"realized_gross_pnl"`
	RealizedCommission  float64      `json:"realized_commission"`
	RealizedNetPnL      float64      `json:"realized_net_pnl"`
	UnrealizedGrossPnL  float64      `json:"unrealized_gross_pnl"`
	Commission          float64      `json:"commission"`
	UnrealizedNetPnL    float64      `json:"unrealized_net_pnl"`
	UnrealizedReturnPct float64      `json:"unrealized_return_pct"`
	SignalTime          time.Time    `json:"signal_time"`
	MarketRegime        string       `json:"market_regime"`
	TradeThesis         string       `json:"trade_thesis"`
	Counterargument     string       `json:"counterargument"`
	NoTradeReason       string       `json:"no_trade_reason"`
	RejectionReasons    []string     `json:"rejection_reasons"`
	Summary             string       `json:"summary"`
	Confidence          float64      `json:"confidence"`
	InvalidatedIf       string       `json:"invalidated_if"`
}

type BacktestReport struct {
	Symbol             string                 `json:"symbol"`
	Date               string                 `json:"date"`
	StartDate          string                 `json:"start_date"`
	EndDate            string                 `json:"end_date"`
	Timeframe          Timeframe              `json:"timeframe"`
	ShareQuantity      int                    `json:"share_quantity"`
	SlippagePerShare   float64                `json:"slippage_per_share"`
	CommissionPerOrder float64                `json:"commission_per_order"`
	StartTime          time.Time              `json:"start_time"`
	EndTime            time.Time              `json:"end_time"`
	BarCount           int                    `json:"bar_count"`
	TradeCount         int                    `json:"trade_count"`
	WinningTrades      int                    `json:"winning_trades"`
	LosingTrades       int                    `json:"losing_trades"`
	WinRatePct         float64                `json:"win_rate_pct"`
	TotalGrossPnL      float64                `json:"total_gross_pnl"`
	TotalCommission    float64                `json:"total_commission"`
	TotalNetPnL        float64                `json:"total_net_pnl"`
	TotalReturnPct     float64                `json:"total_return_pct"`
	MaxDrawdown        float64                `json:"max_drawdown"`
	Trades             []BacktestTrade        `json:"trades"`
	SkippedSetups      []BacktestSkippedSetup `json:"skipped_setups"`
	SkipReasonCounts   map[string]int         `json:"skip_reason_counts"`
	OpenPosition       *BacktestOpenPosition  `json:"open_position,omitempty"`
	GeneratedAt        time.Time              `json:"generated_at"`
}

type SymbolState struct {
	Symbol            string                  `json:"symbol"`
	MarketDataStatus  string                  `json:"market_data_status"`
	LastClosedBarTime *time.Time              `json:"last_closed_bar_time,omitempty"`
	LastAnalysisTime  *time.Time              `json:"last_analysis_time,omitempty"`
	JobStatus         JobStatus               `json:"job_status"`
	CurrentPrice      *float64                `json:"current_price,omitempty"`
	Result            *AnalysisResult         `json:"result,omitempty"`
	AccountContext    *AccountSnapshotContext `json:"account_context,omitempty"`
	Error             string                  `json:"error,omitempty"`
}

type Settings struct {
	IBKRHost               string       `json:"ibkr_host"`
	IBKRPort               int          `json:"ibkr_port"`
	IBKRClientID           int          `json:"ibkr_client_id"`
	Watchlist              []string     `json:"watchlist"`
	SelectedTimeframe      Timeframe    `json:"selected_timeframe"`
	ChartWindow            *ChartWindow `json:"chart_window,omitempty"`
	MaxStockTradeAmountUSD *float64     `json:"max_stock_trade_amount_usd,omitempty"`
}

type ChartWindow struct {
	ID      int    `json:"id"`
	AppName string `json:"app_name"`
	Title   string `json:"title,omitempty"`
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
	if s.ChartWindow != nil {
		s.ChartWindow.AppName = strings.TrimSpace(s.ChartWindow.AppName)
		s.ChartWindow.Title = strings.TrimSpace(s.ChartWindow.Title)
		if s.ChartWindow.ID <= 0 && s.ChartWindow.AppName == "" && s.ChartWindow.Title == "" {
			s.ChartWindow = nil
		}
	}
	input := strings.Join(s.Watchlist, ",")
	s.Watchlist = ParseWatchlist(input)
	return s
}

func (s Settings) Validate() error {
	if s.MaxStockTradeAmountUSD != nil {
		amount := *s.MaxStockTradeAmountUSD
		if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
			return fmt.Errorf("max_stock_trade_amount_usd must be a positive finite number")
		}
	}
	return nil
}

type AppState struct {
	Settings                 Settings             `json:"settings"`
	ConnectionStatus         ConnectionStatus     `json:"connection_status"`
	ScheduledAnalysisEnabled bool                 `json:"scheduled_analysis_enabled"`
	Symbols                  []SymbolState        `json:"symbols"`
	AccountSnapshot          AccountSnapshotState `json:"account_snapshot"`
	LastError                string               `json:"last_error,omitempty"`
}
