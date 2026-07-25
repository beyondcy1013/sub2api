package service

import (
	"context"
	"io"
	"net/http"
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
