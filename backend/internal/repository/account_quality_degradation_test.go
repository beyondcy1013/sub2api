package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newQualityDegradationSQLMock(t *testing.T) (*accountRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return newAccountRepositoryWithSQL(nil, db, nil), mock
}

func expectQualityDegradationRecord(mock sqlmock.Sqlmock, accountID int64, count int, paused bool) {
	mock.ExpectQuery(`WITH current_account`).
		WithArgs(accountID, "rule-fingerprint", "gpt-6-astra", "candy mismatch", sqlmock.AnyArg(), 3, true).
		WillReturnRows(sqlmock.NewRows([]string{"consecutive_count", "paused"}).AddRow(count, paused))
	mock.ExpectExec(`INSERT INTO scheduler_outbox`).
		WithArgs(service.SchedulerOutboxEventAccountChanged, accountID, nil, nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func TestRecordQualityDegradation_AtomicallyCountsAndPausesAtThreshold(t *testing.T) {
	repo, mock := newQualityDegradationSQLMock(t)
	expectQualityDegradationRecord(mock, 17, 1, false)
	expectQualityDegradationRecord(mock, 17, 3, true)

	first, err := repo.RecordQualityDegradation(context.Background(), 17, "rule-fingerprint", "gpt-6-astra", "candy mismatch", 3, true)
	require.NoError(t, err)
	require.Equal(t, 1, first.ConsecutiveCount)
	require.False(t, first.Paused)

	second, err := repo.RecordQualityDegradation(context.Background(), 17, "rule-fingerprint", "gpt-6-astra", "candy mismatch", 3, true)
	require.NoError(t, err)
	require.Equal(t, 3, second.ConsecutiveCount)
	require.True(t, second.Paused)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResetQualityDegradationAtGuardsAgainstLateCleanProbe(t *testing.T) {
	repo, mock := newQualityDegradationSQLMock(t)
	observedAt := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	mock.ExpectExec(`UPDATE accounts[\s\S]*last_detected_at`).
		WithArgs(int64(17), "rule-fingerprint", observedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))
	// A stale clean result is a no-op; the latest confirmed degradation remains.
	require.NoError(t, repo.ResetQualityDegradationAt(context.Background(), 17, "rule-fingerprint", observedAt))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetSchedulableRejectsAutomaticResumeWhileQualityPaused(t *testing.T) {
	repo, mock := newQualityDegradationSQLMock(t)
	mock.ExpectExec(`UPDATE accounts SET schedulable = TRUE`).
		WithArgs(int64(17)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT COALESCE\(extra->'quality_degradation_state'->>'paused'`).
		WithArgs(int64(17)).
		WillReturnRows(sqlmock.NewRows([]string{"paused"}).AddRow("true"))

	err := repo.SetSchedulable(context.Background(), 17, true)
	require.ErrorIs(t, err, service.ErrQualityDegradationPaused)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetSchedulableReturnsNotFoundWhenGuardedRowIsMissing(t *testing.T) {
	repo, mock := newQualityDegradationSQLMock(t)
	mock.ExpectExec(`UPDATE accounts SET schedulable = TRUE`).
		WithArgs(int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT COALESCE\(extra->'quality_degradation_state'->>'paused'`).
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"paused"}))

	err := repo.SetSchedulable(context.Background(), 99, true)
	require.ErrorIs(t, err, service.ErrAccountNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSplitQualityStateMetadataEnvelopePreservesLegacyMetadata(t *testing.T) {
	metadata, state, err := splitQualityStateMetadataEnvelope([]byte(`{"__sub2api_upstream_model_metadata":{"models":["gpt-6"]},"__sub2api_quality_degradation_state":{"paused":true}}`))
	require.NoError(t, err)
	require.JSONEq(t, `{"models":["gpt-6"]}`, string(metadata))
	require.JSONEq(t, `{"paused":true}`, string(state))

	legacy, noState, err := splitQualityStateMetadataEnvelope([]byte(`{"models":["gpt-6"]}`))
	require.NoError(t, err)
	require.JSONEq(t, `{"models":["gpt-6"]}`, string(legacy))
	require.Nil(t, noState)
}
