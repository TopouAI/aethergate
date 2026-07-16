export type IntegrationProbe = { path: string; healthy: boolean; statusCode: number; latencyMs: number; errorCode: string };
export type LiteLLMIntegrationStatus = {
  configured: boolean; baseUrl: string; masterKeyConfigured: boolean;
  overall: "not_configured" | "configured" | "ready" | "not_ready" | "unavailable";
  liveness: IntegrationProbe | null; readiness: IntegrationProbe | null; checkedAt: string | null;
};
