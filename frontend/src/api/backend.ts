import type { AppState, Settings } from '../types/domain';

export interface BackendAPI {
  getState(): Promise<AppState>;
  saveSettings(settings: Settings): Promise<AppState>;
  connectIBKR(): Promise<AppState>;
  disconnectIBKR(): Promise<AppState>;
  runAnalysisNow(): Promise<AppState>;
}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          GetState(): Promise<AppState>;
          SaveSettings(settings: Settings): Promise<AppState>;
          ConnectIBKR(): Promise<AppState>;
          DisconnectIBKR(): Promise<AppState>;
          RunAnalysisNow(): Promise<AppState>;
        };
      };
    };
    runtime?: {
      EventsOn?(eventName: string, callback: (payload: AppState) => void): () => void;
    };
  }
}

export const wailsBackend: BackendAPI = {
  getState: () => app().GetState(),
  saveSettings: (settings) => app().SaveSettings(settings),
  connectIBKR: () => app().ConnectIBKR(),
  disconnectIBKR: () => app().DisconnectIBKR(),
  runAnalysisNow: () => app().RunAnalysisNow(),
};

export function subscribeStateEvents(callback: (state: AppState) => void): () => void {
  const runtime = window.runtime;
  if (!runtime?.EventsOn) return () => undefined;
  const offCallbacks = ['connection:update', 'market:update', 'analysis:update', 'analysis:error', 'settings:update'].map((event) =>
    runtime.EventsOn!(event, callback),
  );
  return () => offCallbacks.forEach((off) => off?.());
}

function app() {
  const appAPI = window.go?.main?.App;
  if (!appAPI) {
    throw new Error('Wails backend is not available');
  }
  return appAPI;
}

