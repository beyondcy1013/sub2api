package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/modules/qualityevidence"
	"github.com/Wei-Shaw/sub2api/internal/modules/rotationnotice"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"golang.org/x/sync/singleflight"
)

// RequestQualityProbeService coordinates per-request capability checks and
// explicit manual tests. Historical proof helpers remain for legacy callers;
// gateway response gates always use StartRequestQualityCheck.
type RequestQualityProbeService struct {
	settingService     *SettingService
	accountRepo        AccountRepository
	ruleRepo           DegradationRuleRepository
	accountTestService qualityProbeTester

	detectionRepo AccountDetectionRepository
	recentSuccess sync.Map // key: "accountID:model" -> time.Time
	probeFlight   singleflight.Group
}

type qualityProbeTester interface {
	RunDegradationProbeRules(context.Context, int64, string, []*DegradationRule) (*ScheduledTestResult, error)
}

// NewRequestQualityProbeService creates a new RequestQualityProbeService.
func NewRequestQualityProbeService(
	settingService *SettingService,
	accountRepo AccountRepository,
	ruleRepo DegradationRuleRepository,
	accountTestService *AccountTestService,
) *RequestQualityProbeService {
	var detectionRepo AccountDetectionRepository
	var tester qualityProbeTester
	if accountTestService != nil {
		accountTestService.qualityRuleRepo = ruleRepo
		detectionRepo = accountTestService.detectionRepo
		tester = accountTestService
	}
	return &RequestQualityProbeService{
		detectionRepo:      detectionRepo,
		settingService:     settingService,
		accountRepo:        accountRepo,
		ruleRepo:           ruleRepo,
		accountTestService: tester,
	}
}

// TouchRecentSuccess records that an account successfully served or tested a model.
func (s *RequestQualityProbeService) TouchRecentSuccess(accountID int64, model string) {
	if s == nil {
		return
	}
	now := time.Now()
	cleanModel := strings.ToLower(strings.TrimSpace(model))
	if cleanModel != "" {
		s.recentSuccess.Store(fmt.Sprintf("%d:%s", accountID, cleanModel), now)
	}

}

// HasRecentSuccess reports whether the account has a successful turn within recentMinutes.
func (s *RequestQualityProbeService) HasRecentSuccess(accountID int64, model string, recentMinutes int) bool {
	if s == nil {
		return false
	}
	if recentMinutes <= 0 {
		recentMinutes = 10
	}
	window := time.Duration(recentMinutes) * time.Minute
	now := time.Now()

	cleanModel := strings.ToLower(strings.TrimSpace(model))
	if cleanModel != "" {
		if val, ok := s.recentSuccess.Load(fmt.Sprintf("%d:%s", accountID, cleanModel)); ok {
			if t, ok := val.(time.Time); ok && now.Sub(t) < window {
				return true
			}
		}
	}

	return false
}

// ClearRecentSuccess invalidates the recent success record for an account.
func (s *RequestQualityProbeService) ClearRecentSuccess(accountID int64, model ...string) {
	if s == nil {
		return
	}
	if len(model) > 0 && model[0] != "" {
		s.recentSuccess.Delete(fmt.Sprintf("%d:%s", accountID, strings.ToLower(strings.TrimSpace(model[0]))))
		return
	}
	prefix := fmt.Sprintf("%d:", accountID)
	s.recentSuccess.Range(func(key, _ any) bool {
		if k, ok := key.(string); ok && strings.HasPrefix(k, prefix) {
			s.recentSuccess.Delete(key)
		}
		return true
	})
}

type probeExecutionResult struct {
	passed bool
	reason string
	err    error
}

// EnsureAccountQuality verifies that an account is capable of serving requestedModel.
// If the account already has recent success within the configured window, it passes
// immediately with zero probe overhead. Otherwise, it triggers an on-demand probe
// before forwarding. If the probe passes, it returns true and caller forwards seamlessly.
// If the probe fails, it returns false and caller fails over to the next candidate account.
func (s *RequestQualityProbeService) EnsureAccountQuality(
	ctx context.Context,
	account *Account,
	requestedModel string,
) (passed bool, reason string, err error) {
	if s == nil || s.settingService == nil || account == nil {
		return true, "", nil
	}

	settings, err := s.settingService.GetRequestQualityProbeSettings(ctx)
	if err != nil {
		return false, "load quality settings failed", err
	}
	if settings == nil || !settings.Enabled || len(settings.RuleIDs) == 0 {
		return true, "", nil
	}

	// 1. Designated models check ("只能指定的大模型才需要测智")
	cleanModel := strings.TrimSpace(requestedModel)
	if IsQualityProbeExcluded(settings, cleanModel) {
		return true, "", nil
	}
	if settings.ModelScope != "all" {
		matched := false
		for _, m := range settings.Models {
			if matchesQualityProbeModel(m, cleanModel) {
				matched = true
				break
			}
		}
		if !matched {
			return true, "", nil
		}
	}

	// 2. Designated accounts / troubled accounts check ("可能出问题的才需要测智")
	scope := strings.ToLower(strings.TrimSpace(settings.AccountScope))
	switch scope {
	case "selected":
		inList := false
		for _, id := range settings.AccountIDs {
			if id == account.ID {
				inList = true
				break
			}
		}
		if !inList && !getExtraBool(account.Extra, "test_on_request_enabled") {
			return true, "", nil
		}
	case "error_or_downgraded":
		isTroubled := account.Status == "error" || getExtraBool(account.Extra, "pelican_downgraded") || account.ErrorMessage != ""
		if !isTroubled {
			// Normal, active accounts without error flag don't need on-request probe in this scope
			return true, "", nil
		}
	default: // "all"
		// Every account supporting the designated models is checked for recent success
	}

	// 3. Recent success check (近期成功有效窗口期内免测直接放行)
	recentMinutes := settings.RecentMinutes
	if recentMinutes <= 0 {
		recentMinutes = 10
	}
	if settings.AllowBusinessSuccess && s.HasRecentSuccess(account.ID, cleanModel, recentMinutes) {
		return true, "", nil
	}

	// 4. No recent success record: execute probe on-demand ("先测智")
	if s.accountTestService == nil {
		return false, "quality tester unavailable", errors.New("quality tester unavailable")
	}

	rules, ruleErr := s.resolveProbeRules(ctx, settings.RuleIDs)
	if ruleErr != nil {
		return false, ruleErr.Error(), ruleErr
	}
	fingerprint := DegradationRulesFingerprint(rules)
	if s.detectionRepo != nil {
		passed, proofErr := s.detectionRepo.HasRecentProof(ctx, account.ID, cleanModel, fingerprint, time.Now().Add(-time.Duration(recentMinutes)*time.Minute))
		if proofErr != nil {
			return false, "read quality proof failed", proofErr
		}
		if passed {
			return true, "", nil
		}
	}
	flightKey := fmt.Sprintf("probe:%d:%s:%s", account.ID, cleanModel, fingerprint)
	pending := s.probeFlight.DoChan(flightKey, func() (any, error) {
		return s.executeProbe(context.WithoutCancel(ctx), account, cleanModel, settings, rules), nil
	})
	var res any
	select {
	case <-ctx.Done():
		return false, ctx.Err().Error(), ctx.Err()
	case completed := <-pending:
		if completed.Err != nil {
			return false, completed.Err.Error(), completed.Err
		}
		res = completed.Val
	}
	result, ok := res.(probeExecutionResult)
	if !ok {
		return true, "", nil
	}
	return result.passed, result.reason, result.err
}

// StartRequestQualityCheck starts at most one probe per matching business
// attempt. A passing probe for the same account, model and question may release
// later attempts inside the configured reuse window (`reuse_seconds`), which
// bounds upstream cost; failures never release a later attempt, and singleflight
// is still bypassed so each window gets its own check. A nil channel means the
// feature is disabled, this attempt is outside scope, or a recent passing probe
// covered it.
func (s *RequestQualityProbeService) StartRequestQualityCheck(ctx context.Context, account *Account, model string) (<-chan error, context.CancelFunc) {
	noop := func() {}
	if s == nil || s.settingService == nil || account == nil {
		return nil, noop
	}
	settings, err := s.GetSettings(ctx)
	result := make(chan error, 1)
	if err != nil {
		result <- err
		return result, noop
	}
	if !settings.Enabled || len(settings.RuleIDs) == 0 || !requestQualityMatches(settings, account, model) {
		return nil, noop
	}
	if s.hasRecentPassingProbe(ctx, account, model, settings) {
		logger.LegacyPrintf("service.request_quality_probe",
			"[RequestQualityProbe] account=%d (%s) model=%s reusing passing probe within %ds",
			account.ID, account.Name, strings.TrimSpace(model), settings.reuseWindowSeconds())
		return nil, noop
	}
	probeCtx, cancel := context.WithTimeout(ctx, time.Duration(settings.TimeoutSeconds)*time.Second)
	go func() {
		if s.accountTestService == nil {
			result <- errors.New("quality tester unavailable")
			return
		}
		ids := settings.RuleIDs
		// Older configurations allowed several rules. Keep the first until the
		// administrator next saves the new single-question configuration.
		if len(ids) > 1 {
			ids = ids[:1]
		}
		rules, err := s.resolveProbeRules(probeCtx, ids)
		if err == nil {
			verdict := s.executeProbe(probeCtx, account, strings.TrimSpace(model), settings, rules)
			err = verdict.err
			if !verdict.passed && err == nil {
				err = errors.New(verdict.reason)
			}
		}
		result <- err
	}()
	return result, cancel
}

// hasRecentPassingProbe reports whether the same account, model and selected
// question passed inside the reuse window. Only a passing latest record counts:
// a newer failure or a changed question invalidates the evidence. Read errors
// fall back to probing instead of releasing an unchecked response.
func (s *RequestQualityProbeService) hasRecentPassingProbe(ctx context.Context, account *Account, model string, settings *RequestQualityProbeSettings) bool {
	if s == nil || s.detectionRepo == nil || account == nil || settings == nil {
		return false
	}
	if IsQualityProbeExcluded(settings, model) {
		return false
	}
	window := settings.reuseWindowSeconds()
	if window <= 0 {
		return false
	}
	ids := settings.RuleIDs
	if len(ids) > 1 {
		ids = ids[:1]
	}
	rules, err := s.resolveProbeRules(ctx, ids)
	if err != nil {
		return false
	}
	passed, err := s.detectionRepo.HasRecentProof(ctx, account.ID, strings.TrimSpace(model),
		DegradationRulesFingerprint(rules), time.Now().Add(-time.Duration(window)*time.Second))
	if err != nil {
		return false
	}
	return passed
}

func requestQualityMatches(settings *RequestQualityProbeSettings, account *Account, model string) bool {
	if IsQualityProbeExcluded(settings, model) {
		return false
	}
	if settings.ModelScope != "all" {
		matched := false
		for _, candidate := range settings.Models {
			if matchesQualityProbeModel(candidate, model) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if enabled, ok := settings.AccountOverrides[account.ID]; ok {
		return enabled
	}
	switch settings.AccountScope {
	case "selected":
		for _, id := range settings.AccountIDs {
			if id == account.ID {
				return true
			}
		}
		return getExtraBool(account.Extra, "test_on_request_enabled")
	case "error_or_downgraded":
		return account.Status == "error" || getExtraBool(account.Extra, "pelican_downgraded") || account.ErrorMessage != ""
	default:
		return true
	}
}

// IsGLMModel reports whether a model identifier belongs to the GLM model family (e.g. glm-5.3, glm-5.3-flash, GLM-5.3).
func IsGLMModel(model string) bool {
	clean := stripModelNamespaceAndProvider(strings.ToLower(strings.TrimSpace(model)))
	return strings.HasPrefix(clean, "glm") || strings.HasPrefix(clean, "chatglm")
}

// IsQualityProbeExcluded reports whether a model is excluded from quality probing.
// GLM* models (such as glm-5.3, glm-5.3-flash) are excluded by policy and never require quality probing.
// In addition, any model matching patterns in settings.ExcludeModels is excluded.
func IsQualityProbeExcluded(settings *RequestQualityProbeSettings, model string) bool {
	clean := stripModelNamespaceAndProvider(strings.ToLower(strings.TrimSpace(model)))
	if clean == "" {
		return false
	}
	if IsGLMModel(clean) {
		return true
	}
	if settings != nil {
		for _, pattern := range settings.ExcludeModels {
			if matchesQualityProbeModel(pattern, model) || matchesQualityProbeModel(pattern, clean) {
				return true
			}
		}
	}
	return false
}

func matchesQualityProbeModel(pattern, model string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	model = strings.ToLower(strings.TrimSpace(model))
	if pattern == "" || model == "" {
		return false
	}

	// 1. Direct candidate matching
	if matchesQualityProbeCandidate(pattern, model) {
		return true
	}

	// 2. Strip namespace and provider prefixes from model:
	// e.g. "global:claude-3-7-sonnet" -> "claude-3-7-sonnet"
	// e.g. "anthropic/claude-3-7-sonnet" -> "claude-3-7-sonnet"
	cleanModel := stripModelNamespaceAndProvider(model)
	if cleanModel != model && matchesQualityProbeCandidate(pattern, cleanModel) {
		return true
	}

	// 3. Strip namespace and provider prefixes from pattern as well:
	// e.g. pattern "anthropic/claude*" with requested model "claude-3-7-sonnet"
	cleanPattern := stripModelNamespaceAndProvider(pattern)
	if cleanPattern != pattern {
		if matchesQualityProbeCandidate(cleanPattern, model) || matchesQualityProbeCandidate(cleanPattern, cleanModel) {
			return true
		}
	}

	return false
}

func stripModelNamespaceAndProvider(s string) string {
	if idx := strings.Index(s, ":"); idx >= 0 {
		s = s[idx+1:]
	}
	if idx := strings.LastIndex(s, "/"); idx >= 0 {
		s = s[idx+1:]
	}
	return s
}

func matchesQualityProbeCandidate(pattern, model string) bool {
	if matchesQualityProbeModelPattern(pattern, model) {
		return true
	}

	// Delimiter normalization between digits:
	// Anthropic models often use "3-7" (claude-3-7-sonnet) while clients/users often use "3.7" (claude-3.7-sonnet).
	// Normalizing both allows "claude-3.7*" to match "claude-3-7-sonnet-20250219" and vice-versa.
	normPattern := normalizeVersionDelimiters(pattern)
	normModel := normalizeVersionDelimiters(model)
	if normPattern != pattern || normModel != model {
		if matchesQualityProbeModelPattern(normPattern, normModel) {
			return true
		}
	}

	return false
}

func normalizeVersionDelimiters(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(runes); i++ {
		if runes[i] == '.' && i > 0 && i+1 < len(runes) && runes[i-1] >= '0' && runes[i-1] <= '9' && runes[i+1] >= '0' && runes[i+1] <= '9' {
			b.WriteRune('-')
		} else {
			b.WriteRune(runes[i])
		}
	}
	return b.String()
}

func matchesQualityProbeModelPattern(pattern, model string) bool {
	if pattern == model {
		return true
	}

	if strings.Contains(pattern, "*") {
		return globMatch(pattern, model)
	}

	// Prefix match: e.g. "gpt-6" matches "gpt-6-sol", "gpt-6.1-sol", "gpt-6-astra", etc.
	// "claude" matches "claude-sonnet-5", "claude-3-7-sonnet", etc.
	if strings.HasPrefix(model, pattern+"-") || strings.HasPrefix(model, pattern+".") {
		return true
	}

	return false
}

func globMatch(pattern, text string) bool {
	if pattern == "*" {
		return text != ""
	}
	if strings.HasSuffix(pattern, "*") && !strings.Contains(pattern[:len(pattern)-1], "*") {
		return strings.HasPrefix(text, pattern[:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "*") && !strings.Contains(pattern[1:], "*") {
		return strings.HasSuffix(text, pattern[1:])
	}
	parts := strings.Split(pattern, "*")
	for i := range parts {
		parts[i] = regexp.QuoteMeta(parts[i])
	}
	matched, err := regexp.MatchString("^"+strings.Join(parts, ".*")+"$", text)
	return err == nil && matched
}

func (s *RequestQualityProbeService) resolveProbeRules(ctx context.Context, ids []int64) ([]*DegradationRule, error) {
	if len(ids) != 1 {
		return nil, errors.New("designate one automatic quality question in account testing")
	}
	return s.resolveSelectedRules(ctx, ids, false)
}

func (s *RequestQualityProbeService) resolveSelectedRules(ctx context.Context, ids []int64, manual bool) ([]*DegradationRule, error) {
	if len(ids) == 0 {
		return BuiltinCapabilityRules(), nil
	}
	if s.ruleRepo == nil {
		return nil, errors.New("rule library unavailable")
	}
	rules := make([]*DegradationRule, 0, len(ids))
	seen := map[int64]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		rule, err := s.ruleRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if !rule.Enabled && !manual {
			return nil, fmt.Errorf("rule %d is disabled", id)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (s *RequestQualityProbeService) executeProbe(ctx context.Context, account *Account, modelID string, settings *RequestQualityProbeSettings, rules []*DegradationRule) probeExecutionResult {
	if IsQualityProbeExcluded(settings, modelID) {
		return probeExecutionResult{passed: true}
	}
	probeCtx, cancel := context.WithTimeout(withDetectionOrigin(ctx, "request", AccountTestModePelican, account.ID, 0, false), time.Duration(settings.TimeoutSeconds)*time.Second)
	defer cancel()

	logger.LegacyPrintf("service.request_quality_probe",
		"[RequestQualityProbe] account=%d (%s) model=%s running request-triggered quality check",
		account.ID, account.Name, modelID)

	testResult, err := s.accountTestService.RunDegradationProbeRules(probeCtx, account.ID, modelID, rules)
	if err != nil {
		logger.LegacyPrintf("service.request_quality_probe",
			"[RequestQualityProbe] account=%d (%s) model=%s probe error: %v",
			account.ID, account.Name, modelID, err)
		return probeExecutionResult{
			passed: false,
			reason: "探测请求异常: " + err.Error(),
			err:    err,
		}
	}

	if testResult == nil {
		return probeExecutionResult{err: errors.New("empty quality result"), reason: "empty quality result"}
	}
	probeObservedAt := testResult.FinishedAt
	if probeObservedAt.IsZero() {
		probeObservedAt = time.Now()
	}
	if testResult.Status != "success" && testResult.UpstreamFailure != nil && (testResult.UpstreamFailure.StatusCode == 402 || testResult.UpstreamFailure.StatusCode == 403) {
		failure := testResult.UpstreamFailure
		reason := GatewayFailureReason("insufficient_balance")
		message := "Upstream account has insufficient balance or a billing issue"
		if failure.StatusCode == 403 {
			reason = "upstream_forbidden"
			message = "Upstream HTTP 403: " + SanitizeUpstreamErrorMessage(ExtractUpstreamErrorMessage([]byte(failure.Body)))
		}
		return probeExecutionResult{passed: false, reason: failure.Error(), err: &UpstreamFailoverError{
			StatusCode: failure.StatusCode, ResponseBody: []byte(failure.Body), Stage: GatewayFailureStageInference,
			Scope: GatewayFailureScopeAccount, Reason: reason, ClientStatusCode: failure.StatusCode,
			ClientMessage: message,
		}}
	}
	if testResult.Status != "success" || testResult.Downgraded {
		reason := testResult.DowngradeReason
		if testResult.ErrorMessage != "" {
			reason = testResult.ErrorMessage
		}
		if reason == "" {
			reason = "探测判定降智或异常"
		}
		logger.LegacyPrintf("service.request_quality_probe",
			"[RequestQualityProbe] account=%d (%s) model=%s probe FAILED status=%s downgraded=%t: %s",
			account.ID, account.Name, modelID, testResult.Status, testResult.Downgraded, reason)

		// Only a completed probe that explicitly reports degradation contributes
		// to the scheduling guard. Transport failures and incomplete tests are
		// intentionally excluded from the counter.
		s.ClearRecentSuccess(account.ID, modelID)
		if testResult.Status == "success" && testResult.Downgraded {
			fingerprint := DegradationRulesFingerprint(rules)
			stateCtx, stateCancel := context.WithTimeout(context.WithoutCancel(probeCtx), 3*time.Second)
			state, stateErr := s.recordQualityDegradation(
				stateCtx,
				account,
				modelID,
				reason,
				fingerprint,
				settings.DowngradeThreshold,
				settings.AutoPauseOnDowngrade,
			)
			stateCancel()
			if stateErr != nil {
				logger.LegacyPrintf("service.request_quality_probe", "persist degradation state account=%d failed: %v", account.ID, stateErr)
			} else if state.Paused {
				logger.LegacyPrintf("service.request_quality_probe", "account=%d paused after %s confirmed degradations", account.ID, state.String())
			}
			s.persistAccountDowngradeFlag(probeCtx, account, true)
		}

		probeErr := fmt.Errorf("quality probe failed: %s", reason)
		var verdictErr error = probeErr
		if testResult.Status == "success" && testResult.Downgraded {
			verdictErr = &rotationnotice.DowngradeError{Cause: probeErr}
		}
		return probeExecutionResult{
			passed: false,
			reason: reason,
			err:    verdictErr,
		}
	}

	logger.LegacyPrintf("service.request_quality_probe",
		"[RequestQualityProbe] account=%d (%s) model=%s probe PASSED; recording success and proceeding with request",
		account.ID, account.Name, modelID)

	// A clean result resets only the consecutive counter. It never silently
	// re-enables an account that was already paused by the quality guard.
	stateCtx, stateCancel := context.WithTimeout(context.WithoutCancel(probeCtx), 3*time.Second)
	resetErr := s.resetQualityDegradation(stateCtx, account, DegradationRulesFingerprint(rules), probeObservedAt)
	stateCancel()
	if resetErr != nil {
		logger.LegacyPrintf("service.request_quality_probe", "reset degradation state account=%d failed: %v", account.ID, resetErr)
	}

	// If account had downgrade flag, clear it
	if getExtraBool(account.Extra, "pelican_downgraded") {
		s.persistAccountDowngradeFlag(probeCtx, account, false)
	}

	return probeExecutionResult{passed: true}
}

func (s *RequestQualityProbeService) persistAccountDowngradeFlag(ctx context.Context, account *Account, downgraded bool) {
	if s.accountRepo == nil || account == nil {
		return
	}
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if err := s.accountRepo.UpdateExtra(persistCtx, account.ID, map[string]any{"pelican_downgraded": downgraded}); err != nil {
		logger.LegacyPrintf("service.request_quality_probe", "persist downgrade account=%d failed: %v", account.ID, err)
	}
}

// GetSettings returns the current request quality probe settings.
func (s *RequestQualityProbeService) GetSettings(ctx context.Context) (*RequestQualityProbeSettings, error) {
	if s == nil || s.settingService == nil {
		return DefaultRequestQualityProbeSettings(), nil
	}
	return s.settingService.GetRequestQualityProbeSettings(ctx)
}

// UpdateSettings persists new request quality probe settings.
func (s *RequestQualityProbeService) UpdateSettings(ctx context.Context, settings *RequestQualityProbeSettings) error {
	if s == nil || s.settingService == nil {
		return errors.New("request quality probe service unavailable")
	}
	return s.settingService.UpdateRequestQualityProbeSettings(ctx, settings)
}

func (s *RequestQualityProbeService) ListDetectionRecords(ctx context.Context, f AccountDetectionFilter) ([]*AccountDetectionRecord, error) {
	if s.detectionRepo == nil {
		return []*AccountDetectionRecord{}, nil
	}
	return s.detectionRepo.List(ctx, f)
}
func (s *RequestQualityProbeService) RunManualRules(ctx context.Context, accountID int64, model string, ids []int64) (*ScheduledTestResult, error) {
	_, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	// An explicit manual selection may use any saved question. Enabled controls
	// automatic probes only; the tester dispatches using the account's platform.
	rules, err := s.resolveSelectedRules(ctx, ids, true)
	if err != nil {
		return nil, err
	}
	if s.accountTestService == nil {
		return nil, errors.New("quality tester unavailable")
	}
	return s.accountTestService.RunDegradationProbeRules(WithAccountDetectionSource(ctx, "manual"), accountID, model, rules)
}

func (s *RequestQualityProbeService) LatestAccountSummaries(ctx context.Context, ids []int64) ([]qualityevidence.AccountSummary, error) {
	reader, ok := s.detectionRepo.(interface {
		LatestAccountSummaries(context.Context, []int64) ([]qualityevidence.AccountSummary, error)
	})
	if !ok {
		return nil, fmt.Errorf("detection summary repository unavailable")
	}
	return reader.LatestAccountSummaries(ctx, ids)
}

func (s *RequestQualityProbeService) DegradationCounts(ctx context.Context, ids []int64, since *time.Time) (map[int64]int64, error) {
	if s == nil || s.detectionRepo == nil {
		return map[int64]int64{}, nil
	}
	return s.detectionRepo.DegradationCounts(ctx, ids, since)
}

// RecentSuccessfulUse is advisory only; it never bypasses the quality gate.
func (s *RequestQualityProbeService) RecentSuccessfulUse(ctx context.Context, attempts []rotationnotice.Attempt, before time.Time) (bool, error) {
	if s == nil {
		return false, nil
	}
	reader, ok := s.detectionRepo.(interface {
		RecentSuccessfulUse(context.Context, []rotationnotice.Attempt, time.Time) (bool, error)
	})
	if !ok {
		return false, nil
	}
	return reader.RecentSuccessfulUse(ctx, attempts, before)
}
