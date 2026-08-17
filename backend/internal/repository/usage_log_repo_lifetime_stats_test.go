package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGetAccountLifetimeStatsBatchReadsDurableTotalsAndFillsMissingAccounts(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	mock.ExpectQuery("FROM account_usage_totals").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"account_id", "requests", "tokens", "account_cost", "standard_cost", "user_cost",
		}).AddRow(int64(7), int64(3), int64(1200), 1.23, 1.0, 1.5))

	stats, err := repo.GetAccountLifetimeStatsBatch(context.Background(), []int64{7, 8})
	require.NoError(t, err)
	require.Equal(t, int64(3), stats[7].Requests)
	require.Equal(t, int64(1200), stats[7].Tokens)
	require.InDelta(t, 1.23, stats[7].Cost, 1e-9)
	require.InDelta(t, 1.0, stats[7].StandardCost, 1e-9)
	require.InDelta(t, 1.5, stats[7].UserCost, 1e-9)
	require.Equal(t, int64(0), stats[8].Requests)
	require.NoError(t, mock.ExpectationsWereMet())
}
