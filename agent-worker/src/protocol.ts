import { z } from "zod";
import { AgentInputSchema, AgentOutputSchema, type AgentInput, type AgentOutput } from "./schema.js";

export const RequestEnvelopeSchema = z.object({
  id: z.string().min(1),
  input: AgentInputSchema
});

export type RequestEnvelope = z.infer<typeof RequestEnvelopeSchema>;

export type SuccessResponseEnvelope = {
  id: string;
  ok: true;
  output: AgentOutput;
};

export type ErrorResponseEnvelope = {
  id: string | null;
  ok: false;
  error: {
    message: string;
  };
};

export type ResponseEnvelope = SuccessResponseEnvelope | ErrorResponseEnvelope;
export type Analyzer = (input: AgentInput) => Promise<AgentOutput>;

export function parseRequestLine(line: string): RequestEnvelope {
  let parsed: unknown;
  try {
    parsed = JSON.parse(line);
  } catch (error) {
    throw new Error(`malformed JSON: ${error instanceof Error ? error.message : String(error)}`);
  }

  return RequestEnvelopeSchema.parse(parsed);
}

export function createSuccessResponse(id: string, output: unknown): SuccessResponseEnvelope {
  return {
    id,
    ok: true,
    output: AgentOutputSchema.parse(output)
  };
}

export function createErrorResponse(id: string | null, error: unknown): ErrorResponseEnvelope {
  return {
    id,
    ok: false,
    error: {
      message: error instanceof Error ? error.message : String(error)
    }
  };
}

export function serializeResponse(response: ResponseEnvelope): string {
  return JSON.stringify(response);
}

export async function handleRequestLine(line: string, analyzer: Analyzer): Promise<ResponseEnvelope> {
  let request: RequestEnvelope;
  try {
    request = parseRequestLine(line);
  } catch (error) {
    return createErrorResponse(null, error);
  }

  try {
    return createSuccessResponse(request.id, await analyzer(request.input));
  } catch (error) {
    return createErrorResponse(request.id, error);
  }
}
