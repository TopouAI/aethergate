package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/topoai/aethergate/apps/api/internal/platform"
)

const vaultSecretSelect = `id,organization_id,name,kind,scope_type,scope_id,status,reference,masked_value,fingerprint,
	current_version,rotation_interval_days,last_rotated_at,rotation_due_at,expires_at,created_by,created_at,updated_at,
	disabled_at,COALESCE(disabled_by,''),COALESCE(disabled_reason,'')`

func (r *Repository) ListVaultSecrets(ctx context.Context, filter platform.VaultSecretFilter) ([]platform.VaultSecret, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+vaultSecretSelect+` FROM vault_secrets
		WHERE organization_id=$1 AND deleted_at IS NULL
		  AND ($2='' OR lower(name) LIKE '%'||lower($2)||'%' OR lower(scope_id) LIKE '%'||lower($2)||'%'
		       OR lower(reference) LIKE '%'||lower($2)||'%' OR lower(fingerprint) LIKE '%'||lower($2)||'%')
		  AND ($3='' OR $3='all' OR kind=$3)
		  AND ($4='' OR $4='all' OR scope_type=$4)
		  AND ($5='' OR $5='all' OR status=$5)
		  AND ($6='' OR $6='all'
		       OR ($6='overdue' AND status='active' AND rotation_due_at < now())
		       OR ($6='due' AND status='active' AND rotation_due_at >= now() AND rotation_due_at <= now()+interval '30 days')
		       OR ($6='healthy' AND status='active' AND rotation_due_at > now()+interval '30 days'))
		ORDER BY created_at DESC`, filter.OrganizationID, filter.Query, filter.Kind, filter.ScopeType, filter.Status, filter.Rotation)
	if err != nil {
		return nil, fmt.Errorf("list vault secrets: %w", err)
	}
	defer rows.Close()
	items := make([]platform.VaultSecret, 0)
	for rows.Next() {
		secret, err := scanVaultSecret(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, secret)
	}
	return items, rows.Err()
}

func (r *Repository) GetVaultSecret(ctx context.Context, organizationID, id string) (platform.VaultSecret, error) {
	secret, err := scanVaultSecret(r.pool.QueryRow(ctx, `SELECT `+vaultSecretSelect+` FROM vault_secrets WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, organizationID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.VaultSecret{}, platform.ErrNotFound
	}
	return secret, err
}

func (r *Repository) CreateVaultSecret(ctx context.Context, secret platform.VaultSecret, version platform.VaultSecretVersion) (platform.VaultSecret, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return platform.VaultSecret{}, fmt.Errorf("begin vault create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO vault_secrets(
		id,organization_id,name,kind,scope_type,scope_id,status,reference,masked_value,fingerprint,current_version,
		rotation_interval_days,last_rotated_at,rotation_due_at,expires_at,created_by,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		secret.ID, secret.OrganizationID, secret.Name, secret.Kind, secret.ScopeType, secret.ScopeID, secret.Status,
		secret.Reference, secret.MaskedValue, secret.Fingerprint, secret.CurrentVersion, secret.RotationIntervalDays,
		secret.LastRotatedAt, secret.RotationDueAt, secret.ExpiresAt, secret.CreatedBy, secret.CreatedAt, secret.UpdatedAt)
	if isForeignKeyViolation(err) {
		return platform.VaultSecret{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.VaultSecret{}, platform.ErrConflict
	}
	if err != nil {
		return platform.VaultSecret{}, fmt.Errorf("insert vault secret: %w", err)
	}
	if err := insertVaultVersion(ctx, tx, version); err != nil {
		return platform.VaultSecret{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return platform.VaultSecret{}, fmt.Errorf("commit vault create: %w", err)
	}
	return secret, nil
}

func (r *Repository) RotateVaultSecret(ctx context.Context, secret platform.VaultSecret, version platform.VaultSecretVersion) (platform.VaultSecret, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return platform.VaultSecret{}, fmt.Errorf("begin vault rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `UPDATE vault_secrets SET masked_value=$1,fingerprint=$2,current_version=$3,
		last_rotated_at=$4,rotation_due_at=$5,updated_at=$6
		WHERE organization_id=$7 AND id=$8 AND status='active' AND current_version=$9 AND deleted_at IS NULL`,
		secret.MaskedValue, secret.Fingerprint, secret.CurrentVersion, secret.LastRotatedAt, secret.RotationDueAt,
		secret.UpdatedAt, secret.OrganizationID, secret.ID, secret.CurrentVersion-1)
	if err != nil {
		return platform.VaultSecret{}, fmt.Errorf("update vault rotation metadata: %w", err)
	}
	if result.RowsAffected() != 1 {
		return platform.VaultSecret{}, platform.ErrConflict
	}
	if _, err := tx.Exec(ctx, `UPDATE vault_secret_versions SET state='superseded'
		WHERE organization_id=$1 AND secret_id=$2 AND state='active'`, secret.OrganizationID, secret.ID); err != nil {
		return platform.VaultSecret{}, fmt.Errorf("supersede vault version: %w", err)
	}
	if err := insertVaultVersion(ctx, tx, version); err != nil {
		return platform.VaultSecret{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return platform.VaultSecret{}, fmt.Errorf("commit vault rotation: %w", err)
	}
	return secret, nil
}

func (r *Repository) DisableVaultSecret(ctx context.Context, secret platform.VaultSecret) (platform.VaultSecret, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return platform.VaultSecret{}, fmt.Errorf("begin vault disable: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `UPDATE vault_secrets SET status='disabled',disabled_at=$1,disabled_by=$2,
		disabled_reason=$3,updated_at=$4 WHERE organization_id=$5 AND id=$6 AND status='active' AND deleted_at IS NULL`,
		secret.DisabledAt, secret.DisabledBy, secret.DisabledReason, secret.UpdatedAt, secret.OrganizationID, secret.ID)
	if err != nil {
		return platform.VaultSecret{}, fmt.Errorf("disable vault secret: %w", err)
	}
	if result.RowsAffected() != 1 {
		return platform.VaultSecret{}, platform.ErrInactive
	}
	if _, err := tx.Exec(ctx, `UPDATE vault_secret_versions SET state='disabled'
		WHERE organization_id=$1 AND secret_id=$2 AND state='active'`, secret.OrganizationID, secret.ID); err != nil {
		return platform.VaultSecret{}, fmt.Errorf("disable vault version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return platform.VaultSecret{}, fmt.Errorf("commit vault disable: %w", err)
	}
	return secret, nil
}

func (r *Repository) GetCurrentVaultSecretVersion(ctx context.Context, organizationID, id string) (platform.VaultSecretVersion, error) {
	var version platform.VaultSecretVersion
	err := r.pool.QueryRow(ctx, `SELECT secret_id,organization_id,version,state,ciphertext,secret_nonce,
		encrypted_data_key,key_nonce,key_version,created_by,created_at FROM vault_secret_versions
		WHERE organization_id=$1 AND secret_id=$2 AND state='active'`, organizationID, id).Scan(
		&version.SecretID, &version.OrganizationID, &version.Version, &version.State, &version.Ciphertext,
		&version.SecretNonce, &version.EncryptedDataKey, &version.KeyNonce, &version.KeyVersion, &version.CreatedBy, &version.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.VaultSecretVersion{}, platform.ErrNotFound
	}
	return version, err
}

func (r *Repository) ListVaultAccessEvents(ctx context.Context, filter platform.VaultAccessFilter) ([]platform.VaultAccessEvent, error) {
	rows, err := r.pool.Query(ctx, `SELECT e.id,e.organization_id,e.secret_id,s.name,e.secret_version,e.actor,e.workload,
		e.purpose,e.outcome,e.request_id,COALESCE(e.source_ip::text,''),e.error_code,e.created_at
		FROM vault_access_events e JOIN vault_secrets s ON s.id=e.secret_id AND s.organization_id=e.organization_id
		WHERE e.organization_id=$1 AND ($2='' OR e.secret_id=$2) AND ($3='' OR $3='all' OR e.outcome=$3)
		  AND ($4='' OR lower(e.actor) LIKE '%'||lower($4)||'%') ORDER BY e.created_at DESC`,
		filter.OrganizationID, filter.SecretID, filter.Outcome, filter.Actor)
	if err != nil {
		return nil, fmt.Errorf("list vault access events: %w", err)
	}
	defer rows.Close()
	items := make([]platform.VaultAccessEvent, 0)
	for rows.Next() {
		var event platform.VaultAccessEvent
		if err := rows.Scan(&event.ID, &event.OrganizationID, &event.SecretID, &event.SecretName, &event.SecretVersion,
			&event.Actor, &event.Workload, &event.Purpose, &event.Outcome, &event.RequestID, &event.SourceIP,
			&event.ErrorCode, &event.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, event)
	}
	return items, rows.Err()
}

func (r *Repository) CreateVaultAccessEvent(ctx context.Context, event platform.VaultAccessEvent) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO vault_access_events(id,organization_id,secret_id,secret_version,actor,
		workload,purpose,outcome,request_id,source_ip,error_code,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::inet,$11,$12)`, event.ID, event.OrganizationID,
		event.SecretID, event.SecretVersion, event.Actor, event.Workload, event.Purpose, event.Outcome,
		event.RequestID, event.SourceIP, event.ErrorCode, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("create vault access event: %w", err)
	}
	return nil
}

func insertVaultVersion(ctx context.Context, tx pgx.Tx, version platform.VaultSecretVersion) error {
	_, err := tx.Exec(ctx, `INSERT INTO vault_secret_versions(secret_id,organization_id,version,state,ciphertext,
		secret_nonce,encrypted_data_key,key_nonce,key_version,created_by,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, version.SecretID, version.OrganizationID,
		version.Version, version.State, version.Ciphertext, version.SecretNonce, version.EncryptedDataKey,
		version.KeyNonce, version.KeyVersion, version.CreatedBy, version.CreatedAt)
	if isUniqueViolation(err) {
		return platform.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("insert vault secret version: %w", err)
	}
	return nil
}

func scanVaultSecret(row rowScanner) (platform.VaultSecret, error) {
	var secret platform.VaultSecret
	err := row.Scan(&secret.ID, &secret.OrganizationID, &secret.Name, &secret.Kind, &secret.ScopeType,
		&secret.ScopeID, &secret.Status, &secret.Reference, &secret.MaskedValue, &secret.Fingerprint,
		&secret.CurrentVersion, &secret.RotationIntervalDays, &secret.LastRotatedAt, &secret.RotationDueAt,
		&secret.ExpiresAt, &secret.CreatedBy, &secret.CreatedAt, &secret.UpdatedAt, &secret.DisabledAt,
		&secret.DisabledBy, &secret.DisabledReason)
	return secret, err
}

var _ platform.VaultRepository = (*Repository)(nil)
