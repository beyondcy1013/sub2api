package autodetection

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandlerRoundTripAndDatabaseIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mainEngine, _, _, _ := fixture(t)
	free := New(&memoryStore{data: map[string]string{}}, &fakeExecutor{})
	for _, engine := range []*Engine{mainEngine, free} {
		router := gin.New()
		h := Handler{engine}
		router.GET("/policies", h.Get)
		router.PUT("/policies", h.Update)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/policies", nil))
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
	c := Defaults()
	c.Policies[0].Enabled = true
	raw, _ := json.Marshal(c)
	router := gin.New()
	router.PUT("/policies", Handler{mainEngine}.Update)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/policies", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	untouched, err := free.Config(context.Background())
	if err != nil || untouched.Policies[0].Enabled {
		t.Fatal("main changed free config")
	}
	for _, body := range []string{`{`, `{"policies":[]}`} {
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPut, "/policies", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	}
}
func TestHandlerUnavailableAndCorruptSettings(t *testing.T) {
	router := gin.New()
	router.GET("/unavailable", Handler{}.Get)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/unavailable", nil))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
	e, s, _, _ := fixture(t)
	_ = s.Set(context.Background(), ConfigKey, "{")
	router.GET("/corrupt", Handler{e}.Get)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/corrupt", nil))
	if w.Code != 500 {
		t.Fatal(w.Code)
	}
}
