package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type accountPolicyAdminService struct {
	*stubAdminService
	account *service.Account
	input   *service.UpdateAccountInput
}

func (s *accountPolicyAdminService) GetAccount(_ context.Context, _ int64) (*service.Account, error) {
	return s.account, nil
}

func (s *accountPolicyAdminService) UpdateAccount(_ context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error) {
	s.input = input
	updated := *s.account
	updated.ID = id
	if input.Credentials != nil {
		updated.Credentials = input.Credentials
	}
	if input.Extra != nil {
		updated.Extra = input.Extra
	}
	if input.RateMultiplier != nil {
		updated.RateMultiplier = input.RateMultiplier
	}
	if input.SchedulingRateSyncMode != nil {
		if updated.Extra == nil {
			updated.Extra = make(map[string]any)
		}
		updated.Extra[service.SchedulingRateSyncModeExtraKey] = *input.SchedulingRateSyncMode
	}
	s.account = &updated
	return &updated, nil
}

func setupAccountPolicyRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.GET("/admin/accounts/:id/policy-settings", handler.GetPolicySettings)
	router.PUT("/admin/accounts/:id/policy-settings", handler.UpdatePolicySettings)
	return router
}

func customRelayPolicyAccount() *service.Account {
	rate := 0.75
	return &service.Account{
		ID:             42,
		Name:           "relay",
		Platform:       service.PlatformOpenAI,
		Type:           service.AccountTypeAPIKey,
		Status:         service.StatusActive,
		Schedulable:    true,
		RateMultiplier: &rate,
		Credentials: map[string]any{
			"api_key":  "sk-secret",
			"base_url": "https://relay.example.com/v1",
		},
		Extra: map[string]any{
			"quota_limit":      20.0,
			"quota_daily_used": 3.0,
		},
	}
}

func TestAccountPolicySettingsGetReturnsNormalizedDefaults(t *testing.T) {
	adminSvc := &accountPolicyAdminService{stubAdminService: newStubAdminService(), account: customRelayPolicyAccount()}
	recorder := httptest.NewRecorder()
	setupAccountPolicyRouter(adminSvc).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin/accounts/42/policy-settings", nil))

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var payload struct {
		Data accountPolicySettingsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, int64(42), payload.Data.AccountID)
	require.True(t, payload.Data.RelayFailureBudget.Supported)
	require.True(t, payload.Data.RelayFailureBudget.Enabled)
	require.Equal(t, 10, payload.Data.RelayFailureBudget.WindowMinutes)
	require.Equal(t, 30, payload.Data.RelayFailureBudget.FailureThresholdPercent)
	require.Equal(t, 10, payload.Data.RelayFailureBudget.MinRequests)
	require.Equal(t, 5, payload.Data.RelayFailureBudget.ConsecutiveFailures)
	require.Equal(t, 2, payload.Data.RelayFailureBudget.CooldownMinutes)
	require.True(t, payload.Data.Quota.Supported)
	require.InDelta(t, 20, payload.Data.Quota.TotalLimit, 1e-9)
	require.InDelta(t, 0.75, payload.Data.SchedulingRate.RateMultiplier, 1e-9)
	require.Equal(t, service.SchedulingRateSyncModeAutoOverwrite, payload.Data.SchedulingRate.SyncMode)
}

func TestAccountPolicySettingsUpdatePersistsAllSectionsTogether(t *testing.T) {
	adminSvc := &accountPolicyAdminService{stubAdminService: newStubAdminService(), account: customRelayPolicyAccount()}
	body := []byte(`{
		"relay_failure_budget":{"enabled":true,"window_minutes":25,"failure_threshold_percent":20,"min_requests":12,"consecutive_failures":6,"cooldown_minutes":4},
		"quota":{"total_limit":100,"daily_limit":8,"weekly_limit":40},
		"scheduling_rate":{"rate_multiplier":0.35,"sync_mode":"manual_lock"}
	}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/accounts/42/policy-settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	setupAccountPolicyRouter(adminSvc).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.NotNil(t, adminSvc.input)
	require.Equal(t, "sk-secret", adminSvc.input.Credentials["api_key"])
	require.Equal(t, 25, adminSvc.input.Credentials["relay_failure_budget_window_minutes"])
	require.Equal(t, 20, adminSvc.input.Credentials["relay_failure_budget_failure_threshold_percent"])
	require.InDelta(t, 100, adminSvc.input.Extra["quota_limit"], 1e-9)
	require.InDelta(t, 8, adminSvc.input.Extra["quota_daily_limit"], 1e-9)
	require.InDelta(t, 40, adminSvc.input.Extra["quota_weekly_limit"], 1e-9)
	require.InDelta(t, 3, adminSvc.input.Extra["quota_daily_used"], 1e-9, "managed usage must be preserved")
	require.NotNil(t, adminSvc.input.RateMultiplier)
	require.InDelta(t, 0.35, *adminSvc.input.RateMultiplier, 1e-9)
	require.Equal(t, service.SchedulingRateSyncModeManualLock, *adminSvc.input.SchedulingRateSyncMode)
	require.Equal(t, service.SchedulingRateSourceManual, *adminSvc.input.SchedulingRateSource)
}

func TestAccountPolicySettingsUpdateDisablesBudgetAndClearsStaleNumbers(t *testing.T) {
	account := customRelayPolicyAccount()
	account.Credentials["relay_failure_budget_enabled"] = true
	account.Credentials["relay_failure_budget_window_minutes"] = 40
	account.Credentials["relay_failure_budget_min_requests"] = 30
	adminSvc := &accountPolicyAdminService{stubAdminService: newStubAdminService(), account: account}
	body := []byte(`{"relay_failure_budget":{"enabled":false}}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/accounts/42/policy-settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	setupAccountPolicyRouter(adminSvc).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, false, adminSvc.input.Credentials["relay_failure_budget_enabled"])
	require.NotContains(t, adminSvc.input.Credentials, "relay_failure_budget_window_minutes")
	require.NotContains(t, adminSvc.input.Credentials, "relay_failure_budget_min_requests")
}

func TestAccountPolicySettingsRejectsBudgetForOfficialOpenAIEndpoint(t *testing.T) {
	account := customRelayPolicyAccount()
	account.Credentials["base_url"] = "https://api.openai.com/v1"
	adminSvc := &accountPolicyAdminService{stubAdminService: newStubAdminService(), account: account}
	body := []byte(`{"relay_failure_budget":{"enabled":true,"window_minutes":10,"failure_threshold_percent":30,"min_requests":10,"consecutive_failures":5,"cooldown_minutes":2}}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/accounts/42/policy-settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	setupAccountPolicyRouter(adminSvc).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	require.Nil(t, adminSvc.input)
}

func TestAccountPolicySettingsRejectsInvalidValues(t *testing.T) {
	adminSvc := &accountPolicyAdminService{stubAdminService: newStubAdminService(), account: customRelayPolicyAccount()}
	for _, body := range []string{
		`{"quota":{"total_limit":-1,"daily_limit":0,"weekly_limit":0}}`,
		`{"scheduling_rate":{"rate_multiplier":-1,"sync_mode":"manual_lock"}}`,
		`{"scheduling_rate":{"rate_multiplier":1,"sync_mode":"invalid"}}`,
		`{"relay_failure_budget":{"enabled":true,"window_minutes":0,"failure_threshold_percent":30,"min_requests":10,"consecutive_failures":5,"cooldown_minutes":2}}`,
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPut, "/admin/accounts/42/policy-settings", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		setupAccountPolicyRouter(adminSvc).ServeHTTP(recorder, request)
		require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
}
