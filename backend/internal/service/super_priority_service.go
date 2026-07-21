package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"gopkg.in/yaml.v3"
)

// SuperPriorityExtraKey marks accounts in the optional first-choice pool.
const SuperPriorityExtraKey = "super_priority"

const (
	superPriorityModeNormal        = "normal"
	superPriorityModeSuperPriority = "super_priority"
)

// SuperPriorityService maintains the super-priority overlay state. It never
// changes an account's persisted schedulable flag; schedulers fall back to the
// configured base strategy when preferred accounts cannot take a request.
type SuperPriorityService struct {
	accountRepo AccountRepository
	cfg         *config.Config

	mu sync.Mutex

	// failureWindows: accountID -> 最近失败时间戳列表（仅保留 1 分钟内）。
	failureWindows map[int64][]time.Time

	// persistFunc 持久化当前 super_priority 段落到 YAML。
	// 测试可覆盖以避免真实文件 IO；生产环境默认写盘。
	persistFunc func() error
}

// NewSuperPriorityService 创建状态机服务。
func NewSuperPriorityService(accountRepo AccountRepository, cfg *config.Config) *SuperPriorityService {
	s := &SuperPriorityService{
		accountRepo:    accountRepo,
		cfg:            cfg,
		failureWindows: make(map[int64][]time.Time),
	}
	s.persistFunc = s.persistConfigImpl
	return s
}

// Configured 报告服务是否已注入配置（供 handler 做 nil 安全检查）。
func (s *SuperPriorityService) Configured() bool {
	return s != nil && s.cfg != nil
}

// Mode 返回当前模式（normal / super_priority）。
func (s *SuperPriorityService) Mode() string {
	if s == nil || s.cfg == nil {
		return superPriorityModeNormal
	}
	mode := strings.TrimSpace(s.cfg.SuperPriority.Mode)
	if mode == "" {
		return superPriorityModeNormal
	}
	return mode
}

// IsActive 报告当前是否处于超级优先模式。
func (s *SuperPriorityService) IsActive() bool {
	return s.Mode() == superPriorityModeSuperPriority
}

// BaseStrategy returns the normalized base selection strategy.
func (s *SuperPriorityService) BaseStrategy() string {
	if s == nil || s.cfg == nil {
		return AccountSchedulingStrategyDefault
	}
	return normalizeAccountSchedulingStrategy(s.cfg.SuperPriority.BaseStrategy)
}

// FailureThreshold 返回降级阈值（默认 2）。
func (s *SuperPriorityService) FailureThreshold() int {
	if s == nil || s.cfg == nil || s.cfg.SuperPriority.FailureThreshold <= 0 {
		return 2
	}
	return s.cfg.SuperPriority.FailureThreshold
}

// CheckInterval 返回定时探测表达式（默认 @every 1m）。
func (s *SuperPriorityService) CheckInterval() string {
	if s == nil || s.cfg == nil || s.cfg.SuperPriority.CheckInterval == "" {
		return "@every 1m"
	}
	return s.cfg.SuperPriority.CheckInterval
}

// TestModelID 返回探测使用的模型 ID（空表示复用平台默认）。
func (s *SuperPriorityService) TestModelID() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.SuperPriority.TestModelID
}

// TestPrompt 返回探测 prompt。
func (s *SuperPriorityService) TestPrompt() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.SuperPriority.TestPrompt
}

// ActivatedAt 返回最近一次激活时间戳字符串。
func (s *SuperPriorityService) ActivatedAt() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.SuperPriority.ActivatedAt
}

// DemotedAt 返回最近一次降级时间戳字符串。
func (s *SuperPriorityService) DemotedAt() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.SuperPriority.DemotedAt
}

// UpdateRuntimeParams 更新可热更新的运行参数（阈值/间隔/测试模型/prompt）。
func (s *SuperPriorityService) UpdateRuntimeParams(failureThreshold int, checkInterval, testModelID, testPrompt, baseStrategy string) {
	if s == nil || s.cfg == nil {
		return
	}
	if failureThreshold < 1 {
		failureThreshold = 2
	}
	if checkInterval == "" {
		checkInterval = "@every 1m"
	}
	s.cfg.SuperPriority.FailureThreshold = failureThreshold
	s.cfg.SuperPriority.CheckInterval = checkInterval
	s.cfg.SuperPriority.TestModelID = testModelID
	s.cfg.SuperPriority.TestPrompt = testPrompt
	s.cfg.SuperPriority.BaseStrategy = normalizeAccountSchedulingStrategy(baseStrategy)
}

// Activate enables the request-time preference overlay.
func (s *SuperPriorityService) Activate(_ context.Context) error {
	if s == nil || s.cfg == nil {
		return fmt.Errorf("super priority service not configured")
	}

	s.cfg.SuperPriority.Mode = superPriorityModeSuperPriority
	s.cfg.SuperPriority.ActivatedAt = time.Now().Format(time.RFC3339)
	s.cfg.SuperPriority.DemotedAt = ""

	s.resetFailures()

	return s.persistConfig()
}

// Deactivate removes the overlay and leaves account state untouched.
// reason 用于日志。
func (s *SuperPriorityService) Deactivate(_ context.Context, reason string) error {
	if s == nil || s.cfg == nil {
		return fmt.Errorf("super priority service not configured")
	}

	s.cfg.SuperPriority.Mode = superPriorityModeNormal
	s.cfg.SuperPriority.DemotedAt = time.Now().Format(time.RFC3339)

	s.resetFailures()

	logger.LegacyPrintf("service.super_priority", "[SuperPriority] overlay disabled (reason=%s, base_strategy=%s)", reason, s.BaseStrategy())

	return s.persistConfig()
}

// RecordFailure 记录某超级优先账号一次探测失败。
// 若该账号在一分钟滚动窗口内失败次数达到阈值，返回 shouldDemote=true。
// 调用方（runner）应在收到 true 时调用 Deactivate。
func (s *SuperPriorityService) RecordFailure(accountID int64) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)
	window := s.failureWindows[accountID]
	// 丢弃超过 1 分钟的旧记录。
	kept := window[:0]
	for _, t := range window {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	kept = append(kept, now)
	s.failureWindows[accountID] = kept

	return len(kept) >= s.FailureThreshold()
}

// resetFailures 清空所有失败窗口（用于激活/降级时重置）。
func (s *SuperPriorityService) resetFailures() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failureWindows = make(map[int64][]time.Time)
}

// persistConfig 委托到可覆盖的 persistFunc（默认写盘），便于测试注入。
func (s *SuperPriorityService) persistConfig() error {
	if s.persistFunc != nil {
		return s.persistFunc()
	}
	return s.persistConfigImpl()
}

// persistConfigImpl 将当前 super_priority 段落写回 YAML 配置文件（mirror balance_check 习惯）。
func (s *SuperPriorityService) persistConfigImpl() error {
	path := superPriorityConfigFilePath()
	return saveSuperPriorityConfig(path, &s.cfg.SuperPriority)
}

// PersistConfig 导出持久化入口，供 handler 更新运行参数后调用。
func (s *SuperPriorityService) PersistConfig() error {
	return s.persistConfig()
}

// SetAccountFlag 标记/取消标记账号为超级优先（写 extra JSONB）。
func (s *SuperPriorityService) SetAccountFlag(ctx context.Context, accountID int64, enabled bool) error {
	return s.accountRepo.UpdateExtra(ctx, accountID, map[string]any{SuperPriorityExtraKey: enabled})
}

// --- 配置文件读写（自包含，避免依赖 handler/admin） ---

func superPriorityConfigFilePath() string {
	if dataDir := strings.TrimSpace(os.Getenv("DATA_DIR")); dataDir != "" {
		return absoluteSuperPriorityPath(filepath.Join(dataDir, "config.yaml"))
	}
	if _, err := os.Stat(filepath.Join("data", "config.yaml")); err == nil {
		return absoluteSuperPriorityPath(filepath.Join("data", "config.yaml"))
	}
	return absoluteSuperPriorityPath("config.yaml")
}

func absoluteSuperPriorityPath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// saveSuperPriorityConfig 仅更新 YAML 根下的 super_priority 段落，保留其它键。
// 自包含实现：直接读/解析/改/原子写，不引入 handler/admin 依赖。
func saveSuperPriorityConfig(path string, cfg *config.SuperPriorityConfig) error {
	if cfg == nil {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			content = nil
		} else {
			return fmt.Errorf("read config %s: %w", path, err)
		}
	}

	root := map[string]any{}
	if len(strings.TrimSpace(string(content))) > 0 {
		if err := yaml.Unmarshal(content, &root); err != nil {
			return fmt.Errorf("parse config %s: %w", path, err)
		}
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal super_priority: %w", err)
	}
	var section map[string]any
	if err := yaml.Unmarshal(out, &section); err != nil {
		return fmt.Errorf("unmarshal super_priority section: %w", err)
	}
	root["super_priority"] = section

	newContent, err := yaml.Marshal(root)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	mode := os.FileMode(0o640)
	if stat, statErr := os.Stat(path); statErr == nil {
		mode = stat.Mode().Perm()
	}
	return writeFileAtomicSuperPriority(path, newContent, mode)
}

// writeFileAtomicSuperPriority 写入临时文件后原子替换，避免半写损坏。
func writeFileAtomicSuperPriority(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".super_priority-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	return os.Rename(tmpName, path)
}

// getExtraBool 安全读取 extra map 中的 bool 值。
func getExtraBool(extra map[string]any, key string) bool {
	if extra == nil {
		return false
	}
	v, ok := extra[key]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}
