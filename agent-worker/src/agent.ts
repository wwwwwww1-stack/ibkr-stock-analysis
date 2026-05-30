import { Agent, run } from "@openai/agents";
import { AgentInputSchema, AgentOutputSchema, type AgentInput, type AgentOutput } from "./schema.js";

export const SYSTEM_INSTRUCTIONS = [
  "You analyze short-horizon equity market data and return a schema-valid JSON object.",
  "This is not financial advice; do not claim that it is financial advice or a recommendation.",
  "Do not claim to access accounts, portfolio data, positions, balances, or live brokerage state.",
  "Never request, store, or mention needing IBKR credentials.",
  "Do not place orders, prepare order tools, call order APIs, or imply order execution capability.",
  "Use only the market data supplied in the prompt.",
  "For neutral direction, omit or null directional fields such as entry_zone, stop_loss, and risk_reward.",
  "For long or short direction, include entry_zone, stop_loss, take_profit, and positive risk_reward.",
  "Return only JSON matching the requested schema."
].join("\n");

export type AnalysisAgent = Agent<unknown, typeof AgentOutputSchema>;
export type AnalysisRunner = (agent: AnalysisAgent, prompt: string) => Promise<unknown>;

type Env = {
  OPENAI_MODEL?: string;
};

export type CreateAnalysisAgentOptions = {
  env?: Env;
};

export type AnalyzeSymbolOptions = CreateAnalysisAgentOptions & {
  runner?: AnalysisRunner;
  now?: () => Date;
};

export function getConfiguredModel(env: Env = process.env): string {
  return env.OPENAI_MODEL?.trim() || "gpt-5.5";
}

export function createAnalysisAgent(options: CreateAnalysisAgentOptions = {}): AnalysisAgent {
  return new Agent({
    name: "IBKR AI Trading Analysis Worker",
    instructions: SYSTEM_INSTRUCTIONS,
    model: getConfiguredModel(options.env),
    tools: [],
    outputType: AgentOutputSchema
  });
}

export function buildAnalysisPrompt(input: AgentInput, generatedAt: Date): string {
  return [
    "Analyze this symbol snapshot. Return only JSON.",
    `Use generated_at exactly as: ${generatedAt.toISOString()}`,
    "Input market data:",
    JSON.stringify(input, null, 2)
  ].join("\n\n");
}

export async function defaultRunner(agent: AnalysisAgent, prompt: string): Promise<unknown> {
  const result = await run(agent, prompt);
  return result.finalOutput;
}

export async function analyzeSymbol(input: unknown, options: AnalyzeSymbolOptions = {}): Promise<AgentOutput> {
  const parsedInput = AgentInputSchema.parse(input);
  const agent = createAnalysisAgent({ env: options.env });
  const prompt = buildAnalysisPrompt(parsedInput, options.now?.() ?? new Date());
  const rawOutput = await (options.runner ?? defaultRunner)(agent, prompt);
  const output = typeof rawOutput === "string" ? JSON.parse(rawOutput) : rawOutput;

  return AgentOutputSchema.parse(output);
}
