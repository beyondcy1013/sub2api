package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type schedulingLivenessQueryMatcher struct {
	actual *string
}

func (m schedulingLivenessQueryMatcher) Match(_ string, actual string) error {
	*m.actual = actual
	return nil
}

func TestUpdateSchedulingLivenessPreservesAccountState(t *testing.T) {
	var statement string
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(schedulingLivenessQueryMatcher{actual: &statement}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	snapshot := &service.AccountSchedulingLiveness{
		Status:    service.SchedulingLivenessStatusDead,
		LastError: "upstream timeout",
	}
	mock.ExpectExec("scheduling liveness update").
		WithArgs(
			sqlmock.AnyArg(),
			int64(17),
			service.SchedulerOutboxEventAccountChanged,
			service.SchedulingLivenessExtraKey,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := newAccountRepositoryWithSQL(nil, db, nil)
	err = repo.UpdateSchedulingLiveness(context.Background(), 17, snapshot)

	require.NoError(t, err)
	require.Contains(t, statement, "jsonb_build_object")
	require.Contains(t, statement, "$4::text")
	require.NotContains(t, statement, "status =")
	require.NotContains(t, statement, "error_message =")
	require.NotContains(t, statement, "schedulable =")
	require.Contains(t, statement, "INSERT INTO scheduler_outbox")
	require.NoError(t, mock.ExpectationsWereMet())
}
