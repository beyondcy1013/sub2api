package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type relayFailureBudgetPolicyDTO struct {
	Supported               bool `json:"supported"`
	Enabled                 bool `json:"enabled"`
	WindowMinutes           int  `json:"window_minutes"`
	FailureThresholdPercent int  `json:"failure_threshold_percent"`
	MinRequests             int  `json:"min_requests"`
	ConsecutiveFailures     int  `json:"consecutive_failures"`
	CooldownMinutes         int  `json:"cooldown_minutes"`
}

type accountQuotaPolicyDTO struct {
	Supported   bool    `json:"supported"`
	TotalLimit  float64 `json:"total_limit"`
	DailyLimit  float64 `json:"daily_limit"`
	WeeklyLimit float64 `json:"weekly_limit"`
}

type accountSchedulingRatePolicyDTO struct {
	RateMultiplier float64 `json:"rate_multiplier"`
	SyncMode       string  `json:"sync_mode"`
}

type accountPolicySettingsResponse struct {
	AccountID          int64                          `json:"account_id"`
	RelayFailureBudget relayFailureBudgetPolicyDTO    `json:"relay_failure_budget"`
	Quota              accountQuotaPolicyDTO          `json:"quota"`
	SchedulingRate     accountSchedulingRatePolicyDTO `json:"scheduling_rate"`
}

type updateAccountPolicySettingsRequest struct {
	RelayFailureBudget *struct {
		Enabled                 bool `json:"enabled"`
		WindowMinutes           int  `json:"window_minutes"`
		FailureThresholdPercent int  `json:"failure_threshold_percent"`
		MinRequests             int  `json:"min_requests"`
		ConsecutiveFailures     int  `json:"consecutive_failures"`
		CooldownMinutes         int  `json:"cooldown_minutes"`
	} `json:"relay_failure_budget"`
	Quota *struct {
		TotalLimit  float64 `json:"total_limit"`
		DailyLimit  float64 `json:"daily_limit"`
		WeeklyLimit float64 `json:"weekly_limit"`
	} `json:"quota"`
	SchedulingRate *struct {
		RateMultiplier float64 `json:"rate_multiplier"`
		SyncMode       string  `json:"sync_mode"`
	} `json:"scheduling_rate"`
}

func buildAccountPolicySettingsResponse(account *service.Account) accountPolicySettingsResponse {
	relaySettings, relaySupported := account.OpenAIRelayFailureBudgetSettings()
	return accountPolicySettingsResponse{
		AccountID: account.ID,
		RelayFailureBudget: relayFailureBudgetPolicyDTO{
			Supported:               relaySupported,
			Enabled:                 relaySettings.Enabled,
			WindowMinutes:           relaySettings.WindowMinutes,
			FailureThresholdPercent: relaySettings.FailureThresholdPercent,
			MinRequests:             relaySettings.MinRequests,
			ConsecutiveFailures:     relaySettings.ConsecutiveFailures,
			CooldownMinutes:         relaySettings.CooldownMinutes,
		},
		Quota: accountQuotaPolicyDTO{
			Supported:   account.IsAPIKeyOrBedrock() && !account.IsCredentialShadow(),
			TotalLimit:  account.GetQuotaLimit(),
			DailyLimit:  account.GetQuotaDailyLimit(),
			WeeklyLimit: account.GetQuotaWeeklyLimit(),
		},
		SchedulingRate: accountSchedulingRatePolicyDTO{
			RateMultiplier: account.BillingRateMultiplier(),
			SyncMode:       account.SchedulingRateSyncMode(),
		},
	}
}

func (h *AccountHandler) GetPolicySettings(c *gin.Context) {
	accountID, ok := accountPolicySettingsID(c)
	if !ok {
		return
	}
	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil || account == nil {
		response.NotFound(c, "Account not found")
		return
	}
	response.Success(c, buildAccountPolicySettingsResponse(account))
}

func (h *AccountHandler) UpdatePolicySettings(c *gin.Context) {
	accountID, ok := accountPolicySettingsID(c)
	if !ok {
		return
	}
	var req updateAccountPolicySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.RelayFailureBudget == nil && req.Quota == nil && req.SchedulingRate == nil {
		response.BadRequest(c, "At least one account policy section is required")
		return
	}

	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil || account == nil {
		response.NotFound(c, "Account not found")
		return
	}
	input := &service.UpdateAccountInput{}

	if policy := req.RelayFailureBudget; policy != nil {
		if !account.SupportsOpenAIRelayFailureBudget() {
			response.BadRequest(c, "Relay failure budget requires a custom OpenAI API-key relay without pool mode or custom error-code handling")
			return
		}
		credentials, applyErr := service.ApplyOpenAIRelayFailureBudgetSettings(account.Credentials, service.OpenAIRelayFailureBudgetSettings{
			Enabled:                 policy.Enabled,
			WindowMinutes:           policy.WindowMinutes,
			FailureThresholdPercent: policy.FailureThresholdPercent,
			MinRequests:             policy.MinRequests,
			ConsecutiveFailures:     policy.ConsecutiveFailures,
			CooldownMinutes:         policy.CooldownMinutes,
		})
		if applyErr != nil {
			response.BadRequest(c, applyErr.Error())
			return
		}
		input.Credentials = credentials
	}

	if quota := req.Quota; quota != nil {
		if !account.IsAPIKeyOrBedrock() || account.IsCredentialShadow() {
			response.BadRequest(c, "Quota policy requires an API-key or Bedrock account")
			return
		}
		if quota.TotalLimit < 0 || quota.DailyLimit < 0 || quota.WeeklyLimit < 0 {
			response.BadRequest(c, "Quota limits must be greater than or equal to zero")
			return
		}
		extra := cloneAccountPolicyMap(account.Extra)
		setOrDeletePositiveAccountPolicyValue(extra, "quota_limit", quota.TotalLimit)
		setOrDeletePositiveAccountPolicyValue(extra, "quota_daily_limit", quota.DailyLimit)
		setOrDeletePositiveAccountPolicyValue(extra, "quota_weekly_limit", quota.WeeklyLimit)
		input.Extra = extra
	}

	if rate := req.SchedulingRate; rate != nil {
		mode := strings.ToLower(strings.TrimSpace(rate.SyncMode))
		if mode != service.SchedulingRateSyncModeAutoOverwrite && mode != service.SchedulingRateSyncModeManualLock {
			response.BadRequest(c, "sync_mode must be auto_overwrite or manual_lock")
			return
		}
		if rate.RateMultiplier < 0 {
			response.BadRequest(c, "rate_multiplier must be greater than or equal to zero")
			return
		}
		compatSource := service.SchedulingRateSourceManual
		if mode == service.SchedulingRateSyncModeAutoOverwrite {
			compatSource = service.SchedulingRateSourceUpstream
		}
		input.RateMultiplier = &rate.RateMultiplier
		input.SchedulingRateSyncMode = &mode
		input.SchedulingRateSource = &compatSource
	}

	updated, err := h.adminService.UpdateAccount(c.Request.Context(), accountID, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, buildAccountPolicySettingsResponse(updated))
}

func accountPolicySettingsID(c *gin.Context) (int64, bool) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return 0, false
	}
	return accountID, true
}

func cloneAccountPolicyMap(source map[string]any) map[string]any {
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func setOrDeletePositiveAccountPolicyValue(target map[string]any, key string, value float64) {
	if value > 0 {
		target[key] = value
		return
	}
	delete(target, key)
}
