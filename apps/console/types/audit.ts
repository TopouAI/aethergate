export type AuditOutcome = "success" | "failure" | "denied";
export type AuditRisk = "low" | "medium" | "high" | "critical";

export type AuditEvent = {
  id: string; organizationId: string; actorId: string; actorEmail: string; action: string;
  resourceType: string; resourceId: string; outcome: AuditOutcome; riskLevel: AuditRisk; source: string;
  reason: string; requestId: string; ipAddress: string; userAgent: string;
  beforeState: Record<string, unknown>; afterState: Record<string, unknown>;
  previousHash: string; integrityHash: string; createdAt: string;
};

export type AuditRetentionPolicy = {
  organizationId: string; retentionDays: number; legalHold: boolean; exportFormat: "csv" | "jsonl";
  updatedBy: string; updatedAt: string;
};

export type AuditExport = {
  id: string; organizationId: string; requestedBy: string; format: "csv" | "jsonl";
  status: "queued" | "running" | "succeeded" | "failed" | "cancelled";
  filters: Record<string, string>; periodStart: string; periodEnd: string; rowCount: number; sizeBytes: number;
  objectKey: string; checksum: string; errorMessage: string; parentId: string | null;
  createdAt: string; completedAt: string | null;
};

export type AuditIntegrityResult = { valid: boolean; eventCount: number; headHash: string; firstInvalidId: string };
