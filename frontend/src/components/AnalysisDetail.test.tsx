import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { AnalysisDetail } from './AnalysisDetail';
import type { SymbolState } from '../types/domain';

describe('AnalysisDetail', () => {
  it('renders multi-timeframe context summaries', () => {
    const symbol: SymbolState = {
      symbol: 'NVDA',
      market_data_status: 'ready',
      job_status: 'complete',
      current_price: 126,
      result: {
        symbol: 'NVDA',
        timeframe: '5m',
        current_price: 126,
        stale: false,
        updated_at: '2026-05-30T18:36:10Z',
        context_summaries: [
          {
            timeframe: '15m',
            available: true,
            current_price: 128,
            bar_count: 2,
            last_closed_bar_time: '2026-05-30T18:00:00Z',
            derived: {
              session_high: 129,
              session_low: 121,
              recent_swing_highs: [129],
              recent_swing_lows: [121],
              atr: 2.4,
              volume_context: 'above_average',
              last_close_relative_to_range: 'near_high',
            },
          },
          {
            timeframe: '1h',
            available: false,
            error: 'no data for NVDA 1h',
            bar_count: 0,
          },
        ],
        output: {
          direction: 'long',
          setup_quality: 'a_plus',
          entry_zone: { low: 125.1, high: 125.5 },
          stop_loss: 124.2,
          take_profit: [126.4, 127.2],
          risk_reward: 2.1,
          confidence: 0.68,
          market_regime: '趋势回踩',
          trade_thesis: '关键位置回踩后重新出现主动买盘。',
          counterargument: '如果重新跌回区间中部，方向优势会消失。',
          no_trade_reason: '',
          rejection_reasons: [],
          summary: 'Price reclaimed the prior high.',
          price_action: ['Breakout retest held'],
          invalidated_if: 'A 5m candle closes below 124.2',
          generated_at: '2026-05-30T18:36:10Z',
        },
      },
    };

    render(<AnalysisDetail symbol={symbol} connected />);

    const context = screen.getByLabelText('Multi-timeframe context');
    expect(within(context).getByText('15m')).toBeInTheDocument();
    expect(within(context).getByText('128.00')).toBeInTheDocument();
    expect(within(context).getAllByText('129.00 / 121.00').length).toBeGreaterThan(0);
    expect(within(context).getByText('above_average')).toBeInTheDocument();
    expect(within(context).getByText('1h')).toBeInTheDocument();
    expect(within(context).getByText('Unavailable')).toBeInTheDocument();
    expect(within(context).getByText('no data for NVDA 1h')).toBeInTheDocument();
  });
});
