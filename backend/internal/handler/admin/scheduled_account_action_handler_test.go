package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type fakeScheduledAccountActionManager struct {
	scheduledAccount int64
	scheduledAction  service.ScheduledAccountActionType
	scheduledDelay   time.Duration
	canceledAccount  int64
	current          *service.ScheduledAccountAction
}

func (m *fakeScheduledAccountActionManager) Schedule(_ context.Context, accountID int64, action service.ScheduledAccountActionType, delay time.Duration) (*service.ScheduledAccountAction, error) {
	m.scheduledAccount = accountID
	m.scheduledAction = action
	m.scheduledDelay = delay
	return &service.ScheduledAccountAction{ID: 1, AccountID: accountID, Action: action, ExecuteAt: time.Now().Add(delay)}, nil
}

func (m *fakeScheduledAccountActionManager) GetScheduledAction(_ context.Context, _ int64) (*service.ScheduledAccountAction, error) {
	return m.current, nil
}

func (m *fakeScheduledAccountActionManager) CancelScheduledAction(_ context.Context, accountID int64) error {
	m.canceledAccount = accountID
	return nil
}

func newScheduledAccountActionTestRouter(manager scheduledAccountActionManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := &ScheduledAccountActionHandler{service: manager}
	router.GET("/accounts/:id/scheduled-action", handler.Get)
	router.PUT("/accounts/:id/scheduled-action", handler.Upsert)
	router.DELETE("/accounts/:id/scheduled-action", handler.Delete)
	return router
}

func TestScheduledAccountActionHandlerUpsertValidatesDelayAndAction(t *testing.T) {
	manager := &fakeScheduledAccountActionManager{}
	router := newScheduledAccountActionTestRouter(manager)

	for _, body := range []string{
		`{"action":"pause","hours":0,"minutes":0}`,
		`{"action":"unknown","hours":0,"minutes":1}`,
		`{"action":"pause","hours":-1,"minutes":1}`,
	} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/accounts/7/scheduled-action", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("body %s status = %d, want 400; response=%s", body, recorder.Code, recorder.Body.String())
		}
	}
}

func TestScheduledAccountActionHandlerCRUD(t *testing.T) {
	manager := &fakeScheduledAccountActionManager{current: &service.ScheduledAccountAction{ID: 3, AccountID: 7, Action: service.ScheduledAccountActionPause}}
	router := newScheduledAccountActionTestRouter(manager)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/accounts/7/scheduled-action", bytes.NewBufferString(`{"action":"enable_and_recover","hours":2,"minutes":15}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("PUT status=%d response=%s", recorder.Code, recorder.Body.String())
	}
	if manager.scheduledAccount != 7 || manager.scheduledAction != service.ScheduledAccountActionEnableAndRecover || manager.scheduledDelay != 2*time.Hour+15*time.Minute {
		t.Fatalf("scheduled account=%d action=%s delay=%s", manager.scheduledAccount, manager.scheduledAction, manager.scheduledDelay)
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/accounts/7/scheduled-action", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status=%d response=%s", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		Data *service.ScheduledAccountAction `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil || envelope.Data == nil || envelope.Data.ID != 3 {
		t.Fatalf("GET body=%s err=%v", recorder.Body.String(), err)
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/accounts/7/scheduled-action", nil))
	if recorder.Code != http.StatusOK || manager.canceledAccount != 7 {
		t.Fatalf("DELETE status=%d account=%d response=%s", recorder.Code, manager.canceledAccount, recorder.Body.String())
	}
}
