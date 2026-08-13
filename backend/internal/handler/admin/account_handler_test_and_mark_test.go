package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type accountTestRunnerStub struct {
	mu      sync.Mutex
	results map[int64]*service.ScheduledTestResult
	seen    []int64
}

func (s *accountTestRunnerStub) TestAccountConnection(*gin.Context, int64, string, string, string, ...service.AccountTestOptions) error {
	return errors.New("upstream rejected credentials")
}

func (s *accountTestRunnerStub) RunTestBackground(_ context.Context, id int64, _ string) (*service.ScheduledTestResult, error) {
	s.mu.Lock()
	s.seen = append(s.seen, id)
	s.mu.Unlock()
	if result := s.results[id]; result != nil {
		return result, nil
	}
	return nil, errors.New("runner failed")
}

type accountTestMarkingAdminService struct {
	*stubAdminService
	mu     sync.Mutex
	marked map[int64]string
}

func (s *accountTestMarkingAdminService) SetAccountError(_ context.Context, id int64, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.marked[id] = message
	return nil
}

func newAccountTestFailureWindow() *service.SuperPriorityService {
	return service.NewSuperPriorityService(nil, &config.Config{SuperPriority: config.SuperPriorityConfig{
		FailureThreshold: 2,
	}})
}

func TestAccountHandlerTestOnlyMarksAfterFailureWindowThreshold(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := &accountTestMarkingAdminService{stubAdminService: newStubAdminService(), marked: map[int64]string{}}
	handler := &AccountHandler{
		adminService:         adminSvc,
		accountTestRunner:    &accountTestRunnerStub{},
		superPriorityService: newAccountTestFailureWindow(),
	}
	router := gin.New()
	router.POST("/accounts/:id/test", handler.Test)

	for attempt := 1; attempt <= 2; attempt++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/accounts/42/test", strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		if attempt == 1 {
			require.Empty(t, adminSvc.marked, "a single failed test must only submit an observation")
		}
	}

	require.Equal(t, "upstream rejected credentials", adminSvc.marked[42])
}

func TestAccountHandlerBatchTestAndMarkContinuesAndSummarizes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := &accountTestMarkingAdminService{stubAdminService: newStubAdminService(), marked: map[int64]string{}}
	runner := &accountTestRunnerStub{results: map[int64]*service.ScheduledTestResult{
		1: {Status: "success"},
		2: {Status: "failed", ErrorMessage: "invalid token"},
		3: {Status: "success"},
	}}
	handler := &AccountHandler{
		adminService:         adminSvc,
		accountTestRunner:    runner,
		superPriorityService: newAccountTestFailureWindow(),
	}
	router := gin.New()
	router.POST("/accounts/batch-test-and-mark", handler.BatchTestAndMark)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/accounts/batch-test-and-mark", strings.NewReader(`{"account_ids":[1,2,3]}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"code":0,"message":"success","data":{"success":2,"failed":1,"marked":0,"results":[{"account_id":1,"success":true},{"account_id":2,"success":false,"error":"invalid token"},{"account_id":3,"success":true}]}}`, recorder.Body.String())
	require.Empty(t, adminSvc.marked, "one batch failure must only submit an observation")
	require.ElementsMatch(t, []int64{1, 2, 3}, runner.seen)
}

func TestAccountHandlerBatchTestAndMarkMarksWhenWindowThresholdIsReached(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := &accountTestMarkingAdminService{stubAdminService: newStubAdminService(), marked: map[int64]string{}}
	runner := &accountTestRunnerStub{results: map[int64]*service.ScheduledTestResult{
		2: {Status: "failed", ErrorMessage: "invalid token"},
	}}
	handler := &AccountHandler{
		adminService:         adminSvc,
		accountTestRunner:    runner,
		superPriorityService: newAccountTestFailureWindow(),
	}
	router := gin.New()
	router.POST("/accounts/batch-test-and-mark", handler.BatchTestAndMark)

	for attempt := 1; attempt <= 2; attempt++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/accounts/batch-test-and-mark", strings.NewReader(`{"account_ids":[2]}`))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		if attempt == 1 {
			require.Contains(t, recorder.Body.String(), `"marked":0`)
		} else {
			require.Contains(t, recorder.Body.String(), `"marked":1`)
		}
	}
	require.Equal(t, "invalid token", adminSvc.marked[2])
}
