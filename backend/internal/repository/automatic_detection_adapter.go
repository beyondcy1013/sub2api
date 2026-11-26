package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// CompareAndSetAutomaticDetectionState keeps the guarded state write and the
// scheduler outbox event atomic. Snapshot synchronization uses the shared cache.
func (r *accountRepository) CompareAndSetAutomaticDetectionState(ctx context.Context, current *service.Account, status string, schedulable bool) error {
	if current == nil {
		return service.ErrAccountNilInput
	}
	result, err := r.sql.ExecContext(ctx, `WITH changed AS (
 UPDATE accounts SET status=$1,schedulable=$2,updated_at=NOW()
 WHERE id=$3 AND updated_at=$4 AND status=$5 AND schedulable=$6 AND deleted_at IS NULL
 AND COALESCE(extra->'deleted','false'::jsonb)<>'true'::jsonb
 AND COALESCE(extra->'recycled','false'::jsonb)<>'true'::jsonb
 AND ($2 = false OR COALESCE(extra->'quality_degradation_state'->>'paused','false') NOT IN ('true','t','1'))
 AND (expires_at IS NULL OR expires_at>NOW()) RETURNING id
 ) INSERT INTO scheduler_outbox(event_type,account_id,payload)
 SELECT $7,id,'{}'::jsonb FROM changed`, status, schedulable, current.ID, current.UpdatedAt, current.Status, current.Schedulable, service.SchedulerOutboxEventAccountChanged)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("account changed; automatic action cancelled")
	}
	r.syncSchedulerAccountSnapshot(ctx, current.ID)
	return nil
}
