import { describe, expect, it } from 'vitest';
import { mergeState, selectSymbol, symbolsFromWatchlist, updateSettings } from './appState';
import type { AppState } from '../types/domain';

const baseState: AppState = {
  settings: {
    ibkr_host: '127.0.0.1',
    ibkr_port: 7497,
    ibkr_client_id: 1001,
    watchlist: ['NVDA'],
    selected_timeframe: '5m',
  },
  connection_status: 'disconnected',
  scheduled_analysis_enabled: false,
  account_snapshot: {
    status: 'unavailable',
  },
  symbols: [
    {
      symbol: 'NVDA',
      market_data_status: 'idle',
      job_status: 'idle',
    },
  ],
};

describe('appState helpers', () => {
  it('builds symbol rows from a watchlist string', () => {
    expect(symbolsFromWatchlist(' nvda, AAPL, nvda ,, tsla ')).toEqual(['NVDA', 'AAPL', 'TSLA']);
  });

  it('updates settings and normalizes watchlist symbols', () => {
    const updated = updateSettings(baseState, {
      ...baseState.settings,
      watchlist: ['aapl', ' nvda ', 'AAPL'],
      selected_timeframe: '15m',
    });

    expect(updated.settings.watchlist).toEqual(['AAPL', 'NVDA']);
    expect(updated.settings.selected_timeframe).toBe('15m');
    expect(updated.symbols.map((row) => row.symbol)).toEqual(['AAPL', 'NVDA']);
  });

  it('selects an existing symbol or falls back to the first row', () => {
    const state = updateSettings(baseState, {
      ...baseState.settings,
      watchlist: ['AAPL', 'NVDA'],
    });

    expect(selectSymbol(state, 'NVDA')?.symbol).toBe('NVDA');
    expect(selectSymbol(state, 'MSFT')?.symbol).toBe('AAPL');
  });

  it('merges backend event state without mutating the previous object', () => {
    const next = mergeState(baseState, {
      connection_status: 'connected',
      symbols: [
        {
          symbol: 'NVDA',
          market_data_status: 'ready',
          job_status: 'complete',
          current_price: 125.3,
        },
      ],
    });

    expect(next.connection_status).toBe('connected');
    expect(next.symbols[0].current_price).toBe(125.3);
    expect(baseState.connection_status).toBe('disconnected');
  });
});
