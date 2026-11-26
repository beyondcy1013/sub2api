package repository

import (
	"context"
	"errors"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

var _ service.AccountQualityDegradationStateStore = (*accountRepository)(nil)

// RecordQualityDegradation atomically increments the consecutive confirmed
// degradation count and, when requested, pauses scheduling at the threshold.
func (r *accountRepository) RecordQualityDegradation(
	ctx context.Context,
	accountID int64,
	ruleFingerprint string,
	model string,
	reason string,
	threshold int,
	pauseOnDowngrade bool,
) (service.QualityDegradationState, error) {
	if r == nil {
		return service.QualityDegradationState{}, errors.New("account repository unavailable")
	}
	if scoped := r.accountTransactionScope(ctx); scoped != r {
		return scoped.RecordQualityDegradation(ctx, accountID, ruleFingerprint, model, reason, threshold, pauseOnDowngrade)
	}
	if r == nil || r.sql == nil {
		return service.QualityDegradationState{}, errors.New("account repository unavailable")
	}
	if threshold < 1 {
		threshold = 3
	}
	if threshold > 10 {
		threshold = 10
	}

	baseCtx := ctx
	contextTx := dbent.TxFromContext(ctx)
	exec := r.sql
	if contextTx != nil {
		exec = contextTx.Client()
	}
	var tx *dbent.Tx
	if contextTx == nil && r.client != nil {
		var err error
		tx, err = r.client.Tx(ctx)
		if err != nil {
			return service.QualityDegradationState{}, err
		}
		defer func() { _ = tx.Rollback() }()
		ctx = dbent.NewTxContext(ctx, tx)
		exec = tx
	}

	rows, err := exec.QueryContext(ctx, `
		WITH current_account AS (
			SELECT id, COALESCE(extra, '{}'::jsonb) AS extra
			FROM accounts
			WHERE id = $1 AND deleted_at IS NULL
			FOR UPDATE
		), computed AS (
			SELECT id, extra,
				CASE
					WHEN COALESCE(extra->'quality_degradation_state'->>'rule_fingerprint', '') = $2::text
					 AND COALESCE(extra->'quality_degradation_state'->>'consecutive_count', '') ~ '^[0-9]+$'
					THEN LEAST((extra->'quality_degradation_state'->>'consecutive_count')::int + 1, $6::integer)
					ELSE 1
				END AS next_count
			FROM current_account
		), updated AS (
			UPDATE accounts AS a
			SET extra = jsonb_set(
					c.extra,
					'{quality_degradation_state}',
					jsonb_build_object(
						'consecutive_count', c.next_count,
						'rule_fingerprint', $2::text,
						'last_model', $3::text,
						'last_reason', $4::text,
						'last_detected_at', $5::timestamptz,
						'threshold', $6::integer,
						'paused', CASE
							WHEN $7::boolean AND c.next_count >= $6::integer THEN true
							WHEN c.extra->'quality_degradation_state'->>'paused' IN ('true', 't', '1') THEN true
							ELSE false
						END
					),
					true
				),
				schedulable = CASE
					WHEN $7::boolean AND c.next_count >= $6::integer THEN false
					ELSE a.schedulable
				END,
				updated_at = NOW()
			FROM computed AS c
			WHERE a.id = c.id
			RETURNING a.id,
				(a.extra->'quality_degradation_state'->>'consecutive_count')::int AS consecutive_count,
				CASE WHEN a.extra->'quality_degradation_state'->>'paused' IN ('true', 't', '1') THEN true ELSE false END AS paused
		)
		SELECT consecutive_count, paused FROM updated
	`, accountID, ruleFingerprint, model, reason, time.Now(), threshold, pauseOnDowngrade)
	if err != nil {
		return service.QualityDegradationState{}, err
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return service.QualityDegradationState{}, err
		}
		if err := rows.Close(); err != nil {
			return service.QualityDegradationState{}, err
		}
		return service.QualityDegradationState{}, service.ErrAccountNotFound
	}
	state := service.QualityDegradationState{RuleFingerprint: ruleFingerprint, LastModel: model, LastReason: reason, Threshold: threshold}
	if err := rows.Scan(&state.ConsecutiveCount, &state.Paused); err != nil {
		_ = rows.Close()
		return service.QualityDegradationState{}, err
	}
	// lib/pq must consume/close the result set before the same transaction
	// connection is used to enqueue the scheduler event.
	if err := rows.Close(); err != nil {
		return service.QualityDegradationState{}, err
	}
	if err := rows.Err(); err != nil {
		return service.QualityDegradationState{}, err
	}

	// The account's schedulable bit is part of the same transaction. Keep the
	// outbox event in that transaction so cached schedulers cannot miss it.
	if err := enqueueSchedulerOutbox(ctx, exec, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
		return service.QualityDegradationState{}, err
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return service.QualityDegradationState{}, err
		}
	}
	if contextTx == nil {
		r.syncSchedulerAccountSnapshot(baseCtx, accountID)
	}
	return state, nil
}

// ResetQualityDegradation clears only the consecutive counter. A pause is
// intentionally retained until an operator explicitly enables scheduling.
func (r *accountRepository) ResetQualityDegradation(ctx context.Context, accountID int64, ruleFingerprint string) error {
	return r.ResetQualityDegradationAt(ctx, accountID, ruleFingerprint, time.Now())
}

// ResetQualityDegradationAt clears the consecutive counter only when the
// clean probe is not older than the latest persisted degradation result. This
// ordering guard matters when a clean and a failed probe finish concurrently:
// a delayed clean write must not erase a newer confirmed degradation.
func (r *accountRepository) ResetQualityDegradationAt(ctx context.Context, accountID int64, ruleFingerprint string, observedAt time.Time) error {
	if r == nil {
		return errors.New("account repository unavailable")
	}
	if scoped := r.accountTransactionScope(ctx); scoped != r {
		return scoped.ResetQualityDegradationAt(ctx, accountID, ruleFingerprint, observedAt)
	}
	if r == nil || r.sql == nil {
		return errors.New("account repository unavailable")
	}
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	result, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET extra = jsonb_set(
			COALESCE(extra, '{}'::jsonb),
			'{quality_degradation_state}',
			(COALESCE(extra, '{}'::jsonb)->'quality_degradation_state') ||
			jsonb_build_object('consecutive_count', 0, 'rule_fingerprint', $2::text),
			true
		), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		  AND COALESCE(extra, '{}'::jsonb) ? 'quality_degradation_state'
		  AND (
			COALESCE(extra->'quality_degradation_state'->>'rule_fingerprint', '') = ''
			OR extra->'quality_degradation_state'->>'rule_fingerprint' = $2
		  )
		  AND (
			COALESCE(extra->'quality_degradation_state'->>'last_detected_at', '') = ''
			OR (
				extra->'quality_degradation_state'->>'last_detected_at' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}'
				AND (extra->'quality_degradation_state'->>'last_detected_at')::timestamptz <= $3::timestamptz
			)
		  )
	`, accountID, ruleFingerprint, observedAt)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 && dbent.TxFromContext(ctx) == nil {
		r.syncSchedulerAccountSnapshotDetached(ctx, accountID)
	}
	return nil
}

// ClearQualityDegradationState is called by explicit operator recovery.
func (r *accountRepository) ClearQualityDegradationState(ctx context.Context, accountID int64) error {
	if r == nil {
		return errors.New("account repository unavailable")
	}
	if scoped := r.accountTransactionScope(ctx); scoped != r {
		return scoped.ClearQualityDegradationState(ctx, accountID)
	}
	if r == nil || r.sql == nil {
		return errors.New("account repository unavailable")
	}
	result, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET extra = COALESCE(extra, '{}'::jsonb) - 'quality_degradation_state', updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, accountID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAccountNotFound
	}
	if dbent.TxFromContext(ctx) == nil {
		r.syncSchedulerAccountSnapshotDetached(ctx, accountID)
	}
	return nil
}

// ClearQualityDegradationStateAndSetSchedulable is the atomic operator
// recovery path. Keeping the state clear and the scheduling bit in one
// transaction prevents a concurrent probe from observing a half-recovered
// account.
func (r *accountRepository) ClearQualityDegradationStateAndSetSchedulable(ctx context.Context, accountID int64) error {
	if r == nil {
		return errors.New("account repository unavailable")
	}
	if scoped := r.accountTransactionScope(ctx); scoped != r {
		return scoped.ClearQualityDegradationStateAndSetSchedulable(ctx, accountID)
	}
	if r.sql == nil {
		return errors.New("account repository unavailable")
	}

	baseCtx := ctx
	contextTx := dbent.TxFromContext(ctx)
	exec := r.sql
	if contextTx != nil {
		exec = contextTx.Client()
	}
	var tx *dbent.Tx
	if contextTx == nil && r.client != nil {
		var err error
		tx, err = r.client.Tx(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()
		ctx = dbent.NewTxContext(ctx, tx)
		exec = tx
	}

	result, err := exec.ExecContext(ctx, `
		UPDATE accounts
		SET extra = COALESCE(extra, '{}'::jsonb) - 'quality_degradation_state',
			schedulable = TRUE,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, accountID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAccountNotFound
	}
	if err := enqueueSchedulerOutbox(ctx, exec, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
		return err
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	if contextTx == nil {
		r.syncSchedulerAccountSnapshot(baseCtx, accountID)
	}
	return nil
}

// setSchedulableWithQualityGuard atomically refuses automatic/non-explicit
// attempts to re-enable an account that remains paused by the quality guard.
// The row predicate is re-evaluated after any concurrent quality-probe row
// lock is released, so a stale pre-read cannot bypass the guard.
func (r *accountRepository) setSchedulableWithQualityGuard(ctx context.Context, accountID int64) error {
	if r == nil {
		return errors.New("account repository unavailable")
	}
	if scoped := r.accountTransactionScope(ctx); scoped != r {
		return scoped.setSchedulableWithQualityGuard(ctx, accountID)
	}
	if r.sql == nil {
		return errors.New("account repository unavailable")
	}

	baseCtx := ctx
	contextTx := dbent.TxFromContext(ctx)
	exec := r.sql
	if contextTx != nil {
		exec = contextTx.Client()
	}
	var tx *dbent.Tx
	if contextTx == nil && r.client != nil {
		var err error
		tx, err = r.client.Tx(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()
		ctx = dbent.NewTxContext(ctx, tx)
		exec = tx
	}

	result, err := exec.ExecContext(ctx, `
		UPDATE accounts
		SET schedulable = TRUE, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		  AND COALESCE(extra->'quality_degradation_state'->>'paused', 'false')
		      NOT IN ('true', 't', '1')
	`, accountID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		rows, queryErr := exec.QueryContext(ctx, `
			SELECT COALESCE(extra->'quality_degradation_state'->>'paused', 'false')
			FROM accounts WHERE id = $1 AND deleted_at IS NULL
		`, accountID)
		if queryErr != nil {
			return queryErr
		}
		var paused string
		found := rows.Next()
		if found {
			queryErr = rows.Scan(&paused)
		}
		closeErr := rows.Close()
		if queryErr != nil {
			return queryErr
		}
		if closeErr != nil {
			return closeErr
		}
		if !found {
			return service.ErrAccountNotFound
		}
		if paused == "true" || paused == "t" || paused == "1" {
			return service.ErrQualityDegradationPaused
		}
		return service.ErrAccountNotFound
	}
	if err := enqueueSchedulerOutbox(ctx, exec, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
		return err
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	if contextTx == nil {
		r.syncSchedulerAccountSnapshot(baseCtx, accountID)
	}
	return nil
}
