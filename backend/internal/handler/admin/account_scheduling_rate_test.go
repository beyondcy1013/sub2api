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

func (s *schedulingRateAdminService) UpdateAccount(_ context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error) {
	s.input = input
	updated := *s.account
	updated.ID = id
	updated.Extra = make(map[string]any, len(s.account.Extra)+1)
	for key, value := range s.account.Extra {
		updated.Extra[key] = value
	}
	updated.Extra[service.SchedulingRateSourceExtraKey] = input.SchedulingRateSource
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
	return router
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
	require.Equal(t, service.SchedulingRateSourceManual, adminSvc.input.SchedulingRateSource)
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

func TestAccountHandlerUpdateSchedulingRateFollowsUpstreamWithoutOverwritingManualRate(t *testing.T) {
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
	require.Equal(t, service.SchedulingRateSourceUpstream, adminSvc.input.SchedulingRateSource)
	require.Nil(t, adminSvc.input.RateMultiplier)
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
		`{"source":"upstream","rate_multiplier":0.1}`,
	} {
		recorder := schedulingRateRequest(t, router, body)
		require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
}

