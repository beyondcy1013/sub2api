package service

import (
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

var lowestCostConnectionShareSequence atomic.Uint64

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

// filterByAccountSchedulingPreference returns the currently preferred strict
// tier. Callers remove failed/full candidates and invoke it again, which makes
// the next price tier the natural fallback.
func filterByAccountSchedulingPreference(accounts []accountWithLoad, cfg *config.Config) []accountWithLoad {
	preferred := accounts
	if accountSchedulingStrategy(cfg) != AccountSchedulingStrategyLowestCost || len(preferred) < 2 {
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
	rotateEqualRateAccounts(accounts)
}

func orderAccountLoadsBySchedulingPreference(accounts []accountWithLoad, cfg *config.Config) {
	if len(accounts) < 2 || !usesCustomAccountSchedulingPreference(cfg) {
		return
	}
	now := time.Now()
	sort.SliceStable(accounts, func(i, j int) bool {
		if preference := compareAccountSchedulingPreferenceAt(accounts[i].account, accounts[j].account, cfg, now); preference != 0 {
			return preference < 0
		}
		return compareAccountConnectionLoad(accounts[i].loadInfo, accounts[j].loadInfo) < 0
	})
	rotateEqualRateAccountLoads(accounts, cfg, now)
}

func compareAccountConnectionLoad(a, b *AccountLoadInfo) int {
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
	for _, values := range [][2]int{
		{a.LoadRate, b.LoadRate},
		{a.CurrentConcurrency, b.CurrentConcurrency},
		{a.WaitingCount, b.WaitingCount},
	} {
		switch {
		case values[0] < values[1]:
			return -1
		case values[0] > values[1]:
			return 1
		}
	}
	return 0
}

func rotateEqualRateAccounts(accounts []*Account) {
	for start := 0; start < len(accounts); {
		end := start + 1
		for end < len(accounts) && accounts[start].BillingRateMultiplier() == accounts[end].BillingRateMultiplier() {
			end++
		}
		if end-start > 1 {
			rotateAccounts(accounts[start:end], lowestCostConnectionShareSequence.Add(1)-1)
		}
		start = end
	}
}

func rotateEqualRateAccountLoads(accounts []accountWithLoad, cfg *config.Config, now time.Time) {
	for start := 0; start < len(accounts); {
		end := start + 1
		for end < len(accounts) &&
			compareAccountSchedulingPreferenceAt(accounts[start].account, accounts[end].account, cfg, now) == 0 &&
			compareAccountConnectionLoad(accounts[start].loadInfo, accounts[end].loadInfo) == 0 {
			end++
		}
		if end-start > 1 {
			rotateAccountLoads(accounts[start:end], lowestCostConnectionShareSequence.Add(1)-1)
		}
		start = end
	}
}

func rotateAccounts(accounts []*Account, sequence uint64) {
	offset := int(sequence % uint64(len(accounts)))
	if offset == 0 {
		return
	}
	copyOfGroup := append([]*Account(nil), accounts...)
	copy(accounts, copyOfGroup[offset:])
	copy(accounts[len(accounts)-offset:], copyOfGroup[:offset])
}

func rotateAccountLoads(accounts []accountWithLoad, sequence uint64) {
	offset := int(sequence % uint64(len(accounts)))
	if offset == 0 {
		return
	}
	copyOfGroup := append([]accountWithLoad(nil), accounts...)
	copy(accounts, copyOfGroup[offset:])
	copy(accounts[len(accounts)-offset:], copyOfGroup[:offset])
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
