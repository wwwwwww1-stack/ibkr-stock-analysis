import type { SymbolState } from '../types/domain';

interface Props {
  symbols: SymbolState[];
  selected?: string;
  onSelect(symbol: string): void;
}

export function SignalTable({ symbols, selected, onSelect }: Props) {
  if (symbols.length === 0) {
    return (
      <section className="panel signal-table empty-state">
        <h2>Signals</h2>
        <p>Enter a watchlist to begin.</p>
      </section>
    );
  }

  return (
    <section className="panel signal-table" aria-label="Batch signal table">
      <h2>Signals</h2>
      <table>
        <thead>
          <tr>
            <th>Symbol</th>
            <th>Direction</th>
            <th>Price</th>
            <th>Entry</th>
            <th>Stop</th>
            <th>Targets</th>
            <th>R:R</th>
            <th>Conf.</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {symbols.map((row) => {
            const output = row.result?.output;
            return (
              <tr key={row.symbol} className={selected === row.symbol ? 'selected' : ''} onClick={() => onSelect(row.symbol)}>
                <td>{row.symbol}</td>
                <td className={`direction ${output?.direction ?? 'neutral'}`}>{output?.direction ?? '-'}</td>
                <td>{formatNumber(row.current_price ?? row.result?.current_price)}</td>
                <td>{output?.entry_zone ? `${formatNumber(output.entry_zone.low)}-${formatNumber(output.entry_zone.high)}` : '-'}</td>
                <td>{formatNumber(output?.stop_loss ?? undefined)}</td>
                <td>{output?.take_profit?.length ? output.take_profit.map((target) => formatNumber(target)).join(', ') : '-'}</td>
                <td>{formatNumber(output?.risk_reward ?? undefined)}</td>
                <td>{output ? `${Math.round(output.confidence * 100)}%` : '-'}</td>
                <td>
                  <span className={`job-status ${row.job_status}`}>{row.job_status}</span>
                  {row.result?.stale ? <span className="stale">stale</span> : null}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </section>
  );
}

function formatNumber(value?: number | null): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-';
  return value.toFixed(2);
}

