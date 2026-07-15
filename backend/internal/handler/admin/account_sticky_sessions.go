package admin

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type stickySessionSourceResponse struct {
	AccountID          int64                           `json:"account_id"`
	AccountName        string                          `json:"account_name"`
	Count              int                             `json:"count"`
	CurrentConcurrency int                             `json:"current_concurrency"`
	Concurrency        int                             `json:"concurrency"`
	RecentCounts       map[string]int                  `json:"recent_counts"`
	RecentSessions     []stickySessionActivityResponse `json:"recent_sessions"`
}

type stickySessionActivityResponse struct {
	SessionSuffix    string `json:"session_suffix"`
	ActiveAgoSeconds int64  `json:"active_ago_seconds"`
}

var stickySessionActivityWindows = []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour}

func stickySessionWindowKey(window time.Duration) string {
	return strconv.FormatInt(int64(window/time.Second), 10)
}

func stickySessionSuffix(sessionHash string) string {
	if len(sessionHash) <= 8 {
		return sessionHash
	}
	return sessionHash[len(sessionHash)-8:]
}

func stickySessionRecentActivity(activities []service.StickySessionBindingActivity) (map[string]int, []stickySessionActivityResponse) {
	counts := make(map[string]int, len(stickySessionActivityWindows))
	for _, window := range stickySessionActivityWindows {
		counts[stickySessionWindowKey(window)] = 0
	}
	sessions := make([]stickySessionActivityResponse, 0, min(len(activities), 100))
	for _, activity := range activities {
		for _, window := range stickySessionActivityWindows {
			if activity.ActiveAgo <= window {
				counts[stickySessionWindowKey(window)]++
			}
		}
		if len(sessions) < 100 {
			sessions = append(sessions, stickySessionActivityResponse{
				SessionSuffix: stickySessionSuffix(activity.SessionHash), ActiveAgoSeconds: int64(activity.ActiveAgo / time.Second),
			})
		}
	}
	return counts, sessions
}

type stickySessionGroupResponse struct {
	GroupID                   int64                         `json:"group_id"`
	GroupName                 string                        `json:"group_name"`
	Total                     int                           `json:"total"`
	ProtectedResponseBindings int                           `json:"protected_response_bindings"`
	Sources                   []stickySessionSourceResponse `json:"sources"`
}

type stickySessionSummaryResponse struct {
	TargetAccountID int64                        `json:"target_account_id"`
	Groups          []stickySessionGroupResponse `json:"groups"`
}

func accountGroupIDs(account *service.Account) []int64 {
	if account == nil {
		return nil
	}
	seen := make(map[int64]struct{})
	out := make([]int64, 0, len(account.GroupIDs)+len(account.AccountGroups))
	for _, groupID := range account.GroupIDs {
		if groupID > 0 {
			seen[groupID] = struct{}{}
		}
	}
	for _, accountGroup := range account.AccountGroups {
		if accountGroup.GroupID > 0 {
			seen[accountGroup.GroupID] = struct{}{}
		}
	}
	for groupID := range seen {
		out = append(out, groupID)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func accountBelongsToStickyGroup(account *service.Account, groupID int64) bool {
	for _, candidate := range accountGroupIDs(account) {
		if candidate == groupID {
			return true
		}
	}
	return false
}

func stickyGroupName(account *service.Account, groupID int64) string {
	if account != nil {
		for _, group := range account.Groups {
			if group != nil && group.ID == groupID && group.Name != "" {
				return group.Name
			}
		}
	}
	return fmt.Sprintf("#%d", groupID)
}

func validateStickySessionTarget(account *service.Account) error {
	if account == nil {
		return service.ErrAccountNotFound
	}
	if account.Platform != service.PlatformOpenAI {
		return fmt.Errorf("sticky session reassignment currently supports OpenAI accounts only")
	}
	if !account.IsSchedulable() {
		return fmt.Errorf("target account must be active and schedulable")
	}
	if len(accountGroupIDs(account)) == 0 {
		return fmt.Errorf("target account must belong to at least one group")
	}
	return nil
}

// GetStickySessionSummary returns live, movable session_hash distribution for
// every group that contains the target account.
func (h *AccountHandler) GetStickySessionSummary(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if h.stickySessionAdminStore == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("STICKY_SESSION_ADMIN_UNAVAILABLE", "Sticky session administration is unavailable"))
		return
	}
	target, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := validateStickySessionTarget(target); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result := stickySessionSummaryResponse{TargetAccountID: target.ID, Groups: []stickySessionGroupResponse{}}
	for _, groupID := range accountGroupIDs(target) {
		summary, summaryErr := h.stickySessionAdminStore.Summarize(c.Request.Context(), groupID, target.Platform)
		if summaryErr != nil {
			response.ErrorFrom(c, summaryErr)
			return
		}

		accountIDs := make([]int64, 0, len(summary.Counts))
		for sourceID := range summary.Counts {
			if sourceID != target.ID {
				accountIDs = append(accountIDs, sourceID)
			}
		}
		accounts, loadErr := h.adminService.GetAccountsByIDs(c.Request.Context(), accountIDs)
		if loadErr != nil {
			response.ErrorFrom(c, loadErr)
			return
		}
		byID := make(map[int64]*service.Account, len(accounts))
		for _, account := range accounts {
			if account != nil {
				byID[account.ID] = account
			}
		}
		current := map[int64]int{}
		if h.concurrencyService != nil && len(accountIDs) > 0 {
			if counts, countErr := h.concurrencyService.GetAccountConcurrencyBatch(c.Request.Context(), accountIDs); countErr == nil {
				current = counts
			}
		}

		group := stickySessionGroupResponse{
			GroupID:                   groupID,
			GroupName:                 stickyGroupName(target, groupID),
			ProtectedResponseBindings: summary.ProtectedResponseBindings,
			Sources:                   []stickySessionSourceResponse{},
		}
		for sourceID, count := range summary.Counts {
			source := byID[sourceID]
			if source == nil || source.Platform != target.Platform || !accountBelongsToStickyGroup(source, groupID) {
				continue
			}
			recentCounts, recentSessions := stickySessionRecentActivity(summary.Activities[sourceID])
			group.Total += count
			group.Sources = append(group.Sources, stickySessionSourceResponse{
				AccountID: sourceID, AccountName: source.Name, Count: count,
				CurrentConcurrency: current[sourceID], Concurrency: source.Concurrency,
				RecentCounts: recentCounts, RecentSessions: recentSessions,
			})
		}
		sort.Slice(group.Sources, func(i, j int) bool {
			if group.Sources[i].Count != group.Sources[j].Count {
				return group.Sources[i].Count > group.Sources[j].Count
			}
			return group.Sources[i].AccountID < group.Sources[j].AccountID
		})
		result.Groups = append(result.Groups, group)
	}
	response.Success(c, result)
}

type reassignStickySessionsRequest struct {
	GroupID             int64 `json:"group_id" binding:"required"`
	SourceAccountID     int64 `json:"source_account_id" binding:"required"`
	Count               int   `json:"count" binding:"required,min=1,max=100"`
	ActiveWithinSeconds int   `json:"active_within_seconds" binding:"required"`
}

func validStickySessionActivityWindow(seconds int) bool {
	for _, window := range stickySessionActivityWindows {
		if seconds == int(window/time.Second) {
			return true
		}
	}
	return false
}

// ReassignStickySessions moves recent session_hash bindings to the target
// account. It deliberately cannot move previous_response_id bindings.
func (h *AccountHandler) ReassignStickySessions(c *gin.Context) {
	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req reassignStickySessionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if h.stickySessionAdminStore == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("STICKY_SESSION_ADMIN_UNAVAILABLE", "Sticky session administration is unavailable"))
		return
	}
	if targetID == req.SourceAccountID {
		response.BadRequest(c, "Source and target accounts must be different")
		return
	}
	if !validStickySessionActivityWindow(req.ActiveWithinSeconds) {
		response.BadRequest(c, "active_within_seconds must be one of 60, 300, 900, or 3600")
		return
	}

	target, err := h.adminService.GetAccount(c.Request.Context(), targetID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := validateStickySessionTarget(target); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	source, err := h.adminService.GetAccount(c.Request.Context(), req.SourceAccountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if source.Platform != target.Platform {
		response.BadRequest(c, "Source and target accounts must use the same platform")
		return
	}
	if !accountBelongsToStickyGroup(target, req.GroupID) || !accountBelongsToStickyGroup(source, req.GroupID) {
		response.BadRequest(c, "Source and target accounts must belong to the selected group")
		return
	}

	result, err := h.stickySessionAdminStore.Reassign(
		c.Request.Context(), req.GroupID, target.Platform, source.ID, target.ID, req.Count,
		time.Duration(req.ActiveWithinSeconds)*time.Second,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	slog.Info("admin_sticky_sessions_reassigned",
		"group_id", req.GroupID,
		"source_account_id", source.ID,
		"target_account_id", target.ID,
		"requested", req.Count,
		"active_within_seconds", req.ActiveWithinSeconds,
		"moved", result.Moved,
	)
	response.Success(c, result)
}
