package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/robfig/cron/v3"
)

// superPriorityCronParser 支持 @every 描述符与传统 5 字段 cron 表达式。
var superPriorityCronParser = cron.NewParser(
	cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

const schedulingLivenessMaxWorkers = 4

type accountBackgroundTester interface {
	RunTestBackground(context.Context, int64, string) (*ScheduledTestResult, error)
}

// SuperPriorityRunner is retained as a compatibility type name. It now probes
// every active account while lowest-cost scheduling is enabled and persists an
// observation-only liveness snapshot; legacy super_priority flags are ignored.
type SuperPriorityRunner struct {
	state          *SuperPriorityService
	accountTestSvc accountBackgroundTester
	accountRepo    AccountRepository

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewSuperPriorityRunner 创建探测运行器。
func NewSuperPriorityRunner(state *SuperPriorityService, accountTestSvc accountBackgroundTester, accountRepo AccountRepository) *SuperPriorityRunner {
	return &SuperPriorityRunner{
		state:          state,
		accountTestSvc: accountTestSvc,
		accountRepo:    accountRepo,
	}
}

// Start scans once per minute. The configured expression is evaluated per
// account, so changing the interval takes effect without restarting the service.
func (r *SuperPriorityRunner) Start() {
	if r == nil || r.state == nil || r.state.cfg == nil {
		return
	}
	r.startOnce.Do(func() {
		loc := time.Local
		if r.state.cfg != nil {
			if parsed, err := time.LoadLocation(r.state.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithParser(superPriorityCronParser), cron.WithLocation(loc))
		if _, err := c.AddFunc("@every 1m", func() { r.runOnce() }); err != nil {
			logger.LegacyPrintf("service.super_priority_runner", "[SchedulingLivenessRunner] not started: %v", err)
			return
		}
		r.cron = c
		r.cron.Start()
		logger.LegacyPrintf("service.super_priority_runner", "[SchedulingLivenessRunner] started (tick=every minute)")
	})
}

// Stop 优雅关闭。
func (r *SuperPriorityRunner) Stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		if r.cron != nil {
			ctx := r.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.super_priority_runner", "[SuperPriorityRunner] cron stop timed out")
			}
		}
	})
}

// RunOnce 暴露单次探测，便于测试与手动触发。
func (r *SuperPriorityRunner) RunOnce(ctx context.Context) {
	r.runOnceOnce(ctx)
}

func (r *SuperPriorityRunner) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	r.runOnceOnce(ctx)
}

func (r *SuperPriorityRunner) runOnceOnce(ctx context.Context) {
	if r == nil || r.state == nil || r.accountRepo == nil || r.accountTestSvc == nil || r.state.BaseStrategy() != AccountSchedulingStrategyLowestCost {
		return
	}

	accounts, err := r.accountRepo.ListAllWithFilters(ctx, "", "", StatusActive, "", 0, "", false)
	if err != nil {
		logger.LegacyPrintf("service.super_priority_runner", "[SchedulingLivenessRunner] list active accounts error: %v", err)
		return
	}
	if len(accounts) == 0 {
		return
	}

	now := time.Now()
	expr := r.state.CheckInterval()
	due := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if schedulingLivenessProbeDue(decodeSchedulingLiveness(account.Extra), now, expr) {
			due = append(due, account)
		}
	}
	if len(due) == 0 {
		return
	}

	modelID := r.state.TestModelID()
	failureThreshold := r.state.FailureThreshold()
	sem := make(chan struct{}, schedulingLivenessMaxWorkers)
	var wg sync.WaitGroup
	for i := range due {
		account := due[i]
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			attemptedAt := time.Now()
			result, testErr := r.accountTestSvc.RunTestBackground(ctx, account.ID, modelID)
			succeeded := testErr == nil && result != nil && result.Status == "success"
			errorMessage := ""
			if testErr != nil {
				errorMessage = testErr.Error()
			} else if result != nil {
				errorMessage = result.ErrorMessage
			}
			previous := decodeSchedulingLiveness(account.Extra)
			snapshot := nextSchedulingLiveness(
				previous,
				attemptedAt,
				schedulingLivenessFreshUntil(attemptedAt, expr),
				succeeded,
				errorMessage,
				failureThreshold,
			)
			if err := r.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{SchedulingLivenessExtraKey: snapshot}); err != nil {
				logger.LegacyPrintf("service.super_priority_runner", "[SchedulingLivenessRunner] persist account=%d failed: %v", account.ID, err)
				return
			}
			if !succeeded {
				logger.LegacyPrintf("service.super_priority_runner", "[SchedulingLivenessRunner] account=%d status=%s failures=%d error=%s", account.ID, snapshot.Status, snapshot.FailureCount, strings.TrimSpace(errorMessage))
			}
		}()
	}
	wg.Wait()
}

func schedulingLivenessProbeDue(snapshot *AccountSchedulingLiveness, now time.Time, expression string) bool {
	if snapshot == nil || snapshot.LastAttemptAt.IsZero() {
		return true
	}
	return !now.Before(schedulingLivenessNextProbeAt(snapshot.LastAttemptAt, expression))
}

func schedulingLivenessNextProbeAt(from time.Time, expression string) time.Time {
	schedule, err := superPriorityCronParser.Parse(strings.TrimSpace(expression))
	if err != nil {
		schedule, _ = superPriorityCronParser.Parse("@every 1m")
	}
	return schedule.Next(from)
}

func schedulingLivenessFreshUntil(now time.Time, expression string) time.Time {
	next := schedulingLivenessNextProbeAt(now, expression)
	return schedulingLivenessNextProbeAt(next, expression)
}
