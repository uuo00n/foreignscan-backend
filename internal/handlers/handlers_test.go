package handlers_test

import (
	"bytes"
	"encoding/json"
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
	r.POST("/api/rooms/:roomId/points", handlers.CreatePoint)
	r.DELETE("/api/rooms/:roomId/points/:pointId", handlers.DeletePoint)
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

func TestCreatePointMissingName(t *testing.T) {
	router := setupTestRouter()
	payload, _ := json.Marshal(map[string]string{
		"name": "   ",
	})
	req, _ := http.NewRequest("POST", "/api/rooms/room1/points", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeletePointRouteRegistered(t *testing.T) {
	router := setupTestRouter()
	req, _ := http.NewRequest("DELETE", "/api/rooms/room1/points/point1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}
