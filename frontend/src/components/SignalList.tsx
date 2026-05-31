import type { RefObject, UIEvent } from 'react';
import type { AnalysisHistoryPage, AnalysisHistoryQuery, AnalysisHistoryRecord, AnalysisResult, Direction, SymbolState, Timeframe } from '../types/domain';

interface Props {
  containerRef: RefObject<HTMLElement>;
  symbols: SymbolState[];
  selectedSymbol?: string;
  historyPage: AnalysisHistoryPage;
  historyQuery: AnalysisHistoryQuery;
  selectedHistoryId?: number;
  historyLoading?: boolean;
  onHistoryQueryChange(query: AnalysisHistoryQuery): void;
  onSelectCurrent(symbol: string): void;
  onSelectHistory(record: AnalysisHistoryRecord): void;
  onExportHistory(): void;
  onLoadMoreHistory(): void;
}

const PAGE_SIZE = 50;
const LOAD_MORE_DISTANCE = 80;

export function SignalList({
  containerRef,
  symbols,
  selectedSymbol,
  historyPage,
  historyQuery,
  selectedHistoryId,
  historyLoading = false,
  onHistoryQueryChange,
  onSelectCurrent,
  onSelectHistory,
  onExportHistory,
  onLoadMoreHistory,
}: Props) {
  const currentKeys = new Set(symbols.flatMap((symbol) => (symbol.result ? [analysisKey(symbol.result)] : [])));
  const historyRecords = historyPage.records.filter((record) => !currentKeys.has(analysisKey(record.result)));
  const hasRows = symbols.length > 0 || historyRecords.length > 0;
  const canLoadMore = historyPage.offset + historyPage.limit < historyPage.total;

  function updateFilter(patch: Partial<AnalysisHistoryQuery>) {
    onHistoryQueryChange(compactQuery({ ...historyQuery, ...patch, offset: 0, limit: PAGE_SIZE }));
  }

  function handleScroll(event: UIEvent<HTMLElement>) {
    const element = event.currentTarget;
    const distanceFromBottom = element.scrollHeight - element.scrollTop - element.clientHeight;
    if (distanceFromBottom <= LOAD_MORE_DISTANCE && canLoadMore && !historyLoading) {
      onLoadMoreHistory();
    }
  }

  return (
    <section ref={containerRef} className={`panel signal-table signal-list${hasRows ? '' : ' empty-state'}`} aria-label="Signals" tabIndex={-1} onScroll={handleScroll}>
      <div className="signal-table-header">
        <h2>Signals</h2>
        <span className="signal-count">
          Current {symbols.length} / History {historyRecords.length}
        </span>
      </div>
      <div className="history-controls">
        <label>
          Symbol filter
          <input
            value={historyQuery.symbol ?? ''}
            onChange={(event) => updateFilter({ symbol: normalizeSymbol(event.target.value) })}
            placeholder="All"
          />
        </label>
        <label>
          Timeframe filter
          <select value={historyQuery.timeframe ?? ''} onChange={(event) => updateFilter({ timeframe: optionalValue<Timeframe>(event.target.value) })}>
            <option value="">All</option>
            <option value="1m">1m</option>
            <option value="5m">5m</option>
            <option value="15m">15m</option>
            <option value="1h">1h</option>
          </select>
        </label>
        <label>
          Direction filter
          <select value={historyQuery.direction ?? ''} onChange={(event) => updateFilter({ direction: optionalValue<Direction>(event.target.value) })}>
            <option value="">All</option>
            <option value="long">long</option>
            <option value="short">short</option>
            <option value="neutral">neutral</option>
          </select>
        </label>
        <button type="button" onClick={onExportHistory} disabled={historyLoading || historyPage.total === 0}>
          Export CSV
        </button>
      </div>

      {hasRows ? (
        <table>
          <thead>
            <tr>
              <th>Symbol</th>
              <th>Direction</th>
              <th>Price</th>
              <th>Pos Qty</th>
              <th>Pos Value</th>
              <th>Advisory Cap</th>
              <th>Max Sh.</th>
              <th>Sizing</th>
              <th>Timeframe</th>
              <th>Entry</th>
              <th>Stop</th>
              <th>Targets</th>
              <th>R:R</th>
              <th>Conf.</th>
              <th>Updated</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {symbols.map((symbol) => (
              <CurrentSignalRow
                key={`current-${symbol.symbol}`}
                symbol={symbol}
                selected={selectedSymbol === symbol.symbol}
                onSelect={() => onSelectCurrent(symbol.symbol)}
              />
            ))}
            {historyRecords.map((record) => (
              <HistorySignalRow
                key={`history-${record.id}`}
                record={record}
                selected={selectedHistoryId === record.id}
                onSelect={() => onSelectHistory(record)}
              />
            ))}
          </tbody>
        </table>
      ) : (
        <p>Enter a watchlist to begin.</p>
      )}

      <div className="history-status" aria-live="polite">
        {historyLoading ? 'Loading' : historyPage.total === 0 ? 'No history' : canLoadMore ? `Loaded ${historyPage.records.length} of ${historyPage.total}` : 'End of history'}
      </div>
    </section>
  );
}

function CurrentSignalRow({ symbol, selected, onSelect }: { symbol: SymbolState; selected: boolean; onSelect(): void }) {
  const output = symbol.result?.output;
  return (
    <tr className={selected ? 'selected' : ''} onClick={onSelect}>
      <td>{symbol.symbol}</td>
      <td className={`direction ${output?.direction ?? 'neutral'}`}>{output?.direction ?? '-'}</td>
      <td>{formatNumber(symbol.current_price ?? symbol.result?.current_price)}</td>
      <td>{formatNumber(currentPosition(symbol)?.quantity)}</td>
      <td>{formatUSD(currentPosition(symbol)?.market_value_usd)}</td>
      <td>{formatUSD(symbol.account_context?.sizing_envelope?.advisory_notional_cap_usd)}</td>
      <td>{formatInteger(symbol.account_context?.sizing_envelope?.advisory_max_shares)}</td>
      <td>{output?.position_management?.sizing_status ?? symbol.account_context?.sizing_envelope?.sizing_status ?? '-'}</td>
      <td>{symbol.result?.timeframe ?? '-'}</td>
      <td>{output?.entry_zone ? `${formatNumber(output.entry_zone.low)}-${formatNumber(output.entry_zone.high)}` : '-'}</td>
      <td>{formatNumber(output?.stop_loss ?? undefined)}</td>
      <td>{output?.take_profit?.length ? output.take_profit.map((target) => formatNumber(target)).join(', ') : '-'}</td>
      <td>{formatNumber(output?.risk_reward ?? undefined)}</td>
      <td>{output ? `${Math.round(output.confidence * 100)}%` : '-'}</td>
      <td>{symbol.result ? formatDate(symbol.result.updated_at) : '-'}</td>
      <td>
        <span className={`job-status ${symbol.job_status}`}>{symbol.job_status}</span>
        {symbol.result?.stale ? <span className="stale">stale</span> : null}
      </td>
    </tr>
  );
}

function HistorySignalRow({ record, selected, onSelect }: { record: AnalysisHistoryRecord; selected: boolean; onSelect(): void }) {
  const output = record.result.output;
  return (
    <tr className={selected ? 'selected' : ''} onClick={onSelect}>
      <td>{record.result.symbol}</td>
      <td className={`direction ${output.direction}`}>{output.direction}</td>
      <td>{formatNumber(record.result.current_price)}</td>
      <td>-</td>
      <td>-</td>
      <td>{formatUSD(output.position_management?.advisory_notional_cap_usd)}</td>
      <td>{formatInteger(output.position_management?.advisory_max_shares)}</td>
      <td>{output.position_management?.sizing_status ?? '-'}</td>
      <td>{record.result.timeframe}</td>
      <td>{output.entry_zone ? `${formatNumber(output.entry_zone.low)}-${formatNumber(output.entry_zone.high)}` : '-'}</td>
      <td>{formatNumber(output.stop_loss ?? undefined)}</td>
      <td>{output.take_profit.length ? output.take_profit.map((target) => formatNumber(target)).join(', ') : '-'}</td>
      <td>{formatNumber(output.risk_reward ?? undefined)}</td>
      <td>{Math.round(output.confidence * 100)}%</td>
      <td>{formatDate(record.result.updated_at)}</td>
      <td>
        <span className="history-badge">history</span>
      </td>
    </tr>
  );
}

function compactQuery(query: AnalysisHistoryQuery): AnalysisHistoryQuery {
  const next: AnalysisHistoryQuery = {
    limit: query.limit || PAGE_SIZE,
    offset: query.offset || 0,
  };
  if (query.symbol) next.symbol = query.symbol;
  if (query.timeframe) next.timeframe = query.timeframe;
  if (query.direction) next.direction = query.direction;
  return next;
}

function analysisKey(result: AnalysisResult): string {
  return `${result.symbol}|${result.timeframe}|${result.updated_at}`;
}

function optionalValue<T extends string>(value: string): T | undefined {
  return value ? (value as T) : undefined;
}

function normalizeSymbol(value: string): string | undefined {
  const symbol = value.trim().toUpperCase();
  return symbol || undefined;
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

function currentPosition(symbol: SymbolState) {
  return symbol.account_context?.positions.find((position) => position.symbol === symbol.symbol);
}

function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleString();
}
