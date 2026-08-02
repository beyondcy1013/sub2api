-- Account deletion is now a recoverable second-level staging state. Restore
-- group bindings saved by the former soft-delete flow before bringing legacy
-- rows back under the normal account-management APIs.
WITH saved_groups AS (
    SELECT
        account.id AS account_id,
        CASE
            WHEN saved.value ~ '^[0-9]+$' THEN saved.value::bigint
            ELSE NULL
        END AS group_id,
        saved.ordinality::integer AS priority
    FROM accounts AS account
    CROSS JOIN LATERAL jsonb_array_elements_text(
        CASE
            WHEN jsonb_typeof(account.extra -> 'recycle_bin_groups') = 'array'
                THEN account.extra -> 'recycle_bin_groups'
            ELSE '[]'::jsonb
        END
    ) WITH ORDINALITY AS saved(value, ordinality)
    WHERE account.deleted_at IS NOT NULL
)
INSERT INTO account_groups (account_id, group_id, priority, created_at)
SELECT saved.account_id, saved.group_id, saved.priority, NOW()
FROM saved_groups AS saved
JOIN groups AS account_group
  ON account_group.id = saved.group_id
 AND account_group.deleted_at IS NULL
WHERE saved.group_id IS NOT NULL
ON CONFLICT (account_id, group_id) DO NOTHING;

UPDATE accounts
SET deleted_at = NULL,
    extra = (COALESCE(extra, '{}'::jsonb) - 'recycle_bin_groups' - 'recycled')
        || '{"deleted": true}'::jsonb,
    updated_at = NOW()
WHERE deleted_at IS NOT NULL;
