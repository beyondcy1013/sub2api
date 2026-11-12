package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// nullableJSONB maps a nil RawMessage to SQL NULL so empty rule results store
// as NULL instead of the jsonb literal 'null'.
func nullableJSONB(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return []byte(raw)
}

// --- Plan Repository ---

type scheduledTestPlanRepository struct {
	db *sql.DB
}

func NewScheduledTestPlanRepository(db *sql.DB) service.ScheduledTestPlanRepository {
	return &scheduledTestPlanRepository{db: db}
}

func (r *scheduledTestPlanRepository) Create(ctx context.Context, plan *service.ScheduledTestPlan) (*service.ScheduledTestPlan, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO scheduled_test_plans (account_id, model_id, cron_expression, enabled, max_results, auto_recover, auto_recover_schedulable, mode, auto_pause_on_downgrade, rule_ids, rule_strategy, downgrade_action, downgrade_threshold, next_run_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
		RETURNING id, account_id, model_id, cron_expression, enabled, max_results, auto_recover, auto_recover_schedulable, mode, auto_pause_on_downgrade, rule_ids, rule_strategy, downgrade_action, downgrade_threshold, consecutive_downgrades, last_rule_index, last_run_at, next_run_at, created_at, updated_at
	`, plan.AccountID, plan.ModelID, plan.CronExpression, plan.Enabled, plan.MaxResults, plan.AutoRecover, plan.AutoRecoverSchedulable, plan.Mode, plan.AutoPauseOnDowngrade,
		pq.Array(plan.RuleIDs), plan.RuleStrategy, plan.DowngradeAction, plan.DowngradeThreshold, plan.NextRunAt)
	return scanPlan(row)
}

func (r *scheduledTestPlanRepository) GetByID(ctx context.Context, id int64) (*service.ScheduledTestPlan, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, account_id, model_id, cron_expression, enabled, max_results, auto_recover, auto_recover_schedulable, mode, auto_pause_on_downgrade, rule_ids, rule_strategy, downgrade_action, downgrade_threshold, consecutive_downgrades, last_rule_index, last_run_at, next_run_at, created_at, updated_at
		FROM scheduled_test_plans WHERE id = $1
	`, id)
	return scanPlan(row)
}

func (r *scheduledTestPlanRepository) ListByAccountID(ctx context.Context, accountID int64) ([]*service.ScheduledTestPlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, account_id, model_id, cron_expression, enabled, max_results, auto_recover, auto_recover_schedulable, mode, auto_pause_on_downgrade, rule_ids, rule_strategy, downgrade_action, downgrade_threshold, consecutive_downgrades, last_rule_index, last_run_at, next_run_at, created_at, updated_at
		FROM scheduled_test_plans WHERE account_id = $1
		ORDER BY created_at DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlans(rows)
}

func (r *scheduledTestPlanRepository) ListDue(ctx context.Context, now time.Time) ([]*service.ScheduledTestPlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT plan.id, plan.account_id, plan.model_id, plan.cron_expression, plan.enabled,
			plan.max_results, plan.auto_recover, plan.auto_recover_schedulable,
			plan.mode, plan.auto_pause_on_downgrade,
			plan.rule_ids, plan.rule_strategy, plan.downgrade_action, plan.downgrade_threshold,
			plan.consecutive_downgrades, plan.last_rule_index,
			plan.last_run_at, plan.next_run_at, plan.created_at, plan.updated_at
		FROM scheduled_test_plans AS plan
		JOIN accounts AS account ON account.id = plan.account_id
		WHERE plan.enabled = true AND plan.next_run_at <= $1
		  AND account.deleted_at IS NULL
		  AND COALESCE(account.extra -> 'deleted', 'false'::jsonb) <> 'true'::jsonb
		ORDER BY plan.next_run_at ASC
	`, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlans(rows)
}

func (r *scheduledTestPlanRepository) ListAll(ctx context.Context) ([]*service.ScheduledTestPlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT plan.id, plan.account_id, plan.model_id, plan.cron_expression, plan.enabled,
			plan.max_results, plan.auto_recover, plan.auto_recover_schedulable,
			plan.mode, plan.auto_pause_on_downgrade,
			plan.rule_ids, plan.rule_strategy, plan.downgrade_action, plan.downgrade_threshold,
			plan.consecutive_downgrades, plan.last_rule_index,
			plan.last_run_at, plan.next_run_at, plan.created_at, plan.updated_at,
			account.name, account.platform
		FROM scheduled_test_plans AS plan
		JOIN accounts AS account ON account.id = plan.account_id
		WHERE account.deleted_at IS NULL
		  AND COALESCE(account.extra -> 'deleted', 'false'::jsonb) <> 'true'::jsonb
		ORDER BY plan.next_run_at ASC NULLS LAST, account.name ASC, plan.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var plans []*service.ScheduledTestPlan
	for rows.Next() {
		plan, err := scanPlanWithAccount(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

func (r *scheduledTestPlanRepository) DeleteByAccountAndCron(ctx context.Context, accountID int64, cronExpression string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM scheduled_test_plans
		WHERE account_id = $1 AND cron_expression = $2
	`, accountID, cronExpression)
	return err
}

func (r *scheduledTestPlanRepository) Update(ctx context.Context, plan *service.ScheduledTestPlan) (*service.ScheduledTestPlan, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE scheduled_test_plans
		SET model_id = $2, cron_expression = $3, enabled = $4, max_results = $5, auto_recover = $6, auto_recover_schedulable = $7, mode = $8, auto_pause_on_downgrade = $9,
		    rule_ids = $10, rule_strategy = $11, downgrade_action = $12, downgrade_threshold = $13, next_run_at = $14, updated_at = NOW()
		WHERE id = $1
		RETURNING id, account_id, model_id, cron_expression, enabled, max_results, auto_recover, auto_recover_schedulable, mode, auto_pause_on_downgrade, rule_ids, rule_strategy, downgrade_action, downgrade_threshold, consecutive_downgrades, last_rule_index, last_run_at, next_run_at, created_at, updated_at
	`, plan.ID, plan.ModelID, plan.CronExpression, plan.Enabled, plan.MaxResults, plan.AutoRecover, plan.AutoRecoverSchedulable, plan.Mode, plan.AutoPauseOnDowngrade,
		pq.Array(plan.RuleIDs), plan.RuleStrategy, plan.DowngradeAction, plan.DowngradeThreshold, plan.NextRunAt)
	return scanPlan(row)
}

func (r *scheduledTestPlanRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM scheduled_test_plans WHERE id = $1`, id)
	return err
}

func (r *scheduledTestPlanRepository) UpdateAfterRun(ctx context.Context, id int64, lastRunAt time.Time, nextRunAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_test_plans SET last_run_at = $2, next_run_at = $3, updated_at = NOW() WHERE id = $1
	`, id, lastRunAt, nextRunAt)
	return err
}

func (r *scheduledTestPlanRepository) BumpConsecutiveDowngrades(ctx context.Context, id int64) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `
		UPDATE scheduled_test_plans SET consecutive_downgrades = consecutive_downgrades + 1, updated_at = NOW() WHERE id = $1
		RETURNING consecutive_downgrades
	`, id).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *scheduledTestPlanRepository) ResetConsecutiveDowngrades(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_test_plans SET consecutive_downgrades = 0, updated_at = NOW() WHERE id = $1
	`, id)
	return err
}

func (r *scheduledTestPlanRepository) BumpLastRuleIndex(ctx context.Context, id int64, selected int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_test_plans SET last_rule_index = $2, updated_at = NOW() WHERE id = $1
	`, id, selected)
	return err
}

// --- Result Repository ---

type scheduledTestResultRepository struct {
	db *sql.DB
}

func NewScheduledTestResultRepository(db *sql.DB) service.ScheduledTestResultRepository {
	return &scheduledTestResultRepository{db: db}
}

func (r *scheduledTestResultRepository) Create(ctx context.Context, result *service.ScheduledTestResult) (*service.ScheduledTestResult, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO scheduled_test_results (plan_id, status, response_text, error_message, downgraded, downgrade_reason, response_model, rule_results, latency_ms, started_at, finished_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		RETURNING id, plan_id, status, response_text, error_message, downgraded, downgrade_reason, response_model, rule_results, latency_ms, started_at, finished_at, created_at
	`, result.PlanID, result.Status, result.ResponseText, result.ErrorMessage, result.Downgraded, result.DowngradeReason, result.ResponseModel,
		nullableJSONB(result.RuleResults), result.LatencyMs, result.StartedAt, result.FinishedAt)

	out := &service.ScheduledTestResult{}
	if err := row.Scan(
		&out.ID, &out.PlanID, &out.Status, &out.ResponseText, &out.ErrorMessage,
		&out.Downgraded, &out.DowngradeReason, &out.ResponseModel, &out.RuleResults,
		&out.LatencyMs, &out.StartedAt, &out.FinishedAt, &out.CreatedAt,
	); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *scheduledTestResultRepository) ListByPlanID(ctx context.Context, planID int64, limit int) ([]*service.ScheduledTestResult, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, plan_id, status, response_text, error_message, downgraded, downgrade_reason, response_model, rule_results, latency_ms, started_at, finished_at, created_at
		FROM scheduled_test_results
		WHERE plan_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, planID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []*service.ScheduledTestResult
	for rows.Next() {
		r := &service.ScheduledTestResult{}
		if err := rows.Scan(
			&r.ID, &r.PlanID, &r.Status, &r.ResponseText, &r.ErrorMessage,
		&r.Downgraded, &r.DowngradeReason, &r.ResponseModel, &r.RuleResults,
		&r.LatencyMs, &r.StartedAt, &r.FinishedAt, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (r *scheduledTestResultRepository) PruneOldResults(ctx context.Context, planID int64, keepCount int) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM scheduled_test_results
		WHERE id IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY plan_id ORDER BY created_at DESC) AS rn
				FROM scheduled_test_results
				WHERE plan_id = $1
			) ranked
			WHERE rn > $2
		)
	`, planID, keepCount)
	return err
}

// --- scan helpers ---

type scannable interface {
	Scan(dest ...any) error
}

func scanPlan(row scannable) (*service.ScheduledTestPlan, error) {
	p := &service.ScheduledTestPlan{}
	if err := row.Scan(
		&p.ID, &p.AccountID, &p.ModelID, &p.CronExpression, &p.Enabled, &p.MaxResults, &p.AutoRecover, &p.AutoRecoverSchedulable,
		&p.Mode, &p.AutoPauseOnDowngrade,
		pq.Array(&p.RuleIDs), &p.RuleStrategy, &p.DowngradeAction, &p.DowngradeThreshold, &p.ConsecutiveDowngrades, &p.LastRuleIndex,
		&p.LastRunAt, &p.NextRunAt, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return p, nil
}

func scanPlans(rows *sql.Rows) ([]*service.ScheduledTestPlan, error) {
	var plans []*service.ScheduledTestPlan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

func scanPlanWithAccount(row scannable) (*service.ScheduledTestPlan, error) {
	plan := &service.ScheduledTestPlan{}
	if err := row.Scan(
		&plan.ID, &plan.AccountID, &plan.ModelID, &plan.CronExpression, &plan.Enabled, &plan.MaxResults,
		&plan.AutoRecover, &plan.AutoRecoverSchedulable, &plan.Mode, &plan.AutoPauseOnDowngrade,
		pq.Array(&plan.RuleIDs), &plan.RuleStrategy, &plan.DowngradeAction, &plan.DowngradeThreshold, &plan.ConsecutiveDowngrades, &plan.LastRuleIndex,
		&plan.LastRunAt, &plan.NextRunAt,
		&plan.CreatedAt, &plan.UpdatedAt, &plan.AccountName, &plan.AccountPlatform,
	); err != nil {
		return nil, err
	}
	return plan, nil
}
