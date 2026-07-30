package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type DailyLog struct {
	ID                int      `json:"id"`
	LogDate           string   `json:"logDate"`
	LogTime           *string  `json:"logTime"`
	DailySummary      string   `json:"dailySummary"`
	StatusWeightG     *int     `json:"statusWeightG"`
	PreFeedWeightG    *int     `json:"preFeedWeightG"`
	PostFeedWeightG   *int     `json:"postFeedWeightG"`
	MilkTransferG     *int     `json:"milkTransferG"`
	HeightCm          *float64 `json:"heightCm"`
	HeadCm            *float64 `json:"headCm"`
	MeasurementWeight *int     `json:"measurementWeightG"`
}

type WeightPoint struct {
	Date   string `json:"date"`
	Weight int    `json:"weight"`
}

type MilkTransferPoint struct {
	Date         string `json:"date"`
	MilkTransfer int    `json:"milkTransferG"`
}

type MilkConsumedPoint struct {
	Date          string `json:"date"`
	MilkConsumedG int    `json:"milkConsumedG"`
}

type GrowthPoint struct {
	Date     string   `json:"date"`
	WeightG  *int     `json:"weightG"`
	HeightCm *float64 `json:"heightCm"`
	HeadCm   *float64 `json:"headCm"`
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

	router := gin.Default()

	router.Static("/static", "./static")

	router.GET("/", dashboard)
	router.GET("/health", health)

	router.GET("/api/logs", getLogs)
	router.GET("/api/logs/:date", getLogByDate)
	router.POST("/api/logs", createLog)
	router.PUT("/api/logs/:id", updateLog)
	router.DELETE("/api/logs/:id", deleteLog)
	router.GET("/api/weights", getWeights)
	router.GET("/api/status-weights", getStatusWeights)
	router.GET("/api/milk-transfer", getMilkTransfer)
	router.GET("/api/milk-consumed", getMilkConsumed)
	router.GET("/api/summary", getSummary)
	router.GET("/api/vitamins", getVitamins)
	router.PUT("/api/vitamins/:key", putVitamin)
	router.GET("/api/growth", getGrowth)
	router.POST("/api/growth", createGrowth)
	router.GET("/api/settings/:key", getSetting)
	router.PUT("/api/settings/:key", putSetting)
	router.GET("/api/sleep-awake", getSleepAwake)
	router.GET("/api/current-status", getCurrentStatus)

	router.GET("/logs", getLogs)
	router.GET("/logs/:date", getLogByDate)
	router.GET("/weights", getWeights)
	router.GET("/milk-transfer", getMilkTransfer)
	router.GET("/summary", getSummary)

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

func dashboard(c *gin.Context) {
	c.File("./static/index.html")
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

func getLogs(c *gin.Context) {
	limit := 100
	offset := 0
	if v := c.Query("offset"); v != "" {
		fmt.Sscanf(v, "%d", &offset)
	}
	search := c.Query("search")
	rows, err := db.Query(`
		SELECT id, log_date, log_time, daily_summary, status_weight_g,
		       pre_feed_weight_g, post_feed_weight_g, milk_transfer_g,
		       height_cm, head_cm, measurement_weight_g
		FROM zili_daily_log
		WHERE ($3 = '' OR daily_summary ILIKE '%' || $3 || '%')
		ORDER BY log_date DESC, COALESCE(log_time, '00:00') DESC, id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var logs []DailyLog
	for rows.Next() {
		var entry DailyLog
		var logTime sql.NullString
		var heightCm, headCm sql.NullFloat64
		err := rows.Scan(
			&entry.ID, &entry.LogDate, &logTime, &entry.DailySummary,
			&entry.StatusWeightG, &entry.PreFeedWeightG, &entry.PostFeedWeightG,
			&entry.MilkTransferG, &heightCm, &headCm, &entry.MeasurementWeight,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if logTime.Valid {
			entry.LogTime = &logTime.String
		}
		if heightCm.Valid {
			entry.HeightCm = &heightCm.Float64
		}
		if headCm.Valid {
			entry.HeadCm = &headCm.Float64
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
		       height_cm, head_cm, measurement_weight_g
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
		var heightCm, headCm sql.NullFloat64
		err := rows.Scan(
			&entry.ID, &entry.LogDate, &logTime, &entry.DailySummary,
			&entry.StatusWeightG, &entry.PreFeedWeightG, &entry.PostFeedWeightG,
			&entry.MilkTransferG, &heightCm, &headCm, &entry.MeasurementWeight,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if logTime.Valid {
			entry.LogTime = &logTime.String
		}
		if heightCm.Valid {
			entry.HeightCm = &heightCm.Float64
		}
		if headCm.Valid {
			entry.HeadCm = &headCm.Float64
		}
		logs = append(logs, entry)
	}
	c.JSON(http.StatusOK, logs)
}

func getWeights(c *gin.Context) {
	to := c.DefaultQuery("to", time.Now().Format("2006-01-02"))
	from := c.DefaultQuery("from", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	rows, err := db.Query(`
		SELECT log_date, measurement_weight_g
		FROM zili_daily_log
		WHERE measurement_weight_g IS NOT NULL
		  AND log_date BETWEEN $1 AND $2
		ORDER BY log_date, id
	`, from, to)
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

func getStatusWeights(c *gin.Context) {
	to := c.DefaultQuery("to", time.Now().Format("2006-01-02"))
	from := c.DefaultQuery("from", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	rows, err := db.Query(`
		SELECT log_date, status_weight_g
		FROM zili_daily_log
		WHERE status_weight_g IS NOT NULL
		  AND log_date BETWEEN $1 AND $2
		ORDER BY log_date, id
	`, from, to)
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

func getMilkConsumed(c *gin.Context) {
	to := c.DefaultQuery("to", time.Now().Format("2006-01-02"))
	from := c.DefaultQuery("from", time.Now().AddDate(0, 0, -6).Format("2006-01-02"))

	rows, err := db.Query(`
		SELECT log_date, SUM(post_feed_weight_g - pre_feed_weight_g) as milk_consumed
		FROM zili_daily_log
		WHERE pre_feed_weight_g IS NOT NULL
		  AND post_feed_weight_g IS NOT NULL
		  AND post_feed_weight_g > pre_feed_weight_g
		  AND log_date BETWEEN $1 AND $2
		GROUP BY log_date
		ORDER BY log_date ASC
	`, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var data []MilkConsumedPoint
	for rows.Next() {
		var item MilkConsumedPoint
		if err := rows.Scan(&item.Date, &item.MilkConsumedG); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		data = append(data, item)
	}
	c.JSON(http.StatusOK, data)
}

func getGrowth(c *gin.Context) {
	rows, err := db.Query(`
		SELECT log_date, measurement_weight_g, height_cm, head_cm
		FROM zili_daily_log
		WHERE height_cm IS NOT NULL OR head_cm IS NOT NULL
		ORDER BY log_date ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var data []GrowthPoint
	for rows.Next() {
		var item GrowthPoint
		var weightG sql.NullInt64
		var heightCm, headCm sql.NullFloat64
		if err := rows.Scan(&item.Date, &weightG, &heightCm, &headCm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if weightG.Valid {
			v := int(weightG.Int64)
			item.WeightG = &v
		}
		if heightCm.Valid {
			item.HeightCm = &heightCm.Float64
		}
		if headCm.Valid {
			item.HeadCm = &headCm.Float64
		}
		data = append(data, item)
	}
	c.JSON(http.StatusOK, data)
}

func createGrowth(c *gin.Context) {
	var body struct {
		LogDate  string   `json:"logDate"`
		WeightG  *int     `json:"weightG"`
		HeightCm *float64 `json:"heightCm"`
		HeadCm   *float64 `json:"headCm"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var id int
	err := db.QueryRow(`
		INSERT INTO zili_daily_log (log_date, measurement_weight_g, height_cm, head_cm, daily_summary)
		VALUES ($1, $2, $3, $4, '')
		RETURNING id
	`, body.LogDate, body.WeightG, body.HeightCm, body.HeadCm).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
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
			height_cm, head_cm, measurement_weight_g
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id
	`,
		entry.LogDate, entry.LogTime, entry.DailySummary,
		entry.StatusWeightG, entry.PreFeedWeightG, entry.PostFeedWeightG,
		entry.MilkTransferG, entry.HeightCm, entry.HeadCm, entry.MeasurementWeight,
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func updateLog(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		DailySummary string   `json:"dailySummary"`
		LogDate      string   `json:"logDate"`
		LogTime      *string  `json:"logTime"`
		HeightCm     *float64 `json:"heightCm"`
		HeadCm       *float64 `json:"headCm"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := db.Exec(
		`UPDATE zili_daily_log SET daily_summary = $1, log_date = $2, log_time = $3, height_cm = $4, head_cm = $5 WHERE id = $6`,
		body.DailySummary, body.LogDate, body.LogTime, body.HeightCm, body.HeadCm, id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func deleteLog(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec(`DELETE FROM zili_daily_log WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
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

type sleepLog struct {
	Date    string
	Time    string
	Summary string
}

type daySleepResult struct {
	Date     string `json:"Date"`
	SleepMin int    `json:"SleepMin"`
	AwakeMin int    `json:"AwakeMin"`
}

func calcSleepAwake(logs []sleepLog, from, to string, now time.Time) []daySleepResult {
	sleepTags := []string{"elaludt", "cicin elaludt"}
	toTime := func(date, t string) time.Time {
		dt, _ := time.ParseInLocation("2006-01-02 15:04", date+" "+t, time.Local)
		return dt
	}
	dayMap := map[string]*daySleepResult{}
	start, _ := time.ParseInLocation("2006-01-02", from, time.Local)
	end, _ := time.ParseInLocation("2006-01-02", to, time.Local)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		dayMap[key] = &daySleepResult{Date: key}
	}
	addInterval := func(sleepStart, wakeTime time.Time) {
		cur := sleepStart
		for cur.Before(wakeTime) {
			dayEnd := time.Date(cur.Year(), cur.Month(), cur.Day()+1, 0, 0, 0, 0, time.Local)
			intervalEnd := wakeTime
			if dayEnd.Before(wakeTime) {
				intervalEnd = dayEnd
			}
			if dr, ok := dayMap[cur.Format("2006-01-02")]; ok {
				dr.SleepMin += int(intervalEnd.Sub(cur).Minutes())
			}
			cur = dayEnd
		}
	}
	var sleepStart *time.Time
	for _, l := range logs {
		isSleep := false
		for _, tag := range sleepTags {
			if strings.Contains(l.Summary, tag) {
				isSleep = true
				break
			}
		}
		isWake := strings.Contains(l.Summary, "ébredt")
		if isSleep && sleepStart == nil {
			t := toTime(l.Date, l.Time)
			sleepStart = &t
		} else if isWake && sleepStart != nil {
			wt := toTime(l.Date, l.Time)
			addInterval(*sleepStart, wt)
			sleepStart = nil
		}
	}
	if sleepStart != nil {
		addInterval(*sleepStart, now)
	}
	var result []daySleepResult
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		dr := dayMap[key]
		dayEndTime := d.AddDate(0, 0, 1)
		if now.Before(dayEndTime) {
			dayEndTime = now
		}
		elapsed := int(dayEndTime.Sub(d).Minutes())
		dr.AwakeMin = elapsed - dr.SleepMin
		if dr.AwakeMin < 0 {
			dr.AwakeMin = 0
		}
		result = append(result, *dr)
	}
	return result
}

func getSleepAwake(c *gin.Context) {
	to := c.DefaultQuery("to", time.Now().Format("2006-01-02"))
	from := c.DefaultQuery("from", time.Now().AddDate(0, 0, -6).Format("2006-01-02"))

	rows, err := db.Query(`
		SELECT log_date::text, log_time::text, daily_summary
		FROM zili_daily_log
		WHERE log_time IS NOT NULL
		  AND daily_summary IS NOT NULL
		  AND (daily_summary ILIKE '%elaludt%' OR daily_summary ILIKE '%ébredt%')
		  AND log_date BETWEEN $1::date - INTERVAL '1 day' AND $2::date
		ORDER BY log_date, log_time
	`, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var logs []sleepLog
	for rows.Next() {
		var r sleepLog
		if err := rows.Scan(&r.Date, &r.Time, &r.Summary); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if len(r.Date) > 10 { r.Date = r.Date[:10] }
		if len(r.Time) > 5 { r.Time = r.Time[:5] }
		logs = append(logs, r)
	}
	c.JSON(http.StatusOK, calcSleepAwake(logs, from, to, time.Now()))
}

func getCurrentStatus(c *gin.Context) {
	now := time.Now()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	rows, err := db.Query(`
		SELECT log_date::text, log_time::text, daily_summary
		FROM zili_daily_log
		WHERE log_time IS NOT NULL AND daily_summary IS NOT NULL
		  AND (daily_summary ILIKE '%elaludt%' OR daily_summary ILIKE '%ébredt%')
		  AND log_date IN ($1, $2)
		ORDER BY log_date, log_time
	`, yesterday, today)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type logRow struct{ Date, Time, Summary string }
	var logs []logRow
	for rows.Next() {
		var r logRow
		rows.Scan(&r.Date, &r.Time, &r.Summary)
		if len(r.Date) > 10 { r.Date = r.Date[:10] }
		if len(r.Time) > 5 { r.Time = r.Time[:5] }
		logs = append(logs, r)
	}

	toTime := func(date, t string) time.Time {
		dt, _ := time.ParseInLocation("2006-01-02 15:04", date+" "+t, time.Local)
		return dt
	}
	sleepTags := []string{"elaludt", "cicin elaludt"}

	var lastSleep, lastWake *time.Time
	for _, l := range logs {
		isSleep := false
		for _, tag := range sleepTags {
			if strings.Contains(l.Summary, tag) { isSleep = true; break }
		}
		t := toTime(l.Date, l.Time)
		if isSleep { lastSleep = &t }
		if strings.Contains(l.Summary, "ébredt") { lastWake = &t }
	}

	fmtDur := func(ms int64) string {
		h := ms / 3600000
		m := (ms % 3600000) / 60000
		if h > 0 { return fmt.Sprintf("%dh %dm", h, m) }
		return fmt.Sprintf("%dm", m)
	}

	// awake status
	isSleeping := lastSleep != nil && (lastWake == nil || lastSleep.After(*lastWake))
	var statusDur, statusState string
	if isSleeping {
		diff := now.Sub(*lastSleep).Milliseconds()
		statusDur = fmtDur(diff)
		statusState = "sleep"
	} else if lastWake != nil {
		diff := now.Sub(*lastWake).Milliseconds()
		statusDur = fmtDur(diff)
		statusState = "awake"
	} else {
		statusDur = ""
		statusState = "sleep"
	}

	// today sleep/awake totals
	midnight, _ := time.ParseInLocation("2006-01-02", today, time.Local)
	result := calcSleepAwake(func() []sleepLog {
		var sl []sleepLog
		for _, l := range logs {
			sl = append(sl, sleepLog{Date: l.Date, Time: l.Time, Summary: l.Summary})
		}
		return sl
	}(), yesterday, today, now)

	var sleepMin, awakeMin int
	for _, r := range result {
		if r.Date == today {
			sleepMin = r.SleepMin
			awakeMin = r.AwakeMin
		}
	}
	_ = midnight

	c.JSON(http.StatusOK, gin.H{
		"state":    statusState,
		"duration": statusDur,
		"sleepMin": sleepMin,
		"awakeMin": awakeMin,
	})
}

func getSetting(c *gin.Context) {
	key := c.Param("key")
	var value string
	err := db.QueryRow(`SELECT value FROM app_settings WHERE key = $1`, key).Scan(&value)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"value": nil})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"value": value})
}

func putSetting(c *gin.Context) {
	key := c.Param("key")
	var body struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = $2
	`, key, body.Value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key, "value": body.Value})
}

