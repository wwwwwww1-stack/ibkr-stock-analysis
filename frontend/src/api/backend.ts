import type {
  AnalysisHistoryPage,
  AnalysisHistoryQuery,
  AppState,
  BacktestProgress,
  BacktestReport,
  BacktestRequest,
  ChartWindow,
  Settings,
} from '../types/domain';

export interface BackendAPI {
  getState(): Promise<AppState>;
  saveSettings(settings: Settings): Promise<AppState>;
  refreshAccountSnapshot(): Promise<AppState>;
  connectIBKR(): Promise<AppState>;
  disconnectIBKR(): Promise<AppState>;
  setScheduledAnalysisEnabled(enabled: boolean): Promise<AppState>;
  runAnalysisNow(symbol: string): Promise<AppState>;
  runScreenshotAnalysis(symbol: string): Promise<AppState>;
  runBacktest(request: BacktestRequest): Promise<BacktestReport>;
  getAnalysisHistory(query: AnalysisHistoryQuery): Promise<AnalysisHistoryPage>;
  exportAnalysisHistoryCSV(query: AnalysisHistoryQuery): Promise<string>;
  listChartWindows(): Promise<ChartWindow[]>;
}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          GetState(): Promise<AppState>;
          SaveSettings(settings: Settings): Promise<AppState>;
          RefreshAccountSnapshot(): Promise<AppState>;
          ConnectIBKR(): Promise<AppState>;
          DisconnectIBKR(): Promise<AppState>;
          SetScheduledAnalysisEnabled(enabled: boolean): Promise<AppState>;
          RunAnalysisNow(symbol: string): Promise<AppState>;
          RunScreenshotAnalysis(symbol: string): Promise<AppState>;
          RunBacktest(request: BacktestRequest): Promise<BacktestReport>;
          GetAnalysisHistory(query: AnalysisHistoryQuery): Promise<AnalysisHistoryPage>;
          ExportAnalysisHistoryCSV(query: AnalysisHistoryQuery): Promise<string>;
          ListChartWindows(): Promise<ChartWindow[]>;
        };
      };
    };
    runtime?: {
      EventsOn?<T = unknown>(eventName: string, callback: (payload: T) => void): () => void;
    };
  }
}

export const wailsBackend: BackendAPI = {
  getState: () => app().GetState(),
  saveSettings: (settings) => app().SaveSettings(settings),
  refreshAccountSnapshot: () => app().RefreshAccountSnapshot(),
  connectIBKR: () => app().ConnectIBKR(),
  disconnectIBKR: () => app().DisconnectIBKR(),
  setScheduledAnalysisEnabled: (enabled) => app().SetScheduledAnalysisEnabled(enabled),
  runAnalysisNow: (symbol) => app().RunAnalysisNow(symbol),
  runScreenshotAnalysis: (symbol) => app().RunScreenshotAnalysis(symbol),
  runBacktest: (request) => app().RunBacktest(request),
  getAnalysisHistory: (query) => app().GetAnalysisHistory(query),
  exportAnalysisHistoryCSV: (query) => app().ExportAnalysisHistoryCSV(query),
  listChartWindows: () => app().ListChartWindows(),
};

export function subscribeStateEvents(callback: (state: AppState) => void): () => void {
  const runtime = window.runtime;
  if (!runtime?.EventsOn) return () => undefined;
  const offCallbacks = ['connection:update', 'market:update', 'analysis:update', 'analysis:error', 'settings:update', 'account:update'].map((event) =>
    runtime.EventsOn!<AppState>(event, callback),
  );
  return () => offCallbacks.forEach((off) => off?.());
}

export function subscribeBacktestProgress(callback: (progress: BacktestProgress) => void): () => void {
  const runtime = window.runtime;
  if (!runtime?.EventsOn) return () => undefined;
  return runtime.EventsOn<BacktestProgress>('backtest:progress', callback);
}

function app() {
  const appAPI = window.go?.main?.App;
  if (!appAPI) {
    throw new Error('Wails backend is not available');
  }
  return appAPI;
}
