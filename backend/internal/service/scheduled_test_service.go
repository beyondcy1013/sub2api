package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/autodetection"
	"github.com/robfig/cron/v3"
)

var scheduledTestCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// ScheduledTestService provides CRUD operations for scheduled test plans and results.
type ScheduledTestService struct {
	automatic   *autodetection.Engine
	planRepo    ScheduledTestPlanRepository
	resultRepo  ScheduledTestResultRepository
	accountRepo AccountRepository
	ruleRepo    DegradationRuleRepository
}

// NewScheduledTestService creates a new ScheduledTestService.
func NewScheduledTestService(
	planRepo ScheduledTestPlanRepository,
	resultRepo ScheduledTestResultRepository,
	accountRepo AccountRepository,
	ruleRepo DegradationRuleRepository,
) *ScheduledTestService {
	return &ScheduledTestService{
		planRepo:    planRepo,
		resultRepo:  resultRepo,
		accountRepo: accountRepo,
		ruleRepo:    ruleRepo,
	}
}

// validatePlan rejects pelican-mode plans on non-OpenAI accounts and
// normalizes/validates the rule selection, execution strategy and downgrade
// handling fields.
func (s *ScheduledTestService) validatePlan(ctx context.Context, plan *ScheduledTestPlan) error {
	if NormalizeScheduledTestMode(plan.Mode) != ScheduledTestModePelican {
		return nil
	}
	plan.RuleStrategy = normalizeRuleStrategy(plan.RuleStrategy)
	if plan.DowngradeAction == "" {
		// Legacy callers only send AutoPauseOnDowngrade.
		if plan.AutoPauseOnDowngrade {
			plan.DowngradeAction = ScheduledTestActionPause
		}
	}
	plan.DowngradeAction = normalizeDowngradeAction(plan.DowngradeAction)
	plan.AutoPauseOnDowngrade = plan.DowngradeAction == ScheduledTestActionPause
	if plan.DowngradeThreshold < 1 {
		plan.DowngradeThreshold = 1
	}
	if plan.RuleIDs == nil {
		plan.RuleIDs = []int64{}
	}
	// Empty selection falls back to the built-in pair at run time; a non-empty
	// selection must reference enabled rules.
	if len(plan.RuleIDs) > 0 && s.ruleRepo != nil {
		rules, err := s.ruleRepo.List(ctx, false)
		if err != nil {
			return fmt.Errorf("加载降智规则失败: %w", err)
		}
		byID := make(map[int64]*DegradationRule, len(rules))
		for _, rule := range rules {
			byID[rule.ID] = rule
		}
		for _, id := range plan.RuleIDs {
			rule, ok := byID[id]
			if !ok {
				return fmt.Errorf("降智规则 %d 不存在", id)
			}
			if !rule.Enabled {
				return fmt.Errorf("降智规则 %q 已停用", rule.Name)
			}
		}
	}
	if s.accountRepo == nil {
		return nil
	}
	account, err := s.accountRepo.GetByID(ctx, plan.AccountID)
	if err != nil {
		return fmt.Errorf("account %d not found: %w", plan.AccountID, err)
	}
	if account.Platform != PlatformOpenAI {
		return fmt.Errorf("降智检测(鹈鹕)计划仅支持 OpenAI 平台账号")
	}
	return nil
}

// ListDegradationRules returns rules, optionally only enabled ones.
func (s *ScheduledTestService) ListDegradationRules(ctx context.Context, onlyEnabled bool) ([]*DegradationRule, error) {
	if s.ruleRepo == nil {
		return nil, fmt.Errorf("规则库不可用")
	}
	return s.ruleRepo.List(ctx, onlyEnabled)
}

// CreateDegradationRule adds a new rule to the library.
func (s *ScheduledTestService) CreateDegradationRule(ctx context.Context, rule *DegradationRule) (*DegradationRule, error) {
	if err := validateDegradationRule(rule); err != nil {
		return nil, err
	}
	return s.ruleRepo.Create(ctx, rule)
}

// UpdateDegradationRule edits an existing rule.
func (s *ScheduledTestService) UpdateDegradationRule(ctx context.Context, rule *DegradationRule) (*DegradationRule, error) {
	if err := validateDegradationRule(rule); err != nil {
		return nil, err
	}
	return s.ruleRepo.Update(ctx, rule)
}

// DeleteDegradationRule removes a rule; plans referencing it skip it at run time.
func (s *ScheduledTestService) DeleteDegradationRule(ctx context.Context, id int64) error {
	return s.ruleRepo.Delete(ctx, id)
}

func validateDegradationRule(rule *DegradationRule) error {
	rule.Prompt = strings.TrimSpace(rule.Prompt)
	if rule.Prompt == "" {
		return fmt.Errorf("规则题面不能为空")
	}
	rule.Name = strings.TrimSpace(rule.Name)
	if rule.Name == "" {
		runes := []rune(rule.Prompt)
		if len(runes) > 40 {
			rule.Name = string(runes[:40]) + "..."
		} else {
			rule.Name = string(runes)
		}
	}
	if rule.Name == "" {
		return fmt.Errorf("规则名称不能为空")
	}
	if rule.Params.ProbeEvaluator != "" {
		switch rule.Params.ProbeEvaluator {
		case "auto", "html", "numeric", "reply":
		default:
			return fmt.Errorf("不支持的直接测试判定方式 %q", rule.Params.ProbeEvaluator)
		}
		// Keep scheduled/request-time judging consistent with the same saved question.
		switch resolveDirectProbeEvaluator(rule.Prompt, rule.Params.ProbeEvaluator) {
		case "html":
			rule.Evaluator = DegradationEvaluatorHTML
		case "numeric":
			rule.Evaluator = DegradationEvaluatorNumeric
		default:
			rule.Evaluator = DegradationEvaluatorRefusal
		}
	}
	switch rule.Evaluator {
	case DegradationEvaluatorNumeric:
	case DegradationEvaluatorKeywords:
		if len(rule.Params.Keywords) == 0 {
			return fmt.Errorf("关键词规则至少需要一个关键词")
		}
	case DegradationEvaluatorRefusal, DegradationEvaluatorHTML:
	default:
		return fmt.Errorf("不支持的评估器类型 %q", rule.Evaluator)
	}
	return nil
}

// CreatePlan validates the cron expression, computes next_run_at, and persists the plan.
func (s *ScheduledTestService) CreatePlan(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	plan.Mode = NormalizeScheduledTestMode(plan.Mode)

	if err := s.validatePlan(ctx, plan); err != nil {
		return nil, err
	}

	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	plan.NextRunAt = &nextRun

	if plan.MaxResults <= 0 {
		plan.MaxResults = 50
	}

	return s.planRepo.Create(ctx, plan)
}

// GetPlan retrieves a plan by ID.
func (s *ScheduledTestService) GetPlan(ctx context.Context, id int64) (*ScheduledTestPlan, error) {
	return s.planRepo.GetByID(ctx, id)
}

// ListPlansByAccount returns all plans for a given account.
func (s *ScheduledTestService) ListPlansByAccount(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error) {
	return s.planRepo.ListByAccountID(ctx, accountID)
}

// ListAllPlans returns every plan that still belongs to a non-deleted account.
func (s *ScheduledTestService) ListAllPlans(ctx context.Context) ([]*ScheduledTestPlan, error) {
	return s.planRepo.ListAll(ctx)
}

// BatchUpsertPlans replaces the matching plan for every selected account and
// creates a new plan using the supplied template. A plan is matched by both
// account ID and cron expression, so other schedules remain untouched.
func (s *ScheduledTestService) BatchUpsertPlans(ctx context.Context, accountIDs []int64, template *ScheduledTestPlan) (created int, failed int, errors []string) {
	for _, accountID := range accountIDs {
		if err := s.planRepo.DeleteByAccountAndCron(ctx, accountID, template.CronExpression); err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("account %d: %v", accountID, err))
			continue
		}

		plan := *template
		plan.ID = 0
		plan.AccountID = accountID
		plan.NextRunAt = nil
		plan.LastRunAt = nil
		plan.CreatedAt = time.Time{}
		plan.UpdatedAt = time.Time{}
		plan.AccountName = ""
		plan.AccountPlatform = ""
		if _, err := s.CreatePlan(ctx, &plan); err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("account %d: %v", accountID, err))
			continue
		}
		created++
	}
	return created, failed, errors
}

// UpdatePlan validates cron and updates the plan.
func (s *ScheduledTestService) UpdatePlan(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	plan.Mode = NormalizeScheduledTestMode(plan.Mode)

	if err := s.validatePlan(ctx, plan); err != nil {
		return nil, err
	}

	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	plan.NextRunAt = &nextRun

	return s.planRepo.Update(ctx, plan)
}

// DeletePlan removes a plan and its results (via CASCADE).
func (s *ScheduledTestService) DeletePlan(ctx context.Context, id int64) error {
	return s.planRepo.Delete(ctx, id)
}

// ListResults returns the most recent results for a plan.
func (s *ScheduledTestService) ListResults(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.resultRepo.ListByPlanID(ctx, planID, limit)
}

// SaveResult inserts a result and prunes old entries beyond maxResults.
func (s *ScheduledTestService) SaveResult(ctx context.Context, planID int64, maxResults int, result *ScheduledTestResult) error {
	result.PlanID = planID
	if _, err := s.resultRepo.Create(ctx, result); err != nil {
		return err
	}
	return s.resultRepo.PruneOldResults(ctx, planID, maxResults)
}

func computeNextRun(cronExpr string, from time.Time) (time.Time, error) {
	sched, err := scheduledTestCronParser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(from), nil
}

// AutomaticDetection is the single policy owner shared with the runner and admin UI.
func (s *ScheduledTestService) AutomaticDetection() *autodetection.Engine { return s.automatic }
