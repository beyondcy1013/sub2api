package repository

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBalanceCheckExtraUpdatesAreSchedulerNeutral(t *testing.T) {
	require.False(t, shouldEnqueueSchedulerOutboxForExtraUpdates(map[string]any{
		"balance":                        12.5,
		service.BalanceCheckTypeExtraKey: service.BalanceCheckTypeSub2API,
	}))
}

func TestSuperPriorityExtraUpdateRefreshesSchedulerSnapshot(t *testing.T) {
	require.True(t, shouldEnqueueSchedulerOutboxForExtraUpdates(map[string]any{
		service.SuperPriorityExtraKey: true,
	}))
}
