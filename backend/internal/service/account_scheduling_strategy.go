package service

import (
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	AccountSchedulingStrategyDefault    = "default"
	AccountSchedulingStrategyLowestCost = "lowest_cost"
)

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
	minRate := preferred[0].account.BillingRateMultiplier()
	for _, item := range preferred[1:] {
		if rate := item.account.BillingRateMultiplier(); rate < minRate {
			minRate = rate
		}
	}
	cheapest := make([]accountWithLoad, 0, len(preferred))
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
	sort.SliceStable(accounts, func(i, j int) bool {
		return compareAccountSchedulingPreference(accounts[i], accounts[j], cfg) < 0
	})
}

func orderAccountLoadsBySchedulingPreference(accounts []accountWithLoad, cfg *config.Config) {
	if len(accounts) < 2 || !usesCustomAccountSchedulingPreference(cfg) {
		return
	}
	sort.SliceStable(accounts, func(i, j int) bool {
		return compareAccountSchedulingPreference(accounts[i].account, accounts[j].account, cfg) < 0
	})
}

func compareAccountSchedulingPreference(a, b *Account, cfg *config.Config) int {
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
