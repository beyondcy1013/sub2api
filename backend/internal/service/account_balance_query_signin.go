package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	accountBalanceSignInDefaultURL   = "http://127.0.0.1:18712"
	accountBalanceSignInQueryTimeout = 65 * time.Second
	accountBalanceSignInPollInterval = 250 * time.Millisecond
	accountBalanceSignInMaxBodyBytes = 1024 * 1024
)

type accountBalanceSignInClient struct {
	baseURL      *url.URL
	httpClient   *http.Client
	pollInterval time.Duration
	configError  string
}

type accountBalanceSignInSite struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	LoginURL         string   `json:"login_url"`
	SignInURL        string   `json:"sign_in_url"`
	Usernames        []string `json:"usernames"`
	TargetAPIKeys    []string `json:"target_api_keys"`
	LastBalanceAfter *string  `json:"last_balance_after"`
}

type accountBalanceSignInJob struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

func newAccountBalanceSignInClientFromEnv() *accountBalanceSignInClient {
	rawURL := strings.TrimSpace(os.Getenv("SIGNIN_BALANCE_SERVICE_URL"))
	if rawURL == "" {
		rawURL = accountBalanceSignInDefaultURL
	}
	baseURL, err := parseAccountBalanceSignInServiceURL(rawURL)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// This integration is loopback-only and must not follow environment proxy settings.
	transport.Proxy = nil
	client := &accountBalanceSignInClient{
		baseURL:      baseURL,
		pollInterval: accountBalanceSignInPollInterval,
		httpClient: &http.Client{
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	if err != nil {
		client.configError = "service_config_invalid"
	}
	return client
}

func parseAccountBalanceSignInServiceURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("signIn balance service URL must be a loopback HTTP URL")
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return nil, fmt.Errorf("signIn balance service URL must use a loopback host")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	return parsed, nil
}

func (c *accountBalanceSignInClient) Query(
	ctx context.Context,
	accountBaseURL string,
	apiKey string,
	rememberedSiteID string,
) (*parsedAccountBalance, string, string) {
	if c == nil || c.baseURL == nil || c.httpClient == nil {
		return nil, "", "service_unavailable"
	}
	if c.configError != "" {
		return nil, "", c.configError
	}
	queryCtx, cancel := context.WithTimeout(ctx, accountBalanceSignInQueryTimeout)
	defer cancel()

	sites, errCode := c.listSites(queryCtx)
	if errCode != "" {
		return nil, "", errCode
	}
	site, errCode := selectAccountBalanceSignInSite(sites, accountBaseURL, apiKey, rememberedSiteID)
	if errCode != "" {
		return nil, "", errCode
	}

	var job accountBalanceSignInJob
	if errCode := c.doJSON(
		queryCtx,
		http.MethodPost,
		"/api/sites/"+url.PathEscape(site.ID)+"/refresh-balance",
		&job,
	); errCode != "" {
		return nil, "", errCode
	}
	if strings.TrimSpace(job.ID) == "" {
		return nil, "", "invalid_response"
	}
	if errCode := c.waitForJob(queryCtx, job); errCode != "" {
		return nil, "", errCode
	}

	sites, errCode = c.listSites(queryCtx)
	if errCode != "" {
		return nil, "", errCode
	}
	for _, refreshed := range sites {
		if refreshed.ID != site.ID || refreshed.LastBalanceAfter == nil {
			continue
		}
		balance, unit, ok := parseAccountBalanceSignInAmount(*refreshed.LastBalanceAfter)
		if !ok {
			return nil, "", "balance_missing"
		}
		return &parsedAccountBalance{Balance: balance, Unit: unit}, site.ID, ""
	}
	return nil, "", "balance_missing"
}

func (c *accountBalanceSignInClient) listSites(ctx context.Context) ([]accountBalanceSignInSite, string) {
	var sites []accountBalanceSignInSite
	if errCode := c.doJSON(ctx, http.MethodGet, "/api/sites", &sites); errCode != "" {
		return nil, errCode
	}
	return sites, ""
}

func (c *accountBalanceSignInClient) waitForJob(ctx context.Context, initial accountBalanceSignInJob) string {
	status := strings.ToLower(strings.TrimSpace(initial.Status))
	if status == "succeeded" {
		return ""
	}
	if status == "failed" {
		return "job_failed"
	}
	interval := c.pollInterval
	if interval <= 0 {
		interval = accountBalanceSignInPollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return "job_timeout"
		case <-ticker.C:
			var jobs []accountBalanceSignInJob
			if errCode := c.doJSON(ctx, http.MethodGet, "/api/jobs", &jobs); errCode != "" {
				return errCode
			}
			for _, job := range jobs {
				if job.ID != initial.ID {
					continue
				}
				switch strings.ToLower(strings.TrimSpace(job.Status)) {
				case "succeeded":
					return ""
				case "failed":
					return "job_failed"
				}
			}
		}
	}
}

func (c *accountBalanceSignInClient) doJSON(ctx context.Context, method, path string, output any) string {
	requestURL := *c.baseURL
	requestURL.Path = strings.TrimRight(c.baseURL.Path, "/") + "/" + strings.TrimLeft(path, "/")
	requestURL.RawPath = ""
	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, method, requestURL.String(), nil)
	if err != nil {
		return "request_build_failed"
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "job_timeout"
		}
		return "service_unavailable"
	}
	if resp == nil || resp.Body == nil {
		return "empty_response"
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, accountBalanceSignInMaxBodyBytes+1))
	if err != nil {
		return "response_read_failed"
	}
	if len(body) > accountBalanceSignInMaxBodyBytes {
		return "response_too_large"
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "http_" + strconv.Itoa(resp.StatusCode)
	}
	if err := json.Unmarshal(body, output); err != nil {
		return "invalid_response"
	}
	return ""
}

func selectAccountBalanceSignInSite(
	sites []accountBalanceSignInSite,
	accountBaseURL string,
	apiKey string,
	rememberedSiteID string,
) (*accountBalanceSignInSite, string) {
	rememberedSiteID = strings.TrimSpace(rememberedSiteID)
	if rememberedSiteID != "" {
		for index := range sites {
			if sites[index].ID == rememberedSiteID {
				if !accountBalanceSignInSiteIsSingleAccount(sites[index]) {
					return nil, "multi_account_site"
				}
				return &sites[index], ""
			}
		}
	}

	apiKeyMatches := make([]int, 0, 1)
	for index, site := range sites {
		if !accountBalanceSignInSiteIsSingleAccount(site) {
			continue
		}
		for _, candidate := range site.TargetAPIKeys {
			if constantTimeAccountBalanceStringEqual(candidate, apiKey) {
				apiKeyMatches = append(apiKeyMatches, index)
				break
			}
		}
	}
	if len(apiKeyMatches) == 1 {
		return &sites[apiKeyMatches[0]], ""
	}
	if len(apiKeyMatches) > 1 {
		return nil, "site_ambiguous"
	}

	baseOrigin := accountBalanceURLOrigin(accountBaseURL)
	originMatches := make([]int, 0, 1)
	if baseOrigin != "" {
		for index, site := range sites {
			if !accountBalanceSignInSiteIsSingleAccount(site) {
				continue
			}
			if accountBalanceURLOrigin(site.LoginURL) == baseOrigin || accountBalanceURLOrigin(site.SignInURL) == baseOrigin {
				originMatches = append(originMatches, index)
			}
		}
	}
	if len(originMatches) == 1 {
		return &sites[originMatches[0]], ""
	}
	if len(originMatches) > 1 {
		return nil, "site_ambiguous"
	}
	return nil, "site_not_found"
}

func accountBalanceSignInSiteIsSingleAccount(site accountBalanceSignInSite) bool {
	return len(site.Usernames) == 1 && len(site.TargetAPIKeys) <= 1
}

func constantTimeAccountBalanceStringEqual(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" || len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func accountBalanceURLOrigin(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host)
}

func parseAccountBalanceSignInAmount(raw string) (float64, string, bool) {
	value := strings.TrimSpace(raw)
	unit := "quota"
	switch {
	case strings.HasPrefix(value, "$"):
		unit = "USD"
		value = strings.TrimSpace(strings.TrimPrefix(value, "$"))
	case strings.HasPrefix(value, "￥"):
		unit = "CNY"
		value = strings.TrimSpace(strings.TrimPrefix(value, "￥"))
	}
	value = strings.ReplaceAll(value, ",", "")
	if value == "" {
		return 0, "", false
	}
	for index, char := range value {
		if char >= '0' && char <= '9' || char == '.' || char == '-' && index == 0 {
			continue
		}
		return 0, "", false
	}
	balance, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(balance) || math.IsInf(balance, 0) {
		return 0, "", false
	}
	return balance, unit, true
}
