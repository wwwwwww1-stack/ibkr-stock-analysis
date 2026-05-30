import { describe, expect, it, vi } from "vitest";
import {
  createErrorResponse,
  createSuccessResponse,
  handleRequestLine,
  parseRequestLine,
  serializeResponse
} from "./protocol.js";
import type { AgentInput, AgentOutput } from "./schema.js";

const input: AgentInput = {
  symbol: "AAPL",
  timeframe: "1m",
  current_price: 195.1,
  bars: [
    {
      time: "2026-05-30T10:00:00Z",
      open: 194.8,
      high: 195.2,
      low: 194.7,
      close: 195.1,
      volume: 98000
    }
  ],
  derived: {
    session_high: 196,
    session_low: 193.5,
    recent_swing_highs: [195.7],
    recent_swing_lows: [194],
    atr: 0.5,
    volume_context: "normal",
    last_close_relative_to_range: "middle"
  }
};

const output: AgentOutput = {
  direction: "neutral",
  take_profit: [],
  confidence: 0.4,
  summary: "Balanced.",
  price_action: ["Inside range"],
  invalidated_if: "Range break.",
  generated_at: "2026-05-30T10:05:00Z"
};

describe("parseRequestLine", () => {
  it("parses a valid request envelope", () => {
    const request = parseRequestLine(JSON.stringify({ id: "req-1", input }));

    expect(request).toEqual({ id: "req-1", input });
  });

  it("throws on malformed JSON", () => {
    expect(() => parseRequestLine("{")).toThrow(/malformed JSON/);
  });
});

describe("response envelopes", () => {
  it("creates and serializes a successful response envelope", () => {
    expect(JSON.parse(serializeResponse(createSuccessResponse("req-1", output)))).toEqual({
      id: "req-1",
      ok: true,
      output
    });
  });

  it("creates an error response envelope", () => {
    expect(createErrorResponse("req-1", new Error("bad input"))).toEqual({
      id: "req-1",
      ok: false,
      error: {
        message: "bad input"
      }
    });
  });
});

describe("handleRequestLine", () => {
  it("returns a success response from the analyzer", async () => {
    const analyzer = vi.fn(async () => output);

    const response = await handleRequestLine(JSON.stringify({ id: "req-1", input }), analyzer);

    expect(response).toEqual({ id: "req-1", ok: true, output });
    expect(analyzer).toHaveBeenCalledWith(input);
  });

  it("returns an error response for malformed JSON", async () => {
    const response = await handleRequestLine("{", async () => output);

    expect(response.ok).toBe(false);
    if (!response.ok) {
      expect(response.id).toBeNull();
      expect(response.error.message).toContain("malformed JSON");
    }
  });
});
