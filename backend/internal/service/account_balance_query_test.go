package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type accountBalanceQueryHTTPStub struct {
	requests []*http.Request
	handler  func(*http.Request) *http.Response
}

func (s *accountBalanceQueryHTTPStub) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	s.requests = append(s.requests, req)
	return s.handler(req), nil
}

func (s *accountBalanceQueryHTTPStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, concurrency)
}

func balanceQueryResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func balanceQueryAccount(extra map[string]any) *Account {
	return &Account{
		ID:          901,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Concurrency: 3,
		Credentials: map[string]any{
			"api_key":  "sk-balance-query",
			"base_url": "https://relay.example/v1",
		},
		Extra: extra,
	}
}

func TestAccountBalanceQueryFallsBackAndPersistsDetectedNewAPIScheme(t *testing.T) {
	account := balanceQueryAccount(map[string]any{
		AccountBalanceQueryExtraKey: map[string]any{"scheme": AccountBalanceQuerySchemeSub2API},
	})
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{account.ID: account}}
	upstream := &accountBalanceQueryHTTPStub{handler: func(req *http.Request) *http.Response {
		switch req.URL.Path {
		case "/v1/usage":
			return balanceQueryResponse(http.StatusNotFound, `{"error":"not found"}`)
		case "/api/usage/token/":
			return balanceQueryResponse(http.StatusOK, `{
				"code":true,
				"data":{"object":"token_usage","total_available":6250000,"unlimited_quota":false}
			}`)
		default:
			return balanceQueryResponse(http.StatusNotFound, `{"error":"unexpected endpoint"}`)
		}
	}}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})

	result, err := svc.QueryAccountBalance(context.Background(), account.ID)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, AccountBalanceQuerySchemeNewAPI, result.Scheme)
	require.Equal(t, 6250000.0, result.Balance)
	require.Equal(t, "quota", result.Unit)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "/v1/usage", upstream.requests[0].URL.Path)
	require.Equal(t, "/api/usage/token/", upstream.requests[1].URL.Path)
	require.Equal(t, "Bearer sk-balance-query", upstream.requests[1].Header.Get("Authorization"))

	config := decodeAccountBalanceQueryConfig(account.Extra)
	require.Equal(t, AccountBalanceQuerySchemeNewAPI, config.Scheme)
	require.Equal(t, result.APIURL, config.DetectedAPIURL)
	require.NotNil(t, config.LastResult)
	require.Equal(t, result.Balance, config.LastResult.Balance)
}

func TestAccountBalanceQueryFallsBackToSignInAndPersistsMatchedSite(t *testing.T) {
	const siteID = "32b00162-427c-41d9-8325-faa4dcc0f3a3"
	signIn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/sites":
			require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{{
				"id":                 siteID,
				"name":               "relay browser",
				"login_url":          "https://relay.example/login",
				"usernames":          []string{"browser-user"},
				"target_api_keys":    []string{"sk-balance-query"},
				"last_balance_after": "$19.75",
			}}))
		case r.Method == http.MethodPost && r.URL.Path == "/api/sites/"+siteID+"/refresh-balance":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id": siteID + "-job", "status": "running", "kind": "balance_refresh",
			}))
		case r.Method == http.MethodGet && r.URL.Path == "/api/jobs":
			require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{{
				"id": siteID + "-job", "status": "succeeded", "kind": "balance_refresh",
			}}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer signIn.Close()
	t.Setenv("SIGNIN_BALANCE_SERVICE_URL", signIn.URL)

	account := balanceQueryAccount(nil)
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{account.ID: account}}
	upstream := &accountBalanceQueryHTTPStub{handler: func(*http.Request) *http.Response {
		return balanceQueryResponse(http.StatusNotFound, `{"error":"not found"}`)
	}}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})

	result, err := svc.QueryAccountBalance(context.Background(), account.ID)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, AccountBalanceQueryScheme("signin"), result.Scheme)
	require.Equal(t, 19.75, result.Balance)
	require.Equal(t, "USD", result.Unit)
	require.Equal(t, "signin://"+siteID, result.APIURL)
	require.Equal(t, siteID, decodeAccountBalanceQueryConfig(account.Extra).SignInSiteID)
}

func TestAccountBalanceQueryUsesRememberedSchemeFirst(t *testing.T) {
	account := balanceQueryAccount(map[string]any{
		AccountBalanceQueryExtraKey: map[string]any{"scheme": AccountBalanceQuerySchemeNewAPI},
	})
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{account.ID: account}}
	upstream := &accountBalanceQueryHTTPStub{handler: func(req *http.Request) *http.Response {
		require.Equal(t, "/api/usage/token/", req.URL.Path)
		return balanceQueryResponse(http.StatusOK, `{
			"code":true,
			"data":{"object":"token_usage","total_available":42,"unlimited_quota":false}
		}`)
	}}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})

	result, err := svc.QueryAccountBalance(context.Background(), account.ID)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, AccountBalanceQuerySchemeNewAPI, result.Scheme)
	require.Len(t, upstream.requests, 1)
}

func TestAccountBalanceQueryCustomAPIIsSavedAndParsed(t *testing.T) {
	account := balanceQueryAccount(nil)
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{account.ID: account}}
	upstream := &accountBalanceQueryHTTPStub{handler: func(req *http.Request) *http.Response {
		require.Equal(t, "/custom/account-balance", req.URL.Path)
		return balanceQueryResponse(http.StatusOK, `{"balance":12.34,"currency":"CNY"}`)
	}}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})

	config, err := svc.UpdateAccountBalanceQueryConfig(context.Background(), account.ID, AccountBalanceQueryConfig{
		Scheme: AccountBalanceQuerySchemeCustom,
		APIURL: "/custom/account-balance",
	})
	require.NoError(t, err)
	require.Equal(t, AccountBalanceQuerySchemeCustom, config.Scheme)

	result, err := svc.QueryAccountBalance(context.Background(), account.ID)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, AccountBalanceQuerySchemeCustom, result.Scheme)
	require.Equal(t, 12.34, result.Balance)
	require.Equal(t, "CNY", result.Unit)
	require.Equal(t, "https://relay.example/custom/account-balance", result.APIURL)
}

func TestAccountBalanceQueryRejectsCrossOriginCustomAPI(t *testing.T) {
	account := balanceQueryAccount(nil)
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{account.ID: account}}
	svc := newUpstreamBillingProbeTestService(repo, &accountBalanceQueryHTTPStub{}, &upstreamBillingProbeSettingRepo{})

	_, err := svc.UpdateAccountBalanceQueryConfig(context.Background(), account.ID, AccountBalanceQueryConfig{
		Scheme: AccountBalanceQuerySchemeCustom,
		APIURL: "https://different.example/balance",
	})

	require.ErrorContains(t, err, "same origin")
}

func TestAccountBalanceQueryRejectsAPIURLWithAutoScheme(t *testing.T) {
	account := balanceQueryAccount(nil)
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{account.ID: account}}
	svc := newUpstreamBillingProbeTestService(repo, &accountBalanceQueryHTTPStub{}, &upstreamBillingProbeSettingRepo{})

	_, err := svc.UpdateAccountBalanceQueryConfig(context.Background(), account.ID, AccountBalanceQueryConfig{
		Scheme: AccountBalanceQuerySchemeAuto,
		APIURL: "/custom/account-balance",
	})

	require.ErrorContains(t, err, "select a scheme")
}

func TestAccountBalanceQueryRejectsUnknownScheme(t *testing.T) {
	account := balanceQueryAccount(nil)
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{account.ID: account}}
	svc := newUpstreamBillingProbeTestService(repo, &accountBalanceQueryHTTPStub{}, &upstreamBillingProbeSettingRepo{})

	_, err := svc.UpdateAccountBalanceQueryConfig(context.Background(), account.ID, AccountBalanceQueryConfig{
		Scheme: AccountBalanceQueryScheme("unknown"),
	})

	require.ErrorContains(t, err, "unsupported balance query scheme")
}

func TestAccountBalanceQueryFallbackClearsSchemeSpecificAPIURL(t *testing.T) {
	account := balanceQueryAccount(map[string]any{
		AccountBalanceQueryExtraKey: map[string]any{
			"scheme":  AccountBalanceQuerySchemeSub2API,
			"api_url": "/custom/sub2api-balance",
		},
	})
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{account.ID: account}}
	upstream := &accountBalanceQueryHTTPStub{handler: func(req *http.Request) *http.Response {
		if req.URL.Path == "/api/usage/token/" {
			return balanceQueryResponse(http.StatusOK, `{"code":true,"data":{"object":"token_usage","total_available":42}}`)
		}
		return balanceQueryResponse(http.StatusNotFound, `{"error":"not found"}`)
	}}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})

	result, err := svc.QueryAccountBalance(context.Background(), account.ID)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, AccountBalanceQuerySchemeNewAPI, result.Scheme)
	config := decodeAccountBalanceQueryConfig(account.Extra)
	require.Equal(t, AccountBalanceQuerySchemeNewAPI, config.Scheme)
	require.Empty(t, config.APIURL)
}

func TestAccountBalanceQueryOpenAICompatibleUsesNormalizedBaseURLForUsage(t *testing.T) {
	account := balanceQueryAccount(map[string]any{
		AccountBalanceQueryExtraKey: map[string]any{"scheme": AccountBalanceQuerySchemeOpenAICompatible},
	})
	account.Credentials["base_url"] = " https://relay.example/v1 "
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{account.ID: account}}
	upstream := &accountBalanceQueryHTTPStub{handler: func(req *http.Request) *http.Response {
		switch req.URL.Path {
		case "/v1/dashboard/billing/subscription":
			return balanceQueryResponse(http.StatusOK, `{"object":"billing_subscription","hard_limit_usd":20}`)
		case "/v1/dashboard/billing/usage":
			return balanceQueryResponse(http.StatusOK, `{"object":"list","total_usage":500}`)
		default:
			return balanceQueryResponse(http.StatusNotFound, `{"error":"unexpected endpoint"}`)
		}
	}}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})

	result, err := svc.QueryAccountBalance(context.Background(), account.ID)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, 15.0, result.Balance)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "relay.example", upstream.requests[1].URL.Host)
	require.Equal(t, "/v1/dashboard/billing/usage", upstream.requests[1].URL.Path)
}

func TestParseSub2APIAccountBalanceRequiresKnownResponseShape(t *testing.T) {
	result, err := parseAccountBalanceResponse(AccountBalanceQuerySchemeSub2API, []byte(`{
		"mode":"unrestricted",
		"isValid":true,
		"remaining":19.75,
		"balance":19.75,
		"unit":"USD"
	}`))

	require.NoError(t, err)
	require.Equal(t, 19.75, result.Balance)
	require.Equal(t, "USD", result.Unit)

	_, err = parseAccountBalanceResponse(AccountBalanceQuerySchemeSub2API, []byte(`{"balance":19.75}`))
	require.Error(t, err)
}
