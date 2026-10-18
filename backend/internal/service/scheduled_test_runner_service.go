package service

import (
	"context"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/robfig/cron/v3"
)

const scheduledTestDefaultMaxWorkers = 10

// scheduledTestAccountTester abstracts the background account test execution so
// unit tests can stub it (same pattern as super_priority_runner.go).
type scheduledTestAccountTester interface {
	IsAutomaticQualityResult(context.Context, *ScheduledTestResult) bool
	RunTestBackgroundWithMode(ctx context.Context, accountID int64, modelID, mode string) (*ScheduledTestResult, error)
	RunDegradationProbeRules(ctx context.Context, accountID int64, modelID string, rules []*DegradationRule) (*ScheduledTestResult, error)
}

// ScheduledTestRunnerService periodically scans due test plans and executes them.
type ScheduledTestRunnerService struct {
	planRepo       ScheduledTestPlanRepository
	scheduledSvc   *ScheduledTestService
	accountTestSvc scheduledTestAccountTester
	rateLimitSvc   *RateLimitService
	cfg            *config.Config

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewScheduledTestRunnerService creates a new runner.
func NewScheduledTestRunnerService(
	planRepo ScheduledTestPlanRepository,
	scheduledSvc *ScheduledTestService,
	accountTestSvc scheduledTestAccountTester,
	rateLimitSvc *RateLimitService,
	cfg *config.Config,
) *ScheduledTestRunnerService {
	return &ScheduledTestRunnerService{
		planRepo:       planRepo,
		scheduledSvc:   scheduledSvc,
		accountTestSvc: accountTestSvc,
		rateLimitSvc:   rateLimitSvc,
		cfg:            cfg,
	}
}

// Start begins the cron ticker (every minute).
func (s *ScheduledTestRunnerService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		loc := time.Local
		if s.cfg != nil {
			if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithParser(scheduledTestCronParser), cron.WithLocation(loc), cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
		_, err := c.AddFunc("* * * * *", func() { s.runScheduled() })
		if err != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] not started (invalid schedule): %v", err)
			return
		}
		if s.scheduledSvc != nil && s.scheduledSvc.automatic != nil {
			_, _ = c.AddFunc("* * * * *", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
				if err := s.scheduledSvc.automatic.Run(ctx); err != nil {
					logger.LegacyPrintf("service.scheduled_test_runner", "automatic detection: %v", err)
				}
			})
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] started (tick=every minute)")
	})
}

// Stop gracefully shuts down the cron scheduler.
func (s *ScheduledTestRunnerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] cron stop timed out")
			}
		}
	})
}

func (s *ScheduledTestRunnerService) runScheduled() {
	// Delay 10s so execution lands at ~:10 of each minute instead of :00.
	time.Sleep(10 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	now := time.Now()
	plans, err := s.planRepo.ListDue(ctx, now)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] ListDue error: %v", err)
		return
	}
	if len(plans) == 0 {
		return
	}

	logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] found %d due plans", len(plans))

	sem := make(chan struct{}, scheduledTestDefaultMaxWorkers)
	var wg sync.WaitGroup

	for _, plan := range plans {
		sem <- struct{}{}
		wg.Add(1)
		go func(p *ScheduledTestPlan) {
			defer wg.Done()
			defer func() { <-sem }()
			s.runOnePlan(ctx, p)
		}(plan)
	}

	wg.Wait()
}

func (s *ScheduledTestRunnerService) runOnePlan(ctx context.Context, plan *ScheduledTestPlan) {
	// Unified policies own matching accounts; keep legacy plans for accounts
	// outside their scope and advance skipped plans without issuing duplicate probes.
	if s.scheduledSvc != nil && s.scheduledSvc.automatic != nil && s.scheduledSvc.accountRepo != nil {
		account, err := s.scheduledSvc.accountRepo.GetByID(ctx, plan.AccountID)
		if err == nil && s.scheduledSvc.automatic.Handles(ctx, automaticDetectionAccount(account)) {
			s.finishPlanRun(ctx, plan)
			return
		}
	}
	mode := NormalizeScheduledTestMode(plan.Mode)

	// Pelican plans are OpenAI-only; skip defensively for plans created before
	// create/update validation existed.
	if mode == ScheduledTestModePelican && !s.accountPlatformOpenAI(ctx, plan.AccountID, plan.ID) {
		return
	}

	var result *ScheduledTestResult
	var err error
	if mode == ScheduledTestModePelican {
		rules := s.resolveRunOrder(ctx, plan)
		if len(rules) == 0 {
			// Every referenced rule is disabled/missing: record a skip, leave
			// the consecutive counter untouched, and keep the cron moving.
			now := time.Now()
			result = &ScheduledTestResult{
				Status:          "success",
				Downgraded:      false,
				DowngradeReason: "规则库全部停用，跳过降智探针",
				StartedAt:       now,
				FinishedAt:      now,
			}
			if err := s.scheduledSvc.SaveResult(ctx, plan.ID, plan.MaxResults, result); err != nil {
				logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d SaveResult error: %v", plan.ID, err)
			}
			s.finishPlanRun(ctx, plan)
			return
		}
		result, err = s.accountTestSvc.RunDegradationProbeRules(WithAccountDetectionSource(ctx, "scheduled"), plan.AccountID, plan.ModelID, rules)
	} else {
		result, err = s.accountTestSvc.RunTestBackgroundWithMode(WithAccountDetectionSource(ctx, "scheduled"), plan.AccountID, plan.ModelID, mode)
	}
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d RunTestBackground error: %v", plan.ID, err)
		return
	}

	if err := s.scheduledSvc.SaveResult(ctx, plan.ID, plan.MaxResults, result); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d SaveResult error: %v", plan.ID, err)
	}

	s.handleDegradationOutcome(ctx, plan, result, mode)

	// A suspected-downgrade result must never be treated as healthy by the
	// auto-recover hooks of the same plan.
	healthy := result.Status == "success" && !result.Downgraded

	// Auto-recover account if test succeeded and auto_recover is enabled.
	if healthy && plan.AutoRecover {
		s.tryRecoverAccount(ctx, plan.AccountID, plan.ID)
	}
	// Auto re-enable scheduling if test succeeded and auto_recover_schedulable is enabled.
	if healthy && plan.AutoRecoverSchedulable {
		s.tryReEnableScheduling(ctx, plan.AccountID, plan.ID)
	}

	s.finishPlanRun(ctx, plan)
}

// finishPlanRun advances the cron schedule after a plan execution.
func (s *ScheduledTestRunnerService) finishPlanRun(ctx context.Context, plan *ScheduledTestPlan) {
	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d computeNextRun error: %v", plan.ID, err)
		return
	}

	if err := s.planRepo.UpdateAfterRun(ctx, plan.ID, time.Now(), nextRun); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d UpdateAfterRun error: %v", plan.ID, err)
	}
}

// resolveRunOrder loads the plan's referenced rules and orders them by the
// plan's execution strategy. Rules that are disabled or deleted are skipped
// with a warning (never counted as failures). An empty rule_ids selection
// falls back to the built-in pelican+candy pair for backward compatibility.
func (s *ScheduledTestRunnerService) resolveRunOrder(ctx context.Context, plan *ScheduledTestPlan) []*DegradationRule {
	allRules, err := s.scheduledSvc.ListDegradationRules(ctx, false)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d list rules failed: %v", plan.ID, err)
		return nil
	}

	var selected []*DegradationRule
	if len(plan.RuleIDs) == 0 {
		// Legacy plan: built-in pair, same behavior as before the rule library.
		for _, rule := range allRules {
			if rule.Name == "pelican" || rule.Name == "candy" {
				selected = append(selected, rule)
			}
		}
	} else {
		byID := make(map[int64]*DegradationRule, len(allRules))
		for _, rule := range allRules {
			byID[rule.ID] = rule
		}
		for _, id := range plan.RuleIDs {
			rule, ok := byID[id]
			if !ok || !rule.Enabled {
				logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d skip rule %d (missing or disabled)", plan.ID, id)
				continue
			}
			selected = append(selected, rule)
		}
	}
	if len(selected) == 0 {
		return nil
	}

	switch plan.RuleStrategy {
	case ScheduledTestStrategyRandom:
		return []*DegradationRule{selected[rand.IntN(len(selected))]}
	case ScheduledTestStrategySequential:
		idx := plan.LastRuleIndex % len(selected)
		next := (plan.LastRuleIndex + 1) % len(selected)
		if err := s.planRepo.BumpLastRuleIndex(ctx, plan.ID, next); err != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d bump rule index failed: %v", plan.ID, err)
		}
		// Keep the in-memory cursor fresh so repeated runs of the same plan
		// object (unit tests, batch scans) rotate instead of repeating.
		plan.LastRuleIndex = next
		return []*DegradationRule{selected[idx]}
	default: // all
		return selected
	}
}

// accountPlatformOpenAI reports whether the plan's account is an OpenAI-platform account.
func (s *ScheduledTestRunnerService) accountPlatformOpenAI(ctx context.Context, accountID, planID int64) bool {
	if s.rateLimitSvc == nil {
		return false
	}
	account, err := s.rateLimitSvc.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d get account=%d failed: %v", planID, accountID, err)
		return false
	}
	if account.Platform != PlatformOpenAI {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d skip pelican: account=%d platform=%s is not OpenAI", planID, accountID, account.Platform)
		return false
	}
	return true
}

// handleDegradationOutcome persists the downgrade marker on the account and
// applies the plan's handling action once the consecutive-downgrade threshold
// is reached. Mirrors the manual batch flow: only completed runs write the
// marker (true on downgrade, false on a clean pass); failed runs write nothing.
func (s *ScheduledTestRunnerService) handleDegradationOutcome(ctx context.Context, plan *ScheduledTestPlan, result *ScheduledTestResult, mode string) {
	if mode != ScheduledTestModePelican || s.rateLimitSvc == nil || result.Status != "success" || !s.accountTestSvc.IsAutomaticQualityResult(ctx, result) {
		return
	}

	if result.Status == "success" {
		if err := s.rateLimitSvc.accountRepo.UpdateExtra(ctx, plan.AccountID, map[string]any{
			"pelican_downgraded": result.Downgraded,
		}); err != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d pelican: account=%d update extra failed: %v", plan.ID, plan.AccountID, err)
		}
		if !result.Downgraded {
			// A clean pass resets the consecutive-downgrade counter.
			if err := s.planRepo.ResetConsecutiveDowngrades(ctx, plan.ID); err != nil {
				logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d reset downgrade counter failed: %v", plan.ID, err)
			}
		}
	}

	if !result.Downgraded {
		return
	}

	// Legacy plans only carry AutoPauseOnDowngrade; derive the action from it.
	action := normalizeDowngradeAction(plan.DowngradeAction)
	if plan.DowngradeAction == "" && plan.AutoPauseOnDowngrade {
		action = ScheduledTestActionPause
	}
	threshold := plan.DowngradeThreshold
	if threshold < 1 {
		threshold = 1
	}
	count, err := s.planRepo.BumpConsecutiveDowngrades(ctx, plan.ID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d bump downgrade counter failed: %v", plan.ID, err)
		return
	}
	if count < threshold {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d downgrade %d/%d (below threshold, no action): %s", plan.ID, count, threshold, result.DowngradeReason)
		return
	}
	if action != ScheduledTestActionPause {
		return
	}

	account, err := s.rateLimitSvc.accountRepo.GetByID(ctx, plan.AccountID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d degradation auto-pause: get account=%d failed: %v", plan.ID, plan.AccountID, err)
		return
	}
	if !account.Schedulable {
		return
	}
	if err := s.rateLimitSvc.accountRepo.SetSchedulable(ctx, plan.AccountID, false); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d degradation auto-pause: account=%d set schedulable failed: %v", plan.ID, plan.AccountID, err)
		return
	}
	logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d degradation auto-pause: account=%d paused after %d consecutive 疑似降智: %s", plan.ID, plan.AccountID, count, result.DowngradeReason)
}

// tryRecoverAccount attempts to recover an account from recoverable runtime state.
func (s *ScheduledTestRunnerService) tryRecoverAccount(ctx context.Context, accountID int64, planID int64) {
	if s.rateLimitSvc == nil {
		return
	}

	recovery, err := s.rateLimitSvc.RecoverAccountAfterSuccessfulTest(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover failed: %v", planID, err)
		return
	}
	if recovery == nil {
		return
	}

	if recovery.ClearedError {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d recovered from error status", planID, accountID)
	}
	if recovery.ClearedRateLimit {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d cleared rate-limit/runtime state", planID, accountID)
	}
}

// tryReEnableScheduling re-enables scheduling for an account that was paused
// (schedulable=false) after a successful scheduled test proves it is healthy.
func (s *ScheduledTestRunnerService) tryReEnableScheduling(ctx context.Context, accountID int64, planID int64) {
	if s.rateLimitSvc == nil {
		return
	}

	// Access the account repository through the rate limit service to avoid
	// adding a new dependency to the runner constructor.
	account, err := s.rateLimitSvc.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-schedulable: get account=%d failed: %v", planID, accountID, err)
		return
	}

	// Only re-enable if currently paused; skip accounts already schedulable.
	if account.Schedulable {
		return
	}

	if err := s.rateLimitSvc.accountRepo.SetSchedulable(ctx, accountID, true); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-schedulable: account=%d set schedulable failed: %v", planID, accountID, err)
		return
	}

	logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-schedulable: account=%d re-enabled scheduling after successful test", planID, accountID)
}
