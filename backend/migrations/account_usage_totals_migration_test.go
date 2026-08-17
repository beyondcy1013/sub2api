package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountUsageTotalsMigrationCreatesDurableInsertOnlyLedger(t *testing.T) {
	content, err := FS.ReadFile("192_add_account_usage_totals.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS account_usage_totals")
	require.Contains(t, sql, "FROM usage_logs")
	require.Contains(t, sql, "ON CONFLICT (account_id) DO NOTHING")
	require.Contains(t, sql, "AFTER INSERT ON usage_logs")
	require.Contains(t, sql, "account_usage_totals.account_cost + EXCLUDED.account_cost")
	require.NotContains(t, sql, "AFTER DELETE")
}
