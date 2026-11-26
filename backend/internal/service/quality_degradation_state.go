package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// QualityDegradationStateExtraKey stores the consecutive request-probe
// failures that are used by the optional scheduling guard.
const QualityDegradationStateExtraKey = "quality_degradation_state"

// ErrQualityDegradationPaused is returned when an automatic recovery path
// attempts to re-enable an account that was paused by the request-quality
// guard. An operator must explicitly clear that guard first.
var ErrQualityDegradationPaused = errors.New("account paused by quality degradation guard")

// QualityDegradationState is deliberately separate from pelican_downgraded.
// The latter is a latest-result marker and must not clear a scheduling pause.
type QualityDegradationState struct {
	ConsecutiveCount int
	RuleFingerprint  string
	LastModel        string
	LastReason       string
	Threshold        int
	Paused           bool
}

// AccountQualityDegradationStateStore is an optional repository capability.
// Keeping it separate from AccountRepository avoids forcing every read-only
// test double to implement the atomic quality-state write.
type AccountQualityDegradationStateStore interface {
	RecordQualityDegradation(
		context.Context,
		int64,
		string,
		string,
		string,
		int,
		bool,
	) (QualityDegradationState, error)
	ResetQualityDegradation(context.Context, int64, string) error
	ClearQualityDegradationState(context.Context, int64) error
}

// AccountQualityDegradationOrderedResetStore is an optional extension used by
// request-triggered probes.  A clean result carries the time at which that
// probe finished; the repository can ignore a late persistence attempt when a
// newer degradation result has already been recorded.
type AccountQualityDegradationOrderedResetStore interface {
	ResetQualityDegradationAt(context.Context, int64, string, time.Time) error
}

// AccountQualityDegradationRecoveryStore atomically clears the quality pause
// and re-enables scheduling. It is optional so narrow repository test doubles
// and compatibility adapters can keep the older two-step behavior.
type AccountQualityDegradationRecoveryStore interface {
	ClearQualityDegradationStateAndSetSchedulable(context.Context, int64) error
}

// IsQualityDegradationPaused reports whether an account extra map contains a
// persisted quality-guard pause. Repository update paths use this to avoid
// overwriting a concurrent request-probe decision with a stale account DTO.
func IsQualityDegradationPaused(extra map[string]any) bool {
	return qualityDegradationStateFromExtra(extra).Paused
}

func normalizedQualityDegradationThreshold(value int) int {
	if value < 1 {
		return 3
	}
	if value > 10 {
		return 10
	}
	return value
}

func qualityDegradationStateFromExtra(extra map[string]any) QualityDegradationState {
	state := QualityDegradationState{}
	if extra == nil {
		return state
	}
	raw, ok := extra[QualityDegradationStateExtraKey]
	if !ok {
		return state
	}
	// JSON-backed account payloads use map[string]any with float64 values;
	// accepting a marshalled value also keeps this helper safe for test fixtures.
	var fields map[string]any
	switch value := raw.(type) {
	case map[string]any:
		fields = value
	default:
		encoded, err := json.Marshal(value)
		if err != nil || json.Unmarshal(encoded, &fields) != nil {
			return state
		}
	}
	state.ConsecutiveCount = qualityStateInt(fields["consecutive_count"])
	state.RuleFingerprint, _ = fields["rule_fingerprint"].(string)
	state.LastModel, _ = fields["last_model"].(string)
	state.LastReason, _ = fields["last_reason"].(string)
	state.Threshold = qualityStateInt(fields["threshold"])
	state.Paused, _ = fields["paused"].(bool)
	return state
}

func qualityStateInt(value any) int {
	switch number := value.(type) {
	case int:
		return number
	case int8:
		return int(number)
	case int16:
		return int(number)
	case int32:
		return int(number)
	case int64:
		return int(number)
	case uint:
		return int(number)
	case uint64:
		return int(number)
	case float64:
		return int(number)
	case json.Number:
		parsed, _ := strconv.Atoi(number.String())
		return parsed
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(number))
		return parsed
	default:
		return 0
	}
}

func qualityDegradationStateExtra(state QualityDegradationState) map[string]any {
	return map[string]any{
		"consecutive_count": state.ConsecutiveCount,
		"rule_fingerprint":  state.RuleFingerprint,
		"last_model":        state.LastModel,
		"last_reason":       state.LastReason,
		"threshold":         state.Threshold,
		"paused":            state.Paused,
	}
}

func (s *RequestQualityProbeService) recordQualityDegradation(
	ctx context.Context,
	account *Account,
	model string,
	reason string,
	fingerprint string,
	threshold int,
	pauseOnDowngrade bool,
) (QualityDegradationState, error) {
	threshold = normalizedQualityDegradationThreshold(threshold)
	if account == nil || s == nil || s.accountRepo == nil {
		return QualityDegradationState{ConsecutiveCount: 1, RuleFingerprint: fingerprint, Threshold: threshold}, nil
	}
	if store, ok := s.accountRepo.(AccountQualityDegradationStateStore); ok {
		state, err := store.RecordQualityDegradation(ctx, account.ID, fingerprint, model, reason, threshold, pauseOnDowngrade)
		if err == nil {
			if account.Extra == nil {
				account.Extra = make(map[string]any)
			}
			stateExtra := qualityDegradationStateExtra(state)
			// The repository response intentionally contains only the fields
			// needed by the counter. Preserve diagnostic fields (notably
			// last_detected_at) already present on a caller-owned account object.
			if previous, ok := account.Extra[QualityDegradationStateExtraKey].(map[string]any); ok {
				for key, value := range previous {
					if _, exists := stateExtra[key]; !exists {
						stateExtra[key] = value
					}
				}
			}
			account.Extra[QualityDegradationStateExtraKey] = stateExtra
			if state.Paused {
				account.Schedulable = false
			}
		}
		return state, err
	}

	// Compatibility fallback for repository test doubles and older adapters.
	// Production uses the atomic repository capability above.
	state := qualityDegradationStateFromExtra(account.Extra)
	if state.RuleFingerprint != fingerprint {
		state.ConsecutiveCount = 0
	}
	state.ConsecutiveCount++
	state.RuleFingerprint = fingerprint
	state.LastModel = strings.TrimSpace(model)
	state.LastReason = strings.TrimSpace(reason)
	state.Threshold = threshold
	if pauseOnDowngrade && state.ConsecutiveCount >= threshold {
		state.Paused = true
		account.Schedulable = false
	}
	if account.Extra == nil {
		account.Extra = make(map[string]any)
	}
	account.Extra[QualityDegradationStateExtraKey] = qualityDegradationStateExtra(state)
	return state, s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{
		QualityDegradationStateExtraKey: qualityDegradationStateExtra(state),
	})
}

func (s *RequestQualityProbeService) resetQualityDegradation(ctx context.Context, account *Account, fingerprint string, observedAt time.Time) error {
	if account == nil || s == nil || s.accountRepo == nil {
		return nil
	}
	if store, ok := s.accountRepo.(AccountQualityDegradationOrderedResetStore); ok {
		return store.ResetQualityDegradationAt(ctx, account.ID, fingerprint, observedAt)
	}
	if store, ok := s.accountRepo.(AccountQualityDegradationStateStore); ok {
		return store.ResetQualityDegradation(ctx, account.ID, fingerprint)
	}
	state := qualityDegradationStateFromExtra(account.Extra)
	if state.RuleFingerprint == "" || state.RuleFingerprint == fingerprint {
		state.ConsecutiveCount = 0
		state.RuleFingerprint = fingerprint
		if account.Extra == nil {
			account.Extra = make(map[string]any)
		}
		account.Extra[QualityDegradationStateExtraKey] = qualityDegradationStateExtra(state)
		return s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{
			QualityDegradationStateExtraKey: qualityDegradationStateExtra(state),
		})
	}
	return nil
}

// ClearQualityDegradationState is used only by an explicit operator recovery.
// A clean probe intentionally calls resetQualityDegradation instead.
func ClearQualityDegradationState(ctx context.Context, repo AccountRepository, accountID int64) error {
	if repo == nil || accountID <= 0 {
		return nil
	}
	if store, ok := repo.(AccountQualityDegradationStateStore); ok {
		return store.ClearQualityDegradationState(ctx, accountID)
	}
	// Older adapters do not expose a safe state writer. Their explicit
	// schedulable toggle remains valid; the next production read will use the
	// atomic repository implementation when available.
	return nil
}

// ClearQualityDegradationStateAndSetSchedulable is the shared service helper
// for an explicit operator recovery. Production repositories implement the
// atomic capability; compatibility adapters fall back to the older sequence.
func ClearQualityDegradationStateAndSetSchedulable(ctx context.Context, repo AccountRepository, accountID int64) error {
	if repo == nil || accountID <= 0 {
		return nil
	}
	if store, ok := repo.(AccountQualityDegradationRecoveryStore); ok {
		return store.ClearQualityDegradationStateAndSetSchedulable(ctx, accountID)
	}
	if err := ClearQualityDegradationState(ctx, repo, accountID); err != nil {
		return err
	}
	return repo.SetSchedulable(ctx, accountID, true)
}

func (s QualityDegradationState) String() string {
	return fmt.Sprintf("%d/%d paused=%t", s.ConsecutiveCount, s.Threshold, s.Paused)
}
