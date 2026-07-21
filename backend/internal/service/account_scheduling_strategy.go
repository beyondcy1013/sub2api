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

	// SchedulingRateSourceExtraKey selects the per-account source used by the
	// lowest-cost scheduler. Missing values retain the historical manual-rate
	// behavior for existing accounts.
	SchedulingRateSourceExtraKey = "scheduling_rate_source"
	SchedulingRateSourceManual   = "manual"
	SchedulingRateSourceUpstream = "upstream"
)

func normalizeSchedulingRateSource(value any) string {
	if source, ok := value.(string); ok && strings.EqualFold(strings.TrimSpace(source), SchedulingRateSourceUpstream) {
		return SchedulingRateSourceUpstream
	}
	return SchedulingRateSourceManual
}

// SchedulingRate returns the account's request-time scheduling rate. Unknown
// upstream data is deliberately represented as unknown instead of falling back
// to the manual value: an operator who selected "follow upstream" must not
// accidentally make a stale declaration look cheap.
func (a *Account) SchedulingRate(now time.Time) (rate float64, known bool, source string) {
	source = SchedulingRateSourceManual
	if a != nil && a.Extra != nil {
		source = normalizeSchedulingRateSource(a.Extra[SchedulingRateSourceExtraKey])
	}
	if source != SchedulingRateSourceUpstream {
		return a.BillingRateMultiplier(), true, SchedulingRateSourceManual
	}
	if now.IsZero() {
		now = time.Now()
	}
	snapshot := decodeUpstreamBillingProbeSnapshot(a.Extra)
	if snapshot == nil || (snapshot.Status != UpstreamBillingProbeStatusOK && snapshot.Status != UpstreamBillingProbeStatusFailed) ||
		snapshot.ReceivedAt == nil || snapshot.FreshUntil == nil || now.Before(*snapshot.ReceivedAt) || !now.Before(*snapshot.FreshUntil) {
		return 0, false, SchedulingRateSourceUpstream
	}
	rate, known = upstreamBillingRateAt(snapshot.Data, now)
	if !known {
		return 0, false, SchedulingRateSourceUpstream
	}
	return rate, true, SchedulingRateSourceUpstream
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
	return cfg != nil && strings.TrimSpace(cfg.SuperPriority.Mode) == superPriorityModeSuperPriority
}

func accountHasSuperPriority(account *Account) bool {
	return account != nil && getExtraBool(account.Extra, SuperPriorityExtraKey)
}

func usesCustomAccountSchedulingPreference(cfg *config.Config) bool {
	return superPrioritySchedulingActive(cfg) || accountSchedulingStrategy(cfg) == AccountSchedulingStrategyLowestCost
}

func movableSessionStickyAllowed(cfg *config.Config) bool {
	return accountSchedulingStrategy(cfg) != AccountSchedulingStrategyLowestCost
}

// filterByAccountSchedulingPreference returns the currently preferred strict
// tier. Callers remove failed/full candidates and invoke it again, which makes
// the next super-priority or price tier the natural fallback.
func filterByAccountSchedulingPreference(accounts []accountWithLoad, cfg *config.Config) []accountWithLoad {
	preferred := accounts
	if superPrioritySchedulingActive(cfg) {
		super := make([]accountWithLoad, 0, len(preferred))
		for _, item := range preferred {
			if accountHasSuperPriority(item.account) {
				super = append(super, item)
			}
		}
		if len(super) > 0 {
			preferred = super
		}
	}

	if accountSchedulingStrategy(cfg) != AccountSchedulingStrategyLowestCost || len(preferred) < 2 {
		return preferred
	}
	now := time.Now()
	cheapest := make([]accountWithLoad, 0, len(preferred))
	var minRate float64
	knownRate := false
	for _, item := range preferred {
		rate, known, _ := item.account.SchedulingRate(now)
		if !known {
			continue
		}
		if !knownRate || rate < minRate {
			minRate, knownRate = rate, true
		}
	}
	if !knownRate {
		return preferred
	}
	for _, item := range preferred {
		rate, known, _ := item.account.SchedulingRate(now)
		if known && rate == minRate {
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
	if superPrioritySchedulingActive(cfg) {
		aSuper, bSuper := accountHasSuperPriority(a), accountHasSuperPriority(b)
		if aSuper != bSuper {
			if aSuper {
				return -1
			}
			return 1
		}
	}
	if accountSchedulingStrategy(cfg) == AccountSchedulingStrategyLowestCost {
		aRate, aKnown, _ := a.SchedulingRate(now)
		bRate, bKnown, _ := b.SchedulingRate(now)
		switch {
		case aKnown != bKnown:
			if aKnown {
				return -1
			}
			return 1
		case !aKnown && !bKnown:
			return 0
		case aRate < bRate:
			return -1
		case aRate > bRate:
			return 1
		}
	}
	return 0
}
