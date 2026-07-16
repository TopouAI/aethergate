import type { Member, ModelRecord, Project, Workspace } from "@/types/foundation";

export const foundationOrganizationId = "org_topoai";

export const seedWorkspaces: Workspace[] = [
  { id: "ws_engineering", organizationId: foundationOrganizationId, name: "Engineering", slug: "engineering", status: "active", environment: "production", projects: 2, createdAt: "2025-11-19T00:00:00Z" },
  { id: "ws_business", organizationId: foundationOrganizationId, name: "Business Operations", slug: "business-operations", status: "active", environment: "shared", projects: 1, createdAt: "2026-01-12T00:00:00Z" },
];

export const seedProjects: Project[] = [
  { id: "project_engineering_copilot", organizationId: foundationOrganizationId, workspaceId: "ws_engineering", workspace: "Engineering", name: "Engineering Copilot", slug: "engineering-copilot", status: "active", owner: "li.ming@topoai.dev", budgetUsd: 6000, monthlyCostUsd: 2148.42, requests: 184230, createdAt: "2025-12-01T00:00:00Z" },
  { id: "project_code_modernization", organizationId: foundationOrganizationId, workspaceId: "ws_engineering", workspace: "Engineering", name: "Code Modernization", slug: "code-modernization", status: "active", owner: "wang.lei@topoai.dev", budgetUsd: 3000, monthlyCostUsd: 631.88, requests: 67341, createdAt: "2026-02-08T00:00:00Z" },
  { id: "project_finance_analyst", organizationId: foundationOrganizationId, workspaceId: "ws_business", workspace: "Business Operations", name: "Finance Analyst", slug: "finance-analyst", status: "active", owner: "finance@topoai.dev", budgetUsd: 2500, monthlyCostUsd: 994.16, requests: 52101, createdAt: "2026-03-16T00:00:00Z" },
];

export const seedMembers: Member[] = [
  { id: "member_holden", organizationId: foundationOrganizationId, email: "holden@topoai.dev", displayName: "Holden", status: "active", identityProvider: "oidc", roles: ["owner"], lastActiveAt: "2026-07-14T13:58:00+08:00", createdAt: "2025-11-18T00:00:00Z" },
  { id: "member_liming", organizationId: foundationOrganizationId, email: "li.ming@topoai.dev", displayName: "Li Ming", status: "active", identityProvider: "oidc", roles: ["admin"], lastActiveAt: "2026-07-14T13:58:00+08:00", createdAt: "2025-12-01T00:00:00Z" },
  { id: "member_wanglei", organizationId: foundationOrganizationId, email: "wang.lei@topoai.dev", displayName: "Wang Lei", status: "active", identityProvider: "oidc", roles: ["developer"], lastActiveAt: "2026-07-14T13:58:00+08:00", createdAt: "2026-02-08T00:00:00Z" },
];

export const seedModels: ModelRecord[] = [
  { id: "claude-sonnet-4", provider: "Anthropic", displayName: "Claude Sonnet 4", status: "active", contextWindow: 200000, maxOutputTokens: 64000, inputPricePerMillion: 3, outputPricePerMillion: 15, supportsTools: true, supportsVision: true, supportsJson: true, regions: ["us", "eu", "apac"], createdAt: "2026-01-01T00:00:00Z" },
  { id: "gpt-5-mini", provider: "OpenAI", displayName: "GPT-5 mini", status: "active", contextWindow: 400000, maxOutputTokens: 128000, inputPricePerMillion: 0.25, outputPricePerMillion: 2, supportsTools: true, supportsVision: true, supportsJson: true, regions: ["us", "eu", "apac"], createdAt: "2026-01-01T00:00:00Z" },
  { id: "gemini-2.5-pro", provider: "Google", displayName: "Gemini 2.5 Pro", status: "active", contextWindow: 1048576, maxOutputTokens: 65536, inputPricePerMillion: 1.25, outputPricePerMillion: 10, supportsTools: true, supportsVision: true, supportsJson: true, regions: ["us", "eu", "apac"], createdAt: "2026-01-01T00:00:00Z" },
  { id: "deepseek-v3", provider: "DeepSeek", displayName: "DeepSeek V3", status: "active", contextWindow: 128000, maxOutputTokens: 8192, inputPricePerMillion: 0.27, outputPricePerMillion: 1.1, supportsTools: true, supportsVision: false, supportsJson: true, regions: ["apac"], createdAt: "2026-01-01T00:00:00Z" },
];
