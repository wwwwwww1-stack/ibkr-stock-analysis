import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import App from './App';
import type { BackendAPI } from './api/backend';
import type { AppState } from './types/domain';

const emptyState: AppState = {
  settings: {
    ibkr_host: '127.0.0.1',
    ibkr_port: 7497,
    ibkr_client_id: 1001,
    watchlist: [],
    selected_timeframe: '5m',
  },
  connection_status: 'disconnected',
  symbols: [],
};

describe('App workspace', () => {
  it('renders disconnected empty watchlist state', async () => {
    render(<App backend={mockBackend(emptyState)} />);

    expect(await screen.findByText('IBKR AI Analysis')).toBeInTheDocument();
    expect(screen.getByText('disconnected')).toBeInTheDocument();
    expect(screen.getByText('Enter a watchlist to begin.')).toBeInTheDocument();
    expect(screen.getByText(/Not financial advice/)).toBeInTheDocument();
  });

  it('saves watchlist settings and displays symbol rows', async () => {
    const backend = mockBackend(emptyState);
    backend.saveSettings = vi.fn(async (): Promise<AppState> => ({
      ...emptyState,
      settings: { ...emptyState.settings, watchlist: ['NVDA', 'AAPL'] },
      symbols: [
        { symbol: 'NVDA', market_data_status: 'idle', job_status: 'idle' as const },
        { symbol: 'AAPL', market_data_status: 'idle', job_status: 'idle' as const },
      ],
    }));

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');
    await userEvent.type(screen.getByPlaceholderText('NVDA, TSLA, AAPL'), 'NVDA, AAPL');
    await userEvent.click(screen.getByText('Save'));

    await waitFor(() => expect(screen.getAllByText('NVDA').length).toBeGreaterThan(0));
    expect(screen.getAllByText('AAPL').length).toBeGreaterThan(0);
  });

  it('shows failed stale prior output in detail panel', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [
        {
          symbol: 'NVDA',
          market_data_status: 'ready',
          job_status: 'failed',
          error: 'Agent Error',
          result: {
            symbol: 'NVDA',
            timeframe: '5m',
            current_price: 125.3,
            stale: true,
            updated_at: '2026-05-30T18:36:10Z',
            output: {
              direction: 'long',
              entry_zone: { low: 125.1, high: 125.5 },
              stop_loss: 124.2,
              take_profit: [126.4, 127.2],
              risk_reward: 2.1,
              confidence: 0.68,
              summary: 'Price reclaimed the prior high.',
              price_action: ['Breakout retest held'],
              invalidated_if: 'A 5m candle closes below 124.2',
              generated_at: '2026-05-30T18:36:10Z',
            },
          },
        },
      ],
    };

    render(<App backend={mockBackend(state)} />);

    expect(await screen.findByText('Agent Error')).toBeInTheDocument();
    expect(screen.getByText('Previous valid result shown as stale.')).toBeInTheDocument();
    expect(screen.getByText('Price reclaimed the prior high.')).toBeInTheDocument();
  });
});

function mockBackend(state: AppState): BackendAPI {
  return {
    getState: vi.fn(async () => state),
    saveSettings: vi.fn(async () => state),
    connectIBKR: vi.fn(async (): Promise<AppState> => ({ ...state, connection_status: 'connected' })),
    disconnectIBKR: vi.fn(async (): Promise<AppState> => ({ ...state, connection_status: 'disconnected' })),
    runAnalysisNow: vi.fn(async () => state),
  };
}
