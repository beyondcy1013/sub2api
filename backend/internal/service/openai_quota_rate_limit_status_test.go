package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimitServiceGetOpenAIQuotaRateLimitStatus(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	resetAt := now.Add(90 * time.Minute)
	account := &Account{
		ID:       701,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"codex_7d_used_percent":   91.0,
			"codex_7d_reset_at":       resetAt.Format(time.RFC3339),
			"codex_usage_updated_at":  now.Format(time.RFC3339),
			"auto_pause_7d_threshold": 0.9,
			"auto_pause_5h_disabled":  true,
			"auto_pause_7d_disabled":  false,
		},
	}
	service := NewRateLimitService(nil, nil, nil, nil, nil)

	status := service.GetOpenAIQuotaRateLimitStatus(context.Background(), account)

	require.NotNil(t, status)
	require.Equal(t, "7d", status.Window)
	require.InDelta(t, 0.9, status.Threshold, 0.0001)
	require.InDelta(t, 0.91, status.Utilization, 0.0001)
	require.NotNil(t, status.ResetAt)
	require.WithinDuration(t, resetAt, *status.ResetAt, time.Second)
}

func TestRateLimitServiceGetOpenAIQuotaRateLimitStatusHonorsDisableFlag(t *testing.T) {
	now := time.Now().UTC()
	account := &Account{
		ID:       702,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"codex_7d_used_percent":   95.0,
			"codex_7d_reset_at":       now.Add(time.Hour).Format(time.RFC3339),
			"codex_usage_updated_at":  now.Format(time.RFC3339),
			"auto_pause_7d_threshold": 0.9,
			"auto_pause_7d_disabled":  true,
		},
	}
	service := NewRateLimitService(nil, nil, nil, nil, nil)

	require.Nil(t, service.GetOpenAIQuotaRateLimitStatus(context.Background(), account))
}
