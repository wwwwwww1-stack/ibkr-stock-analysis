import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import type { BackendAPI } from './api/backend';
import type { AnalysisResult, AppState, BacktestProgress, BacktestReport } from './types/domain';

const emptyState: AppState = {
  settings: {
    ibkr_host: '127.0.0.1',
    ibkr_port: 7497,
    ibkr_client_id: 1001,
    watchlist: [],
    selected_timeframe: '5m',
  },
  connection_status: 'disconnected',
  scheduled_analysis_enabled: false,
  symbols: [],
};

afterEach(() => {
  vi.useRealTimers();
  delete window.runtime;
});

describe('App workspace', () => {
  it('renders empty watchlist state after automatic connection', async () => {
    const backend = mockBackend(emptyState);

    render(<App backend={backend} />);

    expect(await screen.findByText('IBKR AI Analysis')).toBeInTheDocument();
    expect(screen.getByText('connected')).toBeInTheDocument();
    expect(screen.getByText('Enter a watchlist to begin.')).toBeInTheDocument();
    expect(screen.getByText(/Not financial advice/)).toBeInTheDocument();
    expect(backend.connectIBKR).toHaveBeenCalledTimes(1);
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
    const symbolList = screen.getByRole('list', { name: 'Watchlist symbols' });
    await userEvent.click(within(symbolList).getByRole('button', { name: 'Add symbol' }));
    await userEvent.type(screen.getByPlaceholderText('Add symbol or paste a list'), 'NVDA, AAPL{enter}');
    await userEvent.click(screen.getByText('Save'));

    await waitFor(() => expect(screen.getAllByText('NVDA').length).toBeGreaterThan(0));
    expect(screen.getAllByText('AAPL').length).toBeGreaterThan(0);
  });

  it('shows watchlist controls, signals, and selected analysis only in the watchlist workspace', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA', 'AAPL'] },
      symbols: [
        {
          symbol: 'NVDA',
          market_data_status: 'ready',
          job_status: 'complete' as const,
          current_price: 125.3,
          result: analysisResult('NVDA', 'long', 'Price reclaimed the prior high.'),
        },
        { symbol: 'AAPL', market_data_status: 'ready', job_status: 'idle' as const },
      ],
    };

    render(<App backend={mockBackend(state)} />);
    await screen.findByText('IBKR AI Analysis');

    const directory = screen.getByLabelText('Workspace sections');
    expect(within(directory).getByRole('button', { name: /Watchlist/i })).toHaveAttribute('aria-current', 'page');
    const watchlist = screen.getByLabelText('Watchlist workspace');
    expect(within(watchlist).getByLabelText('Watchlist details')).toBeInTheDocument();
    expect(within(watchlist).getByLabelText('Signals')).toBeInTheDocument();
    expect(within(watchlist).getByLabelText('Selected symbol analysis')).toBeInTheDocument();
    expect(within(watchlist).getByText('Price reclaimed the prior high.')).toBeInTheDocument();
    expect(screen.queryByLabelText('Backtest workspace')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Settings workspace')).not.toBeInTheDocument();
  });

  it('shows backtest controls without signals or selected analysis details', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA', 'AAPL'] },
      symbols: [
        {
          symbol: 'NVDA',
          market_data_status: 'ready',
          job_status: 'complete' as const,
          current_price: 125.3,
          result: analysisResult('NVDA', 'long', 'Price reclaimed the prior high.'),
        },
        { symbol: 'AAPL', market_data_status: 'ready', job_status: 'idle' as const },
      ],
    };

    render(<App backend={mockBackend(state)} />);
    await screen.findByText('IBKR AI Analysis');

    const directory = screen.getByLabelText('Workspace sections');
    expect(screen.queryByLabelText('Backtest details')).not.toBeInTheDocument();

    await userEvent.click(within(directory).getByRole('button', { name: /Backtest/i }));

    expect(within(directory).getByRole('button', { name: /Backtest/i })).toHaveAttribute('aria-current', 'page');
    const backtest = screen.getByLabelText('Backtest workspace');
    expect(within(backtest).getByLabelText('Backtest details')).toBeInTheDocument();
    expect(within(backtest).getByText('No backtest run.')).toBeInTheDocument();
    expect(within(backtest).getByRole('button', { name: 'Run Backtest' })).toBeInTheDocument();
    expect(screen.queryByLabelText('Watchlist workspace')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Signals')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Selected symbol analysis')).not.toBeInTheDocument();
  });

  it('shows settings controls without analysis actions, signals, or backtest results', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [
        {
          symbol: 'NVDA',
          market_data_status: 'ready',
          job_status: 'complete' as const,
          current_price: 125.3,
          result: analysisResult('NVDA', 'long', 'Price reclaimed the prior high.'),
        },
      ],
    };

    render(<App backend={mockBackend(state)} />);
    await screen.findByText('IBKR AI Analysis');

    const directory = screen.getByLabelText('Workspace sections');

    await userEvent.click(within(directory).getByRole('button', { name: /Settings/i }));

    expect(within(directory).getByRole('button', { name: /Settings/i })).toHaveAttribute('aria-current', 'page');
    const settings = screen.getByLabelText('Settings workspace');
    expect(within(settings).getByLabelText('IBKR connection')).toBeInTheDocument();
    expect(within(settings).getByLabelText('Chart window settings')).toBeInTheDocument();
    expect(within(settings).getByLabelText('Timeframe settings')).toBeInTheDocument();
    expect(within(settings).getByRole('button', { name: 'Save' })).toBeInTheDocument();
    expect(screen.queryByLabelText('Signals')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Selected symbol analysis')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Backtest details')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Analyze/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Screenshot Analyze/i })).not.toBeInTheDocument();
  });

  it('keeps the completed backtest report when watchlist selection changes', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA', 'AAPL'] },
      symbols: [
        { symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const },
        { symbol: 'AAPL', market_data_status: 'ready', job_status: 'idle' as const },
      ],
    };
    const backend = mockBackend(state);
    backend.runBacktest = vi.fn(async () => backtestReport());

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    const directory = screen.getByLabelText('Workspace sections');
    await userEvent.click(within(directory).getByRole('button', { name: /Backtest/i }));
    const backtest = screen.getByLabelText('Backtest workspace');
    await userEvent.click(within(backtest).getByRole('button', { name: 'Run Backtest' }));
    expect(await within(backtest).findAllByText('$297.00')).toHaveLength(2);

    await userEvent.click(within(directory).getByRole('button', { name: /Watchlist/i }));
    await userEvent.click(within(screen.getByLabelText('Watchlist workspace')).getByRole('button', { name: 'Select AAPL' }));
    await userEvent.click(within(directory).getByRole('button', { name: /Backtest/i }));

    expect(within(screen.getByLabelText('Backtest workspace')).getAllByText('$297.00')).toHaveLength(2);
  });

  it('shows a single automatic IBKR connection action without manual fields', async () => {
    const backend = mockBackend(emptyState);

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    const directory = screen.getByLabelText('Workspace sections');
    await userEvent.click(within(directory).getByRole('button', { name: /Settings/i }));
    const settingsDetails = screen.getByLabelText('Settings details');

    expect(within(settingsDetails).queryByLabelText('Host')).not.toBeInTheDocument();
    expect(within(settingsDetails).queryByLabelText('Port')).not.toBeInTheDocument();
    expect(within(settingsDetails).queryByLabelText('Client ID')).not.toBeInTheDocument();
    expect(within(settingsDetails).queryByRole('button', { name: 'Save Settings' })).not.toBeInTheDocument();

    const disconnectButtons = within(settingsDetails).getAllByRole('button', { name: 'Disconnect' });
    expect(disconnectButtons).toHaveLength(1);
    expect(backend.connectIBKR).toHaveBeenCalledTimes(1);
  });

  it('automatically retries IBKR connection every 5 seconds until connected', async () => {
    vi.useFakeTimers();
    const backend = mockBackend(emptyState);
    backend.connectIBKR = vi
      .fn<BackendAPI['connectIBKR']>()
      .mockRejectedValueOnce(new Error('gateway offline'))
      .mockRejectedValueOnce(new Error('gateway offline'))
      .mockResolvedValueOnce({ ...emptyState, connection_status: 'connected' });

    render(<App backend={backend} />);

    await act(async () => {
      await Promise.resolve();
    });
    expect(backend.connectIBKR).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(4999);
    });
    expect(backend.connectIBKR).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(backend.connectIBKR).toHaveBeenCalledTimes(2);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(backend.connectIBKR).toHaveBeenCalledTimes(3);
    expect(screen.getByText('connected')).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(backend.connectIBKR).toHaveBeenCalledTimes(3);
  });

  it('pauses and resumes scheduled analysis from watchlist actions', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      scheduled_analysis_enabled: true,
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [{ symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const }],
    };
    const backend = mockBackend(state);
    backend.setScheduledAnalysisEnabled = vi.fn(async (enabled): Promise<AppState> => ({
      ...state,
      scheduled_analysis_enabled: enabled,
    }));

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    const actionBar = within(screen.getByLabelText('Watchlist details')).getByLabelText('Pinned watchlist actions');
    await userEvent.click(within(actionBar).getByRole('button', { name: 'Pause Scheduled Analysis' }));

    await waitFor(() => expect(backend.setScheduledAnalysisEnabled).toHaveBeenCalledWith(false));
    await userEvent.click(within(actionBar).getByRole('button', { name: 'Resume Scheduled Analysis' }));

    await waitFor(() => expect(backend.setScheduledAnalysisEnabled).toHaveBeenCalledWith(true));
  });

  it('selects a chart window by name from settings', async () => {
    const backend = mockBackend(emptyState);
    backend.listChartWindows = vi.fn(async () => [
      { id: 42, app_name: 'Trader Workstation', title: 'NVDA 5m' },
      { id: 43, app_name: 'Safari', title: 'Market Notes' },
    ]);
    backend.saveSettings = vi.fn(async (_settings): Promise<AppState> => ({
      ...emptyState,
      settings: { ...emptyState.settings, chart_window: { id: 42, app_name: 'Trader Workstation', title: 'NVDA 5m' } },
    }));

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    await userEvent.click(screen.getByRole('button', { name: /Settings/i }));
    const settingsDetails = screen.getByLabelText('Settings details');
    const windowButton = await within(settingsDetails).findByRole('button', { name: /Trader Workstation.*NVDA 5m/i });
    await userEvent.click(windowButton);

    await waitFor(() =>
      expect(backend.saveSettings).toHaveBeenCalledWith({
        ...emptyState.settings,
        chart_window: { id: 42, app_name: 'Trader Workstation', title: 'NVDA 5m' },
      }),
    );
  });

  it('uses the watchlist detail view to select a symbol before analysis', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA', 'AAPL'] },
      symbols: [
        { symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const },
        { symbol: 'AAPL', market_data_status: 'ready', job_status: 'idle' as const },
      ],
    };
    const backend = mockBackend(state);

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    const watchlistDetails = screen.getByLabelText('Watchlist details');
    await userEvent.click(within(watchlistDetails).getByRole('button', { name: 'Select AAPL' }));
    await userEvent.click(within(watchlistDetails).getByRole('button', { name: 'Analyze AAPL' }));

    await waitFor(() => expect(backend.runAnalysisNow).toHaveBeenCalledWith('AAPL'));
  });

  it('removes watchlist symbols before saving settings', async () => {
    const state: AppState = {
      ...emptyState,
      settings: { ...emptyState.settings, watchlist: ['NVDA', 'AAPL'] },
      symbols: [
        { symbol: 'NVDA', market_data_status: 'idle', job_status: 'idle' as const },
        { symbol: 'AAPL', market_data_status: 'idle', job_status: 'idle' as const },
      ],
    };
    const backend = mockBackend(state);

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    await userEvent.click(screen.getByRole('button', { name: /remove AAPL/i }));
    await userEvent.click(screen.getByText('Save'));

    await waitFor(() =>
      expect(backend.saveSettings).toHaveBeenCalledWith({
        ...state.settings,
        watchlist: ['NVDA'],
      }),
    );
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
              setup_quality: 'a_plus',
              entry_zone: { low: 125.1, high: 125.5 },
              stop_loss: 124.2,
              take_profit: [126.4, 127.2],
              risk_reward: 2.1,
              confidence: 0.68,
              market_regime: '趋势回踩',
              trade_thesis: '关键位置回踩后重新出现主动买盘。',
              counterargument: '如果重新跌回区间中部，方向优势会消失。',
              no_trade_reason: '',
              rejection_reasons: [],
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

  it('runs screenshot analysis for the selected symbol', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA', 'AAPL'] },
      symbols: [
        { symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const },
        { symbol: 'AAPL', market_data_status: 'ready', job_status: 'idle' as const },
      ],
    };
    const backend = mockBackend(state);

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');
    const detail = screen.getByLabelText('Selected symbol analysis');
    await userEvent.click(within(detail).getByRole('button', { name: /screenshot analyze/i }));

    await waitFor(() => expect(backend.runScreenshotAnalysis).toHaveBeenCalledWith('NVDA'));
  });

  it('shows screenshot analysis in the main controls for the selected symbol', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [{ symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const }],
    };
    const backend = mockBackend(state);

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    const controls = screen.getByLabelText('Connection and watchlist');
    const screenshotButton = within(controls).getByRole('button', { name: /screenshot analyze/i });
    expect(screenshotButton).toBeEnabled();

    await userEvent.click(screenshotButton);

    await waitFor(() => expect(backend.runScreenshotAnalysis).toHaveBeenCalledWith('NVDA'));
  });

  it('runs regular analysis for only the selected symbol', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA', 'AAPL'] },
      symbols: [
        { symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const },
        { symbol: 'AAPL', market_data_status: 'ready', job_status: 'idle' as const },
      ],
    };
    const backend = mockBackend(state);

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    await userEvent.click(screen.getByRole('row', { name: /AAPL/ }));
    const controls = screen.getByLabelText('Connection and watchlist');
    await userEvent.click(within(controls).getByRole('button', { name: /^Analyze/ }));

    await waitFor(() => expect(backend.runAnalysisNow).toHaveBeenCalledWith('AAPL'));
  });

  it('runs a 5m backtest from the left directory with the selected date range and time window', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [{ symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const }],
    };
    const backend = mockBackend(state);
    backend.runBacktest = vi.fn(async () => backtestReport());

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    const detail = screen.getByLabelText('Selected symbol analysis');
    expect(within(detail).queryByLabelText('Backtest')).not.toBeInTheDocument();
    const directory = screen.getByLabelText('Workspace sections');
    await userEvent.click(within(directory).getByRole('button', { name: /Backtest/i }));
    const backtest = screen.getByLabelText('Backtest details');
    fireEvent.change(within(backtest).getByLabelText('Start date'), { target: { value: '2026-05-28' } });
    fireEvent.change(within(backtest).getByLabelText('End date'), { target: { value: '2026-05-29' } });
    fireEvent.change(within(backtest).getByLabelText('Start time'), { target: { value: '10:00' } });
    fireEvent.change(within(backtest).getByLabelText('End time'), { target: { value: '10:30' } });
    await userEvent.click(within(backtest).getByRole('button', { name: 'Run Backtest' }));

    await waitFor(() =>
      expect(backend.runBacktest).toHaveBeenCalledWith({
        symbol: 'NVDA',
        start_date: '2026-05-28',
        end_date: '2026-05-29',
        start_time: '10:00',
        end_time: '10:30',
        share_quantity: 100,
        slippage_per_share: 0.01,
        commission_per_order: 1,
      }),
    );
    expect(await within(backtest).findAllByText('$297.00')).toHaveLength(2);
    const trade = within(backtest).getByLabelText('Trade 1 long');
    expect(within(trade).getAllByText('Take profit').length).toBeGreaterThanOrEqual(1);
    expect(within(trade).getByText('A+')).toBeInTheDocument();
    expect(within(trade).getByText('Exit legs')).toBeInTheDocument();
    expect(within(trade).getByText('1R partial')).toBeInTheDocument();
    expect(within(trade).getByText('50 @ 102.00')).toBeInTheDocument();
    expect(within(trade).getByText('Signal')).toBeInTheDocument();
    expect(within(trade).getByText('Entry time')).toBeInTheDocument();
    expect(within(trade).getByText('Exit time')).toBeInTheDocument();
    expect(within(trade).getAllByText('09:35 AM')).toHaveLength(2);
    expect(within(trade).getByText('09:45 AM')).toBeInTheDocument();
    expect(within(trade).getByText('Entry fill')).toBeInTheDocument();
    expect(within(trade).getByText('100.00')).toBeInTheDocument();
    expect(within(trade).getByText('Exit fill')).toBeInTheDocument();
    expect(within(trade).getAllByText('104.00').length).toBeGreaterThanOrEqual(1);
    expect(within(trade).getByText('Stop')).toBeInTheDocument();
    expect(within(trade).getByText('98.00')).toBeInTheDocument();
    expect(within(trade).getByText('Target')).toBeInTheDocument();
    expect(within(trade).getAllByText('104.00').length).toBeGreaterThanOrEqual(1);
    expect(within(trade).getByText('Gross')).toBeInTheDocument();
    expect(within(trade).getByText('$300.00')).toBeInTheDocument();
    expect(within(trade).getByText('Fees')).toBeInTheDocument();
    expect(within(trade).getByText('$3.00')).toBeInTheDocument();
    expect(within(trade).getByText('Return')).toBeInTheDocument();
    expect(within(trade).getByText('2.97%')).toBeInTheDocument();
    expect(within(trade).getByText('Price reclaimed support and has room to target.')).toBeInTheDocument();
    expect(within(trade).getByText('A failed breakout would trap late longs.')).toBeInTheDocument();

    const skipped = within(backtest).getByLabelText('Skipped setups');
    expect(within(skipped).getByText('not_a_plus')).toBeInTheDocument();
    expect(within(skipped).getByText('cooldown')).toBeInTheDocument();
    expect(within(skipped).getByText('Good idea, but still inside the range.')).toBeInTheDocument();

    const openPosition = within(backtest).getByLabelText('Open position long');
    expect(within(openPosition).getByText('Open Position')).toBeInTheDocument();
    expect(within(openPosition).getAllByText('$149.00').length).toBeGreaterThanOrEqual(1);
    expect(within(openPosition).getByText('Remaining')).toBeInTheDocument();
    expect(within(openPosition).getAllByText('50').length).toBeGreaterThanOrEqual(1);
    expect(within(openPosition).getByText('Partial taken')).toBeInTheDocument();
    expect(within(openPosition).getByText('Realized')).toBeInTheDocument();
    expect(within(openPosition).getByText('$99.00')).toBeInTheDocument();
    expect(within(openPosition).getByText('Unrealized')).toBeInTheDocument();
    expect(within(openPosition).getByText('Mark')).toBeInTheDocument();
    expect(within(openPosition).getByText('103.00')).toBeInTheDocument();
  });

  it('blocks unrealistic per-share backtest slippage before calling the backend', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [{ symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const }],
    };
    const backend = mockBackend(state);
    backend.runBacktest = vi.fn(async () => backtestReport());

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    await userEvent.click(within(screen.getByLabelText('Workspace sections')).getByRole('button', { name: /Backtest/i }));
    const backtest = screen.getByLabelText('Backtest details');
    fireEvent.change(within(backtest).getByLabelText(/Slippage/i), { target: { value: '100' } });
    await userEvent.click(within(backtest).getByRole('button', { name: 'Run Backtest' }));

    expect(await screen.findByText('Slippage per share must be $1.00 or less.')).toBeInTheDocument();
    expect(backend.runBacktest).not.toHaveBeenCalled();
  });

  it('shows the completed backtest request summary and keeps share quantity read-only', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [{ symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const }],
    };
    const backend = mockBackend(state);
    backend.runBacktest = vi.fn(async () => backtestReport());

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    await userEvent.click(within(screen.getByLabelText('Workspace sections')).getByRole('button', { name: /Backtest/i }));
    const backtest = screen.getByLabelText('Backtest workspace');
    fireEvent.change(within(backtest).getByLabelText('Start date'), { target: { value: '2026-05-28' } });
    fireEvent.change(within(backtest).getByLabelText('End date'), { target: { value: '2026-05-29' } });
    fireEvent.change(within(backtest).getByLabelText('Start time'), { target: { value: '10:00' } });
    fireEvent.change(within(backtest).getByLabelText('End time'), { target: { value: '10:30' } });
    await userEvent.click(within(backtest).getByRole('button', { name: 'Run Backtest' }));

    const request = await within(backtest).findByLabelText('Backtest request summary');
    expect(within(request).getByText('Symbol')).toBeInTheDocument();
    expect(within(request).getByText('NVDA')).toBeInTheDocument();
    expect(within(request).getByText('Timeframe')).toBeInTheDocument();
    expect(within(request).getByText('5m')).toBeInTheDocument();
    expect(within(request).getByText('Date range')).toBeInTheDocument();
    expect(within(request).getByText('2026-05-28 to 2026-05-29')).toBeInTheDocument();
    expect(within(request).getByText('Time window')).toBeInTheDocument();
    expect(within(request).getByText('10:00-10:30')).toBeInTheDocument();
    expect(within(request).getByText('100 shares')).toBeInTheDocument();
    expect(within(request).getByText('$0.01')).toBeInTheDocument();
    expect(within(request).getByText('$1.00')).toBeInTheDocument();
    expect(within(backtest).queryByRole('spinbutton', { name: /share quantity/i })).not.toBeInTheDocument();
  });

  it('labels a completed backtest report as previous input results after controls change', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [{ symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const }],
    };
    const backend = mockBackend(state);
    backend.runBacktest = vi.fn(async () => backtestReport());

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    await userEvent.click(within(screen.getByLabelText('Workspace sections')).getByRole('button', { name: /Backtest/i }));
    const backtest = screen.getByLabelText('Backtest workspace');
    await userEvent.click(within(backtest).getByRole('button', { name: 'Run Backtest' }));
    expect(await within(backtest).findAllByText('$297.00')).toHaveLength(2);

    fireEvent.change(within(backtest).getByLabelText('Start time'), { target: { value: '10:00' } });

    expect(within(backtest).getByText('Results from previous inputs. Run Backtest again to update this report.')).toBeInTheDocument();
    expect(within(backtest).getAllByText('$297.00')).toHaveLength(2);
  });

  it('keeps the previous backtest report visible while a new run is loading', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [{ symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const }],
    };
    const backend = mockBackend(state);
    const pending = deferred<BacktestReport>();
    backend.runBacktest = vi
      .fn<BackendAPI['runBacktest']>()
      .mockResolvedValueOnce(backtestReport())
      .mockReturnValueOnce(pending.promise);

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    await userEvent.click(within(screen.getByLabelText('Workspace sections')).getByRole('button', { name: /Backtest/i }));
    const backtest = screen.getByLabelText('Backtest workspace');
    await userEvent.click(within(backtest).getByRole('button', { name: 'Run Backtest' }));
    expect(await within(backtest).findAllByText('$297.00')).toHaveLength(2);

    fireEvent.change(within(backtest).getByLabelText('Start time'), { target: { value: '10:00' } });
    await userEvent.click(within(backtest).getByRole('button', { name: 'Run Backtest' }));

    expect(within(backtest).getByRole('button', { name: 'Running' })).toBeDisabled();
    expect(within(backtest).getByText('Running backtest. Previous report remains visible.')).toBeInTheDocument();
    expect(within(backtest).getAllByText('$297.00')).toHaveLength(2);

    await act(async () => {
      pending.resolve(backtestReport());
      await pending.promise;
    });
  });

  it('shows live backtest progress while a run is loading', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [{ symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const }],
    };
    const backend = mockBackend(state);
    const pending = deferred<BacktestReport>();
    backend.runBacktest = vi.fn<BackendAPI['runBacktest']>().mockReturnValue(pending.promise);
    const listeners = new Map<string, (payload: unknown) => void>();
    window.runtime = {
      EventsOn: vi.fn((eventName: string, callback: (payload: unknown) => void) => {
        listeners.set(eventName, callback);
        return vi.fn();
      }),
    };

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    await userEvent.click(within(screen.getByLabelText('Workspace sections')).getByRole('button', { name: /Backtest/i }));
    const backtest = screen.getByLabelText('Backtest workspace');
    await userEvent.click(within(backtest).getByRole('button', { name: 'Run Backtest' }));

    act(() => {
      listeners.get('backtest:progress')?.({
        symbol: 'NVDA',
        stage: 'analyzing',
        processed_bars: 12,
        total_bars: 30,
        message: 'Analyzing bar 13 of 30',
      } satisfies BacktestProgress);
    });

    const progress = within(backtest).getByRole('progressbar', { name: 'Backtest progress' });
    expect(progress).toHaveAttribute('aria-valuenow', '40');
    expect(within(backtest).getByText('Analyzing bar 13 of 30')).toBeInTheDocument();
    expect(within(backtest).getByText('12 / 30 bars')).toBeInTheDocument();

    await act(async () => {
      pending.resolve(backtestReport());
      await pending.promise;
    });
  });

  it('shows a scoped backtest error and preserves the previous report after a failed run', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [{ symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const }],
    };
    const backend = mockBackend(state);
    backend.runBacktest = vi
      .fn<BackendAPI['runBacktest']>()
      .mockResolvedValueOnce(backtestReport())
      .mockRejectedValueOnce(new Error('Backtest data unavailable'));

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    await userEvent.click(within(screen.getByLabelText('Workspace sections')).getByRole('button', { name: /Backtest/i }));
    const backtest = screen.getByLabelText('Backtest workspace');
    await userEvent.click(within(backtest).getByRole('button', { name: 'Run Backtest' }));
    expect(await within(backtest).findAllByText('$297.00')).toHaveLength(2);

    await userEvent.click(within(backtest).getByRole('button', { name: 'Run Backtest' }));

    expect(await within(backtest).findByRole('alert')).toHaveTextContent('Backtest data unavailable');
    expect(within(backtest).getAllByText('$297.00')).toHaveLength(2);
  });

  it('uses watchlist list rows as symbol selectors', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA', 'AAPL'] },
      symbols: [
        { symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const },
        { symbol: 'AAPL', market_data_status: 'ready', job_status: 'idle' as const },
      ],
    };

    render(<App backend={mockBackend(state)} />);
    await screen.findByText('IBKR AI Analysis');

    const controls = screen.getByLabelText('Connection and watchlist');
    await userEvent.click(within(controls).getByRole('button', { name: 'Select AAPL' }));

    expect(within(screen.getByLabelText('Selected symbol analysis')).getByRole('heading', { name: 'AAPL' })).toBeInTheDocument();
  });

  it('keeps analysis actions above the watchlist list and adds symbols from the trailing add row', async () => {
    const state: AppState = {
      ...emptyState,
      connection_status: 'connected',
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [{ symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const }],
    };
    const backend = mockBackend(state);

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    const watchlistDetails = screen.getByLabelText('Watchlist details');
    const actionBar = within(watchlistDetails).getByLabelText('Pinned watchlist actions');
    expect(within(actionBar).getByRole('button', { name: 'Analyze NVDA' })).toBeEnabled();
    expect(within(actionBar).getByRole('button', { name: 'Screenshot Analyze' })).toBeEnabled();
    expect(within(actionBar).getByRole('button', { name: 'Resume Scheduled Analysis' })).toBeEnabled();
    expect(within(actionBar).getByRole('button', { name: 'Open Signals History' })).toBeEnabled();

    const symbolList = within(watchlistDetails).getByRole('list', { name: 'Watchlist symbols' });
    expect(within(symbolList).getByRole('button', { name: 'Select NVDA' })).toBeInTheDocument();
    await userEvent.click(within(symbolList).getByRole('button', { name: 'Add symbol' }));
    await userEvent.type(within(symbolList).getByLabelText('New watchlist symbol'), 'MSFT{enter}');
    await userEvent.click(within(watchlistDetails).getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(backend.saveSettings).toHaveBeenCalledWith({
        ...state.settings,
        watchlist: ['NVDA', 'MSFT'],
      }),
    );
  });

  it('shows current signals above history and skips duplicate latest history rows', async () => {
    const state: AppState = {
      ...emptyState,
      settings: { ...emptyState.settings, watchlist: ['NVDA', 'AAPL'] },
      symbols: [
        {
          symbol: 'NVDA',
          market_data_status: 'ready',
          job_status: 'complete' as const,
          current_price: 125.3,
          result: analysisResult('NVDA', 'long', 'Price reclaimed the prior high.'),
        },
        { symbol: 'AAPL', market_data_status: 'ready', job_status: 'idle' as const },
      ],
    };
    const backend = mockBackend(state);
    backend.getAnalysisHistory = vi.fn(async () => ({
      records: [
        { id: 7, result: analysisResult('NVDA', 'long', 'Price reclaimed the prior high.') },
        { id: 8, result: analysisResult('MSFT', 'short', 'Supply held above the opening range.') },
      ],
      total: 2,
      limit: 50,
      offset: 0,
    }));

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    const signals = screen.getByLabelText('Signals');
    await within(signals).findByRole('row', { name: /MSFT short 125.30/i });
    const rows = within(signals).getAllByRole('row').slice(1);

    expect(rows[0]).toHaveTextContent(/NVDA/);
    expect(rows[1]).toHaveTextContent(/AAPL/);
    expect(rows[2]).toHaveTextContent(/MSFT/);
    expect(within(signals).getAllByRole('row', { name: /NVDA long 125.30/i })).toHaveLength(1);
  });

  it('filters history without hiding current rows, opens a historical row, and exports CSV', async () => {
    const state: AppState = {
      ...emptyState,
      settings: { ...emptyState.settings, watchlist: ['NVDA', 'AAPL'] },
      symbols: [
        { symbol: 'NVDA', market_data_status: 'ready', job_status: 'complete' as const },
        { symbol: 'AAPL', market_data_status: 'ready', job_status: 'idle' as const },
      ],
    };
    const backend = mockBackend(state);
    backend.getAnalysisHistory = vi.fn(async (query) => ({
      records: [{ id: 7, result: analysisResult(query.symbol ?? 'MSFT', 'long', 'Price reclaimed the prior high.') }],
      total: 1,
      limit: 50,
      offset: 0,
    }));
    backend.exportAnalysisHistoryCSV = vi.fn(async () => '/tmp/signals-history.csv');

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');

    expect(await screen.findByRole('row', { name: /MSFT long 125.30/i })).toBeInTheDocument();

    await userEvent.clear(screen.getByLabelText('Symbol filter'));
    await userEvent.type(screen.getByLabelText('Symbol filter'), 'msft');

    await waitFor(() =>
      expect(backend.getAnalysisHistory).toHaveBeenLastCalledWith({
        symbol: 'MSFT',
        limit: 50,
        offset: 0,
      }),
    );

    const signals = screen.getByLabelText('Signals');
    expect(within(signals).getByRole('row', { name: /NVDA/i })).toBeInTheDocument();
    expect(within(signals).getByRole('row', { name: /AAPL/i })).toBeInTheDocument();

    await userEvent.click(screen.getByRole('row', { name: /MSFT long 125.30/i }));
    expect(within(screen.getByLabelText('Selected symbol analysis')).getByText('Price reclaimed the prior high.')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: 'Export CSV' }));
    await waitFor(() =>
      expect(backend.exportAnalysisHistoryCSV).toHaveBeenCalledWith({
        symbol: 'MSFT',
        limit: 50,
        offset: 0,
      }),
    );
  });

  it('opens the merged signals history from the watchlist action bar', async () => {
    const state: AppState = {
      ...emptyState,
      settings: { ...emptyState.settings, watchlist: ['NVDA'] },
      symbols: [{ symbol: 'NVDA', market_data_status: 'ready', job_status: 'idle' as const }],
    };
    const backend = mockBackend(state);
    backend.getAnalysisHistory = vi.fn(async () => ({
      records: [{ id: 7, result: analysisResult('MSFT', 'long', 'Price reclaimed the prior high.') }],
      total: 1,
      limit: 50,
      offset: 0,
    }));

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');
    const actionBar = within(screen.getByLabelText('Watchlist details')).getByLabelText('Pinned watchlist actions');
    await userEvent.click(within(actionBar).getByRole('button', { name: 'Open Signals History' }));

    expect(await screen.findByRole('row', { name: /MSFT long 125.30/i })).toBeInTheDocument();
    await waitFor(() => expect(backend.getAnalysisHistory).toHaveBeenCalledWith({ limit: 50, offset: 0 }));
  });

  it('loads more signal history when the merged list scrolls near the bottom', async () => {
    const backend = mockBackend(emptyState);
    backend.getAnalysisHistory = vi.fn(async (query) => ({
      records: [{ id: query.offset === 50 ? 11 : 10, result: analysisResult(query.offset === 50 ? 'AAPL' : 'NVDA', 'neutral', 'Range remains balanced.') }],
      total: 75,
      limit: 50,
      offset: query.offset ?? 0,
    }));
    backend.exportAnalysisHistoryCSV = vi.fn(async () => '/tmp/signals-history.csv');

    render(<App backend={backend} />);
    await screen.findByText('IBKR AI Analysis');
    expect(await screen.findByRole('row', { name: /NVDA neutral 125.30/i })).toBeInTheDocument();

    const signals = screen.getByLabelText('Signals');
    Object.defineProperty(signals, 'scrollHeight', { configurable: true, value: 1000 });
    Object.defineProperty(signals, 'clientHeight', { configurable: true, value: 500 });
    Object.defineProperty(signals, 'scrollTop', { configurable: true, value: 460 });
    fireEvent.scroll(signals);

    await waitFor(() =>
      expect(backend.getAnalysisHistory).toHaveBeenLastCalledWith({
        limit: 50,
        offset: 50,
      }),
    );
    expect(await screen.findByRole('row', { name: /AAPL neutral 125.30/i })).toBeInTheDocument();
  });
});

function mockBackend(state: AppState): BackendAPI {
  return {
    getState: vi.fn(async () => state),
    saveSettings: vi.fn(async () => state),
    listChartWindows: vi.fn(async () => []),
    connectIBKR: vi.fn(async (): Promise<AppState> => ({ ...state, connection_status: 'connected' })),
    disconnectIBKR: vi.fn(async (): Promise<AppState> => ({ ...state, connection_status: 'disconnected' })),
    setScheduledAnalysisEnabled: vi.fn(async (enabled): Promise<AppState> => ({ ...state, scheduled_analysis_enabled: enabled })),
    runAnalysisNow: vi.fn(async () => state),
    runScreenshotAnalysis: vi.fn(async () => state),
    runBacktest: vi.fn(async () => backtestReport()),
    getAnalysisHistory: vi.fn(async () => ({ records: [], total: 0, limit: 50, offset: 0 })),
    exportAnalysisHistoryCSV: vi.fn(async () => ''),
  };
}

function analysisResult(symbol: string, direction: 'long' | 'short' | 'neutral', summary: string): AnalysisResult {
  return {
    symbol,
    timeframe: '5m' as const,
    current_price: 125.3,
    stale: false,
    updated_at: '2026-05-30T18:36:10Z',
    output: {
      direction,
      setup_quality: direction === 'neutral' ? 'none' : 'a_plus',
      entry_zone: direction === 'neutral' ? null : { low: 125.1, high: 125.5 },
      stop_loss: direction === 'neutral' ? null : 124.2,
      take_profit: direction === 'neutral' ? [] : [126.4, 127.2],
      risk_reward: direction === 'neutral' ? null : 2.1,
      confidence: 0.68,
      market_regime: direction === 'neutral' ? '震荡' : '趋势回踩',
      trade_thesis: direction === 'neutral' ? '' : '关键位置回踩后重新出现主动买盘。',
      counterargument: '如果重新跌回区间中部，方向优势会消失。',
      no_trade_reason: direction === 'neutral' ? '区间中部没有 A+ 触发。' : '',
      rejection_reasons: direction === 'neutral' ? ['区间中部'] : [],
      summary,
      price_action: ['Breakout retest held'],
      invalidated_if: 'A 5m candle closes below 124.2',
      generated_at: '2026-05-30T18:36:10Z',
    },
  };
}

function backtestReport(): BacktestReport {
  return {
    symbol: 'NVDA',
    date: '2026-05-29',
    start_date: '2026-05-28',
    end_date: '2026-05-29',
    timeframe: '5m',
    share_quantity: 100,
    slippage_per_share: 0.01,
    commission_per_order: 1,
    start_time: '2026-05-29T13:30:00Z',
    end_time: '2026-05-29T20:00:00Z',
    bar_count: 3,
    trade_count: 1,
    winning_trades: 1,
    losing_trades: 0,
    win_rate_pct: 100,
    total_gross_pnl: 300,
    total_commission: 3,
    total_net_pnl: 297,
    total_return_pct: 2.97,
    max_drawdown: 0,
    generated_at: '2026-05-30T00:00:00Z',
    open_position: {
      symbol: 'NVDA',
      direction: 'long',
      setup_quality: 'a_plus',
      signal_time: '2026-05-29T19:55:00Z',
      entry_time: '2026-05-29T20:00:00Z',
      entry_price: 100,
      mark_time: '2026-05-29T20:00:00Z',
      mark_price: 103,
      shares: 50,
      remaining_shares: 50,
      initial_shares: 100,
      partial_taken: true,
      initial_stop_loss: 98,
      stop_loss: 100,
      take_profit: 104,
      realized_gross_pnl: 100,
      realized_commission: 1,
      realized_net_pnl: 99,
      unrealized_gross_pnl: 150,
      commission: 2,
      unrealized_net_pnl: 149,
      unrealized_return_pct: 2.98,
      market_regime: '趋势回踩',
      trade_thesis: 'Partial has paid, remainder is protected at breakeven.',
      counterargument: 'Target could still fail if price rejects supply.',
      no_trade_reason: '',
      rejection_reasons: [],
      summary: 'Holding into the next session.',
      confidence: 0.66,
      invalidated_if: 'Close below 103',
    },
    skipped_setups: [
      {
        time: '2026-05-29T14:05:00Z',
        direction: 'long',
        setup_quality: 'b',
        reason: 'not_a_plus',
        market_regime: '震荡',
        trade_thesis: 'Good idea, but still inside the range.',
        counterargument: 'Middle of the range can rotate both ways.',
        no_trade_reason: '不是 A+，位置不够好。',
        rejection_reasons: ['区间中部'],
        summary: 'Good idea, but still inside the range.',
        confidence: 0.54,
      },
      {
        time: '2026-05-29T14:15:00Z',
        direction: 'short',
        setup_quality: 'a_plus',
        reason: 'cooldown',
        market_regime: '趋势回踩',
        trade_thesis: 'A+ short appeared during loss cooldown.',
        counterargument: 'Cooldown protects from revenge trading.',
        no_trade_reason: '亏损后冷却中。',
        rejection_reasons: ['cooldown'],
        summary: 'Cooling down.',
        confidence: 0.7,
      },
    ],
    skip_reason_counts: {
      not_a_plus: 1,
      cooldown: 1,
    },
    trades: [
      {
        symbol: 'NVDA',
        direction: 'long',
        setup_quality: 'a_plus',
        signal_time: '2026-05-29T13:35:00Z',
        entry_time: '2026-05-29T13:35:00Z',
        entry_price: 100,
        exit_time: '2026-05-29T13:45:00Z',
        exit_price: 104,
        exit_reason: 'take_profit',
        exit_legs: [
          {
            time: '2026-05-29T13:40:00Z',
            price: 102,
            reason: 'partial_1r',
            shares: 50,
            gross_pnl: 100,
            commission: 1,
            net_pnl: 99,
            return_pct: 1.98,
          },
          {
            time: '2026-05-29T13:45:00Z',
            price: 104,
            reason: 'take_profit',
            shares: 50,
            gross_pnl: 200,
            commission: 1,
            net_pnl: 199,
            return_pct: 3.98,
          },
        ],
        shares: 100,
        initial_stop_loss: 98,
        stop_loss: 98,
        take_profit: 104,
        gross_pnl: 300,
        commission: 3,
        net_pnl: 297,
        return_pct: 2.97,
        market_regime: '趋势回踩',
        trade_thesis: 'Price reclaimed support and has room to target.',
        counterargument: 'A failed breakout would trap late longs.',
        no_trade_reason: '',
        rejection_reasons: [],
        summary: 'Held support.',
        confidence: 0.7,
        invalidated_if: 'Close below 98',
      },
    ],
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((innerResolve, innerReject) => {
    resolve = innerResolve;
    reject = innerReject;
  });
  return { promise, resolve, reject };
}
