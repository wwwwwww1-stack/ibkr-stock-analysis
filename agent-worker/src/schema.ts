import { z } from "zod";

const parseableDateTime = z.string().refine((value) => !Number.isNaN(Date.parse(value)), {
  message: "must be a parseable date-time"
});

const TimeframeSchema = z.enum(["1m", "5m", "15m", "1h"]);

const BarSchema = z.object({
  time: parseableDateTime,
  open: z.number(),
  high: z.number(),
  low: z.number(),
  close: z.number(),
  volume: z.number().int()
});

export const AgentInputSchema = z.object({
  symbol: z.string().regex(/^[A-Z][A-Z0-9.-]{0,15}$/),
  timeframe: TimeframeSchema,
  current_price: z.number(),
  bars: z.array(BarSchema).min(1),
  derived: z.object({
    session_high: z.number(),
    session_low: z.number(),
    recent_swing_highs: z.array(z.number()),
    recent_swing_lows: z.array(z.number()),
    atr: z.number(),
    volume_context: z.string(),
    last_close_relative_to_range: z.string()
  })
});

const EntryZoneSchema = z
  .object({
    low: z.number(),
    high: z.number()
  })
  .refine((entry) => entry.low <= entry.high, {
    message: "entry_zone.low must be less than or equal to entry_zone.high",
    path: ["low"]
  });

const BaseOutputSchema = z.object({
  direction: z.enum(["long", "short", "neutral"]),
  entry_zone: EntryZoneSchema.nullish(),
  stop_loss: z.number().nullish(),
  take_profit: z.array(z.number()).default([]),
  risk_reward: z.number().nullish(),
  confidence: z.number().min(0).max(1),
  summary: z.string().trim().min(1),
  price_action: z.array(z.string()),
  invalidated_if: z.string().trim().min(1),
  generated_at: parseableDateTime
});

export const AgentOutputSchema = BaseOutputSchema.superRefine((output, ctx) => {
  if (output.direction === "neutral") {
    return;
  }

  if (output.entry_zone == null) {
    ctx.addIssue({
      code: "custom",
      path: ["entry_zone"],
      message: "entry_zone must be present for directional analysis"
    });
  }
  if (output.stop_loss == null) {
    ctx.addIssue({
      code: "custom",
      path: ["stop_loss"],
      message: "stop_loss must be present for directional analysis"
    });
  }
  if (output.take_profit.length === 0) {
    ctx.addIssue({
      code: "custom",
      path: ["take_profit"],
      message: "take_profit must be present for directional analysis"
    });
  }
  if (output.risk_reward == null || output.risk_reward <= 0) {
    ctx.addIssue({
      code: "custom",
      path: ["risk_reward"],
      message: "risk_reward must be positive for directional analysis"
    });
  }
});

export type AgentInput = z.infer<typeof AgentInputSchema>;
export type AgentOutput = z.infer<typeof AgentOutputSchema>;
