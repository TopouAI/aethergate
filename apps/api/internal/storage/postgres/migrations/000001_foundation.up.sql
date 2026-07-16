BEGIN;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE organizations (
    id text PRIMARY KEY,
    name text NOT NULL CHECK (length(btrim(name)) > 0),
    slug text NOT NULL CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    status text NOT NULL DEFAULT 'provisioning' CHECK (status IN ('active', 'provisioning', 'suspended')),
    plan text NOT NULL DEFAULT 'Evaluation',
    region text NOT NULL,
    owner_email text NOT NULL,
    monthly_cost_usd numeric(20, 8) NOT NULL DEFAULT 0 CHECK (monthly_cost_usd >= 0),
    budget_usd numeric(20, 8) NOT NULL DEFAULT 0 CHECK (budget_usd >= 0),
    request_count bigint NOT NULL DEFAULT 0 CHECK (request_count >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX organizations_slug_unique_active
    ON organizations (lower(slug))
    WHERE deleted_at IS NULL;

CREATE TABLE workspaces (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(btrim(name)) > 0),
    slug text NOT NULL CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    environment text NOT NULL DEFAULT 'production' CHECK (environment IN ('development', 'staging', 'production', 'shared')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    UNIQUE (organization_id, id)
);

CREATE UNIQUE INDEX workspaces_org_slug_unique_active
    ON workspaces (organization_id, lower(slug))
    WHERE deleted_at IS NULL;

CREATE TABLE projects (
    id text PRIMARY KEY,
    organization_id text NOT NULL,
    workspace_id text NOT NULL,
    name text NOT NULL CHECK (length(btrim(name)) > 0),
    slug text NOT NULL CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'archived')),
    owner_email text NOT NULL,
    budget_usd numeric(20, 8) NOT NULL DEFAULT 0 CHECK (budget_usd >= 0),
    monthly_cost_usd numeric(20, 8) NOT NULL DEFAULT 0 CHECK (monthly_cost_usd >= 0),
    request_count bigint NOT NULL DEFAULT 0 CHECK (request_count >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    FOREIGN KEY (organization_id, workspace_id)
        REFERENCES workspaces(organization_id, id) ON DELETE CASCADE,
    UNIQUE (organization_id, id)
);

CREATE UNIQUE INDEX projects_workspace_slug_unique_active
    ON projects (workspace_id, lower(slug))
    WHERE deleted_at IS NULL;

CREATE TABLE members (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email text NOT NULL,
    display_name text NOT NULL,
    status text NOT NULL DEFAULT 'invited' CHECK (status IN ('invited', 'active', 'suspended')),
    identity_provider text NOT NULL DEFAULT 'local',
    last_active_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    UNIQUE (organization_id, id)
);

CREATE UNIQUE INDEX members_org_email_unique_active
    ON members (organization_id, lower(email))
    WHERE deleted_at IS NULL;

CREATE TABLE roles (
    key text PRIMARY KEY,
    name text NOT NULL,
    description text NOT NULL,
    permissions text[] NOT NULL,
    system_role boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE role_bindings (
    id text PRIMARY KEY,
    organization_id text NOT NULL,
    member_id text NOT NULL,
    role_key text NOT NULL REFERENCES roles(key) ON DELETE RESTRICT,
    scope_type text NOT NULL CHECK (scope_type IN ('organization', 'workspace', 'project')),
    scope_id text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (organization_id, member_id)
        REFERENCES members(organization_id, id) ON DELETE CASCADE,
    UNIQUE (member_id, role_key, scope_type, scope_id)
);

CREATE INDEX role_bindings_authorization_lookup
    ON role_bindings (organization_id, member_id, scope_type, scope_id);

CREATE TABLE models (
    id text PRIMARY KEY,
    provider text NOT NULL,
    display_name text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'preview', 'deprecated', 'disabled')),
    context_window integer NOT NULL CHECK (context_window > 0),
    max_output_tokens integer NOT NULL CHECK (max_output_tokens > 0),
    input_price_per_million numeric(20, 8) NOT NULL DEFAULT 0 CHECK (input_price_per_million >= 0),
    output_price_per_million numeric(20, 8) NOT NULL DEFAULT 0 CHECK (output_price_per_million >= 0),
    supports_tools boolean NOT NULL DEFAULT false,
    supports_vision boolean NOT NULL DEFAULT false,
    supports_json boolean NOT NULL DEFAULT false,
    regions text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE provider_connections (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL,
    provider text NOT NULL,
    base_url text NOT NULL,
    status text NOT NULL DEFAULT 'configuring' CHECK (status IN ('configuring', 'healthy', 'degraded', 'offline', 'maintenance')),
    credential_state text NOT NULL DEFAULT 'missing' CHECK (credential_state IN ('missing', 'configured', 'rotating')),
    model_count integer NOT NULL DEFAULT 0 CHECK (model_count >= 0),
    p95_latency_ms integer NOT NULL DEFAULT 0 CHECK (p95_latency_ms >= 0),
    success_rate numeric(7, 4) NOT NULL DEFAULT 0 CHECK (success_rate >= 0 AND success_rate <= 100),
    last_checked_at timestamptz,
    routing_eligible boolean NOT NULL DEFAULT false,
    health_source text NOT NULL DEFAULT 'manual' CHECK (health_source IN ('manual', 'active_probe', 'passive_telemetry')),
    health_reason text NOT NULL DEFAULT '',
    error_rate numeric(9, 6) NOT NULL DEFAULT 0 CHECK (error_rate >= 0 AND error_rate <= 100),
    request_count_24h bigint NOT NULL DEFAULT 0 CHECK (request_count_24h >= 0),
    average_latency_ms integer NOT NULL DEFAULT 0 CHECK (average_latency_ms >= 0),
    consecutive_failures integer NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    last_transition_at timestamptz,
    maintenance_until timestamptz,
    maintenance_reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    UNIQUE (organization_id, id)
);

CREATE UNIQUE INDEX provider_connections_org_name_unique_active
    ON provider_connections (organization_id, lower(name))
    WHERE deleted_at IS NULL;
CREATE INDEX provider_connections_org_status_idx
    ON provider_connections (organization_id, status, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX provider_connections_routing_health_idx
    ON provider_connections (organization_id, routing_eligible, status, p95_latency_ms)
    WHERE deleted_at IS NULL;

CREATE TABLE provider_health_probes (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider_id text NOT NULL,
    status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
    region text NOT NULL,
    model text NOT NULL,
    requested_by text NOT NULL,
    requested_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    completed_at timestamptz,
    event_id text,
    error_message text NOT NULL DEFAULT '',
    FOREIGN KEY (organization_id, provider_id)
        REFERENCES provider_connections(organization_id, id) ON DELETE CASCADE
);

CREATE INDEX provider_health_probes_queue_idx
    ON provider_health_probes (status, requested_at)
    WHERE status IN ('queued', 'running');
CREATE INDEX provider_health_probes_provider_idx
    ON provider_health_probes (organization_id, provider_id, requested_at DESC);

CREATE TABLE provider_health_events (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider_id text NOT NULL,
    probe_id text REFERENCES provider_health_probes(id) ON DELETE SET NULL,
    source text NOT NULL CHECK (source IN ('active_probe', 'passive_telemetry')),
    previous_status text NOT NULL CHECK (previous_status IN ('configuring', 'healthy', 'degraded', 'offline', 'maintenance')),
    status text NOT NULL CHECK (status IN ('configuring', 'healthy', 'degraded', 'offline', 'maintenance')),
    is_transition boolean NOT NULL DEFAULT false,
    success boolean NOT NULL,
    routing_eligible boolean NOT NULL,
    request_count bigint NOT NULL CHECK (request_count >= 0),
    error_count bigint NOT NULL CHECK (error_count >= 0 AND error_count <= request_count),
    error_rate numeric(9, 6) NOT NULL CHECK (error_rate >= 0 AND error_rate <= 100),
    average_latency_ms integer NOT NULL DEFAULT 0 CHECK (average_latency_ms >= 0),
    p95_latency_ms integer NOT NULL DEFAULT 0 CHECK (p95_latency_ms >= 0),
    http_status integer CHECK (http_status BETWEEN 100 AND 599),
    consecutive_failures integer NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    reason text NOT NULL,
    observed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (organization_id, provider_id)
        REFERENCES provider_connections(organization_id, id) ON DELETE CASCADE
);

CREATE INDEX provider_health_events_provider_time_idx
    ON provider_health_events (organization_id, provider_id, observed_at DESC);
CREATE INDEX provider_health_events_transition_idx
    ON provider_health_events (organization_id, is_transition, observed_at DESC)
    WHERE is_transition = true;
CREATE INDEX provider_health_events_source_idx
    ON provider_health_events (organization_id, source, observed_at DESC);

CREATE TABLE routing_policies (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL,
    slug text NOT NULL,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'paused')),
    strategy text NOT NULL CHECK (strategy IN ('weighted', 'priority', 'latency')),
    model_pattern text NOT NULL,
    max_retries integer NOT NULL DEFAULT 2 CHECK (max_retries BETWEEN 0 AND 5),
    request_timeout_ms integer NOT NULL DEFAULT 30000 CHECK (request_timeout_ms BETWEEN 1000 AND 300000),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX routing_policies_org_slug_unique_active
    ON routing_policies (organization_id, lower(slug))
    WHERE deleted_at IS NULL;
CREATE INDEX routing_policies_org_status_idx
    ON routing_policies (organization_id, status, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE routing_targets (
    id text PRIMARY KEY,
    policy_id text NOT NULL REFERENCES routing_policies(id) ON DELETE CASCADE,
    provider_id text NOT NULL REFERENCES provider_connections(id) ON DELETE RESTRICT,
    model text NOT NULL,
    priority integer NOT NULL CHECK (priority BETWEEN 1 AND 100),
    weight integer NOT NULL DEFAULT 0 CHECK (weight BETWEEN 0 AND 100),
    enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (policy_id, provider_id, model)
);

CREATE INDEX routing_targets_policy_order_idx
    ON routing_targets (policy_id, priority, id);

CREATE TABLE alert_rules (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'enabled', 'disabled')),
    metric text NOT NULL CHECK (metric IN ('cost', 'error_rate', 'latency', 'requests', 'tokens', 'budget_utilization')),
    operator text NOT NULL CHECK (operator IN ('gt', 'gte', 'lt', 'lte')),
    threshold numeric(20,8) NOT NULL,
    window_unit text NOT NULL CHECK (window_unit IN ('5m', '15m', '1h', '24h')),
    cooldown_minutes integer NOT NULL DEFAULT 30 CHECK (cooldown_minutes BETWEEN 0 AND 10080),
    severity text NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    channels text[] NOT NULL,
    filters jsonb NOT NULL DEFAULT '{}',
    last_triggered_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX alert_rules_org_name_unique_active
    ON alert_rules (organization_id, lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX alert_rules_evaluation_idx
    ON alert_rules (organization_id, status, metric, severity)
    WHERE deleted_at IS NULL;

CREATE TABLE alert_incidents (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    rule_id text NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    status text NOT NULL CHECK (status IN ('open', 'acknowledged', 'resolved')),
    severity text NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    metric text NOT NULL,
    observed_value numeric(20,8) NOT NULL,
    threshold numeric(20,8) NOT NULL,
    summary text NOT NULL,
    started_at timestamptz NOT NULL,
    resolved_at timestamptz,
    acknowledged_by text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX alert_incidents_org_status_idx
    ON alert_incidents (organization_id, status, started_at DESC);
CREATE INDEX alert_incidents_rule_idx
    ON alert_incidents (rule_id, started_at DESC);

CREATE TABLE webhook_endpoints (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(btrim(name)) > 0),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    destination text NOT NULL CHECK (destination ~ '^https?://'),
    version text NOT NULL,
    events text[] NOT NULL CHECK (cardinality(events) > 0),
    sample_rate numeric(6,3) NOT NULL CHECK (sample_rate >= 0.1 AND sample_rate <= 100),
    include_data boolean NOT NULL DEFAULT true,
    property_filters jsonb NOT NULL DEFAULT '[]',
    signing_secret_prefix text NOT NULL,
    signing_secret_digest bytea NOT NULL CHECK (octet_length(signing_secret_digest) = 32),
    secret_reference text NOT NULL,
    max_attempts integer NOT NULL DEFAULT 5 CHECK (max_attempts BETWEEN 1 AND 10),
    timeout_seconds integer NOT NULL DEFAULT 10 CHECK (timeout_seconds BETWEEN 1 AND 30),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    UNIQUE (organization_id, id)
);

CREATE UNIQUE INDEX webhook_endpoints_org_name_unique_active
    ON webhook_endpoints (organization_id, lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX webhook_endpoints_org_status_idx
    ON webhook_endpoints (organization_id, status, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX webhook_endpoints_events_idx
    ON webhook_endpoints USING gin (events) WHERE deleted_at IS NULL;

CREATE TABLE webhook_deliveries (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    webhook_id text NOT NULL,
    event_id text NOT NULL,
    event_type text NOT NULL CHECK (event_type IN (
        'request.completed', 'request.failed', 'alert.triggered', 'alert.resolved',
        'budget.threshold_reached', 'api_key.revoked', 'provider.health_changed'
    )),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'delivering', 'succeeded', 'failed', 'dead_letter')),
    trigger_type text NOT NULL DEFAULT 'event' CHECK (trigger_type IN ('event', 'test', 'retry', 'replay')),
    attempt integer NOT NULL DEFAULT 1 CHECK (attempt > 0),
    max_attempts integer NOT NULL CHECK (max_attempts BETWEEN 1 AND 10),
    response_status integer CHECK (response_status BETWEEN 100 AND 599),
    duration_ms integer NOT NULL DEFAULT 0 CHECK (duration_ms >= 0),
    error_message text NOT NULL DEFAULT '',
    next_retry_at timestamptz,
    delivered_at timestamptz,
    replay_of_id text REFERENCES webhook_deliveries(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (organization_id, webhook_id)
        REFERENCES webhook_endpoints(organization_id, id) ON DELETE CASCADE,
    UNIQUE (webhook_id, event_id, attempt)
);

CREATE INDEX webhook_deliveries_org_created_idx
    ON webhook_deliveries (organization_id, created_at DESC);
CREATE INDEX webhook_deliveries_webhook_status_idx
    ON webhook_deliveries (webhook_id, status, created_at DESC);
CREATE INDEX webhook_deliveries_retry_idx
    ON webhook_deliveries (next_retry_at)
    WHERE status = 'failed' AND next_retry_at IS NOT NULL;
CREATE INDEX webhook_deliveries_event_idx
    ON webhook_deliveries (webhook_id, event_id, attempt DESC);

CREATE TABLE report_schedules (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(btrim(name)) > 0),
    template text NOT NULL CHECK (template IN ('executive_summary', 'usage_cost', 'reliability', 'adoption', 'raw_export')),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused')),
    frequency text NOT NULL CHECK (frequency IN ('daily', 'weekly', 'monthly')),
    day_of_week text NOT NULL DEFAULT '' CHECK (day_of_week IN ('', 'sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday')),
    day_of_month integer NOT NULL DEFAULT 0 CHECK (day_of_month BETWEEN 0 AND 28),
    local_time text NOT NULL CHECK (local_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    timezone text NOT NULL CHECK (length(btrim(timezone)) > 0),
    formats text[] NOT NULL CHECK (cardinality(formats) > 0 AND formats <@ ARRAY['csv','xlsx','pdf']::text[]),
    recipients jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(recipients) = 'array' AND jsonb_array_length(recipients) > 0),
    filters jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(filters) = 'object'),
    include_raw_data boolean NOT NULL DEFAULT false,
    last_run_at timestamptz,
    next_run_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    UNIQUE (organization_id, id),
    CHECK ((frequency = 'weekly' AND day_of_week <> '') OR (frequency <> 'weekly' AND day_of_week = '')),
    CHECK ((frequency = 'monthly' AND day_of_month BETWEEN 1 AND 28) OR (frequency <> 'monthly' AND day_of_month = 0)),
    CHECK ((status = 'active' AND next_run_at IS NOT NULL) OR status = 'paused')
);

CREATE UNIQUE INDEX report_schedules_org_name_unique_active
    ON report_schedules (organization_id, lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX report_schedules_due_idx
    ON report_schedules (next_run_at)
    WHERE status = 'active' AND deleted_at IS NULL;
CREATE INDEX report_schedules_org_status_idx
    ON report_schedules (organization_id, status, created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE report_runs (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    report_id text NOT NULL,
    status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
    trigger_type text NOT NULL CHECK (trigger_type IN ('schedule', 'manual', 'retry')),
    attempt integer NOT NULL DEFAULT 1 CHECK (attempt > 0),
    requested_by text NOT NULL,
    scheduled_for timestamptz NOT NULL,
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    started_at timestamptz,
    completed_at timestamptz,
    artifact_count integer NOT NULL DEFAULT 0 CHECK (artifact_count >= 0),
    row_count bigint NOT NULL DEFAULT 0 CHECK (row_count >= 0),
    size_bytes bigint NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    delivery_status text NOT NULL DEFAULT 'pending' CHECK (delivery_status IN ('pending', 'delivered', 'partial', 'failed')),
    error_message text NOT NULL DEFAULT '',
    parent_run_id text REFERENCES report_runs(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (organization_id, report_id)
        REFERENCES report_schedules(organization_id, id) ON DELETE CASCADE,
    CHECK (period_end > period_start)
);

CREATE INDEX report_runs_queue_idx
    ON report_runs (status, scheduled_for) WHERE status IN ('queued', 'running');
CREATE INDEX report_runs_report_created_idx
    ON report_runs (organization_id, report_id, created_at DESC);
CREATE INDEX report_runs_org_status_idx
    ON report_runs (organization_id, status, created_at DESC);

CREATE TABLE notification_preferences (
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    recipient_id text NOT NULL CHECK (length(btrim(recipient_id)) > 0),
    destinations jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(destinations) = 'array' AND jsonb_array_length(destinations) > 0),
    category_channels jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(category_channels) = 'object'),
    digest_frequency text NOT NULL DEFAULT 'realtime' CHECK (digest_frequency IN ('realtime', 'hourly', 'daily', 'weekly')),
    minimum_severity text NOT NULL DEFAULT 'info' CHECK (minimum_severity IN ('info', 'warning', 'critical')),
    timezone text NOT NULL DEFAULT 'UTC' CHECK (length(btrim(timezone)) > 0),
    quiet_hours_enabled boolean NOT NULL DEFAULT false,
    quiet_start text NOT NULL DEFAULT '22:00' CHECK (quiet_start ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    quiet_end text NOT NULL DEFAULT '08:00' CHECK (quiet_end ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, recipient_id),
    CHECK (NOT quiet_hours_enabled OR quiet_start <> quiet_end)
);

CREATE TABLE notifications (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    recipient_id text NOT NULL CHECK (length(btrim(recipient_id)) > 0),
    category text NOT NULL CHECK (category IN ('alert', 'budget', 'provider', 'report', 'access', 'security', 'platform')),
    severity text NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    title text NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 180),
    body text NOT NULL CHECK (length(btrim(body)) BETWEEN 1 AND 4000),
    source_type text NOT NULL DEFAULT '',
    source_id text NOT NULL DEFAULT '',
    action_url text NOT NULL DEFAULT '' CHECK (action_url = '' OR left(action_url, 1) = '/'),
    status text NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'archived')),
    read_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, id),
    CHECK ((status = 'unread' AND read_at IS NULL) OR (status IN ('read', 'archived') AND read_at IS NOT NULL))
);

CREATE INDEX notifications_recipient_status_idx
    ON notifications (organization_id, recipient_id, status, created_at DESC);
CREATE INDEX notifications_recipient_category_idx
    ON notifications (organization_id, recipient_id, category, created_at DESC);
CREATE INDEX notifications_source_idx
    ON notifications (organization_id, source_type, source_id);

CREATE TABLE notification_escalation_policies (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(btrim(name)) > 0),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused')),
    categories text[] NOT NULL CHECK (cardinality(categories) > 0 AND categories <@ ARRAY['alert','budget','provider','report','access','security','platform']::text[]),
    minimum_severity text NOT NULL CHECK (minimum_severity IN ('info', 'warning', 'critical')),
    acknowledge_within_minutes integer NOT NULL CHECK (acknowledge_within_minutes BETWEEN 1 AND 1440),
    repeat_every_minutes integer NOT NULL DEFAULT 0 CHECK (repeat_every_minutes BETWEEN 0 AND 1440),
    max_escalations integer NOT NULL DEFAULT 1 CHECK (max_escalations BETWEEN 1 AND 5),
    routes jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(routes) = 'array' AND jsonb_array_length(routes) > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    UNIQUE (organization_id, id)
);

CREATE UNIQUE INDEX notification_policies_org_name_unique_active
    ON notification_escalation_policies (organization_id, lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX notification_policies_org_status_idx
    ON notification_escalation_policies (organization_id, status, created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE notification_deliveries (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    notification_id text NOT NULL,
    channel text NOT NULL CHECK (channel IN ('email', 'slack', 'teams', 'webhook')),
    target text NOT NULL CHECK (length(btrim(target)) > 0),
    display_name text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'deferred', 'sending', 'delivered', 'failed', 'suppressed')),
    attempt integer NOT NULL DEFAULT 1 CHECK (attempt > 0),
    available_at timestamptz NOT NULL DEFAULT now(),
    delivered_at timestamptz,
    error_message text NOT NULL DEFAULT '',
    parent_delivery_id text REFERENCES notification_deliveries(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (organization_id, notification_id)
        REFERENCES notifications(organization_id, id) ON DELETE CASCADE,
    CHECK ((status = 'delivered' AND delivered_at IS NOT NULL) OR status <> 'delivered')
);

CREATE INDEX notification_deliveries_queue_idx
    ON notification_deliveries (available_at)
    WHERE status IN ('queued', 'deferred');
CREATE INDEX notification_deliveries_notification_idx
    ON notification_deliveries (organization_id, notification_id, created_at DESC);
CREATE INDEX notification_deliveries_org_status_idx
    ON notification_deliveries (organization_id, status, created_at DESC);

CREATE TABLE budgets (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'paused')),
    scope_type text NOT NULL CHECK (scope_type IN ('organization', 'workspace', 'project')),
    scope_id text NOT NULL,
    period text NOT NULL CHECK (period IN ('monthly', 'quarterly', 'annual')),
    limit_usd numeric(20,8) NOT NULL CHECK (limit_usd > 0),
    warning_percent integer NOT NULL CHECK (warning_percent BETWEEN 1 AND 99),
    critical_percent integer NOT NULL CHECK (critical_percent BETWEEN 2 AND 100 AND critical_percent > warning_percent),
    action text NOT NULL CHECK (action IN ('alert', 'block', 'approval')),
    spent_usd numeric(20,8) NOT NULL DEFAULT 0 CHECK (spent_usd >= 0),
    committed_usd numeric(20,8) NOT NULL DEFAULT 0 CHECK (committed_usd >= 0),
    forecast_usd numeric(20,8) NOT NULL DEFAULT 0 CHECK (forecast_usd >= 0),
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CHECK (ends_at > starts_at)
);

CREATE UNIQUE INDEX budgets_org_name_unique_active
    ON budgets (organization_id, lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX budgets_scope_status_idx
    ON budgets (organization_id, status, scope_type, scope_id, ends_at)
    WHERE deleted_at IS NULL;

CREATE TABLE rate_limit_rules (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'enforced', 'disabled')),
    scope_type text NOT NULL CHECK (scope_type IN ('organization', 'workspace', 'project', 'api_key', 'user')),
    scope_id text NOT NULL,
    metric text NOT NULL CHECK (metric IN ('requests', 'tokens', 'concurrency')),
    window_unit text NOT NULL CHECK (window_unit IN ('second', 'minute', 'hour', 'day')),
    limit_value bigint NOT NULL CHECK (limit_value > 0),
    burst bigint NOT NULL DEFAULT 0 CHECK (burst >= 0),
    action text NOT NULL CHECK (action IN ('reject', 'throttle', 'observe')),
    priority integer NOT NULL DEFAULT 100 CHECK (priority BETWEEN 0 AND 1000),
    matched_requests bigint NOT NULL DEFAULT 0 CHECK (matched_requests >= 0),
    limited_requests bigint NOT NULL DEFAULT 0 CHECK (limited_requests >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX rate_limit_rules_org_name_unique_active
    ON rate_limit_rules (organization_id, lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX rate_limit_rules_lookup_idx
    ON rate_limit_rules (organization_id, status, scope_type, scope_id, metric, priority DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE api_keys (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id text,
    project_name text NOT NULL,
    name text NOT NULL,
    prefix text NOT NULL,
    secret_digest bytea NOT NULL CHECK (octet_length(secret_digest) = 32),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'expired')),
    rpm integer NOT NULL CHECK (rpm > 0),
    tpm integer NOT NULL CHECK (tpm > 0),
    spend_usd numeric(20, 8) NOT NULL DEFAULT 0 CHECK (spend_usd >= 0),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,
    expires_at timestamptz,
    revoked_at timestamptz,
    revoked_by text,
    FOREIGN KEY (organization_id, project_id)
        REFERENCES projects(organization_id, id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX api_keys_prefix_unique ON api_keys(prefix);
CREATE INDEX api_keys_org_status_created_idx ON api_keys(organization_id, status, created_at DESC);
CREATE INDEX api_keys_project_idx ON api_keys(project_id) WHERE project_id IS NOT NULL;

CREATE TABLE api_key_models (
    api_key_id text NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    model_id text NOT NULL REFERENCES models(id) ON DELETE RESTRICT,
    PRIMARY KEY (api_key_id, model_id)
);

CREATE TABLE vault_secrets (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(btrim(name)) > 0),
    kind text NOT NULL CHECK (kind IN ('provider_api_key', 'webhook_signing_secret', 'integration_token', 'smtp_password', 'object_storage_key', 'database_credential', 'generic')),
    scope_type text NOT NULL CHECK (scope_type IN ('provider', 'webhook', 'notification', 'reporting', 'gateway', 'integration', 'organization')),
    scope_id text NOT NULL CHECK (length(btrim(scope_id)) > 0),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    reference text NOT NULL UNIQUE CHECK (reference ~ '^vault://[A-Za-z0-9_-]+/[A-Za-z0-9_-]+$'),
    masked_value text NOT NULL,
    fingerprint text NOT NULL CHECK (fingerprint ~ '^[0-9a-f]{16}$'),
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version > 0),
    rotation_interval_days integer NOT NULL DEFAULT 90 CHECK (rotation_interval_days BETWEEN 1 AND 365),
    last_rotated_at timestamptz NOT NULL,
    rotation_due_at timestamptz NOT NULL,
    expires_at timestamptz,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    disabled_at timestamptz,
    disabled_by text NOT NULL DEFAULT '',
    disabled_reason text NOT NULL DEFAULT '',
    deleted_at timestamptz,
    UNIQUE (organization_id, id),
    CHECK (expires_at IS NULL OR expires_at > created_at)
);

CREATE UNIQUE INDEX vault_secrets_org_name_unique_active
    ON vault_secrets (organization_id, lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX vault_secrets_scope_status_idx
    ON vault_secrets (organization_id, scope_type, scope_id, status) WHERE deleted_at IS NULL;
CREATE INDEX vault_secrets_rotation_due_idx
    ON vault_secrets (rotation_due_at) WHERE status = 'active' AND deleted_at IS NULL;

CREATE TABLE vault_secret_versions (
    secret_id text NOT NULL,
    organization_id text NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    state text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'superseded', 'disabled')),
    ciphertext bytea NOT NULL CHECK (octet_length(ciphertext) >= 16),
    secret_nonce bytea NOT NULL CHECK (octet_length(secret_nonce) = 12),
    encrypted_data_key bytea NOT NULL CHECK (octet_length(encrypted_data_key) >= 48),
    key_nonce bytea NOT NULL CHECK (octet_length(key_nonce) = 12),
    key_version text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (secret_id, version),
    FOREIGN KEY (organization_id, secret_id)
        REFERENCES vault_secrets(organization_id, id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX vault_secret_versions_one_active
    ON vault_secret_versions (secret_id) WHERE state = 'active';
CREATE INDEX vault_secret_versions_org_created_idx
    ON vault_secret_versions (organization_id, created_at DESC);

CREATE TABLE vault_access_events (
    id text PRIMARY KEY,
    organization_id text NOT NULL,
    secret_id text NOT NULL,
    secret_version integer NOT NULL CHECK (secret_version > 0),
    actor text NOT NULL,
    workload text NOT NULL,
    purpose text NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('success', 'denied', 'failure')),
    request_id text NOT NULL DEFAULT '',
    source_ip inet,
    error_code text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (organization_id, secret_id)
        REFERENCES vault_secrets(organization_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (secret_id, secret_version)
        REFERENCES vault_secret_versions(secret_id, version) ON DELETE RESTRICT
);

CREATE INDEX vault_access_events_secret_created_idx
    ON vault_access_events (organization_id, secret_id, created_at DESC);
CREATE INDEX vault_access_events_actor_outcome_idx
    ON vault_access_events (organization_id, actor, outcome, created_at DESC);

CREATE TABLE audit_events (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    actor_id text NOT NULL,
    actor_email text NOT NULL,
    action text NOT NULL,
    resource_type text NOT NULL,
    resource_id text NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('success', 'failure', 'denied')),
    risk_level text NOT NULL CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    source text NOT NULL DEFAULT 'control-plane',
    reason text NOT NULL DEFAULT '',
    request_id text NOT NULL DEFAULT '',
    ip_address inet,
    user_agent text NOT NULL DEFAULT '',
    before_state jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(before_state) = 'object'),
    after_state jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(after_state) = 'object'),
    previous_hash text NOT NULL DEFAULT '' CHECK (previous_hash = '' OR previous_hash ~ '^[0-9a-f]{64}$'),
    integrity_hash text NOT NULL CHECK (integrity_hash ~ '^[0-9a-f]{64}$'),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, previous_hash),
    UNIQUE (organization_id, integrity_hash)
);

CREATE INDEX audit_events_org_created_idx
    ON audit_events (organization_id, created_at DESC);
CREATE INDEX audit_events_resource_idx
    ON audit_events (organization_id, resource_type, resource_id, created_at DESC);
CREATE INDEX audit_events_actor_idx
    ON audit_events (organization_id, actor_email, created_at DESC);
CREATE INDEX audit_events_risk_outcome_idx
    ON audit_events (organization_id, risk_level, outcome, created_at DESC);

CREATE TABLE audit_retention_policies (
    organization_id text PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    retention_days integer NOT NULL DEFAULT 365 CHECK (retention_days BETWEEN 30 AND 2555),
    legal_hold boolean NOT NULL DEFAULT false,
    export_format text NOT NULL DEFAULT 'csv' CHECK (export_format IN ('csv', 'jsonl')),
    updated_by text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE audit_exports (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    requested_by text NOT NULL,
    format text NOT NULL CHECK (format IN ('csv', 'jsonl')),
    status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
    filters jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(filters) = 'object'),
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    row_count bigint NOT NULL DEFAULT 0 CHECK (row_count >= 0),
    size_bytes bigint NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    object_key text NOT NULL DEFAULT '',
    checksum text NOT NULL DEFAULT '' CHECK (checksum = '' OR checksum ~ '^[0-9a-f]{64}$'),
    error_message text NOT NULL DEFAULT '',
    parent_export_id text REFERENCES audit_exports(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CHECK (period_end > period_start)
);

CREATE INDEX audit_exports_queue_idx ON audit_exports (created_at) WHERE status = 'queued';
CREATE INDEX audit_exports_org_status_idx ON audit_exports (organization_id, status, created_at DESC);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION prevent_audit_event_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'audit_events are append-only' USING ERRCODE = '55000';
END;
$$;

CREATE OR REPLACE FUNCTION prevent_vault_access_event_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'vault_access_events are append-only' USING ERRCODE = '55000';
END;
$$;

CREATE TRIGGER audit_events_immutable BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_event_mutation();
CREATE TRIGGER vault_access_events_immutable BEFORE UPDATE OR DELETE ON vault_access_events
    FOR EACH ROW EXECUTE FUNCTION prevent_vault_access_event_mutation();

CREATE TRIGGER organizations_set_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER workspaces_set_updated_at
    BEFORE UPDATE ON workspaces
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER projects_set_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER members_set_updated_at
    BEFORE UPDATE ON members
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER models_set_updated_at
    BEFORE UPDATE ON models
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER provider_connections_set_updated_at
    BEFORE UPDATE ON provider_connections
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER routing_policies_set_updated_at
    BEFORE UPDATE ON routing_policies
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER rate_limit_rules_set_updated_at
    BEFORE UPDATE ON rate_limit_rules
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER budgets_set_updated_at
    BEFORE UPDATE ON budgets
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER alert_rules_set_updated_at
    BEFORE UPDATE ON alert_rules
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER webhook_endpoints_set_updated_at
    BEFORE UPDATE ON webhook_endpoints
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER report_schedules_set_updated_at
    BEFORE UPDATE ON report_schedules
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER notification_preferences_set_updated_at
    BEFORE UPDATE ON notification_preferences
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER notifications_set_updated_at
    BEFORE UPDATE ON notifications
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER notification_escalation_policies_set_updated_at
    BEFORE UPDATE ON notification_escalation_policies
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER audit_retention_policies_set_updated_at
    BEFORE UPDATE ON audit_retention_policies
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER vault_secrets_set_updated_at
    BEFORE UPDATE ON vault_secrets
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
INSERT INTO roles (key, name, description, permissions) VALUES
    ('owner', 'Owner', 'Full organization control.', ARRAY['*']),
    ('admin', 'Administrator', 'Manage tenant resources except ownership and billing contract changes.', ARRAY['organization:read', 'workspace:*', 'project:*', 'member:*', 'model-policy:*', 'api-key:*', 'budget:*', 'alert:*', 'webhook:*', 'report:*', 'notification:*', 'vault:*', 'audit:*', 'observability:*']),
    ('developer', 'Developer', 'Use projects, keys, playground, prompts, and observability.', ARRAY['workspace:read', 'project:read', 'api-key:create-self', 'api-key:read-self', 'playground:*', 'prompt:*', 'dataset:*', 'notification:read-self', 'notification:update-self', 'observability:read']),
    ('analyst', 'Analyst', 'Read observability, cost, quality, and exported reports.', ARRAY['workspace:read', 'project:read', 'observability:read', 'report:read', 'report:export', 'notification:read-self', 'notification:update-self', 'audit:read', 'audit:export']),
    ('billing', 'Billing administrator', 'Manage budgets, price books, statements, and reconciliation.', ARRAY['organization:read', 'project:read', 'budget:*', 'billing:*', 'report:read', 'report:export', 'notification:read-self', 'notification:update-self']),
    ('viewer', 'Viewer', 'Read approved enterprise and observability resources.', ARRAY['organization:read', 'workspace:read', 'project:read', 'notification:read-self', 'notification:update-self', 'observability:read'])
ON CONFLICT (key) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    permissions = EXCLUDED.permissions;

INSERT INTO models (
    id, provider, display_name, status, context_window, max_output_tokens,
    input_price_per_million, output_price_per_million,
    supports_tools, supports_vision, supports_json, regions
) VALUES
    ('claude-sonnet-4', 'Anthropic', 'Claude Sonnet 4', 'active', 200000, 64000, 3.00, 15.00, true, true, true, ARRAY['us', 'eu', 'apac']),
    ('gpt-5-mini', 'OpenAI', 'GPT-5 mini', 'active', 400000, 128000, 0.25, 2.00, true, true, true, ARRAY['us', 'eu', 'apac']),
    ('gemini-2.5-pro', 'Google', 'Gemini 2.5 Pro', 'active', 1048576, 65536, 1.25, 10.00, true, true, true, ARRAY['us', 'eu', 'apac']),
    ('deepseek-v3', 'DeepSeek', 'DeepSeek V3', 'active', 128000, 8192, 0.27, 1.10, true, false, true, ARRAY['apac'])
ON CONFLICT (id) DO NOTHING;

INSERT INTO schema_migrations(version) VALUES (1)
ON CONFLICT (version) DO NOTHING;

COMMIT;
