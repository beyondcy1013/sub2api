package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type schedulingRefreshStub struct {
	result service.SchedulingRefreshResult
	status service.SchedulingLivenessRuntimeStatus
	calls  int
}

func (s *schedulingRefreshStub) RefreshNow(context.Context) (service.SchedulingRefreshResult, error) {
	s.calls++
	return s.result, nil
}

func (s *schedulingRefreshStub) RuntimeStatus() service.SchedulingLivenessRuntimeStatus {
	return s.status
}

type upstreamBillingRefreshStub struct {
	result service.SchedulingRefreshResult
	calls  int
}

func (s *upstreamBillingRefreshStub) RefreshNow(context.Context) (service.SchedulingRefreshResult, error) {
	s.calls++
	return s.result, nil
}

func newSuperPrioritySettingsTestRouter(t *testing.T) (*gin.Engine, *config.Config) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.yaml"), []byte("server:\n  port: 18381\n"), 0o640))

	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{
		Mode:             "normal",
		BaseStrategy:     service.AccountSchedulingStrategyDefault,
		FailureThreshold: 2,
		CheckInterval:    "@every 1m",
	}}
	handler := NewSettingHandler(nil, nil, nil, nil, nil, nil, nil)
	handler.SetSuperPriorityService(service.NewSuperPriorityService(nil, cfg))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/super-priority", handler.GetSuperPrioritySettings)
	router.PUT("/super-priority", handler.UpdateSuperPrioritySettings)
	router.POST("/super-priority/activate", handler.ActivateSuperPriority)
	return router, cfg
}

func decodeSuperPriorityResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	data, ok := payload["data"].(map[string]any)
	require.True(t, ok, "response data missing: %s", recorder.Body.String())
	return data
}

func TestSuperPrioritySettingsHandler_ReadsAndUpdatesBaseStrategy(t *testing.T) {
	router, cfg := newSuperPrioritySettingsTestRouter(t)

	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, httptest.NewRequest(http.MethodGet, "/super-priority", nil))
	require.Equal(t, http.StatusOK, getRecorder.Code)
	initial := decodeSuperPriorityResponse(t, getRecorder)
	require.Equal(t, "default", initial["base_strategy"])
	require.Equal(t, false, initial["liveness_include_unschedulable"])

	body := []byte(`{"base_strategy":"lowest_cost","failure_threshold":3,"check_interval":"@every 2m","liveness_include_unschedulable":true,"test_model_id":"gpt-test","test_prompt":"ping"}`)
	putRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/super-priority", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(putRecorder, request)
	require.Equal(t, http.StatusOK, putRecorder.Code, putRecorder.Body.String())
	require.Equal(t, service.AccountSchedulingStrategyLowestCost, cfg.SuperPriority.BaseStrategy)
	require.True(t, cfg.SuperPriority.LivenessIncludeUnschedulable)

	activateRecorder := httptest.NewRecorder()
	router.ServeHTTP(activateRecorder, httptest.NewRequest(http.MethodPost, "/super-priority/activate", nil))
	require.Equal(t, http.StatusOK, activateRecorder.Code, activateRecorder.Body.String())
	require.Equal(t, "super_priority", cfg.SuperPriority.Mode)

	written, err := os.ReadFile(filepath.Join(os.Getenv("DATA_DIR"), "config.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(written), "base_strategy: lowest_cost")
	require.Contains(t, string(written), "liveness_include_unschedulable: true")
	require.NotContains(t, string(written), "liveness_abnormal_only")
}

func TestSuperPrioritySettingsHandler_RejectsUnknownBaseStrategy(t *testing.T) {
	router, _ := newSuperPrioritySettingsTestRouter(t)
	body := []byte(`{"base_strategy":"random","failure_threshold":2,"check_interval":"@every 1m"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/super-priority", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestSchedulingRulesRefreshRunsLivenessAndBillingRefreshers(t *testing.T) {
	liveness := &schedulingRefreshStub{result: service.SchedulingRefreshResult{Checked: 3, Succeeded: 2, Failed: 1}}
	billing := &upstreamBillingRefreshStub{result: service.SchedulingRefreshResult{Checked: 2, Succeeded: 2}}
	handler := NewSettingHandler(nil, nil, nil, nil, nil, nil, nil)
	handler.SetSchedulingRefreshers(liveness, billing)
	router := gin.New()
	router.POST("/scheduling-rules/refresh", handler.RefreshSchedulingRules)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/scheduling-rules/refresh", nil))

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, 1, liveness.calls)
	require.Equal(t, 1, billing.calls)
	data := decodeSuperPriorityResponse(t, recorder)
	require.Equal(t, float64(3), data["liveness"].(map[string]any)["checked"])
	require.Equal(t, float64(2), data["upstream_billing"].(map[string]any)["checked"])
}

func TestSuperPrioritySettingsHandlerReturnsLivenessRuntimeStatus(t *testing.T) {
	startedAt := time.Date(2026, 7, 27, 20, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	nextRunAt := startedAt.Add(5 * time.Minute)
	liveness := &schedulingRefreshStub{status: service.SchedulingLivenessRuntimeStatus{
		Enabled:   true,
		NextRunAt: &nextRunAt,
		LastRun: &service.SchedulingLivenessRunStatus{
			Trigger: "scheduled", StartedAt: startedAt, FinishedAt: startedAt.Add(10 * time.Second),
			Result: service.SchedulingRefreshResult{Checked: 4, Succeeded: 2, Failed: 1, Skipped: 1},
		},
	}}
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{
		BaseStrategy: service.AccountSchedulingStrategyLowestCost, LivenessIncludeUnschedulable: true,
	}}
	handler := NewSettingHandler(nil, nil, nil, nil, nil, nil, nil)
	handler.SetSuperPriorityService(service.NewSuperPriorityService(nil, cfg))
	handler.SetSchedulingRefreshers(liveness, &upstreamBillingRefreshStub{})
	router := gin.New()
	router.GET("/super-priority", handler.GetSuperPrioritySettings)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/super-priority", nil))

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	data := decodeSuperPriorityResponse(t, recorder)
	require.Equal(t, true, data["liveness_include_unschedulable"])
	runtime := data["liveness_runtime"].(map[string]any)
	require.Equal(t, true, runtime["enabled"])
	require.Equal(t, nextRunAt.Format(time.RFC3339), runtime["next_run_at"])
	lastRun := runtime["last_run"].(map[string]any)
	require.Equal(t, "scheduled", lastRun["trigger"])
	require.Equal(t, float64(1), lastRun["result"].(map[string]any)["skipped"])
}
