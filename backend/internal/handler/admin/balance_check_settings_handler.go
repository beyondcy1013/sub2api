package admin

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type balanceCheckSettingsConfig struct {
	Enabled                 bool    `json:"enabled" yaml:"enabled"`
	Interval                string  `json:"interval" yaml:"interval"`
	BalanceURL              string  `json:"balance_url" yaml:"balance_url"`
	RequestTimeoutSeconds   int     `json:"request_timeout_seconds" yaml:"request_timeout_seconds"`
	MaxConcurrentChecks     int     `json:"max_concurrent_checks" yaml:"max_concurrent_checks"`
	PauseDurationHours      float64 `json:"pause_duration_hours" yaml:"pause_duration_hours"`
	MinDecrease             float64 `json:"min_decrease" yaml:"min_decrease"`
	PauseWhenCurrentBelow   float64 `json:"pause_when_current_below" yaml:"pause_when_current_below"`
	PauseWhenDropPercent    float64 `json:"pause_when_drop_percent" yaml:"pause_when_drop_percent"`
	StopWhenCurrentBelow    float64 `json:"stop_when_current_below" yaml:"stop_when_current_below"`
	ResumeWhenCurrentAbove  float64 `json:"resume_when_current_above" yaml:"resume_when_current_above"`
	RequireQuotaHourlyLimit bool    `json:"require_quota_hourly_limit" yaml:"require_quota_hourly_limit"`
}

type balanceCheckSettingsResponse struct {
	Config          balanceCheckSettingsConfig `json:"config"`
	ConfigPath      string                     `json:"config_path"`
	RestartRequired bool                       `json:"restart_required"`
}

func defaultBalanceCheckSettingsConfig() balanceCheckSettingsConfig {
	return balanceCheckSettingsConfig{
		Enabled:                 true,
		Interval:                "@every 5m",
		BalanceURL:              "https://ai.router.team/api/public/cc-switch/balance",
		RequestTimeoutSeconds:   30,
		MaxConcurrentChecks:     1,
		PauseDurationHours:      5,
		MinDecrease:             5,
		PauseWhenCurrentBelow:   0,
		PauseWhenDropPercent:    0,
		StopWhenCurrentBelow:    0,
		ResumeWhenCurrentAbove:  0,
		RequireQuotaHourlyLimit: true,
	}
}

// GetBalanceCheckSettings returns the YAML-backed balance check settings.
// GET /api/v1/admin/settings/balance-check
func (h *SettingHandler) GetBalanceCheckSettings(c *gin.Context) {
	path := balanceCheckConfigFilePath()
	cfg, _, err := loadBalanceCheckSettings(path)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, balanceCheckSettingsResponse{
		Config:          cfg,
		ConfigPath:      path,
		RestartRequired: false,
	})
}

// UpdateBalanceCheckSettings updates only the balance_check section in config.yaml.
// PUT /api/v1/admin/settings/balance-check
func (h *SettingHandler) UpdateBalanceCheckSettings(c *gin.Context) {
	var req balanceCheckSettingsConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	normalized, err := normalizeBalanceCheckSettings(req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	path := balanceCheckConfigFilePath()
	if err := saveBalanceCheckSettings(path, normalized); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, balanceCheckSettingsResponse{
		Config:          normalized,
		ConfigPath:      path,
		RestartRequired: true,
	})
}

func balanceCheckConfigFilePath() string {
	if dataDir := strings.TrimSpace(os.Getenv("DATA_DIR")); dataDir != "" {
		return absolutePath(filepath.Join(dataDir, "config.yaml"))
	}
	if _, err := os.Stat(filepath.Join("data", "config.yaml")); err == nil {
		return absolutePath(filepath.Join("data", "config.yaml"))
	}
	return absolutePath("config.yaml")
}

func absolutePath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

func loadBalanceCheckSettings(path string) (balanceCheckSettingsConfig, map[string]any, error) {
	root, _, err := readYAMLConfigFile(path)
	if err != nil {
		return balanceCheckSettingsConfig{}, nil, err
	}
	cfg := defaultBalanceCheckSettingsConfig()
	if raw, ok := root["balance_check"]; ok && raw != nil {
		section, err := yaml.Marshal(raw)
		if err != nil {
			return balanceCheckSettingsConfig{}, nil, fmt.Errorf("marshal balance_check section: %w", err)
		}
		if err := yaml.Unmarshal(section, &cfg); err != nil {
			return balanceCheckSettingsConfig{}, nil, fmt.Errorf("parse balance_check section: %w", err)
		}
	}
	normalized, err := normalizeBalanceCheckSettings(cfg)
	if err != nil {
		return balanceCheckSettingsConfig{}, nil, err
	}
	return normalized, root, nil
}

func saveBalanceCheckSettings(path string, cfg balanceCheckSettingsConfig) error {
	root, mode, err := readYAMLConfigFile(path)
	if err != nil {
		return err
	}
	root["balance_check"] = cfg

	content, err := yaml.Marshal(root)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if mode == 0 {
		mode = 0o640
	}
	return writeFileAtomic(path, content, mode)
}

func readYAMLConfigFile(path string) (map[string]any, os.FileMode, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, 0, nil
		}
		return nil, 0, fmt.Errorf("read config %s: %w", path, err)
	}

	root := map[string]any{}
	if len(strings.TrimSpace(string(content))) > 0 {
		if err := yaml.Unmarshal(content, &root); err != nil {
			return nil, 0, fmt.Errorf("parse config %s: %w", path, err)
		}
	}
	stat, err := os.Stat(path)
	if err != nil {
		return nil, 0, fmt.Errorf("stat config %s: %w", path, err)
	}
	return root, stat.Mode().Perm(), nil
}

func writeFileAtomic(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create config dir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace config %s: %w", path, err)
	}
	return nil
}

func normalizeBalanceCheckSettings(cfg balanceCheckSettingsConfig) (balanceCheckSettingsConfig, error) {
	cfg.Interval = strings.TrimSpace(cfg.Interval)
	cfg.BalanceURL = strings.TrimSpace(cfg.BalanceURL)

	if cfg.Interval == "" {
		return cfg, fmt.Errorf("interval is required")
	}
	if err := config.ValidateAbsoluteHTTPURL(cfg.BalanceURL); err != nil {
		return cfg, fmt.Errorf("balance_url is invalid: %w", err)
	}
	if cfg.RequestTimeoutSeconds <= 0 {
		return cfg, fmt.Errorf("request_timeout_seconds must be greater than 0")
	}
	if cfg.MaxConcurrentChecks <= 0 {
		return cfg, fmt.Errorf("max_concurrent_checks must be greater than 0")
	}
	if cfg.PauseDurationHours <= 0 {
		return cfg, fmt.Errorf("pause_duration_hours must be greater than 0")
	}
	if cfg.MinDecrease < 0 {
		return cfg, fmt.Errorf("min_decrease must be greater than or equal to 0")
	}
	if cfg.PauseWhenCurrentBelow < 0 {
		return cfg, fmt.Errorf("pause_when_current_below must be greater than or equal to 0")
	}
	if cfg.PauseWhenDropPercent < 0 {
		return cfg, fmt.Errorf("pause_when_drop_percent must be greater than or equal to 0")
	}
	if cfg.StopWhenCurrentBelow < 0 {
		return cfg, fmt.Errorf("stop_when_current_below must be greater than or equal to 0")
	}
	if cfg.ResumeWhenCurrentAbove < 0 {
		return cfg, fmt.Errorf("resume_when_current_above must be greater than or equal to 0")
	}
	return cfg, nil
}
