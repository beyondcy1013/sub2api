package repository

import (
	"context"
	"strings"
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

func TestUpdateSchedulingLivenessOwnsStatusWithoutChangingSchedulable(t *testing.T) {
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
			service.StatusActive,
			service.StatusError,
			true,
			false,
			strings.TrimSpace(service.SchedulingLivenessErrorPrefix)+"%",
			service.SchedulingLivenessErrorPrefix+"upstream timeout",
			service.SchedulerOutboxEventAccountChanged,
			service.SchedulingLivenessExtraKey,
			service.SchedulingLivenessStatusManagedExtraKey,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := newAccountRepositoryWithSQL(nil, db, nil)
	err = repo.UpdateSchedulingLiveness(context.Background(), 17, snapshot)

	require.NoError(t, err)
	require.Contains(t, statement, "status = CASE")
	require.Contains(t, statement, "status_managed")
	require.NotContains(t, statement, "schedulable =")
	require.Contains(t, statement, "INSERT INTO scheduler_outbox")
	require.NoError(t, mock.ExpectationsWereMet())
}
