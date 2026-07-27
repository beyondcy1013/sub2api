//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newRelayFailureBudgetCacheTest(t *testing.T) service.RelayFailureBudgetCache {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewRelayFailureBudgetCache(rdb)
}

func relayFailureBudgetTestPolicy() service.RelayFailureBudgetPolicy {
	return service.RelayFailureBudgetPolicy{
		Window:                  10 * time.Minute,
		FailureThresholdPercent: 30,
		MinRequests:             10,
		ConsecutiveFailures:     5,
		Cooldown:                2 * time.Minute,
	}
}

func TestRelayFailureBudgetCacheTripsAtConfiguredWindowRatio(t *testing.T) {
	cache := newRelayFailureBudgetCacheTest(t)
	ctx := context.Background()
	policy := relayFailureBudgetTestPolicy()
	now := time.Date(2026, time.July, 27, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 7; i++ {
		decision, err := cache.RecordRelayFailureBudgetEvent(ctx, 83, now.Add(time.Duration(i)*time.Second), service.RelayFailureBudgetSuccess, policy)
		require.NoError(t, err)
		require.False(t, decision.Tripped)
	}
	for i := 0; i < 2; i++ {
		decision, err := cache.RecordRelayFailureBudgetEvent(ctx, 83, now.Add(time.Duration(7+i)*time.Second), service.RelayFailureBudgetFailure, policy)
		require.NoError(t, err)
		require.False(t, decision.Tripped)
	}

	decision, err := cache.RecordRelayFailureBudgetEvent(ctx, 83, now.Add(9*time.Second), service.RelayFailureBudgetFailure, policy)

	require.NoError(t, err)
	require.True(t, decision.Tripped)
	require.Equal(t, 10, decision.WindowRequests)
	require.Equal(t, 3, decision.WindowFailures)
	require.Equal(t, 3, decision.ConsecutiveFailures)
	require.Equal(t, now.Add(9*time.Second).Add(policy.Cooldown), decision.CooldownUntil)
}

func TestRelayFailureBudgetCacheConsecutiveFailuresTripBeforeMinimumRequests(t *testing.T) {
	cache := newRelayFailureBudgetCacheTest(t)
	ctx := context.Background()
	policy := relayFailureBudgetTestPolicy()
	now := time.Date(2026, time.July, 27, 12, 10, 0, 0, time.UTC)

	for i := 0; i < policy.ConsecutiveFailures-1; i++ {
		decision, err := cache.RecordRelayFailureBudgetEvent(ctx, 84, now.Add(time.Duration(i)*time.Second), service.RelayFailureBudgetFailure, policy)
		require.NoError(t, err)
		require.False(t, decision.Tripped)
	}

	decision, err := cache.RecordRelayFailureBudgetEvent(ctx, 84, now.Add(4*time.Second), service.RelayFailureBudgetFailure, policy)

	require.NoError(t, err)
	require.True(t, decision.Tripped)
	require.Equal(t, 5, decision.WindowRequests)
	require.Equal(t, 5, decision.ConsecutiveFailures)
}

func TestRelayFailureBudgetCachePrunesExpiredBuckets(t *testing.T) {
	cache := newRelayFailureBudgetCacheTest(t)
	ctx := context.Background()
	policy := relayFailureBudgetTestPolicy()
	now := time.Date(2026, time.July, 27, 12, 20, 0, 0, time.UTC)

	for i := 0; i < 4; i++ {
		_, err := cache.RecordRelayFailureBudgetEvent(ctx, 85, now.Add(time.Duration(i)*time.Second), service.RelayFailureBudgetFailure, policy)
		require.NoError(t, err)
	}

	decision, err := cache.RecordRelayFailureBudgetEvent(ctx, 85, now.Add(11*time.Minute), service.RelayFailureBudgetSuccess, policy)

	require.NoError(t, err)
	require.False(t, decision.Tripped)
	require.Equal(t, 1, decision.WindowRequests)
	require.Zero(t, decision.WindowFailures)
	require.Zero(t, decision.ConsecutiveFailures)
}

func TestRelayFailureBudgetCacheSuccessfulHalfOpenClearsExpiredCooldownHistory(t *testing.T) {
	cache := newRelayFailureBudgetCacheTest(t)
	ctx := context.Background()
	policy := relayFailureBudgetTestPolicy()
	policy.ConsecutiveFailures = 2
	now := time.Date(2026, time.July, 27, 12, 40, 0, 0, time.UTC)

	_, err := cache.RecordRelayFailureBudgetEvent(ctx, 86, now, service.RelayFailureBudgetFailure, policy)
	require.NoError(t, err)
	tripped, err := cache.RecordRelayFailureBudgetEvent(ctx, 86, now.Add(time.Second), service.RelayFailureBudgetFailure, policy)
	require.NoError(t, err)
	require.True(t, tripped.Tripped)

	recoveredAt := tripped.CooldownUntil.Add(time.Second)
	decision, err := cache.RecordRelayFailureBudgetEvent(ctx, 86, recoveredAt, service.RelayFailureBudgetSuccess, policy)

	require.NoError(t, err)
	require.False(t, decision.Tripped)
	require.True(t, decision.CooldownUntil.IsZero())
	require.Equal(t, 1, decision.WindowRequests)
	require.Zero(t, decision.WindowFailures)
	require.Zero(t, decision.ConsecutiveFailures)
}
