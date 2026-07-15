package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestStickySessionAdminStoreSummarizeAndReassign(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	store := NewStickySessionAdminStore(rdb)
	seed := func(key string, accountID int64, ttl time.Duration) {
		require.NoError(t, rdb.Set(ctx, key, accountID, ttl).Err())
	}

	seed("sticky_session:2:openai:1111111111111111", 12, 55*time.Minute)
	seed("sticky_session:2:openai:2222222222222222", 12, 59*time.Minute+50*time.Second)
	seed("sticky_session:2:openai:3333333333333333", 13, 40*time.Minute)
	seed("sticky_session:2:openai:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 12, 50*time.Minute)
	seed("sticky_session:2:openai:response:chain", 12, 50*time.Minute)
	seed("sticky_session:3:openai:4444444444444444", 12, 50*time.Minute)
	seed("sticky_session:2:gemini:5555555555555555", 12, 50*time.Minute)

	summary, err := store.Summarize(ctx, 2, "openai")
	require.NoError(t, err)
	require.Equal(t, map[int64]int{12: 2, 13: 1}, summary.Counts)
	require.Equal(t, 3, summary.Total)
	require.Equal(t, 1, summary.ProtectedResponseBindings)
	require.Len(t, summary.Activities[12], 2)
	require.Equal(t, "2222222222222222", summary.Activities[12][0].SessionHash)
	require.LessOrEqual(t, summary.Activities[12][0].ActiveAgo, 11*time.Second)

	result, err := store.Reassign(ctx, 2, "openai", 12, 20, 1, 5*time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, result.Moved)
	require.Equal(t, 1, result.RemainingSourceBindings)

	value, getErr := mr.Get("sticky_session:2:openai:1111111111111111")
	require.NoError(t, getErr)
	require.Equal(t, "12", value)
	value, getErr = mr.Get("sticky_session:2:openai:2222222222222222")
	require.NoError(t, getErr)
	require.Equal(t, "20", value)
	value, getErr = mr.Get("sticky_session:2:openai:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.NoError(t, getErr)
	require.Equal(t, "12", value)
	value, getErr = mr.Get("sticky_session:2:openai:response:chain")
	require.NoError(t, getErr)
	require.Equal(t, "12", value)
	value, getErr = mr.Get("sticky_session:3:openai:4444444444444444")
	require.NoError(t, getErr)
	require.Equal(t, "12", value)
	value, getErr = mr.Get("sticky_session:2:gemini:5555555555555555")
	require.NoError(t, getErr)
	require.Equal(t, "12", value)
	require.Equal(t, 59*time.Minute+50*time.Second, mr.TTL("sticky_session:2:openai:2222222222222222"))
}

func TestStickySessionAdminStoreReassignDoesNotOverwriteChangedBinding(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	store := NewStickySessionAdminStore(rdb)
	key := "sticky_session:2:openai:aaaaaaaaaaaaaaaa"
	require.NoError(t, rdb.Set(ctx, key, 12, 20*time.Minute).Err())

	store.beforeCompareAndSet = func() {
		require.NoError(t, rdb.Set(ctx, key, 13, 20*time.Minute).Err())
	}

	result, err := store.Reassign(ctx, 2, "openai", 12, 20, 1, 5*time.Minute)
	require.NoError(t, err)
	require.Zero(t, result.Moved)
	value, getErr := mr.Get(key)
	require.NoError(t, getErr)
	require.Equal(t, "13", value)
}

func TestStickySessionAdminStoreReassignOnlyMovesRecentlyActiveBindings(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	store := NewStickySessionAdminStore(rdb)
	recentKey := "sticky_session:2:openai:aaaaaaaaaaaaaaaa"
	staleKey := "sticky_session:2:openai:bbbbbbbbbbbbbbbb"
	require.NoError(t, rdb.Set(ctx, recentKey, 12, 59*time.Minute).Err())
	require.NoError(t, rdb.Set(ctx, staleKey, 12, 30*time.Minute).Err())

	result, err := store.Reassign(ctx, 2, "openai", 12, 20, 2, 5*time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, result.Moved)
	require.Equal(t, 1, result.RemainingSourceBindings)

	recentValue, err := mr.Get(recentKey)
	require.NoError(t, err)
	require.Equal(t, "20", recentValue)
	staleValue, err := mr.Get(staleKey)
	require.NoError(t, err)
	require.Equal(t, "12", staleValue)
}
