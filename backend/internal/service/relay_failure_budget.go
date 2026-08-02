package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRelayFailureBudgetWindowMinutes       = 10
	defaultRelayFailureBudgetThresholdPercent    = 30
	defaultRelayFailureBudgetMinRequests         = 10
	defaultRelayFailureBudgetConsecutiveFailures = 5
	defaultRelayFailureBudgetCooldownMinutes     = 2
	maxRelayFailureBudgetWindowMinutes           = 1440
	maxRelayFailureBudgetMinRequests             = 10000
	maxRelayFailureBudgetConsecutiveFailures     = 1000
	maxRelayFailureBudgetCooldownMinutes         = 1440
	relayFailureBudgetRuntimeBlockReason         = "openai_relay_failure_budget"
	relayFailureBudgetEnabledCredentialKey       = "relay_failure_budget_enabled"
	relayFailureBudgetWindowCredentialKey        = "relay_failure_budget_window_minutes"
	relayFailureBudgetThresholdCredentialKey     = "relay_failure_budget_failure_threshold_percent"
	relayFailureBudgetMinRequestsCredentialKey   = "relay_failure_budget_min_requests"
	relayFailureBudgetConsecutiveCredentialKey   = "relay_failure_budget_consecutive_failures"
	relayFailureBudgetCooldownCredentialKey      = "relay_failure_budget_cooldown_minutes"
)

type RelayFailureBudgetPolicy struct {
	Window                  time.Duration
	FailureThresholdPercent int
	MinRequests             int
	ConsecutiveFailures     int
	Cooldown                time.Duration
}

// OpenAIRelayFailureBudgetSettings is the account-facing persisted policy.
// Durations use minutes because that is the precision supported by account
// credentials and the admin UI.
type OpenAIRelayFailureBudgetSettings struct {
	Enabled                 bool
	WindowMinutes           int
	FailureThresholdPercent int
	MinRequests             int
	ConsecutiveFailures     int
	CooldownMinutes         int
}

type RelayFailureBudgetOutcome string

const (
	RelayFailureBudgetSuccess RelayFailureBudgetOutcome = "success"
	RelayFailureBudgetFailure RelayFailureBudgetOutcome = "failure"
)

type RelayFailureBudgetDecision struct {
	WindowRequests      int
	WindowFailures      int
	ConsecutiveFailures int
	Tripped             bool
	CooldownUntil       time.Time
}

type RelayFailureBudgetCache interface {
	RecordRelayFailureBudgetEvent(
		ctx context.Context,
		accountID int64,
		at time.Time,
		outcome RelayFailureBudgetOutcome,
		policy RelayFailureBudgetPolicy,
	) (RelayFailureBudgetDecision, error)
}

func (a *Account) OpenAIRelayFailureBudgetPolicy() (RelayFailureBudgetPolicy, bool) {
	settings, supported := a.OpenAIRelayFailureBudgetSettings()
	if !supported || !settings.Enabled {
		return RelayFailureBudgetPolicy{}, false
	}

	return RelayFailureBudgetPolicy{
		Window:                  time.Duration(settings.WindowMinutes) * time.Minute,
		FailureThresholdPercent: settings.FailureThresholdPercent,
		MinRequests:             settings.MinRequests,
		ConsecutiveFailures:     settings.ConsecutiveFailures,
		Cooldown:                time.Duration(settings.CooldownMinutes) * time.Minute,
	}, true
}

// SupportsOpenAIRelayFailureBudget reports whether the relay policy is valid
// for this account, independently of whether the operator has disabled it.
func (a *Account) SupportsOpenAIRelayFailureBudget() bool {
	return isOpenAICustomRelayAPIKey(a) && !a.IsPoolMode() && !a.IsCustomErrorCodesEnabled()
}

// OpenAIRelayFailureBudgetSettings returns normalized persisted values. The
// second result distinguishes unsupported accounts from an explicit opt-out.
func (a *Account) OpenAIRelayFailureBudgetSettings() (OpenAIRelayFailureBudgetSettings, bool) {
	settings := OpenAIRelayFailureBudgetSettings{
		Enabled:                 true,
		WindowMinutes:           defaultRelayFailureBudgetWindowMinutes,
		FailureThresholdPercent: defaultRelayFailureBudgetThresholdPercent,
		MinRequests:             defaultRelayFailureBudgetMinRequests,
		ConsecutiveFailures:     defaultRelayFailureBudgetConsecutiveFailures,
		CooldownMinutes:         defaultRelayFailureBudgetCooldownMinutes,
	}
	if a == nil || !a.SupportsOpenAIRelayFailureBudget() {
		return settings, false
	}
	if enabled, exists := relayFailureBudgetCredentialBool(a.Credentials, relayFailureBudgetEnabledCredentialKey); exists {
		settings.Enabled = enabled
	}
	settings.WindowMinutes = relayFailureBudgetCredentialInt(
		a.Credentials,
		relayFailureBudgetWindowCredentialKey,
		defaultRelayFailureBudgetWindowMinutes,
		1,
		maxRelayFailureBudgetWindowMinutes,
	)
	settings.FailureThresholdPercent = relayFailureBudgetCredentialInt(
		a.Credentials,
		relayFailureBudgetThresholdCredentialKey,
		defaultRelayFailureBudgetThresholdPercent,
		1,
		100,
	)
	settings.MinRequests = relayFailureBudgetCredentialInt(
		a.Credentials,
		relayFailureBudgetMinRequestsCredentialKey,
		defaultRelayFailureBudgetMinRequests,
		1,
		maxRelayFailureBudgetMinRequests,
	)
	settings.ConsecutiveFailures = relayFailureBudgetCredentialInt(
		a.Credentials,
		relayFailureBudgetConsecutiveCredentialKey,
		defaultRelayFailureBudgetConsecutiveFailures,
		1,
		maxRelayFailureBudgetConsecutiveFailures,
	)
	settings.CooldownMinutes = relayFailureBudgetCredentialInt(
		a.Credentials,
		relayFailureBudgetCooldownCredentialKey,
		defaultRelayFailureBudgetCooldownMinutes,
		1,
		maxRelayFailureBudgetCooldownMinutes,
	)
	return settings, true
}

// ApplyOpenAIRelayFailureBudgetSettings returns a shallow credential copy with
// only the relay-policy keys changed. Other credentials remain untouched.
func ApplyOpenAIRelayFailureBudgetSettings(credentials map[string]any, settings OpenAIRelayFailureBudgetSettings) (map[string]any, error) {
	updated := make(map[string]any, len(credentials)+6)
	for key, value := range credentials {
		updated[key] = value
	}
	updated[relayFailureBudgetEnabledCredentialKey] = settings.Enabled
	if !settings.Enabled {
		delete(updated, relayFailureBudgetWindowCredentialKey)
		delete(updated, relayFailureBudgetThresholdCredentialKey)
		delete(updated, relayFailureBudgetMinRequestsCredentialKey)
		delete(updated, relayFailureBudgetConsecutiveCredentialKey)
		delete(updated, relayFailureBudgetCooldownCredentialKey)
		return updated, nil
	}

	if settings.WindowMinutes < 1 || settings.WindowMinutes > maxRelayFailureBudgetWindowMinutes {
		return nil, fmt.Errorf("relay failure budget window_minutes must be between 1 and %d", maxRelayFailureBudgetWindowMinutes)
	}
	if settings.FailureThresholdPercent < 1 || settings.FailureThresholdPercent > 100 {
		return nil, fmt.Errorf("relay failure budget failure_threshold_percent must be between 1 and 100")
	}
	if settings.MinRequests < 1 || settings.MinRequests > maxRelayFailureBudgetMinRequests {
		return nil, fmt.Errorf("relay failure budget min_requests must be between 1 and %d", maxRelayFailureBudgetMinRequests)
	}
	if settings.ConsecutiveFailures < 1 || settings.ConsecutiveFailures > maxRelayFailureBudgetConsecutiveFailures {
		return nil, fmt.Errorf("relay failure budget consecutive_failures must be between 1 and %d", maxRelayFailureBudgetConsecutiveFailures)
	}
	if settings.CooldownMinutes < 1 || settings.CooldownMinutes > maxRelayFailureBudgetCooldownMinutes {
		return nil, fmt.Errorf("relay failure budget cooldown_minutes must be between 1 and %d", maxRelayFailureBudgetCooldownMinutes)
	}

	updated[relayFailureBudgetWindowCredentialKey] = settings.WindowMinutes
	updated[relayFailureBudgetThresholdCredentialKey] = settings.FailureThresholdPercent
	updated[relayFailureBudgetMinRequestsCredentialKey] = settings.MinRequests
	updated[relayFailureBudgetConsecutiveCredentialKey] = settings.ConsecutiveFailures
	updated[relayFailureBudgetCooldownCredentialKey] = settings.CooldownMinutes
	return updated, nil
}

func isOpenAICustomRelayAPIKey(account *Account) bool {
	if account == nil || account.Platform != PlatformOpenAI || account.Type != AccountTypeAPIKey {
		return false
	}
	return isCustomOpenAIRelayBaseURL(account.GetCredential("base_url"))
}

func isCustomOpenAIRelayBaseURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	return host != "" && host != "api.openai.com"
}

func relayFailureBudgetCredentialBool(credentials map[string]any, key string) (bool, bool) {
	if credentials == nil {
		return false, false
	}
	raw, exists := credentials[key]
	if !exists {
		return false, false
	}
	enabled, ok := raw.(bool)
	if !ok {
		return false, false
	}
	return enabled, true
}

func relayFailureBudgetCredentialInt(credentials map[string]any, key string, fallback, minValue, maxValue int) int {
	if credentials == nil {
		return fallback
	}
	raw, ok := credentials[key]
	if !ok || raw == nil {
		return fallback
	}

	var value int64
	switch typed := raw.(type) {
	case int:
		value = int64(typed)
	case int64:
		value = typed
	case float64:
		value = int64(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return fallback
		}
		value = parsed
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return fallback
		}
		value = parsed
	default:
		return fallback
	}
	if value < int64(minValue) || value > int64(maxValue) {
		return fallback
	}
	return int(value)
}

func isOpenAIRelayFailureBudgetHTTPFailure(statusCode int, responseBody []byte) bool {
	if isModelNotFoundError(statusCode, responseBody) ||
		isOpenAICodexPlanGatedModelError(statusCode, responseBody) ||
		isOpenAIModelNotAllowed403Error(statusCode, responseBody) ||
		isAccountTestUnsupportedModelError(string(responseBody)) {
		return false
	}
	if statusCode == http.StatusRequestTimeout {
		return true
	}
	return statusCode >= 500 && statusCode <= 599 && statusCode != 529
}

func (s *RateLimitService) recordOpenAIRelayFailure(ctx context.Context, account *Account, failureKind string) {
	s.recordOpenAIRelayBudgetEvent(ctx, account, RelayFailureBudgetFailure, failureKind)
}

func (s *RateLimitService) recordOpenAIRelaySuccess(ctx context.Context, account *Account) {
	s.recordOpenAIRelayBudgetEvent(ctx, account, RelayFailureBudgetSuccess, "success")
}

func (s *RateLimitService) recordOpenAIRelayBudgetEvent(
	ctx context.Context,
	account *Account,
	outcome RelayFailureBudgetOutcome,
	failureKind string,
) {
	if s == nil || s.relayFailureBudgetCache == nil || account == nil {
		return
	}
	policy, enabled := account.OpenAIRelayFailureBudgetPolicy()
	if !enabled {
		return
	}

	now := time.Now()
	decision, err := s.relayFailureBudgetCache.RecordRelayFailureBudgetEvent(ctx, account.ID, now, outcome, policy)
	if err != nil {
		slog.Warn("openai_relay_failure_budget_record_failed", "account_id", account.ID, "outcome", outcome, "error", err)
		return
	}
	if outcome == RelayFailureBudgetSuccess {
		return
	}

	slog.Warn(
		"openai_relay_failure_budget_failure_recorded",
		"account_id", account.ID,
		"failure_kind", failureKind,
		"window_requests", decision.WindowRequests,
		"window_failures", decision.WindowFailures,
		"consecutive_failures", decision.ConsecutiveFailures,
		"tripped", decision.Tripped,
	)
	if !decision.Tripped {
		return
	}

	until := decision.CooldownUntil
	if !until.After(now) {
		until = now.Add(policy.Cooldown)
	}
	reason := fmt.Sprintf(
		"OpenAI relay failure budget cooldown: failures=%d/%d, consecutive=%d, threshold=%d%%",
		decision.WindowFailures,
		decision.WindowRequests,
		decision.ConsecutiveFailures,
		policy.FailureThresholdPercent,
	)
	s.notifyAccountSchedulingBlocked(account, until, relayFailureBudgetRuntimeBlockReason)
	if s.accountRepo != nil {
		if err := s.accountRepo.SetTempUnschedulable(ctx, account.ID, until, reason); err != nil {
			slog.Warn("openai_relay_failure_budget_persist_failed", "account_id", account.ID, "until", until, "error", err)
		}
	}
}
