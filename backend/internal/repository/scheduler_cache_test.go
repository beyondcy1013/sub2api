package repository

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestFilterSchedulerCredentialsKeepsSubscriptionPlanType(t *testing.T) {
	filtered := filterSchedulerCredentials(map[string]any{
		"plan_type":     "plus",
		"access_token":  "secret-access-token",
		"refresh_token": "secret-refresh-token",
	})

	require.Equal(t, "plus", filtered["plan_type"])
	require.NotContains(t, filtered, "access_token")
	require.NotContains(t, filtered, "refresh_token")
}

func TestSchedulerMetadataAccountKeepsOpenAISubscriptionIdentity(t *testing.T) {
	account := service.Account{
		ID:       24,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type":    "plus",
			"access_token": "secret-access-token",
		},
	}

	metadata := buildSchedulerMetadataAccount(account)

	require.True(t, metadata.IsOpenAIChatGPTSubscription())
	require.Empty(t, metadata.GetCredential("access_token"))
}

func TestSchedulerCacheKeyPrefix(t *testing.T) {
	cache := &schedulerCache{keyPrefix: normalizeRedisKeyPrefix("sub2freeApi:")}
	bucket := service.SchedulerBucket{GroupID: 2, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}

	if got, want := cache.schedulerAccountKey("7"), "sub2freeApi:sched:acc:7"; got != want {
		t.Fatalf("schedulerAccountKey() = %q, want %q", got, want)
	}
	if got, want := cache.schedulerBucketKey(schedulerReadyPrefix, bucket), "sub2freeApi:sched:ready:2:openai:single"; got != want {
		t.Fatalf("schedulerBucketKey() = %q, want %q", got, want)
	}
	if got, want := normalizeRedisKeyPrefix(" sub2freeApi:: "), "sub2freeApi:"; got != want {
		t.Fatalf("normalizeRedisKeyPrefix() = %q, want %q", got, want)
	}
}
