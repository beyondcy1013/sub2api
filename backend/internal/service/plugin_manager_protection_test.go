package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPluginManagerSkipsDisabledOrIncompatibleProtectionRuntime(t *testing.T) {
	manifest := testStateReuseV2ManifestForRouting()
	manager := &PluginManager{
		runtimes: map[int64]*pluginRuntime{
			1: {
				installation: &PluginInstallation{
					ID: 1, State: PluginStateDisabled, Manifest: manifest,
				},
			},
		},
	}
	request := httptest.NewRequest(http.MethodPost, "https://chatgpt.com/backend-api/codex/responses", nil)
	response, handled, err := manager.RoundTripOpenAIProtection(context.Background(), request, "", &Account{
		ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth,
	})
	require.NoError(t, err)
	require.False(t, handled)
	require.Nil(t, response)
}

func testStateReuseV2ManifestForRouting() PluginManifest {
	return PluginManifest{
		SchemaVersion: 2,
		Capabilities: []PluginCapability{{
			ID:          PluginCapabilityOpenAIProtectionTransport,
			Platform:    PlatformOpenAI,
			AccountType: AccountTypeOAuth,
		}},
	}
}
