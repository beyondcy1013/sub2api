package service

import (
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	AccountSchedulingStrategyDefault    = "default"
	AccountSchedulingStrategyLowestCost = "lowest_cost"

	// SchedulingRateSourceExtraKey is retained for backwards compatibility with
	// the first scheduling-rate UI. New writes use SchedulingRateSyncModeExtraKey.
	SchedulingRateSourceExtraKey = "scheduling_rate_source"
	SchedulingRateSourceManual   = "manual"
	SchedulingRateSourceUpstream = "upstream"

	// SchedulingRateSyncModeExtraKey controls whether a successful upstream
	// billing probe may overwrite accounts.rate_multiplier. The persisted column
	// is always the single value used by the lowest-cost scheduler.
	SchedulingRateSyncModeExtraKey      = "scheduling_rate_sync_mode"
	SchedulingRateSyncModeAutoOverwrite = "auto_overwrite"
	SchedulingRateSyncModeManualLock    = "manual_lock"
)

func normalizeSchedulingRateSource(value any) string {
	if source, ok := value.(string); ok && strings.EqualFold(strings.TrimSpace(source), SchedulingRateSourceUpstream) {
		return SchedulingRateSourceUpstream
	}
	return SchedulingRateSourceManual
}

func normalizeSchedulingRateSyncMode(value any) string {
	if mode, ok := value.(string); ok && strings.EqualFold(strings.TrimSpace(mode), SchedulingRateSyncModeManualLock) {
		return SchedulingRateSyncModeManualLock
	}
	return SchedulingRateSyncModeAutoOverwrite
}

// SchedulingRateSyncMode returns the account's automatic-rate overwrite
// policy. Missing values default to automatic overwrite. The legacy source is
// interpreted only when the new field is absent so existing choices migrate
// without a database rewrite.
func (a *Account) SchedulingRateSyncMode() string {
	if a == nil || a.Extra == nil {
		return SchedulingRateSyncModeAutoOverwrite
	}
	if value, ok := a.Extra[SchedulingRateSyncModeExtraKey]; ok {
		return normalizeSchedulingRateSyncMode(value)
	}
	if source, ok := a.Extra[SchedulingRateSourceExtraKey]; ok {
		if normalizeSchedulingRateSource(source) == SchedulingRateSourceManual {
			return SchedulingRateSyncModeManualLock
		}
	}
	return SchedulingRateSyncModeAutoOverwrite
}

// SchedulingRate returns the persisted scheduling multiplier. Upstream probe
// freshness never changes request-time ranking; successful automatic probes
// first copy the stable declared rate into accounts.rate_multiplier.
func (a *Account) SchedulingRate(_ time.Time) (rate float64, known bool, source string) {
	if a.SchedulingRateSyncMode() == SchedulingRateSyncModeAutoOverwrite {
		source = SchedulingRateSourceUpstream
	} else {
		source = SchedulingRateSourceManual
	}
	return a.BillingRateMultiplier(), true, source
}

func normalizeAccountSchedulingStrategy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AccountSchedulingStrategyLowestCost:
		return AccountSchedulingStrategyLowestCost
	default:
		return AccountSchedulingStrategyDefault
	}
}

func accountSchedulingStrategy(cfg *config.Config) string {
	if cfg == nil {
		return AccountSchedulingStrategyDefault
	}
	return normalizeAccountSchedulingStrategy(cfg.SuperPriority.BaseStrategy)
}

func superPrioritySchedulingActive(cfg *config.Config) bool {
	// The overlay is retired. Keep the helper while compatibility endpoints and
	// historical config fields still exist, but never let the old marker affect
	// request routing.
	return false
}

func accountHasSuperPriority(account *Account) bool {
	return account != nil && getExtraBool(account.Extra, SuperPriorityExtraKey)
}

func usesCustomAccountSchedulingPreference(cfg *config.Config) bool {
	return accountSchedulingStrategy(cfg) == AccountSchedulingStrategyLowestCost
}

func movableSessionStickyAllowed(cfg *config.Config) bool {
	return accountSchedulingStrategy(cfg) != AccountSchedulingStrategyLowestCost
}

func accountAllowedBySchedulingLiveness(account *Account, cfg *config.Config) bool {
	return accountSchedulingStrategy(cfg) != AccountSchedulingStrategyLowestCost || !accountSchedulingLivenessDead(account)
}

// filterByAccountSchedulingPreference returns the currently preferred strict
// tier. Callers remove failed/full candidates and invoke it again, which makes
// the next super-priority or price tier the natural fallback.
func filterByAccountSchedulingPreference(accounts []accountWithLoad, cfg *config.Config) []accountWithLoad {
	preferred := accounts
	if accountSchedulingStrategy(cfg) != AccountSchedulingStrategyLowestCost || len(preferred) < 2 {
		if len(preferred) == 1 && accountSchedulingStrategy(cfg) == AccountSchedulingStrategyLowestCost && accountSchedulingLivenessDead(preferred[0].account) {
			return nil
		}
		return preferred
	}
	alive := make([]accountWithLoad, 0, len(preferred))
	for _, item := range preferred {
		if !accountSchedulingLivenessDead(item.account) {
			alive = append(alive, item)
		}
	}
	preferred = alive
	if len(preferred) < 2 {
		return preferred
	}
	cheapest := make([]accountWithLoad, 0, len(preferred))
	var minRate float64
	knownRate := false
	for _, item := range preferred {
		rate := item.account.BillingRateMultiplier()
		if !knownRate || rate < minRate {
			minRate, knownRate = rate, true
		}
	}
	if !knownRate {
		return preferred
	}
	for _, item := range preferred {
		if item.account.BillingRateMultiplier() == minRate {
			cheapest = append(cheapest, item)
		}
	}
	return cheapest
}

// orderAccountsBySchedulingPreference is applied after the existing stable
// default ordering, so it only adds the strict outer tiers and preserves all
// original tie-breaking within a tier.
func orderAccountsBySchedulingPreference(accounts []*Account, cfg *config.Config) {
	if len(accounts) < 2 || !usesCustomAccountSchedulingPreference(cfg) {
		return
	}
	now := time.Now()
	sort.SliceStable(accounts, func(i, j int) bool {
		return compareAccountSchedulingPreferenceAt(accounts[i], accounts[j], cfg, now) < 0
	})
}

func orderAccountLoadsBySchedulingPreference(accounts []accountWithLoad, cfg *config.Config) {
	if len(accounts) < 2 || !usesCustomAccountSchedulingPreference(cfg) {
		return
	}
	now := time.Now()
	sort.SliceStable(accounts, func(i, j int) bool {
		return compareAccountSchedulingPreferenceAt(accounts[i].account, accounts[j].account, cfg, now) < 0
	})
}

func compareAccountSchedulingPreference(a, b *Account, cfg *config.Config) int {
	return compareAccountSchedulingPreferenceAt(a, b, cfg, time.Now())
}

func compareAccountSchedulingPreferenceAt(a, b *Account, cfg *config.Config, now time.Time) int {
	if a == nil || b == nil {
		switch {
		case a == nil && b != nil:
			return 1
		case a != nil && b == nil:
			return -1
		default:
			return 0
		}
	}
	if accountSchedulingStrategy(cfg) == AccountSchedulingStrategyLowestCost {
		aDead, bDead := accountSchedulingLivenessDeadAt(a, now), accountSchedulingLivenessDeadAt(b, now)
		if aDead != bDead {
			if aDead {
				return 1
			}
			return -1
		}
		aRate, bRate := a.BillingRateMultiplier(), b.BillingRateMultiplier()
		switch {
		case aRate < bRate:
			return -1
		case aRate > bRate:
			return 1
		}
	}
	return 0
}
