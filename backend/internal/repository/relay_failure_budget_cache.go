package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const relayFailureBudgetKeyPrefix = "openai_relay_failure_budget:account:"

var recordRelayFailureBudgetEventScript = redis.NewScript(`
local buckets_key = KEYS[1]
local index_key = KEYS[2]

local now_ms = tonumber(ARGV[1])
local bucket = ARGV[2]
local bucket_score = tonumber(ARGV[2])
local window_start = tonumber(ARGV[3])
local outcome = ARGV[4]
local threshold_percent = tonumber(ARGV[5])
local min_requests = tonumber(ARGV[6])
local consecutive_threshold = tonumber(ARGV[7])
local cooldown_ms = tonumber(ARGV[8])
local ttl_seconds = tonumber(ARGV[9])

local cooldown_until = tonumber(redis.call('HGET', buckets_key, 'cooldown_until') or '0')

if outcome == 'success' and cooldown_until > 0 and cooldown_until <= now_ms then
  local prior_buckets = redis.call('ZRANGE', index_key, 0, -1)
  for _, prior_bucket in ipairs(prior_buckets) do
    redis.call('HDEL', buckets_key, 's:' .. prior_bucket, 'f:' .. prior_bucket)
  end
  redis.call('DEL', index_key)
  redis.call('HDEL', buckets_key, 'consecutive_failures', 'cooldown_until')
  cooldown_until = 0
end

local expired_buckets = redis.call('ZRANGEBYSCORE', index_key, '-inf', '(' .. window_start)
for _, expired_bucket in ipairs(expired_buckets) do
  redis.call('HDEL', buckets_key, 's:' .. expired_bucket, 'f:' .. expired_bucket)
end
redis.call('ZREMRANGEBYSCORE', index_key, '-inf', '(' .. window_start)

redis.call('ZADD', index_key, bucket_score, bucket)
local consecutive_failures = tonumber(redis.call('HGET', buckets_key, 'consecutive_failures') or '0')
if outcome == 'success' then
  redis.call('HINCRBY', buckets_key, 's:' .. bucket, 1)
  consecutive_failures = 0
  redis.call('HSET', buckets_key, 'consecutive_failures', 0)
else
  redis.call('HINCRBY', buckets_key, 'f:' .. bucket, 1)
  consecutive_failures = redis.call('HINCRBY', buckets_key, 'consecutive_failures', 1)
end

local requests = 0
local failures = 0
local active_buckets = redis.call('ZRANGE', index_key, 0, -1)
for _, active_bucket in ipairs(active_buckets) do
  requests = requests + tonumber(redis.call('HGET', buckets_key, 's:' .. active_bucket) or '0')
  local bucket_failures = tonumber(redis.call('HGET', buckets_key, 'f:' .. active_bucket) or '0')
  requests = requests + bucket_failures
  failures = failures + bucket_failures
end

local tripped = 0
local returned_cooldown_until = 0
if cooldown_until > now_ms then
  tripped = 1
  returned_cooldown_until = cooldown_until
else
  local ratio_tripped = requests >= min_requests and failures * 100 >= threshold_percent * requests
  local consecutive_tripped = consecutive_failures >= consecutive_threshold
  if ratio_tripped or consecutive_tripped then
    tripped = 1
    cooldown_until = now_ms + cooldown_ms
    returned_cooldown_until = cooldown_until
    redis.call('HSET', buckets_key, 'cooldown_until', cooldown_until)
  end
end

redis.call('EXPIRE', buckets_key, ttl_seconds)
redis.call('EXPIRE', index_key, ttl_seconds)

return {requests, failures, consecutive_failures, tripped, returned_cooldown_until}
`)

type relayFailureBudgetCache struct {
	rdb *redis.Client
}

func NewRelayFailureBudgetCache(rdb *redis.Client) service.RelayFailureBudgetCache {
	return &relayFailureBudgetCache{rdb: rdb}
}

func (c *relayFailureBudgetCache) RecordRelayFailureBudgetEvent(
	ctx context.Context,
	accountID int64,
	at time.Time,
	outcome service.RelayFailureBudgetOutcome,
	policy service.RelayFailureBudgetPolicy,
) (service.RelayFailureBudgetDecision, error) {
	if c == nil || c.rdb == nil {
		return service.RelayFailureBudgetDecision{}, fmt.Errorf("relay failure budget cache is unavailable")
	}
	if outcome != service.RelayFailureBudgetSuccess && outcome != service.RelayFailureBudgetFailure {
		return service.RelayFailureBudgetDecision{}, fmt.Errorf("unsupported relay failure budget outcome %q", outcome)
	}
	policy = normalizeRelayFailureBudgetPolicy(policy)
	bucketStart := at.UTC().Truncate(time.Minute).Unix()
	windowMinutes := int64(policy.Window / time.Minute)
	windowStart := bucketStart - (windowMinutes-1)*60
	ttl := policy.Window + policy.Cooldown + 2*time.Minute
	keyBase := fmt.Sprintf("%s%d", relayFailureBudgetKeyPrefix, accountID)

	result, err := recordRelayFailureBudgetEventScript.Run(
		ctx,
		c.rdb,
		[]string{keyBase + ":buckets", keyBase + ":bucket_index"},
		at.UnixMilli(),
		bucketStart,
		windowStart,
		string(outcome),
		policy.FailureThresholdPercent,
		policy.MinRequests,
		policy.ConsecutiveFailures,
		policy.Cooldown.Milliseconds(),
		int64(ttl/time.Second),
	).Int64Slice()
	if err != nil {
		return service.RelayFailureBudgetDecision{}, fmt.Errorf("record relay failure budget event: %w", err)
	}
	if len(result) != 5 {
		return service.RelayFailureBudgetDecision{}, fmt.Errorf("record relay failure budget event: unexpected result length %d", len(result))
	}

	decision := service.RelayFailureBudgetDecision{
		WindowRequests:      int(result[0]),
		WindowFailures:      int(result[1]),
		ConsecutiveFailures: int(result[2]),
		Tripped:             result[3] == 1,
	}
	if result[4] > 0 {
		decision.CooldownUntil = time.UnixMilli(result[4]).UTC()
	}
	return decision, nil
}

func normalizeRelayFailureBudgetPolicy(policy service.RelayFailureBudgetPolicy) service.RelayFailureBudgetPolicy {
	if policy.Window < time.Minute {
		policy.Window = time.Minute
	}
	if policy.FailureThresholdPercent < 1 || policy.FailureThresholdPercent > 100 {
		policy.FailureThresholdPercent = 100
	}
	if policy.MinRequests < 1 {
		policy.MinRequests = 1
	}
	if policy.ConsecutiveFailures < 1 {
		policy.ConsecutiveFailures = 1
	}
	if policy.Cooldown < time.Minute {
		policy.Cooldown = time.Minute
	}
	return policy
}
