package service

import (
	"context"
	"time"
)

// OpenAIModelRuntimeBlock describes an in-memory account+model cooldown that
// excludes only that model from scheduling. Unlike model_rate_limits it is not
// persisted, so it must be exposed through the admin API to remain visible.
type OpenAIModelRuntimeBlock struct {
	Model        string    `json:"model"`
	BlockedUntil time.Time `json:"blocked_until"`
}

// OpenAIRuntimeStatus exposes ephemeral request-time filters that the admin
// table otherwise cannot see because they are not stored in accounts.status,
// accounts.schedulable, rate_limit_reset_at, or extra.
type OpenAIRuntimeStatus struct {
	AccountBlockedUntil   *time.Time                `json:"account_blocked_until,omitempty"`
	AccountBlockedReason  string                    `json:"account_blocked_reason,omitempty"`
	ProxyQuarantinedUntil *time.Time                `json:"proxy_quarantined_until,omitempty"`
	ModelRuntimeBlocks    []OpenAIModelRuntimeBlock `json:"model_runtime_blocks,omitempty"`
}

// AccountRuntimeStatusProvider is implemented by OpenAIGatewayService so the
// admin handler can read ephemeral runtime state without coupling to the full
// gateway service.
type AccountRuntimeStatusProvider interface {
	OpenAIAccountRuntimeBlockUntil(accountID int64) (time.Time, string, bool)
	OpenAIModelRuntimeBlocks(accountID int64) []OpenAIModelRuntimeBlock
	OpenAIProxyStreamQuarantineUntil(proxyID int64) (time.Time, bool)
}

// OpenAIRuntimeStatus returns the active in-memory OpenAI scheduling filters
// for one account. It returns nil when no provider is wired or no filter is
// active.
func (s *RateLimitService) OpenAIRuntimeStatus(_ context.Context, account *Account) *OpenAIRuntimeStatus {
	if s == nil || account == nil || s.runtimeStatusProvider == nil {
		return nil
	}

	status := &OpenAIRuntimeStatus{}
	if until, reason, ok := s.runtimeStatusProvider.OpenAIAccountRuntimeBlockUntil(account.ID); ok {
		untilCopy := until
		status.AccountBlockedUntil = &untilCopy
		status.AccountBlockedReason = reason
	}
	if account.ProxyID != nil {
		if until, ok := s.runtimeStatusProvider.OpenAIProxyStreamQuarantineUntil(*account.ProxyID); ok {
			untilCopy := until
			status.ProxyQuarantinedUntil = &untilCopy
		}
	}
	if blocks := s.runtimeStatusProvider.OpenAIModelRuntimeBlocks(account.ID); len(blocks) > 0 {
		status.ModelRuntimeBlocks = blocks
	}
	if status.AccountBlockedUntil == nil && status.ProxyQuarantinedUntil == nil && len(status.ModelRuntimeBlocks) == 0 {
		return nil
	}
	return status
}
