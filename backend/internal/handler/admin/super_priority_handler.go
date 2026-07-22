package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// superPrioritySettingsResponse 对外暴露的超级优先模式状态。
type superPrioritySettingsResponse struct {
	Mode             string `json:"mode"`
	BaseStrategy     string `json:"base_strategy"`
	FailureThreshold int    `json:"failure_threshold"`
	CheckInterval    string `json:"check_interval"`
	TestModelID      string `json:"test_model_id"`
	TestPrompt       string `json:"test_prompt"`
	ActivatedAt      string `json:"activated_at"`
	DemotedAt        string `json:"demoted_at"`
	IsActive         bool   `json:"is_active"`
}

// superPrioritySettingsRequest 用于更新阈值/间隔等运行参数（不会直接切换模式）。
type superPrioritySettingsRequest struct {
	FailureThreshold int    `json:"failure_threshold"`
	CheckInterval    string `json:"check_interval"`
	TestModelID      string `json:"test_model_id"`
	TestPrompt       string `json:"test_prompt"`
	BaseStrategy     string `json:"base_strategy" binding:"omitempty,oneof=default lowest_cost"`
}

// GetSuperPrioritySettings 返回当前超级优先模式状态。
// GET /api/v1/admin/settings/super-priority
func (h *SettingHandler) GetSuperPrioritySettings(c *gin.Context) {
	svc := h.superPriorityService
	if svc == nil || !svc.Configured() {
		response.Error(c, 503, "super priority service unavailable")
		return
	}
	response.Success(c, superPrioritySettingsResponse{
		Mode:             svc.Mode(),
		BaseStrategy:     svc.BaseStrategy(),
		FailureThreshold: svc.FailureThreshold(),
		CheckInterval:    svc.CheckInterval(),
		TestModelID:      svc.TestModelID(),
		TestPrompt:       svc.TestPrompt(),
		ActivatedAt:      svc.ActivatedAt(),
		DemotedAt:        svc.DemotedAt(),
		IsActive:         svc.IsActive(),
	})
}

// UpdateSuperPrioritySettings 更新阈值/间隔/测试模型等参数。
// These fields do not switch the strategy. The liveness runner evaluates the
// interval per account, so changes take effect without restarting the service.
// PUT /api/v1/admin/settings/super-priority
func (h *SettingHandler) UpdateSuperPrioritySettings(c *gin.Context) {
	svc := h.superPriorityService
	if svc == nil || !svc.Configured() {
		response.Error(c, 503, "super priority service unavailable")
		return
	}
	var req superPrioritySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	svc.UpdateRuntimeParams(req.FailureThreshold, req.CheckInterval, req.TestModelID, req.TestPrompt, req.BaseStrategy)
	if err := svc.PersistConfig(); err != nil {
		response.Error(c, 500, "persist config failed: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "scheduling settings updated", "restart_required": false})
}

// ActivateSuperPriority 激活超级优先模式。
// POST /api/v1/admin/settings/super-priority/activate
func (h *SettingHandler) ActivateSuperPriority(c *gin.Context) {
	svc := h.superPriorityService
	if svc == nil || !svc.Configured() {
		response.Error(c, 503, "super priority service unavailable")
		return
	}
	if err := svc.Activate(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "super priority mode activated"})
}

// DeactivateSuperPriority removes the super-priority overlay.
// POST /api/v1/admin/settings/super-priority/deactivate
func (h *SettingHandler) DeactivateSuperPriority(c *gin.Context) {
	svc := h.superPriorityService
	if svc == nil || !svc.Configured() {
		response.Error(c, 503, "super priority service unavailable")
		return
	}
	if err := svc.Deactivate(c.Request.Context(), "manual deactivate"); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "super priority mode deactivated"})
}

// SetAccountSuperPriorityRequest per-account 超级优先标记请求体。
type SetAccountSuperPriorityRequest struct {
	Enabled bool `json:"enabled"`
}

// SetAccountSuperPriority 标记/取消标记某账号为超级优先。
// POST /api/v1/admin/accounts/:id/super-priority
func (h *AccountHandler) SetAccountSuperPriority(c *gin.Context) {
	if h.superPriorityService == nil {
		response.Error(c, 503, "super priority service unavailable")
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req SetAccountSuperPriorityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.superPriorityService.SetAccountFlag(c.Request.Context(), accountID, req.Enabled); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "account super priority flag updated", "super_priority": req.Enabled, "updated_at": time.Now().Format(time.RFC3339)})
}
