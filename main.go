package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const createTablesSQLPath = "sql/create_tables.sql"

type DailyLog struct {
	ID                int     `json:"id"`
	LogDate           string  `json:"logDate"`
	LogTime           *string `json:"logTime"`
	DailySummary      string  `json:"dailySummary"`
	StatusWeightG     *int    `json:"statusWeightG"`
	PreFeedWeightG    *int    `json:"preFeedWeightG"`
	PostFeedWeightG   *int    `json:"postFeedWeightG"`
	MilkTransferG     *int    `json:"milkTransferG"`
	ExpressedLeftMl   *int    `json:"expressedLeftMl"`
	MeasurementWeight *int    `json:"measurementWeightG"`
}

type WeightPoint struct {
	Date   string `json:"date"`
	Weight int    `json:"weight"`
}

type MilkTransferPoint struct {
	Date         string `json:"date"`
	MilkTransfer int    `json:"milkTransferG"`
}

type Summary struct {
	TotalLogs     int  `json:"totalLogs"`
	WeightEntries int  `json:"weightEntries"`
	FirstWeight   *int `json:"firstWeight"`
	LatestWeight  *int `json:"latestWeight"`
	WeightGain    *int `json:"weightGain"`
	MilkEntries   int  `json:"milkEntries"`
	AverageMilkG  *int `json:"averageMilkG"`
	MinWeight     *int `json:"minWeight"`
	MaxWeight     *int `json:"maxWeight"`
}

var db *sql.DB

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables or defaults")
	}

	connStr :=
		"host=" + getEnv("DB_HOST", "localhost") +
			" port=" + getEnv("DB_PORT", "5432") +
			" user=" + getEnv("DB_USER", "user") +
			" password=" + getEnv("DB_PASSWORD", "password") +
			" dbname=" + getEnv("DB_NAME", "zili") +
			" sslmode=disable"

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	if err := initSchema(db); err != nil {
		log.Fatalf("failed to initialize database schema: %v", err)
	}
	log.Println("Database schema initialized successfully")

	router := gin.Default()

	router.Static("/static", "./static")

	router.GET("/", dashboard)
	router.GET("/health", health)

	router.GET("/api/logs", getLogs)
	router.GET("/api/logs/:date", getLogByDate)
	router.GET("/api/weights", getWeights)
	router.GET("/api/milk-transfer", getMilkTransfer)
	router.GET("/api/summary", getSummary)
	router.GET("/api/vitamins", getVitamins)
	router.PUT("/api/vitamins/:key", putVitamin)

	router.GET("/logs", getLogs)
	router.GET("/logs/:date", getLogByDate)
	router.GET("/weights", getWeights)
	router.GET("/milk-transfer", getMilkTransfer)
	router.GET("/summary", getSummary)

	router.POST("/api/logs", createLog)

	log.Println("Server started on :8081")
	err = router.Run(":8081")
	if err != nil {
		log.Fatal(err)
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// initSchema reads the SQL schema file and executes it against the
// database, creating any tables that don't already exist. It is safe
// to run on every startup since the schema uses "CREATE TABLE IF NOT
// EXISTS" statements, and it also tolerates "already exists" errors
// from the driver in case that isn't the case for some statement.
func initSchema(db *sql.DB) error {
	schema, err := os.ReadFile(createTablesSQLPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", createTablesSQLPath, err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") {
			log.Printf("schema objects already exist, continuing: %v", err)
			return nil
		}
		return fmt.Errorf("failed to execute schema from %s: %w", createTablesSQLPath, err)
	}

	return nil
}

func dashboard(c *gin.Context) {
	c.File("./static/index.html")
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

func getLogs(c *gin.Context) {
	rows, err := db.Query(`
		SELECT id, log_date, log_time, daily_summary, status_weight_g,
		       pre_feed_weight_g, post_feed_weight_g, milk_transfer_g,
		       expressed_left_ml, measurement_weight_g
		FROM zili_daily_log
		ORDER BY log_date DESC, COALESCE(log_time, '00:00') DESC, id DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var logs []DailyLog
	for rows.Next() {
		var entry DailyLog
		var logTime sql.NullString
		err := rows.Scan(
			&entry.ID, &entry.LogDate, &logTime, &entry.DailySummary,
			&entry.StatusWeightG, &entry.PreFeedWeightG, &entry.PostFeedWeightG,
			&entry.MilkTransferG, &entry.ExpressedLeftMl, &entry.MeasurementWeight,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if logTime.Valid {
			entry.LogTime = &logTime.String
		}
		logs = append(logs, entry)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func getLogByDate(c *gin.Context) {
	date := c.Param("date")
	rows, err := db.Query(`
		SELECT id, log_date, log_time, daily_summary, status_weight_g,
		       pre_feed_weight_g, post_feed_weight_g, milk_transfer_g,
		       expressed_left_ml, measurement_weight_g
		FROM zili_daily_log
		WHERE log_date = $1
		ORDER BY COALESCE(log_time, '00:00'), id
	`, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var logs []DailyLog
	for rows.Next() {
		var entry DailyLog
		var logTime sql.NullString
		err := rows.Scan(
			&entry.ID, &entry.LogDate, &logTime, &entry.DailySummary,
			&entry.StatusWeightG, &entry.PreFeedWeightG, &entry.PostFeedWeightG,
			&entry.MilkTransferG, &entry.ExpressedLeftMl, &entry.MeasurementWeight,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if logTime.Valid {
			entry.LogTime = &logTime.String
		}
		logs = append(logs, entry)
	}
	c.JSON(http.StatusOK, logs)
}

func getWeights(c *gin.Context) {
	rows, err := db.Query(`
		SELECT log_date, measurement_weight_g
		FROM zili_daily_log
		WHERE measurement_weight_g IS NOT NULL
		ORDER BY log_date, id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var data []WeightPoint
	for rows.Next() {
		var item WeightPoint
		if err := rows.Scan(&item.Date, &item.Weight); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		data = append(data, item)
	}
	c.JSON(http.StatusOK, data)
}

func getMilkTransfer(c *gin.Context) {
	rows, err := db.Query(`
		SELECT log_date, milk_transfer_g
		FROM zili_daily_log
		WHERE milk_transfer_g IS NOT NULL
		ORDER BY log_date, id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var data []MilkTransferPoint
	for rows.Next() {
		var item MilkTransferPoint
		if err := rows.Scan(&item.Date, &item.MilkTransfer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		data = append(data, item)
	}
	c.JSON(http.StatusOK, data)
}

func createLog(c *gin.Context) {
	var entry DailyLog
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id int
	err := db.QueryRow(`
		INSERT INTO zili_daily_log (
			log_date, log_time, daily_summary, status_weight_g,
			pre_feed_weight_g, post_feed_weight_g, milk_transfer_g,
			expressed_left_ml, measurement_weight_g
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id
	`,
		entry.LogDate, entry.LogTime, entry.DailySummary,
		entry.StatusWeightG, entry.PreFeedWeightG, entry.PostFeedWeightG,
		entry.MilkTransferG, entry.ExpressedLeftMl, entry.MeasurementWeight,
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func getSummary(c *gin.Context) {
	var summary Summary

	db.QueryRow(`SELECT COUNT(*) FROM zili_daily_log`).Scan(&summary.TotalLogs)
	db.QueryRow(`SELECT COUNT(*) FROM zili_daily_log WHERE measurement_weight_g IS NOT NULL`).Scan(&summary.WeightEntries)
	db.QueryRow(`SELECT COUNT(*) FROM zili_daily_log WHERE milk_transfer_g IS NOT NULL`).Scan(&summary.MilkEntries)

	var firstWeight, latestWeight, minWeight, maxWeight sql.NullInt64
	var averageMilk sql.NullFloat64

	db.QueryRow(`SELECT measurement_weight_g FROM zili_daily_log WHERE measurement_weight_g IS NOT NULL ORDER BY log_date ASC, id ASC LIMIT 1`).Scan(&firstWeight)
	db.QueryRow(`SELECT measurement_weight_g FROM zili_daily_log WHERE measurement_weight_g IS NOT NULL ORDER BY log_date DESC, id DESC LIMIT 1`).Scan(&latestWeight)
	db.QueryRow(`SELECT MIN(measurement_weight_g), MAX(measurement_weight_g) FROM zili_daily_log WHERE measurement_weight_g IS NOT NULL`).Scan(&minWeight, &maxWeight)
	db.QueryRow(`SELECT AVG(milk_transfer_g) FROM zili_daily_log WHERE milk_transfer_g IS NOT NULL`).Scan(&averageMilk)

	if firstWeight.Valid {
		v := int(firstWeight.Int64)
		summary.FirstWeight = &v
	}
	if latestWeight.Valid {
		v := int(latestWeight.Int64)
		summary.LatestWeight = &v
	}
	if firstWeight.Valid && latestWeight.Valid {
		v := int(latestWeight.Int64 - firstWeight.Int64)
		summary.WeightGain = &v
	}
	if minWeight.Valid {
		v := int(minWeight.Int64)
		summary.MinWeight = &v
	}
	if maxWeight.Valid {
		v := int(maxWeight.Int64)
		summary.MaxWeight = &v
	}
	if averageMilk.Valid {
		v := int(averageMilk.Float64 + 0.5)
		summary.AverageMilkG = &v
	}

	c.JSON(http.StatusOK, summary)
}

func getVitamins(c *gin.Context) {
	rows, err := db.Query(`SELECT key, checked, date FROM vitamin_checks`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type entry struct {
		Checked bool    `json:"checked"`
		Date    *string `json:"date"`
	}
	result := map[string]entry{}
	for rows.Next() {
		var key string
		var checked bool
		var date sql.NullString
		if err := rows.Scan(&key, &checked, &date); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		e := entry{Checked: checked}
		if date.Valid {
			e.Date = &date.String
		}
		result[key] = e
	}
	c.JSON(http.StatusOK, result)
}

func putVitamin(c *gin.Context) {
	key := c.Param("key")
	var body struct {
		Checked bool   `json:"checked"`
		Date    string `json:"date"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := db.Exec(`
		INSERT INTO vitamin_checks (key, checked, date) VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET checked = $2, date = $3
	`, key, body.Checked, body.Date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key, "checked": body.Checked})
}

