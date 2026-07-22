package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type trashListAdminService struct {
	*stubAdminService
	trashed []service.Account
}

func (s *trashListAdminService) ListTrashedAccounts(context.Context, int, int, string, string, string) ([]service.Account, int64, error) {
	return s.trashed, int64(len(s.trashed)), nil
}

func TestAccountHandlerListTrashUsesPublicAccountFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	createdAt := time.Date(2026, time.July, 1, 2, 3, 4, 0, time.UTC)
	deletedAt := time.Date(2026, time.July, 22, 10, 11, 12, 0, time.UTC)
	adminSvc := &trashListAdminService{
		stubAdminService: newStubAdminService(),
		trashed: []service.Account{{
			ID:          42,
			Name:        "archived-openai",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Status:      "inactive",
			Schedulable: false,
			Concurrency: 4,
			CreatedAt:   createdAt,
			UpdatedAt:   deletedAt,
			DeletedAt:   &deletedAt,
		}},
	}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.GET("/api/v1/admin/accounts/trash", handler.ListTrash)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/trash?page=1&page_size=20", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Items, 1)
	item := payload.Data.Items[0]
	require.Equal(t, float64(42), item["id"])
	require.Equal(t, "archived-openai", item["name"])
	require.Equal(t, "openai", item["platform"])
	require.Equal(t, "oauth", item["type"])
	require.Equal(t, deletedAt.Format(time.RFC3339), item["deleted_at"])
	require.NotEmpty(t, item["created_at"])
	require.NotContains(t, item, "ID")
	require.NotContains(t, item, "Name")
}
