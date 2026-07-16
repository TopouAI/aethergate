export type VaultSecretKind = "provider_api_key" | "webhook_signing_secret" | "integration_token" | "smtp_password" | "object_storage_key" | "database_credential" | "generic";
export type VaultScopeType = "provider" | "webhook" | "notification" | "reporting" | "gateway" | "integration" | "organization";

export type VaultSecret = {
  id: string; organizationId: string; name: string; kind: VaultSecretKind; scopeType: VaultScopeType; scopeId: string;
  status: "active" | "disabled"; reference: string; maskedValue: string; fingerprint: string; currentVersion: number;
  rotationIntervalDays: number; lastRotatedAt: string; rotationDueAt: string; expiresAt: string | null;
  createdBy: string; createdAt: string; updatedAt: string; disabledAt: string | null; disabledBy: string; disabledReason: string;
};

export type VaultAccessEvent = {
  id: string; organizationId: string; secretId: string; secretName: string; secretVersion: number;
  actor: string; workload: string; purpose: string; outcome: "success" | "denied" | "failure";
  requestId: string; sourceIp: string; errorCode: string; createdAt: string;
};
