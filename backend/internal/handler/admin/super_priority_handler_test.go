package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

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
	require.Equal(t, "default", decodeSuperPriorityResponse(t, getRecorder)["base_strategy"])

	body := []byte(`{"base_strategy":"lowest_cost","failure_threshold":3,"check_interval":"@every 2m","test_model_id":"gpt-test","test_prompt":"ping"}`)
	putRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/super-priority", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(putRecorder, request)
	require.Equal(t, http.StatusOK, putRecorder.Code, putRecorder.Body.String())
	require.Equal(t, service.AccountSchedulingStrategyLowestCost, cfg.SuperPriority.BaseStrategy)

	activateRecorder := httptest.NewRecorder()
	router.ServeHTTP(activateRecorder, httptest.NewRequest(http.MethodPost, "/super-priority/activate", nil))
	require.Equal(t, http.StatusOK, activateRecorder.Code, activateRecorder.Body.String())
	require.Equal(t, "super_priority", cfg.SuperPriority.Mode)

	written, err := os.ReadFile(filepath.Join(os.Getenv("DATA_DIR"), "config.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(written), "base_strategy: lowest_cost")
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
