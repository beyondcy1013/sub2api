CREATE TABLE IF NOT EXISTS scheduled_account_actions (
    id           BIGSERIAL PRIMARY KEY,
    account_id   BIGINT NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    action       VARCHAR(32) NOT NULL CHECK (action IN ('enable_and_recover', 'pause')),
    execute_at   TIMESTAMPTZ NOT NULL,
    status       VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed')),
    attempts     INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    lease_until  TIMESTAMPTZ,
    last_error   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_scheduled_account_actions_due
    ON scheduled_account_actions (execute_at, id)
    WHERE status IN ('pending', 'processing');
