import { useEffect, useMemo, useRef, useState, type ClipboardEvent, type KeyboardEvent, type RefObject } from 'react';
import { symbolsFromWatchlist } from '../state/appState';
import type {
  AnalysisHistoryPage,
  AnalysisHistoryQuery,
  AnalysisHistoryRecord,
  AppState,
  SymbolState,
} from '../types/domain';
import { AnalysisDetail } from './AnalysisDetail';
import { SignalList } from './SignalList';

interface Props {
  state: AppState;
  watchlistText: string;
  selected?: SymbolState;
  detailSymbol?: SymbolState;
  selectedHistoryId?: number;
  signalListRef: RefObject<HTMLElement>;
  historyPage: AnalysisHistoryPage;
  historyQuery: AnalysisHistoryQuery;
  historyLoading: boolean;
  onWatchlistTextChange(value: string): void;
  onSaveSettings(): void;
  onSetScheduledAnalysisEnabled(enabled: boolean): void;
  onAnalyze(): void;
  onAnalyzeSymbol(symbol: string): void;
  onScreenshotAnalyze(symbol: string): void;
  onOpenHistory(): void;
  onSelectSymbol(symbol: string): void;
  onHistoryQueryChange(query: AnalysisHistoryQuery): void;
  onSelectHistory(record: AnalysisHistoryRecord): void;
  onExportHistory(): void;
  onLoadMoreHistory(): void;
}

export function WatchlistWorkspace({
  state,
  watchlistText,
  selected,
  detailSymbol,
  selectedHistoryId,
  signalListRef,
  historyPage,
  historyQuery,
  historyLoading,
  onWatchlistTextChange,
  onSaveSettings,
  onSetScheduledAnalysisEnabled,
  onAnalyze,
  onAnalyzeSymbol,
  onScreenshotAnalyze,
  onOpenHistory,
  onSelectSymbol,
  onHistoryQueryChange,
  onSelectHistory,
  onExportHistory,
  onLoadMoreHistory,
}: Props) {
  const settings = state.settings;
  const connected = state.connection_status === 'connected';
  const selectedSymbol = selected?.symbol;
  const canAnalyzeSelected = connected && Boolean(selectedSymbol);
  const canScreenshotAnalyze = connected && Boolean(selectedSymbol);
  const [draftSymbol, setDraftSymbol] = useState('');
  const [isAddingSymbol, setIsAddingSymbol] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const watchlistSymbols = useMemo(() => symbolsFromWatchlist(watchlistText), [watchlistText]);
  const savedWatchlist = useMemo(() => symbolsFromWatchlist(settings.watchlist.join(',')), [settings.watchlist]);
  const savedSymbolSet = useMemo(() => new Set(state.symbols.map((symbol) => symbol.symbol)), [state.symbols]);
  const symbolStatusByName = useMemo(() => new Map(state.symbols.map((symbol) => [symbol.symbol, symbol])), [state.symbols]);
  const hasWatchlistChanges = watchlistSymbols.join(',') !== savedWatchlist.join(',');
  const selectedStatus = state.symbols.find((symbol) => symbol.symbol === selectedSymbol);

  useEffect(() => {
    if (isAddingSymbol) {
      inputRef.current?.focus();
    }
  }, [isAddingSymbol]);

  function setWatchlistSymbols(symbols: string[]) {
    onWatchlistTextChange(symbols.join(', '));
  }

  function addSymbols(raw: string) {
    const additions = symbolsFromWatchlist(raw);
    if (additions.length === 0) return;
    setWatchlistSymbols(symbolsFromWatchlist([...watchlistSymbols, ...additions].join(',')));
    setDraftSymbol('');
    setIsAddingSymbol(false);
  }

  function removeSymbol(symbol: string) {
    setWatchlistSymbols(watchlistSymbols.filter((item) => item !== symbol));
  }

  function handleDraftKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Escape') {
      setDraftSymbol('');
      setIsAddingSymbol(false);
      return;
    }
    if (['Enter', ',', ';', 'Tab'].includes(event.key) && draftSymbol.trim()) {
      event.preventDefault();
      addSymbols(draftSymbol);
    }
  }

  function handleDraftPaste(event: ClipboardEvent<HTMLInputElement>) {
    const pasted = event.clipboardData.getData('text');
    if (!/[,\n;\t]/.test(pasted)) return;
    event.preventDefault();
    addSymbols(pasted);
  }

  return (
    <section className="workspace-section watchlist-workspace" aria-label="Watchlist workspace">
      <section className="panel watchlist-controls" aria-label="Watchlist details">
        <div className="watchlist-control-surface" aria-label="Connection and watchlist">
          <div className="detail-section-heading">
            <div>
              <h2>Watchlist</h2>
              <p>{selectedSymbol ? `Selected ${selectedSymbol}` : 'Select one symbol to analyze.'}</p>
            </div>
            {selectedStatus ? <span className={`job-status ${selectedStatus.job_status}`}>{selectedStatus.job_status}</span> : null}
          </div>

          <div className="watchlist-pinned-actions" aria-label="Pinned watchlist actions">
            <div className="button-row watchlist-actions">
              <button type="button" className="primary" onClick={onAnalyze} disabled={!canAnalyzeSelected}>
                {selectedSymbol ? `Analyze ${selectedSymbol}` : 'Analyze'}
              </button>
              <button
                type="button"
                className="screenshot-action"
                onClick={() => selectedSymbol && onScreenshotAnalyze(selectedSymbol)}
                disabled={!canScreenshotAnalyze}
              >
                Screenshot Analyze
              </button>
              <button
                type="button"
                className={state.scheduled_analysis_enabled ? 'scheduled-action' : 'primary scheduled-action'}
                onClick={() => onSetScheduledAnalysisEnabled(!state.scheduled_analysis_enabled)}
              >
                {state.scheduled_analysis_enabled ? 'Pause Scheduled Analysis' : 'Resume Scheduled Analysis'}
              </button>
              <button type="button" className="history-action" aria-label="Open Signals History" onClick={onOpenHistory}>
                History
              </button>
            </div>
          </div>

          <div className="settings-group" aria-label="Watchlist editor">
            <div className="section-heading">
              <h2>Symbols</h2>
              <div className="section-heading-actions">
                <span>{watchlistSymbols.length} symbols</span>
                <button type="button" onClick={onSaveSettings}>
                  Save
                </button>
              </div>
            </div>
            <div className="watchlist-symbol-list" role="list" aria-label="Watchlist symbols">
              {watchlistSymbols.map((symbol) => {
                const isSelected = selectedSymbol === symbol;
                const canSelect = savedSymbolSet.has(symbol);
                const status = symbolStatusByName.get(symbol);
                return (
                  <div className={`watchlist-symbol-item ${isSelected ? 'selected' : ''}`} role="listitem" key={symbol}>
                    <button
                      type="button"
                      className="watchlist-symbol-select"
                      aria-label={`Select ${symbol}`}
                      aria-pressed={isSelected}
                      disabled={!canSelect}
                      onClick={() => onSelectSymbol(symbol)}
                    >
                      <strong>{symbol}</strong>
                      <span>{status?.market_data_status ?? 'Unsaved'}</span>
                      <small>{status?.last_closed_bar_time ? new Date(status.last_closed_bar_time).toLocaleTimeString() : 'No closed bar'}</small>
                    </button>
                    <button type="button" className="watchlist-symbol-remove" aria-label={`Remove ${symbol}`} onClick={() => removeSymbol(symbol)}>
                      &times;
                    </button>
                  </div>
                );
              })}
              <div className="watchlist-add-item" role="listitem">
                {isAddingSymbol ? (
                  <form
                    className="watchlist-add-form"
                    onSubmit={(event) => {
                      event.preventDefault();
                      addSymbols(draftSymbol);
                    }}
                  >
                    <input
                      id="watchlist-symbol-input"
                      ref={inputRef}
                      aria-label="New watchlist symbol"
                      value={draftSymbol}
                      placeholder={watchlistSymbols.length === 0 ? 'Add symbol or paste a list' : 'Add symbol'}
                      onChange={(event) => setDraftSymbol(event.target.value.toUpperCase())}
                      onKeyDown={handleDraftKeyDown}
                      onPaste={handleDraftPaste}
                    />
                    <button type="submit" disabled={!symbolsFromWatchlist(draftSymbol).length}>
                      Add
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        setDraftSymbol('');
                        setIsAddingSymbol(false);
                      }}
                    >
                      Cancel
                    </button>
                  </form>
                ) : (
                  <button type="button" className="watchlist-add-button" onClick={() => setIsAddingSymbol(true)}>
                    Add symbol
                  </button>
                )}
              </div>
            </div>
            {hasWatchlistChanges ? <p className="pending-note">Unsaved watchlist changes</p> : null}
          </div>

          <div className="settings-group symbol-status-group" aria-label="Symbol data status">
            <div className="section-heading">
              <h2>Status</h2>
            </div>
            <div className="symbol-status-list">
              {state.symbols.length === 0 ? <p className="empty-copy">Save a watchlist to load market data.</p> : null}
              {state.symbols.map((symbol) => (
                <button
                  type="button"
                  className={`symbol-status ${selectedSymbol === symbol.symbol ? 'selected' : ''}`}
                  key={symbol.symbol}
                  onClick={() => onSelectSymbol(symbol.symbol)}
                >
                  <strong>{symbol.symbol}</strong>
                  <span>{symbol.market_data_status}</span>
                  <small>{symbol.last_closed_bar_time ? new Date(symbol.last_closed_bar_time).toLocaleTimeString() : 'No closed bar'}</small>
                </button>
              ))}
            </div>
          </div>
        </div>
      </section>

      <SignalList
        containerRef={signalListRef}
        symbols={state.symbols}
        selectedSymbol={selectedHistoryId ? undefined : selectedSymbol}
        historyPage={historyPage}
        historyQuery={historyQuery}
        selectedHistoryId={selectedHistoryId}
        historyLoading={historyLoading}
        onHistoryQueryChange={onHistoryQueryChange}
        onSelectCurrent={onSelectSymbol}
        onSelectHistory={onSelectHistory}
        onExportHistory={onExportHistory}
        onLoadMoreHistory={onLoadMoreHistory}
      />

      <AnalysisDetail
        symbol={detailSymbol}
        connected={connected}
        onAnalyze={onAnalyzeSymbol}
        onScreenshotAnalyze={onScreenshotAnalyze}
      />
    </section>
  );
}
