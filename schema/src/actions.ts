import { z } from "zod";

// ── Domain enums ──

export const BrowserActionType = z.enum([
  "navigate", "click", "fill", "select", "scroll", "wait", "screenshot",
]);

export const ApiActionType = z.enum([
  "get", "post", "put", "patch", "delete",
]);

export const DatabaseActionType = z.enum([
  "select", "insert", "update", "delete", "create_table", "drop_table",
]);

export const IacActionType = z.enum([
  "create", "modify", "destroy", "plan", "apply",
]);

export const ActionType = z.union([
  BrowserActionType,
  ApiActionType,
  DatabaseActionType,
  IacActionType,
]);

export const Domain = z.enum(["browser", "api", "database", "iac"]);

// ── Context ──

export const Context = z.object({
  authenticated: z.boolean(),
  session_id: z.string().min(1),
  previous_action: ActionType.optional(),
  domain: Domain,
});

// ── Metadata ──

export const Metadata = z.object({
  timestamp: z.string().datetime(),
  agent_id: z.string().min(1),
  request_id: z.string().min(1),
});

// ── Unified Action ──

export const Action = z.object({
  action_type: ActionType,
  target: z.string().min(1),
  parameters: z.record(z.string(), z.unknown()),
  context: Context,
  metadata: Metadata,
});

// ── Inferred TypeScript types ──

export type BrowserActionType = z.infer<typeof BrowserActionType>;
export type ApiActionType = z.infer<typeof ApiActionType>;
export type DatabaseActionType = z.infer<typeof DatabaseActionType>;
export type IacActionType = z.infer<typeof IacActionType>;
export type ActionType = z.infer<typeof ActionType>;
export type Domain = z.infer<typeof Domain>;
export type Context = z.infer<typeof Context>;
export type Metadata = z.infer<typeof Metadata>;
export type Action = z.infer<typeof Action>;
