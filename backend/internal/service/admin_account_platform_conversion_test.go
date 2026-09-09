//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateAccount_ConvertsAnthropicAPIKeyToOpenAIAndBack(t *testing.T) {
	for _, tc := range []struct {
		name         string
		fromPlatform string
		toPlatform   string
	}{
		{name: "anthropic to openai", fromPlatform: PlatformAnthropic, toPlatform: PlatformOpenAI},
		{name: "openai to anthropic", fromPlatform: PlatformOpenAI, toPlatform: PlatformAnthropic},
	} {
		t.Run(tc.name, func(t *testing.T) {
			accountID := int64(310)
			repo := &updateAccountCredsRepoStub{
				account: &Account{
					ID:          accountID,
					Platform:    tc.fromPlatform,
					Type:        AccountTypeAPIKey,
					Status:      StatusActive,
					Credentials: map[string]any{"api_key": "sk-existing"},
				},
			}
			svc := &adminServiceImpl{accountRepo: repo}

			updated, err := svc.UpdateAccount(context.Background(), accountID, &UpdateAccountInput{
				Platform:    tc.toPlatform,
				Type:        AccountTypeAPIKey,
				Credentials: map[string]any{"base_url": "https://relay.example"},
			})

			require.NoError(t, err)
			require.Equal(t, tc.toPlatform, updated.Platform)
			require.Equal(t, AccountTypeAPIKey, updated.Type)
			require.Equal(t, "sk-existing", repo.account.Credentials["api_key"])
			require.Equal(t, "https://relay.example", repo.account.Credentials["base_url"])
		})
	}
}

func TestUpdateAccount_RejectsPlatformChangeOutsideAPIKeyPair(t *testing.T) {
	accountID := int64(311)
	repo := &updateAccountCredsRepoStub{
		account: &Account{
			ID:          accountID,
			Platform:    PlatformAnthropic,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Credentials: map[string]any{"refresh_token": "rt-existing"},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	updated, err := svc.UpdateAccount(context.Background(), accountID, &UpdateAccountInput{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	})

	require.Error(t, err)
	require.Nil(t, updated)
	require.Equal(t, PlatformAnthropic, repo.account.Platform)
}
