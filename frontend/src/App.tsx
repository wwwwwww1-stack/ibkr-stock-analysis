import { useEffect, useMemo, useState } from 'react';
import './App.css';
import { wailsBackend, subscribeStateEvents, type BackendAPI } from './api/backend';
import { AnalysisDetail } from './components/AnalysisDetail';
import { ConnectionPanel } from './components/ConnectionPanel';
import { Disclaimer } from './components/Disclaimer';
import { SignalTable } from './components/SignalTable';
import { defaultState, mergeState, selectSymbol, symbolsFromWatchlist, updateSettings } from './state/appState';
import type { AppState, Settings } from './types/domain';

interface Props {
  backend?: BackendAPI;
}

function App({ backend = wailsBackend }: Props) {
  const [state, setState] = useState<AppState>(defaultState());
  const [watchlistText, setWatchlistText] = useState('');
  const [selectedSymbol, setSelectedSymbol] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>();

  useEffect(() => {
    let active = true;
    backend
      .getState()
      .then((next) => {
        if (!active) return;
        setState(next);
        setWatchlistText(next.settings.watchlist.join(', '));
        setSelectedSymbol(next.symbols[0]?.symbol);
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => active && setLoading(false));
    const unsubscribe = subscribeStateEvents((next) => {
      setState((current) => mergeState(current, next));
    });
    return () => {
      active = false;
      unsubscribe();
    };
  }, [backend]);

  const selected = useMemo(() => selectSymbol(state, selectedSymbol), [state, selectedSymbol]);

  function handleSettingsChange(settings: Settings) {
    const withWatchlist = {
      ...settings,
      watchlist: symbolsFromWatchlist(watchlistText),
    };
    setState((current) => updateSettings(current, withWatchlist));
  }

  async function saveSettings() {
    const next = await backend.saveSettings({
      ...state.settings,
      watchlist: symbolsFromWatchlist(watchlistText),
    });
    setState(next);
    setSelectedSymbol(next.symbols[0]?.symbol);
  }

  async function run(action: () => Promise<AppState>) {
    setError(undefined);
    try {
      const next = await action();
      setState(next);
      if (!selectedSymbol) setSelectedSymbol(next.symbols[0]?.symbol);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  if (loading) {
    return <main className="app-shell loading">Loading</main>;
  }

  return (
    <main className="app-shell">
      <ConnectionPanel
        state={state}
        watchlistText={watchlistText}
        onWatchlistTextChange={setWatchlistText}
        onSettingsChange={handleSettingsChange}
        onSaveSettings={saveSettings}
        onConnect={() => run(backend.connectIBKR)}
        onDisconnect={() => run(backend.disconnectIBKR)}
        onAnalyze={() => run(backend.runAnalysisNow)}
      />
      <SignalTable symbols={state.symbols} selected={selected?.symbol} onSelect={setSelectedSymbol} />
      <AnalysisDetail symbol={selected} />
      <Disclaimer />
      {error ? <div className="toast-error">{error}</div> : null}
    </main>
  );
}

export default App;
