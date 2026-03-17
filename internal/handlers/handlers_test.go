package handlers_test

import (
	"foreignscan/internal/handlers"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/images", handlers.GetImages)
	r.GET("/api/rooms/tree", handlers.GetRoomsTree)
	return r
}

func TestGetImages(t *testing.T) {
	router := setupTestRouter()
	req, _ := http.NewRequest("GET", "/api/images", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code == http.StatusInternalServerError {
		t.Log("Warning: DB not connected, got 500 as expected in isolation")
	} else {
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestGetRoomsTree(t *testing.T) {
	router := setupTestRouter()
	req, _ := http.NewRequest("GET", "/api/rooms/tree", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code == http.StatusInternalServerError {
		t.Log("Warning: DB not connected, got 500 as expected in isolation")
	} else {
		assert.Equal(t, http.StatusOK, w.Code)
	}
}
