package main

import (
        "database/sql"
        "encoding/json"
        "fmt"
        "log"
        "net/http"
        "os"
        "strings"
        "time"
        _ "time/tzdata"

        "github.com/SherClockHolmes/webpush-go"
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
        SleepEvent        *string  `json:"sleepEvent"`
        Diaper            *string  `json:"diaper"`
        FedBreast         bool     `json:"fedBreast"`
        FedBottle         bool     `json:"fedBottle"`
        FedFormula        bool     `json:"fedFormula"`
        Bathed            bool     `json:"bathed"`
        Milestone         bool     `json:"milestone"`
        Pending           bool     `json:"pending"`
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

type ChecklistItem struct {
        ID       int    `json:"id"`
        ListID   int    `json:"listId"`
        Text     string `json:"text"`
        Checked  bool   `json:"checked"`
        Position int    `json:"position"`
}

type Event struct {
        ID          int             `json:"id"`
        Title       string          `json:"title"`
        Category    string          `json:"category"`
        EventDate   string          `json:"eventDate"`
        EventTime   *string         `json:"eventTime"`
        DurationMin int             `json:"durationMin"`
        Notes       *string         `json:"notes"`
        Recurring   string          `json:"recurring"`
        AllDay      bool            `json:"allDay"`
        Items       []ChecklistItem `json:"items,omitempty"`
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
var vapidPublic, vapidPrivate string

func budapest() *time.Location {
        loc, _ := time.LoadLocation("Europe/Budapest")
        return loc
}

func nowBp() time.Time {
        return time.Now().In(budapest())
}

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

        if err = loadOrGenerateVAPIDKeys(); err != nil {
                log.Fatal("vapid init failed:", err)
        }

        router := gin.Default()

        router.Static("/static", "./static")
        router.StaticFile("/sw.js", "./static/sw.js")
        router.StaticFile("/manifest.json", "./static/manifest.json")
        router.StaticFile("/manifest-quick.json", "./static/manifest-quick.json")

        router.GET("/", dashboard)
        router.GET("/quick", func(c *gin.Context) { c.File("./static/quick.html") })
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
        router.GET("/api/events", getEvents)
        router.POST("/api/events", createEvent)
        router.PUT("/api/events/:id", updateEvent)
        router.DELETE("/api/events/:id", deleteEvent)
        router.POST("/api/events/:id/items", addChecklistItem)
        router.PUT("/api/checklist-items/:itemId", toggleChecklistItem)
        router.DELETE("/api/checklist-items/:itemId", deleteChecklistItem)
        router.GET("/api/events/calendar.ics", getCalendar)
        router.GET("/api/last-feed", getLastFeed)
        router.GET("/api/push-vapid-key", getVapidPublicKey)
        router.POST("/api/push-subscribe", pushSubscribe)
        router.GET("/api/pending-feed", getPendingFeed)
        router.DELETE("/api/pending-feed/:id", deletePendingFeed)
        router.GET("/api/diaper-alert", getDiaperAlert)
        router.GET("/api/words", getWords)
        router.POST("/api/words", createWord)
        router.DELETE("/api/words/:id", deleteWord)
        router.GET("/api/words/random", getRandomWord)
        router.POST("/api/send-report", func(c *gin.Context) {
                if err := sendWeeklyReport(); err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }
                c.JSON(http.StatusOK, gin.H{"status": "sent"})
        })

        go scheduleWeeklyReport()
        go scheduleNapReminder()
        go scheduleFeedReminder()

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

func scheduleWeeklyReport() {
        for {
                now := nowBp()
                // next Monday 08:00
                daysUntilMonday := (8 - int(now.Weekday())) % 7
                next := time.Date(now.Year(), now.Month(), now.Day()+daysUntilMonday, 8, 0, 0, 0, budapest())
                if !next.After(now) {
                        next = next.AddDate(0, 0, 7)
                }
                time.Sleep(time.Until(next))
                if err := sendWeeklyReport(); err != nil {
                        log.Println("weekly report error:", err)
                }
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
                       height_cm, head_cm, measurement_weight_g,
                       sleep_event, diaper, fed_breast, fed_bottle, fed_formula, bathed, milestone
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
                var sleepEvent, diaper sql.NullString
                err := rows.Scan(
                        &entry.ID, &entry.LogDate, &logTime, &entry.DailySummary,
                        &entry.StatusWeightG, &entry.PreFeedWeightG, &entry.PostFeedWeightG,
                        &entry.MilkTransferG, &heightCm, &headCm, &entry.MeasurementWeight,
                        &sleepEvent, &diaper, &entry.FedBreast, &entry.FedBottle, &entry.FedFormula, &entry.Bathed, &entry.Milestone,
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
                if sleepEvent.Valid {
                        entry.SleepEvent = &sleepEvent.String
                }
                if diaper.Valid {
                        entry.Diaper = &diaper.String
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
                       height_cm, head_cm, measurement_weight_g,
                       sleep_event, diaper, fed_breast, fed_bottle, fed_formula, bathed, milestone
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
                var sleepEvent, diaper sql.NullString
                err := rows.Scan(
                        &entry.ID, &entry.LogDate, &logTime, &entry.DailySummary,
                        &entry.StatusWeightG, &entry.PreFeedWeightG, &entry.PostFeedWeightG,
                        &entry.MilkTransferG, &heightCm, &headCm, &entry.MeasurementWeight,
                        &sleepEvent, &diaper, &entry.FedBreast, &entry.FedBottle, &entry.FedFormula, &entry.Bathed, &entry.Milestone,
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
                if sleepEvent.Valid {
                        entry.SleepEvent = &sleepEvent.String
                }
                if diaper.Valid {
                        entry.Diaper = &diaper.String
                }
                logs = append(logs, entry)
        }
        c.JSON(http.StatusOK, logs)
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
                        height_cm, head_cm, measurement_weight_g,
                        sleep_event, diaper, fed_breast, fed_bottle, fed_formula, bathed, milestone, pending
                ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
                RETURNING id
        `,
                entry.LogDate, entry.LogTime, entry.DailySummary,
                entry.StatusWeightG, entry.PreFeedWeightG, entry.PostFeedWeightG,
                entry.MilkTransferG, entry.HeightCm, entry.HeadCm, entry.MeasurementWeight,
                entry.SleepEvent, entry.Diaper, entry.FedBreast, entry.FedBottle, entry.FedFormula, entry.Bathed, entry.Milestone, entry.Pending,
        ).Scan(&id)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.JSON(http.StatusCreated, gin.H{"id": id})
}

func getPendingFeed(c *gin.Context) {
        var entry DailyLog
        var logTime sql.NullString
        var statusWeightG, measurementWeightG sql.NullInt64
        err := db.QueryRow(`
                SELECT id, log_date::text, log_time::text, pre_feed_weight_g, post_feed_weight_g,
                       fed_breast, fed_bottle, status_weight_g, measurement_weight_g, daily_summary
                FROM zili_daily_log WHERE pending = true ORDER BY id DESC LIMIT 1
        `).Scan(&entry.ID, &entry.LogDate, &logTime, &entry.PreFeedWeightG, &entry.PostFeedWeightG,
                &entry.FedBreast, &entry.FedBottle, &statusWeightG, &measurementWeightG, &entry.DailySummary)
        if err == sql.ErrNoRows {
                c.JSON(http.StatusOK, nil)
                return
        }
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        if logTime.Valid {
                entry.LogTime = &logTime.String
        }
        if statusWeightG.Valid {
                v := int(statusWeightG.Int64)
                entry.StatusWeightG = &v
        }
        if measurementWeightG.Valid {
                v := int(measurementWeightG.Int64)
                entry.MeasurementWeight = &v
        }
        c.JSON(http.StatusOK, entry)
}

func deletePendingFeed(c *gin.Context) {
        id := c.Param("id")
        _, err := db.Exec(`DELETE FROM zili_daily_log WHERE id = $1 AND pending = true`, id)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.Status(http.StatusNoContent)
}

func updateLog(c *gin.Context) {
        id := c.Param("id")
        var body struct {
                DailySummary      string   `json:"dailySummary"`
                LogDate           string   `json:"logDate"`
                LogTime           *string  `json:"logTime"`
                HeightCm          *float64 `json:"heightCm"`
                HeadCm            *float64 `json:"headCm"`
                PreFeedWeightG    *int     `json:"preFeedWeightG"`
                PostFeedWeightG   *int     `json:"postFeedWeightG"`
                MilkTransferG     *int     `json:"milkTransferG"`
                StatusWeightG     *int     `json:"statusWeightG"`
                MeasurementWeightG *int    `json:"measurementWeightG"`
                FedBreast         bool     `json:"fedBreast"`
                FedBottle         bool     `json:"fedBottle"`
                FedFormula        bool     `json:"fedFormula"`
                Diaper            *string  `json:"diaper"`
                SleepEvent        *string  `json:"sleepEvent"`
                Bathed            bool     `json:"bathed"`
                Milestone         bool     `json:"milestone"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        _, err := db.Exec(`
                UPDATE zili_daily_log SET
                        daily_summary = $1, log_date = $2, log_time = $3,
                        height_cm = $4, head_cm = $5,
                        pre_feed_weight_g = $6, post_feed_weight_g = $7, milk_transfer_g = $8,
                        status_weight_g = $9, measurement_weight_g = $10,
                        fed_breast = $11, fed_bottle = $12, fed_formula = $13,
                        diaper = $14, sleep_event = $15,
                        bathed = $16, milestone = $17
                WHERE id = $18`,
                body.DailySummary, body.LogDate, body.LogTime,
                body.HeightCm, body.HeadCm,
                body.PreFeedWeightG, body.PostFeedWeightG, body.MilkTransferG,
                body.StatusWeightG, body.MeasurementWeightG,
                body.FedBreast, body.FedBottle, body.FedFormula,
                body.Diaper, body.SleepEvent,
                body.Bathed, body.Milestone, id,
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

func getWeights(c *gin.Context) {
        to := c.DefaultQuery("to", nowBp().Format("2006-01-02"))
        from := c.DefaultQuery("from", nowBp().AddDate(0, -1, 0).Format("2006-01-02"))
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
        data := []WeightPoint{}
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
        to := c.DefaultQuery("to", nowBp().Format("2006-01-02"))
        from := c.DefaultQuery("from", nowBp().AddDate(0, -1, 0).Format("2006-01-02"))
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
        data := []WeightPoint{}
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
        to := c.DefaultQuery("to", nowBp().Format("2006-01-02"))
        from := c.DefaultQuery("from", nowBp().AddDate(0, 0, -6).Format("2006-01-02"))
        rows, err := db.Query(`
                SELECT log_date::text, SUM(milk_transfer_g) as milk_consumed
                FROM zili_daily_log
                WHERE milk_transfer_g IS NOT NULL
                  AND log_date BETWEEN $1 AND $2
                GROUP BY log_date
                ORDER BY log_date ASC
        `, from, to)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        defer rows.Close()
        data := []MilkConsumedPoint{}
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
                VALUES ($1, $2, $3, $4, '') RETURNING id
        `, body.LogDate, body.WeightG, body.HeightCm, body.HeadCm).Scan(&id)
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
        bp := budapest()
        toTime := func(date, t string) time.Time {
                dt, _ := time.ParseInLocation("2006-01-02 15:04", date+" "+t, bp)
                return dt
        }
        dayMap := map[string]*daySleepResult{}
        start, _ := time.ParseInLocation("2006-01-02", from, bp)
        end, _ := time.ParseInLocation("2006-01-02", to, bp)
        for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
                key := d.Format("2006-01-02")
                dayMap[key] = &daySleepResult{Date: key}
        }
        addInterval := func(sleepStart, wakeTime time.Time) {
                cur := sleepStart
                for cur.Before(wakeTime) {
                        dayEnd := time.Date(cur.Year(), cur.Month(), cur.Day()+1, 0, 0, 0, 0, bp)
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
        result := []daySleepResult{}
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
        to := c.DefaultQuery("to", nowBp().Format("2006-01-02"))
        from := c.DefaultQuery("from", nowBp().AddDate(0, 0, -6).Format("2006-01-02"))
        rows, err := db.Query(`
                SELECT log_date::text, log_time::text, sleep_event
                FROM zili_daily_log
                WHERE log_time IS NOT NULL
                  AND sleep_event IS NOT NULL
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
                var sleepEvent string
                if err := rows.Scan(&r.Date, &r.Time, &sleepEvent); err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }
                if len(r.Date) > 10 {
                        r.Date = r.Date[:10]
                }
                if len(r.Time) > 5 {
                        r.Time = r.Time[:5]
                }
                if sleepEvent == "fell_asleep" {
                        r.Summary = "elaludt"
                }
                if sleepEvent == "woke_up" {
                        r.Summary = "ébredt"
                }
                logs = append(logs, r)
        }
        c.JSON(http.StatusOK, calcSleepAwake(logs, from, to, nowBp()))
}

func getCurrentStatus(c *gin.Context) {
        now := nowBp()
        today := now.Format("2006-01-02")
        yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
        rows, err := db.Query(`
                SELECT log_date::text, log_time::text, sleep_event,
                        EXTRACT(EPOCH FROM (
                                NOW() AT TIME ZONE 'Europe/Budapest'
                                - (log_date::text || ' ' || log_time::text)::timestamp
                        )) AS seconds_ago
                FROM zili_daily_log
                WHERE log_time IS NOT NULL
                  AND sleep_event IS NOT NULL
                  AND log_date IN ($1, $2)
                ORDER BY log_date, log_time
        `, yesterday, today)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        defer rows.Close()
        type logRow struct {
                Date       string
                Time       string
                SleepEvent string
                SecondsAgo float64
        }
        var logs []logRow
        for rows.Next() {
                var r logRow
                rows.Scan(&r.Date, &r.Time, &r.SleepEvent, &r.SecondsAgo)
                if len(r.Date) > 10 {
                        r.Date = r.Date[:10]
                }
                if len(r.Time) > 5 {
                        r.Time = r.Time[:5]
                }
                logs = append(logs, r)
        }
        type entry struct{ SecondsAgo float64 }
        var lastSleep, lastWake *entry
        for _, l := range logs {
                e := &entry{SecondsAgo: l.SecondsAgo}
                if l.SleepEvent == "fell_asleep" {
                        lastSleep = e
                }
                if l.SleepEvent == "woke_up" {
                        lastWake = e
                }
        }
        fmtDur := func(seconds float64) string {
                total := int64(seconds)
                h := total / 3600
                m := (total % 3600) / 60
                if h > 0 {
                        return fmt.Sprintf("%dh %dm", h, m)
                }
                return fmt.Sprintf("%dm", m)
        }
        isSleeping := lastSleep != nil && (lastWake == nil || lastSleep.SecondsAgo < lastWake.SecondsAgo)
        var statusDur, statusState string
        if isSleeping {
                statusDur = fmtDur(lastSleep.SecondsAgo)
                statusState = "sleep"
        } else if lastWake != nil {
                statusDur = fmtDur(lastWake.SecondsAgo)
                statusState = "awake"
        } else {
                statusState = "sleep"
        }
        sleepLogs := make([]sleepLog, 0, len(logs))
        for _, l := range logs {
                var summary string
                if l.SleepEvent == "fell_asleep" {
                        summary = "elaludt"
                }
                if l.SleepEvent == "woke_up" {
                        summary = "ébredt"
                }
                sleepLogs = append(sleepLogs, sleepLog{Date: l.Date, Time: l.Time, Summary: summary})
        }
        result := calcSleepAwake(sleepLogs, yesterday, today, nowBp())
        var sleepMin, awakeMin int
        for _, r := range result {
                if r.Date == today {
                        sleepMin = r.SleepMin
                        awakeMin = r.AwakeMin
                }
        }
        c.JSON(http.StatusOK, gin.H{
                "state":    statusState,
                "duration": statusDur,
                "sleepMin": sleepMin,
                "awakeMin": awakeMin,
        })
}

func getEvents(c *gin.Context) {
        rows, err := db.Query(`
                SELECT id, title, category, event_date::text, event_time::text, duration_min, notes, recurring, all_day
                FROM zili_events
                ORDER BY event_date ASC, COALESCE(event_time, '00:00') ASC
        `)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        defer rows.Close()
        var events []Event
        eventIdx := map[int]int{}
        for rows.Next() {
                var e Event
                var eventTime, notes sql.NullString
                if err := rows.Scan(&e.ID, &e.Title, &e.Category, &e.EventDate, &eventTime, &e.DurationMin, &notes, &e.Recurring, &e.AllDay); err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }
                if eventTime.Valid {
                        v := eventTime.String
                        if len(v) > 5 {
                                v = v[:5]
                        }
                        e.EventTime = &v
                }
                if notes.Valid {
                        e.Notes = &notes.String
                }
                if len(e.EventDate) > 10 {
                        e.EventDate = e.EventDate[:10]
                }
                e.Items = []ChecklistItem{}
                eventIdx[e.ID] = len(events)
                events = append(events, e)
        }
        itemRows, err := db.Query(`SELECT id, list_id, text, checked, position FROM zili_checklist_items ORDER BY list_id, position, id`)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        defer itemRows.Close()
        for itemRows.Next() {
                var item ChecklistItem
                if err := itemRows.Scan(&item.ID, &item.ListID, &item.Text, &item.Checked, &item.Position); err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }
                if i, ok := eventIdx[item.ListID]; ok {
                        events[i].Items = append(events[i].Items, item)
                }
        }
        if events == nil {
                c.JSON(http.StatusOK, []Event{})
                return
        }
        c.JSON(http.StatusOK, events)
}

func createEvent(c *gin.Context) {
        var body Event
        if err := c.ShouldBindJSON(&body); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        var id int
        err := db.QueryRow(`
                INSERT INTO zili_events (title, category, event_date, event_time, duration_min, notes, recurring, all_day)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id
        `, body.Title, body.Category, body.EventDate, body.EventTime, body.DurationMin, body.Notes, body.Recurring, body.AllDay).Scan(&id)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.JSON(http.StatusCreated, gin.H{"id": id})
}

func updateEvent(c *gin.Context) {
        id := c.Param("id")
        var body Event
        if err := c.ShouldBindJSON(&body); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        _, err := db.Exec(`
                UPDATE zili_events SET title=$1, category=$2, event_date=$3, event_time=$4, duration_min=$5, notes=$6, recurring=$7, all_day=$8 WHERE id=$9
        `, body.Title, body.Category, body.EventDate, body.EventTime, body.DurationMin, body.Notes, body.Recurring, body.AllDay, id)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.JSON(http.StatusOK, gin.H{"id": id})
}

func deleteEvent(c *gin.Context) {
        id := c.Param("id")
        _, err := db.Exec(`DELETE FROM zili_events WHERE id = $1`, id)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.Status(http.StatusNoContent)
}

func addChecklistItem(c *gin.Context) {
        eventID := c.Param("id")
        var body struct {
                Text string `json:"text"`
        }
        if err := c.ShouldBindJSON(&body); err != nil || body.Text == "" {
                c.JSON(http.StatusBadRequest, gin.H{"error": "text required"})
                return
        }
        var pos int
        db.QueryRow(`SELECT COALESCE(MAX(position), -1) + 1 FROM zili_checklist_items WHERE list_id = $1`, eventID).Scan(&pos)
        var id int
        err := db.QueryRow(`INSERT INTO zili_checklist_items (list_id, text, position) VALUES ($1, $2, $3) RETURNING id`,
                eventID, body.Text, pos).Scan(&id)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.JSON(http.StatusCreated, ChecklistItem{ID: id, Text: body.Text, Checked: false, Position: pos})
}

func toggleChecklistItem(c *gin.Context) {
        itemID := c.Param("itemId")
        var body struct {
                Checked bool `json:"checked"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        _, err := db.Exec(`UPDATE zili_checklist_items SET checked = $1 WHERE id = $2`, body.Checked, itemID)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.JSON(http.StatusOK, gin.H{"id": itemID, "checked": body.Checked})
}

func deleteChecklistItem(c *gin.Context) {
        itemID := c.Param("itemId")
        _, err := db.Exec(`DELETE FROM zili_checklist_items WHERE id = $1`, itemID)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.Status(http.StatusNoContent)
}

type Word struct {
        ID        int     `json:"id"`
        Word      string  `json:"word"`
        NotedDate string  `json:"notedDate"`
        Notes     *string `json:"notes"`
}

func getWords(c *gin.Context) {
        rows, err := db.Query(`SELECT id, word, noted_date::text, notes FROM zili_words ORDER BY noted_date DESC, id DESC`)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        defer rows.Close()
        var words []Word
        for rows.Next() {
                var w Word
                var notes sql.NullString
                rows.Scan(&w.ID, &w.Word, &w.NotedDate, &notes)
                if len(w.NotedDate) > 10 {
                        w.NotedDate = w.NotedDate[:10]
                }
                if notes.Valid {
                        w.Notes = &notes.String
                }
                words = append(words, w)
        }
        if words == nil {
                words = []Word{}
        }
        c.JSON(http.StatusOK, words)
}

func getRandomWord(c *gin.Context) {
        var w Word
        var notes sql.NullString
        err := db.QueryRow(`SELECT id, word, noted_date::text, notes FROM zili_words ORDER BY RANDOM() LIMIT 1`).Scan(&w.ID, &w.Word, &w.NotedDate, &notes)
        if err == sql.ErrNoRows {
                c.JSON(http.StatusNotFound, gin.H{})
                return
        }
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        if len(w.NotedDate) > 10 {
                w.NotedDate = w.NotedDate[:10]
        }
        if notes.Valid {
                w.Notes = &notes.String
        }
        c.JSON(http.StatusOK, w)
}

func createWord(c *gin.Context) {
        var body Word
        if err := c.ShouldBindJSON(&body); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        if body.NotedDate == "" {
                body.NotedDate = nowBp().Format("2006-01-02")
        }
        var id int
        err := db.QueryRow(`INSERT INTO zili_words (word, noted_date, notes) VALUES ($1, $2, $3) RETURNING id`,
                body.Word, body.NotedDate, body.Notes).Scan(&id)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.JSON(http.StatusCreated, gin.H{"id": id})
}

func deleteWord(c *gin.Context) {
        id := c.Param("id")
        _, err := db.Exec(`DELETE FROM zili_words WHERE id = $1`, id)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.Status(http.StatusNoContent)
}

func getDiaperAlert(c *gin.Context) {
        now := nowBp()
        today := now.Format("2006-01-02")

        var lastDirtyDate string
        var lastDirtyTime sql.NullString
        err := db.QueryRow(`
                SELECT log_date::text, log_time::text
                FROM zili_daily_log
                WHERE diaper IN ('dirty', 'both')
                ORDER BY log_date DESC, log_time DESC NULLS LAST
                LIMIT 1
        `).Scan(&lastDirtyDate, &lastDirtyTime)

        if err == sql.ErrNoRows {
                // No dirty diaper ever logged -- nothing to measure elapsed time against.
                c.JSON(http.StatusOK, gin.H{
                        "consecutiveDaysWithoutDirty": 0,
                        "todayHasDirty":               false,
                })
                return
        }
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        if len(lastDirtyDate) > 10 {
                lastDirtyDate = lastDirtyDate[:10]
        }

        timePart := "00:00"
        if lastDirtyTime.Valid && lastDirtyTime.String != "" {
                t := lastDirtyTime.String
                if len(t) >= 5 {
                        timePart = t[:5]
                }
        }

        lastDirtyAt, err := time.ParseInLocation(
                "2006-01-02 15:04",
                lastDirtyDate+" "+timePart,
                now.Location(),
        )
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }

        // True rolling window: how many full 24h periods have elapsed since
        // the last actual dirty diaper timestamp. This increments continuously
        // as real time passes, regardless of calendar-day boundaries.
        elapsed := now.Sub(lastDirtyAt)
        consecutiveDays := int(elapsed.Hours()) / 24

        todayHasDirty := lastDirtyDate == today

        c.JSON(http.StatusOK, gin.H{
                "consecutiveDaysWithoutDirty": consecutiveDays,
                "todayHasDirty":               todayHasDirty,
        })
}

func getCalendar(c *gin.Context) {
        rows, err := db.Query(`
                SELECT id, title, category, event_date::text, event_time::text, duration_min, notes, recurring, all_day
                FROM zili_events
                ORDER BY event_date ASC
        `)
        if err != nil {
                c.String(http.StatusInternalServerError, "error fetching events")
                return
        }
        defer rows.Close()

        now := time.Now().UTC().Format("20060102T150405Z")
        var sb strings.Builder
        sb.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Zili//Zili Dashboard//EN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\n")

        for rows.Next() {
                var e Event
                var eventTime, notes sql.NullString
                if err := rows.Scan(&e.ID, &e.Title, &e.Category, &e.EventDate, &eventTime, &e.DurationMin, &notes, &e.Recurring, &e.AllDay); err != nil {
                        continue
                }
                if len(e.EventDate) > 10 {
                        e.EventDate = e.EventDate[:10]
                }
                if eventTime.Valid && len(eventTime.String) >= 5 {
                        v := eventTime.String[:5]
                        e.EventTime = &v
                }
                if notes.Valid {
                        e.Notes = &notes.String
                }

                dateStr := strings.ReplaceAll(e.EventDate, "-", "")
                uid := fmt.Sprintf("%d@zili", e.ID)

                sb.WriteString("BEGIN:VEVENT\r\n")
                sb.WriteString("UID:" + uid + "\r\n")
                sb.WriteString("DTSTAMP:" + now + "\r\n")
                sb.WriteString("SUMMARY:" + e.Title + "\r\n")

                if e.AllDay {
                        sb.WriteString("DTSTART;VALUE=DATE:" + dateStr + "\r\n")
                        sb.WriteString("DTEND;VALUE=DATE:" + dateStr + "\r\n")
                } else if e.EventTime != nil {
                        timeStr := strings.ReplaceAll(*e.EventTime, ":", "")
                        sb.WriteString("DTSTART;TZID=Europe/Budapest:" + dateStr + "T" + timeStr + "00\r\n")
                        end := fmt.Sprintf("DURATION:PT%dM\r\n", e.DurationMin)
                        sb.WriteString(end)
                } else {
                        sb.WriteString("DTSTART;VALUE=DATE:" + dateStr + "\r\n")
                        sb.WriteString("DTEND;VALUE=DATE:" + dateStr + "\r\n")
                }

                if e.Notes != nil && *e.Notes != "" {
                        sb.WriteString("DESCRIPTION:" + *e.Notes + "\r\n")
                }

                switch e.Recurring {
                case "daily":
                        sb.WriteString("RRULE:FREQ=DAILY\r\n")
                case "weekly":
                        sb.WriteString("RRULE:FREQ=WEEKLY\r\n")
                case "monthly":
                        sb.WriteString("RRULE:FREQ=MONTHLY\r\n")
                case "yearly":
                        sb.WriteString("RRULE:FREQ=YEARLY\r\n")
                }

                sb.WriteString("END:VEVENT\r\n")
        }

        sb.WriteString("END:VCALENDAR\r\n")
        c.Header("Content-Type", "text/calendar; charset=utf-8")
        c.Header("Content-Disposition", "inline; filename=zili.ics")
        c.String(http.StatusOK, sb.String())
}

func getVapidPublicKey(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"publicKey": vapidPublic})
}

func pushSubscribe(c *gin.Context) {
	var sub webpush.Subscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := db.Exec(`
		INSERT INTO push_subscriptions (endpoint, p256dh, auth)
		VALUES ($1, $2, $3)
		ON CONFLICT (endpoint) DO NOTHING
	`, sub.Endpoint, sub.Keys.P256dh, sub.Keys.Auth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusCreated)
}

func sendNapPush() {
	rows, err := db.Query(`SELECT endpoint, p256dh, auth FROM push_subscriptions`)
	if err != nil {
		log.Println("nap push: query subs:", err)
		return
	}
	defer rows.Close()
	payload, _ := json.Marshal(map[string]string{
		"title": "😴 Álmos lehet!",
		"body":  "Több mint 1.5 órája ébren van.",
	})
	for rows.Next() {
		var sub webpush.Subscription
		rows.Scan(&sub.Endpoint, &sub.Keys.P256dh, &sub.Keys.Auth)
		resp, err := webpush.SendNotification(payload, &sub, &webpush.Options{
			VAPIDPublicKey:  vapidPublic,
			VAPIDPrivateKey: vapidPrivate,
			TTL:             3600,
		})
		if err != nil {
			log.Println("nap push send error:", err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == 410 || resp.StatusCode == 404 {
			db.Exec(`DELETE FROM push_subscriptions WHERE endpoint = $1`, sub.Endpoint)
		}
	}
}

func scheduleNapReminder() {
	const napThreshold = 90 * time.Minute
	const checkInterval = 60 * time.Second

	for {
		time.Sleep(checkInterval)

		var lastWakeStr, lastSleepStr sql.NullString
		db.QueryRow(`
			SELECT log_date::text || ' ' || log_time::text
			FROM zili_daily_log
			WHERE sleep_event = 'woke_up' AND log_time IS NOT NULL
			ORDER BY log_date DESC, log_time DESC LIMIT 1
		`).Scan(&lastWakeStr)
		db.QueryRow(`
			SELECT log_date::text || ' ' || log_time::text
			FROM zili_daily_log
			WHERE sleep_event = 'fell_asleep' AND log_time IS NOT NULL
			ORDER BY log_date DESC, log_time DESC LIMIT 1
		`).Scan(&lastSleepStr)

		if !lastWakeStr.Valid {
			continue
		}
		bp := budapest()
		lastWake, err := time.ParseInLocation("2006-01-02 15:04:05", lastWakeStr.String, bp)
		if err != nil {
			continue
		}
		if lastSleepStr.Valid {
			lastSleep, err := time.ParseInLocation("2006-01-02 15:04:05", lastSleepStr.String, bp)
			if err == nil && lastSleep.After(lastWake) {
				continue
			}
		}
		awake := time.Now().In(bp).Sub(lastWake)
		if awake < napThreshold {
			continue
		}
		// persist lastFired in DB so server restarts don't re-fire for the same wake cycle
		var lastFiredStr sql.NullString
		db.QueryRow(`SELECT value FROM app_settings WHERE key = 'nap_reminder_last_fired'`).Scan(&lastFiredStr)
		if lastFiredStr.Valid && lastFiredStr.String != "" {
			lastFired, err := time.ParseInLocation("2006-01-02 15:04:05", lastFiredStr.String, bp)
			if err == nil && !lastFired.Before(lastWake) {
				continue
			}
		}
		db.Exec(`INSERT INTO app_settings (key, value) VALUES ('nap_reminder_last_fired', $1)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
			time.Now().In(bp).Format("2006-01-02 15:04:05"))
		go sendNapPush()
	}
}

func getLastFeed(c *gin.Context) {
	now := nowBp()
	var lastFeedStr sql.NullString
	db.QueryRow(`
		SELECT log_date::text || ' ' || log_time::text
		FROM zili_daily_log
		WHERE log_time IS NOT NULL
		  AND (fed_breast = true OR fed_bottle = true OR fed_formula = true
		       OR milk_transfer_g IS NOT NULL)
		  AND pending = false
		ORDER BY log_date DESC, log_time DESC LIMIT 1
	`).Scan(&lastFeedStr)
	if !lastFeedStr.Valid {
		c.JSON(http.StatusOK, gin.H{"secondsAgo": nil})
		return
	}
	bp := budapest()
	lastFeed, err := time.ParseInLocation("2006-01-02 15:04:05", lastFeedStr.String, bp)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"secondsAgo": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"secondsAgo": int(now.Sub(lastFeed).Seconds())})
}

func sendFeedPush() {
	rows, err := db.Query(`SELECT endpoint, p256dh, auth FROM push_subscriptions`)
	if err != nil {
		return
	}
	defer rows.Close()
	payload, _ := json.Marshal(map[string]string{
		"title": "🍼 Ideje enni!",
		"body":  "Több mint 3 órája evett utoljára.",
	})
	for rows.Next() {
		var sub webpush.Subscription
		rows.Scan(&sub.Endpoint, &sub.Keys.P256dh, &sub.Keys.Auth)
		resp, err := webpush.SendNotification(payload, &sub, &webpush.Options{
			VAPIDPublicKey:  vapidPublic,
			VAPIDPrivateKey: vapidPrivate,
			TTL:             3600,
		})
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == 410 || resp.StatusCode == 404 {
			db.Exec(`DELETE FROM push_subscriptions WHERE endpoint = $1`, sub.Endpoint)
		}
	}
}

func scheduleFeedReminder() {
	const feedThreshold = 3 * time.Hour
	const checkInterval = 60 * time.Second

	for {
		time.Sleep(checkInterval)
		var lastFeedStr sql.NullString
		db.QueryRow(`
			SELECT log_date::text || ' ' || log_time::text
			FROM zili_daily_log
			WHERE log_time IS NOT NULL
			  AND (fed_breast = true OR fed_bottle = true OR fed_formula = true
			       OR milk_transfer_g IS NOT NULL)
			  AND pending = false
			ORDER BY log_date DESC, log_time DESC LIMIT 1
		`).Scan(&lastFeedStr)
		if !lastFeedStr.Valid {
			continue
		}
		bp := budapest()
		lastFeed, err := time.ParseInLocation("2006-01-02 15:04:05", lastFeedStr.String, bp)
		if err != nil {
			continue
		}
		if time.Now().In(bp).Sub(lastFeed) < feedThreshold {
			continue
		}
		var lastFiredStr sql.NullString
		db.QueryRow(`SELECT value FROM app_settings WHERE key = 'feed_reminder_last_fired'`).Scan(&lastFiredStr)
		if lastFiredStr.Valid && lastFiredStr.String != "" {
			lastFired, err := time.ParseInLocation("2006-01-02 15:04:05", lastFiredStr.String, bp)
			if err == nil && !lastFired.Before(lastFeed) {
				continue
			}
		}
		db.Exec(`INSERT INTO app_settings (key, value) VALUES ('feed_reminder_last_fired', $1)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
			time.Now().In(bp).Format("2006-01-02 15:04:05"))
		go sendFeedPush()
	}
}

func loadOrGenerateVAPIDKeys() error {
	pub, priv := "", ""
	db.QueryRow(`SELECT value FROM app_settings WHERE key = 'vapid_public'`).Scan(&pub)
	db.QueryRow(`SELECT value FROM app_settings WHERE key = 'vapid_private'`).Scan(&priv)
	if pub != "" && priv != "" {
		vapidPublic, vapidPrivate = pub, priv
		log.Println("VAPID keys loaded from DB")
		return nil
	}
	var err error
	vapidPrivate, vapidPublic, err = webpush.GenerateVAPIDKeys()
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('vapid_public', $1), ('vapid_private', $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, vapidPublic, vapidPrivate)
	if err != nil {
		return err
	}
	log.Println("VAPID keys generated and saved to DB")
	return nil
}

