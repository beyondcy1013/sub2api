package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type schedulingLivenessTester struct {
	mu      sync.Mutex
	results map[int64]*ScheduledTestResult
	calls   []int64
}

func (t *schedulingLivenessTester) RunTestBackground(_ context.Context, accountID int64, _ string) (*ScheduledTestResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls = append(t.calls, accountID)
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

func TestSchedulingLivenessRunnerChecksEveryActiveAccountInLowestCostMode(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{
		makeSuperPriorityTestAccount(1, true, true),
		makeSuperPriorityTestAccount(2, false, true),
	}
	tester := &schedulingLivenessTester{results: map[int64]*ScheduledTestResult{
		1: {Status: "success"},
		2: {Status: "failed", ErrorMessage: "timeout"},
	}}
	state := NewSuperPriorityService(repo, &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy:     AccountSchedulingStrategyLowestCost,
		FailureThreshold: 2,
		CheckInterval:    "@every 5m",
	}})
	runner := NewSuperPriorityRunner(state, tester, repo)

	runner.RunOnce(context.Background())

	require.ElementsMatch(t, []int64{1, 2}, tester.calls)
	alive, ok := repo.extraWrites[1][SchedulingLivenessExtraKey].(*AccountSchedulingLiveness)
	require.True(t, ok)
	require.Equal(t, SchedulingLivenessStatusAlive, alive.Status)
	suspect, ok := repo.extraWrites[2][SchedulingLivenessExtraKey].(*AccountSchedulingLiveness)
	require.True(t, ok)
	require.Equal(t, SchedulingLivenessStatusSuspect, suspect.Status)
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
