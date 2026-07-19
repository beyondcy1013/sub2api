package service

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type ScheduledAccountActionType string

const (
	ScheduledAccountActionEnableAndRecover ScheduledAccountActionType = "enable_and_recover"
	ScheduledAccountActionPause            ScheduledAccountActionType = "pause"
)

const (
	scheduledAccountActionStatusPending    = "pending"
	scheduledAccountActionStatusProcessing = "processing"
	scheduledAccountActionStatusCompleted  = "completed"
	scheduledAccountActionLeaseDuration    = 2 * time.Minute
	scheduledAccountActionRetryDelay       = time.Minute
	scheduledAccountActionMaxDelay         = 365 * 24 * time.Hour
)

var (
	ErrInvalidScheduledAccountAction = errors.New("invalid scheduled account action")
	ErrInvalidScheduledActionDelay   = errors.New("scheduled action delay must be between 1 minute and 365 days")
)

func (a ScheduledAccountActionType) String() string { return string(a) }

func (a ScheduledAccountActionType) Valid() bool {
	return a == ScheduledAccountActionEnableAndRecover || a == ScheduledAccountActionPause
}

type ScheduledAccountAction struct {
	ID          int64                      `json:"id"`
	AccountID   int64                      `json:"account_id"`
	Action      ScheduledAccountActionType `json:"action"`
	ExecuteAt   time.Time                  `json:"execute_at"`
	Status      string                     `json:"status"`
	Attempts    int                        `json:"attempts"`
	LeaseUntil  *time.Time                 `json:"lease_until,omitempty"`
	LastError   *string                    `json:"last_error,omitempty"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	CompletedAt *time.Time                 `json:"completed_at,omitempty"`
}

type ScheduledAccountActionRepository interface {
	Upsert(ctx context.Context, action *ScheduledAccountAction) (*ScheduledAccountAction, error)
	GetPendingByAccountID(ctx context.Context, accountID int64) (*ScheduledAccountAction, error)
	DeletePendingByAccountID(ctx context.Context, accountID int64) error
	ClaimDue(ctx context.Context, now, leaseUntil time.Time, limit int) ([]*ScheduledAccountAction, error)
	MarkCompleted(ctx context.Context, id int64, completedAt time.Time) error
	MarkFailed(ctx context.Context, id int64, message string, retryAt time.Time) error
}

type ScheduledAccountActionPerformer interface {
	Perform(ctx context.Context, accountID int64, action ScheduledAccountActionType) error
}

type ScheduledAccountActionService struct {
	repo      ScheduledAccountActionRepository
	performer ScheduledAccountActionPerformer
	now       func() time.Time
}

func NewScheduledAccountActionService(
	repo ScheduledAccountActionRepository,
	performer ScheduledAccountActionPerformer,
) *ScheduledAccountActionService {
	return &ScheduledAccountActionService{repo: repo, performer: performer, now: time.Now}
}

func (s *ScheduledAccountActionService) Schedule(
	ctx context.Context,
	accountID int64,
	action ScheduledAccountActionType,
	delay time.Duration,
) (*ScheduledAccountAction, error) {
	if !action.Valid() {
		return nil, ErrInvalidScheduledAccountAction
	}
	if delay < time.Minute || delay > scheduledAccountActionMaxDelay {
		return nil, ErrInvalidScheduledActionDelay
	}

	task := &ScheduledAccountAction{
		AccountID: accountID,
		Action:    action,
		ExecuteAt: s.now().Add(delay),
		Status:    scheduledAccountActionStatusPending,
	}
	return s.repo.Upsert(ctx, task)
}

func (s *ScheduledAccountActionService) GetScheduledAction(ctx context.Context, accountID int64) (*ScheduledAccountAction, error) {
	return s.repo.GetPendingByAccountID(ctx, accountID)
}

func (s *ScheduledAccountActionService) CancelScheduledAction(ctx context.Context, accountID int64) error {
	return s.repo.DeletePendingByAccountID(ctx, accountID)
}

func (s *ScheduledAccountActionService) ProcessDue(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 20
	}
	now := s.now()
	tasks, err := s.repo.ClaimDue(ctx, now, now.Add(scheduledAccountActionLeaseDuration), limit)
	if err != nil {
		return 0, err
	}

	var executionErrors []error
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if err := s.performer.Perform(ctx, task.AccountID, task.Action); err != nil {
			if markErr := s.repo.MarkFailed(ctx, task.ID, err.Error(), now.Add(scheduledAccountActionRetryDelay)); markErr != nil {
				executionErrors = append(executionErrors, fmt.Errorf("task %d failed: %v; mark failed: %w", task.ID, err, markErr))
			} else {
				executionErrors = append(executionErrors, fmt.Errorf("task %d failed: %w", task.ID, err))
			}
			continue
		}
		if err := s.repo.MarkCompleted(ctx, task.ID, s.now()); err != nil {
			executionErrors = append(executionErrors, fmt.Errorf("mark task %d completed: %w", task.ID, err))
		}
	}
	return len(tasks), errors.Join(executionErrors...)
}

type scheduledActionSchedulableSetter interface {
	SetAccountSchedulable(ctx context.Context, id int64, schedulable bool) (*Account, error)
}

type scheduledActionRecoverer interface {
	RecoverAccountState(ctx context.Context, accountID int64, options AccountRecoveryOptions) (*SuccessfulTestRecoveryResult, error)
}

type scheduledAccountActionPerformer struct {
	setter    scheduledActionSchedulableSetter
	recoverer scheduledActionRecoverer
}

func newScheduledAccountActionPerformer(
	setter scheduledActionSchedulableSetter,
	recoverer scheduledActionRecoverer,
) ScheduledAccountActionPerformer {
	return &scheduledAccountActionPerformer{setter: setter, recoverer: recoverer}
}

func (p *scheduledAccountActionPerformer) Perform(ctx context.Context, accountID int64, action ScheduledAccountActionType) error {
	switch action {
	case ScheduledAccountActionPause:
		_, err := p.setter.SetAccountSchedulable(ctx, accountID, false)
		return err
	case ScheduledAccountActionEnableAndRecover:
		if _, err := p.recoverer.RecoverAccountState(ctx, accountID, AccountRecoveryOptions{InvalidateToken: true}); err != nil {
			return err
		}
		_, err := p.setter.SetAccountSchedulable(ctx, accountID, true)
		return err
	default:
		return ErrInvalidScheduledAccountAction
	}
}
