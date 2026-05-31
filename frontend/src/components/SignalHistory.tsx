import type { AnalysisHistoryPage, AnalysisHistoryQuery, AnalysisHistoryRecord, Direction, Timeframe } from '../types/domain';
import { SignalHeader } from './SignalTable';

interface Props {
  page: AnalysisHistoryPage;
  query: AnalysisHistoryQuery;
  selectedId?: number;
  loading?: boolean;
  view: 'current' | 'history';
  onViewChange(view: 'current' | 'history'): void;
  onQueryChange(query: AnalysisHistoryQuery): void;
  onSelect(record: AnalysisHistoryRecord): void;
  onExport(): void;
}

const PAGE_SIZE = 50;

export function SignalHistory({
  page,
  query,
  selectedId,
  loading = false,
  view,
  onViewChange,
  onQueryChange,
  onSelect,
  onExport,
}: Props) {
  const pageStart = page.total === 0 ? 0 : page.offset + 1;
  const pageEnd = Math.min(page.offset + page.records.length, page.total);
  const canGoPrevious = page.offset > 0;
  const canGoNext = page.offset + page.limit < page.total;

  function updateFilter(patch: Partial<AnalysisHistoryQuery>) {
    onQueryChange(compactQuery({ ...query, ...patch, offset: 0, limit: PAGE_SIZE }));
  }

  return (
    <section className="panel signal-table signal-history" aria-label="Signal history">
      <SignalHeader view={view} onViewChange={onViewChange} />
      <div className="history-controls">
        <label>
          Symbol filter
          <input
            value={query.symbol ?? ''}
            onChange={(event) => updateFilter({ symbol: normalizeSymbol(event.target.value) })}
            placeholder="All"
          />
        </label>
        <label>
          Timeframe filter
          <select value={query.timeframe ?? ''} onChange={(event) => updateFilter({ timeframe: optionalValue<Timeframe>(event.target.value) })}>
            <option value="">All</option>
            <option value="1m">1m</option>
            <option value="5m">5m</option>
            <option value="15m">15m</option>
            <option value="1h">1h</option>
          </select>
        </label>
        <label>
          Direction filter
          <select value={query.direction ?? ''} onChange={(event) => updateFilter({ direction: optionalValue<Direction>(event.target.value) })}>
            <option value="">All</option>
            <option value="long">long</option>
            <option value="short">short</option>
            <option value="neutral">neutral</option>
          </select>
        </label>
        <button type="button" onClick={onExport} disabled={loading || page.total === 0}>
          Export CSV
        </button>
      </div>

      <table>
        <thead>
          <tr>
            <th>Symbol</th>
            <th>Direction</th>
            <th>Price</th>
            <th>Timeframe</th>
            <th>Entry</th>
            <th>Stop</th>
            <th>Targets</th>
            <th>Conf.</th>
            <th>Updated</th>
          </tr>
        </thead>
        <tbody>
          {page.records.map((record) => {
            const output = record.result.output;
            return (
              <tr key={record.id} className={selectedId === record.id ? 'selected' : ''} onClick={() => onSelect(record)}>
                <td>{record.result.symbol}</td>
                <td className={`direction ${output.direction}`}>{output.direction}</td>
                <td>{formatNumber(record.result.current_price)}</td>
                <td>{record.result.timeframe}</td>
                <td>{output.entry_zone ? `${formatNumber(output.entry_zone.low)}-${formatNumber(output.entry_zone.high)}` : '-'}</td>
                <td>{formatNumber(output.stop_loss ?? undefined)}</td>
                <td>{output.take_profit.length ? output.take_profit.map((target) => formatNumber(target)).join(', ') : '-'}</td>
                <td>{Math.round(output.confidence * 100)}%</td>
                <td>{formatDate(record.result.updated_at)}</td>
              </tr>
            );
          })}
        </tbody>
      </table>

      <div className="history-pagination">
        <span>
          {loading ? 'Loading' : `${pageStart}-${pageEnd} of ${page.total}`}
        </span>
        <div>
          <button
            type="button"
            aria-label="Previous history page"
            onClick={() => onQueryChange(compactQuery({ ...query, offset: Math.max(0, query.offset - PAGE_SIZE), limit: PAGE_SIZE }))}
            disabled={!canGoPrevious || loading}
          >
            Previous
          </button>
          <button
            type="button"
            aria-label="Next history page"
            onClick={() => onQueryChange(compactQuery({ ...query, offset: query.offset + PAGE_SIZE, limit: PAGE_SIZE }))}
            disabled={!canGoNext || loading}
          >
            Next
          </button>
        </div>
      </div>
    </section>
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

function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleString();
}
