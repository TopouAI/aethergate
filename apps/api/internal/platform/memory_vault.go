package platform

import (
	"context"
	"slices"
	"strings"
	"time"
)

func (r *MemoryRepository) ListVaultSecrets(_ context.Context, filter VaultSecretFilter) ([]VaultSecret, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query := strings.ToLower(filter.Query)
	now := time.Now().UTC()
	items := make([]VaultSecret, 0, len(r.vaultSecrets))
	for _, secret := range r.vaultSecrets {
		if secret.OrganizationID != filter.OrganizationID {
			continue
		}
		if filter.Kind != "" && filter.Kind != "all" && secret.Kind != filter.Kind {
			continue
		}
		if filter.ScopeType != "" && filter.ScopeType != "all" && secret.ScopeType != filter.ScopeType {
			continue
		}
		if filter.Status != "" && filter.Status != "all" && secret.Status != filter.Status {
			continue
		}
		if filter.Rotation == "overdue" && (secret.Status != "active" || !secret.RotationDueAt.Before(now)) {
			continue
		}
		if filter.Rotation == "due" && (secret.Status != "active" || secret.RotationDueAt.Before(now) || secret.RotationDueAt.After(now.AddDate(0, 0, 30))) {
			continue
		}
		if filter.Rotation == "healthy" && (secret.Status != "active" || !secret.RotationDueAt.After(now.AddDate(0, 0, 30))) {
			continue
		}
		fields := []string{secret.Name, secret.Kind, secret.ScopeType, secret.ScopeID, secret.Reference, secret.Fingerprint}
		if query != "" && !slices.ContainsFunc(fields, func(value string) bool { return strings.Contains(strings.ToLower(value), query) }) {
			continue
		}
		items = append(items, secret)
	}
	return items, nil
}

func (r *MemoryRepository) GetVaultSecret(_ context.Context, organizationID, id string) (VaultSecret, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, secret := range r.vaultSecrets {
		if secret.OrganizationID == organizationID && secret.ID == id {
			return secret, nil
		}
	}
	return VaultSecret{}, ErrNotFound
}

func (r *MemoryRepository) CreateVaultSecret(_ context.Context, secret VaultSecret, version VaultSecretVersion) (VaultSecret, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := findOrganization(r.organizations, secret.OrganizationID); !found {
		return VaultSecret{}, ErrNotFound
	}
	if slices.ContainsFunc(r.vaultSecrets, func(existing VaultSecret) bool {
		return existing.OrganizationID == secret.OrganizationID && strings.EqualFold(existing.Name, secret.Name)
	}) {
		return VaultSecret{}, ErrConflict
	}
	r.vaultSecrets = append([]VaultSecret{secret}, r.vaultSecrets...)
	r.vaultSecretVersions = append([]VaultSecretVersion{cloneVaultVersion(version)}, r.vaultSecretVersions...)
	return secret, nil
}

func (r *MemoryRepository) RotateVaultSecret(_ context.Context, secret VaultSecret, version VaultSecretVersion) (VaultSecret, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	secretIndex := slices.IndexFunc(r.vaultSecrets, func(existing VaultSecret) bool {
		return existing.OrganizationID == secret.OrganizationID && existing.ID == secret.ID
	})
	if secretIndex < 0 {
		return VaultSecret{}, ErrNotFound
	}
	if r.vaultSecrets[secretIndex].CurrentVersion+1 != version.Version {
		return VaultSecret{}, ErrConflict
	}
	for index := range r.vaultSecretVersions {
		if r.vaultSecretVersions[index].SecretID == secret.ID && r.vaultSecretVersions[index].State == "active" {
			r.vaultSecretVersions[index].State = "superseded"
		}
	}
	r.vaultSecrets[secretIndex] = secret
	r.vaultSecretVersions = append([]VaultSecretVersion{cloneVaultVersion(version)}, r.vaultSecretVersions...)
	return secret, nil
}

func (r *MemoryRepository) DisableVaultSecret(_ context.Context, secret VaultSecret) (VaultSecret, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.vaultSecrets {
		if r.vaultSecrets[index].OrganizationID != secret.OrganizationID || r.vaultSecrets[index].ID != secret.ID {
			continue
		}
		if r.vaultSecrets[index].Status != "active" {
			return VaultSecret{}, ErrInactive
		}
		r.vaultSecrets[index] = secret
		for versionIndex := range r.vaultSecretVersions {
			if r.vaultSecretVersions[versionIndex].SecretID == secret.ID && r.vaultSecretVersions[versionIndex].State == "active" {
				r.vaultSecretVersions[versionIndex].State = "disabled"
			}
		}
		return secret, nil
	}
	return VaultSecret{}, ErrNotFound
}

func (r *MemoryRepository) GetCurrentVaultSecretVersion(_ context.Context, organizationID, id string) (VaultSecretVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, version := range r.vaultSecretVersions {
		if version.OrganizationID == organizationID && version.SecretID == id && version.State == "active" {
			return cloneVaultVersion(version), nil
		}
	}
	return VaultSecretVersion{}, ErrNotFound
}

func (r *MemoryRepository) ListVaultAccessEvents(_ context.Context, filter VaultAccessFilter) ([]VaultAccessEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]VaultAccessEvent, 0, len(r.vaultAccessEvents))
	for _, event := range r.vaultAccessEvents {
		if event.OrganizationID != filter.OrganizationID {
			continue
		}
		if filter.SecretID != "" && event.SecretID != filter.SecretID {
			continue
		}
		if filter.Outcome != "" && filter.Outcome != "all" && event.Outcome != filter.Outcome {
			continue
		}
		if filter.Actor != "" && !strings.Contains(strings.ToLower(event.Actor), strings.ToLower(filter.Actor)) {
			continue
		}
		items = append(items, event)
	}
	return items, nil
}

func (r *MemoryRepository) CreateVaultAccessEvent(_ context.Context, event VaultAccessEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.vaultAccessEvents = append([]VaultAccessEvent{event}, r.vaultAccessEvents...)
	return nil
}

func cloneVaultVersion(version VaultSecretVersion) VaultSecretVersion {
	version.Ciphertext = slices.Clone(version.Ciphertext)
	version.SecretNonce = slices.Clone(version.SecretNonce)
	version.EncryptedDataKey = slices.Clone(version.EncryptedDataKey)
	version.KeyNonce = slices.Clone(version.KeyNonce)
	return version
}
