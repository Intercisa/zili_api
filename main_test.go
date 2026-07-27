package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/health", health)
	router.GET("/api/logs", getLogs)
	router.GET("/api/logs/:date", getLogByDate)
	router.GET("/api/weights", getWeights)
	router.GET("/api/milk-transfer", getMilkTransfer)
	router.GET("/api/summary", getSummary)
	router.GET("/api/vitamins", getVitamins)
	router.PUT("/api/vitamins/:key", putVitamin)
	router.POST("/api/logs", createLog)
	router.PUT("/api/logs/:id", updateLog)
	router.DELETE("/api/logs/:id", deleteLog)
	return router
}

func TestHealth(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "UP" {
		t.Errorf("expected status UP, got %s", body["status"])
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_KEY", "hello")
	if got := getEnv("TEST_KEY", "default"); got != "hello" {
		t.Errorf("expected hello, got %s", got)
	}
	if got := getEnv("MISSING_KEY", "default"); got != "default" {
		t.Errorf("expected default, got %s", got)
	}
}

func TestCreateLog_InvalidJSON(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/logs", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPutVitamin_InvalidJSON(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/vitamins/d-vitamin", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetLogs_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/logs", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

func TestUpdateLog_InvalidJSON(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/logs/1", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateLog_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"dailySummary": "updated summary"})
	req, _ := http.NewRequest("PUT", "/api/logs/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

func TestDeleteLog_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/logs/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

func TestUpdateLog_InvalidJSON_MissingDate(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{
		"dailySummary": "updated summary",
		"logDate":      "",
		"logTime":      nil,
	})
	req, _ := http.NewRequest("PUT", "/api/logs/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

func TestUpdateLog_WithDateAndTime_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	logTime := "14:30"
	body, _ := json.Marshal(map[string]interface{}{
		"dailySummary": "updated summary",
		"logDate":      "2026-07-27",
		"logTime":      logTime,
	})
	req, _ := http.NewRequest("PUT", "/api/logs/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

func TestUpdateLog_WithDateNoTime_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{
		"dailySummary": "updated summary",
		"logDate":      "2026-07-27",
		"logTime":      nil,
	})
	req, _ := http.NewRequest("PUT", "/api/logs/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

