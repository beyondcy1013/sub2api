-- Restore only account errors written by the former scheduling-liveness
-- implementation. Real authentication, quota, and upstream errors are left
-- untouched. Future liveness probes are observation-only.
UPDATE accounts
SET status = 'active',
    error_message = '',
    extra = jsonb_set(
        COALESCE(extra, '{}'::jsonb),
        '{scheduling_liveness,status_managed}',
        'false'::jsonb,
        true
    ),
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND status = 'error'
  AND COALESCE(extra #>> '{scheduling_liveness,status_managed}', '') = 'true'
  AND BTRIM(COALESCE(error_message, '')) LIKE 'Scheduling liveness probe failed:%';
