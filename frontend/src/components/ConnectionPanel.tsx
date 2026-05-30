import type { AppState, Settings, Timeframe } from '../types/domain';

interface Props {
  state: AppState;
  watchlistText: string;
  onWatchlistTextChange(value: string): void;
  onSettingsChange(settings: Settings): void;
  onSaveSettings(): void;
  onConnect(): void;
  onDisconnect(): void;
  onAnalyze(): void;
}

const timeframes: Timeframe[] = ['1m', '5m', '15m', '1h'];

export function ConnectionPanel({
  state,
  watchlistText,
  onWatchlistTextChange,
  onSettingsChange,
  onSaveSettings,
  onConnect,
  onDisconnect,
  onAnalyze,
}: Props) {
  const settings = state.settings;
  const connected = state.connection_status === 'connected';

  function patch(patch: Partial<Settings>) {
    onSettingsChange({ ...settings, ...patch });
  }

  return (
    <aside className="panel connection-panel" aria-label="Connection and watchlist">
      <div className="panel-heading">
        <h1>IBKR AI Analysis</h1>
        <span className={`status-dot ${state.connection_status}`}>{state.connection_status}</span>
      </div>

      <label>
        Host
        <input value={settings.ibkr_host} onChange={(event) => patch({ ibkr_host: event.target.value })} />
      </label>
      <div className="field-row">
        <label>
          Port
          <input type="number" value={settings.ibkr_port} onChange={(event) => patch({ ibkr_port: Number(event.target.value) })} />
        </label>
        <label>
          Client ID
          <input
            type="number"
            value={settings.ibkr_client_id}
            onChange={(event) => patch({ ibkr_client_id: Number(event.target.value) })}
          />
        </label>
      </div>

      <label>
        Watchlist
        <textarea value={watchlistText} onChange={(event) => onWatchlistTextChange(event.target.value)} placeholder="NVDA, TSLA, AAPL" />
      </label>

      <div className="segmented" aria-label="Timeframe">
        {timeframes.map((timeframe) => (
          <button
            key={timeframe}
            className={settings.selected_timeframe === timeframe ? 'active' : ''}
            onClick={() => patch({ selected_timeframe: timeframe })}
          >
            {timeframe}
          </button>
        ))}
      </div>

      <div className="button-row">
        <button onClick={onSaveSettings}>Save</button>
        <button className="primary" onClick={connected ? onDisconnect : onConnect}>
          {connected ? 'Disconnect' : 'Connect'}
        </button>
        <button onClick={onAnalyze} disabled={!connected || state.symbols.length === 0}>
          Analyze
        </button>
      </div>

      <div className="symbol-status-list">
        {state.symbols.map((symbol) => (
          <div className="symbol-status" key={symbol.symbol}>
            <strong>{symbol.symbol}</strong>
            <span>{symbol.market_data_status}</span>
            <small>{symbol.last_closed_bar_time ? new Date(symbol.last_closed_bar_time).toLocaleTimeString() : 'No closed bar'}</small>
          </div>
        ))}
      </div>
    </aside>
  );
}
