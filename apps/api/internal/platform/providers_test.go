package platform

import (
	"context"
	"testing"
)

func TestProviderLifecycle(t *testing.T) {
	service := NewProviderService(NewMemoryRepository())
	created, err := service.CreateProvider(context.Background(), CreateProviderInput{Name: "Azure East", Provider: "Azure OpenAI", BaseURL: "https://example.openai.azure.com/"})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if created.BaseURL != "https://example.openai.azure.com" || created.CredentialState != "missing" {
		t.Fatalf("unexpected provider: %+v", created)
	}
	items, err := service.ListProviders(context.Background(), ProviderFilter{OrganizationID: "org_topoai", Query: "azure"})
	if err != nil || len(items) != 1 {
		t.Fatalf("list providers: count=%d err=%v", len(items), err)
	}
}

func TestProviderRejectsUnsafeBaseURL(t *testing.T) {
	service := NewProviderService(NewMemoryRepository())
	if _, err := service.CreateProvider(context.Background(), CreateProviderInput{Name: "Unsafe", Provider: "Custom", BaseURL: "file:///etc/passwd"}); err == nil {
		t.Fatal("expected invalid provider URL")
	}
}
