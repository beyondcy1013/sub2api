package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/require"
)

type schedulingLivenessTester struct {
	mu      sync.Mutex
	results map[int64]*ScheduledTestResult
	calls   []int64
	models  map[int64]string
}

type schedulingLivenessRecoverer struct {
	mu    sync.Mutex
	calls []int64
	err   error
}

func (r *schedulingLivenessRecoverer) RecoverAccountAfterSuccessfulTest(_ context.Context, accountID int64) (*SuccessfulTestRecoveryResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, accountID)
	if r.err != nil {
		return nil, r.err
	}
	return &SuccessfulTestRecoveryResult{ClearedError: true}, nil
}

func TestSchedulingLivenessNextScheduledScanAtUsesRunnerTick(t *testing.T) {
	now := time.Date(2026, 7, 22, 3, 0, 15, 0, time.UTC)
	schedule, err := superPriorityCronParser.Parse("@every 1m")
	require.NoError(t, err)
	entry := cron.Entry{ID: 1, Next: now.Add(15 * time.Second), Schedule: schedule}

	next := schedulingLivenessNextScheduledScanAt(now.Add(2*time.Minute+5*time.Second), entry, now)
	require.Equal(t, now.Add(2*time.Minute+15*time.Second), next)

	due := schedulingLivenessNextScheduledScanAt(now.Add(-time.Second), entry, now)
	require.Equal(t, now.Add(15*time.Second), due)
}

func (t *schedulingLivenessTester) RunTestBackground(_ context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls = append(t.calls, accountID)
	if t.models == nil {
		t.models = make(map[int64]string)
	}
	t.models[accountID] = modelID
	return t.results[accountID], nil
}

func TestNextSchedulingLivenessFailureBecomesSuspectThenDeadAndCanRecover(t *testing.T) {
	now := time.Date(2026, 7, 22, 3, 0, 0, 0, time.UTC)
	first := nextSchedulingLiveness(nil, now, now.Add(5*time.Minute), false, "timeout", 2)
	require.Equal(t, SchedulingLivenessStatusSuspect, first.Status)
	require.Equal(t, 1, first.FailureCount)

	second := nextSchedulingLiveness(first, now.Add(time.Minute), now.Add(6*time.Minute), false, "timeout", 2)
	require.Equal(t, SchedulingLivenessStatusDead, second.Status)
	require.Equal(t, 2, second.FailureCount)

	recovered := nextSchedulingLiveness(second, now.Add(2*time.Minute), now.Add(7*time.Minute), true, "", 2)
	require.Equal(t, SchedulingLivenessStatusAlive, recovered.Status)
	require.Zero(t, recovered.FailureCount)
	require.NotNil(t, recovered.LastSuccessAt)
}

func TestSchedulingLivenessUnknownAndSuspectRemainEligibleFallbacks(t *testing.T) {
	now := time.Now()
	require.False(t, accountSchedulingLivenessDead(&Account{}))
	require.False(t, accountSchedulingLivenessDead(&Account{Extra: map[string]any{
		SchedulingLivenessExtraKey: map[string]any{"status": SchedulingLivenessStatusSuspect},
	}}))
	require.False(t, accountSchedulingLivenessDeadAt(&Account{Extra: map[string]any{
		SchedulingLivenessExtraKey: map[string]any{"status": SchedulingLivenessStatusDead},
	}}, now))
	require.True(t, accountSchedulingLivenessDeadAt(&Account{Extra: map[string]any{
		SchedulingLivenessExtraKey: map[string]any{
			"status":      SchedulingLivenessStatusDead,
			"fresh_until": now.Add(time.Minute),
		},
	}}, now))
	require.False(t, accountSchedulingLivenessDeadAt(&Account{Extra: map[string]any{
		SchedulingLivenessExtraKey: map[string]any{
			"status":      SchedulingLivenessStatusDead,
			"fresh_until": now.Add(-time.Minute),
		},
	}}, now))
}

func TestSchedulingLivenessRunnerSkipsHealthyAndManuallyPausedAccounts(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{
		makeSuperPriorityTestAccount(1, true, true),
		makeSuperPriorityTestAccount(2, false, true),
		makeSuperPriorityTestAccount(3, false, false),
	}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{
		1: {Status: "success"},
		2: {Status: "failed", ErrorMessage: "timeout"},
		3: {Status: "success"},
	}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy:     AccountSchedulingStrategyLowestCost,
		FailureThreshold: 2,
		CheckInterval:    "@every 5m",
	}})
	recoverer := &schedulingLivenessRecoverer{}
	runner := NewSuperPriorityRunner(state, tester, repo, recoverer)

	runner.RunOnce(context.Background())

	require.Empty(t, tester.calls)
	require.Empty(t, recoverer.calls)
	require.Empty(t, repo.livenessWrites)
}

func TestSchedulingLivenessRunnerIgnoresLegacyIncludePausedSetting(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{makeSuperPriorityTestAccount(1, false, false)}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{
		1: {Status: "success"},
	}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost, CheckInterval: "@every 5m", LivenessIncludeUnschedulable: true,
	}})
	recoverer := &schedulingLivenessRecoverer{}
	runner := NewSuperPriorityRunner(state, tester, repo, recoverer)

	result, err := runner.RefreshNow(context.Background())

	require.NoError(t, err)
	require.Empty(t, tester.calls)
	require.Empty(t, recoverer.calls)
	require.Zero(t, result.Checked)
	status := runner.RuntimeStatus()
	require.True(t, status.Enabled)
	require.False(t, status.Running)
	require.NotNil(t, status.LastRun)
	require.Equal(t, "manual", status.LastRun.Trigger)
	require.Equal(t, result, status.LastRun.Result)
}

func TestSchedulingLivenessRunnerOnlyProbesAPIKeyAccounts(t *testing.T) {
	apiKey := makeSuperPriorityTestAccount(1, false, false)
	apiKey.Status = StatusError
	oauth := makeSuperPriorityTestAccount(2, false, true)
	oauth.Type = AccountTypeOAuth
	oauth.Status = StatusError
	setupToken := makeSuperPriorityTestAccount(3, false, true)
	setupToken.Type = AccountTypeSetupToken
	setupToken.Status = StatusError
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{apiKey, oauth, setupToken}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{
		1: {Status: "success"},
		2: {Status: "success"},
		3: {Status: "success"},
	}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost, LivenessIncludeUnschedulable: true,
	}})

	recoverer := &schedulingLivenessRecoverer{}
	result, err := NewSuperPriorityRunner(state, tester, repo, recoverer).RefreshNow(context.Background())

	require.NoError(t, err)
	require.Equal(t, []int64{1}, tester.calls)
	require.Equal(t, []int64{1}, recoverer.calls)
	require.True(t, repo.schedulable[1])
	require.Equal(t, SchedulingRefreshResult{Checked: 1, Succeeded: 1}, result)
	require.NotContains(t, repo.livenessWrites, int64(2))
	require.NotContains(t, repo.livenessWrites, int64(3))
}

func TestSchedulingLivenessRunnerUsesAnAccountSupportedModelForOpenAI(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{
		{
			ID:          1,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusError,
			Schedulable: false,
			Credentials: map[string]any{"model_mapping": map[string]any{
				"gpt-5.6-sol": "gpt-5.6-sol",
			}},
		},
		{
			ID:          2,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusError,
			Schedulable: false,
			Credentials: map[string]any{"model_mapping": map[string]any{
				"gpt-5.6-terra": "gpt-5.6-terra",
			}},
		},
		{ID: 3, Platform: PlatformGrok, Type: AccountTypeAPIKey, Status: StatusError, Schedulable: false},
		{ID: 4, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusError, Schedulable: false},
		{ID: 5, Platform: PlatformGemini, Type: AccountTypeAPIKey, Status: StatusError, Schedulable: false},
	}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{
		1: {Status: "success"},
		2: {Status: "success"},
		3: {Status: "success"},
		4: {Status: "success"},
		5: {Status: "success"},
	}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost,
		TestModelID:  "gpt-5.6-sol",
	}})

	NewSuperPriorityRunner(state, tester, repo, &schedulingLivenessRecoverer{}).RunOnce(context.Background())

	require.Equal(t, "gpt-5.6-sol", tester.models[1])
	require.Equal(t, "gpt-5.6-terra", tester.models[2])
	require.Empty(t, tester.models[3])
	require.Empty(t, tester.models[4])
	require.Empty(t, tester.models[5])
}

func TestSchedulingLivenessPrefersAccountModelBeforeConfiguredModel(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Credentials: map[string]any{"model_mapping": map[string]any{
			"gpt-5.6-sol":   "gpt-5.6-sol",
			"gpt-5.6-terra": "gpt-5.6-terra",
		}},
	}

	require.Equal(t, "gpt-5.6-terra", schedulingLivenessTestModel(account, "gpt-5.6-sol"))
}

func TestSchedulingLivenessDoesNotUseUnverifiedConfiguredModel(t *testing.T) {
	configuredOnly := &Account{Platform: PlatformOpenAI}
	unsupportedConfigured := &Account{
		Platform: PlatformOpenAI,
		Credentials: map[string]any{"model_mapping": map[string]any{
			"gpt-5.6-terra": "gpt-5.6-terra",
		}},
	}

	require.Empty(t, schedulingLivenessTestModel(configuredOnly, "gpt-5.6-sol"))
	require.Equal(t, "gpt-5.6-terra", schedulingLivenessTestModel(unsupportedConfigured, "gpt-5.6-sol"))
}

func TestSchedulingLivenessSkipsOpenAIAccountWithoutVerifiedModel(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{{
		ID:          1,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusError,
		Schedulable: false,
	}}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{
		1: {Status: "success"},
	}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost,
		TestModelID:  "gpt-5.6-sol",
	}})

	NewSuperPriorityRunner(state, tester, repo, &schedulingLivenessRecoverer{}).RunOnce(context.Background())

	require.Empty(t, tester.calls)
	require.Empty(t, repo.livenessWrites)
}

func TestSchedulingLivenessSkipsUnsupportedAccountModelFailure(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{{
		ID:          83,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusError,
		Schedulable: false,
		Credentials: map[string]any{"model_mapping": map[string]any{
			"gpt-5.6-luna": "gpt-5.6-luna",
		}},
	}}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{
		83: {
			Status: "failed",
			ErrorMessage: `API returned 503: {"error":{"code":"model_not_found",` +
				`"message":"No available channel for model gpt-5.6-luna under group codex"}}`,
		},
	}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost,
		TestModelID:  "gpt-5.6-sol",
	}})

	result, err := NewSuperPriorityRunner(state, tester, repo, &schedulingLivenessRecoverer{}).RefreshNow(context.Background())

	require.NoError(t, err)
	require.Equal(t, []int64{83}, tester.calls)
	require.Empty(t, repo.livenessWrites)
	require.Equal(t, SchedulingRefreshResult{Checked: 1, Skipped: 1}, result)
}

func TestSchedulingLivenessRefreshNowSkipsManuallyPausedAccountsByDefault(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{makeSuperPriorityTestAccount(1, false, false)}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{1: {Status: "success"}}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost, CheckInterval: "@every 5m",
	}})
	runner := NewSuperPriorityRunner(state, tester, repo, &schedulingLivenessRecoverer{})

	result, err := runner.RefreshNow(context.Background())

	require.NoError(t, err)
	require.Empty(t, tester.calls)
	require.Zero(t, result.Checked)
}

func TestSchedulingLivenessEmptyScheduledScanKeepsLatestRealResult(t *testing.T) {
	now := time.Now()
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{makeSuperPriorityTestAccount(1, false, true)}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{1: {Status: "success"}}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost, CheckInterval: "@every 5m",
	}})
	runner := NewSuperPriorityRunner(state, tester, repo, &schedulingLivenessRecoverer{})

	_, err := runner.RefreshNow(context.Background())
	require.NoError(t, err)
	require.Equal(t, "manual", runner.RuntimeStatus().LastRun.Trigger)

	fresh := makeSuperPriorityTestAccount(1, false, true)
	fresh.Extra[SchedulingLivenessExtraKey] = map[string]any{
		"status": SchedulingLivenessStatusAlive, "last_attempt_at": now, "fresh_until": now.Add(10 * time.Minute),
	}
	repo.accounts = []Account{fresh}
	runner.RunOnce(context.Background())

	require.Equal(t, "manual", runner.RuntimeStatus().LastRun.Trigger)
}

func TestSchedulingLivenessRefreshNowIgnoresTheConfiguredInterval(t *testing.T) {
	now := time.Now()
	repo := newSuperPriorityFakeRepo()
	account := makeSuperPriorityTestAccount(1, false, true)
	account.Status = StatusError
	account.Schedulable = false
	account.Extra[SchedulingLivenessExtraKey] = map[string]any{
		"status":          SchedulingLivenessStatusAlive,
		"last_attempt_at": now,
		"fresh_until":     now.Add(10 * time.Minute),
	}
	repo.accounts = []Account{account}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{1: {Status: "success"}}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy:  AccountSchedulingStrategyLowestCost,
		CheckInterval: "@every 5m",
	}})

	recoverer := &schedulingLivenessRecoverer{}
	result, err := NewSuperPriorityRunner(state, tester, repo, recoverer).RefreshNow(context.Background())

	require.NoError(t, err)
	require.Equal(t, []int64{1}, tester.calls)
	require.Equal(t, []int64{1}, recoverer.calls)
	require.True(t, repo.schedulable[1])
	require.Equal(t, SchedulingRefreshResult{Checked: 1, Succeeded: 1}, result)
}

func TestSchedulingLivenessRunnerDoesNotProbeInDefaultMode(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{makeSuperPriorityTestAccount(1, false, true)}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{1: {Status: "success"}}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyDefault,
	}})

	NewSuperPriorityRunner(state, tester, repo, &schedulingLivenessRecoverer{}).RunOnce(context.Background())

	require.Empty(t, tester.calls)
}

func TestSchedulingLivenessRunnerRequiresSuccessfulProbeBeforeRecoveringLegacyError(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{{
		ID:           1,
		Type:         AccountTypeAPIKey,
		Status:       StatusError,
		Schedulable:  false,
		ErrorMessage: "Scheduling liveness probe failed: previous connection failure",
		Extra: map[string]any{SchedulingLivenessExtraKey: map[string]any{
			"status":         SchedulingLivenessStatusDead,
			"status_managed": true,
		}},
	}}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{1: {Status: "success"}}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy:  AccountSchedulingStrategyLowestCost,
		CheckInterval: "@every 5m",
	}})

	recoverer := &schedulingLivenessRecoverer{}
	NewSuperPriorityRunner(state, tester, repo, recoverer).RunOnce(context.Background())

	require.Equal(t, []int64{1}, tester.calls)
	require.Equal(t, []int64{1}, recoverer.calls)
	require.True(t, repo.schedulable[1])
	require.Empty(t, repo.clearErrorIDs)
}

func TestSchedulingLivenessRunnerKeepsFailedAccountStopped(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{{
		ID:           1,
		Type:         AccountTypeAPIKey,
		Status:       StatusError,
		Schedulable:  false,
		ErrorMessage: "Authentication failed (401)",
		Extra: map[string]any{SchedulingLivenessExtraKey: map[string]any{
			"status":         SchedulingLivenessStatusDead,
			"status_managed": true,
		}},
	}}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{1: {Status: "failed", ErrorMessage: "Authentication failed (401)"}}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost,
	}})

	recoverer := &schedulingLivenessRecoverer{}
	NewSuperPriorityRunner(state, tester, repo, recoverer).RunOnce(context.Background())

	require.Equal(t, []int64{1}, tester.calls)
	require.Empty(t, recoverer.calls)
	require.Empty(t, repo.schedulable)
	require.Empty(t, repo.clearErrorIDs)
	require.Equal(t, SchedulingLivenessStatusSuspect, repo.livenessWrites[1].Status)
}

func TestSchedulingLivenessRunnerKeepsAccountStoppedWhenRecoveryFails(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	account := makeSuperPriorityTestAccount(1, false, false)
	account.Status = StatusError
	repo.accounts = []Account{account}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{1: {Status: "success"}}}
	recoverer := &schedulingLivenessRecoverer{err: errors.New("database unavailable")}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost,
	}})

	result, err := NewSuperPriorityRunner(state, tester, repo, recoverer).RefreshNow(context.Background())

	require.NoError(t, err)
	require.Equal(t, []int64{1}, tester.calls)
	require.Equal(t, []int64{1}, recoverer.calls)
	require.Empty(t, repo.schedulable)
	require.Equal(t, SchedulingRefreshResult{Checked: 1, Failed: 1}, result)
	require.Equal(t, SchedulingLivenessStatusSuspect, repo.livenessWrites[1].Status)
	require.Contains(t, repo.livenessWrites[1].LastError, "recover account state")
}
