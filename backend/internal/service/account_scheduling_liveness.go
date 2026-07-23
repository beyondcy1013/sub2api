package service

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	SchedulingLivenessExtraKey              = "scheduling_liveness"
	SchedulingLivenessStatusManagedExtraKey = "status_managed"
	SchedulingLivenessStatusUnknown         = "unknown"
	SchedulingLivenessStatusAlive           = "alive"
	SchedulingLivenessStatusSuspect         = "suspect"
	SchedulingLivenessStatusDead            = "dead"
	SchedulingLivenessErrorPrefix           = "Scheduling liveness probe failed: "
)

// AccountSchedulingLiveness stores probe history. The status_managed marker is
// only an ownership marker for automatic recovery; it is not a routing signal.
type AccountSchedulingLiveness struct {
	Status        string     `json:"status"`
	StatusManaged bool       `json:"status_managed,omitempty"`
	FailureCount  int        `json:"failure_count"`
	LastAttemptAt time.Time  `json:"last_attempt_at"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	FreshUntil    time.Time  `json:"fresh_until"`
	LastError     string     `json:"last_error,omitempty"`
}

func schedulingLivenessErrorMessage(errorMessage string) string {
	errorMessage = strings.TrimSpace(errorMessage)
	if errorMessage == "" {
		return strings.TrimSpace(SchedulingLivenessErrorPrefix)
	}
	return SchedulingLivenessErrorPrefix + errorMessage
}

func schedulingLivenessOwnsAccountStatus(account *Account) bool {
	if account == nil || account.Status != StatusError {
		return false
	}
	snapshot := decodeSchedulingLiveness(account.Extra)
	return snapshot != nil && snapshot.StatusManaged &&
		strings.HasPrefix(strings.TrimSpace(account.ErrorMessage), strings.TrimSpace(SchedulingLivenessErrorPrefix))
}

func schedulingLivenessProbeEligible(account *Account) bool {
	if account == nil {
		return false
	}
	return account.Status == StatusActive || schedulingLivenessOwnsAccountStatus(account)
}

func decodeSchedulingLiveness(extra map[string]any) *AccountSchedulingLiveness {
	if extra == nil {
		return nil
	}
	value, ok := extra[SchedulingLivenessExtraKey]
	if !ok || value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var snapshot AccountSchedulingLiveness
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(snapshot.Status)) {
	case SchedulingLivenessStatusAlive, SchedulingLivenessStatusSuspect, SchedulingLivenessStatusDead:
		return &snapshot
	default:
		return nil
	}
}

func accountSchedulingLivenessDead(account *Account) bool {
	return accountSchedulingLivenessDeadAt(account, time.Now())
}

func accountSchedulingLivenessDeadAt(account *Account, now time.Time) bool {
	if account == nil {
		return false
	}
	snapshot := decodeSchedulingLiveness(account.Extra)
	if snapshot == nil || snapshot.Status != SchedulingLivenessStatusDead {
		return false
	}
	// A stale observation becomes unknown instead of permanently excluding an
	// account when the probe runner has been interrupted.
	return !snapshot.FreshUntil.IsZero() && now.Before(snapshot.FreshUntil)
}

func (a *Account) SchedulingLivenessStatus(now time.Time) string {
	snapshot := decodeSchedulingLiveness(a.Extra)
	if snapshot == nil || (!snapshot.FreshUntil.IsZero() && !now.Before(snapshot.FreshUntil)) {
		return SchedulingLivenessStatusUnknown
	}
	return snapshot.Status
}

func nextSchedulingLiveness(
	previous *AccountSchedulingLiveness,
	now time.Time,
	freshUntil time.Time,
	succeeded bool,
	errorMessage string,
	failureThreshold int,
) *AccountSchedulingLiveness {
	if failureThreshold < 1 {
		failureThreshold = 1
	}
	next := &AccountSchedulingLiveness{
		Status:        SchedulingLivenessStatusAlive,
		LastAttemptAt: now,
		FreshUntil:    freshUntil,
	}
	if previous != nil {
		next.LastSuccessAt = previous.LastSuccessAt
	}
	if succeeded {
		successAt := now
		next.LastSuccessAt = &successAt
		return next
	}

	next.Status = SchedulingLivenessStatusSuspect
	next.FailureCount = 1
	next.LastError = strings.TrimSpace(errorMessage)
	if previous != nil {
		next.FailureCount = previous.FailureCount + 1
	}
	if next.FailureCount >= failureThreshold {
		next.Status = SchedulingLivenessStatusDead
	}
	return next
}
