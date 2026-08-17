package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupAccountDataClearRouter() (*gin.Engine, *stubAdminService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()
	adminSvc.filterAccountLifecycle = true
	h := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/data/clear/preview", h.PreviewImportedDataClear)
	router.POST("/api/v1/admin/accounts/data/clear", h.ClearImportedData)
	return router, adminSvc
}

func TestMatchImportedDataAccountUsesStableCredentialIdentity(t *testing.T) {
	item := DataAccount{
		Name:     "renamed-import",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"email":              "owner@example.com",
			"chatgpt_account_id": "acct-123",
		},
	}
	candidates := []service.Account{
		{ID: 10, Name: "edited", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-123"}},
		{ID: 11, Name: "renamed-import", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-other"}},
		{ID: 12, Name: "edited", Platform: service.PlatformAnthropic, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-123"}},
	}

	matches, ambiguous := matchImportedDataAccount(item, candidates)
	require.False(t, ambiguous)
	require.Len(t, matches, 1)
	require.Equal(t, int64(10), matches[0].ID)
}

func TestMatchImportedDataAccountRejectsAmbiguousNameFallback(t *testing.T) {
	item := DataAccount{Name: "same", Platform: service.PlatformAnthropic, Type: service.AccountTypeOAuth, Credentials: map[string]any{"unknown": "value"}}
	candidates := []service.Account{
		{ID: 20, Name: "same", Platform: service.PlatformAnthropic, Type: service.AccountTypeOAuth},
		{ID: 21, Name: "same", Platform: service.PlatformAnthropic, Type: service.AccountTypeOAuth},
	}

	matches, ambiguous := matchImportedDataAccount(item, candidates)
	require.True(t, ambiguous)
	require.Empty(t, matches)
}

func TestClearImportedDataMovesAllCredentialMatchesToDeletedStaging(t *testing.T) {
	router, adminSvc := setupAccountDataClearRouter()
	adminSvc.accounts = []service.Account{
		{ID: 31, Name: "first copy", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-123"}},
		{ID: 32, Name: "second copy", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-123"}},
		{ID: 33, Name: "other", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-other"}},
	}
	adminSvc.deleteAccountErrors = map[int64]error{32: errors.New("delete unavailable")}

	body, err := json.Marshal(DataClearRequest{Data: DataPayload{
		Type: dataType, Version: dataVersion, Proxies: []DataProxy{},
		Accounts: []DataAccount{
			{Name: "source", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-123"}},
			{Name: "missing", Platform: service.PlatformAnthropic, Type: service.AccountTypeOAuth, Credentials: map[string]any{"email": "missing@example.com"}},
		},
	}})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data/clear", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var responseBody struct {
		Code int             `json:"code"`
		Data DataClearResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responseBody))
	require.Equal(t, 0, responseBody.Code)
	require.Equal(t, 2, responseBody.Data.AccountRequested)
	require.Equal(t, 2, responseBody.Data.AccountMatched)
	require.Equal(t, 1, responseBody.Data.AccountCleared)
	require.Equal(t, 1, responseBody.Data.AccountDeletedStaged)
	require.Zero(t, responseBody.Data.AccountPermanentlyDeleted)
	require.Equal(t, 1, responseBody.Data.AccountNotFound)
	require.Equal(t, 1, responseBody.Data.AccountFailed)
	require.Len(t, responseBody.Data.Errors, 2)

	sort.Slice(adminSvc.deletedAccountIDs, func(i, j int) bool { return adminSvc.deletedAccountIDs[i] < adminSvc.deletedAccountIDs[j] })
	require.Equal(t, []int64{31, 32}, adminSvc.deletedAccountIDs)
	require.Empty(t, adminSvc.permanentlyDeletedAccountIDs)
}

func TestPreviewImportedDataClearMatchesWithoutChangingAccounts(t *testing.T) {
	router, adminSvc := setupAccountDataClearRouter()
	adminSvc.accounts = []service.Account{
		{ID: 41, Name: "matching", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-preview"}},
		{ID: 42, Name: "deleted matching", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-preview"}, Extra: map[string]any{service.AccountDeletedStagingExtraKey: true}},
	}

	body, err := json.Marshal(DataClearRequest{Data: DataPayload{
		Type: dataType, Version: dataVersion, Proxies: []DataProxy{},
		Accounts: []DataAccount{{Name: "source", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-preview"}}},
	}, PermanentDelete: true})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data/clear/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var responseBody struct {
		Code int                    `json:"code"`
		Data DataClearPreviewResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responseBody))
	require.Equal(t, 0, responseBody.Code)
	require.Equal(t, 1, responseBody.Data.AccountRequested)
	require.Equal(t, 2, responseBody.Data.AccountMatched)
	require.Empty(t, adminSvc.deletedAccountIDs)
	require.Empty(t, adminSvc.permanentlyDeletedAccountIDs)
}

func TestPreviewImportedDataClearExcludesDeletedStagingByDefault(t *testing.T) {
	router, adminSvc := setupAccountDataClearRouter()
	adminSvc.accounts = []service.Account{{
		ID: 43, Name: "deleted only", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth,
		Credentials: map[string]any{"chatgpt_account_id": "acct-deleted-only"},
		Extra:       map[string]any{service.AccountDeletedStagingExtraKey: true},
	}}

	body, err := json.Marshal(DataClearRequest{Data: DataPayload{
		Type: dataType, Version: dataVersion, Proxies: []DataProxy{},
		Accounts: []DataAccount{{Name: "source", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-deleted-only"}}},
	}})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data/clear/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var responseBody struct {
		Data DataClearPreviewResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responseBody))
	require.Zero(t, responseBody.Data.AccountMatched)
	require.Equal(t, 1, responseBody.Data.AccountNotFound)
}

func TestClearImportedDataPermanentModeDeletesNormalAndDeletedStagingMatches(t *testing.T) {
	router, adminSvc := setupAccountDataClearRouter()
	adminSvc.accounts = []service.Account{
		{ID: 51, Name: "normal", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-hard"}},
		{ID: 52, Name: "already deleted", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-hard"}, Extra: map[string]any{service.AccountDeletedStagingExtraKey: true}},
	}

	body, err := json.Marshal(DataClearRequest{Data: DataPayload{
		Type: dataType, Version: dataVersion, Proxies: []DataProxy{},
		Accounts: []DataAccount{{Name: "source", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_id": "acct-hard"}}},
	}, PermanentDelete: true})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data/clear", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var responseBody struct {
		Code int             `json:"code"`
		Data DataClearResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responseBody))
	require.Equal(t, 0, responseBody.Code)
	require.Equal(t, 2, responseBody.Data.AccountMatched)
	require.Equal(t, 2, responseBody.Data.AccountCleared)
	require.Zero(t, responseBody.Data.AccountDeletedStaged)
	require.Equal(t, 2, responseBody.Data.AccountPermanentlyDeleted)
	require.Equal(t, []int64{51}, adminSvc.deletedAccountIDs)
	sort.Slice(adminSvc.permanentlyDeletedAccountIDs, func(i, j int) bool {
		return adminSvc.permanentlyDeletedAccountIDs[i] < adminSvc.permanentlyDeletedAccountIDs[j]
	})
	require.Equal(t, []int64{51, 52}, adminSvc.permanentlyDeletedAccountIDs)
}
