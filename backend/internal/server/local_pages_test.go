package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func TestLocalBalanceCheckSettingsPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware2.CSPNonceKey, "test-nonce")
		c.Next()
	})
	registerLocalPages(r)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/local/balance-check-settings", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"余额检测设置",
		"/api/v1/admin/settings/balance-check",
		`nonce="test-nonce"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("page missing %q", want)
		}
	}
}
