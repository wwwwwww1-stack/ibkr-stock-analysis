#!/usr/bin/env node
import { createInterface } from "node:readline/promises";
import { pathToFileURL } from "node:url";
import { analyzeSymbol } from "./agent.js";
import { handleRequestLine, serializeResponse } from "./protocol.js";

export async function runCli(
  input: NodeJS.ReadableStream = process.stdin,
  output: NodeJS.WritableStream = process.stdout,
  diagnostics: NodeJS.WritableStream = process.stderr
): Promise<void> {
  const lines = createInterface({ input });

  for await (const line of lines) {
    if (line.trim() === "") {
      continue;
    }

    const response = await handleRequestLine(line, analyzeSymbol);
    output.write(`${serializeResponse(response)}\n`);

    if (!response.ok) {
      diagnostics.write(`agent-worker error${response.id ? ` (${response.id})` : ""}: ${response.error.message}\n`);
    }
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  runCli().catch((error: unknown) => {
    const message = error instanceof Error ? error.message : String(error);
    process.stderr.write(`agent-worker fatal: ${message}\n`);
    process.exitCode = 1;
  });
}
