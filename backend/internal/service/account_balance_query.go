package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/google/uuid"
)

const (
	AccountBalanceQueryExtraKey = "balance_query"

	accountBalanceQueryRequestTimeout = 6 * time.Second
	accountBalanceQueryMaxBodyBytes   = 64 * 1024
	accountBalanceQueryMaxURLLength   = 2048
)

type AccountBalanceQueryScheme string

const (
	AccountBalanceQuerySchemeAuto    AccountBalanceQueryScheme = "auto"
	AccountBalanceQuerySchemeSub2API AccountBalanceQueryScheme = "sub2api"
	AccountBalanceQuerySchemeNikoAPI AccountBalanceQueryScheme = "nikoapi"
	// AccountBalanceQuerySchemeNewAPI is accepted as a legacy alias for nikoapi.
	AccountBalanceQuerySchemeNewAPI           AccountBalanceQueryScheme = "newapi"
	AccountBalanceQuerySchemeOpenAICompatible AccountBalanceQueryScheme = "openai_compatible"
	AccountBalanceQuerySchemeCPA              AccountBalanceQueryScheme = "cpa"
	AccountBalanceQuerySchemeCustom           AccountBalanceQueryScheme = "custom"
	AccountBalanceQuerySchemeSignIn           AccountBalanceQueryScheme = "signin"
)

var ErrAccountBalanceQueryInvalid = infraerrors.BadRequest(
	"ACCOUNT_BALANCE_QUERY_INVALID",
	"account balance query requires an API key account with an upstream base URL",
)

type AccountBalanceQueryLastResult struct {
	Balance   float64   `json:"balance"`
	Unit      string    `json:"unit"`
	Unlimited bool      `json:"unlimited,omitempty"`
	QueriedAt time.Time `json:"queried_at"`
}

// AccountBalanceQueryConfig is managed server-side in accounts.extra. Scheme
// is automatically replaced after a successful fallback probe.
type AccountBalanceQueryConfig struct {
	Scheme         AccountBalanceQueryScheme      `json:"scheme"`
	APIURL         string                         `json:"api_url,omitempty"`
	SignInSiteID   string                         `json:"sign_in_site_id,omitempty"`
	DetectedAPIURL string                         `json:"detected_api_url,omitempty"`
	LastResult     *AccountBalanceQueryLastResult `json:"last_result,omitempty"`
}

type AccountBalanceQueryAttempt struct {
	Scheme     AccountBalanceQueryScheme `json:"scheme"`
	APIURL     string                    `json:"api_url"`
	HTTPStatus int                       `json:"http_status,omitempty"`
	Error      string                    `json:"error,omitempty"`
}

type AccountBalanceQueryResult struct {
	AccountID    int64                        `json:"account_id"`
	Success      bool                         `json:"success"`
	Scheme       AccountBalanceQueryScheme    `json:"scheme,omitempty"`
	APIURL       string                       `json:"api_url,omitempty"`
	SignInSiteID string                       `json:"sign_in_site_id,omitempty"`
	Balance      float64                      `json:"balance"`
	Unit         string                       `json:"unit,omitempty"`
	Unlimited    bool                         `json:"unlimited,omitempty"`
	QueriedAt    time.Time                    `json:"queried_at"`
	Attempts     []AccountBalanceQueryAttempt `json:"attempts"`
}

type accountBalanceQueryCandidate struct {
	scheme AccountBalanceQueryScheme
	apiURL string
}

type parsedAccountBalance struct {
	Balance      float64
	Unit         string
	Unlimited    bool
	APIURL       string
	SignInSiteID string
}

func (s *UpstreamBillingProbeService) UpdateAccountBalanceQueryConfig(
	ctx context.Context,
	accountID int64,
	input AccountBalanceQueryConfig,
) (*AccountBalanceQueryConfig, error) {
	if s == nil || s.accountRepo == nil || s.accountTestService == nil {
		return nil, ErrUpstreamBillingProbeUnavailable
	}
	account, normalizedBaseURL, err := s.loadBalanceQueryAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	if !isSupportedAccountBalanceQueryScheme(input.Scheme) {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_BALANCE_QUERY_SCHEME", "unsupported balance query scheme")
	}
	input.Scheme = normalizeAccountBalanceQueryScheme(input.Scheme)
	input.APIURL = strings.TrimSpace(input.APIURL)
	input.SignInSiteID = strings.TrimSpace(input.SignInSiteID)
	if input.APIURL != "" {
		if _, err := resolveAccountBalanceQueryURL(normalizedBaseURL, input.APIURL); err != nil {
			return nil, infraerrors.BadRequest("INVALID_ACCOUNT_BALANCE_QUERY_URL", err.Error())
		}
	}
	if input.Scheme == AccountBalanceQuerySchemeCustom && input.APIURL == "" {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_BALANCE_QUERY_URL", "custom balance query requires api_url")
	}
	if input.Scheme == AccountBalanceQuerySchemeAuto && input.APIURL != "" {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_BALANCE_QUERY_URL", "select a scheme before setting api_url")
	}
	if input.Scheme == AccountBalanceQuerySchemeSignIn && input.APIURL != "" {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_BALANCE_QUERY_URL", "signIn balance query does not use api_url")
	}
	if input.SignInSiteID != "" {
		if _, err := uuid.Parse(input.SignInSiteID); err != nil {
			return nil, infraerrors.BadRequest("INVALID_ACCOUNT_BALANCE_QUERY_SIGNIN_SITE", "sign_in_site_id must be a UUID")
		}
	}

	previous := decodeAccountBalanceQueryConfig(account.Extra)
	if sameBalanceQuerySource(previous, input) {
		// 数据源未变化时保留已检测端点与最近查询结果。
		input.DetectedAPIURL = previous.DetectedAPIURL
		input.LastResult = previous.LastResult
	}
	// 数据源变化时不复制旧源的缓存结果，避免界面继续展示旧端点余额。
	if err := s.accountRepo.UpdateExtra(ctx, accountID, map[string]any{AccountBalanceQueryExtraKey: input}); err != nil {
		return nil, err
	}
	return &input, nil
}

func (s *UpstreamBillingProbeService) QueryAccountBalance(ctx context.Context, accountID int64) (*AccountBalanceQueryResult, error) {
	if s == nil || s.accountRepo == nil || s.accountTestService == nil || s.accountTestService.httpUpstream == nil {
		return nil, ErrUpstreamBillingProbeUnavailable
	}
	account, normalizedBaseURL, err := s.loadBalanceQueryAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	proxyURL, err := accountBalanceQueryProxyURL(account)
	if err != nil {
		return nil, err
	}
	config := decodeAccountBalanceQueryConfig(account.Extra)
	candidates, err := accountBalanceQueryCandidates(normalizedBaseURL, config)
	if err != nil {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_BALANCE_QUERY_URL", err.Error())
	}

	result := &AccountBalanceQueryResult{
		AccountID: account.ID,
		QueriedAt: s.currentTime().UTC(),
		Attempts:  make([]AccountBalanceQueryAttempt, 0, len(candidates)),
	}
	for _, candidate := range candidates {
		parsed, attempt := s.queryAccountBalanceCandidate(ctx, account, normalizedBaseURL, proxyURL, candidate)
		result.Attempts = append(result.Attempts, attempt)
		if parsed == nil {
			continue
		}

		result.Success = true
		result.Scheme = candidate.scheme
		result.APIURL = candidate.apiURL
		if parsed.APIURL != "" {
			result.APIURL = parsed.APIURL
		}
		result.Balance = parsed.Balance
		result.Unit = parsed.Unit
		result.Unlimited = parsed.Unlimited
		if parsed.SignInSiteID != "" {
			result.SignInSiteID = parsed.SignInSiteID
		}

		// 查询可能耗时数秒，期间管理员可能已切换数据源。落盘前重新读取当前
		// 配置，只把本次查询结果合并进最新配置，避免旧查询覆盖新数据源。
		freshAccount, freshErr := s.accountRepo.GetByID(ctx, account.ID)
		if freshErr != nil {
			return nil, freshErr
		}
		current := decodeAccountBalanceQueryConfig(freshAccount.Extra)
		if sameBalanceQuerySource(current, config) {
			if current.Scheme != candidate.scheme {
				current.APIURL = ""
			}
			current.Scheme = candidate.scheme
			current.DetectedAPIURL = result.APIURL
			if parsed.SignInSiteID != "" {
				current.SignInSiteID = parsed.SignInSiteID
			}
		}
		current.LastResult = &AccountBalanceQueryLastResult{
			Balance:   parsed.Balance,
			Unit:      parsed.Unit,
			Unlimited: parsed.Unlimited,
			QueriedAt: result.QueriedAt,
		}
		if err := s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{AccountBalanceQueryExtraKey: current}); err != nil {
			return nil, err
		}
		return result, nil
	}
	return result, nil
}

func (s *UpstreamBillingProbeService) loadBalanceQueryAccount(ctx context.Context, accountID int64) (*Account, string, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, "", err
	}
	if account == nil || account.Type != AccountTypeAPIKey || strings.TrimSpace(account.GetCredential("api_key")) == "" {
		return nil, "", ErrAccountBalanceQueryInvalid
	}
	baseURL := strings.TrimSpace(account.GetCredential("base_url"))
	if baseURL == "" {
		return nil, "", ErrAccountBalanceQueryInvalid
	}
	normalizedBaseURL, err := s.accountTestService.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, "", infraerrors.BadRequest("INVALID_ACCOUNT_BALANCE_QUERY_BASE_URL", "invalid account upstream base URL")
	}
	return account, normalizedBaseURL, nil
}

func accountBalanceQueryProxyURL(account *Account) (string, error) {
	if account.ProxyID == nil {
		return "", nil
	}
	if account.Proxy == nil || account.Proxy.ID != *account.ProxyID {
		return "", ErrUpstreamBillingProbeIdentityChanged
	}
	return account.Proxy.URL(), nil
}

func accountBalanceQueryCandidates(baseURL string, config AccountBalanceQueryConfig) ([]accountBalanceQueryCandidate, error) {
	seen := make(map[AccountBalanceQueryScheme]struct{})
	candidates := make([]accountBalanceQueryCandidate, 0, 5)
	add := func(scheme AccountBalanceQueryScheme, configuredURL string) error {
		if scheme == AccountBalanceQuerySchemeAuto {
			return nil
		}
		if _, exists := seen[scheme]; exists {
			return nil
		}
		if scheme == AccountBalanceQuerySchemeSignIn {
			siteID := strings.TrimSpace(config.SignInSiteID)
			endpoint := "signin://auto"
			if siteID != "" {
				endpoint = "signin://" + siteID
			}
			seen[scheme] = struct{}{}
			candidates = append(candidates, accountBalanceQueryCandidate{scheme: scheme, apiURL: endpoint})
			return nil
		}
		endpoint := configuredURL
		if endpoint == "" {
			endpoint = defaultAccountBalanceQueryEndpoint(scheme)
		}
		if endpoint == "" {
			return nil
		}
		resolved, err := resolveAccountBalanceQueryURL(baseURL, endpoint)
		if err != nil {
			return err
		}
		seen[scheme] = struct{}{}
		candidates = append(candidates, accountBalanceQueryCandidate{scheme: scheme, apiURL: resolved})
		return nil
	}

	remembered := normalizeAccountBalanceQueryScheme(config.Scheme)
	if remembered == AccountBalanceQuerySchemeSignIn {
		if err := add(remembered, ""); err != nil {
			return nil, err
		}
	} else if remembered == AccountBalanceQuerySchemeCustom || config.APIURL != "" {
		if err := add(remembered, config.APIURL); err != nil {
			return nil, err
		}
	} else if err := add(remembered, ""); err != nil {
		return nil, err
	}
	for _, scheme := range []AccountBalanceQueryScheme{
		AccountBalanceQuerySchemeSub2API,
		AccountBalanceQuerySchemeNikoAPI,
		AccountBalanceQuerySchemeOpenAICompatible,
		AccountBalanceQuerySchemeCPA,
		AccountBalanceQuerySchemeSignIn,
	} {
		if err := add(scheme, ""); err != nil {
			return nil, err
		}
	}
	return candidates, nil
}

func defaultAccountBalanceQueryEndpoint(scheme AccountBalanceQueryScheme) string {
	switch scheme {
	case AccountBalanceQuerySchemeSub2API:
		return "/v1/usage"
	case AccountBalanceQuerySchemeNikoAPI:
		return "/api/usage/token/"
	case AccountBalanceQuerySchemeOpenAICompatible:
		return "/v1/dashboard/billing/subscription"
	case AccountBalanceQuerySchemeCPA:
		return "/v0/management/api-key-usage"
	default:
		return ""
	}
}

func (s *UpstreamBillingProbeService) queryAccountBalanceCandidate(
	ctx context.Context,
	account *Account,
	normalizedBaseURL string,
	proxyURL string,
	candidate accountBalanceQueryCandidate,
) (*parsedAccountBalance, AccountBalanceQueryAttempt) {
	attempt := AccountBalanceQueryAttempt{Scheme: candidate.scheme, APIURL: candidate.apiURL}
	if candidate.scheme == AccountBalanceQuerySchemeSignIn {
		if s.balanceSignIn == nil {
			attempt.Error = "service_unavailable"
			return nil, attempt
		}
		parsed, siteID, errCode := s.balanceSignIn.Query(
			ctx,
			normalizedBaseURL,
			account.GetCredential("api_key"),
			decodeAccountBalanceQueryConfig(account.Extra).SignInSiteID,
		)
		if errCode != "" {
			attempt.Error = errCode
			return nil, attempt
		}
		attempt.APIURL = "signin://" + siteID
		parsed.APIURL = attempt.APIURL
		parsed.SignInSiteID = siteID
		return parsed, attempt
	}
	body, status, errCode := s.doAccountBalanceQueryRequest(ctx, account, proxyURL, candidate.apiURL)
	attempt.HTTPStatus = status
	if errCode != "" {
		attempt.Error = errCode
		return nil, attempt
	}

	if candidate.scheme == AccountBalanceQuerySchemeOpenAICompatible {
		subscription, err := parseOpenAICompatibleSubscription(body)
		if err != nil {
			attempt.Error = "invalid_response"
			return nil, attempt
		}
		usageURL, err := resolveAccountBalanceQueryURL(normalizedBaseURL, "/v1/dashboard/billing/usage")
		if err != nil {
			attempt.Error = "invalid_usage_url"
			return nil, attempt
		}
		usageBody, usageStatus, usageErrCode := s.doAccountBalanceQueryRequest(ctx, account, proxyURL, usageURL)
		if usageErrCode != "" {
			attempt.HTTPStatus = usageStatus
			attempt.Error = usageErrCode
			return nil, attempt
		}
		usage, err := parseOpenAICompatibleUsage(usageBody)
		if err != nil {
			attempt.Error = "invalid_response"
			return nil, attempt
		}
		return &parsedAccountBalance{Balance: subscription - usage, Unit: "USD"}, attempt
	}

	parsed, err := parseAccountBalanceResponse(candidate.scheme, body)
	if err != nil {
		attempt.Error = "invalid_response"
		return nil, attempt
	}
	if candidate.scheme == AccountBalanceQuerySchemeNikoAPI {
		statusURL, resolveErr := resolveAccountBalanceQueryURL(normalizedBaseURL, "/api/status")
		if resolveErr == nil {
			statusBody, _, statusErrCode := s.doAccountBalanceQueryRequest(ctx, account, proxyURL, statusURL)
			if statusErrCode == "" {
				if converted, convertErr := convertNikoAPIQuotaBalance(parsed, statusBody); convertErr == nil {
					parsed = converted
				}
			}
		}
	}
	return parsed, attempt
}

func (s *UpstreamBillingProbeService) doAccountBalanceQueryRequest(
	ctx context.Context,
	account *Account,
	proxyURL string,
	apiURL string,
) ([]byte, int, string) {
	requestCtx, cancel := context.WithTimeout(ctx, accountBalanceQueryRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, apiURL, bytes.NewReader(nil))
	if err != nil {
		return nil, 0, "request_build_failed"
	}
	reqCtx := WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI)
	req = req.WithContext(WithHTTPUpstreamRedirectsDisabled(reqCtx))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+account.GetCredential("api_key"))
	req.Header.Set("x-api-key", account.GetCredential("api_key"))
	account.ApplyHeaderOverrides(req.Header)
	var tlsProfile *tlsfingerprint.Profile
	if s.accountTestService.tlsFPProfileService != nil {
		tlsProfile = s.accountTestService.tlsFPProfileService.ResolveTLSProfile(account)
	}
	resp, err := s.accountTestService.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil {
		return nil, 0, "request_failed"
	}
	if resp == nil || resp.Body == nil {
		return nil, 0, "empty_response"
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, accountBalanceQueryMaxBodyBytes+1))
	if err != nil {
		return nil, resp.StatusCode, "response_read_failed"
	}
	if len(body) > accountBalanceQueryMaxBodyBytes {
		return nil, resp.StatusCode, "response_too_large"
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, resp.StatusCode, "http_" + strconv.Itoa(resp.StatusCode)
	}
	return body, resp.StatusCode, ""
}

func parseAccountBalanceResponse(scheme AccountBalanceQueryScheme, body []byte) (*parsedAccountBalance, error) {
	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	switch scheme {
	case AccountBalanceQuerySchemeSub2API:
		mode, _ := payload["mode"].(string)
		isValid, _ := payload["isValid"].(bool)
		if !isValid || (mode != "unrestricted" && mode != "quota_limited") {
			return nil, fmt.Errorf("unexpected Sub2API usage response")
		}
		balance, ok := firstAccountBalanceNumber(payload, "balance", "remaining", "quota.remaining")
		if !ok {
			return nil, fmt.Errorf("Sub2API response has no balance")
		}
		return validatedParsedAccountBalance(balance, accountBalanceString(payload, "unit", "quota.unit", "USD"), false)
	case AccountBalanceQuerySchemeNikoAPI:
		code, _ := payload["code"].(bool)
		object, _ := accountBalanceValue(payload, "data.object").(string)
		if !code || object != "token_usage" {
			return nil, fmt.Errorf("unexpected NewAPI token usage response")
		}
		balance, ok := firstAccountBalanceNumber(payload, "data.total_available")
		if !ok {
			return nil, fmt.Errorf("NewAPI response has no available quota")
		}
		unlimited, _ := accountBalanceValue(payload, "data.unlimited_quota").(bool)
		return validatedParsedAccountBalance(balance, "quota", unlimited)
	case AccountBalanceQuerySchemeCPA:
		balance, ok := firstAccountBalanceNumber(payload, "balance", "remaining", "data.balance", "data.remaining")
		if !ok {
			return nil, fmt.Errorf("CPA response has no balance")
		}
		return validatedParsedAccountBalance(balance, accountBalanceString(payload, "unit", "currency", "quota"), false)
	case AccountBalanceQuerySchemeCustom:
		balance, ok := firstAccountBalanceNumber(
			payload,
			"balance", "remaining", "available_balance", "remaining_balance",
			"data.balance", "data.remaining", "data.available_balance", "data.remaining_balance", "data.total_available",
			"quota.remaining",
		)
		if !ok {
			return nil, fmt.Errorf("custom response has no recognized balance field")
		}
		unlimited, _ := accountBalanceValue(payload, "unlimited").(bool)
		if nested, ok := accountBalanceValue(payload, "data.unlimited_quota").(bool); ok {
			unlimited = nested
		}
		return validatedParsedAccountBalance(balance, accountBalanceString(payload, "unit", "currency", "data.unit", "data.currency", "quota"), unlimited)
	default:
		return nil, fmt.Errorf("unsupported balance query scheme")
	}
}

func convertNikoAPIQuotaBalance(balance *parsedAccountBalance, statusBody []byte) (*parsedAccountBalance, error) {
	if balance == nil {
		return nil, fmt.Errorf("nikoapi balance is missing")
	}
	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(statusBody))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	if success, _ := payload["success"].(bool); !success {
		return nil, fmt.Errorf("nikoapi status query failed")
	}
	quotaPerUnit, ok := firstAccountBalanceNumber(payload, "data.quota_per_unit")
	if !ok || quotaPerUnit <= 0 || math.IsNaN(quotaPerUnit) || math.IsInf(quotaPerUnit, 0) {
		return nil, fmt.Errorf("nikoapi status has invalid quota_per_unit")
	}
	displayType, _ := accountBalanceValue(payload, "data.quota_display_type").(string)
	displayType = strings.ToUpper(strings.TrimSpace(displayType))
	converted := balance.Balance
	var unit string
	switch displayType {
	case "USD":
		converted /= quotaPerUnit
		unit = "USD"
	case "CNY":
		exchangeRate, ok := firstAccountBalanceNumber(payload, "data.usd_exchange_rate")
		if !ok || exchangeRate <= 0 || math.IsNaN(exchangeRate) || math.IsInf(exchangeRate, 0) {
			return nil, fmt.Errorf("nikoapi status has invalid usd_exchange_rate")
		}
		converted = converted / quotaPerUnit * exchangeRate
		unit = "CNY"
	case "CUSTOM":
		exchangeRate, ok := firstAccountBalanceNumber(payload, "data.custom_currency_exchange_rate")
		if !ok || exchangeRate <= 0 || math.IsNaN(exchangeRate) || math.IsInf(exchangeRate, 0) {
			return nil, fmt.Errorf("nikoapi status has invalid custom currency exchange rate")
		}
		converted = converted / quotaPerUnit * exchangeRate
		unit = accountBalanceString(payload, "data.custom_currency_symbol", "CUSTOM")
	case "TOKENS":
		unit = "tokens"
	default:
		return nil, fmt.Errorf("nikoapi status has unsupported quota display type")
	}
	return validatedParsedAccountBalance(converted, unit, balance.Unlimited)
}

func parseOpenAICompatibleSubscription(body []byte) (float64, error) {
	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return 0, err
	}
	if object, _ := payload["object"].(string); object != "billing_subscription" {
		return 0, fmt.Errorf("unexpected subscription response")
	}
	value, ok := firstAccountBalanceNumber(payload, "hard_limit_usd")
	if !ok {
		return 0, fmt.Errorf("subscription response has no hard limit")
	}
	return value, nil
}

func parseOpenAICompatibleUsage(body []byte) (float64, error) {
	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return 0, err
	}
	if object, _ := payload["object"].(string); object != "list" {
		return 0, fmt.Errorf("unexpected usage response")
	}
	value, ok := firstAccountBalanceNumber(payload, "total_usage")
	if !ok {
		return 0, fmt.Errorf("usage response has no total usage")
	}
	return value / 100, nil
}

func validatedParsedAccountBalance(balance float64, unit string, unlimited bool) (*parsedAccountBalance, error) {
	if math.IsNaN(balance) || math.IsInf(balance, 0) {
		return nil, fmt.Errorf("invalid balance")
	}
	unit = strings.TrimSpace(unit)
	if unit == "" {
		unit = "quota"
	}
	if len(unit) > 16 {
		return nil, fmt.Errorf("invalid balance unit")
	}
	return &parsedAccountBalance{Balance: balance, Unit: unit, Unlimited: unlimited}, nil
}

func firstAccountBalanceNumber(payload map[string]any, paths ...string) (float64, bool) {
	for _, path := range paths {
		if value, ok := numericAccountBalanceValue(accountBalanceValue(payload, path)); ok {
			return value, true
		}
	}
	return 0, false
}

func numericAccountBalanceValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case float64:
		return typed, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func accountBalanceValue(payload map[string]any, path string) any {
	var current any = payload
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = object[part]
		if !ok {
			return nil
		}
	}
	return current
}

func accountBalanceString(payload map[string]any, pathsAndFallback ...string) string {
	if len(pathsAndFallback) == 0 {
		return ""
	}
	for _, path := range pathsAndFallback[:len(pathsAndFallback)-1] {
		if value, ok := accountBalanceValue(payload, path).(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return pathsAndFallback[len(pathsAndFallback)-1]
}

func normalizeAccountBalanceQueryScheme(scheme AccountBalanceQueryScheme) AccountBalanceQueryScheme {
	switch AccountBalanceQueryScheme(strings.ToLower(strings.TrimSpace(string(scheme)))) {
	case AccountBalanceQuerySchemeSub2API:
		return AccountBalanceQuerySchemeSub2API
	case AccountBalanceQuerySchemeNikoAPI, AccountBalanceQuerySchemeNewAPI:
		return AccountBalanceQuerySchemeNikoAPI
	case AccountBalanceQuerySchemeOpenAICompatible:
		return AccountBalanceQuerySchemeOpenAICompatible
	case AccountBalanceQuerySchemeCPA:
		return AccountBalanceQuerySchemeCPA
	case AccountBalanceQuerySchemeCustom:
		return AccountBalanceQuerySchemeCustom
	case AccountBalanceQuerySchemeSignIn:
		return AccountBalanceQuerySchemeSignIn
	default:
		return AccountBalanceQuerySchemeAuto
	}
}

// sameBalanceQuerySource reports whether two configs point at the same effective
// balance data source (scheme + endpoint). Detected metadata and cached query
// results are not part of the source identity.
func sameBalanceQuerySource(left, right AccountBalanceQueryConfig) bool {
	return normalizeAccountBalanceQueryScheme(left.Scheme) == normalizeAccountBalanceQueryScheme(right.Scheme) &&
		strings.TrimSpace(left.APIURL) == strings.TrimSpace(right.APIURL) &&
		strings.TrimSpace(left.SignInSiteID) == strings.TrimSpace(right.SignInSiteID)
}

func isSupportedAccountBalanceQueryScheme(scheme AccountBalanceQueryScheme) bool {
	normalized := AccountBalanceQueryScheme(strings.ToLower(strings.TrimSpace(string(scheme))))
	return normalized == AccountBalanceQuerySchemeAuto ||
		normalized == AccountBalanceQuerySchemeSub2API ||
		normalized == AccountBalanceQuerySchemeNikoAPI ||
		normalized == AccountBalanceQuerySchemeNewAPI ||
		normalized == AccountBalanceQuerySchemeOpenAICompatible ||
		normalized == AccountBalanceQuerySchemeCPA ||
		normalized == AccountBalanceQuerySchemeCustom ||
		normalized == AccountBalanceQuerySchemeSignIn
}

func decodeAccountBalanceQueryConfig(extra map[string]any) AccountBalanceQueryConfig {
	config := AccountBalanceQueryConfig{Scheme: AccountBalanceQuerySchemeAuto}
	if extra == nil {
		return config
	}
	value, ok := extra[AccountBalanceQueryExtraKey]
	if !ok || value == nil {
		return config
	}
	raw, err := json.Marshal(value)
	if err != nil || json.Unmarshal(raw, &config) != nil {
		return AccountBalanceQueryConfig{Scheme: AccountBalanceQuerySchemeAuto}
	}
	config.Scheme = normalizeAccountBalanceQueryScheme(config.Scheme)
	config.APIURL = strings.TrimSpace(config.APIURL)
	config.SignInSiteID = strings.TrimSpace(config.SignInSiteID)
	return config
}

func resolveAccountBalanceQueryURL(baseURL, endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("balance query API URL is required")
	}
	if len(endpoint) > accountBalanceQueryMaxURLLength {
		return "", fmt.Errorf("balance query API URL is too long")
	}
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("invalid account upstream base URL")
	}
	if base.User != nil {
		return "", fmt.Errorf("account upstream base URL cannot contain user info")
	}

	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil || parsedEndpoint.Fragment != "" || parsedEndpoint.User != nil {
		return "", fmt.Errorf("invalid balance query API URL")
	}
	if parsedEndpoint.IsAbs() {
		if !strings.EqualFold(parsedEndpoint.Scheme, base.Scheme) || !strings.EqualFold(parsedEndpoint.Host, base.Host) {
			return "", fmt.Errorf("balance query API URL must use the same origin as the account base URL")
		}
		return parsedEndpoint.String(), nil
	}
	if parsedEndpoint.Host != "" {
		return "", fmt.Errorf("balance query API URL must use the same origin as the account base URL")
	}

	rootPath := strings.TrimRight(base.Path, "/")
	if slash := strings.LastIndex(rootPath, "/"); slash >= 0 && openAIBaseURLHasVersionSuffix(rootPath) {
		rootPath = rootPath[:slash]
	}
	base.Path = strings.TrimRight(rootPath, "/") + "/" + strings.TrimLeft(parsedEndpoint.Path, "/")
	base.RawPath = ""
	base.RawQuery = parsedEndpoint.RawQuery
	base.Fragment = ""
	return base.String(), nil
}
