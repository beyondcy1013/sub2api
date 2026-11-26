//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeRequestQualityProbeSettingsClampsLegacyBounds(t *testing.T) {
	settings := &RequestQualityProbeSettings{
		Enabled:             true,
		TimeoutSeconds:      600,
		StatusRecentMinutes: 5000,
		RecentMinutes:       5000,
		ReuseSeconds:        intPtr(99999),
		AccountScope:        "all",
		ModelScope:          "all",
		RuleIDs:             []int64{2},
	}
	// 未归一化的历史值仍会被写入校验拒绝
	require.Error(t, ValidateRequestQualityProbeSettings(settings))

	normalized := normalizeRequestQualityProbeSettings(settings)
	require.Equal(t, maxRequestQualityProbeTimeoutSeconds, normalized.TimeoutSeconds)
	require.Equal(t, maxRequestQualityProbeRecentMinutes, normalized.StatusRecentMinutes)
	require.Equal(t, maxRequestQualityProbeRecentMinutes, normalized.RecentMinutes)
	require.Equal(t, maxRequestQualityProbeReuseSeconds, *normalized.ReuseSeconds)
	require.NoError(t, ValidateRequestQualityProbeSettings(normalized))
	// 读取路径同样归一化，账号开关回显后再保存不会因历史值 400
	require.Equal(t, maxRequestQualityProbeTimeoutSeconds, normalizeRequestQualityProbeSettings(normalized).TimeoutSeconds)
}

func TestRequestQualityProbeSettingsDefaultDegradationGuard(t *testing.T) {
	defaults := DefaultRequestQualityProbeSettings()
	require.True(t, defaults.AutoPauseOnDowngrade)
	require.Equal(t, 3, defaults.DowngradeThreshold)

	legacy := normalizeRequestQualityProbeSettings(&RequestQualityProbeSettings{Enabled: true})
	require.True(t, legacy.AutoPauseOnDowngrade)
	require.Equal(t, 3, legacy.DowngradeThreshold)

	tooHigh := normalizeRequestQualityProbeSettings(&RequestQualityProbeSettings{Enabled: true, DowngradeThreshold: 99})
	require.Equal(t, maxRequestQualityProbeDowngradeThreshold, tooHigh.DowngradeThreshold)
	require.NoError(t, ValidateRequestQualityProbeSettings(tooHigh))
}
