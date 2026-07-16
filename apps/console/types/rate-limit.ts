export type RateLimitStatus = "draft" | "enforced" | "disabled";
export type RateLimitScope = "organization" | "workspace" | "project" | "api_key" | "user";
export type RateLimitMetric = "requests" | "tokens" | "concurrency";

export type RateLimitRule = {
  id: string; organizationId: string; name: string; status: RateLimitStatus; scopeType: RateLimitScope; scopeId: string;
  metric: RateLimitMetric; window: "second" | "minute" | "hour" | "day"; limit: number; burst: number;
  action: "reject" | "throttle" | "observe"; priority: number; matchedRequests: number; limitedRequests: number;
  createdAt: string; updatedAt: string;
};

export type RateLimitDecision = {
  allowed: boolean; mode: "dry-run"; reason: string;
  matches: Array<{ ruleId: string; ruleName: string; scopeType: RateLimitScope; scopeId: string; window: string; action: string; limit: number; burst: number; currentUsage: number; projectedUsage: number; remaining: number; exceeded: boolean; retryAfterSeconds: number }>;
};
