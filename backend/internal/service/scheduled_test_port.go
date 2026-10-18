package service

import (
	"context"
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/modules/qualityevidence"
	"time"
)

// Scheduled test plan modes.
const (
	// ScheduledTestModeDefault runs the standard connectivity test.
	ScheduledTestModeDefault = "default"
	// ScheduledTestModePelican runs the rule-driven downgrade (降智) probe.
	// Only OpenAI-platform accounts are eligible.
	ScheduledTestModePelican = "pelican"
)

// Rule execution strategies for plans with multiple selected rules.
const (
	// ScheduledTestStrategyAll runs every selected rule on each execution.
	ScheduledTestStrategyAll = "all"
	// ScheduledTestStrategyRandom runs one randomly picked rule per execution.
	ScheduledTestStrategyRandom = "random"
	// ScheduledTestStrategySequential rotates through the selected rules.
	ScheduledTestStrategySequential = "sequential"
)

// Downgrade handling actions.
const (
	// ScheduledTestActionMark only records the downgrade marker on the account.
	ScheduledTestActionMark = "mark"
	// ScheduledTestActionPause additionally disables scheduling once the
	// consecutive-downgrade threshold is reached.
	ScheduledTestActionPause = "pause"
)

// NormalizeScheduledTestMode maps empty or unknown modes to the default mode.
func NormalizeScheduledTestMode(mode string) string {
	if mode == ScheduledTestModePelican {
		return ScheduledTestModePelican
	}
	return ScheduledTestModeDefault
}

func normalizeRuleStrategy(strategy string) string {
	switch strategy {
	case ScheduledTestStrategyRandom, ScheduledTestStrategySequential:
		return strategy
	default:
		return ScheduledTestStrategyAll
	}
}

func normalizeDowngradeAction(action string) string {
	if action == ScheduledTestActionPause {
		return ScheduledTestActionPause
	}
	return ScheduledTestActionMark
}

// DegradationRule is a user-manageable probe rule (降智规则库 entry).
type DegradationRule struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Enabled   bool              `json:"enabled"`
	Evaluator string            `json:"evaluator"`
	Prompt    string            `json:"prompt"`
	Params    DegradationParams `json:"params"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// DegradationParams carries evaluator-specific detection criteria.
type DegradationParams struct {
	// ProbeEvaluator preserves the direct account-test editor's automatic mode.
	ProbeEvaluator string `json:"probe_evaluator,omitempty"`
	// ExpectedAnswer is required by the numeric_answer evaluator.
	ExpectedAnswer int `json:"expected_answer,omitempty"`
	// Keywords are required by the contains_keywords evaluator.
	Keywords []string `json:"keywords,omitempty"`
	// KeywordMatch is "all" (default) or "any".
	KeywordMatch string `json:"keyword_match,omitempty"`
}

// Degradation rule evaluators.
const (
	DegradationEvaluatorNumeric  = "numeric_answer"
	DegradationEvaluatorKeywords = "contains_keywords"
	DegradationEvaluatorRefusal  = "refusal_only"
	DegradationEvaluatorHTML     = "html_document"
)

// ScheduledTestPlan represents a scheduled test plan domain model.
type ScheduledTestPlan struct {
	ID                     int64      `json:"id"`
	AccountID              int64      `json:"account_id"`
	ModelID                string     `json:"model_id"`
	CronExpression         string     `json:"cron_expression"`
	Enabled                bool       `json:"enabled"`
	MaxResults             int        `json:"max_results"`
	AutoRecover            bool       `json:"auto_recover"`
	AutoRecoverSchedulable bool       `json:"auto_recover_schedulable"`
	Mode                   string     `json:"mode"`
	AutoPauseOnDowngrade   bool       `json:"auto_pause_on_downgrade"`
	RuleIDs                []int64    `json:"rule_ids"`
	RuleStrategy           string     `json:"rule_strategy"`
	DowngradeAction        string     `json:"downgrade_action"`
	DowngradeThreshold     int        `json:"downgrade_threshold"`
	ConsecutiveDowngrades  int        `json:"consecutive_downgrades"`
	LastRuleIndex          int        `json:"last_rule_index"`
	LastRunAt              *time.Time `json:"last_run_at"`
	NextRunAt              *time.Time `json:"next_run_at"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	AccountName            string     `json:"account_name,omitempty"`
	AccountPlatform        string     `json:"account_platform,omitempty"`
}

// RuleOutcome is the per-rule verdict of one probe execution.
type RuleOutcome struct {
	PromptSummary string `json:"prompt_summary,omitempty"`
	RuleID        int64  `json:"rule_id"`
	RuleName      string `json:"rule_name"`
	Evaluator     string `json:"evaluator"`
	Passed        bool   `json:"passed"`
	Reason        string `json:"reason,omitempty"`
}

// ScheduledTestResult represents a single test execution result.
type ScheduledTestResult struct {
	QualityQuestion *qualityevidence.Question `json:"quality_question,omitempty"`
	QuestionSummary string                    `json:"question_summary,omitempty"`
	ID              int64                     `json:"id"`
	PlanID          int64                     `json:"plan_id"`
	Status          string                    `json:"status"`
	ResponseText    string                    `json:"response_text"`
	ErrorMessage    string                    `json:"error_message"`
	Downgraded      bool                      `json:"downgraded"`
	DowngradeReason string                    `json:"downgrade_reason"`
	ResponseModel   string                    `json:"response_model"`
	RuleResults     json.RawMessage           `json:"rule_results,omitempty"`
	LatencyMs       int64                     `json:"latency_ms"`
	StartedAt       time.Time                 `json:"started_at"`
	FinishedAt      time.Time                 `json:"finished_at"`
	CreatedAt       time.Time                 `json:"created_at"`
}

// ScheduledTestPlanRepository defines the data access interface for test plans.
type ScheduledTestPlanRepository interface {
	Create(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error)
	GetByID(ctx context.Context, id int64) (*ScheduledTestPlan, error)
	ListByAccountID(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error)
	ListAll(ctx context.Context) ([]*ScheduledTestPlan, error)
	DeleteByAccountAndCron(ctx context.Context, accountID int64, cronExpression string) error
	ListDue(ctx context.Context, now time.Time) ([]*ScheduledTestPlan, error)
	Update(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error)
	Delete(ctx context.Context, id int64) error
	UpdateAfterRun(ctx context.Context, id int64, lastRunAt time.Time, nextRunAt time.Time) error
	// BumpConsecutiveDowngrades increments the plan's consecutive downgrade
	// counter and returns the new value.
	BumpConsecutiveDowngrades(ctx context.Context, id int64) (int, error)
	// ResetConsecutiveDowngrades clears the consecutive downgrade counter.
	ResetConsecutiveDowngrades(ctx context.Context, id int64) error
	// BumpLastRuleIndex advances the sequential rotation cursor.
	BumpLastRuleIndex(ctx context.Context, id int64, selected int) error
}

// ScheduledTestResultRepository defines the data access interface for test results.
type ScheduledTestResultRepository interface {
	Create(ctx context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error)
	ListByPlanID(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error)
	PruneOldResults(ctx context.Context, planID int64, keepCount int) error
}

// DegradationRuleRepository defines the data access interface for the
// user-manageable probe rule library.
type DegradationRuleRepository interface {
	List(ctx context.Context, onlyEnabled bool) ([]*DegradationRule, error)
	GetByID(ctx context.Context, id int64) (*DegradationRule, error)
	Create(ctx context.Context, rule *DegradationRule) (*DegradationRule, error)
	Update(ctx context.Context, rule *DegradationRule) (*DegradationRule, error)
	Delete(ctx context.Context, id int64) error
}
