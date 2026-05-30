import { describe, expect, it, vi } from "vitest";
import {
  type AnalysisRunner,
  SYSTEM_INSTRUCTIONS,
  analyzeSymbol,
  buildAnalysisPrompt,
  createAnalysisAgent,
  getConfiguredModel
} from "./agent.js";
import type { AgentInput, AgentOutput } from "./schema.js";

const input: AgentInput = {
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
    recent_swing_highs: [125.8],
    recent_swing_lows: [124.1],
    atr: 0.9,
    volume_context: "above average",
    last_close_relative_to_range: "upper third"
  }
};

const output: AgentOutput = {
  direction: "neutral",
  confidence: 0.42,
  take_profit: [],
  summary: "Price is balanced inside the recent range.",
  price_action: ["Range remains intact"],
  invalidated_if: "A close outside the range changes the setup.",
  generated_at: "2026-05-30T10:05:00Z"
};

describe("createAnalysisAgent", () => {
  it("uses gpt-5.5 by default and allows OPENAI_MODEL override", () => {
    expect(getConfiguredModel({})).toBe("gpt-5.5");
    expect(getConfiguredModel({ OPENAI_MODEL: "gpt-test" })).toBe("gpt-test");

    const agent = createAnalysisAgent({ env: { OPENAI_MODEL: "gpt-test" } });
    expect(agent.model).toBe("gpt-test");
    expect(agent.tools).toEqual([]);
  });

  it("forbids advice claims, account access, credentials, and order tools", () => {
    expect(SYSTEM_INSTRUCTIONS).toContain("not financial advice");
    expect(SYSTEM_INSTRUCTIONS).toContain("Do not claim to access accounts");
    expect(SYSTEM_INSTRUCTIONS).toContain("IBKR credentials");
    expect(SYSTEM_INSTRUCTIONS).toContain("Do not place orders");
  });
});

describe("analyzeSymbol", () => {
  it("uses an injectable runner and validates its deterministic response", async () => {
    const runner = vi.fn<AnalysisRunner>(async () => output);

    const result = await analyzeSymbol(input, { runner, now: () => new Date("2026-05-30T10:05:00Z") });

    expect(result).toEqual(output);
    expect(runner).toHaveBeenCalledOnce();
    const [agent, prompt] = runner.mock.calls[0]!;
    expect(agent.model).toBe("gpt-5.5");
    expect(prompt).toContain(JSON.stringify(input, null, 2));
  });

  it("parses JSON string runner output", async () => {
    const result = await analyzeSymbol(input, { runner: async () => JSON.stringify(output) });

    expect(result.direction).toBe("neutral");
  });

  it("builds a prompt with timestamped market data and JSON-only instructions", () => {
    const prompt = buildAnalysisPrompt(input, new Date("2026-05-30T10:05:00Z"));

    expect(prompt).toContain("2026-05-30T10:05:00.000Z");
    expect(prompt).toContain("Return only JSON");
    expect(prompt).toContain('"symbol": "NVDA"');
  });
});
