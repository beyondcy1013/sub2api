package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type schedulingRateUpdateRequest struct {
	Source         string   `json:"source" binding:"required"`
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
	if source != service.SchedulingRateSourceManual && source != service.SchedulingRateSourceUpstream {
		response.BadRequest(c, "source must be manual or upstream")
		return
	}
	if source == service.SchedulingRateSourceManual {
		if req.RateMultiplier == nil || *req.RateMultiplier < 0 {
			response.BadRequest(c, "manual scheduling rate requires rate_multiplier >= 0")
			return
		}
	} else if req.RateMultiplier != nil {
		response.BadRequest(c, "rate_multiplier is only valid for manual scheduling rate")
		return
	}

	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil || account == nil {
		response.NotFound(c, "Account not found")
		return
	}
	updated, err := h.adminService.UpdateAccount(c.Request.Context(), accountID, &service.UpdateAccountInput{
		RateMultiplier:       req.RateMultiplier,
		SchedulingRateSource: &source,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), updated))
}
