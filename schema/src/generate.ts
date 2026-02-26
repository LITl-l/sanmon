import { zodToJsonSchema } from "zod-to-json-schema";
import { writeFileSync, mkdirSync } from "node:fs";
import { join, dirname } from "node:path";
import { Action } from "./actions";

const outDir = join(dirname(import.meta.dirname ?? __dirname), "generated");
mkdirSync(outDir, { recursive: true });

// Generate JSON Schema compatible with constrained decoding engines
// (Outlines, XGrammar, Bedrock Structured Outputs)
const schema = zodToJsonSchema(Action, {
  name: "AgentAction",
  $refStrategy: "none", // Inline all refs for compatibility with constrained decoders
});

const outPath = join(outDir, "action-schema.json");
writeFileSync(outPath, JSON.stringify(schema, null, 2) + "\n");
console.log(`JSON Schema written to ${outPath}`);
