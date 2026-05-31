import type { AccountSnapshotContext, PositionManagementOutput, TimeframeContextSummary, SymbolState } from '../types/domain';

interface Props {
  symbol?: SymbolState;
  connected?: boolean;
  onAnalyze?(symbol: string): void;
  onScreenshotAnalyze?(symbol: string): void;
}

export function AnalysisDetail({
  symbol,
  connected = false,
  onAnalyze,
  onScreenshotAnalyze,
}: Props) {
  if (!symbol) {
    return (
      <aside className="panel detail-panel">
        <h2>Details</h2>
        <p>Select a symbol.</p>
      </aside>
    );
  }

  const output = symbol.result?.output;
  const contextSummaries = symbol.result?.context_summaries ?? [];
  return (
    <aside className="panel detail-panel" aria-label="Selected symbol analysis">
      <div className="detail-heading">
        <h2>{symbol.symbol}</h2>
        <div className="detail-actions">
          <button type="button" onClick={() => onAnalyze?.(symbol.symbol)} disabled={!connected || !onAnalyze}>
            Analyze
          </button>
          <button
            type="button"
            onClick={() => onScreenshotAnalyze?.(symbol.symbol)}
            disabled={!connected || !onScreenshotAnalyze}
          >
            Screenshot Analyze
          </button>
          <span className={`job-status ${symbol.job_status}`}>{symbol.job_status}</span>
        </div>
      </div>

      {symbol.error ? <div className="error-box">{symbol.error}</div> : null}
      {symbol.result?.stale ? <div className="warning-box">Previous valid result shown as stale.</div> : null}

      {output ? (
        <>
          <section>
            <h3>Summary</h3>
            <p>{output.summary}</p>
          </section>
          <section>
            <h3>Price Action</h3>
            <ul>
              {output.price_action.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </section>
          <section>
            <h3>Trade Context</h3>
            <dl>
              <div>
                <dt>Direction</dt>
                <dd>{output.direction}</dd>
              </div>
              <div>
                <dt>Invalidated if</dt>
                <dd>{output.invalidated_if}</dd>
              </div>
              <div>
                <dt>Generated</dt>
                <dd>{new Date(output.generated_at).toLocaleString()}</dd>
              </div>
            </dl>
          </section>
          {symbol.account_context ? <AccountContextPanel symbol={symbol.symbol} context={symbol.account_context} /> : null}
          {output.position_management ? <PositionManagementPanel output={output.position_management} /> : null}
          {contextSummaries.length ? (
            <section className="timeframe-context" aria-label="Multi-timeframe context">
              <h3>Multi-Timeframe Context</h3>
              <div className="timeframe-context-list">
                {contextSummaries.map((summary) => (
                  <TimeframeContextRow key={summary.timeframe} summary={summary} />
                ))}
              </div>
            </section>
          ) : null}
        </>
      ) : (
        <p>No analysis yet.</p>
      )}
    </aside>
  );
}

function AccountContextPanel({ symbol, context }: { symbol: string; context: AccountSnapshotContext }) {
  const position = context.positions.find((item) => item.symbol === symbol);
  const envelope = context.sizing_envelope;
  return (
    <section className="account-context-panel" aria-label="Account context">
      <h3>Account Context</h3>
      <dl>
        <div>
          <dt>Position</dt>
          <dd>{formatNumber(position?.quantity)}</dd>
        </div>
        <div>
          <dt>Average cost</dt>
          <dd>{formatNumber(position?.average_cost)}</dd>
        </div>
        <div>
          <dt>Market value</dt>
          <dd>{formatUSD(position?.market_value_usd)}</dd>
        </div>
        <div>
          <dt>Unrealized P&L</dt>
          <dd>{formatUSD(position?.unrealized_pnl_usd)}</dd>
        </div>
        <div>
          <dt>Direction</dt>
          <dd>{positionDirection(position?.quantity)}</dd>
        </div>
        <div>
          <dt>User cap</dt>
          <dd>{formatUSD(context.max_stock_trade_amount_usd)}</dd>
        </div>
        <div>
          <dt>Advisory cap</dt>
          <dd>{formatUSD(envelope?.advisory_notional_cap_usd)}</dd>
        </div>
        <div>
          <dt>Advisory max shares</dt>
          <dd>{formatInteger(envelope?.advisory_max_shares)}</dd>
        </div>
        <div>
          <dt>Estimated risk</dt>
          <dd>{formatUSD(envelope?.estimated_risk_usd)}</dd>
        </div>
      </dl>
    </section>
  );
}

function PositionManagementPanel({ output }: { output: PositionManagementOutput }) {
  return (
    <section className="position-management-panel" aria-label="Position management">
      <h3>Position Management</h3>
      <dl>
        <div>
          <dt>Action</dt>
          <dd>{output.advisory_action}</dd>
        </div>
        <div>
          <dt>Sizing</dt>
          <dd>{output.sizing_status}</dd>
        </div>
        <div>
          <dt>Review</dt>
          <dd>{output.manual_review_required ? 'Review manually' : '-'}</dd>
        </div>
        <div>
          <dt>Advisory max shares</dt>
          <dd>{formatInteger(output.advisory_max_shares)}</dd>
        </div>
        <div>
          <dt>Advisory cap</dt>
          <dd>{formatUSD(output.advisory_notional_cap_usd)}</dd>
        </div>
      </dl>
      <ul>
        {output.management_notes.map((note) => (
          <li key={note}>{note}</li>
        ))}
      </ul>
    </section>
  );
}

function TimeframeContextRow({ summary }: { summary: TimeframeContextSummary }) {
  if (!summary.available) {
    return (
      <article className="timeframe-context-row unavailable">
        <header>
          <strong>{summary.timeframe}</strong>
          <span>Unavailable</span>
        </header>
        <p>{summary.error || 'No Data'}</p>
      </article>
    );
  }

  const derived = summary.derived;
  return (
    <article className="timeframe-context-row">
      <header>
        <strong>{summary.timeframe}</strong>
        <span>{summary.bar_count} bars</span>
      </header>
      <dl>
        <div>
          <dt>Price</dt>
          <dd>{formatNumber(summary.current_price)}</dd>
        </div>
        <div>
          <dt>Range</dt>
          <dd>{derived ? `${formatNumber(derived.session_high)} / ${formatNumber(derived.session_low)}` : '-'}</dd>
        </div>
        <div>
          <dt>Swings</dt>
          <dd>
            {derived
              ? `${formatValues(derived.recent_swing_highs)} / ${formatValues(derived.recent_swing_lows)}`
              : '-'}
          </dd>
        </div>
        <div>
          <dt>ATR</dt>
          <dd>{formatNumber(derived?.atr)}</dd>
        </div>
        <div>
          <dt>Volume</dt>
          <dd>{derived?.volume_context ?? '-'}</dd>
        </div>
      </dl>
    </article>
  );
}

function formatNumber(value?: number | null): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-';
  return value.toFixed(2);
}

function formatInteger(value?: number | null): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-';
  return String(Math.trunc(value));
}

function formatUSD(value?: number | null): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-';
  return value.toLocaleString(undefined, { style: 'currency', currency: 'USD' });
}

function positionDirection(quantity?: number | null): string {
  if (typeof quantity !== 'number' || Number.isNaN(quantity) || quantity === 0) return '-';
  return quantity > 0 ? 'long' : 'short';
}

function formatValues(values?: number[]): string {
  if (!values?.length) return '-';
  return values.map((value) => formatNumber(value)).join(', ');
}
