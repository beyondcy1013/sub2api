package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type schedulingRateUpdateRequest struct {
	Source         string   `json:"source"`
	SyncMode       string   `json:"sync_mode"`
	RateMultiplier *float64 `json:"rate_multiplier"`
}

// UpdateSchedulingRate changes the account's scheduling-rate source. Manual
// mode stores an explicit multiplier; upstream mode follows the latest valid
// billing probe and treats stale/unsupported snapshots as unknown at runtime.
// PUT /api/v1/admin/accounts/:id/scheduling-rate
func (h *AccountHandler) UpdateSchedulingRate(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req schedulingRateUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	source := strings.ToLower(strings.TrimSpace(req.Source))
	mode := strings.ToLower(strings.TrimSpace(req.SyncMode))
	if mode == "" {
		switch source {
		case service.SchedulingRateSourceManual:
			mode = service.SchedulingRateSyncModeManualLock
		case service.SchedulingRateSourceUpstream:
			mode = service.SchedulingRateSyncModeAutoOverwrite
		default:
			response.BadRequest(c, "sync_mode must be auto_overwrite or manual_lock")
			return
		}
	}
	if mode != service.SchedulingRateSyncModeManualLock && mode != service.SchedulingRateSyncModeAutoOverwrite {
		response.BadRequest(c, "sync_mode must be auto_overwrite or manual_lock")
		return
	}
	if req.RateMultiplier != nil && *req.RateMultiplier < 0 {
		response.BadRequest(c, "rate_multiplier must be >= 0")
		return
	}

	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil || account == nil {
		response.NotFound(c, "Account not found")
		return
	}
	rate := req.RateMultiplier
	if rate == nil {
		fallback := account.BillingRateMultiplier()
		rate = &fallback
	}
	compatSource := service.SchedulingRateSourceManual
	if mode == service.SchedulingRateSyncModeAutoOverwrite {
		compatSource = service.SchedulingRateSourceUpstream
	}
	updated, err := h.adminService.UpdateAccount(c.Request.Context(), accountID, &service.UpdateAccountInput{
		RateMultiplier:         rate,
		SchedulingRateSource:   &compatSource,
		SchedulingRateSyncMode: &mode,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), updated))
}
