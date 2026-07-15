package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const stickySessionAdminScanLimit = 100000

var stickySessionCompareAndSet = redis.NewScript(`
if redis.call("GET", KEYS[1]) ~= ARGV[1] then
  return 0
end
redis.call("SET", KEYS[1], ARGV[2], "KEEPTTL")
return 1
`)

type stickySessionAdminStore struct {
	rdb                 *redis.Client
	sessionTTL          time.Duration
	beforeCompareAndSet func()
}

func NewStickySessionAdminStore(rdb *redis.Client) *stickySessionAdminStore {
	return &stickySessionAdminStore{rdb: rdb, sessionTTL: time.Hour}
}

func ProvideStickySessionAdminStore(rdb *redis.Client, cfg *config.Config) *stickySessionAdminStore {
	store := NewStickySessionAdminStore(rdb)
	if cfg != nil && cfg.Gateway.OpenAIWS.StickySessionTTLSeconds > 0 {
		store.sessionTTL = time.Duration(cfg.Gateway.OpenAIWS.StickySessionTTLSeconds) * time.Second
	}
	return store
}

type stickySessionAdminBinding struct {
	key         string
	sessionHash string
	accountID   int64
	ttl         time.Duration
	activeAgo   time.Duration
}

func validateStickySessionNamespace(groupID int64, platform string) (string, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if groupID <= 0 {
		return "", errors.New("group id must be positive")
	}
	if platform == "" {
		return "", errors.New("platform is required")
	}
	for _, r := range platform {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return "", errors.New("invalid platform")
		}
	}
	return fmt.Sprintf("%s%d:%s:", stickySessionPrefix, groupID, platform), nil
}

func isProtectedResponseBinding(prefix, key string) bool {
	return strings.HasPrefix(strings.TrimPrefix(key, prefix), "response:")
}

func isCurrentOpenAISessionBinding(prefix, key string) bool {
	tail := strings.TrimPrefix(key, prefix)
	if len(tail) != 16 {
		return false
	}
	for _, r := range tail {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func (s *stickySessionAdminStore) scan(ctx context.Context, groupID int64, platform string) ([]stickySessionAdminBinding, int, error) {
	if s == nil || s.rdb == nil {
		return nil, 0, errors.New("sticky session store is unavailable")
	}
	prefix, err := validateStickySessionNamespace(groupID, platform)
	if err != nil {
		return nil, 0, err
	}

	bindings := make([]stickySessionAdminBinding, 0)
	protected := 0
	scanned := 0
	var cursor uint64
	for {
		keys, next, scanErr := s.rdb.Scan(ctx, cursor, prefix+"*", 500).Result()
		if scanErr != nil {
			return nil, 0, scanErr
		}
		for _, key := range keys {
			scanned++
			if scanned > stickySessionAdminScanLimit {
				return nil, 0, fmt.Errorf("sticky session scan exceeds %d keys", stickySessionAdminScanLimit)
			}
			if isProtectedResponseBinding(prefix, key) {
				protected++
				continue
			}
			// Current OpenAI session hashes are 16 lowercase hex characters. Ignore
			// 64-character legacy dual-write keys so one client is not counted twice.
			if !isCurrentOpenAISessionBinding(prefix, key) {
				continue
			}
			accountID, getErr := s.rdb.Get(ctx, key).Int64()
			if getErr != nil {
				if errors.Is(getErr, redis.Nil) {
					continue
				}
				return nil, 0, getErr
			}
			ttl, ttlErr := s.rdb.PTTL(ctx, key).Result()
			if ttlErr != nil && !errors.Is(ttlErr, redis.Nil) {
				return nil, 0, ttlErr
			}
			if ttl <= 0 {
				continue
			}
			activeAgo := s.sessionTTL - ttl
			if activeAgo < 0 {
				activeAgo = 0
			}
			bindings = append(bindings, stickySessionAdminBinding{
				key: key, sessionHash: strings.TrimPrefix(key, prefix), accountID: accountID,
				ttl: ttl, activeAgo: activeAgo,
			})
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return bindings, protected, nil
}

func (s *stickySessionAdminStore) Summarize(ctx context.Context, groupID int64, platform string) (*service.StickySessionBindingSummary, error) {
	bindings, protected, err := s.scan(ctx, groupID, platform)
	if err != nil {
		return nil, err
	}
	result := &service.StickySessionBindingSummary{
		Counts:                    make(map[int64]int),
		Activities:                make(map[int64][]service.StickySessionBindingActivity),
		Total:                     len(bindings),
		ProtectedResponseBindings: protected,
	}
	for _, binding := range bindings {
		result.Counts[binding.accountID]++
		result.Activities[binding.accountID] = append(result.Activities[binding.accountID], service.StickySessionBindingActivity{
			SessionHash: binding.sessionHash, ActiveAgo: binding.activeAgo, RemainingTTL: binding.ttl,
		})
	}
	for accountID := range result.Activities {
		sort.SliceStable(result.Activities[accountID], func(i, j int) bool {
			left := result.Activities[accountID][i]
			right := result.Activities[accountID][j]
			if left.ActiveAgo != right.ActiveAgo {
				return left.ActiveAgo < right.ActiveAgo
			}
			return left.SessionHash < right.SessionHash
		})
	}
	return result, nil
}

func (s *stickySessionAdminStore) Reassign(ctx context.Context, groupID int64, platform string, sourceAccountID, targetAccountID int64, count int, activeWithin time.Duration) (*service.StickySessionReassignResult, error) {
	if sourceAccountID <= 0 || targetAccountID <= 0 || sourceAccountID == targetAccountID {
		return nil, errors.New("source and target accounts must be different positive ids")
	}
	if count <= 0 || count > 100 {
		return nil, errors.New("count must be between 1 and 100")
	}
	if activeWithin <= 0 {
		return nil, errors.New("active window must be positive")
	}

	bindings, _, err := s.scan(ctx, groupID, platform)
	if err != nil {
		return nil, err
	}
	candidates := make([]stickySessionAdminBinding, 0)
	for _, binding := range bindings {
		if binding.accountID == sourceAccountID && binding.activeAgo <= activeWithin {
			candidates = append(candidates, binding)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].activeAgo != candidates[j].activeAgo {
			return candidates[i].activeAgo < candidates[j].activeAgo
		}
		return candidates[i].key < candidates[j].key
	})
	if len(candidates) > count {
		candidates = candidates[:count]
	}

	if s.beforeCompareAndSet != nil {
		s.beforeCompareAndSet()
	}
	moved := 0
	for _, candidate := range candidates {
		changed, runErr := stickySessionCompareAndSet.Run(
			ctx,
			s.rdb,
			[]string{candidate.key},
			fmt.Sprintf("%d", sourceAccountID),
			fmt.Sprintf("%d", targetAccountID),
		).Int()
		if runErr != nil {
			return nil, runErr
		}
		if changed == 1 {
			moved++
		}
	}

	remaining := 0
	after, _, err := s.scan(ctx, groupID, platform)
	if err != nil {
		return nil, err
	}
	for _, binding := range after {
		if binding.accountID == sourceAccountID {
			remaining++
		}
	}
	return &service.StickySessionReassignResult{Moved: moved, RemainingSourceBindings: remaining}, nil
}

var _ service.StickySessionAdminStore = (*stickySessionAdminStore)(nil)
