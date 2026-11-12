package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ScheduledTestHandler handles admin scheduled-test-plan management.
type ScheduledTestHandler struct {
	scheduledTestSvc *service.ScheduledTestService
}

// NewScheduledTestHandler creates a new ScheduledTestHandler.
func NewScheduledTestHandler(scheduledTestSvc *service.ScheduledTestService) *ScheduledTestHandler {
	return &ScheduledTestHandler{scheduledTestSvc: scheduledTestSvc}
}

type createScheduledTestPlanRequest struct {
	AccountID              int64  `json:"account_id" binding:"required"`
	ModelID                string `json:"model_id"`
	CronExpression         string `json:"cron_expression" binding:"required"`
	Enabled                *bool  `json:"enabled"`
	MaxResults             int    `json:"max_results"`
	AutoRecover            *bool  `json:"auto_recover"`
	AutoRecoverSchedulable *bool  `json:"auto_recover_schedulable"`
	Mode                   *string `json:"mode"`
	AutoPauseOnDowngrade   *bool   `json:"auto_pause_on_downgrade"`
	RuleIDs                *[]int64 `json:"rule_ids"`
	RuleStrategy           string  `json:"rule_strategy"`
	DowngradeAction        string  `json:"downgrade_action"`
	DowngradeThreshold     *int    `json:"downgrade_threshold"`
}

type batchScheduledTestPlanRequest struct {
	AccountIDs             []int64 `json:"account_ids" binding:"required,min=1,max=200"`
	ModelID                string  `json:"model_id"`
	CronExpression         string  `json:"cron_expression" binding:"required"`
	Enabled                *bool   `json:"enabled"`
	MaxResults             int     `json:"max_results"`
	AutoRecover            *bool   `json:"auto_recover"`
	AutoRecoverSchedulable *bool   `json:"auto_recover_schedulable"`
	Mode                   *string `json:"mode"`
	AutoPauseOnDowngrade   *bool   `json:"auto_pause_on_downgrade"`
	RuleIDs                *[]int64 `json:"rule_ids"`
	RuleStrategy           string  `json:"rule_strategy"`
	DowngradeAction        string  `json:"downgrade_action"`
	DowngradeThreshold     *int    `json:"downgrade_threshold"`
}

type updateScheduledTestPlanRequest struct {
	ModelID                string  `json:"model_id"`
	CronExpression         string  `json:"cron_expression"`
	Enabled                *bool   `json:"enabled"`
	MaxResults             int     `json:"max_results"`
	AutoRecover            *bool   `json:"auto_recover"`
	AutoRecoverSchedulable *bool   `json:"auto_recover_schedulable"`
	Mode                   *string `json:"mode"`
	AutoPauseOnDowngrade   *bool   `json:"auto_pause_on_downgrade"`
	RuleIDs                *[]int64 `json:"rule_ids"`
	RuleStrategy           string  `json:"rule_strategy"`
	DowngradeAction        string  `json:"downgrade_action"`
	DowngradeThreshold     *int    `json:"downgrade_threshold"`
}

// ListByAccount GET /admin/accounts/:id/scheduled-test-plans
func (h *ScheduledTestHandler) ListByAccount(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid account id")
		return
	}

	plans, err := h.scheduledTestSvc.ListPlansByAccount(c.Request.Context(), accountID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, plans)
}

// ListAll GET /admin/scheduled-test-plans
func (h *ScheduledTestHandler) ListAll(c *gin.Context) {
	plans, err := h.scheduledTestSvc.ListAllPlans(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if plans == nil {
		plans = make([]*service.ScheduledTestPlan, 0)
	}
	c.JSON(http.StatusOK, plans)
}

// Create POST /admin/scheduled-test-plans
func (h *ScheduledTestHandler) Create(c *gin.Context) {
	var req createScheduledTestPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	plan := &service.ScheduledTestPlan{
		AccountID:      req.AccountID,
		ModelID:        req.ModelID,
		CronExpression: req.CronExpression,
		Enabled:        true,
		MaxResults:     req.MaxResults,
	}
	if req.Enabled != nil {
		plan.Enabled = *req.Enabled
	}
	if req.AutoRecover != nil {
		plan.AutoRecover = *req.AutoRecover
	}
	if req.AutoRecoverSchedulable != nil {
		plan.AutoRecoverSchedulable = *req.AutoRecoverSchedulable
	}
	if req.Mode != nil {
		plan.Mode = service.NormalizeScheduledTestMode(*req.Mode)
	}
	if req.AutoPauseOnDowngrade != nil {
		plan.AutoPauseOnDowngrade = *req.AutoPauseOnDowngrade
	}
	applyRuleFields(&plan.RuleIDs, &plan.RuleStrategy, &plan.DowngradeAction, &plan.DowngradeThreshold,
		req.RuleIDs, req.RuleStrategy, req.DowngradeAction, req.DowngradeThreshold)

	created, err := h.scheduledTestSvc.CreatePlan(c.Request.Context(), plan)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, created)
}

// BatchCreate POST /admin/scheduled-test-plans/batch
func (h *ScheduledTestHandler) BatchCreate(c *gin.Context) {
	var req batchScheduledTestPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	template := &service.ScheduledTestPlan{
		ModelID:        req.ModelID,
		CronExpression: req.CronExpression,
		Enabled:        true,
		MaxResults:     req.MaxResults,
	}
	if req.Enabled != nil {
		template.Enabled = *req.Enabled
	}
	if req.AutoRecover != nil {
		template.AutoRecover = *req.AutoRecover
	}
	if req.AutoRecoverSchedulable != nil {
		template.AutoRecoverSchedulable = *req.AutoRecoverSchedulable
	}
	if req.Mode != nil {
		template.Mode = service.NormalizeScheduledTestMode(*req.Mode)
	}
	if req.AutoPauseOnDowngrade != nil {
		template.AutoPauseOnDowngrade = *req.AutoPauseOnDowngrade
	}
	applyRuleFields(&template.RuleIDs, &template.RuleStrategy, &template.DowngradeAction, &template.DowngradeThreshold,
		req.RuleIDs, req.RuleStrategy, req.DowngradeAction, req.DowngradeThreshold)

	created, failed, errs := h.scheduledTestSvc.BatchUpsertPlans(c.Request.Context(), req.AccountIDs, template)
	c.JSON(http.StatusOK, gin.H{
		"created": created,
		"failed":  failed,
		"errors":  errs,
	})
}

// Update PUT /admin/scheduled-test-plans/:id
func (h *ScheduledTestHandler) Update(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}

	existing, err := h.scheduledTestSvc.GetPlan(c.Request.Context(), planID)
	if err != nil {
		response.NotFound(c, "plan not found")
		return
	}

	var req updateScheduledTestPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.ModelID != "" {
		existing.ModelID = req.ModelID
	}
	if req.CronExpression != "" {
		existing.CronExpression = req.CronExpression
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.MaxResults > 0 {
		existing.MaxResults = req.MaxResults
	}
	if req.AutoRecover != nil {
		existing.AutoRecover = *req.AutoRecover
	}
	if req.AutoRecoverSchedulable != nil {
		existing.AutoRecoverSchedulable = *req.AutoRecoverSchedulable
	}
	if req.Mode != nil {
		existing.Mode = service.NormalizeScheduledTestMode(*req.Mode)
	}
	if req.AutoPauseOnDowngrade != nil {
		existing.AutoPauseOnDowngrade = *req.AutoPauseOnDowngrade
	}
	applyRuleFields(&existing.RuleIDs, &existing.RuleStrategy, &existing.DowngradeAction, &existing.DowngradeThreshold,
		req.RuleIDs, req.RuleStrategy, req.DowngradeAction, req.DowngradeThreshold)

	updated, err := h.scheduledTestSvc.UpdatePlan(c.Request.Context(), existing)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, updated)
}

// Delete DELETE /admin/scheduled-test-plans/:id
func (h *ScheduledTestHandler) Delete(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}

	if err := h.scheduledTestSvc.DeletePlan(c.Request.Context(), planID); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ListResults GET /admin/scheduled-test-plans/:id/results
func (h *ScheduledTestHandler) ListResults(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	results, err := h.scheduledTestSvc.ListResults(c.Request.Context(), planID, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, results)
}

// applyRuleFields copies the optional rule-selection/handling fields from the
// request onto the plan (pointer fields = omitted keeps the current value).
func applyRuleFields(ruleIDs *[]int64, strategy, action *string, threshold *int,
	reqRuleIDs *[]int64, reqStrategy, reqAction string, reqThreshold *int) {
	if reqRuleIDs != nil {
		*ruleIDs = *reqRuleIDs
	}
	if reqStrategy != "" {
		*strategy = reqStrategy
	}
	if reqAction != "" {
		*action = reqAction
	}
	if reqThreshold != nil && *reqThreshold >= 1 {
		*threshold = *reqThreshold
	}
}
