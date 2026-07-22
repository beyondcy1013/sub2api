-- 186: Add auto_recover_schedulable column to scheduled_test_plans
-- When enabled, automatically re-enables scheduling (schedulable=true) on successful test,
-- so accounts that were paused can be auto-restored to the scheduling pool.

ALTER TABLE scheduled_test_plans ADD COLUMN IF NOT EXISTS auto_recover_schedulable BOOLEAN NOT NULL DEFAULT false;
