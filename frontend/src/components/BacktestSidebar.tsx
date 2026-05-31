import type {
  BacktestExitLeg,
  BacktestOpenPosition,
  BacktestProgress,
  BacktestReport,
  BacktestRequest,
  BacktestSkippedSetup,
  BacktestTrade,
} from '../types/domain';

interface Props {
  symbols: string[];
  symbol?: string;
  connected: boolean;
  startDate: string;
  endDate: string;
  startTime: string;
  endTime: string;
  slippage: string;
  commission: string;
  loading: boolean;
  progress?: BacktestProgress;
  error?: string;
  report?: BacktestReport;
  lastRequest?: BacktestRequest;
  reportFromPreviousInputs?: boolean;
  embedded?: boolean;
  onSymbolChange(symbol: string): void;
  onStartDateChange(value: string): void;
  onEndDateChange(value: string): void;
  onStartTimeChange(value: string): void;
  onEndTimeChange(value: string): void;
  onSlippageChange(value: string): void;
  onCommissionChange(value: string): void;
  onRun(): void;
}

export function BacktestSidebar({
  symbols,
  symbol,
  connected,
  startDate,
  endDate,
  startTime,
  endTime,
  slippage,
  commission,
  loading,
  progress,
  error,
  report,
  lastRequest,
  reportFromPreviousInputs = false,
  embedded = false,
  onSymbolChange,
  onStartDateChange,
  onEndDateChange,
  onStartTimeChange,
  onEndTimeChange,
  onSlippageChange,
  onCommissionChange,
  onRun,
}: Props) {
  const canRun = Boolean(symbol && connected && startDate && endDate && startTime && endTime && !loading);
  const Container = embedded ? 'section' : 'aside';

  return (
    <Container
      className="backtest-sidebar"
      aria-label={embedded ? 'Backtest details' : 'Backtest sidebar'}
    >
      <div className="panel backtest-run-panel">
        <div className="section-heading">
          <h2>Backtest</h2>
          <span>{symbol ? `${symbol} · 100 shares` : '100 shares'}</span>
        </div>

        <div className="backtest-controls">
          <label>
            Symbol
            <select value={symbol ?? ''} onChange={(event) => onSymbolChange(event.target.value)} disabled={loading || symbols.length === 0}>
              {symbols.length === 0 ? <option value="">No saved symbol</option> : null}
              {symbols.map((item) => (
                <option value={item} key={item}>
                  {item}
                </option>
              ))}
            </select>
          </label>
          <label>
            Start date
            <input type="date" value={startDate} max={endDate || undefined} onChange={(event) => onStartDateChange(event.target.value)} />
          </label>
          <label>
            End date
            <input type="date" value={endDate} min={startDate || undefined} onChange={(event) => onEndDateChange(event.target.value)} />
          </label>
          <label>
            Start time
            <input
              type="time"
              min="09:30"
              max="16:00"
              step="300"
              value={startTime}
              onChange={(event) => onStartTimeChange(event.target.value)}
            />
          </label>
          <label>
            End time
            <input
              type="time"
              min="09:30"
              max="16:00"
              step="300"
              value={endTime}
              onChange={(event) => onEndTimeChange(event.target.value)}
            />
          </label>
          <label>
            Slippage / share ($)
            <input
              type="number"
              min="0"
              max="1"
              step="0.01"
              value={slippage}
              onChange={(event) => onSlippageChange(event.target.value)}
            />
          </label>
          <label>
            Commission / order
            <input
              type="number"
              min="0"
              step="0.01"
              value={commission}
              onChange={(event) => onCommissionChange(event.target.value)}
            />
          </label>
          <button type="button" className="primary" onClick={onRun} disabled={!canRun}>
            {loading ? 'Running' : 'Run Backtest'}
          </button>
        </div>

        <p className="assumption-copy">Share quantity assumption: 100 shares</p>
        {loading ? <BacktestProgressView progress={progress} /> : null}
        {error ? <div className="error-box" role="alert">{error}</div> : null}
      </div>

      <div className="panel backtest-report-panel">
        {loading && report ? <div className="info-box">Running backtest. Previous report remains visible.</div> : null}
        {report ? (
          <BacktestReportView report={report} request={lastRequest} stale={reportFromPreviousInputs} />
        ) : (
          <p className="empty-copy">No backtest run.</p>
        )}
      </div>
    </Container>
  );
}

function BacktestProgressView({ progress }: { progress?: BacktestProgress }) {
  const percent = progress && progress.total_bars > 0 ? Math.round((progress.processed_bars / progress.total_bars) * 100) : undefined;
  const message = progress?.message ?? 'Starting backtest';
  const barCount = progress && progress.total_bars > 0 ? `${progress.processed_bars} / ${progress.total_bars} bars` : 'Preparing bars';

  return (
    <div className="backtest-progress" aria-live="polite">
      <div
        className="backtest-progress-track"
        role="progressbar"
        aria-label="Backtest progress"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={percent}
      >
        <span style={{ width: `${percent ?? 18}%` }} />
      </div>
      <div className="backtest-progress-copy">
        <span>{message}</span>
        <strong>{barCount}</strong>
      </div>
    </div>
  );
}

function BacktestReportView({
  report,
  request,
  stale,
}: {
  report: BacktestReport;
  request?: BacktestRequest;
  stale: boolean;
}) {
  return (
    <div className="backtest-report">
      {stale ? (
        <div className="warning-box">
          Results from previous inputs. Run Backtest again to update this report.
        </div>
      ) : null}
      <dl className="backtest-request-summary" aria-label="Backtest request summary">
        <RequestMetric label="Symbol" value={request?.symbol ?? report.symbol} />
        <RequestMetric label="Timeframe" value={report.timeframe} />
        <RequestMetric label="Date range" value={`${request?.start_date ?? report.start_date} to ${request?.end_date ?? report.end_date}`} />
        <RequestMetric label="Time window" value={`${request?.start_time ?? formatTime(report.start_time)}-${request?.end_time ?? formatTime(report.end_time)}`} />
        <RequestMetric label="Share quantity" value={`${request?.share_quantity ?? report.share_quantity} shares`} />
        <RequestMetric label="Slippage" value={formatCurrency(request?.slippage_per_share ?? report.slippage_per_share)} />
        <RequestMetric label="Commission" value={formatCurrency(request?.commission_per_order ?? report.commission_per_order)} />
      </dl>
      <dl className="backtest-summary">
        <div>
          <dt>Closed P&L</dt>
          <dd>{formatCurrency(report.total_net_pnl)}</dd>
        </div>
        <div>
          <dt>Return</dt>
          <dd>{formatPercent(report.total_return_pct)}</dd>
        </div>
        <div>
          <dt>Trades</dt>
          <dd>{report.trade_count}</dd>
        </div>
        <div>
          <dt>Win rate</dt>
          <dd>{formatPercent(report.win_rate_pct)}</dd>
        </div>
        <div>
          <dt>Max DD</dt>
          <dd>{formatCurrency(report.max_drawdown)}</dd>
        </div>
        <div>
          <dt>Bars</dt>
          <dd>{report.bar_count}</dd>
        </div>
        <div>
          <dt>Window</dt>
          <dd>{`${formatTime(report.start_time)}-${formatTime(report.end_time)} ET`}</dd>
        </div>
      </dl>
      {report.trades.length ? (
        <div className="backtest-trade-list">
          {report.trades.map((trade, index) => (
            <BacktestTradeCard key={`${trade.entry_time}-${trade.exit_time}-${trade.direction}-${index}`} index={index} trade={trade} />
          ))}
        </div>
      ) : (
        <p className="empty-copy">No simulated trades.</p>
      )}
      {report.skipped_setups?.length ? (
        <BacktestSkippedSetups setups={report.skipped_setups} counts={report.skip_reason_counts ?? {}} />
      ) : null}
      {report.open_position ? <BacktestOpenPositionCard position={report.open_position} /> : null}
    </div>
  );
}

function RequestMetric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

function BacktestTradeCard({ index, trade }: { index: number; trade: BacktestTrade }) {
  return (
    <article className="backtest-trade-card" aria-label={`Trade ${index + 1} ${trade.direction}`} title={trade.summary}>
      <header>
        <div>
          <span className={`direction ${trade.direction}`}>{trade.direction}</span>
          <span className="setup-quality">{formatSetupQuality(trade.setup_quality)}</span>
          <strong>{formatExitReason(trade.exit_reason)}</strong>
        </div>
        <span className={trade.net_pnl >= 0 ? 'trade-net positive' : 'trade-net negative'}>{formatCurrency(trade.net_pnl)}</span>
      </header>

      <dl className="trade-timeline">
        <TradeMetric label="Signal" value={formatTime(trade.signal_time)} />
        <TradeMetric label="Entry time" value={formatTime(trade.entry_time)} />
        <TradeMetric label="Exit time" value={formatTime(trade.exit_time)} />
      </dl>

      <dl className="trade-metrics">
        <TradeMetric label="Entry fill" value={formatNumber(trade.entry_price)} />
        <TradeMetric label="Exit fill" value={formatNumber(trade.exit_price)} />
        <TradeMetric label="Stop" value={formatNumber(trade.stop_loss)} />
        <TradeMetric label="Target" value={formatNumber(trade.take_profit)} />
        <TradeMetric label="Shares" value={String(trade.shares)} />
        <TradeMetric label="Gross" value={formatCurrency(trade.gross_pnl)} />
        <TradeMetric label="Fees" value={formatCurrency(trade.commission)} />
        <TradeMetric label="Return" value={formatPercent(trade.return_pct)} />
      </dl>

      {trade.exit_legs?.length ? (
        <section className="exit-leg-list" aria-label={`Trade ${index + 1} exit legs`}>
          <h3>Exit legs</h3>
          {trade.exit_legs.map((leg) => (
            <BacktestExitLegRow key={`${leg.time}-${leg.reason}-${leg.shares}`} leg={leg} />
          ))}
        </section>
      ) : null}

      {trade.trade_thesis ? <p className="trade-summary">{trade.trade_thesis}</p> : null}
      {trade.counterargument ? <p className="trade-invalidation">{trade.counterargument}</p> : null}
      <p className="trade-summary">{trade.summary}</p>
      <p className="trade-invalidation">{trade.invalidated_if}</p>
    </article>
  );
}

function BacktestExitLegRow({ leg }: { leg: BacktestExitLeg }) {
  return (
    <div className="exit-leg-row">
      <span>{formatExitReason(leg.reason)}</span>
      <strong>{`${leg.shares} @ ${formatNumber(leg.price)}`}</strong>
      <span className={leg.net_pnl >= 0 ? 'positive' : 'negative'}>{formatCurrency(leg.net_pnl)}</span>
    </div>
  );
}

function BacktestSkippedSetups({ setups, counts }: { setups: BacktestSkippedSetup[]; counts: Record<string, number> }) {
  return (
    <section className="backtest-skipped" aria-label="Skipped setups">
      <header>
        <h3>Skipped setups</h3>
        <div className="skip-counts">
          {Object.entries(counts).map(([reason, count]) => (
            <span key={reason}>{`${reason} ${count}`}</span>
          ))}
        </div>
      </header>
      <div className="skipped-list">
        {setups.map((setup) => (
          <article key={`${setup.time}-${setup.direction}-${setup.reason}`} className="skipped-setup">
            <div>
              <span className={`direction ${setup.direction}`}>{setup.direction}</span>
              <span className="setup-quality">{formatSetupQuality(setup.setup_quality)}</span>
              <strong>{setup.reason}</strong>
            </div>
            <p>{setup.summary || setup.no_trade_reason}</p>
            {setup.trade_thesis && setup.trade_thesis !== setup.summary ? <p>{setup.trade_thesis}</p> : null}
            {setup.counterargument ? <p className="trade-invalidation">{setup.counterargument}</p> : null}
          </article>
        ))}
      </div>
    </section>
  );
}

function BacktestOpenPositionCard({ position }: { position: BacktestOpenPosition }) {
  return (
    <article
      className="backtest-trade-card"
      aria-label={`Open position ${position.direction}`}
      title={position.summary}
    >
      <header>
        <div>
          <span className={`direction ${position.direction}`}>{position.direction}</span>
          <span className="setup-quality">{formatSetupQuality(position.setup_quality)}</span>
          <strong>Open Position</strong>
        </div>
        <span className={position.unrealized_net_pnl >= 0 ? 'trade-net positive' : 'trade-net negative'}>
          {formatCurrency(position.unrealized_net_pnl)}
        </span>
      </header>

      <dl className="trade-timeline">
        <TradeMetric label="Signal" value={formatTime(position.signal_time)} />
        <TradeMetric label="Entry time" value={formatTime(position.entry_time)} />
        <TradeMetric label="Mark time" value={formatTime(position.mark_time)} />
      </dl>

      <dl className="trade-metrics">
        <TradeMetric label="Entry fill" value={formatNumber(position.entry_price)} />
        <TradeMetric label="Mark" value={formatNumber(position.mark_price)} />
        <TradeMetric label="Stop" value={formatNumber(position.stop_loss)} />
        <TradeMetric label="Target" value={formatNumber(position.take_profit)} />
        <TradeMetric label="Shares" value={String(position.shares)} />
        <TradeMetric label="Remaining" value={String(position.remaining_shares ?? position.shares)} />
        <TradeMetric label="Partial" value={position.partial_taken ? 'Partial taken' : 'No partial'} />
        <TradeMetric label="Realized" value={formatCurrency(position.realized_net_pnl)} />
        <TradeMetric label="Unrealized" value={formatCurrency(position.unrealized_net_pnl)} />
        <TradeMetric label="Unrealized gross" value={formatCurrency(position.unrealized_gross_pnl)} />
        <TradeMetric label="Fees" value={formatCurrency(position.commission)} />
        <TradeMetric label="Return" value={formatPercent(position.unrealized_return_pct)} />
      </dl>

      {position.trade_thesis ? <p className="trade-summary">{position.trade_thesis}</p> : null}
      {position.counterargument ? <p className="trade-invalidation">{position.counterargument}</p> : null}
      <p className="trade-summary">{position.summary}</p>
      <p className="trade-invalidation">{position.invalidated_if}</p>
    </article>
  );
}

function TradeMetric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

function formatNumber(value?: number | null): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-';
  return value.toFixed(2);
}

function formatCurrency(value?: number | null): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-';
  return `${value < 0 ? '-' : ''}$${Math.abs(value).toFixed(2)}`;
}

function formatPercent(value?: number | null): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-';
  return `${value.toFixed(2)}%`;
}

function formatExitReason(value: string): string {
  switch (value) {
    case 'take_profit':
      return 'Take profit';
    case 'stop_loss':
      return 'Stop loss';
    case 'end_of_day':
      return 'End of day';
    case 'partial_1r':
      return '1R partial';
    case 'breakeven':
      return 'Breakeven';
    default:
      return value;
  }
}

function formatSetupQuality(value?: string): string {
  if (value === 'a_plus') return 'A+';
  if (!value || value === 'none') return 'No trade';
  return value.toUpperCase();
}

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', timeZone: 'America/New_York' });
}
