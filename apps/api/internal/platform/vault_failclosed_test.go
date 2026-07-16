package platform

import (
	"context"
	"errors"
	"testing"
)

type failingVaultAccessRepository struct{ *MemoryRepository }

func (f failingVaultAccessRepository) CreateVaultAccessEvent(context.Context, VaultAccessEvent) error {
	return errors.New("access evidence unavailable")
}

func TestVaultResolveFailsClosedWhenAccessEvidenceCannotBeWritten(t *testing.T) {
	repository := failingVaultAccessRepository{MemoryRepository: NewMemoryRepository()}
	protector, err := NewEnvelopeCipher(developmentVaultKEK, "test-v1")
	if err != nil {
		t.Fatal(err)
	}
	service := NewVaultService(repository, protector)
	created, err := service.Create(context.Background(), CreateVaultSecretInput{
		OrganizationID: "org_topoai", Name: "Fail Closed Key", Kind: "provider_api_key",
		ScopeType: "provider", ScopeID: "provider_fail_closed", SecretValue: "secret-must-not-escape",
		RotationIntervalDays: 90, CreatedBy: "security@topoai.dev",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	plaintext, err := service.Resolve(context.Background(), ResolveVaultSecretInput{
		OrganizationID: "org_topoai", SecretID: created.ID, Actor: "gateway-worker",
		Workload: "gateway-worker", Purpose: "provider request", RequestID: "req_fail_closed",
	})
	if err == nil || len(plaintext) != 0 {
		t.Fatalf("resolution must fail closed without evidence, plaintext=%q err=%v", plaintext, err)
	}
}
