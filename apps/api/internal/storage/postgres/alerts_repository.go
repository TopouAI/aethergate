package postgres

import (
	"context"
	"encoding/json"
	"github.com/topoai/aethergate/apps/api/internal/platform"
	"time"
)

func (r *Repository) ListAlertRules(ctx context.Context, f platform.AlertFilter) ([]platform.AlertRule, error) {
	rows, err := r.pool.Query(ctx, `SELECT a.id,a.organization_id,a.name,a.status,a.metric,a.operator,a.threshold,a.window_unit,a.cooldown_minutes,a.severity,a.channels,a.filters,count(i.id),a.last_triggered_at,a.created_at,a.updated_at FROM alert_rules a LEFT JOIN alert_incidents i ON i.rule_id=a.id WHERE a.organization_id=$1 AND a.deleted_at IS NULL AND ($2='' OR $2='all' OR a.status=$2) AND ($3='' OR $3='all' OR a.severity=$3) AND ($4='' OR lower(a.name) LIKE '%'||lower($4)||'%' OR lower(a.metric) LIKE '%'||lower($4)||'%') GROUP BY a.id ORDER BY a.created_at DESC`, f.OrganizationID, f.Status, f.Severity, f.Query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]platform.AlertRule, 0)
	for rows.Next() {
		var a platform.AlertRule
		var raw []byte
		if err := rows.Scan(&a.ID, &a.OrganizationID, &a.Name, &a.Status, &a.Metric, &a.Operator, &a.Threshold, &a.Window, &a.CooldownMinutes, &a.Severity, &a.Channels, &raw, &a.IncidentCount, &a.LastTriggeredAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &a.Filters)
		out = append(out, a)
	}
	return out, rows.Err()
}
func (r *Repository) GetAlertRule(ctx context.Context, org, id string) (platform.AlertRule, error) {
	items, err := r.ListAlertRules(ctx, platform.AlertFilter{OrganizationID: org})
	if err != nil {
		return platform.AlertRule{}, err
	}
	for _, a := range items {
		if a.ID == id {
			return a, nil
		}
	}
	return platform.AlertRule{}, platform.ErrNotFound
}
func (r *Repository) CreateAlertRule(ctx context.Context, a platform.AlertRule) (platform.AlertRule, error) {
	raw, _ := json.Marshal(a.Filters)
	_, err := r.pool.Exec(ctx, `INSERT INTO alert_rules(id,organization_id,name,status,metric,operator,threshold,window_unit,cooldown_minutes,severity,channels,filters,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, a.ID, a.OrganizationID, a.Name, a.Status, a.Metric, a.Operator, a.Threshold, a.Window, a.CooldownMinutes, a.Severity, a.Channels, raw, a.CreatedAt, a.UpdatedAt)
	if isForeignKeyViolation(err) {
		return platform.AlertRule{}, platform.ErrNotFound
	}
	if isUniqueViolation(err) {
		return platform.AlertRule{}, platform.ErrConflict
	}
	return a, err
}
func (r *Repository) UpdateAlertRuleStatus(ctx context.Context, org, id, status string, updated time.Time) (platform.AlertRule, error) {
	c, err := r.pool.Exec(ctx, `UPDATE alert_rules SET status=$3,updated_at=$4 WHERE organization_id=$1 AND id=$2 AND deleted_at IS NULL`, org, id, status, updated)
	if err != nil {
		return platform.AlertRule{}, err
	}
	if c.RowsAffected() == 0 {
		return platform.AlertRule{}, platform.ErrNotFound
	}
	return r.GetAlertRule(ctx, org, id)
}
func (r *Repository) ListAlertIncidents(ctx context.Context, f platform.AlertFilter) ([]platform.AlertIncident, error) {
	rows, err := r.pool.Query(ctx, `SELECT i.id,i.organization_id,i.rule_id,a.name,i.status,i.severity,i.metric,i.observed_value,i.threshold,i.summary,i.started_at,i.resolved_at FROM alert_incidents i JOIN alert_rules a ON a.id=i.rule_id WHERE i.organization_id=$1 AND ($2='' OR $2='all' OR i.status=$2) AND ($3='' OR $3='all' OR i.severity=$3) ORDER BY i.started_at DESC`, f.OrganizationID, f.Status, f.Severity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]platform.AlertIncident, 0)
	for rows.Next() {
		var i platform.AlertIncident
		if err := rows.Scan(&i.ID, &i.OrganizationID, &i.RuleID, &i.RuleName, &i.Status, &i.Severity, &i.Metric, &i.ObservedValue, &i.Threshold, &i.Summary, &i.StartedAt, &i.ResolvedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

var _ platform.AlertRepository = (*Repository)(nil)
