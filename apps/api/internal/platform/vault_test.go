package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestVaultEnvelopeLifecycleAndAccessEvidence(t *testing.T) {
	repository := NewMemoryRepository()
	protector, err := NewEnvelopeCipher(developmentVaultKEK, "test-v1")
	if err != nil {
		t.Fatal(err)
	}
	service := NewVaultService(repository, protector)
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	plaintext := "sk-test-vault-secret-123456"
	created, err := service.Create(context.Background(), CreateVaultSecretInput{
		OrganizationID: "org_topoai", Name: "Test Provider Key", Kind: "provider_api_key",
		ScopeType: "provider", ScopeID: "provider_test", SecretValue: plaintext,
		RotationIntervalDays: 30, ExpiresAt: "2027-01-01T00:00:00Z", CreatedBy: "security@topoai.dev",
		RequestID: "req_vault_create", SourceIP: "10.0.0.8",
	})
	if err != nil || created.CurrentVersion != 1 || created.Reference == "" || created.Fingerprint == "" {
		t.Fatalf("create vault secret: %#v %v", created, err)
	}
	if created.MaskedValue == plaintext || bytes.Contains([]byte(created.MaskedValue), []byte(plaintext)) {
		t.Fatal("public metadata exposed plaintext")
	}
	version, err := repository.GetCurrentVaultSecretVersion(context.Background(), "org_topoai", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(version.Ciphertext, []byte(plaintext)) || bytes.Equal(version.EncryptedDataKey, []byte(plaintext)) {
		t.Fatal("encrypted storage contains plaintext secret")
	}
	encoded, _ := json.Marshal(created)
	if bytes.Contains(encoded, []byte(plaintext)) || bytes.Contains(encoded, []byte("ciphertext")) || bytes.Contains(encoded, []byte("encryptedDataKey")) {
		t.Fatalf("public serialization leaked secret internals: %s", encoded)
	}

	resolved, err := service.Resolve(context.Background(), ResolveVaultSecretInput{
		OrganizationID: "org_topoai", SecretID: created.ID, Actor: "provider-health-worker",
		Workload: "provider-health-worker", Purpose: "connectivity probe", RequestID: "probe_test", SourceIP: "10.20.0.10",
	})
	if err != nil || string(resolved) != plaintext {
		t.Fatalf("resolve vault secret: %q %v", resolved, err)
	}

	rotatedValue := "sk-test-vault-secret-rotated-654321"
	rotated, err := service.Rotate(context.Background(), created.ID, RotateVaultSecretInput{
		OrganizationID: "org_topoai", SecretValue: rotatedValue, Reason: "scheduled rotation",
		RotatedBy: "security@topoai.dev", RequestID: "req_vault_rotate", SourceIP: "10.0.0.8",
	})
	if err != nil || rotated.CurrentVersion != 2 || rotated.Fingerprint == created.Fingerprint {
		t.Fatalf("rotate vault secret: %#v %v", rotated, err)
	}
	resolved, err = service.Resolve(context.Background(), ResolveVaultSecretInput{
		OrganizationID: "org_topoai", SecretID: created.ID, Actor: "gateway-worker",
		Workload: "gateway-worker", Purpose: "provider request", RequestID: "req_gateway", SourceIP: "10.20.0.11",
	})
	if err != nil || string(resolved) != rotatedValue {
		t.Fatalf("resolve rotated vault secret: %q %v", resolved, err)
	}

	disabled, err := service.Disable(context.Background(), created.ID, DisableVaultSecretInput{
		OrganizationID: "org_topoai", Reason: "provider retired", DisabledBy: "security@topoai.dev",
		RequestID: "req_vault_disable", SourceIP: "10.0.0.8",
	})
	if err != nil || disabled.Status != "disabled" {
		t.Fatalf("disable vault secret: %#v %v", disabled, err)
	}
	_, err = service.Resolve(context.Background(), ResolveVaultSecretInput{
		OrganizationID: "org_topoai", SecretID: created.ID, Actor: "gateway-worker",
		Workload: "gateway-worker", Purpose: "provider request", RequestID: "req_denied", SourceIP: "10.20.0.11",
	})
	if !errors.Is(err, ErrInactive) {
		t.Fatalf("expected inactive resolution, got %v", err)
	}
	events, err := service.ListAccessEvents(context.Background(), VaultAccessFilter{OrganizationID: "org_topoai", SecretID: created.ID})
	if err != nil || len(events) != 6 || events[0].Outcome != "denied" {
		t.Fatalf("vault access evidence: %#v %v", events, err)
	}
}

func TestVaultEnvelopeRejectsWrongAADAndInvalidInputs(t *testing.T) {
	protector, err := NewEnvelopeCipher(developmentVaultKEK, "test-v1")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := protector.Protect([]byte("secret-material"), []byte("tenant-a"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := protector.Unprotect(encrypted, []byte("tenant-b")); err == nil {
		t.Fatal("expected authenticated decryption to reject different tenant AAD")
	}

	service := NewVaultService(NewMemoryRepository(), protector)
	_, err = service.Create(context.Background(), CreateVaultSecretInput{
		OrganizationID: "org_topoai", Name: "Weak", Kind: "provider_api_key", ScopeType: "provider",
		ScopeID: "provider_test", SecretValue: "short", RotationIntervalDays: 90,
	})
	assertValidationCode(t, err, "vault_secret_invalid")
	_, err = service.Create(context.Background(), CreateVaultSecretInput{
		OrganizationID: "org_topoai", Name: "Wrong Kind", Kind: "unknown", ScopeType: "provider",
		ScopeID: "provider_test", SecretValue: "long-enough-secret", RotationIntervalDays: 90,
	})
	assertValidationCode(t, err, "vault_kind_invalid")
}
