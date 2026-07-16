package platform

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"slices"
	"strings"
	"time"
)

const (
	vaultAlgorithm  = "AES-256-GCM"
	vaultKeyEnvName = "AETHERGATE_VAULT_KEK"
)

var developmentVaultKEK = []byte("aethergate-development-vault-key")

type VaultSecret struct {
	ID                   string     `json:"id"`
	OrganizationID       string     `json:"organizationId"`
	Name                 string     `json:"name"`
	Kind                 string     `json:"kind"`
	ScopeType            string     `json:"scopeType"`
	ScopeID              string     `json:"scopeId"`
	Status               string     `json:"status"`
	Reference            string     `json:"reference"`
	MaskedValue          string     `json:"maskedValue"`
	Fingerprint          string     `json:"fingerprint"`
	CurrentVersion       int        `json:"currentVersion"`
	RotationIntervalDays int        `json:"rotationIntervalDays"`
	LastRotatedAt        time.Time  `json:"lastRotatedAt"`
	RotationDueAt        time.Time  `json:"rotationDueAt"`
	ExpiresAt            *time.Time `json:"expiresAt"`
	CreatedBy            string     `json:"createdBy"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	DisabledAt           *time.Time `json:"disabledAt"`
	DisabledBy           string     `json:"disabledBy"`
	DisabledReason       string     `json:"disabledReason"`
}

type VaultSecretVersion struct {
	SecretID         string    `json:"-"`
	OrganizationID   string    `json:"-"`
	Version          int       `json:"-"`
	State            string    `json:"-"`
	Ciphertext       []byte    `json:"-"`
	SecretNonce      []byte    `json:"-"`
	EncryptedDataKey []byte    `json:"-"`
	KeyNonce         []byte    `json:"-"`
	KeyVersion       string    `json:"-"`
	CreatedBy        string    `json:"-"`
	CreatedAt        time.Time `json:"-"`
}

type VaultAccessEvent struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	SecretID       string    `json:"secretId"`
	SecretName     string    `json:"secretName"`
	SecretVersion  int       `json:"secretVersion"`
	Actor          string    `json:"actor"`
	Workload       string    `json:"workload"`
	Purpose        string    `json:"purpose"`
	Outcome        string    `json:"outcome"`
	RequestID      string    `json:"requestId"`
	SourceIP       string    `json:"sourceIp"`
	ErrorCode      string    `json:"errorCode"`
	CreatedAt      time.Time `json:"createdAt"`
}

type VaultSecretFilter struct {
	OrganizationID string
	Query          string
	Kind           string
	ScopeType      string
	Status         string
	Rotation       string
}

type VaultAccessFilter struct {
	OrganizationID string
	SecretID       string
	Outcome        string
	Actor          string
}

type CreateVaultSecretInput struct {
	OrganizationID       string `json:"organizationId"`
	Name                 string `json:"name"`
	Kind                 string `json:"kind"`
	ScopeType            string `json:"scopeType"`
	ScopeID              string `json:"scopeId"`
	SecretValue          string `json:"secretValue"`
	RotationIntervalDays int    `json:"rotationIntervalDays"`
	ExpiresAt            string `json:"expiresAt"`
	CreatedBy            string `json:"createdBy"`
	RequestID            string `json:"requestId"`
	SourceIP             string `json:"sourceIp"`
}

type RotateVaultSecretInput struct {
	OrganizationID string `json:"organizationId"`
	SecretValue    string `json:"secretValue"`
	Reason         string `json:"reason"`
	RotatedBy      string `json:"rotatedBy"`
	RequestID      string `json:"requestId"`
	SourceIP       string `json:"sourceIp"`
}

type DisableVaultSecretInput struct {
	OrganizationID string `json:"organizationId"`
	Reason         string `json:"reason"`
	DisabledBy     string `json:"disabledBy"`
	RequestID      string `json:"requestId"`
	SourceIP       string `json:"sourceIp"`
}

type ResolveVaultSecretInput struct {
	OrganizationID string
	SecretID       string
	Actor          string
	Workload       string
	Purpose        string
	RequestID      string
	SourceIP       string
}

type EncryptedSecret struct {
	Ciphertext       []byte
	SecretNonce      []byte
	EncryptedDataKey []byte
	KeyNonce         []byte
	KeyVersion       string
}

type SecretProtector interface {
	Protect([]byte, []byte) (EncryptedSecret, error)
	Unprotect(EncryptedSecret, []byte) ([]byte, error)
	Algorithm() string
}

type EnvelopeCipher struct {
	key        []byte
	keyVersion string
}

type unavailableSecretProtector struct{ err error }

func NewEnvelopeCipher(key []byte, keyVersion string) (*EnvelopeCipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("vault key-encryption key must be exactly 32 bytes")
	}
	if strings.TrimSpace(keyVersion) == "" {
		return nil, fmt.Errorf("vault key version is required")
	}
	return &EnvelopeCipher{key: slices.Clone(key), keyVersion: strings.TrimSpace(keyVersion)}, nil
}

func NewVaultProtector(source string) SecretProtector {
	if source == "development-memory" {
		protector, _ := NewEnvelopeCipher(developmentVaultKEK, "development-only-v1")
		return protector
	}
	encoded := strings.TrimSpace(os.Getenv(vaultKeyEnvName))
	if encoded == "" {
		return unavailableSecretProtector{err: fmt.Errorf("%s is required for persistent Vault writes", vaultKeyEnvName)}
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return unavailableSecretProtector{err: fmt.Errorf("decode %s: %w", vaultKeyEnvName, err)}
	}
	protector, err := NewEnvelopeCipher(key, "env-v1")
	if err != nil {
		return unavailableSecretProtector{err: err}
	}
	return protector
}

func (c *EnvelopeCipher) Algorithm() string { return vaultAlgorithm }

func (c *EnvelopeCipher) Protect(plaintext, aad []byte) (EncryptedSecret, error) {
	dataKey := make([]byte, 32)
	if _, err := rand.Read(dataKey); err != nil {
		return EncryptedSecret{}, fmt.Errorf("generate vault data key: %w", err)
	}
	ciphertext, secretNonce, err := sealAESGCM(dataKey, plaintext, aad)
	if err != nil {
		return EncryptedSecret{}, err
	}
	wrappedKey, keyNonce, err := sealAESGCM(c.key, dataKey, append(slices.Clone(aad), []byte("|dek")...))
	if err != nil {
		return EncryptedSecret{}, err
	}
	return EncryptedSecret{Ciphertext: ciphertext, SecretNonce: secretNonce, EncryptedDataKey: wrappedKey, KeyNonce: keyNonce, KeyVersion: c.keyVersion}, nil
}

func (c *EnvelopeCipher) Unprotect(encrypted EncryptedSecret, aad []byte) ([]byte, error) {
	dataKey, err := openAESGCM(c.key, encrypted.EncryptedDataKey, encrypted.KeyNonce, append(slices.Clone(aad), []byte("|dek")...))
	if err != nil {
		return nil, fmt.Errorf("unwrap vault data key: %w", err)
	}
	plaintext, err := openAESGCM(dataKey, encrypted.Ciphertext, encrypted.SecretNonce, aad)
	if err != nil {
		return nil, fmt.Errorf("decrypt vault secret: %w", err)
	}
	return plaintext, nil
}

func (p unavailableSecretProtector) Protect([]byte, []byte) (EncryptedSecret, error) {
	return EncryptedSecret{}, p.err
}

func (p unavailableSecretProtector) Unprotect(EncryptedSecret, []byte) ([]byte, error) {
	return nil, p.err
}

func (p unavailableSecretProtector) Algorithm() string { return "unavailable" }

func sealAESGCM(key, plaintext, aad []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("generate GCM nonce: %w", err)
	}
	return gcm.Seal(nil, nonce, plaintext, aad), nonce, nil
}

func openAESGCM(key, ciphertext, nonce, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid GCM nonce length")
	}
	return gcm.Open(nil, nonce, ciphertext, aad)
}

type VaultRepository interface {
	Repository
	ListVaultSecrets(context.Context, VaultSecretFilter) ([]VaultSecret, error)
	GetVaultSecret(context.Context, string, string) (VaultSecret, error)
	CreateVaultSecret(context.Context, VaultSecret, VaultSecretVersion) (VaultSecret, error)
	RotateVaultSecret(context.Context, VaultSecret, VaultSecretVersion) (VaultSecret, error)
	DisableVaultSecret(context.Context, VaultSecret) (VaultSecret, error)
	GetCurrentVaultSecretVersion(context.Context, string, string) (VaultSecretVersion, error)
	ListVaultAccessEvents(context.Context, VaultAccessFilter) ([]VaultAccessEvent, error)
	CreateVaultAccessEvent(context.Context, VaultAccessEvent) error
}

type VaultService struct {
	repository VaultRepository
	protector  SecretProtector
	now        func() time.Time
}

func NewVaultService(repository VaultRepository, protector SecretProtector) *VaultService {
	return &VaultService{repository: repository, protector: protector, now: time.Now}
}

func (s *VaultService) List(ctx context.Context, filter VaultSecretFilter) ([]VaultSecret, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Kind = strings.TrimSpace(filter.Kind)
	filter.ScopeType = strings.TrimSpace(filter.ScopeType)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Rotation = strings.TrimSpace(filter.Rotation)
	return s.repository.ListVaultSecrets(ctx, filter)
}

func (s *VaultService) Get(ctx context.Context, organizationID, id string) (VaultSecret, error) {
	return s.repository.GetVaultSecret(ctx, defaultOrganization(organizationID), strings.TrimSpace(id))
}

func (s *VaultService) Create(ctx context.Context, input CreateVaultSecretInput) (VaultSecret, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.Name = strings.TrimSpace(input.Name)
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	input.ScopeType = strings.ToLower(strings.TrimSpace(input.ScopeType))
	input.ScopeID = strings.TrimSpace(input.ScopeID)
	input.CreatedBy = defaultActor(input.CreatedBy)
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	if err := validateVaultDefinition(input.Name, input.Kind, input.ScopeType, input.ScopeID, input.SecretValue, input.RotationIntervalDays, input.SourceIP); err != nil {
		return VaultSecret{}, err
	}
	now := s.now().UTC()
	expiresAt, err := optionalFutureTime(input.ExpiresAt, now)
	if err != nil {
		return VaultSecret{}, err
	}
	id, err := randomIdentifier("vsec_", 12)
	if err != nil {
		return VaultSecret{}, err
	}
	aad := vaultAAD(input.OrganizationID, id, 1)
	encrypted, err := s.protector.Protect([]byte(input.SecretValue), aad)
	if err != nil {
		return VaultSecret{}, &ValidationError{Code: "vault_key_unavailable", Message: "Vault encryption is not configured for persistent secret writes."}
	}
	rotationDays := input.RotationIntervalDays
	if rotationDays == 0 {
		rotationDays = 90
	}
	secret := VaultSecret{
		ID: id, OrganizationID: input.OrganizationID, Name: input.Name, Kind: input.Kind,
		ScopeType: input.ScopeType, ScopeID: input.ScopeID, Status: "active",
		Reference:   "vault://" + input.OrganizationID + "/" + id,
		MaskedValue: maskSecret(input.Kind, input.SecretValue), Fingerprint: secretFingerprint(input.SecretValue), CurrentVersion: 1,
		RotationIntervalDays: rotationDays, LastRotatedAt: now, RotationDueAt: now.AddDate(0, 0, rotationDays),
		ExpiresAt: expiresAt, CreatedBy: input.CreatedBy, CreatedAt: now, UpdatedAt: now,
	}
	version := VaultSecretVersion{
		SecretID: id, OrganizationID: input.OrganizationID, Version: 1, State: "active",
		Ciphertext: encrypted.Ciphertext, SecretNonce: encrypted.SecretNonce,
		EncryptedDataKey: encrypted.EncryptedDataKey, KeyNonce: encrypted.KeyNonce, KeyVersion: encrypted.KeyVersion,
		CreatedBy: input.CreatedBy, CreatedAt: now,
	}
	created, err := s.repository.CreateVaultSecret(ctx, secret, version)
	if err != nil {
		return VaultSecret{}, err
	}
	_ = s.recordAccess(ctx, created, 1, input.CreatedBy, "control-plane", "create secret", "success", input.RequestID, input.SourceIP, "")
	return created, nil
}

func (s *VaultService) Rotate(ctx context.Context, id string, input RotateVaultSecretInput) (VaultSecret, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.RotatedBy = defaultActor(input.RotatedBy)
	input.Reason = strings.TrimSpace(input.Reason)
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	if input.Reason == "" {
		return VaultSecret{}, &ValidationError{Code: "vault_rotation_reason_required", Message: "Vault rotation requires a reason."}
	}
	if err := validateSecretMaterial(input.SecretValue); err != nil {
		return VaultSecret{}, err
	}
	if err := validateOptionalIP(input.SourceIP); err != nil {
		return VaultSecret{}, err
	}
	secret, err := s.repository.GetVaultSecret(ctx, input.OrganizationID, strings.TrimSpace(id))
	if err != nil {
		return VaultSecret{}, err
	}
	if secret.Status != "active" {
		return VaultSecret{}, ErrInactive
	}
	now := s.now().UTC()
	nextVersion := secret.CurrentVersion + 1
	encrypted, err := s.protector.Protect([]byte(input.SecretValue), vaultAAD(secret.OrganizationID, secret.ID, nextVersion))
	if err != nil {
		return VaultSecret{}, &ValidationError{Code: "vault_key_unavailable", Message: "Vault encryption is not configured for persistent secret writes."}
	}
	secret.CurrentVersion = nextVersion
	secret.MaskedValue = maskSecret(secret.Kind, input.SecretValue)
	secret.Fingerprint = secretFingerprint(input.SecretValue)
	secret.LastRotatedAt = now
	secret.RotationDueAt = now.AddDate(0, 0, secret.RotationIntervalDays)
	secret.UpdatedAt = now
	version := VaultSecretVersion{
		SecretID: secret.ID, OrganizationID: secret.OrganizationID, Version: nextVersion, State: "active",
		Ciphertext: encrypted.Ciphertext, SecretNonce: encrypted.SecretNonce,
		EncryptedDataKey: encrypted.EncryptedDataKey, KeyNonce: encrypted.KeyNonce, KeyVersion: encrypted.KeyVersion,
		CreatedBy: input.RotatedBy, CreatedAt: now,
	}
	rotated, err := s.repository.RotateVaultSecret(ctx, secret, version)
	if err != nil {
		return VaultSecret{}, err
	}
	_ = s.recordAccess(ctx, rotated, nextVersion, input.RotatedBy, "control-plane", "rotate secret: "+input.Reason, "success", input.RequestID, input.SourceIP, "")
	return rotated, nil
}

func (s *VaultService) Disable(ctx context.Context, id string, input DisableVaultSecretInput) (VaultSecret, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.DisabledBy = defaultActor(input.DisabledBy)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	if input.Reason == "" {
		return VaultSecret{}, &ValidationError{Code: "vault_disable_reason_required", Message: "Disabling a Vault secret requires a reason."}
	}
	if err := validateOptionalIP(input.SourceIP); err != nil {
		return VaultSecret{}, err
	}
	secret, err := s.repository.GetVaultSecret(ctx, input.OrganizationID, strings.TrimSpace(id))
	if err != nil {
		return VaultSecret{}, err
	}
	if secret.Status != "active" {
		return VaultSecret{}, ErrInactive
	}
	now := s.now().UTC()
	secret.Status = "disabled"
	secret.DisabledAt = &now
	secret.DisabledBy = input.DisabledBy
	secret.DisabledReason = input.Reason
	secret.UpdatedAt = now
	disabled, err := s.repository.DisableVaultSecret(ctx, secret)
	if err != nil {
		return VaultSecret{}, err
	}
	_ = s.recordAccess(ctx, disabled, disabled.CurrentVersion, input.DisabledBy, "control-plane", "disable secret: "+input.Reason, "success", input.RequestID, input.SourceIP, "")
	return disabled, nil
}

func (s *VaultService) Resolve(ctx context.Context, input ResolveVaultSecretInput) ([]byte, error) {
	input.OrganizationID = defaultOrganization(input.OrganizationID)
	input.SecretID = strings.TrimSpace(input.SecretID)
	input.Actor = strings.TrimSpace(input.Actor)
	input.Workload = strings.TrimSpace(input.Workload)
	input.Purpose = strings.TrimSpace(input.Purpose)
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	if input.Actor == "" || input.Workload == "" || input.Purpose == "" {
		return nil, &ValidationError{Code: "vault_access_context_required", Message: "Vault resolution requires actor, workload, and purpose."}
	}
	if err := validateOptionalIP(input.SourceIP); err != nil {
		return nil, err
	}
	secret, err := s.repository.GetVaultSecret(ctx, input.OrganizationID, input.SecretID)
	if err != nil {
		return nil, err
	}
	if secret.Status != "active" || (secret.ExpiresAt != nil && !secret.ExpiresAt.After(s.now().UTC())) {
		_ = s.recordAccess(ctx, secret, secret.CurrentVersion, input.Actor, input.Workload, input.Purpose, "denied", input.RequestID, input.SourceIP, "vault_secret_inactive")
		return nil, ErrInactive
	}
	version, err := s.repository.GetCurrentVaultSecretVersion(ctx, input.OrganizationID, input.SecretID)
	if err != nil {
		_ = s.recordAccess(ctx, secret, secret.CurrentVersion, input.Actor, input.Workload, input.Purpose, "failure", input.RequestID, input.SourceIP, "vault_version_unavailable")
		return nil, err
	}
	plaintext, err := s.protector.Unprotect(EncryptedSecret{
		Ciphertext: version.Ciphertext, SecretNonce: version.SecretNonce,
		EncryptedDataKey: version.EncryptedDataKey, KeyNonce: version.KeyNonce, KeyVersion: version.KeyVersion,
	}, vaultAAD(secret.OrganizationID, secret.ID, version.Version))
	if err != nil {
		_ = s.recordAccess(ctx, secret, version.Version, input.Actor, input.Workload, input.Purpose, "failure", input.RequestID, input.SourceIP, "vault_decryption_failed")
		return nil, err
	}
	if err := s.recordAccess(ctx, secret, version.Version, input.Actor, input.Workload, input.Purpose, "success", input.RequestID, input.SourceIP, ""); err != nil {
		clear(plaintext)
		return nil, fmt.Errorf("record vault access evidence: %w", err)
	}
	return plaintext, nil
}

func (s *VaultService) ListAccessEvents(ctx context.Context, filter VaultAccessFilter) ([]VaultAccessEvent, error) {
	filter.OrganizationID = defaultOrganization(filter.OrganizationID)
	filter.SecretID = strings.TrimSpace(filter.SecretID)
	filter.Outcome = strings.TrimSpace(filter.Outcome)
	filter.Actor = strings.TrimSpace(filter.Actor)
	return s.repository.ListVaultAccessEvents(ctx, filter)
}

func (s *VaultService) recordAccess(ctx context.Context, secret VaultSecret, version int, actor, workload, purpose, outcome, requestID, sourceIP, errorCode string) error {
	id, err := randomIdentifier("vacc_", 10)
	if err != nil {
		return err
	}
	return s.repository.CreateVaultAccessEvent(ctx, VaultAccessEvent{
		ID: id, OrganizationID: secret.OrganizationID, SecretID: secret.ID, SecretName: secret.Name,
		SecretVersion: version, Actor: actor, Workload: workload, Purpose: purpose, Outcome: outcome,
		RequestID: requestID, SourceIP: sourceIP, ErrorCode: errorCode, CreatedAt: s.now().UTC(),
	})
}

func validateVaultDefinition(name, kind, scopeType, scopeID, value string, rotationDays int, sourceIP string) error {
	if name == "" || scopeID == "" {
		return &ValidationError{Code: "vault_identity_required", Message: "Vault name and scope identity are required."}
	}
	if !slices.Contains([]string{"provider_api_key", "webhook_signing_secret", "integration_token", "smtp_password", "object_storage_key", "database_credential", "generic"}, kind) {
		return &ValidationError{Code: "vault_kind_invalid", Message: "Vault secret kind is invalid."}
	}
	if !slices.Contains([]string{"provider", "webhook", "notification", "reporting", "gateway", "integration", "organization"}, scopeType) {
		return &ValidationError{Code: "vault_scope_invalid", Message: "Vault scope type is invalid."}
	}
	if rotationDays < 0 || rotationDays > 365 {
		return &ValidationError{Code: "vault_rotation_interval_invalid", Message: "Vault rotation interval must be between 1 and 365 days, or zero for the 90-day default."}
	}
	if rotationDays == 0 {
		rotationDays = 90
	}
	if rotationDays < 1 {
		return &ValidationError{Code: "vault_rotation_interval_invalid", Message: "Vault rotation interval is invalid."}
	}
	if err := validateSecretMaterial(value); err != nil {
		return err
	}
	return validateOptionalIP(sourceIP)
}

func validateSecretMaterial(value string) error {
	if len(value) < 8 || len(value) > 64*1024 {
		return &ValidationError{Code: "vault_secret_invalid", Message: "Vault secret material must be between 8 bytes and 64 KiB."}
	}
	return nil
}

func validateOptionalIP(value string) error {
	if value != "" && net.ParseIP(value) == nil {
		return &ValidationError{Code: "vault_ip_invalid", Message: "Vault source IP address is invalid."}
	}
	return nil
}

func optionalFutureTime(value string, now time.Time) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil || !parsed.After(now) {
		return nil, &ValidationError{Code: "vault_expiry_invalid", Message: "Vault expiry must be a future RFC3339 timestamp."}
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func vaultAAD(organizationID, secretID string, version int) []byte {
	return []byte(fmt.Sprintf("aethergate-vault|%s|%s|%d", organizationID, secretID, version))
}

func secretFingerprint(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:8])
}

func maskSecret(kind, value string) string {
	if slices.Contains([]string{"smtp_password", "database_credential", "generic"}, kind) {
		return "••••••••"
	}
	runes := []rune(value)
	if len(runes) <= 8 {
		return "••••••••"
	}
	return string(runes[:4]) + "••••••••" + string(runes[len(runes)-4:])
}

func defaultActor(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "holden@topoai.dev"
	}
	return value
}
