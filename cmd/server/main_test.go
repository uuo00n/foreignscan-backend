package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDocsRoutesRedirectToSwagger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	setupRoutes(router)

	testCases := []string{"/", "/docs", "/docs/"}
	for _, path := range testCases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusTemporaryRedirect {
			t.Fatalf("expected %s to redirect with 307, got %d", path, resp.Code)
		}
		if location := resp.Header().Get("Location"); location != "/swagger/index.html" {
			t.Fatalf("expected %s to redirect to /swagger/index.html, got %q", path, location)
		}
	}
}
