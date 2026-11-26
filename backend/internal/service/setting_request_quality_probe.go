package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SettingKeyRequestQualityProbeSettings stores the configuration for on-demand
// an independent quality request alongside each matching downstream request.
const SettingKeyRequestQualityProbeSettings = "request_quality_probe_settings"

const requestQualityProbeSettingsCacheTTL = 15 * time.Second

type cachedRequestQualityProbeSettings struct {
	settings  *RequestQualityProbeSettings
	expiresAt time.Time
}

// RequestQualityProbeSettings configures the per-request response quality gate.
type RequestQualityProbeSettings struct {
	// Enabled is the master switch. Disabled requests use the original forwarder.
	Enabled bool `json:"enabled"`
	// Models specifies the models requiring testing. An empty selected list tests
	// no models; testing all models requires ModelScope explicitly set to "all".
	Models               []string `json:"models"`
	ModelScope           string   `json:"model_scope"`
	ExcludeModels        []string `json:"exclude_models,omitempty"`
	RuleIDs              []int64  `json:"rule_ids"`
	AllowBusinessSuccess bool     `json:"allow_business_success"`
	TimeoutSeconds       int      `json:"timeout_seconds"`
	// RecentMinutes and AllowBusinessSuccess are retained for legacy clients.
	// The per-request gate never uses historical exemptions.
	RecentMinutes int `json:"recent_minutes"`
	// StatusRecentMinutes only controls the account table freshness display.
	StatusRecentMinutes int `json:"status_recent_minutes"`
	// AccountScope controls which accounts are subject to testing:
	// "all": all accounts supporting the designated models
	// "error_or_downgraded": only accounts in error or pelican_downgraded state (可能出问题)
	// "selected": only explicitly listed AccountIDs (or account.Extra["test_on_request_enabled"])
	AccountScope string `json:"account_scope"`
	// AccountIDs contains the explicit account IDs when AccountScope == "selected".
	AccountIDs []int64 `json:"account_ids"`
	// AccountOverrides explicitly includes/excludes individual accounts without changing the scope.
	AccountOverrides map[int64]bool `json:"account_overrides,omitempty"`
	// ReuseSeconds releases a request without a new probe when the same account,
	// model and selected question passed within this window. Only a passing
	// record is reused; failures always probe again. Zero keeps the strict
	// per-request behaviour, a missing value falls back to the default.
	ReuseSeconds *int `json:"reuse_seconds,omitempty"`
	// AutoPauseOnDowngrade removes an account from scheduling after the
	// configured number of consecutive confirmed quality failures.
	AutoPauseOnDowngrade bool `json:"auto_pause_on_downgrade"`
	// DowngradeThreshold is the consecutive confirmed-degradation limit.
	DowngradeThreshold int `json:"downgrade_threshold"`
}

// Upper bounds shared by the normalizer and the write validator. Values above
// these bounds exist in configurations saved before the bounds were tightened
// (for example timeout_seconds=600). The normalizer clamps them so a legacy
// value can no longer make every later save fail validation.
const (
	maxRequestQualityProbeTimeoutSeconds     = 120
	maxRequestQualityProbeRecentMinutes      = 1440
	maxRequestQualityProbeReuseSeconds       = 3600
	maxRequestQualityProbeDowngradeThreshold = 10
)

// DefaultRequestQualityProbeReuseSeconds reuses a passing probe for five
// minutes, cutting one upstream check per request down to at most one per
// window while still detecting degradation within that window.
const DefaultRequestQualityProbeReuseSeconds = 300

// reuseWindowSeconds resolves the effective reuse window.
func (s *RequestQualityProbeSettings) reuseWindowSeconds() int {
	if s == nil || s.ReuseSeconds == nil {
		return DefaultRequestQualityProbeReuseSeconds
	}
	value := *s.ReuseSeconds
	if value < 0 {
		return 0
	}
	return value
}

// DefaultRequestQualityProbeSettings returns the default configuration.
func DefaultRequestQualityProbeSettings() *RequestQualityProbeSettings {
	return &RequestQualityProbeSettings{
		Enabled:              false,
		ModelScope:           "all",
		RuleIDs:              []int64{},
		TimeoutSeconds:       35,
		Models:               []string{},
		ExcludeModels:        []string{"glm*"},
		RecentMinutes:        10,
		StatusRecentMinutes:  5,
		AccountScope:         "all",
		AccountIDs:           []int64{},
		ReuseSeconds:         intPtr(DefaultRequestQualityProbeReuseSeconds),
		AutoPauseOnDowngrade: true,
		DowngradeThreshold:   3,
	}
}

func normalizeRequestQualityProbeSettings(s *RequestQualityProbeSettings) *RequestQualityProbeSettings {
	if s == nil {
		return DefaultRequestQualityProbeSettings()
	}
	res := *s
	if res.StatusRecentMinutes <= 0 {
		res.StatusRecentMinutes = 5
	}
	if res.StatusRecentMinutes > maxRequestQualityProbeRecentMinutes {
		res.StatusRecentMinutes = maxRequestQualityProbeRecentMinutes
	}
	if res.ModelScope != "all" {
		res.ModelScope = "selected"
	}
	if res.TimeoutSeconds <= 0 {
		res.TimeoutSeconds = 35
	}
	if res.TimeoutSeconds > maxRequestQualityProbeTimeoutSeconds {
		res.TimeoutSeconds = maxRequestQualityProbeTimeoutSeconds
	}
	if res.RuleIDs == nil {
		res.RuleIDs = []int64{}
	}
	if res.RecentMinutes <= 0 {
		res.RecentMinutes = 10
	}
	if res.RecentMinutes > maxRequestQualityProbeRecentMinutes {
		res.RecentMinutes = maxRequestQualityProbeRecentMinutes
	}
	scope := strings.ToLower(strings.TrimSpace(res.AccountScope))
	switch scope {
	case "all", "selected":
		res.AccountScope = scope
	default:
		res.AccountScope = "error_or_downgraded"
	}
	if res.Models == nil {
		res.Models = []string{}
	} else {
		var cleaned []string
		for _, m := range res.Models {
			trimmed := strings.TrimSpace(m)
			if trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		res.Models = cleaned
	}
	if res.ExcludeModels == nil {
		res.ExcludeModels = []string{"glm*"}
	} else {
		var cleaned []string
		for _, m := range res.ExcludeModels {
			trimmed := strings.TrimSpace(m)
			if trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		res.ExcludeModels = cleaned
	}
	if res.ReuseSeconds == nil {
		res.ReuseSeconds = intPtr(DefaultRequestQualityProbeReuseSeconds)
	} else if *res.ReuseSeconds < 0 {
		res.ReuseSeconds = intPtr(0)
	} else if *res.ReuseSeconds > maxRequestQualityProbeReuseSeconds {
		res.ReuseSeconds = intPtr(maxRequestQualityProbeReuseSeconds)
	}
	if res.DowngradeThreshold <= 0 {
		res.DowngradeThreshold = 3
		// A legacy JSON object has neither the threshold nor the pause flag.
		// Treat that shape as the safe default. Explicitly disabled settings
		// saved by the current UI carry a positive threshold and stay false.
		if !res.AutoPauseOnDowngrade {
			res.AutoPauseOnDowngrade = true
		}
	}
	if res.DowngradeThreshold > maxRequestQualityProbeDowngradeThreshold {
		res.DowngradeThreshold = maxRequestQualityProbeDowngradeThreshold
	}
	if res.AccountIDs == nil {
		res.AccountIDs = []int64{}
	}
	return &res
}

// GetRequestQualityProbeSettings returns the cached or database settings.
func (s *SettingService) GetRequestQualityProbeSettings(ctx context.Context) (*RequestQualityProbeSettings, error) {
	if s == nil || s.settingRepo == nil {
		return DefaultRequestQualityProbeSettings(), nil
	}
	if cached, ok := s.requestQualityProbeCache.Load().(*cachedRequestQualityProbeSettings); ok && cached != nil && time.Now().Before(cached.expiresAt) {
		return cached.settings, nil
	}

	value, err := s.settingRepo.GetValue(ctx, SettingKeyRequestQualityProbeSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			defaults := DefaultRequestQualityProbeSettings()
			s.requestQualityProbeCache.Store(&cachedRequestQualityProbeSettings{
				settings:  defaults,
				expiresAt: time.Now().Add(requestQualityProbeSettingsCacheTTL),
			})
			return defaults, nil
		}
		return nil, fmt.Errorf("get request quality probe settings: %w", err)
	}

	settings := DefaultRequestQualityProbeSettings()
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		// Saved pre-scope configurations selected explicit models. Only a new
		// unsaved configuration defaults to all models.
		settings.ModelScope = "selected"
		if err := json.Unmarshal([]byte(trimmed), settings); err != nil {
			return nil, fmt.Errorf("parse request quality probe settings: %w", err)
		}
	}
	settings = normalizeRequestQualityProbeSettings(settings)
	s.requestQualityProbeCache.Store(&cachedRequestQualityProbeSettings{
		settings:  settings,
		expiresAt: time.Now().Add(requestQualityProbeSettingsCacheTTL),
	})
	return settings, nil
}

// UpdateRequestQualityProbeSettings persists and caches the settings.
func (s *SettingService) UpdateRequestQualityProbeSettings(ctx context.Context, settings *RequestQualityProbeSettings) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting repository unavailable")
	}
	normalized := normalizeRequestQualityProbeSettings(settings)
	data, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("marshal request quality probe settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyRequestQualityProbeSettings, string(data)); err != nil {
		return fmt.Errorf("persist request quality probe settings: %w", err)
	}
	s.requestQualityProbeCache.Store(&cachedRequestQualityProbeSettings{
		settings:  normalized,
		expiresAt: time.Now().Add(requestQualityProbeSettingsCacheTTL),
	})
	return nil
}

// InvalidateRequestQualityProbeSettingsCache drops the cached settings.
func (s *SettingService) InvalidateRequestQualityProbeSettingsCache() {
	if s == nil {
		return
	}
	s.requestQualityProbeCache.Store((*cachedRequestQualityProbeSettings)(nil))
}

func ValidateRequestQualityProbeSettings(s *RequestQualityProbeSettings) error {
	if s.StatusRecentMinutes != 0 && (s.StatusRecentMinutes < 1 || s.StatusRecentMinutes > maxRequestQualityProbeRecentMinutes) {
		return fmt.Errorf("status_recent_minutes must be 1..1440")
	}
	// Disabling must always work, including old incomplete configurations.
	if !s.Enabled {
		return nil
	}
	if s.RecentMinutes != 0 && (s.RecentMinutes < 1 || s.RecentMinutes > maxRequestQualityProbeRecentMinutes) {
		return fmt.Errorf("recent_minutes must be 1..1440")
	}
	if s.TimeoutSeconds != 0 && (s.TimeoutSeconds < 5 || s.TimeoutSeconds > maxRequestQualityProbeTimeoutSeconds) {
		return fmt.Errorf("timeout_seconds must be 5..120")
	}
	if s.ReuseSeconds != nil && (*s.ReuseSeconds < 0 || *s.ReuseSeconds > maxRequestQualityProbeReuseSeconds) {
		return fmt.Errorf("reuse_seconds must be 0..3600")
	}
	if s.DowngradeThreshold != 0 && (s.DowngradeThreshold < 1 || s.DowngradeThreshold > maxRequestQualityProbeDowngradeThreshold) {
		return fmt.Errorf("downgrade_threshold must be 1..10")
	}
	if s.ModelScope != "" && s.ModelScope != "selected" && s.ModelScope != "all" {
		return fmt.Errorf("invalid model_scope")
	}
	if s.AccountScope != "all" && s.AccountScope != "selected" && s.AccountScope != "error_or_downgraded" {
		return fmt.Errorf("invalid account_scope")
	}
	if s.Enabled && len(s.RuleIDs) > 1 {
		return fmt.Errorf("select at most one rule for the extra quality request")
	}
	for id := range s.AccountOverrides {
		if id <= 0 {
			return fmt.Errorf("account override IDs must be positive")
		}
	}
	for _, id := range append(append([]int64{}, s.AccountIDs...), s.RuleIDs...) {
		if id <= 0 {
			return fmt.Errorf("account and rule IDs must be positive")
		}
	}
	return nil
}
