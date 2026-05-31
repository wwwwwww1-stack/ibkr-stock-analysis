import type { BacktestProgress, BacktestReport, BacktestRequest } from '../types/domain';
import { BacktestSidebar } from './BacktestSidebar';

interface Props {
  symbols: string[];
  symbol?: string;
  connected: boolean;
  startDate: string;
  endDate: string;
  startTime: string;
  endTime: string;
  slippage: string;
  commission: string;
  loading: boolean;
  progress?: BacktestProgress;
  error?: string;
  report?: BacktestReport;
  lastRequest?: BacktestRequest;
  reportFromPreviousInputs: boolean;
  onSymbolChange(symbol: string): void;
  onStartDateChange(value: string): void;
  onEndDateChange(value: string): void;
  onStartTimeChange(value: string): void;
  onEndTimeChange(value: string): void;
  onSlippageChange(value: string): void;
  onCommissionChange(value: string): void;
  onRun(): void;
}

export function BacktestWorkspace(props: Props) {
  return (
    <section className="workspace-section backtest-workspace" aria-label="Backtest workspace">
      <BacktestSidebar {...props} embedded />
    </section>
  );
}
