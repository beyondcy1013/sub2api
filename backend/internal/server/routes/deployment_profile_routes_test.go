//go:build unit

package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDeploymentProfileAdminRouteMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		profile     string
		wantSticky  bool
		wantBalance bool
	}{
		{profile: config.DeploymentProfileMain, wantSticky: true, wantBalance: false},
		{profile: config.DeploymentProfileFree, wantSticky: false, wantBalance: true},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			router := gin.New()
			settingService := service.NewSettingService(nil, &config.Config{
				Deployment: config.DeploymentConfig{Profile: tt.profile},
			})
			registerAccountRoutes(router.Group("/api/v1/admin"), &handler.Handlers{Admin: &handler.AdminHandlers{}}, nil, settingService.RuntimeCapabilities().StickySessionReassignment)
			registerSettingsRoutes(router.Group("/api/v1/admin"), &handler.Handlers{Admin: &handler.AdminHandlers{}}, settingService.RuntimeCapabilities().BalanceCheck)

			routes := make(map[string]struct{})
			for _, route := range router.Routes() {
				routes[route.Method+" "+route.Path] = struct{}{}
			}

			_, hasStickySummary := routes["GET /api/v1/admin/accounts/:id/sticky-sessions"]
			_, hasStickyReassign := routes["POST /api/v1/admin/accounts/:id/sticky-sessions/reassign"]
			_, hasBalanceGet := routes["GET /api/v1/admin/settings/balance-check"]
			_, hasBalancePut := routes["PUT /api/v1/admin/settings/balance-check"]

			require.Equal(t, tt.wantSticky, hasStickySummary)
			require.Equal(t, tt.wantSticky, hasStickyReassign)
			require.Equal(t, tt.wantBalance, hasBalanceGet)
			require.Equal(t, tt.wantBalance, hasBalancePut)
		})
	}
}
