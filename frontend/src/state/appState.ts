import type { AppState, Settings, SymbolState } from '../types/domain';

export function symbolsFromWatchlist(input: string): string[] {
  const seen = new Set<string>();
  return input
    .split(/[,\n;\t]/)
    .map((part) => part.trim().toUpperCase())
    .filter((symbol) => /^[A-Z][A-Z0-9.-]{0,15}$/.test(symbol))
    .filter((symbol) => {
      if (seen.has(symbol)) return false;
      seen.add(symbol);
      return true;
    });
}

export function updateSettings(state: AppState, settings: Settings): AppState {
  const watchlist = symbolsFromWatchlist(settings.watchlist.join(','));
  return {
    ...state,
    settings: {
      ...settings,
      watchlist,
    },
    symbols: watchlist.map((symbol) => existingOrNew(state.symbols, symbol)),
  };
}

export function selectSymbol(state: AppState, selected?: string): SymbolState | undefined {
  return state.symbols.find((row) => row.symbol === selected) ?? state.symbols[0];
}

export function mergeState(state: AppState, patch: Partial<AppState>): AppState {
  return {
    ...state,
    ...patch,
    settings: patch.settings ? { ...state.settings, ...patch.settings } : state.settings,
    symbols: patch.symbols ?? state.symbols,
  };
}

export function defaultState(): AppState {
  return {
    settings: {
      ibkr_host: '127.0.0.1',
      ibkr_port: 7497,
      ibkr_client_id: 1001,
      watchlist: [],
      selected_timeframe: '5m',
      chart_window: null,
    },
    connection_status: 'disconnected',
    scheduled_analysis_enabled: false,
    account_snapshot: {
      status: 'unavailable',
    },
    symbols: [],
  };
}

function existingOrNew(symbols: SymbolState[], symbol: string): SymbolState {
  return (
    symbols.find((row) => row.symbol === symbol) ?? {
      symbol,
      market_data_status: 'idle',
      job_status: 'idle',
    }
  );
}
