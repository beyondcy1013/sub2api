-- Persist lifetime usage totals per account independently of usage_logs retention.
CREATE TABLE IF NOT EXISTS account_usage_totals (
    account_id     BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    requests       BIGINT NOT NULL DEFAULT 0,
    tokens         BIGINT NOT NULL DEFAULT 0,
    account_cost   DECIMAL(30, 12) NOT NULL DEFAULT 0,
    standard_cost  DECIMAL(30, 12) NOT NULL DEFAULT 0,
    user_cost      DECIMAL(30, 12) NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE account_usage_totals IS
    'Lifetime per-account usage totals retained independently of removable usage logs.';

-- Seed totals from all detail rows that still exist when this migration first runs.
-- DO NOTHING keeps the migration idempotent without resetting already accumulated totals.
INSERT INTO account_usage_totals (
    account_id,
    requests,
    tokens,
    account_cost,
    standard_cost,
    user_cost
)
SELECT
    account_id,
    COUNT(*),
    COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0),
    COALESCE(SUM(COALESCE(account_stats_cost, total_cost) * COALESCE(account_rate_multiplier, 1)), 0),
    COALESCE(SUM(total_cost), 0),
    COALESCE(SUM(actual_cost), 0)
FROM usage_logs
GROUP BY account_id
ON CONFLICT (account_id) DO NOTHING;

CREATE OR REPLACE FUNCTION accumulate_account_usage_totals()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO account_usage_totals (
        account_id,
        requests,
        tokens,
        account_cost,
        standard_cost,
        user_cost
    ) VALUES (
        NEW.account_id,
        1,
        NEW.input_tokens + NEW.output_tokens + NEW.cache_creation_tokens + NEW.cache_read_tokens,
        COALESCE(NEW.account_stats_cost, NEW.total_cost) * COALESCE(NEW.account_rate_multiplier, 1),
        NEW.total_cost,
        NEW.actual_cost
    )
    ON CONFLICT (account_id) DO UPDATE SET
        requests = account_usage_totals.requests + EXCLUDED.requests,
        tokens = account_usage_totals.tokens + EXCLUDED.tokens,
        account_cost = account_usage_totals.account_cost + EXCLUDED.account_cost,
        standard_cost = account_usage_totals.standard_cost + EXCLUDED.standard_cost,
        user_cost = account_usage_totals.user_cost + EXCLUDED.user_cost,
        updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_accumulate_account_usage_totals ON usage_logs;
CREATE TRIGGER trg_accumulate_account_usage_totals
AFTER INSERT ON usage_logs
FOR EACH ROW
EXECUTE FUNCTION accumulate_account_usage_totals();
