package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeletedStagingMigrationPreservesLegacyAccountsAndGroups(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "191_convert_account_soft_delete_to_deleted_staging.sql")
	payload, err := os.ReadFile(path)
	require.NoError(t, err)

	sql := string(payload)
	require.Contains(t, sql, "INSERT INTO account_groups")
	require.Contains(t, sql, "account.extra -> 'recycle_bin_groups'")
	require.Contains(t, sql, "SET deleted_at = NULL")
	require.Contains(t, sql, `'{"deleted": true}'::jsonb`)
	require.NotContains(t, strings.ToUpper(sql), "DELETE FROM ACCOUNTS")
}
