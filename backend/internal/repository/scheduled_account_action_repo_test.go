package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func scheduledAccountActionRows(now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "account_id", "action", "execute_at", "status", "attempts",
		"lease_until", "last_error", "created_at", "updated_at", "completed_at",
	}).AddRow(12, 7, "pause", now.Add(time.Hour), "pending", 0, nil, nil, now, now, nil)
}

func TestScheduledAccountActionRepositoryUpsertReplacesAccountTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO scheduled_account_actions")).
		WithArgs(int64(7), service.ScheduledAccountActionPause, now.Add(time.Hour)).
		WillReturnRows(scheduledAccountActionRows(now))

	repo := NewScheduledAccountActionRepository(db)
	got, err := repo.Upsert(context.Background(), &service.ScheduledAccountAction{
		AccountID: 7,
		Action:    service.ScheduledAccountActionPause,
		ExecuteAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if got.ID != 12 || got.AccountID != 7 || got.Action != service.ScheduledAccountActionPause {
		t.Fatalf("Upsert() = %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestScheduledAccountActionRepositoryClaimDueUsesLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	leaseUntil := now.Add(2 * time.Minute)
	mock.ExpectQuery("(?s)WITH due AS .*FOR UPDATE SKIP LOCKED.*status = 'processing'.*RETURNING action\\.id").
		WithArgs(now, 20, leaseUntil).
		WillReturnRows(scheduledAccountActionRows(now))

	repo := NewScheduledAccountActionRepository(db)
	got, err := repo.ClaimDue(context.Background(), now, leaseUntil, 20)
	if err != nil {
		t.Fatalf("ClaimDue() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != 12 {
		t.Fatalf("ClaimDue() = %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestScheduledAccountActionRepositoryGetMissingReturnsNil(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("(?s)SELECT .*FROM scheduled_account_actions").
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "account_id", "action", "execute_at", "status", "attempts",
			"lease_until", "last_error", "created_at", "updated_at", "completed_at",
		}))

	repo := NewScheduledAccountActionRepository(db)
	got, err := repo.GetPendingByAccountID(context.Background(), 99)
	if err != nil || got != nil {
		t.Fatalf("GetPendingByAccountID() = %#v, %v", got, err)
	}
}
