import type { SymbolState } from '../types/domain';

interface Props {
  symbol?: SymbolState;
}

export function AnalysisDetail({ symbol }: Props) {
  if (!symbol) {
    return (
      <aside className="panel detail-panel">
        <h2>Details</h2>
        <p>Select a symbol.</p>
      </aside>
    );
  }

  const output = symbol.result?.output;
  return (
    <aside className="panel detail-panel" aria-label="Selected symbol analysis">
      <div className="detail-heading">
        <h2>{symbol.symbol}</h2>
        <span className={`job-status ${symbol.job_status}`}>{symbol.job_status}</span>
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
        </>
      ) : (
        <p>No analysis yet.</p>
      )}
    </aside>
  );
}

