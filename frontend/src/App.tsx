import { useEffect, useMemo, useRef, useState } from 'react';
import './App.css';
import { wailsBackend, subscribeBacktestProgress, subscribeStateEvents, type BackendAPI } from './api/backend';
import { BacktestWorkspace } from './components/BacktestWorkspace';
import { Disclaimer } from './components/Disclaimer';
import { SettingsWorkspace } from './components/SettingsWorkspace';
import { WatchlistWorkspace } from './components/WatchlistWorkspace';
import { WorkspaceNav, type WorkspaceSection } from './components/WorkspaceNav';
import { defaultState, mergeState, selectSymbol, symbolsFromWatchlist, updateSettings } from './state/appState';
import type {
  AnalysisHistoryPage,
  AnalysisHistoryQuery,
  AnalysisHistoryRecord,
  AppState,
  BacktestReport,
  BacktestProgress,
  BacktestRequest,
  ChartWindow,
  Settings,
  SymbolState,
} from './types/domain';

interface Props {
  backend?: BackendAPI;
}

const HISTORY_PAGE_SIZE = 50;
const MAX_BACKTEST_SLIPPAGE_PER_SHARE = 1;
const BACKTEST_SLIPPAGE_LIMIT_MESSAGE = 'Slippage per share must be $1.00 or less.';
const AUTO_CONNECT_RETRY_MS = 5000;

function App({ backend = wailsBackend }: Props) {
  const [state, setState] = useState<AppState>(defaultState());
  const [activeSection, setActiveSection] = useState<WorkspaceSection>('watchlist');
  const [watchlistText, setWatchlistText] = useState('');
  const [selectedSymbol, setSelectedSymbol] = useState<string | undefined>();
  const [historyQuery, setHistoryQuery] = useState<AnalysisHistoryQuery>({ limit: HISTORY_PAGE_SIZE, offset: 0 });
  const [historyPage, setHistoryPage] = useState<AnalysisHistoryPage>({ records: [], total: 0, limit: HISTORY_PAGE_SIZE, offset: 0 });
  const [historyLoading, setHistoryLoading] = useState(false);
  const [selectedHistory, setSelectedHistory] = useState<AnalysisHistoryRecord | undefined>();
  const [chartWindows, setChartWindows] = useState<ChartWindow[]>([]);
  const [chartWindowsLoading, setChartWindowsLoading] = useState(false);
  const [backtestStartDate, setBacktestStartDate] = useState(defaultBacktestDate());
  const [backtestEndDate, setBacktestEndDate] = useState(defaultBacktestDate());
  const [backtestStartTime, setBacktestStartTime] = useState('09:30');
  const [backtestEndTime, setBacktestEndTime] = useState('16:00');
  const [backtestSlippage, setBacktestSlippage] = useState('0.01');
  const [backtestCommission, setBacktestCommission] = useState('1.00');
  const [backtestSymbol, setBacktestSymbol] = useState<string | undefined>();
  const [backtestReport, setBacktestReport] = useState<BacktestReport | undefined>();
  const [lastBacktestRequest, setLastBacktestRequest] = useState<BacktestRequest | undefined>();
  const [backtestProgress, setBacktestProgress] = useState<BacktestProgress | undefined>();
  const [backtestLoading, setBacktestLoading] = useState(false);
  const [backtestError, setBacktestError] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>();
  const signalListRef = useRef<HTMLElement | null>(null);
  const connectionStatusRef = useRef(state.connection_status);
  const requestedChartWindows = useRef(false);

  useEffect(() => {
    connectionStatusRef.current = state.connection_status;
  }, [state.connection_status]);

  useEffect(() => {
    let active = true;
    let connected = false;
    let connecting = false;
    let retryTimer: number | undefined;

    function isConnected() {
      return connected || connectionStatusRef.current === 'connected';
    }

    function clearRetryTimer() {
      if (retryTimer === undefined) return;
      window.clearTimeout(retryTimer);
      retryTimer = undefined;
    }

    function markConnectionStatus(next: AppState) {
      connectionStatusRef.current = next.connection_status;
      if (next.connection_status !== 'connected') return;
      connected = true;
      clearRetryTimer();
      setError(undefined);
    }

    function scheduleRetry() {
      if (!active || isConnected() || retryTimer !== undefined) return;
      retryTimer = window.setTimeout(() => {
        retryTimer = undefined;
        void connectOnce();
      }, AUTO_CONNECT_RETRY_MS);
    }

    async function connectOnce() {
      if (!active || isConnected() || connecting) return;
      connecting = true;
      try {
        const next = await backend.connectIBKR();
        if (!active || isConnected()) return;
        markConnectionStatus(next);
        setState(next);
        if (next.connection_status !== 'connected') {
          scheduleRetry();
        }
      } catch (err) {
        if (!active || isConnected()) return;
        setError(err instanceof Error ? err.message : String(err));
        scheduleRetry();
      } finally {
        connecting = false;
      }
    }

    backend
      .getState()
      .then((next) => {
        if (!active) return;
        markConnectionStatus(next);
        setState(next);
        setWatchlistText(next.settings.watchlist.join(', '));
        setSelectedSymbol(next.symbols[0]?.symbol);
        if (next.connection_status !== 'connected') {
          void connectOnce();
        }
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => active && setLoading(false));
    const unsubscribe = subscribeStateEvents((next) => {
      markConnectionStatus(next);
      setState((current) => mergeState(current, next));
    });
    return () => {
      active = false;
      clearRetryTimer();
      unsubscribe();
    };
  }, [backend]);

  const selected = useMemo(() => selectSymbol(state, selectedSymbol), [state, selectedSymbol]);
  const watchlistSymbols = useMemo(() => symbolsFromWatchlist(watchlistText), [watchlistText]);
  const savedSymbols = useMemo(() => state.symbols.map((symbol) => symbol.symbol), [state.symbols]);
  const historyRefreshKey = useMemo(
    () => state.symbols.map((row) => `${row.symbol}:${row.result?.updated_at ?? ''}`).join('|'),
    [state.symbols],
  );
  const detailSymbol = selectedHistory ? historyRecordToSymbolState(selectedHistory) : selected;
  const backtestRequest = useMemo(
    () =>
      backtestSymbol
        ? buildBacktestRequest({
            symbol: backtestSymbol,
            startDate: backtestStartDate,
            endDate: backtestEndDate,
            startTime: backtestStartTime,
            endTime: backtestEndTime,
            slippage: backtestSlippage,
            commission: backtestCommission,
          })
        : undefined,
    [
      backtestSymbol,
      backtestStartDate,
      backtestEndDate,
      backtestStartTime,
      backtestEndTime,
      backtestSlippage,
      backtestCommission,
    ],
  );
  const backtestReportFromPreviousInputs = Boolean(
    backtestReport &&
      lastBacktestRequest &&
      backtestRequest &&
      !sameBacktestRequest(lastBacktestRequest, backtestRequest),
  );

  useEffect(() => {
    if (loading) return;
    let active = true;
    const query = compactHistoryQuery({ ...historyQuery, limit: HISTORY_PAGE_SIZE, offset: 0 });
    setHistoryLoading(true);
    backend
      .getAnalysisHistory(query)
      .then((page) => {
        if (!active) return;
        setHistoryPage(page);
      })
      .catch((err: Error) => active && setError(err.message))
      .finally(() => active && setHistoryLoading(false));
    return () => {
      active = false;
    };
  }, [
    backend,
    loading,
    historyQuery.symbol,
    historyQuery.timeframe,
    historyQuery.direction,
    historyRefreshKey,
  ]);

  useEffect(() => {
    if (activeSection !== 'backtest') return;
    if (backtestSymbol && savedSymbols.includes(backtestSymbol)) return;
    const fallback = selected?.symbol && savedSymbols.includes(selected.symbol) ? selected.symbol : savedSymbols[0];
    setBacktestSymbol(fallback);
  }, [activeSection, backtestSymbol, savedSymbols, selected?.symbol]);

  useEffect(() => {
    if (activeSection !== 'settings' || requestedChartWindows.current) return;
    requestedChartWindows.current = true;
    void refreshChartWindows();
  }, [activeSection]);

  useEffect(() => subscribeBacktestProgress(setBacktestProgress), []);

  function handleSettingsChange(settings: Settings) {
    const withWatchlist = {
      ...settings,
      watchlist: symbolsFromWatchlist(watchlistText),
    };
    setState((current) => updateSettings(current, withWatchlist));
  }

  async function saveSettings() {
    setError(undefined);
    try {
      const next = await backend.saveSettings({
        ...state.settings,
        watchlist: symbolsFromWatchlist(watchlistText),
      });
      setState(next);
      setSelectedSymbol((current) => next.symbols.find((row) => row.symbol === current)?.symbol ?? next.symbols[0]?.symbol);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function run(action: () => Promise<AppState>) {
    setError(undefined);
    try {
      const next = await action();
      setState(next);
      setSelectedSymbol((current) => next.symbols.find((row) => row.symbol === current)?.symbol ?? next.symbols[0]?.symbol);
      setSelectedHistory(undefined);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function analyzeSelectedSymbol() {
    if (!selected?.symbol) return;
    void run(() => backend.runAnalysisNow(selected.symbol));
  }

  function setScheduledAnalysisEnabled(enabled: boolean) {
    void run(() => backend.setScheduledAnalysisEnabled(enabled));
  }

  async function refreshAccountSnapshot() {
    setError(undefined);
    setState((current) => ({
      ...current,
      account_snapshot: {
        ...current.account_snapshot,
        status: 'loading',
        error: undefined,
      },
    }));
    try {
      const next = await backend.refreshAccountSnapshot();
      setState(next);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(message);
      setState((current) => ({
        ...current,
        account_snapshot: {
          status: 'failed',
          error: message,
        },
      }));
    }
  }

  function openHistory() {
    if (!signalListRef.current) return;
    if (typeof signalListRef.current.scrollIntoView === 'function') {
      signalListRef.current.scrollIntoView({ block: 'start' });
    }
    signalListRef.current.focus({ preventScroll: true });
  }

  function selectCurrentSymbol(symbol: string) {
    setSelectedSymbol(symbol);
    setSelectedHistory(undefined);
  }

  function selectHistoryRecord(record: AnalysisHistoryRecord) {
    setSelectedHistory(record);
    setSelectedSymbol(record.result.symbol);
  }

  async function runBacktest() {
    if (!backtestRequest) return;
    setBacktestError(undefined);
    const slippagePerShare = numericInput(backtestSlippage);
    if (slippagePerShare > MAX_BACKTEST_SLIPPAGE_PER_SHARE) {
      setBacktestError(BACKTEST_SLIPPAGE_LIMIT_MESSAGE);
      return;
    }
    setBacktestProgress({
      symbol: backtestRequest.symbol,
      stage: 'fetching',
      processed_bars: 0,
      total_bars: 0,
      message: 'Starting backtest',
    });
    setBacktestLoading(true);
    try {
      const report = await backend.runBacktest(backtestRequest);
      setBacktestReport(report);
      setLastBacktestRequest(backtestRequest);
    } catch (err) {
      setBacktestError(err instanceof Error ? err.message : String(err));
    } finally {
      setBacktestLoading(false);
      setBacktestProgress(undefined);
    }
  }

  async function exportHistory() {
    setError(undefined);
    try {
      await backend.exportAnalysisHistoryCSV(compactHistoryQuery({ ...historyQuery, limit: HISTORY_PAGE_SIZE, offset: 0 }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function loadMoreHistory() {
    const nextOffset = historyPage.offset + historyPage.limit;
    if (historyLoading || nextOffset >= historyPage.total) return;
    setError(undefined);
    setHistoryLoading(true);
    try {
      const page = await backend.getAnalysisHistory(
        compactHistoryQuery({
          ...historyQuery,
          limit: HISTORY_PAGE_SIZE,
          offset: nextOffset,
        }),
      );
      setHistoryPage((current) => ({
        records: mergeHistoryRecords(current.records, page.records),
        total: page.total,
        limit: page.limit,
        offset: page.offset,
      }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setHistoryLoading(false);
    }
  }

  async function refreshChartWindows() {
    setChartWindowsLoading(true);
    setError(undefined);
    try {
      setChartWindows(await backend.listChartWindows());
    } catch (err) {
      setChartWindows([]);
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setChartWindowsLoading(false);
    }
  }

  function selectChartWindow(window: ChartWindow) {
    void run(() =>
      backend.saveSettings({
        ...state.settings,
        watchlist: symbolsFromWatchlist(watchlistText),
        chart_window: window,
      }),
    );
  }

  if (loading) {
    return <main className="app-shell loading">Loading</main>;
  }

  return (
    <main className="app-shell">
      <WorkspaceNav
        activeSection={activeSection}
        connectionStatus={state.connection_status}
        symbolCount={watchlistSymbols.length}
        backtestSymbol={backtestSymbol ?? selected?.symbol ?? savedSymbols[0]}
        timeframe={state.settings.selected_timeframe}
        onSectionChange={setActiveSection}
      />
      <div className="workspace-main">
        {activeSection === 'watchlist' ? (
          <WatchlistWorkspace
            state={state}
            watchlistText={watchlistText}
            selected={selected}
            detailSymbol={detailSymbol}
            selectedHistoryId={selectedHistory?.id}
            signalListRef={signalListRef}
            historyPage={historyPage}
            historyQuery={historyQuery}
            historyLoading={historyLoading}
            onWatchlistTextChange={setWatchlistText}
            onSaveSettings={saveSettings}
            onSetScheduledAnalysisEnabled={setScheduledAnalysisEnabled}
            onAnalyze={analyzeSelectedSymbol}
            onAnalyzeSymbol={(symbol) => run(() => backend.runAnalysisNow(symbol))}
            onScreenshotAnalyze={(symbol) => run(() => backend.runScreenshotAnalysis(symbol))}
            onOpenHistory={openHistory}
            onSelectSymbol={selectCurrentSymbol}
            onHistoryQueryChange={setHistoryQuery}
            onSelectHistory={selectHistoryRecord}
            onExportHistory={exportHistory}
            onLoadMoreHistory={loadMoreHistory}
          />
        ) : null}
        {activeSection === 'backtest' ? (
          <BacktestWorkspace
            symbols={savedSymbols}
            symbol={backtestSymbol}
            connected={state.connection_status === 'connected'}
            startDate={backtestStartDate}
            endDate={backtestEndDate}
            startTime={backtestStartTime}
            endTime={backtestEndTime}
            slippage={backtestSlippage}
            commission={backtestCommission}
            loading={backtestLoading}
            progress={backtestLoading ? backtestProgress : undefined}
            error={backtestError}
            report={backtestReport}
            lastRequest={lastBacktestRequest}
            reportFromPreviousInputs={backtestReportFromPreviousInputs}
            onSymbolChange={setBacktestSymbol}
            onStartDateChange={setBacktestStartDate}
            onEndDateChange={setBacktestEndDate}
            onStartTimeChange={setBacktestStartTime}
            onEndTimeChange={setBacktestEndTime}
            onSlippageChange={setBacktestSlippage}
            onCommissionChange={setBacktestCommission}
            onRun={() => void runBacktest()}
          />
        ) : null}
        {activeSection === 'settings' ? (
          <SettingsWorkspace
            state={state}
            chartWindows={chartWindows}
            chartWindowsLoading={chartWindowsLoading}
            onSettingsChange={handleSettingsChange}
            onSaveSettings={saveSettings}
            onConnect={() => run(backend.connectIBKR)}
            onDisconnect={() => run(backend.disconnectIBKR)}
            onRefreshAccountSnapshot={() => void refreshAccountSnapshot()}
            onRefreshChartWindows={refreshChartWindows}
            onSelectChartWindow={selectChartWindow}
          />
        ) : null}
      </div>
      <Disclaimer />
      {error ? <div className="toast-error">{error}</div> : null}
    </main>
  );
}

function historyRecordToSymbolState(record: AnalysisHistoryRecord): SymbolState {
  return {
    symbol: record.result.symbol,
    market_data_status: 'historical',
    last_analysis_time: record.result.updated_at,
    job_status: 'complete',
    current_price: record.result.current_price,
    result: record.result,
  };
}

function compactHistoryQuery(query: AnalysisHistoryQuery): AnalysisHistoryQuery {
  const next: AnalysisHistoryQuery = {
    limit: query.limit || HISTORY_PAGE_SIZE,
    offset: query.offset || 0,
  };
  if (query.symbol) next.symbol = query.symbol;
  if (query.timeframe) next.timeframe = query.timeframe;
  if (query.direction) next.direction = query.direction;
  return next;
}

function mergeHistoryRecords(current: AnalysisHistoryRecord[], incoming: AnalysisHistoryRecord[]): AnalysisHistoryRecord[] {
  const seen = new Set(current.map((record) => record.id));
  const next = [...current];
  for (const record of incoming) {
    if (seen.has(record.id)) continue;
    seen.add(record.id);
    next.push(record);
  }
  return next;
}

function buildBacktestRequest({
  symbol,
  startDate,
  endDate,
  startTime,
  endTime,
  slippage,
  commission,
}: {
  symbol: string;
  startDate: string;
  endDate: string;
  startTime: string;
  endTime: string;
  slippage: string;
  commission: string;
}): BacktestRequest {
  return {
    symbol,
    start_date: startDate,
    end_date: endDate,
    start_time: startTime,
    end_time: endTime,
    share_quantity: 100,
    slippage_per_share: numericInput(slippage),
    commission_per_order: numericInput(commission),
  };
}

function sameBacktestRequest(left: BacktestRequest, right: BacktestRequest): boolean {
  return (
    left.symbol === right.symbol &&
    left.start_date === right.start_date &&
    left.end_date === right.end_date &&
    left.start_time === right.start_time &&
    left.end_time === right.end_time &&
    left.share_quantity === right.share_quantity &&
    left.slippage_per_share === right.slippage_per_share &&
    left.commission_per_order === right.commission_per_order
  );
}

function numericInput(value: string): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0;
}

function defaultBacktestDate(): string {
  const date = new Date();
  date.setDate(date.getDate() - 1);
  while (date.getDay() === 0 || date.getDay() === 6) {
    date.setDate(date.getDate() - 1);
  }
  return formatLocalDate(date);
}

function formatLocalDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export default App;
