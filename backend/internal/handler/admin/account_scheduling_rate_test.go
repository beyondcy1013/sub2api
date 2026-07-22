package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type schedulingRateAdminService struct {
	*stubAdminService
	account *service.Account
	input   *service.UpdateAccountInput
}

func (s *schedulingRateAdminService) GetAccount(_ context.Context, _ int64) (*service.Account, error) {
	return s.account, nil
}

func (s *schedulingRateAdminService) ListAccounts(_ context.Context, _ int, _ int, _ string, _ string, _ string, _ string, _ int64, _ string, _ string, _ string, _ bool) ([]service.Account, int64, error) {
	return []service.Account{*s.account}, 1, nil
}

func (s *schedulingRateAdminService) UpdateAccount(_ context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error) {
	s.input = input
	updated := *s.account
	updated.ID = id
	updated.Extra = make(map[string]any, len(s.account.Extra)+1)
	for key, value := range s.account.Extra {
		updated.Extra[key] = value
	}
	if input.SchedulingRateSource != nil {
		updated.Extra[service.SchedulingRateSourceExtraKey] = *input.SchedulingRateSource
	}
	if input.RateMultiplier != nil {
		updated.RateMultiplier = input.RateMultiplier
	}
	s.account = &updated
	return &updated, nil
}

func setupSchedulingRateRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.PUT("/admin/accounts/:id/scheduling-rate", handler.UpdateSchedulingRate)
	router.GET("/admin/accounts", handler.List)
	return router
}

func TestAccountHandlerListIncludesSchedulingRateMetadata(t *testing.T) {
	rate := 0.35
	adminSvc := &schedulingRateAdminService{
		stubAdminService: newStubAdminService(),
		account: &service.Account{
			ID:             42,
			Platform:       service.PlatformOpenAI,
			Type:           service.AccountTypeAPIKey,
			Status:         service.StatusActive,
			RateMultiplier: &rate,
			Extra:          map[string]any{service.SchedulingRateSourceExtraKey: service.SchedulingRateSourceManual},
		},
	}
	recorder := httptest.NewRecorder()
	setupSchedulingRateRouter(adminSvc).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin/accounts?page=1&page_size=20", nil))

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var payload struct {
		Data struct {
			Items []struct {
				SchedulingRateMultiplier *float64 `json:"scheduling_rate_multiplier"`
				SchedulingRateKnown      bool     `json:"scheduling_rate_known"`
				SchedulingRateSource     string   `json:"scheduling_rate_source"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Items, 1)
	require.True(t, payload.Data.Items[0].SchedulingRateKnown)
	require.Equal(t, service.SchedulingRateSourceManual, payload.Data.Items[0].SchedulingRateSource)
	require.NotNil(t, payload.Data.Items[0].SchedulingRateMultiplier)
	require.InDelta(t, 0.35, *payload.Data.Items[0].SchedulingRateMultiplier, 1e-9)
}

func TestBuildSchedulingRateOptimalAccountIDsUsesLiveSchedulableGroupMinimum(t *testing.T) {
	now := time.Now().UTC()
	groupOne := int64(11)
	groupTwo := int64(22)
	rate := func(value float64) *float64 { return &value }
	liveness := func(status string) map[string]any {
		return map[string]any{
			service.SchedulingLivenessExtraKey: map[string]any{
				"status":          status,
				"last_attempt_at": now.Add(-time.Minute),
				"fresh_until":     now.Add(time.Hour),
			},
		}
	}
	account := func(id int64, groupID int64, multiplier float64, status string, schedulable bool, livenessStatus string) service.Account {
		return service.Account{
			ID:             id,
			Platform:       service.PlatformOpenAI,
			Type:           service.AccountTypeAPIKey,
			Status:         status,
			Schedulable:    schedulable,
			RateMultiplier: rate(multiplier),
			GroupIDs:       []int64{groupID},
			Extra:          liveness(livenessStatus),
		}
	}

	optimal := buildSchedulingRateOptimalAccountIDs([]service.Account{
		account(1, groupOne, 0.2, service.StatusActive, true, service.SchedulingLivenessStatusAlive),
		account(2, groupOne, 0.8, service.StatusActive, true, service.SchedulingLivenessStatusAlive),
		account(3, groupOne, 0.1, service.StatusActive, true, service.SchedulingLivenessStatusDead),
		account(4, groupOne, 0.1, service.StatusDisabled, true, service.SchedulingLivenessStatusAlive),
		account(5, groupOne, 0.1, service.StatusActive, false, service.SchedulingLivenessStatusAlive),
		account(6, groupTwo, 0.9, service.StatusActive, true, service.SchedulingLivenessStatusAlive),
		account(7, groupOne, 0.2, service.StatusActive, true, service.SchedulingLivenessStatusAlive),
	}, now)

	require.Contains(t, optimal, int64(1))
	require.Contains(t, optimal, int64(6))
	require.Contains(t, optimal, int64(7))
	require.NotContains(t, optimal, int64(2))
	require.NotContains(t, optimal, int64(3))
	require.NotContains(t, optimal, int64(4))
	require.NotContains(t, optimal, int64(5))
}

func TestAccountHandlerListSchedulingOptimalIgnoresPagination(t *testing.T) {
	router, adminSvc := setupAccountListRouter()
	now := time.Now().UTC()
	groupID := int64(31)
	aliveExtra := map[string]any{
		service.SchedulingLivenessExtraKey: map[string]any{
			"status":          service.SchedulingLivenessStatusAlive,
			"last_attempt_at": now.Add(-time.Minute),
			"fresh_until":     now.Add(time.Hour),
		},
	}
	visibleRate := 0.8
	hiddenRate := 0.2
	visible := service.Account{
		ID: 81, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, RateMultiplier: &visibleRate,
		GroupIDs: []int64{groupID}, Extra: aliveExtra, CreatedAt: now, UpdatedAt: now,
	}
	hiddenCheaper := service.Account{
		ID: 82, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, RateMultiplier: &hiddenRate,
		GroupIDs: []int64{groupID}, Extra: aliveExtra, CreatedAt: now, UpdatedAt: now,
	}
	adminSvc.accounts = []service.Account{visible}
	adminSvc.accountSchedulerScoreFilterAccounts = []service.Account{visible, hiddenCheaper}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts?page=1&page_size=1&include_scheduling_optimal=1", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, 1, adminSvc.schedulerScoreFilterCalls)
	var payload struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Items, 1)
	require.Contains(t, payload.Data.Items[0], "scheduling_rate_optimal")
	require.Equal(t, false, payload.Data.Items[0]["scheduling_rate_optimal"])
}

func schedulingRateRequest(t *testing.T, router http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/accounts/42/scheduling-rate", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestAccountHandlerUpdateSchedulingRateStoresManualRate(t *testing.T) {
	manual := 0.9
	adminSvc := &schedulingRateAdminService{
		stubAdminService: newStubAdminService(),
		account: &service.Account{
			ID:             42,
			RateMultiplier: &manual,
			Extra:          map[string]any{},
		},
	}

	recorder := schedulingRateRequest(t, setupSchedulingRateRouter(adminSvc), `{"source":"manual","rate_multiplier":0.35}`)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.NotNil(t, adminSvc.input)
	require.NotNil(t, adminSvc.input.SchedulingRateSource)
	require.Equal(t, service.SchedulingRateSourceManual, *adminSvc.input.SchedulingRateSource)
	require.NotNil(t, adminSvc.input.RateMultiplier)
	require.InDelta(t, 0.35, *adminSvc.input.RateMultiplier, 1e-9)

	var payload struct {
		Data struct {
			SchedulingRateMultiplier *float64 `json:"scheduling_rate_multiplier"`
			SchedulingRateKnown      bool     `json:"scheduling_rate_known"`
			SchedulingRateSource     string   `json:"scheduling_rate_source"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Data.SchedulingRateKnown)
	require.Equal(t, service.SchedulingRateSourceManual, payload.Data.SchedulingRateSource)
	require.NotNil(t, payload.Data.SchedulingRateMultiplier)
	require.InDelta(t, 0.35, *payload.Data.SchedulingRateMultiplier, 1e-9)
}

func TestAccountHandlerUpdateSchedulingRateEnablesAutomaticOverwriteWhileKeepingCurrentRate(t *testing.T) {
	now := time.Now()
	manual := 0.9
	adminSvc := &schedulingRateAdminService{
		stubAdminService: newStubAdminService(),
		account: &service.Account{
			ID:             42,
			RateMultiplier: &manual,
			Extra: map[string]any{
				service.UpstreamBillingProbeExtraKey: map[string]any{
					"status":      service.UpstreamBillingProbeStatusOK,
					"received_at": now.Add(-time.Minute),
					"fresh_until": now.Add(time.Hour),
					"data": map[string]any{
						"resolved_rate_multiplier": 0.2,
					},
				},
			},
		},
	}

	recorder := schedulingRateRequest(t, setupSchedulingRateRouter(adminSvc), `{"source":"upstream"}`)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.NotNil(t, adminSvc.input)
	require.NotNil(t, adminSvc.input.SchedulingRateSource)
	require.Equal(t, service.SchedulingRateSourceUpstream, *adminSvc.input.SchedulingRateSource)
	require.NotNil(t, adminSvc.input.SchedulingRateSyncMode)
	require.Equal(t, service.SchedulingRateSyncModeAutoOverwrite, *adminSvc.input.SchedulingRateSyncMode)
	require.NotNil(t, adminSvc.input.RateMultiplier)
	require.InDelta(t, 0.9, *adminSvc.input.RateMultiplier, 1e-9)
	require.InDelta(t, 0.9, adminSvc.account.BillingRateMultiplier(), 1e-9)
}

func TestAccountHandlerUpdateSchedulingRateValidatesRequest(t *testing.T) {
	rate := 1.0
	adminSvc := &schedulingRateAdminService{
		stubAdminService: newStubAdminService(),
		account:          &service.Account{ID: 42, RateMultiplier: &rate},
	}
	router := setupSchedulingRateRouter(adminSvc)

	for _, body := range []string{
		`{"source":"unknown"}`,
		`{"source":"manual","rate_multiplier":-0.1}`,
		`{"sync_mode":"invalid","rate_multiplier":0.1}`,
	} {
		recorder := schedulingRateRequest(t, router, body)
		require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
}
