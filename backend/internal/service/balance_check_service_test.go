//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type balanceCheckRepoStub struct {
	accountRepoStub
	schedulableAccounts []Account
	tempAccounts        []Account
	updateExtraCalls    []balanceCheckUpdateExtraCall
	pauseCalls          []balanceCheckPauseCall
	clearCalls          []int64
}

type balanceCheckUpdateExtraCall struct {
	id      int64
	updates map[string]any
}

type balanceCheckPauseCall struct {
	id     int64
	until  time.Time
	reason string
}

func balanceCheckBoolPtr(v bool) *bool {
	return &v
}

func (r *balanceCheckRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	return r.schedulableAccounts, nil
}

func (r *balanceCheckRepoStub) ListWithFilters(_ context.Context, _ pagination.PaginationParams, _, _, status, _ string, _ int64, _ string, _ bool) ([]Account, *pagination.PaginationResult, error) {
	if status == "temp_unschedulable" {
		return r.tempAccounts, &pagination.PaginationResult{Total: int64(len(r.tempAccounts)), Page: 1, PageSize: len(r.tempAccounts)}, nil
	}
	return nil, nil, fmt.Errorf("unexpected status %q", status)
}

func (r *balanceCheckRepoStub) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	r.pauseCalls = append(r.pauseCalls, balanceCheckPauseCall{id: id, until: until, reason: reason})
	return nil
}

func (r *balanceCheckRepoStub) ClearTempUnschedulable(_ context.Context, id int64) error {
	r.clearCalls = append(r.clearCalls, id)
	return nil
}

func (r *balanceCheckRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	r.updateExtraCalls = append(r.updateExtraCalls, balanceCheckUpdateExtraCall{id: id, updates: updates})
	return nil
}

type balanceRoundTripper func(*http.Request) (*http.Response, error)

func (f balanceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func balanceHTTPClient(balanceByToken map[string]float64) *http.Client {
	return &http.Client{Transport: balanceRoundTripper(func(req *http.Request) (*http.Response, error) {
		token := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
		balance, ok := balanceByToken[token]
		if !ok {
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
		}
		body := fmt.Sprintf(`{"balance":%.4f}`, balance)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
}

func TestBalanceCheckService_CheckAccountBalance_AllowsFiveYuanDecrease(t *testing.T) {
	repo := &balanceCheckRepoStub{}
	svc := NewBalanceCheckService(repo, nil)
	svc.httpClient = balanceHTTPClient(map[string]float64{"k": 95})
	svc.balanceCache[101] = 100

	account := &Account{
		ID:          101,
		Name:        "api-key",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "k"},
	}

	svc.checkAccountBalance(context.Background(), account)

	require.Empty(t, repo.pauseCalls)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, 95.0, repo.updateExtraCalls[0].updates["balance"])
	require.Equal(t, 95.0, svc.balanceCache[101])
}

func TestBalanceCheckService_CheckAccountBalance_PausesWhenDecreaseExceedsFiveYuan(t *testing.T) {
	repo := &balanceCheckRepoStub{}
	svc := NewBalanceCheckService(repo, nil)
	svc.httpClient = balanceHTTPClient(map[string]float64{"k": 94.99})
	svc.balanceCache[102] = 100

	account := &Account{
		ID:          102,
		Name:        "api-key",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "k"},
	}

	svc.checkAccountBalance(context.Background(), account)

	require.Len(t, repo.pauseCalls, 1)
	require.Equal(t, int64(102), repo.pauseCalls[0].id)
	require.Contains(t, repo.pauseCalls[0].reason, "decreased by 5.0100")
	require.Equal(t, 94.99, repo.updateExtraCalls[0].updates["balance"])
}

func TestBalanceCheckService_CheckAccountBalance_UsesPersistedBalanceWhenCacheIsEmpty(t *testing.T) {
	repo := &balanceCheckRepoStub{}
	svc := NewBalanceCheckService(repo, nil)
	svc.httpClient = balanceHTTPClient(map[string]float64{"k": 94})

	account := &Account{
		ID:          103,
		Name:        "api-key",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "k"},
		Extra:       map[string]any{"balance": 100.0},
	}

	svc.checkAccountBalance(context.Background(), account)

	require.Len(t, repo.pauseCalls, 1)
	require.Equal(t, int64(103), repo.pauseCalls[0].id)
}

func TestBalanceCheckService_CheckAccountBalance_UsesConfiguredDecreaseThreshold(t *testing.T) {
	repo := &balanceCheckRepoStub{}
	svc := NewBalanceCheckService(repo, &config.Config{
		BalanceCheck: config.BalanceCheckConfig{MinDecrease: 10, PauseDurationHours: 5},
	})
	svc.httpClient = balanceHTTPClient(map[string]float64{"k": 94.99})
	svc.balanceCache[104] = 100

	account := &Account{
		ID:          104,
		Name:        "api-key",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "k"},
	}

	svc.checkAccountBalance(context.Background(), account)

	require.Empty(t, repo.pauseCalls)
	require.Equal(t, 94.99, repo.updateExtraCalls[0].updates["balance"])
}

func TestBalanceCheckService_CheckAccountBalance_PausesWhenCurrentBalanceBelowConfiguredThreshold(t *testing.T) {
	repo := &balanceCheckRepoStub{}
	svc := NewBalanceCheckService(repo, &config.Config{
		BalanceCheck: config.BalanceCheckConfig{MinDecrease: 100, PauseWhenCurrentBelow: 50, PauseDurationHours: 5},
	})
	svc.httpClient = balanceHTTPClient(map[string]float64{"k": 49.5})
	svc.balanceCache[105] = 100

	account := &Account{
		ID:          105,
		Name:        "api-key",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "k"},
	}

	svc.checkAccountBalance(context.Background(), account)

	require.Len(t, repo.pauseCalls, 1)
	require.Equal(t, int64(105), repo.pauseCalls[0].id)
	require.Contains(t, repo.pauseCalls[0].reason, "Balance 49.5000 below threshold 50.0000")
}

func TestBalanceCheckService_CheckAccountBalance_StopsWhenCurrentBalanceBelowConfiguredThreshold(t *testing.T) {
	repo := &balanceCheckRepoStub{}
	svc := NewBalanceCheckService(repo, &config.Config{
		BalanceCheck: config.BalanceCheckConfig{MinDecrease: 100, StopWhenCurrentBelow: 10},
	})
	svc.httpClient = balanceHTTPClient(map[string]float64{"k": 9.5})
	svc.balanceCache[107] = 100

	account := &Account{
		ID:          107,
		Name:        "api-key",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "k"},
	}

	svc.checkAccountBalance(context.Background(), account)

	require.Len(t, repo.pauseCalls, 1)
	require.Equal(t, int64(107), repo.pauseCalls[0].id)
	require.Contains(t, repo.pauseCalls[0].reason, balanceCheckStopReasonMarker)
	require.True(t, time.Until(repo.pauseCalls[0].until) > 24*time.Hour*365*50)
	require.Equal(t, 9.5, repo.updateExtraCalls[0].updates["balance"])
}

func TestBalanceCheckService_CheckAccountBalance_ResumesStoppedAccountWhenBalanceRecovers(t *testing.T) {
	repo := &balanceCheckRepoStub{}
	svc := NewBalanceCheckService(repo, &config.Config{
		BalanceCheck: config.BalanceCheckConfig{ResumeWhenCurrentAbove: 20},
	})
	svc.httpClient = balanceHTTPClient(map[string]float64{"k": 21})

	account := &Account{
		ID:                      108,
		Name:                    "api-key",
		Platform:                PlatformOpenAI,
		Type:                    AccountTypeAPIKey,
		Credentials:             map[string]any{"api_key": "k"},
		TempUnschedulableReason: "Balance 9.5000 below stop threshold 10.0000 (" + balanceCheckStopReasonMarker + ")",
	}

	svc.checkAccountBalance(context.Background(), account)

	require.Equal(t, []int64{108}, repo.clearCalls)
	require.Empty(t, repo.pauseCalls)
	require.Equal(t, 21.0, repo.updateExtraCalls[0].updates["balance"])
}

func TestBalanceCheckService_CheckAccountBalance_AccountExtraOverridesPauseThreshold(t *testing.T) {
	repo := &balanceCheckRepoStub{}
	svc := NewBalanceCheckService(repo, &config.Config{
		BalanceCheck: config.BalanceCheckConfig{MinDecrease: 10, PauseDurationHours: 5},
	})
	svc.httpClient = balanceHTTPClient(map[string]float64{"k": 97})
	svc.balanceCache[106] = 100

	account := &Account{
		ID:          106,
		Name:        "api-key",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "k"},
		Extra:       map[string]any{"balance_pause_min_decrease": 2.0},
	}

	svc.checkAccountBalance(context.Background(), account)

	require.Len(t, repo.pauseCalls, 1)
	require.Equal(t, int64(106), repo.pauseCalls[0].id)
}

func TestBalanceCheckService_RunBalanceCheck_SkipsDisabledAccountAndCanDisableQuotaRequirement(t *testing.T) {
	repo := &balanceCheckRepoStub{
		schedulableAccounts: []Account{
			{
				ID:          301,
				Type:        AccountTypeAPIKey,
				Credentials: map[string]any{"api_key": "no-quota"},
				Extra:       map[string]any{},
			},
			{
				ID:          302,
				Type:        AccountTypeAPIKey,
				Credentials: map[string]any{"api_key": "disabled"},
				Extra:       map[string]any{"balance_check_disabled": true},
			},
		},
	}
	svc := NewBalanceCheckService(repo, &config.Config{
		BalanceCheck: config.BalanceCheckConfig{RequireQuotaHourlyLimit: balanceCheckBoolPtr(false), PauseDurationHours: 5},
	})
	svc.httpClient = balanceHTTPClient(map[string]float64{"no-quota": 50, "disabled": 49})

	svc.runBalanceCheck()

	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, int64(301), repo.updateExtraCalls[0].id)
}

func TestBalanceCheckService_RunBalanceCheck_RefreshesAutoPausedAccounts(t *testing.T) {
	future := time.Now().Add(time.Hour)
	repo := &balanceCheckRepoStub{
		schedulableAccounts: []Account{
			{
				ID:          201,
				Type:        AccountTypeAPIKey,
				Credentials: map[string]any{"api_key": "active"},
				Extra:       map[string]any{"quota_hourly_limit": 1.0},
			},
		},
		tempAccounts: []Account{
			{
				ID:                      202,
				Type:                    AccountTypeAPIKey,
				Credentials:             map[string]any{"api_key": "paused"},
				Extra:                   map[string]any{"quota_hourly_limit": 1.0, "balance": 100.0},
				TempUnschedulableUntil:  &future,
				TempUnschedulableReason: "Balance decreased from 100.0000 to 94.0000 (auto-pause by balance check)",
			},
		},
	}
	svc := NewBalanceCheckService(repo, nil)
	svc.httpClient = balanceHTTPClient(map[string]float64{"active": 50, "paused": 49})

	svc.runBalanceCheck()

	require.Len(t, repo.updateExtraCalls, 2)
	require.ElementsMatch(t, []int64{201, 202}, []int64{repo.updateExtraCalls[0].id, repo.updateExtraCalls[1].id})
	require.Empty(t, repo.clearCalls)
}
