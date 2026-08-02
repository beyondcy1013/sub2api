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

type schedulingLivenessPersistence interface {
	UpdateSchedulingLiveness(context.Context, int64, *AccountSchedulingLiveness) error
}

// SuperPriorityRunner is retained as a compatibility type name. It probes
// schedulable active API-key accounts while lowest-cost scheduling is enabled.
// OAuth and other credential types are always excluded. Manual pauses are
// included only when explicitly configured.
type SuperPriorityRunner struct {
	state          *SuperPriorityService
	accountTestSvc accountBackgroundTester
	accountRepo    AccountRepository
	livenessRepo   schedulingLivenessPersistence

	cron        *cron.Cron
	scanEntryID cron.EntryID
	runMu       sync.Mutex
	statusMu    sync.RWMutex
	runtime     SchedulingLivenessRuntimeStatus
	startOnce   sync.Once
	stopOnce    sync.Once
}

// NewSuperPriorityRunner 创建探测运行器。
func NewSuperPriorityRunner(state *SuperPriorityService, accountTestSvc accountBackgroundTester, accountRepo AccountRepository) *SuperPriorityRunner {
	livenessRepo, _ := accountRepo.(schedulingLivenessPersistence)
	return &SuperPriorityRunner{
		state:          state,
		accountTestSvc: accountTestSvc,
		accountRepo:    accountRepo,
		livenessRepo:   livenessRepo,
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
		entryID, err := c.AddFunc("@every 1m", func() { r.runOnce() })
		if err != nil {
			logger.LegacyPrintf("service.super_priority_runner", "[SchedulingLivenessRunner] not started: %v", err)
			return
		}
		r.scanEntryID = entryID
		r.cron = c
		r.cron.Start()
		r.setNextRunAt(r.nextScheduledScanAt(time.Now()))
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
	_, _ = r.runOnceOnce(ctx, false, "scheduled")
}

// RefreshNow ignores the configured interval while preserving the configured
// paused-account scope.
func (r *SuperPriorityRunner) RefreshNow(ctx context.Context) (SchedulingRefreshResult, error) {
	return r.runOnceOnce(ctx, true, "manual")
}

// RuntimeStatus returns a snapshot safe for concurrent handler reads.
func (r *SuperPriorityRunner) RuntimeStatus() SchedulingLivenessRuntimeStatus {
	if r == nil {
		return SchedulingLivenessRuntimeStatus{}
	}
	r.statusMu.RLock()
	status := r.runtime
	if r.runtime.NextRunAt != nil {
		next := *r.runtime.NextRunAt
		status.NextRunAt = &next
	}
	if r.runtime.LastRun != nil {
		last := *r.runtime.LastRun
		status.LastRun = &last
	}
	r.statusMu.RUnlock()
	status.Enabled = r.state != nil && r.state.BaseStrategy() == AccountSchedulingStrategyLowestCost
	if !status.Enabled {
		status.Running = false
		status.NextRunAt = nil
	} else if !status.Running && status.NextRunAt != nil {
		next := r.nextScheduledScanAt(*status.NextRunAt)
		status.NextRunAt = schedulingLivenessTimePtr(next)
	}
	return status
}

func (r *SuperPriorityRunner) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	_, _ = r.runOnceOnce(ctx, false, "scheduled")
}

func (r *SuperPriorityRunner) runOnceOnce(ctx context.Context, force bool, trigger string) (SchedulingRefreshResult, error) {
	var refreshResult SchedulingRefreshResult
	if r == nil || r.state == nil || r.accountRepo == nil || r.livenessRepo == nil || r.accountTestSvc == nil || r.state.BaseStrategy() != AccountSchedulingStrategyLowestCost {
		return refreshResult, nil
	}
	r.runMu.Lock()
	defer r.runMu.Unlock()

	accounts, err := r.accountRepo.ListAllWithFilters(ctx, "", "", "", "", 0, "", false)
	if err != nil {
		logger.LegacyPrintf("service.super_priority_runner", "[SchedulingLivenessRunner] list accounts error: %v", err)
		r.finishRun(trigger, time.Now(), refreshResult, err, schedulingLivenessTimePtr(time.Now().Add(time.Minute)))
		return refreshResult, err
	}
	if len(accounts) == 0 {
		r.setNextRunAt(time.Time{})
		if force {
			r.finishRun(trigger, time.Now(), refreshResult, nil, nil)
		}
		return refreshResult, nil
	}

	now := time.Now()
	expr := r.state.CheckInterval()
	includeUnschedulable := r.state.LivenessIncludeUnschedulable()
	due := make([]Account, 0, len(accounts))
	var nextRunAt time.Time
	updateNextRunAt := func(candidate time.Time) {
		if candidate.IsZero() {
			return
		}
		if nextRunAt.IsZero() || candidate.Before(nextRunAt) {
			nextRunAt = candidate
		}
	}
	for _, account := range accounts {
		if legacySchedulingLivenessOwnsAccountError(&account) {
			if err := r.accountRepo.ClearError(ctx, account.ID); err != nil {
				logger.LegacyPrintf("service.super_priority_runner", "[SchedulingLivenessRunner] clear legacy managed error account=%d failed: %v", account.ID, err)
				continue
			}
			if snapshot := decodeSchedulingLiveness(account.Extra); snapshot != nil {
				snapshot.FailureCount = 0
				snapshot.LastError = ""
				if snapshot.FreshUntil.IsZero() || now.Before(snapshot.FreshUntil) {
					snapshot.FreshUntil = now.Add(-time.Nanosecond)
				}
				if err := r.livenessRepo.UpdateSchedulingLiveness(ctx, account.ID, snapshot); err != nil {
					logger.LegacyPrintf("service.super_priority_runner", "[SchedulingLivenessRunner] clear legacy liveness marker account=%d failed: %v", account.ID, err)
				}
			}
			continue
		}
		if !schedulingLivenessProbeEligible(&account, includeUnschedulable) {
			continue
		}
		snapshot := decodeSchedulingLiveness(account.Extra)
		if force || schedulingLivenessProbeDue(snapshot, now, expr) {
			due = append(due, account)
		} else {
			updateNextRunAt(schedulingLivenessNextProbeAt(snapshot.LastAttemptAt, expr))
		}
	}
	if len(due) == 0 {
		r.setNextRunAt(r.nextScheduledScanAt(nextRunAt))
		if force {
			r.finishRun(trigger, now, refreshResult, nil, schedulingLivenessTimePtr(nextRunAt))
		}
		return refreshResult, nil
	}
	startedAt := time.Now()
	r.startRun()
	refreshResult.Checked = len(due)

	configuredModelID := r.state.TestModelID()
	failureThreshold := r.state.FailureThreshold()
	sem := make(chan struct{}, schedulingLivenessMaxWorkers)
	var wg sync.WaitGroup
	var resultMu sync.Mutex
	for i := range due {
		account := due[i]
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			attemptedAt := time.Now()
			modelID := schedulingLivenessTestModel(&account, configuredModelID)
			if account.IsOpenAI() && modelID == "" {
				// An empty mapping is not proof that a shared model is supported.
				// Skip this account instead of sending an unverifiable model and
				// turning a harmless model mismatch into a liveness failure.
				resultMu.Lock()
				refreshResult.Skipped++
				updateNextRunAt(schedulingLivenessNextProbeAt(attemptedAt, expr))
				resultMu.Unlock()
				return
			}
			result, testErr := r.accountTestSvc.RunTestBackground(ctx, account.ID, modelID)
			if (testErr != nil && isAccountTestUnsupportedModelError(testErr.Error())) ||
				(result != nil && isAccountTestUnsupportedModelError(result.ErrorMessage)) {
				resultMu.Lock()
				refreshResult.Skipped++
				updateNextRunAt(schedulingLivenessNextProbeAt(attemptedAt, expr))
				resultMu.Unlock()
				return
			}
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
			if err := r.livenessRepo.UpdateSchedulingLiveness(ctx, account.ID, snapshot); err != nil {
				logger.LegacyPrintf("service.super_priority_runner", "[SchedulingLivenessRunner] persist account=%d failed: %v", account.ID, err)
				resultMu.Lock()
				refreshResult.Failed++
				updateNextRunAt(schedulingLivenessNextProbeAt(attemptedAt, expr))
				resultMu.Unlock()
				return
			}
			resultMu.Lock()
			if succeeded {
				refreshResult.Succeeded++
			} else {
				refreshResult.Failed++
			}
			updateNextRunAt(schedulingLivenessNextProbeAt(attemptedAt, expr))
			resultMu.Unlock()
			if !succeeded {
				logger.LegacyPrintf("service.super_priority_runner", "[SchedulingLivenessRunner] account=%d status=%s failures=%d error=%s", account.ID, snapshot.Status, snapshot.FailureCount, strings.TrimSpace(errorMessage))
			}
		}()
	}
	wg.Wait()
	r.finishRun(trigger, startedAt, refreshResult, nil, schedulingLivenessTimePtr(nextRunAt))
	return refreshResult, nil
}

func (r *SuperPriorityRunner) startRun() {
	r.statusMu.Lock()
	r.runtime.Enabled = true
	r.runtime.Running = true
	r.runtime.NextRunAt = nil
	r.statusMu.Unlock()
}

func (r *SuperPriorityRunner) finishRun(trigger string, startedAt time.Time, result SchedulingRefreshResult, runErr error, nextRunAt *time.Time) {
	finishedAt := time.Now()
	last := &SchedulingLivenessRunStatus{
		Trigger: trigger, StartedAt: startedAt, FinishedAt: finishedAt, Result: result,
	}
	if runErr != nil {
		last.Error = runErr.Error()
	}
	if nextRunAt != nil {
		next := r.nextScheduledScanAt(*nextRunAt)
		nextRunAt = schedulingLivenessTimePtr(next)
	}
	r.statusMu.Lock()
	r.runtime.Enabled = true
	r.runtime.Running = false
	r.runtime.NextRunAt = nextRunAt
	r.runtime.LastRun = last
	r.statusMu.Unlock()
}

func (r *SuperPriorityRunner) setNextRunAt(next time.Time) {
	next = r.nextScheduledScanAt(next)
	r.statusMu.Lock()
	r.runtime.Enabled = r.state != nil && r.state.BaseStrategy() == AccountSchedulingStrategyLowestCost
	r.runtime.Running = false
	r.runtime.NextRunAt = schedulingLivenessTimePtr(next)
	r.statusMu.Unlock()
}

func (r *SuperPriorityRunner) nextScheduledScanAt(candidate time.Time) time.Time {
	if r == nil || r.cron == nil || r.scanEntryID == 0 {
		return candidate
	}
	return schedulingLivenessNextScheduledScanAt(candidate, r.cron.Entry(r.scanEntryID), time.Now())
}

// schedulingLivenessNextScheduledScanAt aligns an account due time to the
// runner's next minute scan. The API should never expose a due timestamp that
// is already in the past while the runner is still waiting for that scan.
func schedulingLivenessNextScheduledScanAt(candidate time.Time, entry cron.Entry, now time.Time) time.Time {
	if candidate.IsZero() || entry.Next.IsZero() || entry.Schedule == nil {
		return candidate
	}

	next := entry.Next
	for next.Before(candidate) || !next.After(now) {
		candidateNext := entry.Schedule.Next(next)
		if candidateNext.IsZero() || !candidateNext.After(next) {
			return candidate
		}
		next = candidateNext
	}
	return next
}

func schedulingLivenessTimePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	copy := value
	return &copy
}

// schedulingLivenessTestModel keeps a configured OpenAI probe model only when
// the account exposes it. A mapped account otherwise uses one of its explicit
// model-list keys, which is the same list returned by the account models API.
func schedulingLivenessTestModel(account *Account, configuredModelID string) string {
	if account == nil {
		return ""
	}

	configuredModelID = strings.TrimSpace(configuredModelID)
	candidates := accountTestModelCandidates(account)
	if account.IsOpenAI() {
		// Prefer a concrete model owned by this account. The configured model is
		// a group-wide fallback and must not hide a better account-specific one.
		for _, candidate := range candidates {
			if configuredModelID == "" || !strings.EqualFold(candidate, configuredModelID) {
				return candidate
			}
		}
		// Only use the shared model when an explicit account mapping verifies it.
		if configuredModelID != "" && len(account.GetModelMapping()) > 0 && account.IsModelSupported(configuredModelID) {
			return configuredModelID
		}
		return ""
	}

	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
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
