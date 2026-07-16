package platform

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("resource not found")
	ErrConflict = errors.New("resource conflict")
	ErrInactive = errors.New("resource is not active")
)

type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

type Organization struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Status         string    `json:"status"`
	Plan           string    `json:"plan"`
	Region         string    `json:"region"`
	Workspaces     int       `json:"workspaces"`
	Projects       int       `json:"projects"`
	Members        int       `json:"members"`
	MonthlyCostUSD float64   `json:"monthlyCostUsd"`
	BudgetUSD      float64   `json:"budgetUsd"`
	Requests       int64     `json:"requests"`
	Owner          string    `json:"owner"`
	CreatedAt      time.Time `json:"createdAt"`
}

type OrganizationFilter struct {
	Query  string
	Status string
}

type CreateOrganizationInput struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Plan   string `json:"plan"`
	Region string `json:"region"`
	Owner  string `json:"owner"`
}

type APIKey struct {
	ID           string     `json:"id"`
	Organization string     `json:"organizationId"`
	Name         string     `json:"name"`
	Prefix       string     `json:"prefix"`
	ProjectID    *string    `json:"projectId"`
	Project      string     `json:"project"`
	Status       string     `json:"status"`
	Models       []string   `json:"models"`
	RPM          int        `json:"rpm"`
	TPM          int        `json:"tpm"`
	SpendUSD     float64    `json:"spendUsd"`
	CreatedBy    string     `json:"createdBy"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastUsedAt   *time.Time `json:"lastUsedAt"`
	ExpiresAt    *time.Time `json:"expiresAt"`
	SecretDigest [32]byte   `json:"-"`
}

type APIKeyFilter struct {
	OrganizationID string
	Query          string
	Status         string
}

type CreateAPIKeyInput struct {
	OrganizationID string   `json:"organizationId"`
	Name           string   `json:"name"`
	ProjectID      *string  `json:"projectId"`
	Project        string   `json:"project"`
	Models         []string `json:"models"`
	RPM            int      `json:"rpm"`
	TPM            int      `json:"tpm"`
	CreatedBy      string   `json:"createdBy"`
	ExpiresAt      *string  `json:"expiresAt"`
}

type CreatedAPIKey struct {
	Record APIKey `json:"data"`
	Secret string `json:"secret"`
}

type Repository interface {
	ListOrganizations(context.Context, OrganizationFilter) ([]Organization, error)
	GetOrganization(context.Context, string) (Organization, error)
	CreateOrganization(context.Context, Organization) (Organization, error)
	ListAPIKeys(context.Context, APIKeyFilter) ([]APIKey, error)
	CreateAPIKey(context.Context, APIKey) (APIKey, error)
	RevokeAPIKey(context.Context, string, string, time.Time) (APIKey, error)
}
