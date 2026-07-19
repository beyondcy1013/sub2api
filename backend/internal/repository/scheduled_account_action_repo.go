package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type scheduledAccountActionRepository struct {
	db *sql.DB
}

func NewScheduledAccountActionRepository(db *sql.DB) service.ScheduledAccountActionRepository {
	return &scheduledAccountActionRepository{db: db}
}

const scheduledAccountActionColumns = `
	id, account_id, action, execute_at, status, attempts, lease_until,
	last_error, created_at, updated_at, completed_at`

const scheduledAccountActionQualifiedColumns = `
	action.id, action.account_id, action.action, action.execute_at, action.status,
	action.attempts, action.lease_until, action.last_error, action.created_at,
	action.updated_at, action.completed_at`

func (r *scheduledAccountActionRepository) Upsert(ctx context.Context, action *service.ScheduledAccountAction) (*service.ScheduledAccountAction, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO scheduled_account_actions (
			account_id, action, execute_at, status, attempts, lease_until, last_error, created_at, updated_at, completed_at
		) VALUES ($1, $2, $3, 'pending', 0, NULL, NULL, NOW(), NOW(), NULL)
		ON CONFLICT (account_id) DO UPDATE SET
			action = EXCLUDED.action,
			execute_at = EXCLUDED.execute_at,
			status = 'pending',
			attempts = 0,
			lease_until = NULL,
			last_error = NULL,
			updated_at = NOW(),
			completed_at = NULL
		RETURNING `+scheduledAccountActionColumns,
		action.AccountID, action.Action, action.ExecuteAt,
	)
	return scanScheduledAccountAction(row)
}

func (r *scheduledAccountActionRepository) GetPendingByAccountID(ctx context.Context, accountID int64) (*service.ScheduledAccountAction, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT `+scheduledAccountActionColumns+`
		FROM scheduled_account_actions
		WHERE account_id = $1 AND status IN ('pending', 'processing')
	`, accountID)
	action, err := scanScheduledAccountAction(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return action, err
}

func (r *scheduledAccountActionRepository) DeletePendingByAccountID(ctx context.Context, accountID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM scheduled_account_actions
		WHERE account_id = $1 AND status = 'pending'
	`, accountID)
	return err
}

func (r *scheduledAccountActionRepository) ClaimDue(ctx context.Context, now, leaseUntil time.Time, limit int) ([]*service.ScheduledAccountAction, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH due AS (
			SELECT id
			FROM scheduled_account_actions
			WHERE execute_at <= $1
			  AND (status = 'pending' OR (status = 'processing' AND lease_until <= $1))
			ORDER BY execute_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE scheduled_account_actions AS action
		SET status = 'processing',
			attempts = action.attempts + 1,
			lease_until = $3,
			updated_at = NOW()
		FROM due
		WHERE action.id = due.id
		RETURNING `+scheduledAccountActionQualifiedColumns,
		now, limit, leaseUntil,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var actions []*service.ScheduledAccountAction
	for rows.Next() {
		action, err := scanScheduledAccountAction(rows)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, rows.Err()
}

func (r *scheduledAccountActionRepository) MarkCompleted(ctx context.Context, id int64, completedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_account_actions
		SET status = 'completed', lease_until = NULL, last_error = NULL,
			completed_at = $2, updated_at = NOW()
		WHERE id = $1 AND status = 'processing'
	`, id, completedAt)
	return err
}

func (r *scheduledAccountActionRepository) MarkFailed(ctx context.Context, id int64, message string, retryAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_account_actions
		SET status = 'pending', execute_at = $3, lease_until = NULL,
			last_error = $2, updated_at = NOW()
		WHERE id = $1 AND status = 'processing'
	`, id, message, retryAt)
	return err
}

func scanScheduledAccountAction(row scannable) (*service.ScheduledAccountAction, error) {
	action := &service.ScheduledAccountAction{}
	err := row.Scan(
		&action.ID,
		&action.AccountID,
		&action.Action,
		&action.ExecuteAt,
		&action.Status,
		&action.Attempts,
		&action.LeaseUntil,
		&action.LastError,
		&action.CreatedAt,
		&action.UpdatedAt,
		&action.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return action, nil
}
