export type Timeframe = '1m' | '5m' | '15m' | '1h';
export type Direction = 'long' | 'short' | 'neutral';
export type SetupQuality = 'a_plus' | 'a' | 'b' | 'c' | 'none';
export type JobStatus = 'idle' | 'queued' | 'analyzing' | 'complete' | 'failed' | 'no_data';
export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'failed';
export type BacktestExitReason = 'take_profit' | 'stop_loss' | 'end_of_day' | 'partial_1r' | 'breakeven';
export type AccountSnapshotStatus = 'unavailable' | 'loading' | 'ready' | 'stale' | 'failed';
export type SizingStatus =
  | 'available'
  | 'blocked_by_cash'
  | 'blocked_by_cap'
  | 'existing_position_over_cap'
  | 'missing_account_snapshot'
  | 'missing_directional_levels'
  | 'not_a_plus'
  | 'no_trade';
export type AdvisoryAction = 'no_trade' | 'watch' | 'consider_setup' | 'manage_existing' | 'review_risk';

export interface Settings {
  ibkr_host: string;
  ibkr_port: number;
  ibkr_client_id: number;
  watchlist: string[];
  selected_timeframe: Timeframe;
  chart_window?: ChartWindow | null;
  max_stock_trade_amount_usd?: number | null;
}

export interface ChartWindow {
  id: number;
  app_name: string;
  title?: string;
}

export interface EntryZone {
  low: number;
  high: number;
}

export interface DerivedFeatures {
  session_high: number;
  session_low: number;
  recent_swing_highs: number[];
  recent_swing_lows: number[];
  atr: number;
  volume_context: string;
  last_close_relative_to_range: string;
}

export interface TimeframeContextSummary {
  timeframe: Timeframe;
  available: boolean;
  error?: string;
  current_price?: number;
  bar_count: number;
  last_closed_bar_time?: string;
  derived?: DerivedFeatures | null;
}

export interface AccountSnapshotState {
  status: AccountSnapshotStatus;
  snapshot?: AccountSnapshot | null;
  error?: string;
  updated_at?: string;
}

export interface AccountSnapshot {
  available_cash_usd: number;
  buying_power_usd: number;
  snapshot_at: string;
  positions: AccountPosition[];
  notes?: string[];
}

export interface AccountPosition {
  symbol: string;
  quantity: number;
  average_cost: number;
  market_price: number;
  market_value_usd: number;
  unrealized_pnl_usd: number;
}

export interface SizingEnvelope {
  symbol: string;
  sizing_status: SizingStatus;
  account_notional_available_usd: number;
  user_notional_cap_usd: number;
  new_exposure_cap_usd: number;
  existing_symbol_exposure_usd: number;
  remaining_symbol_cap_usd: number;
  advisory_notional_cap_usd: number;
  reference_entry_price?: number | null;
  advisory_max_shares?: number | null;
  risk_per_share?: number | null;
  estimated_risk_usd?: number | null;
}

export interface AccountSnapshotContext {
  available_cash_usd: number;
  buying_power_usd: number;
  snapshot_at: string;
  positions: AccountPositionContext[];
  max_stock_trade_amount_usd: number;
  sizing_envelope?: SizingEnvelope | null;
}

export interface AccountPositionContext {
  symbol: string;
  quantity: number;
  average_cost: number;
  market_price: number;
  market_value_usd: number;
  unrealized_pnl_usd: number;
}

export interface PositionManagementOutput {
  account_aware: boolean;
  sizing_status: SizingStatus;
  advisory_action: AdvisoryAction;
  advisory_max_shares?: number | null;
  advisory_notional_cap_usd?: number | null;
  estimated_risk_usd?: number | null;
  existing_exposure_usd?: number | null;
  management_notes: string[];
  manual_review_required: boolean;
}

export interface AgentOutput {
  direction: Direction;
  setup_quality: SetupQuality;
  entry_zone?: EntryZone | null;
  stop_loss?: number | null;
  take_profit: number[];
  risk_reward?: number | null;
  confidence: number;
  market_regime: string;
  trade_thesis: string;
  counterargument: string;
  no_trade_reason: string;
  rejection_reasons: string[];
  summary: string;
  price_action: string[];
  invalidated_if: string;
  generated_at: string;
  position_management?: PositionManagementOutput | null;
}

export interface AnalysisResult {
  symbol: string;
  timeframe: Timeframe;
  current_price: number;
  output: AgentOutput;
  context_summaries?: TimeframeContextSummary[];
  stale: boolean;
  error?: string;
  updated_at: string;
}

export interface AnalysisHistoryQuery {
  symbol?: string;
  timeframe?: Timeframe;
  direction?: Direction;
  limit: number;
  offset: number;
}

export interface AnalysisHistoryRecord {
  id: number;
  result: AnalysisResult;
}

export interface AnalysisHistoryPage {
  records: AnalysisHistoryRecord[];
  total: number;
  limit: number;
  offset: number;
}

export interface BacktestRequest {
  symbol: string;
  date?: string;
  start_date?: string;
  end_date?: string;
  start_time?: string;
  end_time?: string;
  share_quantity: number;
  slippage_per_share: number;
  commission_per_order: number;
}

export type BacktestProgressStage = 'fetching' | 'analyzing' | 'complete';

export interface BacktestProgress {
  symbol: string;
  stage: BacktestProgressStage;
  processed_bars: number;
  total_bars: number;
  current_time?: string;
  message: string;
}

export interface BacktestTrade {
  symbol: string;
  direction: Direction;
  setup_quality: SetupQuality;
  entry_time: string;
  entry_price: number;
  exit_time: string;
  exit_price: number;
  exit_reason: BacktestExitReason;
  exit_legs: BacktestExitLeg[];
  shares: number;
  initial_stop_loss: number;
  stop_loss: number;
  take_profit: number;
  gross_pnl: number;
  commission: number;
  net_pnl: number;
  return_pct: number;
  signal_time: string;
  market_regime: string;
  trade_thesis: string;
  counterargument: string;
  no_trade_reason: string;
  rejection_reasons: string[];
  summary: string;
  confidence: number;
  invalidated_if: string;
}

export interface BacktestExitLeg {
  time: string;
  price: number;
  reason: BacktestExitReason;
  shares: number;
  gross_pnl: number;
  commission: number;
  net_pnl: number;
  return_pct: number;
}

export interface BacktestSkippedSetup {
  time: string;
  direction: Direction;
  setup_quality: SetupQuality;
  reason: string;
  market_regime: string;
  trade_thesis: string;
  counterargument: string;
  no_trade_reason: string;
  rejection_reasons: string[];
  summary: string;
  confidence: number;
}

export interface BacktestOpenPosition {
  symbol: string;
  direction: Direction;
  setup_quality: SetupQuality;
  entry_time: string;
  entry_price: number;
  mark_time: string;
  mark_price: number;
  shares: number;
  remaining_shares: number;
  initial_shares: number;
  partial_taken: boolean;
  initial_stop_loss: number;
  stop_loss: number;
  take_profit: number;
  realized_gross_pnl: number;
  realized_commission: number;
  realized_net_pnl: number;
  unrealized_gross_pnl: number;
  commission: number;
  unrealized_net_pnl: number;
  unrealized_return_pct: number;
  signal_time: string;
  market_regime: string;
  trade_thesis: string;
  counterargument: string;
  no_trade_reason: string;
  rejection_reasons: string[];
  summary: string;
  confidence: number;
  invalidated_if: string;
}

export interface BacktestReport {
  symbol: string;
  date: string;
  start_date: string;
  end_date: string;
  timeframe: Timeframe;
  share_quantity: number;
  slippage_per_share: number;
  commission_per_order: number;
  start_time: string;
  end_time: string;
  bar_count: number;
  trade_count: number;
  winning_trades: number;
  losing_trades: number;
  win_rate_pct: number;
  total_gross_pnl: number;
  total_commission: number;
  total_net_pnl: number;
  total_return_pct: number;
  max_drawdown: number;
  trades: BacktestTrade[];
  skipped_setups: BacktestSkippedSetup[];
  skip_reason_counts: Record<string, number>;
  open_position?: BacktestOpenPosition;
  generated_at: string;
}

export interface SymbolState {
  symbol: string;
  market_data_status: string;
  last_closed_bar_time?: string;
  last_analysis_time?: string;
  job_status: JobStatus;
  current_price?: number;
  result?: AnalysisResult;
  account_context?: AccountSnapshotContext | null;
  error?: string;
}

export interface AppState {
  settings: Settings;
  connection_status: ConnectionStatus;
  scheduled_analysis_enabled: boolean;
  symbols: SymbolState[];
  account_snapshot: AccountSnapshotState;
  last_error?: string;
}
