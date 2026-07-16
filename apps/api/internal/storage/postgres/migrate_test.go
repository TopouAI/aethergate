package postgres

import (
	"strings"
	"testing"
)

func TestFoundationMigrationContainsOwnedSecurityBoundaries(t *testing.T) {
	script, err := migrationFiles.ReadFile("migrations/000001_foundation.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	text := strings.ReplaceAll(string(script), "\r\n", "\n")
	for _, required := range []string{
		"CREATE TABLE organizations", "CREATE TABLE workspaces", "CREATE TABLE projects",
		"CREATE TABLE members", "CREATE TABLE role_bindings", "CREATE TABLE models",
		"CREATE TABLE provider_connections", "CREATE TABLE api_keys (\n    id text PRIMARY KEY",
		"CREATE TABLE provider_health_probes", "CREATE TABLE provider_health_events", "routing_eligible boolean",
		"CREATE TABLE routing_policies", "CREATE TABLE routing_targets",
		"CREATE TABLE rate_limit_rules",
		"CREATE TABLE budgets",
		"CREATE TABLE alert_rules", "CREATE TABLE alert_incidents",
		"CREATE TABLE report_schedules", "CREATE TABLE report_runs",
		"CREATE TABLE notification_preferences", "CREATE TABLE notifications",
		"CREATE TABLE notification_escalation_policies", "CREATE TABLE notification_deliveries",
		"CREATE TABLE audit_retention_policies", "CREATE TABLE audit_exports", "prevent_audit_event_mutation", "integrity_hash",
		"CREATE TABLE vault_secrets", "CREATE TABLE vault_secret_versions", "CREATE TABLE vault_access_events",
		"encrypted_data_key bytea", "prevent_vault_access_event_mutation", "vault_access_events_immutable",
		"CREATE TABLE webhook_endpoints", "CREATE TABLE webhook_deliveries",
		"CREATE TABLE api_keys", "secret_digest bytea", "CREATE TABLE audit_events",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
	if strings.Contains(strings.ToLower(text), "litellm") {
		t.Fatal("AetherGate migration must not reference LiteLLM tables")
	}
	if strings.Contains(text, "id text PRIMARY KEY,\nCREATE TABLE") {
		t.Fatal("migration contains a table declaration inserted inside a preceding table body")
	}
	if !strings.Contains(text, "CREATE TRIGGER audit_retention_policies_set_updated_at\n    BEFORE UPDATE ON audit_retention_policies\n    FOR EACH ROW EXECUTE FUNCTION set_updated_at();\nCREATE TRIGGER vault_secrets_set_updated_at\n    BEFORE UPDATE ON vault_secrets\n    FOR EACH ROW EXECUTE FUNCTION set_updated_at();\nINSERT INTO roles") {
		t.Fatal("notification and audit triggers must be complete before role seed data begins")
	}
}
