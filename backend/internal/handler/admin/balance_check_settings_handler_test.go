package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newBalanceCheckSettingsTestRouter(h *SettingHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/balance-check", h.GetBalanceCheckSettings)
	r.PUT("/balance-check", h.UpdateBalanceCheckSettings)
	return r
}

func writeBalanceCheckTestConfig(t *testing.T, dataDir string, body string) string {
	t.Helper()
	path := filepath.Join(dataDir, "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o640); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func decodeBalanceCheckResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v\nbody=%s", err, w.Body.String())
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data object: %#v", payload)
	}
	return data
}

func TestBalanceCheckSettingsHandler_GetReadsConfigFile(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	configPath := writeBalanceCheckTestConfig(t, dataDir, `
server:
  port: 18382
balance_check:
  enabled: true
  interval: "@every 7m"
  balance_url: "https://example.com/balance"
  request_timeout_seconds: 11
  max_concurrent_checks: 3
  pause_duration_hours: 4.5
  min_decrease: 8.25
  pause_when_current_below: 2
  pause_when_drop_percent: 12.5
  stop_when_current_below: 1
  resume_when_current_above: 10
  require_quota_hourly_limit: false
`)

	r := newBalanceCheckSettingsTestRouter(NewSettingHandler(nil, nil, nil, nil, nil, nil, nil))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/balance-check", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
	data := decodeBalanceCheckResponse(t, w)
	if data["config_path"] != configPath {
		t.Fatalf("config_path=%v, want %s", data["config_path"], configPath)
	}
	if data["restart_required"] != false {
		t.Fatalf("restart_required=%v, want false", data["restart_required"])
	}
	cfg, ok := data["config"].(map[string]any)
	if !ok {
		t.Fatalf("missing config object: %#v", data)
	}
	if cfg["interval"] != "@every 7m" || cfg["balance_url"] != "https://example.com/balance" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if cfg["require_quota_hourly_limit"] != false {
		t.Fatalf("require_quota_hourly_limit=%v, want false", cfg["require_quota_hourly_limit"])
	}
	if cfg["stop_when_current_below"] != float64(1) || cfg["resume_when_current_above"] != float64(10) {
		t.Fatalf("unexpected stop/resume config: %#v", cfg)
	}
}

func TestBalanceCheckSettingsHandler_UpdateWritesBalanceCheckSection(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	configPath := writeBalanceCheckTestConfig(t, dataDir, `
server:
  port: 18382
database:
  dbname: sub2freeApi
balance_check:
  enabled: true
  interval: "@every 5m"
`)

	body := `{
  "enabled": false,
  "interval": "@every 2m",
  "balance_url": "https://example.com/new-balance",
  "request_timeout_seconds": 9,
  "max_concurrent_checks": 2,
  "pause_duration_hours": 1.5,
  "min_decrease": 3,
  "pause_when_current_below": 1,
  "pause_when_drop_percent": 20,
  "stop_when_current_below": 0.5,
  "resume_when_current_above": 5,
  "require_quota_hourly_limit": false
}`
	r := newBalanceCheckSettingsTestRouter(NewSettingHandler(nil, nil, nil, nil, nil, nil, nil))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/balance-check", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
	data := decodeBalanceCheckResponse(t, w)
	if data["restart_required"] != true {
		t.Fatalf("restart_required=%v, want true", data["restart_required"])
	}
	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}
	text := string(written)
	for _, want := range []string{
		"dbname: sub2freeApi",
		"balance_check:",
		"enabled: false",
		"interval: '@every 2m'",
		"balance_url: https://example.com/new-balance",
		"stop_when_current_below: 0.5",
		"resume_when_current_above: 5",
		"require_quota_hourly_limit: false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("written config missing %q:\n%s", want, text)
		}
	}
}

func TestBalanceCheckSettingsHandler_UpdateRejectsInvalidValues(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	writeBalanceCheckTestConfig(t, dataDir, "server:\n  port: 18382\n")

	r := newBalanceCheckSettingsTestRouter(NewSettingHandler(nil, nil, nil, nil, nil, nil, nil))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/balance-check", bytes.NewBufferString(`{
  "enabled": true,
  "interval": "",
  "balance_url": "file:///tmp/balance",
  "request_timeout_seconds": -1,
  "max_concurrent_checks": 0,
  "pause_duration_hours": 0,
  "min_decrease": -1,
  "pause_when_current_below": -1,
  "pause_when_drop_percent": -1,
  "stop_when_current_below": -1,
  "resume_when_current_above": -1,
  "require_quota_hourly_limit": true
}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400, body=%s", w.Code, w.Body.String())
	}
}
