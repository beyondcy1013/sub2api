package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestScheduledTestPlanListAllJoinsActiveAccountAndExcludesDeletedStaging(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	nextRun := time.Date(2026, 9, 18, 4, 0, 0, 0, time.UTC)
	now := nextRun.Add(-time.Hour)
	rows := sqlmock.NewRows([]string{
		"id", "account_id", "model_id", "cron_expression", "enabled", "max_results",
		"auto_recover", "auto_recover_schedulable", "last_run_at", "next_run_at",
		"created_at", "updated_at", "name", "platform",
	}).AddRow(
		1, 2, "gpt-5.4", "0 4 * * *", true, 50, true, false,
		nil, nextRun, now, now, "daily probe", "openai",
	)
	mock.ExpectQuery(regexp.QuoteMeta(`FROM scheduled_test_plans AS plan`)).
		WillReturnRows(rows)

	repo := &scheduledTestPlanRepository{db: db}
	plans, err := repo.ListAll(context.Background())

	require.NoError(t, err)
	require.Len(t, plans, 1)
	require.Equal(t, int64(2), plans[0].AccountID)
	require.Equal(t, "daily probe", plans[0].AccountName)
	require.Equal(t, "openai", plans[0].AccountPlatform)
	require.NoError(t, mock.ExpectationsWereMet())
}

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
