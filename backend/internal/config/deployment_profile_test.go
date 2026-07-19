package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeploymentProfileDefaultsToMain(t *testing.T) {
	resetViperWithJWTSecret(t)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, DeploymentProfileMain, cfg.DeploymentProfile())
	require.Equal(t, RuntimeCapabilities{
		BalanceCheck:              false,
		StickySessionReassignment: true,
		BrandedErrors:             false,
	}, cfg.RuntimeCapabilities())
	require.NotNil(t, cfg.BalanceCheck.Enabled)
	require.False(t, *cfg.BalanceCheck.Enabled)
}

func TestDeploymentProfileFreeFromEnvironment(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("DEPLOYMENT_PROFILE", DeploymentProfileFree)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, DeploymentProfileFree, cfg.DeploymentProfile())
	require.Equal(t, RuntimeCapabilities{
		BalanceCheck:              true,
		StickySessionReassignment: false,
		BrandedErrors:             true,
	}, cfg.RuntimeCapabilities())
	require.NotNil(t, cfg.BalanceCheck.Enabled)
	require.True(t, *cfg.BalanceCheck.Enabled)
}

func TestDeploymentProfileRejectsUnknownValue(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("DEPLOYMENT_PROFILE", "enterprise")

	_, err := Load()
	require.ErrorContains(t, err, "deployment.profile")
}

func TestDeploymentProfileExplicitBalanceCheckOverrideWins(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("DEPLOYMENT_PROFILE", DeploymentProfileFree)
	t.Setenv("BALANCE_CHECK_ENABLED", "false")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg.BalanceCheck.Enabled)
	require.False(t, *cfg.BalanceCheck.Enabled)
}
