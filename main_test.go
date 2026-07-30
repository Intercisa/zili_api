package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	router.GET("/api/status-weights", getStatusWeights)
	router.GET("/api/milk-transfer", getMilkTransfer)
	router.GET("/api/milk-consumed", getMilkConsumed)
	router.GET("/api/summary", getSummary)
	router.GET("/api/vitamins", getVitamins)
	router.PUT("/api/vitamins/:key", putVitamin)
	router.POST("/api/logs", createLog)
	router.PUT("/api/logs/:id", updateLog)
	router.DELETE("/api/logs/:id", deleteLog)
	router.GET("/api/growth", getGrowth)
	router.POST("/api/growth", createGrowth)
	router.GET("/api/settings/:key", getSetting)
	router.PUT("/api/settings/:key", putSetting)
	router.GET("/api/sleep-awake", getSleepAwake)
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

func TestGetStatusWeights_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/status-weights", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

func TestGetMilkConsumed_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/milk-consumed", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

func TestGetGrowth_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/growth", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

func TestCreateGrowth_InvalidJSON(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/growth", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateGrowth_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{"logDate": "2026-01-01"})
	req, _ := http.NewRequest("POST", "/api/growth", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

func TestGetSetting_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/settings/birth-date", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

func TestPutSetting_InvalidJSON(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/settings/birth-date", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPutSetting_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"value": "2025-01-15"})
	req, _ := http.NewRequest("PUT", "/api/settings/birth-date", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

func TestGetSleepAwake_NoDB(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sleep-awake", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when db is nil, got %d", w.Code)
	}
}

// Pure logic tests for calcSleepAwake — no DB needed

func mustLocal(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
	return t
}

func TestCalcSleepAwake_SimpleSameDay(t *testing.T) {
	logs := []sleepLog{
		{Date: "2025-01-10", Time: "22:00", Summary: "elaludt"},
	}
	now := mustLocal("2025-01-11 08:00")
	result := calcSleepAwake(logs, "2025-01-10", "2025-01-11", now)

	if len(result) != 2 {
		t.Fatalf("expected 2 days, got %d", len(result))
	}
	// day1: sleep from 22:00 to midnight = 120 min
	if result[0].SleepMin != 120 {
		t.Errorf("day1 SleepMin: expected 120, got %d", result[0].SleepMin)
	}
	// day2: sleep from midnight to 08:00 = 480 min (ongoing)
	if result[1].SleepMin != 480 {
		t.Errorf("day2 SleepMin: expected 480, got %d", result[1].SleepMin)
	}
}

func TestCalcSleepAwake_OvernightSleepWithWake(t *testing.T) {
	logs := []sleepLog{
		{Date: "2025-01-10", Time: "22:00", Summary: "elaludt"},
		{Date: "2025-01-11", Time: "07:00", Summary: "ébredt"},
	}
	now := mustLocal("2025-01-11 12:00")
	result := calcSleepAwake(logs, "2025-01-10", "2025-01-11", now)

	// day1: 22:00 to midnight = 120 min sleep, 22*60=1320 awake
	if result[0].SleepMin != 120 {
		t.Errorf("day1 SleepMin: expected 120, got %d", result[0].SleepMin)
	}
	// day2: midnight to 07:00 = 420 min sleep
	if result[1].SleepMin != 420 {
		t.Errorf("day2 SleepMin: expected 420, got %d", result[1].SleepMin)
	}
	// day2: elapsed = 12h = 720 min, awake = 720 - 420 = 300
	if result[1].AwakeMin != 300 {
		t.Errorf("day2 AwakeMin: expected 300, got %d", result[1].AwakeMin)
	}
}

func TestCalcSleepAwake_MultipleSleepsOneDay(t *testing.T) {
	logs := []sleepLog{
		{Date: "2025-01-10", Time: "09:00", Summary: "elaludt"},
		{Date: "2025-01-10", Time: "10:00", Summary: "ébredt"},
		{Date: "2025-01-10", Time: "13:00", Summary: "cicin elaludt"},
		{Date: "2025-01-10", Time: "14:30", Summary: "ébredt"},
	}
	now := mustLocal("2025-01-10 23:59")
	result := calcSleepAwake(logs, "2025-01-10", "2025-01-10", now)

	// 60 + 90 = 150 min sleep
	if result[0].SleepMin != 150 {
		t.Errorf("SleepMin: expected 150, got %d", result[0].SleepMin)
	}
}

func TestCalcSleepAwake_NoLogs(t *testing.T) {
	now := mustLocal("2025-01-10 12:00")
	result := calcSleepAwake([]sleepLog{}, "2025-01-10", "2025-01-10", now)

	if len(result) != 1 {
		t.Fatalf("expected 1 day, got %d", len(result))
	}
	if result[0].SleepMin != 0 {
		t.Errorf("expected 0 sleep, got %d", result[0].SleepMin)
	}
	// awake = elapsed since midnight = 12*60 = 720
	if result[0].AwakeMin != 720 {
		t.Errorf("expected 720 awake, got %d", result[0].AwakeMin)
	}
}

func TestCalcSleepAwake_SleepWithoutWake_Ongoing(t *testing.T) {
	logs := []sleepLog{
		{Date: "2025-01-10", Time: "20:00", Summary: "elaludt"},
	}
	now := mustLocal("2025-01-10 22:00")
	result := calcSleepAwake(logs, "2025-01-10", "2025-01-10", now)

	// sleep from 20:00 to now 22:00 = 120 min
	if result[0].SleepMin != 120 {
		t.Errorf("SleepMin: expected 120, got %d", result[0].SleepMin)
	}
}

