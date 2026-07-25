package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type accountBalanceQueryHandlerRepo struct {
	service.AccountRepository
	account *service.Account
}

func (r *accountBalanceQueryHandlerRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	if r.account == nil || r.account.ID != id {
		return nil, service.ErrAccountNotFound
	}
	return r.account, nil
}

func (r *accountBalanceQueryHandlerRepo) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	if r.account == nil || r.account.ID != id {
		return service.ErrAccountNotFound
	}
	if r.account.Extra == nil {
		r.account.Extra = make(map[string]any)
	}
	for key, value := range updates {
		r.account.Extra[key] = value
	}
	return nil
}

type accountBalanceQueryHandlerUpstream struct{}

func (accountBalanceQueryHandlerUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return accountBalanceQueryHandlerResponse(req), nil
}

func (accountBalanceQueryHandlerUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return accountBalanceQueryHandlerResponse(req), nil
}

func accountBalanceQueryHandlerResponse(req *http.Request) *http.Response {
	body := `{"mode":"quota_limited","isValid":true,"balance":0,"unit":"USD"}`
	if req.URL.Path != "/v1/usage" {
		body = `{"error":"unexpected endpoint"}`
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(body))}
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
}

func setupAccountBalanceQueryRouter() (*gin.Engine, *accountBalanceQueryHandlerRepo) {
	gin.SetMode(gin.TestMode)
	repo := &accountBalanceQueryHandlerRepo{account: &service.Account{
		ID:          77,
		Name:        "balance relay",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Concurrency: 2,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://relay.example/v1"},
	}}
	cfg := &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}}
	accountTest := service.NewAccountTestService(repo, nil, nil, nil, nil, accountBalanceQueryHandlerUpstream{}, cfg, nil)
	probe := service.NewUpstreamBillingProbeService(repo, accountTest, nil)
	handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.SetUpstreamBillingProbeService(probe)
	router := gin.New()
	router.PUT("/admin/accounts/:id/balance-query", handler.UpdateAccountBalanceQueryConfig)
	router.POST("/admin/accounts/:id/balance-query", handler.QueryAccountBalance)
	return router, repo
}

func TestAccountBalanceQueryRoutesSaveConfigAndReturnZeroBalance(t *testing.T) {
	router, repo := setupAccountBalanceQueryRouter()

	updateRecorder := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(
		http.MethodPut,
		"/admin/accounts/77/balance-query",
		bytes.NewBufferString(`{"scheme":"sub2api","api_url":"/v1/usage"}`),
	)
	updateRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(updateRecorder, updateRequest)
	require.Equal(t, http.StatusOK, updateRecorder.Code)
	require.Contains(t, repo.account.Extra, service.AccountBalanceQueryExtraKey)

	queryRecorder := httptest.NewRecorder()
	router.ServeHTTP(queryRecorder, httptest.NewRequest(http.MethodPost, "/admin/accounts/77/balance-query", nil))
	require.Equal(t, http.StatusOK, queryRecorder.Code)
	var response struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(queryRecorder.Body.Bytes(), &response))
	require.Equal(t, true, response.Data["success"])
	require.Contains(t, response.Data, "balance")
	require.Equal(t, float64(0), response.Data["balance"])
	require.Equal(t, "sub2api", response.Data["scheme"])
}

func TestAccountBalanceQueryRoutePersistsSignInSiteID(t *testing.T) {
	router, repo := setupAccountBalanceQueryRouter()
	const siteID = "32b00162-427c-41d9-8325-faa4dcc0f3a3"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/admin/accounts/77/balance-query",
		bytes.NewBufferString(`{"scheme":"signin","sign_in_site_id":"`+siteID+`"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	stored, ok := repo.account.Extra[service.AccountBalanceQueryExtraKey].(service.AccountBalanceQueryConfig)
	require.True(t, ok)
	require.Equal(t, service.AccountBalanceQuerySchemeSignIn, stored.Scheme)
	require.Equal(t, siteID, stored.SignInSiteID)
}
