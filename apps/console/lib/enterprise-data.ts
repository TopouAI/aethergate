import type { APIKeyRecord, Organization } from "@/types/enterprise";

export const organizations: Organization[] = [
  { id: "org_topoai", name: "TopoAI", slug: "topoai", status: "active", plan: "Enterprise", region: "China East", workspaces: 4, projects: 18, members: 46, monthlyCostUsd: 6842, budgetUsd: 12000, requests: 482340, owner: "holden@topoai.dev", createdAt: "2025-11-18" },
  { id: "org_acme", name: "Acme Manufacturing", slug: "acme-manufacturing", status: "active", plan: "Enterprise", region: "Singapore", workspaces: 6, projects: 24, members: 91, monthlyCostUsd: 8931, budgetUsd: 15000, requests: 591208, owner: "platform@acme.cn", createdAt: "2026-01-07" },
  { id: "org_northstar", name: "Northstar Financial", slug: "northstar-financial", status: "active", plan: "Evaluation", region: "Hong Kong", workspaces: 2, projects: 7, members: 19, monthlyCostUsd: 1840, budgetUsd: 4000, requests: 128920, owner: "ai-office@northstar.com", createdAt: "2026-05-26" },
  { id: "org_meridian", name: "Meridian Health", slug: "meridian-health", status: "provisioning", plan: "Evaluation", region: "Singapore", workspaces: 1, projects: 2, members: 8, monthlyCostUsd: 392, budgetUsd: 2500, requests: 24518, owner: "security@meridian.health", createdAt: "2026-07-11" },
  { id: "org_labs", name: "Open Research Labs", slug: "open-research-labs", status: "suspended", plan: "Open Source", region: "US West", workspaces: 2, projects: 5, members: 13, monthlyCostUsd: 0, budgetUsd: 0, requests: 8912, owner: "admin@orlabs.org", createdAt: "2026-02-19" },
];

export const apiKeys: APIKeyRecord[] = [
  { id: "key_01JY8KX2F3", name: "Engineering Copilot · Production", prefix: "ag_live_7Tx9", project: "Engineering Copilot", status: "active", models: ["claude-sonnet-4", "gpt-5-mini"], rpm: 600, tpm: 1_200_000, spendUsd: 2148.42, createdBy: "li.ming@topoai.dev", createdAt: "2026-04-12", lastUsedAt: "2026-07-14T14:08:22+08:00", expiresAt: "2027-04-12" },
  { id: "key_01JY8KPGN7", name: "Customer Support · Production", prefix: "ag_live_2Qm4", project: "Customer Support", status: "active", models: ["gpt-5-mini", "gemini-2.5-pro"], rpm: 900, tpm: 1_800_000, spendUsd: 1862.07, createdBy: "platform@acme.cn", createdAt: "2026-03-28", lastUsedAt: "2026-07-14T14:07:41+08:00", expiresAt: null },
  { id: "key_01JY8KHC91", name: "Finance Analyst · Scheduled", prefix: "ag_live_9Ka2", project: "Finance Analyst", status: "active", models: ["claude-sonnet-4"], rpm: 120, tpm: 480_000, spendUsd: 994.16, createdBy: "finance-platform@acme.cn", createdAt: "2026-05-03", lastUsedAt: "2026-07-14T13:27:22+08:00", expiresAt: "2026-11-03" },
  { id: "key_01JY8K89LX", name: "Contract Intelligence · Staging", prefix: "ag_test_4Vn8", project: "Contract Intelligence", status: "expired", models: ["gemini-2.5-pro"], rpm: 60, tpm: 240_000, spendUsd: 182.34, createdBy: "chen.yu@acme.cn", createdAt: "2026-01-15", lastUsedAt: "2026-06-28T09:12:04+08:00", expiresAt: "2026-06-30" },
  { id: "key_01JY8JZ4DA", name: "Legacy Migration", prefix: "ag_live_1Lp7", project: "Code Modernization", status: "revoked", models: ["deepseek-v3"], rpm: 300, tpm: 900_000, spendUsd: 631.88, createdBy: "wang.lei@topoai.dev", createdAt: "2026-02-08", lastUsedAt: "2026-07-01T17:44:19+08:00", expiresAt: null },
];
