package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeScheduledAccountActionRepository struct {
	upserted       *ScheduledAccountAction
	current        *ScheduledAccountAction
	claimed        []*ScheduledAccountAction
	deletedAccount int64
	completedID    int64
	failedID       int64
	failedError    string
	failedRetryAt  time.Time
}

func (r *fakeScheduledAccountActionRepository) Upsert(_ context.Context, action *ScheduledAccountAction) (*ScheduledAccountAction, error) {
	copy := *action
	r.upserted = &copy
	copy.ID = 11
	return &copy, nil
}

func (r *fakeScheduledAccountActionRepository) GetPendingByAccountID(_ context.Context, accountID int64) (*ScheduledAccountAction, error) {
	if r.current == nil || r.current.AccountID != accountID {
		return nil, nil
	}
	copy := *r.current
	return &copy, nil
}

func (r *fakeScheduledAccountActionRepository) DeletePendingByAccountID(_ context.Context, accountID int64) error {
	r.deletedAccount = accountID
	return nil
}

func (r *fakeScheduledAccountActionRepository) ClaimDue(_ context.Context, _ time.Time, _ time.Time, _ int) ([]*ScheduledAccountAction, error) {
	return r.claimed, nil
}

func (r *fakeScheduledAccountActionRepository) MarkCompleted(_ context.Context, id int64, _ time.Time) error {
	r.completedID = id
	return nil
}

func (r *fakeScheduledAccountActionRepository) MarkFailed(_ context.Context, id int64, message string, retryAt time.Time) error {
	r.failedID = id
	r.failedError = message
	r.failedRetryAt = retryAt
	return nil
}

type fakeScheduledAccountActionPerformer struct {
	calls []string
	err   error
}

func (p *fakeScheduledAccountActionPerformer) Perform(_ context.Context, accountID int64, action ScheduledAccountActionType) error {
	p.calls = append(p.calls, action.String()+":"+time.Unix(accountID, 0).UTC().Format(time.RFC3339))
	return p.err
}

func TestScheduledAccountActionServiceScheduleValidatesAndPersistsDelay(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 30, 0, 0, time.UTC)
	repo := &fakeScheduledAccountActionRepository{}
	svc := NewScheduledAccountActionService(repo, &fakeScheduledAccountActionPerformer{})
	svc.now = func() time.Time { return now }

	if _, err := svc.Schedule(context.Background(), 42, ScheduledAccountActionEnableAndRecover, 0); err == nil {
		t.Fatal("expected zero delay to be rejected")
	}
	if _, err := svc.Schedule(context.Background(), 42, ScheduledAccountActionType("unknown"), time.Minute); err == nil {
		t.Fatal("expected unknown action to be rejected")
	}

	created, err := svc.Schedule(context.Background(), 42, ScheduledAccountActionPause, 2*time.Hour+15*time.Minute)
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	if created.AccountID != 42 || created.Action != ScheduledAccountActionPause {
		t.Fatalf("created = %#v", created)
	}
	wantExecuteAt := now.Add(2*time.Hour + 15*time.Minute)
	if !created.ExecuteAt.Equal(wantExecuteAt) || repo.upserted == nil || !repo.upserted.ExecuteAt.Equal(wantExecuteAt) {
		t.Fatalf("execute_at = %v, want %v", created.ExecuteAt, wantExecuteAt)
	}
}

func TestScheduledAccountActionServiceGetAndCancel(t *testing.T) {
	repo := &fakeScheduledAccountActionRepository{current: &ScheduledAccountAction{ID: 7, AccountID: 9, Action: ScheduledAccountActionPause}}
	svc := NewScheduledAccountActionService(repo, &fakeScheduledAccountActionPerformer{})

	got, err := svc.GetScheduledAction(context.Background(), 9)
	if err != nil || got == nil || got.ID != 7 {
		t.Fatalf("GetScheduledAction() = %#v, %v", got, err)
	}
	if err := svc.CancelScheduledAction(context.Background(), 9); err != nil {
		t.Fatalf("CancelScheduledAction() error = %v", err)
	}
	if repo.deletedAccount != 9 {
		t.Fatalf("deleted account = %d, want 9", repo.deletedAccount)
	}
}

func TestScheduledAccountActionServiceProcessDueCompletesSuccessfulActions(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	repo := &fakeScheduledAccountActionRepository{claimed: []*ScheduledAccountAction{{ID: 31, AccountID: 8, Action: ScheduledAccountActionPause}}}
	performer := &fakeScheduledAccountActionPerformer{}
	svc := NewScheduledAccountActionService(repo, performer)
	svc.now = func() time.Time { return now }

	processed, err := svc.ProcessDue(context.Background(), 20)
	if err != nil {
		t.Fatalf("ProcessDue() error = %v", err)
	}
	if processed != 1 || repo.completedID != 31 || repo.failedID != 0 {
		t.Fatalf("processed=%d completed=%d failed=%d", processed, repo.completedID, repo.failedID)
	}
	if len(performer.calls) != 1 {
		t.Fatalf("performer calls = %v", performer.calls)
	}
}

func TestScheduledAccountActionServiceProcessDueRetriesFailures(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	repo := &fakeScheduledAccountActionRepository{claimed: []*ScheduledAccountAction{{ID: 32, AccountID: 10, Action: ScheduledAccountActionEnableAndRecover}}}
	performer := &fakeScheduledAccountActionPerformer{err: errors.New("database unavailable")}
	svc := NewScheduledAccountActionService(repo, performer)
	svc.now = func() time.Time { return now }

	processed, err := svc.ProcessDue(context.Background(), 20)
	if processed != 1 || err == nil {
		t.Fatalf("ProcessDue() = %d, %v", processed, err)
	}
	if repo.completedID != 0 || repo.failedID != 32 || repo.failedError != "database unavailable" {
		t.Fatalf("completed=%d failed=%d message=%q", repo.completedID, repo.failedID, repo.failedError)
	}
	if !repo.failedRetryAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("retry_at=%v, want %v", repo.failedRetryAt, now.Add(time.Minute))
	}
}

type fakeScheduledActionSchedulableSetter struct{ calls *[]string }

func (s fakeScheduledActionSchedulableSetter) SetAccountSchedulable(_ context.Context, id int64, enabled bool) (*Account, error) {
	*s.calls = append(*s.calls, "schedulable:"+time.Unix(id, 0).UTC().Format(time.RFC3339)+":"+map[bool]string{true: "true", false: "false"}[enabled])
	return &Account{ID: id, Schedulable: enabled}, nil
}

type fakeScheduledActionRecoverer struct{ calls *[]string }

func (r fakeScheduledActionRecoverer) RecoverAccountState(_ context.Context, id int64, options AccountRecoveryOptions) (*SuccessfulTestRecoveryResult, error) {
	*r.calls = append(*r.calls, "recover:"+time.Unix(id, 0).UTC().Format(time.RFC3339)+":"+map[bool]string{true: "invalidate", false: "keep"}[options.InvalidateToken])
	return &SuccessfulTestRecoveryResult{}, nil
}

func TestDefaultScheduledAccountActionPerformerSemantics(t *testing.T) {
	var calls []string
	performer := newScheduledAccountActionPerformer(
		fakeScheduledActionSchedulableSetter{calls: &calls},
		fakeScheduledActionRecoverer{calls: &calls},
	)

	if err := performer.Perform(context.Background(), 5, ScheduledAccountActionPause); err != nil {
		t.Fatalf("pause error = %v", err)
	}
	wantPause := []string{"schedulable:1970-01-01T00:00:05Z:false"}
	if !reflect.DeepEqual(calls, wantPause) {
		t.Fatalf("pause calls = %v, want %v", calls, wantPause)
	}

	calls = nil
	if err := performer.Perform(context.Background(), 5, ScheduledAccountActionEnableAndRecover); err != nil {
		t.Fatalf("enable_and_recover error = %v", err)
	}
	wantRecover := []string{
		"recover:1970-01-01T00:00:05Z:invalidate",
		"schedulable:1970-01-01T00:00:05Z:true",
	}
	if !reflect.DeepEqual(calls, wantRecover) {
		t.Fatalf("enable calls = %v, want %v", calls, wantRecover)
	}
}
