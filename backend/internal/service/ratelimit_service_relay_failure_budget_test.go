//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type relayFailureBudgetCacheCall struct {
	accountID int64
	at        time.Time
	outcome   RelayFailureBudgetOutcome
	policy    RelayFailureBudgetPolicy
}

type relayFailureBudgetCacheStub struct {
	calls     []relayFailureBudgetCacheCall
	decisions []RelayFailureBudgetDecision
	err       error
}

func (s *relayFailureBudgetCacheStub) RecordRelayFailureBudgetEvent(
	_ context.Context,
	accountID int64,
	at time.Time,
	outcome RelayFailureBudgetOutcome,
	policy RelayFailureBudgetPolicy,
) (RelayFailureBudgetDecision, error) {
	s.calls = append(s.calls, relayFailureBudgetCacheCall{
		accountID: accountID,
		at:        at,
		outcome:   outcome,
		policy:    policy,
	})
	if s.err != nil {
		return RelayFailureBudgetDecision{}, s.err
	}
	if len(s.decisions) == 0 {
		return RelayFailureBudgetDecision{}, nil
	}
	decision := s.decisions[0]
	s.decisions = s.decisions[1:]
	return decision, nil
}

type relayFailureBudgetAccountRepo struct {
	mockAccountRepoForGemini
	setErrorCalls int
	tempCalls     int
	lastErrorID   int64
	lastTempID    int64
	lastTempUntil time.Time
	lastReason    string
}

func (r *relayFailureBudgetAccountRepo) SetError(_ context.Context, id int64, _ string) error {
	r.setErrorCalls++
	r.lastErrorID = id
	return nil
}

func (r *relayFailureBudgetAccountRepo) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	r.tempCalls++
	r.lastTempID = id
	r.lastTempUntil = until
	r.lastReason = reason
	return nil
}

type relayFailureBudgetRuntimeBlocker struct {
	accountID int64
	until     time.Time
	reason    string
}

func (b *relayFailureBudgetRuntimeBlocker) BlockAccountScheduling(account *Account, until time.Time, reason string) {
	b.accountID = account.ID
	b.until = until
	b.reason = reason
}

func (b *relayFailureBudgetRuntimeBlocker) ClearAccountSchedulingBlock(int64) {}

func newRelayFailureBudgetAccount(baseURL string) *Account {
	return &Account{
		ID:          183,
		Name:        "relay-flaky",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"api_key":  "relay-key",
			"base_url": baseURL,
		},
	}
}

func TestAccountOpenAIRelayFailureBudgetPolicyDefaultsAndScope(t *testing.T) {
	account := newRelayFailureBudgetAccount("https://relay.example.com/v1")

	policy, ok := account.OpenAIRelayFailureBudgetPolicy()

	require.True(t, ok)
	require.Equal(t, 10*time.Minute, policy.Window)
	require.Equal(t, 30, policy.FailureThresholdPercent)
	require.Equal(t, 10, policy.MinRequests)
	require.Equal(t, 5, policy.ConsecutiveFailures)
	require.Equal(t, 2*time.Minute, policy.Cooldown)

	tests := []struct {
		name   string
		mutate func(*Account)
	}{
		{name: "official endpoint", mutate: func(a *Account) { a.Credentials["base_url"] = "https://api.openai.com/v1" }},
		{name: "official endpoint case insensitive", mutate: func(a *Account) { a.Credentials["base_url"] = "https://API.OPENAI.COM/v1" }},
		{name: "missing base url", mutate: func(a *Account) { delete(a.Credentials, "base_url") }},
		{name: "oauth", mutate: func(a *Account) { a.Type = AccountTypeOAuth }},
		{name: "other platform", mutate: func(a *Account) { a.Platform = PlatformAnthropic }},
		{name: "pool mode", mutate: func(a *Account) { a.Credentials["pool_mode"] = true }},
		{name: "custom error policy", mutate: func(a *Account) { a.Credentials["custom_error_codes_enabled"] = true }},
		{name: "explicitly disabled", mutate: func(a *Account) { a.Credentials["relay_failure_budget_enabled"] = false }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := newRelayFailureBudgetAccount("https://relay.example.com/v1")
			tt.mutate(candidate)
			_, enabled := candidate.OpenAIRelayFailureBudgetPolicy()
			require.False(t, enabled)
		})
	}
}

func TestAccountOpenAIRelayFailureBudgetPolicyReadsConfiguredValues(t *testing.T) {
	account := newRelayFailureBudgetAccount("https://relay.example.com/v1")
	account.Credentials["relay_failure_budget_enabled"] = true
	account.Credentials["relay_failure_budget_window_minutes"] = float64(20)
	account.Credentials["relay_failure_budget_failure_threshold_percent"] = float64(15)
	account.Credentials["relay_failure_budget_min_requests"] = float64(25)
	account.Credentials["relay_failure_budget_consecutive_failures"] = float64(7)
	account.Credentials["relay_failure_budget_cooldown_minutes"] = float64(4)

	policy, ok := account.OpenAIRelayFailureBudgetPolicy()

	require.True(t, ok)
	require.Equal(t, 20*time.Minute, policy.Window)
	require.Equal(t, 15, policy.FailureThresholdPercent)
	require.Equal(t, 25, policy.MinRequests)
	require.Equal(t, 7, policy.ConsecutiveFailures)
	require.Equal(t, 4*time.Minute, policy.Cooldown)
}

func TestRateLimitServiceRelayFailureBudgetClassifiesOnlyManagedFailures(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       []byte
		want       bool
	}{
		{name: "request timeout", statusCode: http.StatusRequestTimeout, want: true},
		{name: "internal server error", statusCode: http.StatusInternalServerError, want: true},
		{name: "bad gateway", statusCode: http.StatusBadGateway, want: true},
		{name: "service unavailable", statusCode: http.StatusServiceUnavailable, want: true},
		{name: "gateway timeout", statusCode: http.StatusGatewayTimeout, want: true},
		{name: "cloudflare 520", statusCode: 520, want: true},
		{name: "overload 529 separate policy", statusCode: 529, want: false},
		{name: "quota 429 separate policy", statusCode: http.StatusTooManyRequests, want: false},
		{name: "client request 400", statusCode: http.StatusBadRequest, want: false},
		{name: "model not found", statusCode: http.StatusNotFound, body: []byte(`{"error":{"message":"model not found"}}`), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isOpenAIRelayFailureBudgetHTTPFailure(tt.statusCode, tt.body))
		})
	}
}

func TestRateLimitServiceHandleUpstreamErrorRelayInternal401RecordsFailureWithoutPermanentError(t *testing.T) {
	repo := &relayFailureBudgetAccountRepo{}
	cache := &relayFailureBudgetCacheStub{}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc.SetRelayFailureBudgetCache(cache)
	account := newRelayFailureBudgetAccount("https://relay.example.com/v1")
	body := []byte(`{"error":{"message":"Your authentication token has been invalidated. Please try signing in again.","type":"invalid_request_error"},"status":401}`)

	shouldDisable := svc.HandleUpstreamError(context.Background(), account, http.StatusUnauthorized, http.Header{}, body)

	require.False(t, shouldDisable)
	require.Zero(t, repo.setErrorCalls)
	require.Zero(t, repo.tempCalls)
	require.Len(t, cache.calls, 1)
	require.Equal(t, RelayFailureBudgetFailure, cache.calls[0].outcome)
}

func TestRateLimitServiceHandleUpstreamErrorRelayBudgetTripTemporarilyBlocksWithoutErrorStatus(t *testing.T) {
	until := time.Now().Add(2 * time.Minute).Round(time.Second)
	repo := &relayFailureBudgetAccountRepo{}
	cache := &relayFailureBudgetCacheStub{decisions: []RelayFailureBudgetDecision{{
		WindowRequests:      10,
		WindowFailures:      3,
		ConsecutiveFailures: 3,
		Tripped:             true,
		CooldownUntil:       until,
	}}}
	blocker := &relayFailureBudgetRuntimeBlocker{}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc.SetRelayFailureBudgetCache(cache)
	svc.SetAccountRuntimeBlocker(blocker)
	account := newRelayFailureBudgetAccount("https://relay.example.com/v1")

	shouldDisable := svc.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusServiceUnavailable,
		http.Header{},
		[]byte(`{"error":{"message":"temporary relay failure"}}`),
	)

	require.False(t, shouldDisable, "ratio policy must not trigger the indefinite runtime-block fast path")
	require.Zero(t, repo.setErrorCalls)
	require.Equal(t, StatusActive, account.Status)
	require.Equal(t, 1, repo.tempCalls)
	require.Equal(t, account.ID, repo.lastTempID)
	require.Equal(t, until, repo.lastTempUntil)
	require.Contains(t, repo.lastReason, "3/10")
	require.Equal(t, account.ID, blocker.accountID)
	require.Equal(t, until, blocker.until)
	require.Equal(t, "openai_relay_failure_budget", blocker.reason)
}

func TestRateLimitServiceHandleUpstreamErrorRelayHardAuthErrorTakesPrecedence(t *testing.T) {
	repo := &relayFailureBudgetAccountRepo{}
	cache := &relayFailureBudgetCacheStub{}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc.SetRelayFailureBudgetCache(cache)
	account := newRelayFailureBudgetAccount("https://relay.example.com/v1")
	body := []byte(`{"error":{"message":"Your authentication token has been invalidated.","code":"invalid_api_key"}}`)

	shouldDisable := svc.HandleUpstreamError(context.Background(), account, http.StatusUnauthorized, http.Header{}, body)

	require.True(t, shouldDisable)
	require.Equal(t, 1, repo.setErrorCalls)
	require.Empty(t, cache.calls)
}

func TestRateLimitServiceHandleUpstreamErrorOfficialOpenAITokenInvalidatedRemainsHardError(t *testing.T) {
	repo := &relayFailureBudgetAccountRepo{}
	cache := &relayFailureBudgetCacheStub{}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc.SetRelayFailureBudgetCache(cache)
	account := newRelayFailureBudgetAccount("https://api.openai.com/v1")
	body := []byte(`{"error":{"message":"token invalidated","code":"token_invalidated"}}`)

	shouldDisable := svc.HandleUpstreamError(context.Background(), account, http.StatusUnauthorized, http.Header{}, body)

	require.True(t, shouldDisable)
	require.Equal(t, 1, repo.setErrorCalls)
	require.Empty(t, cache.calls)
}

func TestRateLimitServiceRecordsRelaySuccessAndIgnoresCacheFailure(t *testing.T) {
	cache := &relayFailureBudgetCacheStub{err: errors.New("redis unavailable")}
	svc := NewRateLimitService(nil, nil, &config.Config{}, nil, nil)
	svc.SetRelayFailureBudgetCache(cache)
	account := newRelayFailureBudgetAccount("https://relay.example.com/v1")

	svc.recordOpenAIRelaySuccess(context.Background(), account)

	require.Len(t, cache.calls, 1)
	require.Equal(t, RelayFailureBudgetSuccess, cache.calls[0].outcome)
}
