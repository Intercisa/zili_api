package main

import (
	"database/sql"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

type weeklyReport struct {
	From        string
	To          string
	TotalMilkG  int
	FeedCount   int
	BreastCount int
	BottleCount int
	SleepMin    int
	AwakeMin    int
	LatestWeight *int
	WeightGainG  *int
	Milestones  []string
	Events      []string
}

func buildWeeklyReport() (*weeklyReport, error) {
	bp := budapest()
	now := time.Now().In(bp)
	to := now.Format("2006-01-02")
	from := now.AddDate(0, 0, -6).Format("2006-01-02")

	r := &weeklyReport{From: from, To: to}

	// Milk totals
	rows, err := db.Query(`
		SELECT
			COALESCE(SUM(post_feed_weight_g - pre_feed_weight_g), 0),
			COUNT(*),
			SUM(CASE WHEN fed_breast THEN 1 ELSE 0 END),
			SUM(CASE WHEN fed_bottle THEN 1 ELSE 0 END)
		FROM zili_daily_log
		WHERE pre_feed_weight_g IS NOT NULL
		  AND post_feed_weight_g IS NOT NULL
		  AND post_feed_weight_g > pre_feed_weight_g
		  AND log_date BETWEEN $1 AND $2
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&r.TotalMilkG, &r.FeedCount, &r.BreastCount, &r.BottleCount)
	}

	// Sleep summary
	sleepRows, err := db.Query(`
		SELECT log_date::text, log_time::text, sleep_event
		FROM zili_daily_log
		WHERE log_time IS NOT NULL
		  AND sleep_event IS NOT NULL
		  AND log_date BETWEEN $1::date - INTERVAL '1 day' AND $2::date
		ORDER BY log_date, log_time
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer sleepRows.Close()
	var sleepLogs []sleepLog
	for sleepRows.Next() {
		var l sleepLog
		var event string
		sleepRows.Scan(&l.Date, &l.Time, &event)
		if len(l.Date) > 10 { l.Date = l.Date[:10] }
		if len(l.Time) > 5 { l.Time = l.Time[:5] }
		if event == "fell_asleep" { l.Summary = "elaludt" }
		if event == "woke_up" { l.Summary = "ébredt" }
		sleepLogs = append(sleepLogs, l)
	}
	sleepResult := calcSleepAwake(sleepLogs, from, to, now)
	for _, d := range sleepResult {
		r.SleepMin += d.SleepMin
		r.AwakeMin += d.AwakeMin
	}

	// Latest weight
	var latestWeight sql.NullInt64
	db.QueryRow(`
		SELECT measurement_weight_g FROM zili_daily_log
		WHERE measurement_weight_g IS NOT NULL AND log_date BETWEEN $1 AND $2
		ORDER BY log_date DESC, id DESC LIMIT 1
	`, from, to).Scan(&latestWeight)
	if latestWeight.Valid {
		v := int(latestWeight.Int64)
		r.LatestWeight = &v

		var prevWeight sql.NullInt64
		db.QueryRow(`
			SELECT measurement_weight_g FROM zili_daily_log
			WHERE measurement_weight_g IS NOT NULL AND log_date < $1
			ORDER BY log_date DESC, id DESC LIMIT 1
		`, from).Scan(&prevWeight)
		if prevWeight.Valid {
			gain := v - int(prevWeight.Int64)
			r.WeightGainG = &gain
		}
	}

	// Milestones
	mRows, err := db.Query(`
		SELECT daily_summary FROM zili_daily_log
		WHERE milestone = true AND log_date BETWEEN $1 AND $2
		ORDER BY log_date, log_time
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer mRows.Close()
	for mRows.Next() {
		var s string
		mRows.Scan(&s)
		r.Milestones = append(r.Milestones, s)
	}

	// Events
	eRows, err := db.Query(`
		SELECT title, event_date::text FROM zili_events
		WHERE event_date BETWEEN $1 AND $2
		ORDER BY event_date
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer eRows.Close()
	for eRows.Next() {
		var title, date string
		eRows.Scan(&title, &date)
		if len(date) > 10 { date = date[:10] }
		r.Events = append(r.Events, fmt.Sprintf("%s — %s", date, title))
	}

	return r, nil
}

func formatReport(r *weeklyReport) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Weekly report: %s – %s\n\n", r.From, r.To))

	sb.WriteString("🍼 Feeding\n")
	sb.WriteString(fmt.Sprintf("  Total milk: %d g\n", r.TotalMilkG))
	sb.WriteString(fmt.Sprintf("  Feedings: %d (breast: %d, bottle: %d)\n\n", r.FeedCount, r.BreastCount, r.BottleCount))

	sb.WriteString("😴 Sleep\n")
	sb.WriteString(fmt.Sprintf("  Total sleep: %dh %dm\n", r.SleepMin/60, r.SleepMin%60))
	sb.WriteString(fmt.Sprintf("  Total awake: %dh %dm\n\n", r.AwakeMin/60, r.AwakeMin%60))

	sb.WriteString("⚖️ Weight\n")
	if r.LatestWeight != nil {
		sb.WriteString(fmt.Sprintf("  Latest: %d g\n", *r.LatestWeight))
	}
	if r.WeightGainG != nil {
		sb.WriteString(fmt.Sprintf("  Gain since last measurement: %+d g\n", *r.WeightGainG))
	}
	sb.WriteString("\n")

	if len(r.Milestones) > 0 {
		sb.WriteString("🎉 Milestones\n")
		for _, m := range r.Milestones {
			sb.WriteString(fmt.Sprintf("  • %s\n", m))
		}
		sb.WriteString("\n")
	}

	if len(r.Events) > 0 {
		sb.WriteString("📅 Events\n")
		for _, e := range r.Events {
			sb.WriteString(fmt.Sprintf("  • %s\n", e))
		}
	}

	return sb.String()
}

func sendWeeklyReport() error {
	r, err := buildWeeklyReport()
	if err != nil {
		return err
	}

	from := getEnv("SMTP_FROM", "")
	toRaw := getEnv("SMTP_TO", "")
	var toList []string
	for _, addr := range strings.Split(toRaw, ",") {
		if t := strings.TrimSpace(addr); t != "" {
			toList = append(toList, t)
		}
	}
	password := getEnv("SMTP_PASSWORD", "")
	host := "smtp.gmail.com"
	smtpAddr := host + ":587"

	subject := fmt.Sprintf("Zili weekly report %s – %s", r.From, r.To)
	body := formatReport(r)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, strings.Join(toList, ", "), subject, body)

	auth := smtp.PlainAuth("", from, password, host)
	return smtp.SendMail(smtpAddr, auth, from, toList, []byte(msg))
}

