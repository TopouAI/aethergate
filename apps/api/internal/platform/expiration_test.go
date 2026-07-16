package platform

import (
	"context"
	"testing"
	"time"
)

func TestServiceAcceptsDateOnlyAPIKeyExpiration(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(repository)
	expiration := "2027-01-31"
	created, err := service.CreateAPIKey(context.Background(), CreateAPIKeyInput{
		OrganizationID: "org_topoai",
		Name:           "Expiring production key",
		Project:        "Engineering Copilot",
		Models:         []string{"gpt-5-mini"},
		ExpiresAt:      &expiration,
	})
	if err != nil {
		t.Fatalf("create key with date-only expiration: %v", err)
	}
	want := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)
	if created.Record.ExpiresAt == nil || !created.Record.ExpiresAt.Equal(want) {
		t.Fatalf("expiration=%v, want %v", created.Record.ExpiresAt, want)
	}
}

func TestServiceRejectsInvalidAPIKeyExpiration(t *testing.T) {
	expiration := "next quarter"
	_, err := NewService(NewMemoryRepository()).CreateAPIKey(context.Background(), CreateAPIKeyInput{
		OrganizationID: "org_topoai",
		Name:           "Invalid expiration",
		Project:        "Engineering Copilot",
		Models:         []string{"gpt-5-mini"},
		ExpiresAt:      &expiration,
	})
	if err == nil {
		t.Fatal("expected invalid expiration error")
	}
}
