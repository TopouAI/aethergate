package platform

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceCreatesOrganizationWithNormalizedSlug(t *testing.T) {
	service := NewService(NewMemoryRepository())
	service.now = func() time.Time { return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC) }
	created, err := service.CreateOrganization(context.Background(), CreateOrganizationInput{Name: "Example Industries (APAC)"})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	if created.Slug != "example-industries-apac" || created.Status != "provisioning" {
		t.Fatalf("unexpected organization: %+v", created)
	}
}

func TestServiceCreatesOneTimeAPIKeyAndRepositoryStoresDigest(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(repository)
	created, err := service.CreateAPIKey(context.Background(), CreateAPIKeyInput{
		OrganizationID: "org_topoai", Name: "Production", Project: "Engineering Copilot", Models: []string{"gpt-5-mini"},
	})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}
	if len(created.Secret) < 32 || created.Record.SecretDigest == [32]byte{} {
		t.Fatal("expected a high-entropy secret and stored digest")
	}
	listed, err := service.ListAPIKeys(context.Background(), APIKeyFilter{OrganizationID: "org_topoai", Query: created.Record.Prefix})
	if err != nil || len(listed) != 1 {
		t.Fatalf("list api keys: %v, count=%d", err, len(listed))
	}
	if listed[0].SecretDigest == [32]byte{} {
		t.Fatal("repository lost the digest")
	}
}

func TestRepositoryRejectsDuplicateOrganizationSlug(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, err := service.CreateOrganization(context.Background(), CreateOrganizationInput{Name: "TopoAI"})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}
