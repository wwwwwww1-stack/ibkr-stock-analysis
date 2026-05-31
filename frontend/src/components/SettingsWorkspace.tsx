import type { AppState, ChartWindow, Settings, Timeframe } from '../types/domain';

interface Props {
  state: AppState;
  chartWindows: ChartWindow[];
  chartWindowsLoading: boolean;
  onSettingsChange(settings: Settings): void;
  onSaveSettings(): void;
  onConnect(): void;
  onDisconnect(): void;
  onRefreshAccountSnapshot(): void;
  onRefreshChartWindows(): void;
  onSelectChartWindow(window: ChartWindow): void;
}

const timeframes: Timeframe[] = ['1m', '5m', '15m', '1h'];

export function SettingsWorkspace({
  state,
  chartWindows,
  chartWindowsLoading,
  onSettingsChange,
  onSaveSettings,
  onConnect,
  onDisconnect,
  onRefreshAccountSnapshot,
  onRefreshChartWindows,
  onSelectChartWindow,
}: Props) {
  const settings = state.settings;
  const connected = state.connection_status === 'connected';

  function patch(patch: Partial<Settings>) {
    onSettingsChange({ ...settings, ...patch });
  }

  function chartWindowName(window: ChartWindow) {
    return window.title ? `${window.app_name} - ${window.title}` : window.app_name;
  }

  return (
    <section className="workspace-section settings-workspace" aria-label="Settings workspace">
      <section className="panel settings-panel" aria-label="Settings details">
        <div className="detail-section-heading">
          <div>
            <h2>Settings</h2>
            <p>Connection and analysis timeframe.</p>
          </div>
          <span className={`status-dot ${state.connection_status}`}>{state.connection_status}</span>
        </div>

        <div className="settings-group" aria-label="IBKR connection">
          <button type="button" className="primary connection-action" onClick={connected ? onDisconnect : onConnect}>
            {connected ? 'Disconnect' : 'Connect'}
          </button>
        </div>

        <div className="settings-group account-settings" aria-label="Account snapshot settings">
          <div className="section-heading">
            <h2>Account</h2>
            <button type="button" onClick={onRefreshAccountSnapshot} disabled={!connected || state.account_snapshot.status === 'loading'}>
              {state.account_snapshot.status === 'loading' ? 'Refreshing' : 'Refresh Account Snapshot'}
            </button>
          </div>
          <div className="account-status-row">
            <span className={`status-dot ${state.account_snapshot.status}`}>{state.account_snapshot.status}</span>
            {state.account_snapshot.error ? <span className="account-error">{state.account_snapshot.error}</span> : null}
          </div>
          {state.account_snapshot.snapshot ? (
            <dl className="account-metrics">
              <div>
                <dt>Cash</dt>
                <dd>{formatUSD(state.account_snapshot.snapshot.available_cash_usd)}</dd>
              </div>
              <div>
                <dt>Buying power</dt>
                <dd>{formatUSD(state.account_snapshot.snapshot.buying_power_usd)}</dd>
              </div>
              <div>
                <dt>Snapshot</dt>
                <dd>{formatDate(state.account_snapshot.snapshot.snapshot_at)}</dd>
              </div>
              <div>
                <dt>Positions</dt>
                <dd>{state.account_snapshot.snapshot.positions.length}</dd>
              </div>
            </dl>
          ) : (
            <p className="empty-copy">Account-aware sizing is unavailable until a read-only snapshot is ready.</p>
          )}
          <label>
            Maximum stock trade amount
            <input
              type="number"
              min="0"
              step="100"
              value={settings.max_stock_trade_amount_usd ?? ''}
              onChange={(event) => patch({ max_stock_trade_amount_usd: parseOptionalNumber(event.target.value) })}
              placeholder="Not configured"
            />
          </label>
        </div>

        <div className="settings-group" aria-label="Chart window settings">
          <div className="section-heading">
            <h2>Chart Window</h2>
            <button type="button" onClick={onRefreshChartWindows} disabled={chartWindowsLoading}>
              {chartWindowsLoading ? 'Refreshing' : 'Refresh'}
            </button>
          </div>
          <p className="window-target-summary">
            {settings.chart_window ? `Selected ${chartWindowName(settings.chart_window)}` : 'Select the chart app window used by screenshot analysis.'}
          </p>
          <div className="chart-window-list" role="list" aria-label="Chart windows">
            {chartWindowsLoading ? <p className="empty-copy">Loading windows.</p> : null}
            {!chartWindowsLoading && chartWindows.length === 0 ? <p className="empty-copy">No windows found.</p> : null}
            {chartWindows.map((window) => {
              const selected = settings.chart_window?.id === window.id;
              return (
                <button
                  type="button"
                  key={`${window.id}-${window.app_name}-${window.title ?? ''}`}
                  className={`chart-window-item ${selected ? 'selected' : ''}`}
                  aria-pressed={selected}
                  onClick={() => onSelectChartWindow(window)}
                >
                  <strong>{window.app_name}</strong>
                  <span>{window.title || 'Untitled window'}</span>
                </button>
              );
            })}
          </div>
        </div>

        <div className="settings-group" aria-label="Timeframe settings">
          <div className="section-heading">
            <h2>Timeframe</h2>
            <button type="button" onClick={onSaveSettings}>
              Save
            </button>
          </div>
          <div className="segmented" aria-label="Timeframe">
            {timeframes.map((timeframe) => (
              <button
                type="button"
                key={timeframe}
                className={settings.selected_timeframe === timeframe ? 'active' : ''}
                onClick={() => patch({ selected_timeframe: timeframe })}
              >
                {timeframe}
              </button>
            ))}
          </div>
        </div>
      </section>
    </section>
  );
}

function parseOptionalNumber(value: string): number | null {
  if (value.trim() === '') return null;
  return Number(value);
}

function formatUSD(value?: number | null): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-';
  return value.toLocaleString(undefined, { style: 'currency', currency: 'USD' });
}

function formatDate(value?: string): string {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleString();
}
