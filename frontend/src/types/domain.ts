export type Timeframe = '1m' | '5m' | '15m' | '1h';
export type Direction = 'long' | 'short' | 'neutral';
export type JobStatus = 'idle' | 'queued' | 'analyzing' | 'complete' | 'failed' | 'no_data';
export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'failed';

export interface Settings {
  ibkr_host: string;
  ibkr_port: number;
  ibkr_client_id: number;
  watchlist: string[];
  selected_timeframe: Timeframe;
}

export interface EntryZone {
  low: number;
  high: number;
}

export interface AgentOutput {
  direction: Direction;
  entry_zone?: EntryZone | null;
  stop_loss?: number | null;
  take_profit: number[];
  risk_reward?: number | null;
  confidence: number;
  summary: string;
  price_action: string[];
  invalidated_if: string;
  generated_at: string;
}

export interface AnalysisResult {
  symbol: string;
  timeframe: Timeframe;
  current_price: number;
  output: AgentOutput;
  stale: boolean;
  error?: string;
  updated_at: string;
}

export interface SymbolState {
  symbol: string;
  market_data_status: string;
  last_closed_bar_time?: string;
  last_analysis_time?: string;
  job_status: JobStatus;
  current_price?: number;
  result?: AnalysisResult;
  error?: string;
}

export interface AppState {
  settings: Settings;
  connection_status: ConnectionStatus;
  symbols: SymbolState[];
  last_error?: string;
}

