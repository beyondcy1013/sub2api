package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stickySessionAdminStoreStub struct {
	calls          int
	summarizeCalls int
	summary        *service.StickySessionBindingSummary
	activeWithin   time.Duration
}

type stickySessionAdminServiceStub struct {
	*stubAdminService
	byID map[int64]service.Account
}

func (s *stickySessionAdminServiceStub) GetAccount(_ context.Context, id int64) (*service.Account, error) {
	account, ok := s.byID[id]
	if !ok {
		return nil, service.ErrAccountNotFound
	}
	copy := account
	return &copy, nil
}

func (s *stickySessionAdminServiceStub) GetAccountsByIDs(_ context.Context, ids []int64) ([]*service.Account, error) {
	accounts := make([]*service.Account, 0, len(ids))
	for _, id := range ids {
		if account, ok := s.byID[id]; ok {
			copy := account
			accounts = append(accounts, &copy)
		}
	}
	return accounts, nil
}

func (s *stickySessionAdminStoreStub) Summarize(context.Context, int64, string) (*service.StickySessionBindingSummary, error) {
	s.summarizeCalls++
	if s.summary != nil {
		return s.summary, nil
	}
	return &service.StickySessionBindingSummary{Counts: map[int64]int{}}, nil
}

func (s *stickySessionAdminStoreStub) Reassign(_ context.Context, _ int64, _ string, _, _ int64, _ int, activeWithin time.Duration) (*service.StickySessionReassignResult, error) {
	s.calls++
	s.activeWithin = activeWithin
	return &service.StickySessionReassignResult{Moved: 2, RemainingSourceBindings: 3}, nil
}

func newStickySessionTestRouter(t *testing.T, sourceGroupID int64) (*gin.Engine, *stickySessionAdminStoreStub) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	accounts := []service.Account{
		{
			ID: 12, Name: "source", Platform: service.PlatformOpenAI, Status: service.StatusActive,
			Schedulable: true, AccountGroups: []service.AccountGroup{{GroupID: sourceGroupID}}, GroupIDs: []int64{sourceGroupID},
		},
		{
			ID: 20, Name: "target", Platform: service.PlatformOpenAI, Status: service.StatusActive,
			Schedulable: true, AccountGroups: []service.AccountGroup{{GroupID: 2}}, GroupIDs: []int64{2},
		},
	}
	base := newStubAdminService()
	base.accounts = accounts
	adminService := &stickySessionAdminServiceStub{stubAdminService: base, byID: map[int64]service.Account{}}
	for _, account := range accounts {
		adminService.byID[account.ID] = account
	}
	store := &stickySessionAdminStoreStub{}
	handler := NewAccountHandler(adminService, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.stickySessionAdminStore = store
	router := gin.New()
	router.POST("/accounts/:id/sticky-sessions/reassign", handler.ReassignStickySessions)
	return router, store
}

func TestReassignStickySessionsRequiresSharedGroup(t *testing.T) {
	router, store := newStickySessionTestRouter(t, 3)
	req := httptest.NewRequest(http.MethodPost, "/accounts/20/sticky-sessions/reassign", bytes.NewBufferString(`{"group_id":2,"source_account_id":12,"count":2,"active_within_seconds":300}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, store.calls)
}

func TestReassignStickySessionsMovesWithinSharedGroup(t *testing.T) {
	router, store := newStickySessionTestRouter(t, 2)
	req := httptest.NewRequest(http.MethodPost, "/accounts/20/sticky-sessions/reassign", bytes.NewBufferString(`{"group_id":2,"source_account_id":12,"count":2,"active_within_seconds":300}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, store.calls)
	require.Equal(t, 5*time.Minute, store.activeWithin)
	require.Contains(t, rec.Body.String(), `"moved":2`)
}

func TestGetStickySessionSummaryRejectsUnschedulableTarget(t *testing.T) {
	_, store := newStickySessionTestRouter(t, 2)
	base := newStubAdminService()
	target := service.Account{
		ID: 20, Name: "target", Platform: service.PlatformOpenAI, Status: service.StatusActive,
		Schedulable: false, AccountGroups: []service.AccountGroup{{GroupID: 2}}, GroupIDs: []int64{2},
	}
	adminService := &stickySessionAdminServiceStub{
		stubAdminService: base,
		byID:             map[int64]service.Account{target.ID: target},
	}
	handler := NewAccountHandler(adminService, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.stickySessionAdminStore = store
	router := gin.New()
	router.GET("/accounts/:id/sticky-sessions", handler.GetStickySessionSummary)

	req := httptest.NewRequest(http.MethodGet, "/accounts/20/sticky-sessions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, store.summarizeCalls)
}

func TestGetStickySessionSummaryOnlyReturnsMovableSources(t *testing.T) {
	accounts := []service.Account{
		{ID: 20, Name: "target", Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true, AccountGroups: []service.AccountGroup{{GroupID: 2}}, GroupIDs: []int64{2}},
		{ID: 12, Name: "valid-source", Platform: service.PlatformOpenAI, AccountGroups: []service.AccountGroup{{GroupID: 2}}, GroupIDs: []int64{2}},
		{ID: 13, Name: "cross-platform", Platform: service.PlatformGrok, AccountGroups: []service.AccountGroup{{GroupID: 2}}, GroupIDs: []int64{2}},
		{ID: 14, Name: "old-group", Platform: service.PlatformOpenAI, AccountGroups: []service.AccountGroup{{GroupID: 3}}, GroupIDs: []int64{3}},
	}
	base := newStubAdminService()
	adminService := &stickySessionAdminServiceStub{stubAdminService: base, byID: map[int64]service.Account{}}
	for _, account := range accounts {
		adminService.byID[account.ID] = account
	}
	store := &stickySessionAdminStoreStub{summary: &service.StickySessionBindingSummary{
		Counts: map[int64]int{12: 2, 13: 3, 14: 4, 20: 5, 99: 6}, Total: 20,
		Activities: map[int64][]service.StickySessionBindingActivity{
			12: {
				{SessionHash: "11111111aaaaaaaa", ActiveAgo: 10 * time.Second},
				{SessionHash: "22222222bbbbbbbb", ActiveAgo: 10 * time.Minute},
			},
		},
	}}
	handler := NewAccountHandler(adminService, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.stickySessionAdminStore = store
	router := gin.New()
	router.GET("/accounts/:id/sticky-sessions", handler.GetStickySessionSummary)

	req := httptest.NewRequest(http.MethodGet, "/accounts/20/sticky-sessions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"total":2`)
	require.Contains(t, rec.Body.String(), `"account_name":"valid-source"`)
	require.Contains(t, rec.Body.String(), `"60":1`)
	require.Contains(t, rec.Body.String(), `"300":1`)
	require.Contains(t, rec.Body.String(), `"900":2`)
	require.Contains(t, rec.Body.String(), `"3600":2`)
	require.Contains(t, rec.Body.String(), `"session_suffix":"aaaaaaaa"`)
	require.NotContains(t, rec.Body.String(), "cross-platform")
	require.NotContains(t, rec.Body.String(), "old-group")
}
