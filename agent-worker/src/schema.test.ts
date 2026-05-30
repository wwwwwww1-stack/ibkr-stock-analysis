import { describe, expect, it } from "vitest";
import { AgentInputSchema, AgentOutputSchema } from "./schema.js";

const validInput = {
  symbol: "NVDA",
  timeframe: "5m",
  current_price: 125.42,
  bars: [
    {
      time: "2026-05-30T10:00:00Z",
      open: 124.9,
      high: 125.6,
      low: 124.7,
      close: 125.42,
      volume: 125000
    }
  ],
  derived: {
    session_high: 126.1,
    session_low: 123.8,
    recent_swing_highs: [125.8, 126.1],
    recent_swing_lows: [124.1, 123.8],
    atr: 0.9,
    volume_context: "above average",
    last_close_relative_to_range: "upper third"
  }
};

describe("AgentInputSchema", () => {
  it("accepts valid Go-compatible input", () => {
    expect(AgentInputSchema.parse(validInput)).toEqual(validInput);
  });

  it("rejects invalid input", () => {
    const result = AgentInputSchema.safeParse({
      ...validInput,
      symbol: "nvda",
      timeframe: "30m",
      bars: []
    });

    expect(result.success).toBe(false);
  });
});

describe("AgentOutputSchema", () => {
  it("accepts a valid long output", () => {
    const output = AgentOutputSchema.parse({
      direction: "long",
      entry_zone: { low: 125.1, high: 125.5 },
      stop_loss: 124.2,
      take_profit: [126.4, 127.2],
      risk_reward: 2.1,
      confidence: 0.68,
      summary: "Price reclaimed the prior high and held above support.",
      price_action: ["Higher low formed", "Breakout retest held"],
      invalidated_if: "A 5m candle closes below 124.2.",
      generated_at: "2026-05-30T10:05:00Z"
    });

    expect(output.direction).toBe("long");
  });

  it("accepts neutral output without directional fields", () => {
    const output = AgentOutputSchema.parse({
      direction: "neutral",
      confidence: 0.31,
      summary: "Price is balanced inside the recent range.",
      price_action: ["Range remains intact"],
      invalidated_if: "A close outside the range changes the setup.",
      generated_at: "2026-05-30T10:05:00Z"
    });

    expect(output.entry_zone).toBeUndefined();
    expect(output.risk_reward).toBeUndefined();
  });

  it("rejects invalid risk, confidence, and entry values", () => {
    const result = AgentOutputSchema.safeParse({
      direction: "short",
      entry_zone: { low: 126, high: 125 },
      stop_loss: 127,
      take_profit: [124],
      risk_reward: 0,
      confidence: 1.2,
      summary: "Invalid setup.",
      price_action: ["Invalid"],
      invalidated_if: "Invalid",
      generated_at: "2026-05-30T10:05:00Z"
    });

    expect(result.success).toBe(false);
  });
});
