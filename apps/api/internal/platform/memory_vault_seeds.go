package platform

import "time"

func developmentVaultSecrets() []VaultSecret {
	return []VaultSecret{
		{
			ID: "vsec_openai_primary", OrganizationID: "org_topoai", Name: "OpenAI Primary API Key",
			Kind: "provider_api_key", ScopeType: "provider", ScopeID: "provider_openai_primary", Status: "active",
			Reference: "vault://org_topoai/vsec_openai_primary", MaskedValue: "sk-p••••••••MOCK",
			Fingerprint: secretFingerprint("sk-prod-openai-development-mock"), CurrentVersion: 2, RotationIntervalDays: 90,
			LastRotatedAt: time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC), RotationDueAt: time.Date(2026, 9, 13, 2, 0, 0, 0, time.UTC),
			CreatedBy: "holden@topoai.dev", CreatedAt: time.Date(2026, 3, 17, 2, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC),
		},
		{
			ID: "vsec_anthropic_primary", OrganizationID: "org_topoai", Name: "Anthropic Primary API Key",
			Kind: "provider_api_key", ScopeType: "provider", ScopeID: "provider_anthropic_primary", Status: "active",
			Reference: "vault://org_topoai/vsec_anthropic_primary", MaskedValue: "sk-a••••••••MOCK",
			Fingerprint: secretFingerprint("sk-anthropic-development-mock"), CurrentVersion: 1, RotationIntervalDays: 60,
			LastRotatedAt: time.Date(2026, 5, 1, 2, 0, 0, 0, time.UTC), RotationDueAt: time.Date(2026, 6, 30, 2, 0, 0, 0, time.UTC),
			CreatedBy: "security@topoai.dev", CreatedAt: time.Date(2026, 5, 1, 2, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 5, 1, 2, 0, 0, 0, time.UTC),
		},
		{
			ID: "vsec_slack_legacy", OrganizationID: "org_topoai", Name: "Legacy Slack Bot Token",
			Kind: "integration_token", ScopeType: "notification", ScopeID: "slack_legacy", Status: "disabled",
			Reference: "vault://org_topoai/vsec_slack_legacy", MaskedValue: "xoxb••••••••MOCK",
			Fingerprint: secretFingerprint("xoxb-development-disabled-mock"), CurrentVersion: 1, RotationIntervalDays: 90,
			LastRotatedAt: time.Date(2026, 2, 1, 2, 0, 0, 0, time.UTC), RotationDueAt: time.Date(2026, 5, 2, 2, 0, 0, 0, time.UTC),
			CreatedBy: "security@topoai.dev", CreatedAt: time.Date(2026, 2, 1, 2, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 6, 1, 2, 0, 0, 0, time.UTC),
			DisabledBy: "security@topoai.dev", DisabledReason: "Connector retired",
		},
	}
}

func developmentVaultSecretVersions() []VaultSecretVersion {
	protector, _ := NewEnvelopeCipher(developmentVaultKEK, "development-only-v1")
	definitions := []struct {
		secretID, organizationID, value, createdBy string
		version                                    int
		createdAt                                  time.Time
	}{
		{"vsec_openai_primary", "org_topoai", "sk-prod-openai-development-mock", "holden@topoai.dev", 2, time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC)},
		{"vsec_anthropic_primary", "org_topoai", "sk-anthropic-development-mock", "security@topoai.dev", 1, time.Date(2026, 5, 1, 2, 0, 0, 0, time.UTC)},
	}
	versions := make([]VaultSecretVersion, 0, len(definitions))
	for _, definition := range definitions {
		encrypted, _ := protector.Protect([]byte(definition.value), vaultAAD(definition.organizationID, definition.secretID, definition.version))
		versions = append(versions, VaultSecretVersion{
			SecretID: definition.secretID, OrganizationID: definition.organizationID, Version: definition.version, State: "active",
			Ciphertext: encrypted.Ciphertext, SecretNonce: encrypted.SecretNonce, EncryptedDataKey: encrypted.EncryptedDataKey,
			KeyNonce: encrypted.KeyNonce, KeyVersion: encrypted.KeyVersion, CreatedBy: definition.createdBy, CreatedAt: definition.createdAt,
		})
	}
	return versions
}

func developmentVaultAccessEvents() []VaultAccessEvent {
	return []VaultAccessEvent{
		{ID: "vacc_provider_health", OrganizationID: "org_topoai", SecretID: "vsec_openai_primary", SecretName: "OpenAI Primary API Key", SecretVersion: 2, Actor: "provider-health-worker", Workload: "provider-health-worker", Purpose: "active provider probe", Outcome: "success", RequestID: "probe_openai_20260715", SourceIP: "10.20.0.12", CreatedAt: time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)},
		{ID: "vacc_rotation", OrganizationID: "org_topoai", SecretID: "vsec_openai_primary", SecretName: "OpenAI Primary API Key", SecretVersion: 2, Actor: "holden@topoai.dev", Workload: "control-plane", Purpose: "rotate secret: scheduled rotation", Outcome: "success", RequestID: "req_vault_rotate_001", SourceIP: "10.12.0.8", CreatedAt: time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC)},
	}
}
