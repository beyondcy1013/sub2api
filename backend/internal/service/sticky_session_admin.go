package service

import (
	"context"
	"time"
)

type StickySessionBindingActivity struct {
	SessionHash  string
	ActiveAgo    time.Duration
	RemainingTTL time.Duration
}

// StickySessionBindingSummary describes live, movable session_hash bindings.
// previous_response_id bindings are counted separately and never moved here.
type StickySessionBindingSummary struct {
	Counts                    map[int64]int
	Activities                map[int64][]StickySessionBindingActivity
	Total                     int
	ProtectedResponseBindings int
}

type StickySessionReassignResult struct {
	Moved                   int `json:"moved"`
	RemainingSourceBindings int `json:"remaining_source_bindings"`
}

// StickySessionAdminStore exposes the narrow Redis operations needed by the
// account-management UI. It is intentionally separate from the hot-path cache.
type StickySessionAdminStore interface {
	Summarize(ctx context.Context, groupID int64, platform string) (*StickySessionBindingSummary, error)
	Reassign(ctx context.Context, groupID int64, platform string, sourceAccountID, targetAccountID int64, count int, activeWithin time.Duration) (*StickySessionReassignResult, error)
}
