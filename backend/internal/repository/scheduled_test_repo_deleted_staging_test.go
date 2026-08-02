package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestScheduledTestPlanListDueExcludesDeletedStagingAccounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC()
	mock.ExpectQuery(regexp.MustCompile(`(?s)FROM scheduled_test_plans AS plan.*JOIN accounts AS account.*account\.deleted_at IS NULL.*account\.extra -> 'deleted'`).String()).
		WithArgs(now).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "account_id", "model_id", "cron_expression", "enabled", "max_results",
			"auto_recover", "auto_recover_schedulable", "last_run_at", "next_run_at", "created_at", "updated_at",
		}))

	repo := &scheduledTestPlanRepository{db: db}
	plans, err := repo.ListDue(context.Background(), now)

	require.NoError(t, err)
	require.Empty(t, plans)
	require.NoError(t, mock.ExpectationsWereMet())
}
