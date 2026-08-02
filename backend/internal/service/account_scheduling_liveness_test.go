package service

import (
	"context"
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

func TestSchedulingLivenessRunnerSkipsManuallyPausedAccountsByDefault(t *testing.T) {
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
	runner := NewSuperPriorityRunner(state, tester, repo)

	runner.RunOnce(context.Background())

	require.ElementsMatch(t, []int64{1, 2}, tester.calls)
	alive, ok := repo.livenessWrites[1]
	require.True(t, ok)
	require.Equal(t, SchedulingLivenessStatusAlive, alive.Status)
	suspect, ok := repo.livenessWrites[2]
	require.True(t, ok)
	require.Equal(t, SchedulingLivenessStatusSuspect, suspect.Status)
	require.NotContains(t, repo.livenessWrites, int64(3))
}

func TestSchedulingLivenessRunnerIncludesManuallyPausedAccountsWhenEnabled(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{makeSuperPriorityTestAccount(1, false, false)}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{
		1: {Status: "success"},
	}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost, CheckInterval: "@every 5m", LivenessIncludeUnschedulable: true,
	}})
	runner := NewSuperPriorityRunner(state, tester, repo)

	result, err := runner.RefreshNow(context.Background())

	require.NoError(t, err)
	require.Equal(t, []int64{1}, tester.calls)
	require.Equal(t, SchedulingRefreshResult{Checked: 1, Succeeded: 1}, result)
	status := runner.RuntimeStatus()
	require.True(t, status.Enabled)
	require.False(t, status.Running)
	require.NotNil(t, status.NextRunAt)
	require.NotNil(t, status.LastRun)
	require.Equal(t, "manual", status.LastRun.Trigger)
	require.Equal(t, result, status.LastRun.Result)
}

func TestSchedulingLivenessRunnerOnlyProbesAPIKeyAccounts(t *testing.T) {
	apiKey := makeSuperPriorityTestAccount(1, false, false)
	oauth := makeSuperPriorityTestAccount(2, false, true)
	oauth.Type = AccountTypeOAuth
	setupToken := makeSuperPriorityTestAccount(3, false, true)
	setupToken.Type = AccountTypeSetupToken
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

	result, err := NewSuperPriorityRunner(state, tester, repo).RefreshNow(context.Background())

	require.NoError(t, err)
	require.Equal(t, []int64{1}, tester.calls)
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
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{"model_mapping": map[string]any{
				"gpt-5.6-sol": "gpt-5.6-sol",
			}},
		},
		{
			ID:          2,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{"model_mapping": map[string]any{
				"gpt-5.6-terra": "gpt-5.6-terra",
			}},
		},
		{ID: 3, Platform: PlatformGrok, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true},
		{ID: 4, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true},
		{ID: 5, Platform: PlatformGemini, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true},
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

	NewSuperPriorityRunner(state, tester, repo).RunOnce(context.Background())

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
		Status:      StatusActive,
		Schedulable: true,
	}}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{
		1: {Status: "success"},
	}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost,
		TestModelID:  "gpt-5.6-sol",
	}})

	NewSuperPriorityRunner(state, tester, repo).RunOnce(context.Background())

	require.Empty(t, tester.calls)
	require.Empty(t, repo.livenessWrites)
}

func TestSchedulingLivenessSkipsUnsupportedAccountModelFailure(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{{
		ID:          83,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
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

	result, err := NewSuperPriorityRunner(state, tester, repo).RefreshNow(context.Background())

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
	runner := NewSuperPriorityRunner(state, tester, repo)

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
	runner := NewSuperPriorityRunner(state, tester, repo)

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

	result, err := NewSuperPriorityRunner(state, tester, repo).RefreshNow(context.Background())

	require.NoError(t, err)
	require.Equal(t, []int64{1}, tester.calls)
	require.Equal(t, SchedulingRefreshResult{Checked: 1, Succeeded: 1}, result)
}

func TestSchedulingLivenessRunnerDoesNotProbeInDefaultMode(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{makeSuperPriorityTestAccount(1, false, true)}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{1: {Status: "success"}}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyDefault,
	}})

	NewSuperPriorityRunner(state, tester, repo).RunOnce(context.Background())

	require.Empty(t, tester.calls)
}

func TestSchedulingLivenessRunnerClearsLegacyManagedErrorWithoutProbing(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{{
		ID:           1,
		Status:       StatusError,
		Schedulable:  true,
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

	NewSuperPriorityRunner(state, tester, repo).RunOnce(context.Background())

	require.Empty(t, tester.calls)
	require.Equal(t, []int64{1}, repo.clearErrorIDs)
}

func TestSchedulingLivenessRunnerDoesNotProbeOtherErrorStates(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{{
		ID:           1,
		Status:       StatusError,
		Schedulable:  true,
		ErrorMessage: "Authentication failed (401)",
		Extra: map[string]any{SchedulingLivenessExtraKey: map[string]any{
			"status":         SchedulingLivenessStatusDead,
			"status_managed": true,
		}},
	}}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{1: {Status: "success"}}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: AccountSchedulingStrategyLowestCost,
	}})

	NewSuperPriorityRunner(state, tester, repo).RunOnce(context.Background())

	require.Empty(t, tester.calls)
	require.Empty(t, repo.clearErrorIDs)
	require.Empty(t, repo.livenessWrites)
}
