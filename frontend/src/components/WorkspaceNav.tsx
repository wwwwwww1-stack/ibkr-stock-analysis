import type { ConnectionStatus, Timeframe } from '../types/domain';

export type WorkspaceSection = 'watchlist' | 'backtest' | 'settings';

interface Props {
  activeSection: WorkspaceSection;
  connectionStatus: ConnectionStatus;
  symbolCount: number;
  backtestSymbol?: string;
  timeframe: Timeframe;
  onSectionChange(section: WorkspaceSection): void;
}

export function WorkspaceNav({
  activeSection,
  connectionStatus,
  symbolCount,
  backtestSymbol,
  timeframe,
  onSectionChange,
}: Props) {
  return (
    <aside className="panel workspace-rail" aria-label="Workspace navigation">
      <div className="panel-heading">
        <div>
          <p className="eyebrow">Read-only market data</p>
          <h1>IBKR AI Analysis</h1>
        </div>
        <span className={`status-dot ${connectionStatus}`}>{connectionStatus}</span>
      </div>

      <nav className="directory-nav" aria-label="Workspace sections">
        <WorkspaceButton
          section="watchlist"
          activeSection={activeSection}
          label="Watchlist"
          detail={`${symbolCount} symbols`}
          onSectionChange={onSectionChange}
        />
        <WorkspaceButton
          section="backtest"
          activeSection={activeSection}
          label="Backtest"
          detail={backtestSymbol ?? 'No symbol'}
          onSectionChange={onSectionChange}
        />
        <WorkspaceButton
          section="settings"
          activeSection={activeSection}
          label="Settings"
          detail={timeframe}
          onSectionChange={onSectionChange}
        />
      </nav>
    </aside>
  );
}

function WorkspaceButton({
  section,
  activeSection,
  label,
  detail,
  onSectionChange,
}: {
  section: WorkspaceSection;
  activeSection: WorkspaceSection;
  label: string;
  detail: string;
  onSectionChange(section: WorkspaceSection): void;
}) {
  const active = activeSection === section;
  return (
    <button
      type="button"
      className={active ? 'directory-item active' : 'directory-item'}
      aria-current={active ? 'page' : undefined}
      onClick={() => onSectionChange(section)}
    >
      <span>{label}</span>
      <small>{detail}</small>
    </button>
  );
}
