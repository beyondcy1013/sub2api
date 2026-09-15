package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type pinAdminServiceStub struct {
	*stubAdminService
	pinnedIDs   []int64
	unpinnedIDs []int64
}

func (s *pinAdminServiceStub) PinAccount(_ context.Context, id int64) error {
	s.pinnedIDs = append(s.pinnedIDs, id)
	return nil
}

func (s *pinAdminServiceStub) UnpinAccount(_ context.Context, id int64) error {
	s.unpinnedIDs = append(s.unpinnedIDs, id)
	return nil
}

func TestAccountHandlerPinAndUnpin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := &pinAdminServiceStub{
		stubAdminService: newStubAdminService(),
	}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.POST("/api/v1/admin/accounts/:id/pin", handler.Pin)
	router.POST("/api/v1/admin/accounts/:id/unpin", handler.Unpin)

	t.Run("pin valid account", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/123/pin", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(t, float64(0), resp["code"])
		require.Equal(t, []int64{123}, adminSvc.pinnedIDs)
	})

	t.Run("unpin valid account", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/123/unpin", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(t, float64(0), resp["code"])
		require.Equal(t, []int64{123}, adminSvc.unpinnedIDs)
	})

	t.Run("invalid account id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/abc/pin", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
