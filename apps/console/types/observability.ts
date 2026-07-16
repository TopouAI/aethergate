export type RequestStatus = "success" | "error" | "rate_limited";

export type LlmRequest = {
  id: string;
  timestamp: string;
  model: string;
  provider: string;
  project: string;
  user: string;
  status: RequestStatus;
  latencyMs: number;
  inputTokens: number;
  outputTokens: number;
  costUsd: number;
  cached: boolean;
  prompt: string;
  response: string;
};

export type OverviewMetric = {
  label: string;
  value: string;
  change: number;
  hint: string;
  tone: "accent" | "success" | "warning" | "danger";
};

