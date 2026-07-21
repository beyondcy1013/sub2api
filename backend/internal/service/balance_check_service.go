package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/robfig/cron/v3"
)

const defaultBalanceCheckInterval = "@every 5m" // Run every 5 minutes
const defaultBalanceCheckTimeout = 30 * time.Second
const defaultPauseDuration = 5 * time.Hour
const defaultBalanceDecreasePauseThreshold = 5.0
const defaultBalanceCheckURL = "https://ai.router.team/api/public/cc-switch/balance"
const defaultBalanceCheckMaxConcurrent = 1
const balanceCheckPauseReasonMarker = "auto-pause by balance check"
const balanceCheckStopReasonMarker = "auto-stop by balance check"
const maxBalanceCheckResponseBytes int64 = 1 << 20

const (
	BalanceCheckTypeExtraKey      = "balance_check_type"
	BalanceCheckTypeConfiguredAPI = "configured_api"
	BalanceCheckTypeSub2API       = "sub2api"
)

// BalanceCheckResult represents the result of a balance check
type BalanceCheckResult struct {
	AccountID   int64
	Platform    string
	APIKey      string
	BaseURL     string
	PreviousBal float64
	CurrentBal  float64
	IsDecreased bool
	CheckType   string
	Error       string
}

// BalanceCheckService monitors account balances and pauses accounts when balance decreases
type BalanceCheckService struct {
	accountRepo  AccountRepository
	cfg          *config.Config
	httpClient   *http.Client
	cron         *cron.Cron
	startOnce    sync.Once
	stopOnce     sync.Once
	balanceCache map[int64]float64 // accountID -> last known balance
	cacheMu      sync.RWMutex
}

// NewBalanceCheckService creates a new balance check service
func NewBalanceCheckService(
	accountRepo AccountRepository,
	cfg *config.Config,
) *BalanceCheckService {
	runtimeCfg := resolveBalanceCheckRuntimeConfig(cfg)
	return &BalanceCheckService{
		accountRepo: accountRepo,
		cfg:         cfg,
		httpClient: &http.Client{
			Timeout: runtimeCfg.RequestTimeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		balanceCache: make(map[int64]float64),
	}
}

// Start begins the cron scheduler for balance checking
func (s *BalanceCheckService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		if s.cfg != nil && !s.cfg.RuntimeCapabilities().BalanceCheck {
			logger.LegacyPrintf("service.balance_check", "[BalanceCheck] disabled by deployment profile")
			return
		}
		runtimeCfg := s.runtimeConfig()
		if !runtimeCfg.Enabled {
			logger.LegacyPrintf("service.balance_check", "[BalanceCheck] disabled")
			return
		}
		loc := time.Local
		if s.cfg != nil && strings.TrimSpace(s.cfg.Timezone) != "" {
			if parsed, err := time.LoadLocation(strings.TrimSpace(s.cfg.Timezone)); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithLocation(loc))
		_, err := c.AddFunc(runtimeCfg.Interval, func() { s.runBalanceCheck() })
		if err != nil {
			logger.LegacyPrintf("service.balance_check", "[BalanceCheck] not started (invalid schedule): %v", err)
			return
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.balance_check", "[BalanceCheck] started (tick=%s)", runtimeCfg.Interval)
		go s.runBalanceCheck()
	})
}

// Stop gracefully shuts down the cron scheduler
func (s *BalanceCheckService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.balance_check", "[BalanceCheck] cron stop timed out")
			}
		}
	})
}

func (s *BalanceCheckService) runBalanceCheck() {
	runtimeCfg := s.runtimeConfig()
	if !runtimeCfg.Enabled {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// First, check and resume any paused accounts that have exceeded 5 hours.
	// Keep balance-check paused accounts in the refresh set so the displayed balance
	// does not stay stale during the pause window.
	pausedAccounts := s.checkAndResumePausedAccounts(ctx)

	// Get all active accounts
	accounts, err := s.accountRepo.ListSchedulable(ctx)
	if err != nil {
		logger.LegacyPrintf("service.balance_check", "[BalanceCheck] ListSchedulable error: %v", err)
		return
	}

	logger.LegacyPrintf("service.balance_check", "[BalanceCheck] checking %d accounts", len(accounts))

	var wg sync.WaitGroup
	sem := make(chan struct{}, runtimeCfg.MaxConcurrentChecks)

	for _, account := range mergeBalanceCheckAccounts(accounts, pausedAccounts) {
		if !runtimeCfg.isAccountEnabled(account) {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(acc Account) {
			defer wg.Done()
			defer func() { <-sem }()
			s.checkAccountBalance(ctx, &acc)
		}(account)
	}

	wg.Wait()
}

// checkAndResumePausedAccounts checks all balance-check paused accounts with
// temp_unschedulable_until set, resumes expired ones, and returns them for balance refresh.
func (s *BalanceCheckService) checkAndResumePausedAccounts(ctx context.Context) []Account {
	// Find accounts that are temporarily unschedulable
	accounts := s.listTempUnschedulableAccounts(ctx)
	if len(accounts) == 0 {
		return nil
	}

	balanceCheckPaused := make([]Account, 0, len(accounts))
	now := time.Now()
	for _, account := range accounts {
		if !isBalanceCheckHeldAccount(account) {
			continue
		}
		balanceCheckPaused = append(balanceCheckPaused, account)
		if strings.Contains(account.TempUnschedulableReason, balanceCheckStopReasonMarker) {
			continue
		}
		if account.TempUnschedulableUntil == nil {
			continue
		}

		// Check if pause duration has exceeded
		if now.After(*account.TempUnschedulableUntil) {
			logger.LegacyPrintf("service.balance_check",
				"[BalanceCheck] account=%d (%s) pause expired, resuming",
				account.ID, account.Name)

			if err := s.accountRepo.ClearTempUnschedulable(ctx, account.ID); err != nil {
				logger.LegacyPrintf("service.balance_check", "[BalanceCheck] failed to resume account=%d: %v", account.ID, err)
			} else {
				logger.LegacyPrintf("service.balance_check", "[BalanceCheck] account=%d resumed", account.ID)
			}
		}
	}
	return balanceCheckPaused
}

func (s *BalanceCheckService) listTempUnschedulableAccounts(ctx context.Context) []Account {
	const pageSize = 1000
	var accounts []Account
	for page := 1; ; page++ {
		batch, result, err := s.accountRepo.ListWithFilters(
			ctx,
			pagination.PaginationParams{Page: page, PageSize: pageSize},
			"",
			"",
			"temp_unschedulable",
			"",
			0,
			"",
			false,
		)
		if err != nil {
			logger.LegacyPrintf("service.balance_check", "[BalanceCheck] ListWithFilters temp_unschedulable error: %v", err)
			return nil
		}
		accounts = append(accounts, batch...)
		if len(batch) == 0 || result == nil || page >= result.Pages {
			return accounts
		}
	}
}

func (s *BalanceCheckService) checkAccountBalance(ctx context.Context, account *Account) {
	runtimeCfg := s.runtimeConfig()
	result := s.fetchBalance(ctx, account)
	if result == nil {
		return
	}

	if result.Error != "" {
		logger.LegacyPrintf("service.balance_check", "[BalanceCheck] account=%d fetch error: %s", account.ID, result.Error)
		return
	}

	// Get previous balance from cache, falling back to the last persisted value.
	// The fallback keeps decrease detection working after a service restart.
	s.cacheMu.Lock()
	previousBal, exists := s.balanceCache[account.ID]
	s.cacheMu.Unlock()
	if !exists {
		previousBal, exists = accountExtraFloat64(account.Extra, "balance")
	}

	if runtimeCfg.shouldResumeStopped(account, result.CurrentBal) {
		logger.LegacyPrintf("service.balance_check",
			"[BalanceCheck] account=%d (%s) balance recovered to %.4f, resuming",
			account.ID, account.Name, result.CurrentBal)
		if err := s.accountRepo.ClearTempUnschedulable(ctx, account.ID); err != nil {
			logger.LegacyPrintf("service.balance_check", "[BalanceCheck] failed to resume stopped account=%d: %v", account.ID, err)
		}
	} else if runtimeCfg.isStopped(account) {
		s.updateBalanceAfterCheck(ctx, account.ID, result.CurrentBal, result.CheckType)
		return
	}

	stop, stopReasonDetail := runtimeCfg.shouldStop(account, result.CurrentBal)
	if stop {
		stopUntil := time.Now().AddDate(100, 0, 0)
		reason := fmt.Sprintf("%s (%s)", stopReasonDetail, balanceCheckStopReasonMarker)
		logger.LegacyPrintf("service.balance_check",
			"[BalanceCheck] account=%d (%s) balance trigger: %s, stopping",
			account.ID, account.Name, stopReasonDetail)
		if err := s.accountRepo.SetTempUnschedulable(ctx, account.ID, stopUntil, reason); err != nil {
			logger.LegacyPrintf("service.balance_check", "[BalanceCheck] failed to stop account=%d: %v", account.ID, err)
		} else {
			logger.LegacyPrintf("service.balance_check", "[BalanceCheck] account=%d stopped until %v", account.ID, stopUntil)
		}
		s.updateBalanceAfterCheck(ctx, account.ID, result.CurrentBal, result.CheckType)
		return
	}

	// If balance exists and decreased beyond the allowed fluctuation, pause the account for a configured cycle.
	pause, reasonDetail := runtimeCfg.shouldPause(account, exists, previousBal, result.CurrentBal)
	if pause {
		logger.LegacyPrintf("service.balance_check",
			"[BalanceCheck] account=%d (%s) balance trigger: %s, pausing for %s",
			account.ID, account.Name, reasonDetail, runtimeCfg.PauseDuration)

		pauseUntil := time.Now().Add(runtimeCfg.PauseDuration)
		reason := fmt.Sprintf("%s (%s)", reasonDetail, balanceCheckPauseReasonMarker)

		if err := s.accountRepo.SetTempUnschedulable(ctx, account.ID, pauseUntil, reason); err != nil {
			logger.LegacyPrintf("service.balance_check", "[BalanceCheck] failed to pause account=%d: %v", account.ID, err)
		} else {
			logger.LegacyPrintf("service.balance_check", "[BalanceCheck] account=%d paused until %v", account.ID, pauseUntil)
		}
	}

	s.updateBalanceAfterCheck(ctx, account.ID, result.CurrentBal, result.CheckType)
}

func (s *BalanceCheckService) updateBalanceAfterCheck(ctx context.Context, accountID int64, balance float64, checkType string) {
	updates := map[string]any{"balance": balance}
	if checkType != "" {
		updates[BalanceCheckTypeExtraKey] = checkType
	}
	if err := s.accountRepo.UpdateExtra(ctx, accountID, updates); err != nil {
		logger.LegacyPrintf("service.balance_check", "[BalanceCheck] failed to update balance for account=%d: %v", accountID, err)
	}
	s.cacheMu.Lock()
	s.balanceCache[accountID] = balance
	s.cacheMu.Unlock()
}

func mergeBalanceCheckAccounts(primary, secondary []Account) []Account {
	merged := make([]Account, 0, len(primary)+len(secondary))
	seen := make(map[int64]struct{}, len(primary)+len(secondary))
	for _, account := range append(primary, secondary...) {
		if _, ok := seen[account.ID]; ok {
			continue
		}
		seen[account.ID] = struct{}{}
		merged = append(merged, account)
	}
	return merged
}

func isBalanceCheckEnabledAccount(account Account) bool {
	if account.Type != AccountTypeAPIKey {
		return false
	}
	quotaLimit, ok := accountExtraFloat64(account.Extra, "quota_hourly_limit")
	return ok && quotaLimit > 0
}

type balanceCheckRuntimeConfig struct {
	Enabled                 bool
	Interval                string
	BalanceURL              string
	RequestTimeout          time.Duration
	MaxConcurrentChecks     int
	PauseDuration           time.Duration
	MinDecrease             float64
	PauseWhenCurrentBelow   float64
	PauseWhenDropPercent    float64
	StopWhenCurrentBelow    float64
	ResumeWhenCurrentAbove  float64
	RequireQuotaHourlyLimit bool
}

func (s *BalanceCheckService) runtimeConfig() balanceCheckRuntimeConfig {
	if s == nil {
		return resolveBalanceCheckRuntimeConfig(nil)
	}
	return resolveBalanceCheckRuntimeConfig(s.cfg)
}

func resolveBalanceCheckRuntimeConfig(cfg *config.Config) balanceCheckRuntimeConfig {
	out := balanceCheckRuntimeConfig{
		Enabled:                 true,
		Interval:                defaultBalanceCheckInterval,
		BalanceURL:              defaultBalanceCheckURL,
		RequestTimeout:          defaultBalanceCheckTimeout,
		MaxConcurrentChecks:     defaultBalanceCheckMaxConcurrent,
		PauseDuration:           defaultPauseDuration,
		MinDecrease:             defaultBalanceDecreasePauseThreshold,
		RequireQuotaHourlyLimit: true,
	}
	if cfg == nil {
		return out
	}
	raw := cfg.BalanceCheck
	if raw.Enabled != nil {
		out.Enabled = *raw.Enabled
	}
	if strings.TrimSpace(raw.Interval) != "" {
		out.Interval = strings.TrimSpace(raw.Interval)
	}
	if strings.TrimSpace(raw.BalanceURL) != "" {
		out.BalanceURL = strings.TrimSpace(raw.BalanceURL)
	}
	if raw.RequestTimeoutSeconds > 0 {
		out.RequestTimeout = time.Duration(raw.RequestTimeoutSeconds) * time.Second
	}
	if raw.MaxConcurrentChecks > 0 {
		out.MaxConcurrentChecks = raw.MaxConcurrentChecks
	}
	if raw.PauseDurationHours > 0 {
		out.PauseDuration = time.Duration(raw.PauseDurationHours * float64(time.Hour))
	}
	if raw.MinDecrease > 0 {
		out.MinDecrease = raw.MinDecrease
	}
	if raw.PauseWhenCurrentBelow > 0 {
		out.PauseWhenCurrentBelow = raw.PauseWhenCurrentBelow
	}
	if raw.PauseWhenDropPercent > 0 {
		out.PauseWhenDropPercent = raw.PauseWhenDropPercent
	}
	if raw.StopWhenCurrentBelow > 0 {
		out.StopWhenCurrentBelow = raw.StopWhenCurrentBelow
	}
	if raw.ResumeWhenCurrentAbove > 0 {
		out.ResumeWhenCurrentAbove = raw.ResumeWhenCurrentAbove
	}
	if raw.RequireQuotaHourlyLimit != nil {
		out.RequireQuotaHourlyLimit = *raw.RequireQuotaHourlyLimit
	}
	return out
}

func (c balanceCheckRuntimeConfig) isAccountEnabled(account Account) bool {
	if account.Type != AccountTypeAPIKey {
		return false
	}
	if accountExtraBool(account.Extra, "balance_check_disabled") {
		return false
	}
	checkType := accountBalanceCheckType(account)
	if checkType == BalanceCheckTypeSub2API {
		return accountHasSub2APIBalanceCredentials(account)
	}
	if !c.RequireQuotaHourlyLimit {
		return true
	}
	if checkType == "" && accountHasSub2APIBalanceCredentials(account) {
		return true
	}
	return isBalanceCheckEnabledAccount(account)
}

func accountBalanceCheckType(account Account) string {
	value, _ := account.Extra[BalanceCheckTypeExtraKey].(string)
	switch strings.ToLower(strings.TrimSpace(value)) {
	case BalanceCheckTypeSub2API:
		return BalanceCheckTypeSub2API
	case BalanceCheckTypeConfiguredAPI:
		return BalanceCheckTypeConfiguredAPI
	default:
		return ""
	}
}

func accountHasSub2APIBalanceCredentials(account Account) bool {
	return strings.TrimSpace(account.GetCredential("api_key")) != "" &&
		strings.TrimSpace(account.GetCredential("base_url")) != ""
}

func isBalanceCheckHeldAccount(account Account) bool {
	return strings.Contains(account.TempUnschedulableReason, balanceCheckPauseReasonMarker) ||
		strings.Contains(account.TempUnschedulableReason, balanceCheckStopReasonMarker)
}

func (c balanceCheckRuntimeConfig) isStopped(account *Account) bool {
	return account != nil && strings.Contains(account.TempUnschedulableReason, balanceCheckStopReasonMarker)
}

func (c balanceCheckRuntimeConfig) shouldResumeStopped(account *Account, currentBal float64) bool {
	return c.isStopped(account) && c.ResumeWhenCurrentAbove > 0 && currentBal >= c.ResumeWhenCurrentAbove
}

func (c balanceCheckRuntimeConfig) shouldStop(account *Account, currentBal float64) (bool, string) {
	if account == nil {
		return false, ""
	}
	stopBelow := c.StopWhenCurrentBelow
	if v, ok := accountExtraFloat64(account.Extra, "balance_stop_below"); ok && v > 0 {
		stopBelow = v
	}
	if stopBelow > 0 && currentBal < stopBelow {
		return true, fmt.Sprintf("Balance %.4f below stop threshold %.4f", currentBal, stopBelow)
	}
	return false, ""
}

func (c balanceCheckRuntimeConfig) shouldPause(account *Account, hasPrevious bool, previousBal, currentBal float64) (bool, string) {
	if account == nil {
		return false, ""
	}
	minDecrease := c.MinDecrease
	if v, ok := accountExtraFloat64(account.Extra, "balance_pause_min_decrease"); ok && v > 0 {
		minDecrease = v
	}
	pauseBelow := c.PauseWhenCurrentBelow
	if v, ok := accountExtraFloat64(account.Extra, "balance_pause_below"); ok && v > 0 {
		pauseBelow = v
	}
	dropPercent := c.PauseWhenDropPercent
	if v, ok := accountExtraFloat64(account.Extra, "balance_pause_drop_percent"); ok && v > 0 {
		dropPercent = v
	}

	if pauseBelow > 0 && currentBal < pauseBelow {
		return true, fmt.Sprintf("Balance %.4f below threshold %.4f", currentBal, pauseBelow)
	}
	if !hasPrevious || previousBal <= 0 {
		return false, ""
	}
	decrease := previousBal - currentBal
	if minDecrease > 0 && decrease > minDecrease {
		return true, fmt.Sprintf("Balance decreased from %.4f to %.4f, decreased by %.4f", previousBal, currentBal, decrease)
	}
	if dropPercent > 0 {
		percent := decrease / previousBal * 100
		if percent > dropPercent {
			return true, fmt.Sprintf("Balance decreased from %.4f to %.4f, dropped %.2f%% over threshold %.2f%%", previousBal, currentBal, percent, dropPercent)
		}
	}
	return false, ""
}

func accountExtraFloat64(extra map[string]any, key string) (float64, bool) {
	if extra == nil {
		return 0, false
	}
	switch v := extra[key].(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		parsed, err := v.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func accountExtraBool(extra map[string]any, key string) bool {
	if extra == nil {
		return false
	}
	value, ok := extra[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		return err == nil && parsed
	case float64:
		return v != 0
	case float32:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i != 0
		}
	}
	return false
}

// BalanceAPIResponse accepts the wallet and quota response shapes exposed by
// sub2api-compatible /v1/usage endpoints and the configured legacy API.
type BalanceAPIResponse struct {
	Balance   json.RawMessage `json:"balance"`
	Remaining json.RawMessage `json:"remaining"`
	Quota     struct {
		Remaining json.RawMessage `json:"remaining"`
	} `json:"quota"`
}

// fetchBalance calls the balance check API and returns the current balance
func (s *BalanceCheckService) fetchBalance(ctx context.Context, account *Account) *BalanceCheckResult {
	if account == nil {
		return nil
	}
	result := &BalanceCheckResult{
		AccountID: account.ID,
		Platform:  account.Platform,
		APIKey:    strings.TrimSpace(account.GetCredential("api_key")),
	}
	if result.APIKey == "" {
		result.Error = "missing api_key"
		return result
	}

	switch accountBalanceCheckType(*account) {
	case BalanceCheckTypeSub2API:
		return s.fetchSub2APIBalance(ctx, account, result)
	case BalanceCheckTypeConfiguredAPI:
		return s.fetchBalanceFromURL(ctx, result, s.runtimeConfig().BalanceURL, BalanceCheckTypeConfiguredAPI)
	}

	// Untyped custom upstreams are probed once. A successful detector is stored
	// with the balance so later checks use only the classified API.
	if accountHasSub2APIBalanceCredentials(*account) {
		if detected := s.fetchSub2APIBalance(ctx, account, result); detected.Error == "" {
			return detected
		}
	}
	return s.fetchBalanceFromURL(ctx, result, s.runtimeConfig().BalanceURL, BalanceCheckTypeConfiguredAPI)
}

func (s *BalanceCheckService) fetchSub2APIBalance(ctx context.Context, account *Account, base *BalanceCheckResult) *BalanceCheckResult {
	usageURL, err := buildSub2APIUsageURL(account.GetCredential("base_url"))
	if err != nil {
		result := *base
		result.CheckType = BalanceCheckTypeSub2API
		result.Error = fmt.Sprintf("invalid sub2api base_url: %v", err)
		return &result
	}
	return s.fetchBalanceFromURL(ctx, base, usageURL, BalanceCheckTypeSub2API)
}

func buildSub2APIUsageURL(rawBaseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawBaseURL))
	if err != nil {
		return "", err
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("http(s) URL with host is required")
	}

	cleanPath := strings.TrimRight(parsed.Path, "/")
	switch {
	case strings.HasSuffix(cleanPath, "/v1/usage"):
	case strings.HasSuffix(cleanPath, "/v1"):
		cleanPath += "/usage"
	default:
		cleanPath += "/v1/usage"
	}
	if cleanPath == "" || cleanPath[0] != '/' {
		cleanPath = "/" + cleanPath
	}
	parsed.Path = cleanPath
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func (s *BalanceCheckService) fetchBalanceFromURL(
	ctx context.Context,
	base *BalanceCheckResult,
	balanceURL string,
	checkType string,
) *BalanceCheckResult {
	result := *base
	result.BaseURL = balanceURL
	result.CheckType = checkType

	req, err := http.NewRequestWithContext(ctx, "GET", balanceURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return &result
	}

	req.Header.Set("Authorization", "Bearer "+result.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("HTTP request failed: %v", err)
		return &result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return &result
	}

	var apiResp BalanceAPIResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBalanceCheckResponseBytes)).Decode(&apiResp); err != nil {
		result.Error = fmt.Sprintf("failed to decode response: %v", err)
		return &result
	}

	balance, ok := firstBalanceValue(apiResp.Balance, apiResp.Remaining, apiResp.Quota.Remaining)
	if !ok {
		result.Error = "balance field missing or invalid"
		return &result
	}
	result.CurrentBal = balance
	return &result
}

func firstBalanceValue(values ...json.RawMessage) (float64, bool) {
	for _, raw := range values {
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		var number json.Number
		if err := json.Unmarshal(raw, &number); err == nil {
			if parsed, err := number.Float64(); err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0) {
				return parsed, true
			}
		}
		var text string
		if err := json.Unmarshal(raw, &text); err == nil {
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0) {
				return parsed, true
			}
		}
	}
	return 0, false
}
