//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeletedStagingAccountIsExcludedFromAutomatedUse(t *testing.T) {
	account := &Account{
		ID:          42,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Extra:       map[string]any{AccountDeletedStagingExtraKey: true},
	}

	require.True(t, account.IsDeletedStaging())
	require.False(t, account.IsSchedulable())
	require.False(t, account.IsCredentialUsableForShadow())
	require.False(t, schedulingLivenessProbeEligible(account, true))
}

func TestDuplicateAccountDropsLifecycleStagingMarkers(t *testing.T) {
	extra, err := duplicateAccountExtra(map[string]any{
		AccountDeletedStagingExtraKey: true,
		"recycled":                    true,
		"operator_note":               "keep",
	})

	require.NoError(t, err)
	require.NotContains(t, extra, AccountDeletedStagingExtraKey)
	require.NotContains(t, extra, "recycled")
	require.Equal(t, "keep", extra["operator_note"])
}

func TestNormalAccountRemainsEligibleAfterDeletedStagingGuard(t *testing.T) {
	account := &Account{
		ID:          43,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Extra:       map[string]any{},
	}

	require.False(t, account.IsDeletedStaging())
	require.True(t, account.IsSchedulable())
	require.True(t, schedulingLivenessProbeEligible(account, false))
}

func TestAdminAccountEditPreservesDeletedStagingLifecycleMarker(t *testing.T) {
	repo := &longContextBillingRepoStub{account: &Account{
		ID:       44,
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			AccountDeletedStagingExtraKey: true,
			"recycled":                    false,
			"operator_note":               "keep",
		},
	}}
	svc := &adminServiceImpl{accountRepo: repo}

	updated, err := svc.UpdateAccount(context.Background(), 44, &UpdateAccountInput{
		Extra: map[string]any{"operator_note": "changed"},
	})

	require.NoError(t, err)
	require.Equal(t, true, updated.Extra[AccountDeletedStagingExtraKey])
	require.Equal(t, false, updated.Extra["recycled"])
	require.Equal(t, "changed", updated.Extra["operator_note"])
}
