package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseVersion(t *testing.T) {
	got := parseVersion([]byte("VERSION=0.1.9-mu.16\n"))
	if got != "0.1.9-mu.16" {
		t.Fatalf("parseVersion = %q, want %q", got, "0.1.9-mu.16")
	}
}

func TestCacheControlMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(cacheControlMiddleware())
	r.GET("/fs/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/other", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/fs/x", nil))
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("/fs/ Cache-Control = %q, want immutable", cc)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/other", nil))
	if cc := w.Header().Get("Cache-Control"); cc != "" {
		t.Fatalf("/other Cache-Control = %q, want empty", cc)
	}
}
